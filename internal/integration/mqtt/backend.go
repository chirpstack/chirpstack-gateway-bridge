package mqtt

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/integration/mqtt/auth"
	"github.com/brocaar/lorawan"
)

// Backend implements a MQTT backend.
type Backend struct {
	auth auth.Authentication

	conn       paho.Client
	connMux    sync.RWMutex
	connClosed bool
	clientOpts *paho.ClientOptions

	downlinkFrameFunc             func(gw.DownlinkFrame)
	gatewayConfigurationFunc      func(gw.GatewayConfiguration)
	gatewayCommandExecRequestFunc func(gw.GatewayCommandExecRequest)
	rawPacketForwarderCommandFunc func(gw.RawPacketForwarderCommand)

	gatewaysMux             sync.RWMutex
	gateways                map[lorawan.EUI64]struct{}
	gatewaysSubscribedMux   sync.Mutex
	gatewaysSubscribed      map[lorawan.EUI64]struct{}
	terminateOnConnectError bool

	qos                  uint8
	eventTopicTemplate   *template.Template
	commandTopicTemplate *template.Template

	marshal   func(msg proto.Message) ([]byte, error)
	unmarshal func(b []byte, msg proto.Message) error
}

// NewBackend creates a new Backend.
func NewBackend(conf config.Config) (*Backend, error) {
	var err error

	b := Backend{
		qos:                     conf.Integration.MQTT.Auth.Generic.QOS,
		terminateOnConnectError: conf.Integration.MQTT.TerminateOnConnectError,
		clientOpts:              paho.NewClientOptions(),
		gateways:                make(map[lorawan.EUI64]struct{}),
		gatewaysSubscribed:      make(map[lorawan.EUI64]struct{}),
	}

	switch conf.Integration.MQTT.Auth.Type {
	case "generic":
		b.auth, err = auth.NewGenericAuthentication(conf)
		if err != nil {
			return nil, errors.Wrap(err, "integation/mqtt: new generic authentication error")
		}
	case "gcp_cloud_iot_core":
		b.auth, err = auth.NewGCPCloudIoTCoreAuthentication(conf)
		if err != nil {
			return nil, errors.Wrap(err, "integration/mqtt: new GCP Cloud IoT Core authentication error")
		}

		conf.Integration.MQTT.EventTopicTemplate = "/devices/gw-{{ .GatewayID }}/events/{{ .EventType }}"
		conf.Integration.MQTT.CommandTopicTemplate = "/devices/gw-{{ .GatewayID }}/commands/#"
	case "azure_iot_hub":
		b.auth, err = auth.NewAzureIoTHubAuthentication(conf)
		if err != nil {
			return nil, errors.Wrap(err, "integration/mqtt: new azure iot hub authentication error")
		}

		conf.Integration.MQTT.EventTopicTemplate = "devices/{{ .GatewayID }}/messages/events/{{ .EventType }}"
		conf.Integration.MQTT.CommandTopicTemplate = "devices/{{ .GatewayID }}/messages/devicebound/#"
	default:
		return nil, fmt.Errorf("integration/mqtt: unknown auth type: %s", conf.Integration.MQTT.Auth.Type)
	}

	switch conf.Integration.Marshaler {
	case "json":
		b.marshal = func(msg proto.Message) ([]byte, error) {
			marshaler := &jsonpb.Marshaler{
				EnumsAsInts:  false,
				EmitDefaults: true,
			}
			str, err := marshaler.MarshalToString(msg)
			return []byte(str), err
		}

		b.unmarshal = func(b []byte, msg proto.Message) error {
			unmarshaler := &jsonpb.Unmarshaler{
				AllowUnknownFields: true, // we don't want to fail on unknown fields
			}
			return unmarshaler.Unmarshal(bytes.NewReader(b), msg)
		}
	case "protobuf":
		b.marshal = func(msg proto.Message) ([]byte, error) {
			return proto.Marshal(msg)
		}

		b.unmarshal = func(b []byte, msg proto.Message) error {
			return proto.Unmarshal(b, msg)
		}
	default:
		return nil, fmt.Errorf("integration/mqtt: unknown marshaler: %s", conf.Integration.Marshaler)
	}

	b.eventTopicTemplate, err = template.New("event").Parse(conf.Integration.MQTT.EventTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "integration/mqtt: parse event-topic template error")
	}

	b.commandTopicTemplate, err = template.New("event").Parse(conf.Integration.MQTT.CommandTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "integration/mqtt: parse event-topic template error")
	}

	b.clientOpts.SetProtocolVersion(4)
	b.clientOpts.SetAutoReconnect(true) // this is required for buffering messages in case offline!
	b.clientOpts.SetOnConnectHandler(b.onConnected)
	b.clientOpts.SetConnectionLostHandler(b.onConnectionLost)
	b.clientOpts.SetKeepAlive(conf.Integration.MQTT.KeepAlive)
	b.clientOpts.SetMaxReconnectInterval(conf.Integration.MQTT.MaxReconnectInterval)

	if err = b.auth.Init(b.clientOpts); err != nil {
		return nil, errors.Wrap(err, "mqtt: init authentication error")
	}

	return &b, nil
}

