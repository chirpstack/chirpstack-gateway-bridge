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

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/integration/mqtt/auth"
	"github.com/brocaar/chirpstack-api/go/gw"
	"github.com/brocaar/lorawan"
)

// Backend implements a MQTT backend.
type Backend struct {
	sync.RWMutex

	auth                          auth.Authentication
	conn                          paho.Client
	closed                        bool
	clientOpts                    *paho.ClientOptions
	downlinkFrameChan             chan gw.DownlinkFrame
	gatewayConfigurationChan      chan gw.GatewayConfiguration
	gatewayCommandExecRequestChan chan gw.GatewayCommandExecRequest
	gateways                      map[lorawan.EUI64]struct{}

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
		qos:                           conf.Integration.MQTT.Auth.Generic.QOS,
		clientOpts:                    paho.NewClientOptions(),
		downlinkFrameChan:             make(chan gw.DownlinkFrame),
		gatewayConfigurationChan:      make(chan gw.GatewayConfiguration),
		gatewayCommandExecRequestChan: make(chan gw.GatewayCommandExecRequest),
		gateways:                      make(map[lorawan.EUI64]struct{}),
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
	b.clientOpts.SetMaxReconnectInterval(conf.Integration.MQTT.MaxReconnectInterval)

	if err = b.auth.Init(b.clientOpts); err != nil {
		return nil, errors.Wrap(err, "mqtt: init authentication error")
	}

	b.connectLoop()
	go b.reconnectLoop()

	return &b, nil
}

// Close closes the backend.
func (b *Backend) Close() error {
	b.Lock()
	b.closed = true
	b.Unlock()

	b.conn.Disconnect(250)
	return nil
}

// GetDownlinkFrameChan returns the downlink frame channel.
func (b *Backend) GetDownlinkFrameChan() chan gw.DownlinkFrame {
	return b.downlinkFrameChan
}

// GetGatewayConfigurationChan returns the gateway configuration channel.
func (b *Backend) GetGatewayConfigurationChan() chan gw.GatewayConfiguration {
	return b.gatewayConfigurationChan
}

// GetGatewayCommandExecRequestChan() returns the channel for gateway command execution.
func (b *Backend) GetGatewayCommandExecRequestChan() chan gw.GatewayCommandExecRequest {
	return b.gatewayCommandExecRequestChan
}

// SubscribeGateway subscribes a gateway to its topics.
func (b *Backend) SubscribeGateway(gatewayID lorawan.EUI64) error {
	b.Lock()
	defer b.Unlock()

	if err := b.subscribeGateway(gatewayID); err != nil {
		return err
	}

	b.gateways[gatewayID] = struct{}{}
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

// UnsubscribeGateway unsubscribes the gateway from its topics.
func (b *Backend) UnsubscribeGateway(gatewayID lorawan.EUI64) error {
	b.Lock()
	defer b.Unlock()

	topic := bytes.NewBuffer(nil)
	if err := b.commandTopicTemplate.Execute(topic, struct{ GatewayID lorawan.EUI64 }{gatewayID}); err != nil {
		return errors.Wrap(err, "execute command topic template error")
	}
	log.WithFields(log.Fields{
		"topic": topic.String(),
	}).Info("integration/mqtt: unsubscribe topic")

	if token := b.conn.Unsubscribe(topic.String()); token.Wait() && token.Error() != nil {
		return errors.Wrap(token.Error(), "unsubscribe topic error")
	}

	delete(b.gateways, gatewayID)
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
	}
	return b.publish(gatewayID, event, log.Fields{
		idPrefix[event] + "id": id,
	}, v)
}

func (b *Backend) connect() error {
	b.Lock()
	defer b.Unlock()

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
			log.WithError(err).Error("integration/mqtt: connection error")
			time.Sleep(time.Second * 2)

		} else {
			break
		}
	}
}

func (b *Backend) disconnect() error {
	mqttDisconnectCounter().Inc()

	b.Lock()
	defer b.Unlock()

	b.conn.Disconnect(250)
	return nil
}

func (b *Backend) reconnectLoop() {
	if b.auth.ReconnectAfter() > 0 {
		for {
			if b.closed {
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

	b.RLock()
	defer b.RUnlock()

	log.Info("integration/mqtt: connected to mqtt broker")

	for gatewayID := range b.gateways {
		for {
			if err := b.subscribeGateway(gatewayID); err != nil {
				log.WithError(err).WithField("gateway_id", gatewayID).Error("integration/mqtt: subscribe gateway error")
				time.Sleep(time.Second)
				continue
			}

			break
		}
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

	var gatewayID lorawan.EUI64
	var downID uuid.UUID
	copy(gatewayID[:], downlinkFrame.GetTxInfo().GetGatewayId())
	copy(downID[:], downlinkFrame.GetDownlinkId())

	log.WithFields(log.Fields{
		"gateway_id":  gatewayID,
		"downlink_id": downID,
	}).Info("integration/mqtt: downlink frame received")

	b.downlinkFrameChan <- downlinkFrame
}

// TODO: this feature is deprecated. Remove this in the next major release.
func (b *Backend) handleGatewayConfiguration(c paho.Client, msg paho.Message) {
	log.WithFields(log.Fields{
		"topic": msg.Topic(),
	}).Info("integration/mqtt: gateway configuration received")

	var gatewayConfig gw.GatewayConfiguration
	if err := b.unmarshal(msg.Payload(), &gatewayConfig); err != nil {
		log.WithError(err).Error("integration/mqtt: unmarshal gateway configuration error")
		return
	}

	b.gatewayConfigurationChan <- gatewayConfig
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

	b.gatewayCommandExecRequestChan <- gatewayCommandExecRequest
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
