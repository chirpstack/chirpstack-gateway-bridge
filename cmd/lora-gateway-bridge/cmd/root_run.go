package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brocaar/lora-gateway-bridge/internal/backend/mqttpubsub"
	"github.com/brocaar/lora-gateway-bridge/internal/config"
	"github.com/brocaar/lora-gateway-bridge/internal/gateway"
	"github.com/brocaar/lorawan"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, args []string) error {
	log.SetLevel(log.Level(uint8(config.C.General.LogLevel)))

	log.WithFields(log.Fields{
		"version": version,
		"docs":    "https://www.loraserver.io/lora-gateway-bridge/",
	}).Info("starting LoRa Gateway Bridge")

	// always subscribe to managed configurations so that when an invalid
	// configuration terminates the packet-forwarder process, it can be
	// restarted by a valid configuration update
	for i := range config.C.PacketForwarder.Configuration {
		config.C.Backend.MQTT.AlwaysSubscribeMACs = append(config.C.Backend.MQTT.AlwaysSubscribeMACs, config.C.PacketForwarder.Configuration[i].MAC)
	}

	var pubsub *mqttpubsub.Backend
	for {
		var err error
		pubsub, err = mqttpubsub.NewBackend(config.C.Backend.MQTT)
		if err == nil {
			break
		}

		log.Errorf("could not setup mqtt backend, retry in 2 seconds: %s", err)
		time.Sleep(2 * time.Second)
	}
	defer pubsub.Close()

	onNew := func(mac lorawan.EUI64) error {
		return pubsub.SubscribeGatewayTopics(mac)
	}

	onDelete := func(mac lorawan.EUI64) error {
		return pubsub.UnSubscribeGatewayTopics(mac)
	}

	gw, err := gateway.NewBackend(config.C.PacketForwarder.UDPBind, onNew, onDelete, config.C.PacketForwarder.SkipCRCCheck, config.C.PacketForwarder.Configuration)
	if err != nil {
		log.Fatalf("could not setup gateway backend: %s", err)
	}
	defer gw.Close()

	go func() {
		for rxPacket := range gw.RXPacketChan() {
			if err := pubsub.PublishGatewayRX(rxPacket.RXInfo.MAC, rxPacket); err != nil {
				log.WithError(err).Error("publish uplink message error")
			}
		}
	}()

	go func() {
		for stats := range gw.StatsChan() {
			if err := pubsub.PublishGatewayStats(stats.MAC, stats); err != nil {
				log.WithError(err).Error("publish gateway stats message error")
			}
		}
	}()

	go func() {
		for txPacket := range pubsub.TXPacketChan() {
			if err := gw.Send(txPacket); err != nil {
				log.WithError(err).Error("send downlink packet error")
			}
		}
	}()

	go func() {
		for txAck := range gw.TXAckChan() {
			if err := pubsub.PublishGatewayTXAck(txAck.MAC, txAck); err != nil {
				log.WithError(err).Error("publish downlink ack error")
			}
		}
	}()

	go func() {
		for configPacket := range pubsub.ConfigPacketChan() {
			if err := gw.ApplyConfiguration(configPacket); err != nil {
				log.WithError(err).Error("apply configuration error")
			}
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("signal", <-sigChan).Info("signal received")
	log.Warning("shutting down server")
	return nil
}
