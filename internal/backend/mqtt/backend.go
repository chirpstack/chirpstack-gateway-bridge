package mqtt

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/lora-gateway-bridge/internal/backend/mqtt/auth"
	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
)

// BackendAuthConfig holds the MQTT pub-sub backend auth configuration.
type BackendAuthConfig struct {
	Type            string
	Generic         auth.GenericConfig
	GCPCloudIoTCore auth.GCPCloudIoTCoreConfig `mapstructure:"gcp_cloud_iot_core"`
}

// BackendConfig holds the MQTT pub-sub backend configuration.
type BackendConfig struct {
	UplinkTopicTemplate   string `mapstructure:"uplink_topic_template"`
	DownlinkTopicTemplate string `mapstructure:"downlink_topic_template"`
	StatsTopicTemplate    string `mapstructure:"stats_topic_template"`
	AckTopicTemplate      string `mapstructure:"ack_topic_template"`
	ConfigTopicTemplate   string `mapstructure:"config_topic_template"`
	Marshaler             string `mapstructure:"marshaler"`
	Auth                  BackendAuthConfig

	// for backwards compatibility
	Server               string
	Username             string
	Password             string
	CACert               string        `mapstructure:"ca_cert"`
	TLSCert              string        `mapstructure:"tls_cert"`
	TLSKey               string        `mapstructure:"tls_key"`
	QOS                  uint8         `mapstructure:"qos"`
	CleanSession         bool          `mapstructure:"clean_session"`
	ClientID             string        `mapstructure:"client_id"`
	MaxReconnectInterval time.Duration `mapstructure:"max_reconnect_interval"`

	AlwaysSubscribeMACs []lorawan.EUI64 `mapstructure:"-"`
}

// Backend implements a MQTT backend.
type Backend struct {
	sync.RWMutex

	auth                     auth.Authentication
	conn                     paho.Client
	closed                   bool
	clientOpts               *paho.ClientOptions
	downlinkFrameChan        chan gw.DownlinkFrame
	gatewayConfigurationChan chan gw.GatewayConfiguration
	gateways                 map[lorawan.EUI64]bool // the bool indicates if the gateway must always be subscribed
	topicHandlers            []topicHandler

	qos              uint8
	uplinkTemplate   *template.Template
	downlinkTemplate *template.Template
	statsTemplate    *template.Template
	ackTemplate      *template.Template
	configTemplate   *template.Template

	marshal   func(msg proto.Message) ([]byte, error)
	unmarshal func(b []byte, msg proto.Message) error
}

type topicHandler struct {
	topicTemplate *template.Template
	handler       paho.MessageHandler
}

// NewBackend creates a new Backend.
func NewBackend(config BackendConfig) (*Backend, error) {
	var err error

	b := Backend{
		clientOpts:               paho.NewClientOptions(),
		downlinkFrameChan:        make(chan gw.DownlinkFrame),
		gatewayConfigurationChan: make(chan gw.GatewayConfiguration),
		gateways:                 make(map[lorawan.EUI64]bool),
	}

	for i := range config.AlwaysSubscribeMACs {
		b.gateways[config.AlwaysSubscribeMACs[i]] = true
	}

	switch config.Auth.Type {
	case "generic":
		b.auth, err = auth.NewGenericAuthentication(config.Auth.Generic)
		if err != nil {
			return nil, errors.Wrap(err, "mqtt: new generic authentication error")
		}
	case "gcp_cloud_iot_core":
		b.auth, err = auth.NewGCPCloudIoTCoreAuthentication(config.Auth.GCPCloudIoTCore)
		if err != nil {
			return nil, errors.Wrap(err, "mqtt: new GCP Cloud IoT Core authentication error")
		}

		config.UplinkTopicTemplate = `/devices/gw-{{ .MAC }}/events/up`
		config.StatsTopicTemplate = `/devices/gw-{{ .MAC }}/events/stats`
		config.AckTopicTemplate = `/devices/gw-{{ .MAC }}/events/ack`
		config.DownlinkTopicTemplate = `/devices/gw-{{ .MAC }}/commands/#`
		config.ConfigTopicTemplate = `/devices/gw-{{ .MAC }}/commands/#`
	default:
		return nil, fmt.Errorf("mqtt: unknown auth type: %s", config.Auth.Type)
	}

	switch config.Marshaler {
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
		return nil, fmt.Errorf("mqtt: unkown marshaler: %s", config.Marshaler)
	}

	b.uplinkTemplate, err = template.New("uplink").Parse(config.UplinkTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "mqtt: parse uplink template error")
	}

	b.downlinkTemplate, err = template.New("downlink").Parse(config.DownlinkTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "mqtt: parse downlink template error")
	}

	b.statsTemplate, err = template.New("stats").Parse(config.StatsTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "mqtt: parse stats template error")
	}

	b.ackTemplate, err = template.New("ack").Parse(config.AckTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "mqtt: parse ack template error")
	}

	b.configTemplate, err = template.New("config").Parse(config.ConfigTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "mqtt: parse config template error")
	}

	switch config.Auth.Type {
	case "gcp_cloud_iot_core":
		b.topicHandlers = []topicHandler{
			{
				topicTemplate: b.downlinkTemplate,
				handler:       b.commandHandler,
			},
		}
	default:
		b.topicHandlers = []topicHandler{
			{
				topicTemplate: b.downlinkTemplate,
				handler:       b.downlinkFrameHandler,
			},
			{
				topicTemplate: b.configTemplate,
				handler:       b.gatewayConfigHandler,
			},
		}
	}

	b.clientOpts.SetProtocolVersion(4)
	b.clientOpts.SetAutoReconnect(false)
	b.clientOpts.SetOnConnectHandler(b.onConnected)
	b.clientOpts.SetConnectionLostHandler(b.onConnectionLost)

	if err = b.auth.Init(b.clientOpts); err != nil {
		return nil, errors.Wrap(err, "mqtt: init authentication error")
	}

	b.connectLoop()
	go b.reconnectLoop()

	return &b, nil
}

