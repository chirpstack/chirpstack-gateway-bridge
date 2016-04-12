package mqttpubsub

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/brocaar/loraserver/models"
	"github.com/brocaar/lorawan"
	"github.com/eclipse/paho.mqtt.golang"
)

// Backend implements a MQTT pub-sub backend.
type Backend struct {
	conn         mqtt.Client
	txPacketChan chan models.TXPacket
}

// NewBackend creates a new Backend.
func NewBackend(server, username, password string) (*Backend, error) {
	b := Backend{
		txPacketChan: make(chan models.TXPacket),
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(server)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetClientID("lora-semtech-bridge")

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
	topic := fmt.Sprintf("gateway/%s/tx", mac.String())
	log.WithField("topic", topic).Info("backend/mqttpubsub: subscribing to topic")
	if token := b.conn.Subscribe(topic, 0, b.txPacketHandler); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// UnSubscribeGatewayTX unsubscribes the backend from the gateway TXPacket
// topic.
func (b *Backend) UnSubscribeGatewayTX(mac lorawan.EUI64) error {
	topic := fmt.Sprintf("gateway/%s/tx", mac.String())
	log.WithField("topic", topic).Info("backend/mqttpubsub: unsubscribing from topic")
	if token := b.conn.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		return token.Error()
	}
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