// Start starts the integration.
func (b *Backend) Start() error {
	b.connectLoop()
	go b.reconnectLoop()
	go b.subscribeLoop()
	return nil
}

// Stop stops the integration.
func (b *Backend) Stop() error {
	b.connMux.Lock()
	defer b.connMux.Unlock()

	b.conn.Disconnect(250)
	b.connClosed = true
	return nil
}

// SetDownlinkFrameFunc sets the DownlinkFrame handler func.
func (b *Backend) SetDownlinkFrameFunc(f func(gw.DownlinkFrame)) {
	b.downlinkFrameFunc = f
}

// SetGatewayConfigurationFunc sets the GatewayConfiguration handler func.
func (b *Backend) SetGatewayConfigurationFunc(f func(gw.GatewayConfiguration)) {
	b.gatewayConfigurationFunc = f
}

// SetGatewayCommandExecRequestFunc sets the GatewayCommandExecRequest handler func.
func (b *Backend) SetGatewayCommandExecRequestFunc(f func(gw.GatewayCommandExecRequest)) {
	b.gatewayCommandExecRequestFunc = f
}

// SetRawPacketForwarderCommandFunc sets the RawPacketForwarderCommand handler func.
func (b *Backend) SetRawPacketForwarderCommandFunc(f func(gw.RawPacketForwarderCommand)) {
	b.rawPacketForwarderCommandFunc = f
}

// SetGatewaySubscription sets or unsets the gateway.
// Note: the actual MQTT (un)subscribe happens in a separate function to avoid
// race conditions in case of connection issues. This way, the gateways map
// always reflect the desired state.
func (b *Backend) SetGatewaySubscription(subscribe bool, gatewayID lorawan.EUI64) error {
	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"subscribe":  subscribe,
	}).Debug("integration/mqtt: set gateway subscription")

	b.gatewaysMux.Lock()
	defer b.gatewaysMux.Unlock()

	if subscribe {
		b.gateways[gatewayID] = struct{}{}
	} else {
		delete(b.gateways, gatewayID)
	}

	return nil
}

func (b *Backend) subscribeGateway(gatewayID lorawan.EUI64) error {
	topic := bytes.NewBuffer(nil)
	if err := b.commandTopicTemplate.Execute(topic, struct{ GatewayID lorawan.EUI64 }{gatewayID}); err != nil {
		return errors.Wrap(err, "execute command topic template error")
	}
	log.WithFields(log.Fields{
		"topic": topic.String(),
		"qos":   b.qos,
	}).Info("integration/mqtt: subscribing to topic")

	if token := b.conn.Subscribe(topic.String(), b.qos, b.handleCommand); token.Wait() && token.Error() != nil {
		return errors.Wrap(token.Error(), "subscribe topic error")
	}
	return nil
}

func (b *Backend) unsubscribeGateway(gatewayID lorawan.EUI64) error {
	topic := bytes.NewBuffer(nil)
	if err := b.commandTopicTemplate.Execute(topic, struct{ GatewayID lorawan.EUI64 }{gatewayID}); err != nil {
		return errors.Wrap(err, "execute command topic template error")
	}
	log.WithFields(log.Fields{
		"topic": topic.String(),
	}).Info("integration/mqtt: unsubscribing from topic")

	if token := b.conn.Unsubscribe(topic.String()); token.Wait() && token.Error() != nil {
		return errors.Wrap(token.Error(), "unsubscribe topic error")
	}

	return nil
}

// PublishEvent publishes the given event.
func (b *Backend) PublishEvent(gatewayID lorawan.EUI64, event string, id uuid.UUID, v proto.Message) error {
	mqttEventCounter(event).Inc()
	idPrefix := map[string]string{
		"up":    "uplink_",
		"ack":   "downlink_",
		"stats": "stats_",
		"exec":  "exec_",
		"raw":   "raw_",
	}
	return b.publish(gatewayID, event, log.Fields{
		idPrefix[event] + "id": id,
	}, v)
}