// Close closes the backend.
func (b *Backend) Close() {
	b.Lock()
	b.closed = true
	b.Unlock()

	b.conn.Disconnect(250)
}

// DownlinkFrameChan returns the downlink frame channel.
func (b *Backend) DownlinkFrameChan() chan gw.DownlinkFrame {
	return b.downlinkFrameChan
}

// GatewayConfigurationChan returns the gateway configuration channel.
func (b *Backend) GatewayConfigurationChan() chan gw.GatewayConfiguration {
	return b.gatewayConfigurationChan
}

// SubscribeGateway subscribes a gateway to its topics.
func (b *Backend) SubscribeGateway(gatewayID lorawan.EUI64) error {
	mqttEventCounter("subscribe_gateway")

	b.Lock()
	defer b.Unlock()

	if alwaysSubscribe, ok := b.gateways[gatewayID]; ok && alwaysSubscribe {
		return nil
	}

	for _, th := range b.topicHandlers {
		topic := bytes.NewBuffer(nil)
		if err := th.topicTemplate.Execute(topic, struct{ MAC lorawan.EUI64 }{gatewayID}); err != nil {
			return errors.Wrap(err, "execute downlink template error")
		}
		log.WithFields(log.Fields{
			"topic": topic.String(),
			"qos":   b.qos,
		}).Info("mqtt: subscribing to topic")

		err := mqttSubscribeTimer(false, func() error {
			if token := b.conn.Subscribe(topic.String(), b.qos, th.handler); token.Wait() && token.Error() != nil {
				return errors.Wrap(token.Error(), "subscribe topic error")
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	b.gateways[gatewayID] = false
	return nil
}

// UnsubscribeGateway unsubscribes the gateway from its topics.
func (b *Backend) UnsubscribeGateway(gatewayID lorawan.EUI64) error {
	mqttEventCounter("unsubscribe_gateway")

	b.Lock()
	defer b.Unlock()

	if alwaysSubscribe, ok := b.gateways[gatewayID]; ok && alwaysSubscribe {
		return nil
	}

	for _, th := range b.topicHandlers {
		topic := bytes.NewBuffer(nil)
		if err := th.topicTemplate.Execute(topic, struct{ MAC lorawan.EUI64 }{gatewayID}); err != nil {
			return errors.Wrap(err, "execute template error")
		}
		log.WithFields(log.Fields{
			"topic": topic.String(),
		}).Info("mqtt: unsubscribe topic")

		err := mqttUnsubscribeTimer(func() error {
			if token := b.conn.Unsubscribe(topic.String()); token.Wait() && token.Error() != nil {
				return errors.Wrap(token.Error(), "unsubscribe topic error")
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	delete(b.gateways, gatewayID)
	return nil
}

// PublishUplinkFrame publishes an uplink-frame to the MQTT broker.
func (b *Backend) PublishUplinkFrame(gatewayID lorawan.EUI64, msg gw.UplinkFrame) error {
	return mqttPublishTimer("uplink", func() error {
		return b.publish(gatewayID, b.uplinkTemplate, &msg)
	})
}

// PublishGatewayStats publishes a gateway stats message to the MQTT broker.
func (b *Backend) PublishGatewayStats(gatewayID lorawan.EUI64, msg gw.GatewayStats) error {
	return mqttPublishTimer("stats", func() error {
		return b.publish(gatewayID, b.statsTemplate, &msg)
	})
}

// PublishDownlinkTXAck publishes a downlink ack to the MQTT broker.
func (b *Backend) PublishDownlinkTXAck(gatewayID lorawan.EUI64, msg gw.DownlinkTXAck) error {
	return mqttPublishTimer("ack", func() error {
		return b.publish(gatewayID, b.ackTemplate, &msg)
	})
}

func (b *Backend) connect() error {
	b.Lock()
	defer b.Unlock()

	if err := b.auth.Update(b.clientOpts); err != nil {
		return errors.Wrap(err, "mqtt: update authentication error")
	}

	b.conn = paho.NewClient(b.clientOpts)

	return mqttConnectTimer(func() error {
		if token := b.conn.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
		return nil
	})
}

// connectLoop blocks until the client is connected
func (b *Backend) connectLoop() {
	for {
		if err := b.connect(); err != nil {
			log.WithError(err).Error("mqtt: connection error")
			time.Sleep(time.Second * 2)

		} else {
			break
		}
	}
}

func (b *Backend) disconnect() error {
	mqttEventCounter("disconnect")

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

			b.disconnect()
			b.connectLoop()
		}
	}
}

func (b *Backend) onConnected(c paho.Client) {
	mqttEventCounter("connected")

	b.RLock()
	defer b.RUnlock()

	log.Info("mqtt: connected to mqtt broker")
	if len(b.gateways) != 0 {
		for _, th := range b.topicHandlers {
			for {
				topics := make(map[string]byte)
				for k := range b.gateways {
					topic := bytes.NewBuffer(nil)
					if err := th.topicTemplate.Execute(topic, struct{ MAC lorawan.EUI64 }{k}); err != nil {
						log.WithError(err).Error("mqtt: execute downlink template error")
						time.Sleep(time.Second)
						continue
					}
					topics[topic.String()] = b.qos

					log.WithFields(log.Fields{
						"topic": topic.String(),
						"qos":   b.qos,
					}).Info("mqtt: subscribing to topic")
				}

				err := mqttSubscribeTimer(true, func() error {
					token := b.conn.SubscribeMultiple(topics, th.handler)
					token.Wait()
					return token.Error()
				})
				if err != nil {
					log.WithError(err).WithFields(log.Fields{
						"topic_count": len(topics),
					}).Error("mqtt: subscribe topics error")
				}

				break
			}
		}
	}
}

func (b *Backend) onConnectionLost(c paho.Client, err error) {
	mqttEventCounter("connection_lost")
	log.WithError(err).Error("mqtt: connection error")
	b.connectLoop()
}

func (b *Backend) downlinkFrameHandler(c paho.Client, msg paho.Message) {
	log.WithFields(log.Fields{
		"topic": msg.Topic(),
	}).Info("mqtt: downlink frame received")

	_ = mqttHandleTimer("downlink", func() error {
		var downlinkFrame gw.DownlinkFrame
		if err := b.unmarshal(msg.Payload(), &downlinkFrame); err != nil {
			log.WithError(err).Error("mqtt: unmarshal downlink frame error")
			return err
		}

		b.downlinkFrameChan <- downlinkFrame
		return nil
	})
}

func (b *Backend) gatewayConfigHandler(c paho.Client, msg paho.Message) {
	log.WithFields(log.Fields{
		"topic": msg.Topic(),
	}).Info("mqtt: gateway configuration received")

	_ = mqttHandleTimer("config", func() error {
		var gatewayConfig gw.GatewayConfiguration
		if err := b.unmarshal(msg.Payload(), &gatewayConfig); err != nil {
			log.WithError(err).Error("mqtt: unmarshal gateway configuration error")
			return err
		}

		b.gatewayConfigurationChan <- gatewayConfig
		return nil
	})
}

func (b *Backend) commandHandler(c paho.Client, msg paho.Message) {
	if strings.HasSuffix(msg.Topic(), "down") {
		b.downlinkFrameHandler(c, msg)
	} else if strings.HasSuffix(msg.Topic(), "config") {
		b.gatewayConfigHandler(c, msg)
	} else {
		log.WithFields(log.Fields{
			"topic": msg.Topic(),
		}).Warning("unexpected command received")
	}
}

func (b *Backend) publish(gatewayID lorawan.EUI64, topicTemplate *template.Template, msg proto.Message) error {
	topic := bytes.NewBuffer(nil)
	if err := topicTemplate.Execute(topic, struct{ MAC lorawan.EUI64 }{gatewayID}); err != nil {
		return errors.Wrap(err, "execute template error")
	}

	bytes, err := b.marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal message error")
	}

	log.WithFields(log.Fields{
		"topic": topic.String(),
		"qos":   b.qos,
	}).Info("mqtt: publishing message")
	if token := b.conn.Publish(topic.String(), b.qos, false, bytes); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
