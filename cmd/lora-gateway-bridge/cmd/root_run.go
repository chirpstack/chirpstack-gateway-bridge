package cmd

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/brocaar/lora-gateway-bridge/internal/backend/mqtt"
	"github.com/brocaar/lora-gateway-bridge/internal/config"
	"github.com/brocaar/lora-gateway-bridge/internal/gateway/semtech"
	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
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

	go func() {
		if !config.C.Metrics.Prometheus.EndpointEnabled {
			return
		}

		log.WithFields(log.Fields{
			"bind": config.C.Metrics.Prometheus.Bind,
		}).Info("starting prometheus metrics server")

		server := http.Server{
			Handler: promhttp.Handler(),
			Addr:    config.C.Metrics.Prometheus.Bind,
		}

		go func() {
			err := server.ListenAndServe()
			log.WithError(err).Error("prometheus metrics server error")
		}()
	}()

	return runV3(cmd, args)
}

func runV3(cmd *cobra.Command, args []string) error {
	backend, err := mqtt.NewBackend(config.C.Backend.MQTT)
	if err != nil {
		return errors.Wrap(err, "new mqtt backend error")
	}
	defer backend.Close()

	onNew := func(gatewayID lorawan.EUI64) error {
		return backend.SubscribeGateway(gatewayID)
	}

	onDelete := func(gatewayID lorawan.EUI64) error {
		return backend.UnsubscribeGateway(gatewayID)
	}

	gateway, err := semtech.NewBackend(config.C.PacketForwarder.UDPBind, onNew, onDelete, config.C.PacketForwarder.Configuration)
	if err != nil {
		return errors.Wrap(err, "new gateway backend error")
	}
	defer gateway.Close()

	go func() {
		for uplinkFrame := range gateway.UplinkFrameChan() {
			var gatewayID lorawan.EUI64
			copy(gatewayID[:], uplinkFrame.RxInfo.GatewayId)

			// publish/wait in a go routine because with QoS > 0 it can take hundreds of milliseconds
			// and we don't want to block other uplink frames while waiting for the PUBACK
			go func(uplinkFrame gw.UplinkFrame) {
				if err := backend.PublishUplinkFrame(gatewayID, uplinkFrame); err != nil {
					log.WithError(err).Error("publish uplink frame error")
				}
			}(uplinkFrame)
		}
	}()

	go func() {
		for stats := range gateway.GatewayStatsChan() {
			var gatewayID lorawan.EUI64
			copy(gatewayID[:], stats.GatewayId)
			if err := backend.PublishGatewayStats(gatewayID, stats); err != nil {
				log.WithError(err).Error("publish gateway stats error")
			}
		}
	}()

	go func() {
		for txAck := range gateway.DownlinkTXAckChan() {
			var gatewayID lorawan.EUI64
			copy(gatewayID[:], txAck.GatewayId)
			if err := backend.PublishDownlinkTXAck(gatewayID, txAck); err != nil {
				log.WithError(err).Error("publish downlink tx ack error")
			}
		}
	}()

	go func() {
		for downlinkFrame := range backend.DownlinkFrameChan() {
			if err := gateway.SendDownlinkFrame(downlinkFrame); err != nil {
				log.WithError(err).Error("send downlink udp packet error")
			}
		}
	}()

	go func() {
		for gatewayConfig := range backend.GatewayConfigurationChan() {
			if err := gateway.ApplyConfiguration(gatewayConfig); err != nil {
				log.WithError(err).Error("apply packet-forwarder configuration error")
			}
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("signal", <-sigChan).Info("signal received")
	log.Warning("shutting down server")

	return nil
}