func (b *Backend) connect() error {
	b.connMux.Lock()
	defer b.connMux.Unlock()

	if err := b.auth.Update(b.clientOpts); err != nil {
		return errors.Wrap(err, "integration/mqtt: update authentication error")
	}

	b.conn = paho.NewClient(b.clientOpts)
	if token := b.conn.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

// connectLoop blocks until the client is connected
func (b *Backend) connectLoop() {
	for {
		if err := b.connect(); err != nil {
			if b.terminateOnConnectError {
				log.Fatal(err)
			}

			log.WithError(err).Error("integration/mqtt: connection error")
			time.Sleep(time.Second * 2)

		} else {
			break
		}
	}
}

func (b *Backend) disconnect() error {
	mqttDisconnectCounter().Inc()

	b.connMux.Lock()
	defer b.connMux.Unlock()

	b.conn.Disconnect(250)
	return nil
}

func (b *Backend) reconnectLoop() {
	if b.auth.ReconnectAfter() > 0 {
		for {
			b.connMux.RLock()
			closed := b.connClosed
			b.connMux.RUnlock()

			if closed {
				break
			}
			time.Sleep(b.auth.ReconnectAfter())
			log.Info("mqtt: re-connect triggered")

			mqttReconnectCounter().Inc()

			b.disconnect()
			b.connectLoop()
		}
	}
}

func (b *Backend) onConnected(c paho.Client) {
	mqttConnectCounter().Inc()
	log.Info("integration/mqtt: connected to mqtt broker")

	b.gatewaysSubscribedMux.Lock()
	defer b.gatewaysSubscribedMux.Unlock()

	// reset the subscriptions as we have a new connection
	// note: this is done in the onConnected function because the subscribeLoop
	// locks the gatewaysSubscribedMux and will only release it after all
	// (un)subscribe operations have been completed. If it would be done in the
	// onConnectionLost function, the function could block until the connection
	// is restored because the (un)subscribe operations will block until then.
	b.gatewaysSubscribed = make(map[lorawan.EUI64]struct{})
}

func (b *Backend) subscribeLoop() {
	for {
		b.connMux.RLock()
		closed := b.connClosed
		b.connMux.RUnlock()
		if closed {
			break
		}

		var subscribe []lorawan.EUI64
		var unsubscribe []lorawan.EUI64

		b.gatewaysMux.RLock()
		b.gatewaysSubscribedMux.Lock()

		// subscribe
		for gatewayID := range b.gateways {
			if _, ok := b.gatewaysSubscribed[gatewayID]; !ok {
				subscribe = append(subscribe, gatewayID)
			}
		}

		// unsubscribe
		for gatewayID := range b.gatewaysSubscribed {
			if _, ok := b.gateways[gatewayID]; !ok {
				unsubscribe = append(unsubscribe, gatewayID)
			}
		}

		// unlock gatewaysMux so that SetGatewaySubscription can write again
		// to the map, in which case changes are picked up in the next run
		b.gatewaysMux.RUnlock()

		for _, gatewayID := range subscribe {
			if err := b.subscribeGateway(gatewayID); err != nil {
				log.WithError(err).WithField("gateway_id", gatewayID).Error("integration/mqtt: subscribe gateway error")
			} else {
				b.gatewaysSubscribed[gatewayID] = struct{}{}
			}
		}

		for _, gatewayID := range unsubscribe {
			if err := b.unsubscribeGateway(gatewayID); err != nil {
				log.WithError(err).WithField("gateway_id", gatewayID).Error("integration/mqtt: unsubscribe gateway error")
			} else {
				delete(b.gatewaysSubscribed, gatewayID)
			}
		}

		b.gatewaysSubscribedMux.Unlock()
		time.Sleep(time.Millisecond * 100)
	}
}

func (b *Backend) onConnectionLost(c paho.Client, err error) {
	mqttDisconnectCounter().Inc()
	log.WithError(err).Error("mqtt: connection error")
}

func (b *Backend) handleDownlinkFrame(c paho.Client, msg paho.Message) {
	var downlinkFrame gw.DownlinkFrame
	if err := b.unmarshal(msg.Payload(), &downlinkFrame); err != nil {
		log.WithFields(log.Fields{
			"topic": msg.Topic(),
		}).WithError(err).Error("integration/mqtt: unmarshal downlink frame error")
		return
	}

	var downID uuid.UUID
	copy(downID[:], downlinkFrame.GetDownlinkId())

	// For backwards compatibility.
	if len(downlinkFrame.Items) == 0 && (downlinkFrame.TxInfo != nil && len(downlinkFrame.PhyPayload) != 0) {
		downlinkFrame.Items = append(downlinkFrame.Items, &gw.DownlinkFrameItem{
			PhyPayload: downlinkFrame.PhyPayload,
			TxInfo:     downlinkFrame.TxInfo,
		})

		downlinkFrame.GatewayId = downlinkFrame.Items[0].GetTxInfo().GetGatewayId()
	}

	if len(downlinkFrame.Items) == 0 {
		log.WithFields(log.Fields{
			"downlink_id": downID,
		}).Error("integration/mqtt: downlink must have at least one item")
		return
	}

	var gatewayID lorawan.EUI64
	copy(gatewayID[:], downlinkFrame.GatewayId)

	log.WithFields(log.Fields{
		"gateway_id":  gatewayID,
		"downlink_id": downID,
	}).Info("integration/mqtt: downlink frame received")

	if b.downlinkFrameFunc != nil {
		b.downlinkFrameFunc(downlinkFrame)
	}
}

func (b *Backend) handleGatewayConfiguration(c paho.Client, msg paho.Message) {
	log.WithFields(log.Fields{
		"topic": msg.Topic(),
	}).Info("integration/mqtt: gateway configuration received")

	var gatewayConfig gw.GatewayConfiguration
	if err := b.unmarshal(msg.Payload(), &gatewayConfig); err != nil {
		log.WithError(err).Error("integration/mqtt: unmarshal gateway configuration error")
		return
	}

	if b.gatewayConfigurationFunc != nil {
		b.gatewayConfigurationFunc(gatewayConfig)
	}
}

func (b *Backend) handleGatewayCommandExecRequest(c paho.Client, msg paho.Message) {
	var gatewayCommandExecRequest gw.GatewayCommandExecRequest
	if err := b.unmarshal(msg.Payload(), &gatewayCommandExecRequest); err != nil {
		log.WithFields(log.Fields{
			"topic": msg.Topic(),
		}).WithError(err).Error("integration/mqtt: unmarshal gateway command execution request error")
		return
	}

	var gatewayID lorawan.EUI64
	var execID uuid.UUID
	copy(gatewayID[:], gatewayCommandExecRequest.GetGatewayId())
	copy(execID[:], gatewayCommandExecRequest.GetExecId())

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"exec_id":    execID,
	}).Info("integration/mqtt: gateway command execution request received")

	if b.gatewayCommandExecRequestFunc != nil {
		b.gatewayCommandExecRequestFunc(gatewayCommandExecRequest)
	}
}

