package mqttpubsub

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"sync"
	"text/template"
	"time"

	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// BackendConfig holds the MQTT pub-sub backend configuration.
type BackendConfig struct {
	Server                string
	Username              string
	Password              string
	QOS                   uint8           `mapstructure:"qos"`
	CleanSession          bool            `mapstructure:"clean_session"`
	ClientID              string          `mapstructure:"client_id"`
	CACert                string          `mapstructure:"ca_cert"`
	TLSCert               string          `mapstructure:"tls_cert"`
	TLSKey                string          `mapstructure:"tls_key"`
	UplinkTopicTemplate   string          `mapstructure:"uplink_topic_template"`
	DownlinkTopicTemplate string          `mapstructure:"downlink_topic_template"`
	StatsTopicTemplate    string          `mapstructure:"stats_topic_template"`
	AckTopicTemplate      string          `mapstructure:"ack_topic_template"`
	ConfigTopicTemplate   string          `mapstructure:"config_topic_template"`
	AlwaysSubscribeMACs   []lorawan.EUI64 `mapstructure:"-"`
}

// Backend implements a MQTT pub-sub backend.
type Backend struct {
	conn             mqtt.Client
	txPacketChan     chan gw.TXPacketBytes
	configPacketChan chan gw.GatewayConfigPacket
	gateways         map[lorawan.EUI64]bool // the bool indicates if the gateway must always be subscribed
	mutex            sync.RWMutex
	config           BackendConfig

	UplinkTemplate   *template.Template
	DownlinkTemplate *template.Template
	StatsTemplate    *template.Template
	AckTemplate      *template.Template
	ConfigTemplate   *template.Template
}

// NewBackend creates a new Backend.
func NewBackend(c BackendConfig) (*Backend, error) {
	var err error

	b := Backend{
		txPacketChan:     make(chan gw.TXPacketBytes),
		configPacketChan: make(chan gw.GatewayConfigPacket),
		gateways:         make(map[lorawan.EUI64]bool),
		config:           c,
	}

	for i := range c.AlwaysSubscribeMACs {
		b.gateways[c.AlwaysSubscribeMACs[i]] = true
	}

	b.UplinkTemplate, err = template.New("uplink").Parse(b.config.UplinkTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "parse uplink template error")
	}

	b.DownlinkTemplate, err = template.New("downlink").Parse(b.config.DownlinkTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "parse downlink template error")
	}

	b.StatsTemplate, err = template.New("stats").Parse(b.config.StatsTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "parse stats template error")
	}

	b.AckTemplate, err = template.New("ack").Parse(b.config.AckTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "parse ack template error")
	}

	b.ConfigTemplate, err = template.New("config").Parse(b.config.ConfigTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "parse config template error")
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(b.config.Server)
	opts.SetUsername(b.config.Username)
	opts.SetPassword(b.config.Password)
	opts.SetCleanSession(b.config.CleanSession)
	opts.SetClientID(b.config.ClientID)
	opts.SetOnConnectHandler(b.onConnected)
	opts.SetConnectionLostHandler(b.onConnectionLost)

	tlsconfig, err := newTLSConfig(b.config.CACert, b.config.TLSCert, b.config.TLSKey)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"ca_cert":  b.config.CACert,
			"tls_cert": b.config.TLSCert,
			"tls_key":  b.config.TLSKey,
		}).Fatal("error loading mqtt certificate files")
	}
	if tlsconfig != nil {
		opts.SetTLSConfig(tlsconfig)
	}

	log.WithField("server", b.config.Server).Info("backend: connecting to mqtt broker")
	b.conn = mqtt.NewClient(opts)
	if token := b.conn.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &b, nil
}

// Close closes the backend.
func (b *Backend) Close() {
	b.conn.Disconnect(250) // wait 250 milisec to complete pending actions
}

// TXPacketChan returns the TXPacketBytes channel.
func (b *Backend) TXPacketChan() chan gw.TXPacketBytes {
	return b.txPacketChan
}

