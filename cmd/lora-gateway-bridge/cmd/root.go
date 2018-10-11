package cmd

import (
	"bytes"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/brocaar/lora-gateway-bridge/internal/config"
)

var cfgFile string // config file
var version string

var rootCmd = &cobra.Command{
	Use:   "lora-gateway-bridge",
	Short: "abstracts the packet_forwarder protocol into JSON over MQTT",
	Long: `LoRa Gateway Bridge abstracts the packet_forwarder protocol into JSON over MQTT
	> documentation & support: https://www.loraserver.io/lora-gateway-bridge
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
	viper.BindEnv("backend.mqtt.auth.generic.server", "MQTT_SERVER")
	viper.BindEnv("backend.mqtt.auth.generic.username", "MQTT_USERNAME")
	viper.BindEnv("backend.mqtt.auth.generic.password", "MQTT_PASSWORD")
	viper.BindEnv("backend.mqtt.auth.generic.ca_cert", "MQTT_CA_CERT")
	viper.BindEnv("backend.mqtt.auth.generic.tls_cert", "MQTT_TLS_CERT")
	viper.BindEnv("backend.mqtt.auth.generic.tls_key", "MQTT_TLS_KEY")

	// for backwards compatibility
	viper.BindPFlag("general.log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("packet_forwarder.udp_bind", rootCmd.PersistentFlags().Lookup("udp-bind"))
	viper.BindPFlag("packet_forwarder.skip_crc_check", rootCmd.PersistentFlags().Lookup("skip-crc-check"))
	viper.BindPFlag("backend.mqtt.auth.generic.server", rootCmd.PersistentFlags().Lookup("mqtt-server"))
	viper.BindPFlag("backend.mqtt.auth.generic.username", rootCmd.PersistentFlags().Lookup("mqtt-username"))
	viper.BindPFlag("backend.mqtt.auth.generic.password", rootCmd.PersistentFlags().Lookup("mqtt-password"))
	viper.BindPFlag("backend.mqtt.auth.generic.ca_cert", rootCmd.PersistentFlags().Lookup("mqtt-ca-cert"))
	viper.BindPFlag("backend.mqtt.auth.generic.tls_cert", rootCmd.PersistentFlags().Lookup("mqtt-tls-cert"))
	viper.BindPFlag("backend.mqtt.auth.generic.tls_key", rootCmd.PersistentFlags().Lookup("mqtt-tls-key"))

	// default values
	viper.SetDefault("packet_forwarder.udp_bind", "0.0.0.0:1700")

	viper.SetDefault("backend.mqtt.uplink_topic_template", "gateway/{{ .MAC }}/rx")
	viper.SetDefault("backend.mqtt.downlink_topic_template", "gateway/{{ .MAC }}/tx")
	viper.SetDefault("backend.mqtt.stats_topic_template", "gateway/{{ .MAC }}/stats")
	viper.SetDefault("backend.mqtt.ack_topic_template", "gateway/{{ .MAC }}/ack")
	viper.SetDefault("backend.mqtt.config_topic_template", "gateway/{{ .MAC }}/config")

	viper.SetDefault("backend.mqtt.marshaler", "v2_json")
	viper.SetDefault("backend.mqtt.auth.type", "generic")

	viper.SetDefault("backend.mqtt.auth.generic.server", "tcp://127.0.0.1:1883")
	viper.SetDefault("backend.mqtt.auth.generic.clean_session", true)
	viper.SetDefault("backend.mqtt.auth.generic.max_reconnect_interval", 10*time.Minute)

	viper.SetDefault("backend.mqtt.auth.gcp_cloud_iot_core.server", "ssl://mqtt.googleapis.com:8883")
	viper.SetDefault("backend.mqtt.auth.gcp_cloud_iot_core.jwt_expiration", time.Hour*24)

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

	for i := range config.C.PacketForwarder.Configuration {
		if err := config.C.PacketForwarder.Configuration[i].MAC.UnmarshalText([]byte(config.C.PacketForwarder.Configuration[i].MACString)); err != nil {
			log.WithError(err).Fatal("unmarshal config error")
		}
	}

	// migrate config
	if v := config.C.Backend.MQTT.Server; v != "" {
		config.C.Backend.MQTT.Auth.Generic.Server = v
	}
	if v := config.C.Backend.MQTT.Username; v != "" {
		config.C.Backend.MQTT.Auth.Generic.Username = v
	}
	if v := config.C.Backend.MQTT.Password; v != "" {
		config.C.Backend.MQTT.Auth.Generic.Password = v
	}
	if v := config.C.Backend.MQTT.Password; v != "" {
		config.C.Backend.MQTT.Auth.Generic.Password = v
	}
	if v := config.C.Backend.MQTT.CACert; v != "" {
		config.C.Backend.MQTT.Auth.Generic.CACert = v
	}
	if v := config.C.Backend.MQTT.TLSCert; v != "" {
		config.C.Backend.MQTT.Auth.Generic.TLSCert = v
	}
	if v := config.C.Backend.MQTT.TLSKey; v != "" {
		config.C.Backend.MQTT.Auth.Generic.TLSKey = v
	}
	if v := config.C.Backend.MQTT.QOS; v != 0 {
		config.C.Backend.MQTT.Auth.Generic.QOS = v
	}
	if v := config.C.Backend.MQTT.CleanSession; v {
		config.C.Backend.MQTT.Auth.Generic.CleanSession = v
	}
	if v := config.C.Backend.MQTT.ClientID; v != "" {
		config.C.Backend.MQTT.Auth.Generic.ClientID = v
	}
	if v := config.C.Backend.MQTT.MaxReconnectInterval; v != 0 {
		config.C.Backend.MQTT.Auth.Generic.MaxReconnectInterval = v
	}
}
