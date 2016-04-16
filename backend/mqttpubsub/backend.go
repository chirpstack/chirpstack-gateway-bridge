package mqttpubsub

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/brocaar/loraserver/models"
	"github.com/brocaar/lorawan"
	"github.com/eclipse/paho.mqtt.golang"
)

// Backend implements a MQTT pub-sub backend.
type Backend struct {
	conn         mqtt.Client
	txPacketChan chan models.TXPacket
	gateways     map[lorawan.EUI64]struct{}
	mutex        sync.RWMutex
}

// NewBackend creates a new Backend.
func NewBackend(server, username, password string) (*Backend, error) {
	b := Backend{
		txPacketChan: make(chan models.TXPacket),
		gateways:     make(map[lorawan.EUI64]struct{}),
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(server)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetOnConnectHandler(b.onConnected)
	opts.SetConnectionLostHandler(b.onConnectionLost)

	log.WithField("server", server).Info("backend/mqttpubsub: connecting to MQTT server")
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

// TXPacketChan returns the TXPacket channel.
func (b *Backend) TXPacketChan() chan models.TXPacket {
	return b.txPacketChan
}

// SubscribeGatewayTX subscribes the backend to the gateway TXPacket
// topic (packets the gateway needs to transmit).
func (b *Backend) SubscribeGatewayTX(mac lorawan.EUI64) error {
	defer b.mutex.Unlock()
	b.mutex.Lock()

	topic := fmt.Sprintf("gateway/%s/tx", mac.String())
	log.WithField("topic", topic).Info("backend/mqttpubsub: subscribing to topic")
	if token := b.conn.Subscribe(topic, 0, b.txPacketHandler); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	b.gateways[mac] = struct{}{}
	return nil
}

// UnSubscribeGatewayTX unsubscribes the backend from the gateway TXPacket
// topic.
func (b *Backend) UnSubscribeGatewayTX(mac lorawan.EUI64) error {
	defer b.mutex.Unlock()
	b.mutex.Lock()

	topic := fmt.Sprintf("gateway/%s/tx", mac.String())
	log.WithField("topic", topic).Info("backend/mqttpubsub: unsubscribing from topic")
	if token := b.conn.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	delete(b.gateways, mac)
	return nil
}

// PublishGatewayRX publishes a RX packet to the MQTT broker.
func (b *Backend) PublishGatewayRX(mac lorawan.EUI64, rxPacket models.RXPacket) error {
	topic := fmt.Sprintf("gateway/%s/rx", mac.String())
	return b.publish(topic, rxPacket)
}

// PublishGatewayStats publishes a GatewayStatsPacket to the MQTT broker.
func (b *Backend) PublishGatewayStats(mac lorawan.EUI64, stats models.GatewayStatsPacket) error {
	topic := fmt.Sprintf("gateway/%s/stats", mac.String())
	return b.publish(topic, stats)
}

func (b *Backend) publish(topic string, v interface{}) error {
	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	log.WithField("topic", topic).Info("backend/mqttpubsub: publishing message")
	if token := b.conn.Publish(topic, 0, false, bytes); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (b *Backend) txPacketHandler(c mqtt.Client, msg mqtt.Message) {
	log.WithField("topic", msg.Topic()).Info("backend/mqttpubsub: message received")
	var txPacket models.TXPacket
	if err := json.Unmarshal(msg.Payload(), &txPacket); err != nil {
		log.Errorf("backend/mqttpubsub: could not decode TXPacket: %s", err)
		return
	}
	b.txPacketChan <- txPacket
}

func (b *Backend) onConnected(c mqtt.Client) {
	defer b.mutex.RUnlock()
	b.mutex.RLock()

	log.Info("backend/mqttpubsub: connected to mqtt server")
	if len(b.gateways) > 0 {
		for {
			log.WithField("topic_count", len(b.gateways)).Info("Backend/mqttpubsub: re-registering to gateway topics")
			topics := make(map[string]byte)
			for k := range b.gateways {
				topics[fmt.Sprintf("gateway/%s/tx", k)] = 0
			}
			if token := b.conn.SubscribeMultiple(topics, b.txPacketHandler); token.Wait() && token.Error() != nil {
				log.WithField("topic_count", len(topics)).Errorf("backend/mqttpubsub: subscribe multiple failed: %s", token.Error())
				time.Sleep(time.Second)
				continue
			}
			return
		}
	}
}

func (b *Backend) onConnectionLost(c mqtt.Client, reason error) {
	log.Errorf("backend/mqttpubsub: mqtt connection error: %s", reason)
}