// ConfigPacketChan returns the ConfigPacketChan channel.
func (b *Backend) ConfigPacketChan() chan gw.GatewayConfigPacket {
	return b.configPacketChan
}

// SubscribeGatewayTopics subscribes the backend to the gateway topics.
// (downlink and configuration).
func (b *Backend) SubscribeGatewayTopics(mac lorawan.EUI64) error {
	defer b.mutex.Unlock()
	b.mutex.Lock()

	if alwaysSubscribe, ok := b.gateways[mac]; ok && alwaysSubscribe {
		return nil
	}

	// downlink topic
	topic := bytes.NewBuffer(nil)
	if err := b.DownlinkTemplate.Execute(topic, struct{ MAC lorawan.EUI64 }{mac}); err != nil {
		return errors.Wrap(err, "execute downlink template error")
	}
	log.WithFields(log.Fields{
		"topic": topic.String(),
		"qos":   b.config.QOS,
	}).Info("backend: subscribing to topic")
	if token := b.conn.Subscribe(topic.String(), b.config.QOS, b.txPacketHandler); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	topic.Reset()
	if err := b.ConfigTemplate.Execute(topic, struct{ MAC lorawan.EUI64 }{mac}); err != nil {
		return errors.Wrap(err, "execute config template error")
	}
	log.WithFields(log.Fields{
		"topic": topic.String(),
		"qos":   b.config.QOS,
	}).Info("backend: subscribing to topic")
	if token := b.conn.Subscribe(topic.String(), b.config.QOS, b.configPacketHandler); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	b.gateways[mac] = false
	return nil
}

// UnSubscribeGatewayTopics unsubscribes the backend from the gateway topics.
// (downlink and configuration).
func (b *Backend) UnSubscribeGatewayTopics(mac lorawan.EUI64) error {
	defer b.mutex.Unlock()
	b.mutex.Lock()

	if alwaysSubscribe, ok := b.gateways[mac]; ok && alwaysSubscribe {
		return nil
	}

	for _, t := range []*template.Template{b.DownlinkTemplate, b.ConfigTemplate} {
		topic := bytes.NewBuffer(nil)
		if err := t.Execute(topic, struct{ MAC lorawan.EUI64 }{mac}); err != nil {
			return errors.Wrap(err, "execute template error")
		}
		log.WithField("topic", topic.String()).Info("backend: unsubscribing from topic")
		if token := b.conn.Unsubscribe(topic.String()); token.Wait() && token.Error() != nil {
			return token.Error()
		}
	}

	delete(b.gateways, mac)
	return nil
}

// PublishGatewayRX publishes a RX packet to the MQTT broker.
func (b *Backend) PublishGatewayRX(mac lorawan.EUI64, rxPacket gw.RXPacketBytes) error {
	return b.publish(mac, b.UplinkTemplate, rxPacket)
}

// PublishGatewayStats publishes a GatewayStatsPacket to the MQTT broker.
func (b *Backend) PublishGatewayStats(mac lorawan.EUI64, stats gw.GatewayStatsPacket) error {
	return b.publish(mac, b.StatsTemplate, stats)
}

// PublishGatewayTXAck publishes a TX ack to the MQTT broker.
func (b *Backend) PublishGatewayTXAck(mac lorawan.EUI64, ack gw.TXAck) error {
	return b.publish(mac, b.AckTemplate, ack)
}

func (b *Backend) publish(mac lorawan.EUI64, topicTemplate *template.Template, v interface{}) error {
	topic := bytes.NewBuffer(nil)
	if err := topicTemplate.Execute(topic, struct{ MAC lorawan.EUI64 }{mac}); err != nil {
		return errors.Wrap(err, "execute template error")
	}

	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"topic": topic.String(),
		"qos":   b.config.QOS,
	}).Info("backend: publishing packet")
	if token := b.conn.Publish(topic.String(), b.config.QOS, false, bytes); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (b *Backend) txPacketHandler(c mqtt.Client, msg mqtt.Message) {
	log.WithField("topic", msg.Topic()).Info("backend: downlink packet received")
	var txPacket gw.TXPacketBytes
	if err := json.Unmarshal(msg.Payload(), &txPacket); err != nil {
		log.WithError(err).Error("backend: decode tx packet error")
		return
	}
	b.txPacketChan <- txPacket
}

