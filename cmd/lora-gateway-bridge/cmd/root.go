package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/brocaar/lora-gateway-bridge/internal/backend/mqttpubsub"
	"github.com/brocaar/lora-gateway-bridge/internal/config"
	"github.com/brocaar/lora-gateway-bridge/internal/gateway"
	"github.com/brocaar/lorawan"
)

var cfgFile string // config file
var version string

var rootCmd = &cobra.Command{
	Use:   "lora-gateway-bridge",
	Short: "abstracts the packet_forwarder protocol into JSON over MQTT",
	Long: `LoRa Gateway Bridge abstracts the packet_forwarder protocol into JSON over MQTT
	> documentation & support: https://docs.loraserver.io/lora-gateway-bridge
	> source & copyright information: https://github.com/brocaar/lora-gateway-bridge`,
	RunE: run,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "path to configuration file (optional)")
	rootCmd.PersistentFlags().Int("log-level", 4, "debug=5, info=4, error=2, fatal=1, panic=0")

	// for backwards compatibility
	rootCmd.PersistentFlags().String("udp-bind", "", "")
	rootCmd.PersistentFlags().String("mqtt-server", "", "")
	rootCmd.PersistentFlags().String("mqtt-username", "", "")
	rootCmd.PersistentFlags().String("mqtt-password", "", "")
	rootCmd.PersistentFlags().String("mqtt-ca-cert", "", "")
	rootCmd.PersistentFlags().String("mqtt-tls-cert", "", "")
	rootCmd.PersistentFlags().String("mqtt-tls-key", "", "")
	rootCmd.PersistentFlags().Bool("skip-crc-check", false, "")
	rootCmd.PersistentFlags().MarkHidden("udp-bind")
	rootCmd.PersistentFlags().MarkHidden("mqtt-server")
	rootCmd.PersistentFlags().MarkHidden("mqtt-username")
	rootCmd.PersistentFlags().MarkHidden("mqtt-password")
	rootCmd.PersistentFlags().MarkHidden("mqtt-ca-cert")
	rootCmd.PersistentFlags().MarkHidden("mqtt-tls-cert")
	rootCmd.PersistentFlags().MarkHidden("mqtt-tls-key")
	rootCmd.PersistentFlags().MarkHidden("skip-crc-check")

	// for backwards compatibility
	viper.BindEnv("general.log_level", "LOG_LEVEL")
	viper.BindEnv("packet_forwarder.udp_bind", "UDP_BIND")
	viper.BindEnv("packet_forwarder.skip_crc_check", "SKIP_CRC_CHECK")
	viper.BindEnv("backend.mqtt.server", "MQTT_SERVER")
	viper.BindEnv("backend.mqtt.username", "MQTT_USERNAME")
	viper.BindEnv("backend.mqtt.password", "MQTT_PASSWORD")
	viper.BindEnv("backend.mqtt.ca_cert", "MQTT_CA_CERT")
	viper.BindEnv("backend.mqtt.tls_cert", "MQTT_TLS_CERT")
	viper.BindEnv("backend.mqtt.tls_key", "MQTT_TLS_KEY")

	// for backwards compatibility
	viper.BindPFlag("general.log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("packet_forwarder.udp_bind", rootCmd.PersistentFlags().Lookup("udp-bind"))
	viper.BindPFlag("packet_forwarder.skip_crc_check", rootCmd.PersistentFlags().Lookup("skip-crc-check"))
	viper.BindPFlag("backend.mqtt.server", rootCmd.PersistentFlags().Lookup("mqtt-server"))
	viper.BindPFlag("backend.mqtt.username", rootCmd.PersistentFlags().Lookup("mqtt-username"))
	viper.BindPFlag("backend.mqtt.password", rootCmd.PersistentFlags().Lookup("mqtt-password"))
	viper.BindPFlag("backend.mqtt.ca_cert", rootCmd.PersistentFlags().Lookup("mqtt-ca-cert"))
	viper.BindPFlag("backend.mqtt.tls_cert", rootCmd.PersistentFlags().Lookup("mqtt-tls-cert"))
	viper.BindPFlag("backend.mqtt.tls_key", rootCmd.PersistentFlags().Lookup("mqtt-tls-key"))

	// default values
	viper.SetDefault("packet_forwarder.udp_bind", "0.0.0.0:1700")

	viper.SetDefault("backend.mqtt.uplink_topic_template", "gateway/{{ .MAC }}/rx")
	viper.SetDefault("backend.mqtt.downlink_topic_template", "gateway/{{ .MAC }}/tx")
	viper.SetDefault("backend.mqtt.stats_topic_template", "gateway/{{ .MAC }}/stats")
	viper.SetDefault("backend.mqtt.ack_topic_template", "gateway/{{ .MAC }}/ack")
	viper.SetDefault("backend.mqtt.server", "tcp://127.0.0.1:1883")
	viper.SetDefault("backend.mqtt.clean_session", true)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
}

// Execute executes the root command.
func Execute(v string) {
	version = v
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func run(cmd *cobra.Command, args []string) error {
	log.SetLevel(log.Level(uint8(config.C.General.LogLevel)))

	log.WithFields(log.Fields{
		"version": version,
		"docs":    "https://docs.loraserver.io/lora-gateway-bridge/",
	}).Info("starting LoRa Gateway Bridge")

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
		return pubsub.SubscribeGatewayTX(mac)
	}

	onDelete := func(mac lorawan.EUI64) error {
		return pubsub.UnSubscribeGatewayTX(mac)
	}

	gw, err := gateway.NewBackend(config.C.PacketForwarder.UDPBind, onNew, onDelete, config.C.PacketForwarder.SkipCRCCheck)
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

	go func() {
		for txAck := range gw.TXAckChan() {
			if err := pubsub.PublishGatewayTXAck(txAck.MAC, txAck); err != nil {
				log.Errorf("could not publish TXAck: %s", err)
			}
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("signal", <-sigChan).Info("signal received")
	log.Warning("shutting down server")
	return nil
}

func initConfig() {
	if cfgFile != "" {
		b, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			log.WithError(err).WithField("config", cfgFile).Fatal("error loading config file")
		}
		viper.SetConfigType("toml")
		if err := viper.ReadConfig(bytes.NewBuffer(b)); err != nil {
			log.WithError(err).WithField("config", cfgFile).Fatal("error loading config file")
		}
	} else {
		viper.SetConfigName("lora-gateway-bridge")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/lora-gateway-bridge")
		viper.AddConfigPath("/etc/lora-gateway-bridge/")
		if err := viper.ReadInConfig(); err != nil {
			switch err.(type) {
			case viper.ConfigFileNotFoundError:
				log.Warning("Deprecation warning! no configuration file found, falling back on environment variables. Update your configuration, see: https://docs.loraserver.io/lora-gateway-bridge/install/config/")
			default:
				log.WithError(err).Fatal("read configuration file error")
			}
		}
	}

	if err := viper.Unmarshal(&config.C); err != nil {
		log.WithError(err).Fatal("unmarshal config error")
	}
}
