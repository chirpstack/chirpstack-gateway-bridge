package main

//go:generate ./doc.sh

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/brocaar/lora-semtech-bridge/backend/mqttpubsub"
	"github.com/brocaar/lora-semtech-bridge/gateway"
	"github.com/brocaar/lorawan"
	"github.com/codegangsta/cli"
)

var version string // set by the compiler

func run(c *cli.Context) {
	pubsub, err := mqttpubsub.NewBackend(c.String("mqtt-server"), c.String("mqtt-username"), c.String("mqtt-password"))
	if err != nil {
		log.Fatalf("could not setup mqtt backend: %s", err)
	}
	defer pubsub.Close()

	onNew := func(mac lorawan.EUI64) error {
		return pubsub.SubscribeGatewayTX(mac)
	}

	onDelete := func(mac lorawan.EUI64) error {
		return pubsub.UnSubscribeGatewayTX(mac)
	}

	gw, err := gateway.NewBackend(c.String("udp-bind"), onNew, onDelete)
	if err != nil {
		log.Fatalf("could not setup gateway backend: %s", err)
	}
	defer gw.Close()

	go func() {
		for rxPacket := range gw.RXPacketChan() {
			if err := pubsub.PublishGatewayRX(rxPacket.RXInfo.MAC, rxPacket); err != nil {
				log.Errorf("could not publish RXPacket: %s", err)
			}
		}
	}()

	go func() {
		for stats := range gw.StatsChan() {
			if err := pubsub.PublishGatewayStats(stats.MAC, stats); err != nil {
				log.Errorf("could not publish GatewayStatsPacket: %s", err)
			}
		}
	}()

	go func() {
		for txPacket := range pubsub.TXPacketChan() {
			if err := gw.Send(txPacket); err != nil {
				log.Errorf("could not send TXPacket: %s", err)
			}
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("signal", <-sigChan).Info("signal received")
	log.Warning("shutting down server")
}

func main() {
	app := cli.NewApp()
	app.Name = "semtech-bridge"
	app.Usage = "Semtech UDP protocol speaking gateway <-> MQTT"
	app.Version = version
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "udp-bind",
			Usage:  "ip:port to bind the UDP listener to",
			Value:  "0.0.0.0:1700",
			EnvVar: "UDP_BIND",
		},
		cli.StringFlag{
			Name:   "mqtt-server",
			Usage:  "MQTT server",
			Value:  "tcp://127.0.0.1:1883",
			EnvVar: "MQTT_SERVER",
		},
		cli.StringFlag{
			Name:   "mqtt-username",
			Usage:  "MQTT username",
			EnvVar: "MQTT_USERNAME",
		},
		cli.StringFlag{
			Name:   "mqtt-password",
			Usage:  "MQTT password",
			EnvVar: "MQTT_PASSWORD",
		},
	}
	app.Run(os.Args)
}