func (b *Backend) configPacketHandler(c mqtt.Client, msg mqtt.Message) {
	log.WithField("topic", msg.Topic()).Info("backend: config packet received")
	var configPacket gw.GatewayConfigPacket
	if err := json.Unmarshal(msg.Payload(), &configPacket); err != nil {
		log.WithError(err).Error("backend: decode config packet error")
		return
	}
	b.configPacketChan <- configPacket
}

func (b *Backend) onConnected(c mqtt.Client) {
	defer b.mutex.RUnlock()
	b.mutex.RLock()

	log.Info("backend: connected to mqtt broker")

	if len(b.gateways) > 0 {
		// downlink
		for {
			topics := make(map[string]byte)
			for k := range b.gateways {
				topic := bytes.NewBuffer(nil)
				if err := b.DownlinkTemplate.Execute(topic, struct{ MAC lorawan.EUI64 }{k}); err != nil {
					log.WithError(err).Error("backend: execute downlink template error")
					time.Sleep(time.Second)
					continue
				}
				topics[topic.String()] = b.config.QOS

				log.WithFields(log.Fields{
					"topic": topic.String(),
					"qos":   b.config.QOS,
				}).Info("backend: subscribing to topic")
			}
			if token := b.conn.SubscribeMultiple(topics, b.txPacketHandler); token.Wait() && token.Error() != nil {
				log.WithField("topic_count", len(topics)).Errorf("backend: subscribe multiple failed: %s", token.Error())
				time.Sleep(time.Second)
				continue
			}
			break
		}

		// config
		for {
			topics := make(map[string]byte)
			for k := range b.gateways {
				topic := bytes.NewBuffer(nil)
				if err := b.ConfigTemplate.Execute(topic, struct{ MAC lorawan.EUI64 }{k}); err != nil {
					log.WithError(err).Error("backend: execute config template error")
					time.Sleep(time.Second)
					continue
				}
				topics[topic.String()] = b.config.QOS

				log.WithFields(log.Fields{
					"topic": topic.String(),
					"qos":   b.config.QOS,
				}).Info("backend: subscribing to topic")
			}
			if token := b.conn.SubscribeMultiple(topics, b.configPacketHandler); token.Wait() && token.Error() != nil {
				log.WithField("topic_count", len(topics)).Errorf("backend: subscribe multiple failed: %s", token.Error())
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}
}

func (b *Backend) onConnectionLost(c mqtt.Client, reason error) {
	log.WithError(reason).Error("backend: mqtt connection error")
}

// newTLSConfig returns the TLS configuration.
func newTLSConfig(cafile, certFile, certKeyFile string) (*tls.Config, error) {
	// Here are three valid options:
	//   - Only CA
	//   - TLS cert + key
	//   - CA, TLS cert + key

	if cafile == "" && certFile == "" && certKeyFile == "" {
		log.Info("backend: TLS config is empty")
		return nil, nil
	}

	tlsConfig := &tls.Config{}

	// Import trusted certificates from CAfile.pem.
	if cafile != "" {
		cacert, err := ioutil.ReadFile(cafile)
		if err != nil {
			log.Errorf("backend: couldn't load cafile: %s", err)
			return nil, err
		}
		certpool := x509.NewCertPool()
		certpool.AppendCertsFromPEM(cacert)

		tlsConfig.RootCAs = certpool // RootCAs = certs used to verify server cert.
	}

	// Import certificate and the key
	if certFile != "" && certKeyFile != "" {
		kp, err := tls.LoadX509KeyPair(certFile, certKeyFile)
		if err != nil {
			log.Errorf("backend: couldn't load MQTT TLS key pair: %s", err)
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{kp}
	}

	return tlsConfig, nil
}