func (b *Backend) handleRawPacketForwarderCommand(c paho.Client, msg paho.Message) {
	var rawPacketForwarderCommand gw.RawPacketForwarderCommand
	if err := b.unmarshal(msg.Payload(), &rawPacketForwarderCommand); err != nil {
		log.WithFields(log.Fields{
			"topic": msg.Topic(),
		}).WithError(err).Error("integration/mqtt: unmarshal raw packet-forwarder command error")
		return
	}

	var gatewayID lorawan.EUI64
	var rawID uuid.UUID
	copy(gatewayID[:], rawPacketForwarderCommand.GetGatewayId())
	copy(rawID[:], rawPacketForwarderCommand.GetRawId())

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"raw_id":     rawID,
	}).Info("integration/mqtt: raw packet-forwarder command received")

	if b.rawPacketForwarderCommandFunc != nil {
		b.rawPacketForwarderCommandFunc(rawPacketForwarderCommand)
	}
}

func (b *Backend) handleCommand(c paho.Client, msg paho.Message) {
	if strings.HasSuffix(msg.Topic(), "down") || strings.Contains(msg.Topic(), "command=down") {
		mqttCommandCounter("down").Inc()
		b.handleDownlinkFrame(c, msg)
	} else if strings.HasSuffix(msg.Topic(), "config") || strings.Contains(msg.Topic(), "command=config") {
		mqttCommandCounter("config").Inc()
		b.handleGatewayConfiguration(c, msg)
	} else if strings.HasSuffix(msg.Topic(), "exec") || strings.Contains(msg.Topic(), "command=exec") {
		b.handleGatewayCommandExecRequest(c, msg)
	} else if strings.HasSuffix(msg.Topic(), "raw") || strings.Contains(msg.Topic(), "command=raw") {
		b.handleRawPacketForwarderCommand(c, msg)
	} else {
		log.WithFields(log.Fields{
			"topic": msg.Topic(),
		}).Warning("integration/mqtt: unexpected command received")
	}
}

func (b *Backend) publish(gatewayID lorawan.EUI64, event string, fields log.Fields, msg proto.Message) error {
	topic := bytes.NewBuffer(nil)
	if err := b.eventTopicTemplate.Execute(topic, struct {
		GatewayID lorawan.EUI64
		EventType string
	}{gatewayID, event}); err != nil {
		return errors.Wrap(err, "execute event template error")
	}

	bytes, err := b.marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal message error")
	}

	fields["topic"] = topic.String()
	fields["qos"] = b.qos
	fields["event"] = event

	log.WithFields(fields).Info("integration/mqtt: publishing event")
	if token := b.conn.Publish(topic.String(), b.qos, false, bytes); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
