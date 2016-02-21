package mqttpubsub

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	log "github.com/Sirupsen/logrus"
	"github.com/brocaar/loraserver"
	"github.com/brocaar/lorawan"
)

// Backend implements a MQTT pub-sub backend.
type Backend struct {
	conn         *mqtt.Client
	txPacketChan chan loraserver.TXPacket
}

// NewBackend creates a new Backend.
func NewBackend(server, username, password string) (*Backend, error) {
	b := Backend{
		txPacketChan: make(chan loraserver.TXPacket),
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
func (b *Backend) TXPacketChan() chan loraserver.TXPacket {
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
func (b *Backend) PublishGatewayRX(mac lorawan.EUI64, rxPacket loraserver.RXPacket) error {
	topic := fmt.Sprintf("gateway/%s/rx", mac.String())
	return b.publish(topic, rxPacket)
}

// PublishGatewayStats publishes a GatewayStatsPacket to the MQTT broker.
func (b *Backend) PublishGatewayStats(mac lorawan.EUI64, stats loraserver.GatewayStatsPacket) error {
	topic := fmt.Sprintf("gateway/%s/stats", mac.String())
	return b.publish(topic, stats)
}

func (b *Backend) publish(topic string, v interface{}) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"topic": topic,
	}).Info("backend/mqttpubsub: publishing message")
	if token := b.conn.Publish(topic, 0, false, buf.Bytes()); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (b *Backend) txPacketHandler(c *mqtt.Client, msg mqtt.Message) {
	var txPacket loraserver.TXPacket
	dec := gob.NewDecoder(bytes.NewReader(msg.Payload()))
	if err := dec.Decode(&txPacket); err != nil {
		log.Errorf("backend/mqttpubsub: could not decode TXPacket: %s", err)
		return
	}
	b.txPacketChan <- txPacket
}
