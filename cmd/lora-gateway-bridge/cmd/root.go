package cmd

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"strings"
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

	viper.BindPFlag("general.log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	// default values
	viper.SetDefault("general.log_level", 4)
	viper.SetDefault("backend.type", "semtech_udp")
	viper.SetDefault("backend.semtech_udp.udp_bind", "0.0.0.0:1700")

	viper.SetDefault("backend.basic_station.bind", ":3001")
	viper.SetDefault("backend.basic_station.ping_interval", time.Minute)
	viper.SetDefault("backend.basic_station.read_timeout", time.Minute+(5*time.Second))
	viper.SetDefault("backend.basic_station.write_timeout", time.Second)
	viper.SetDefault("backend.basic_station.filters.net_ids", []string{"000000"})
	viper.SetDefault("backend.basic_station.filters.join_euis", [][2]string{{"0000000000000000", "ffffffffffffffff"}})
	viper.SetDefault("backend.basic_station.region", "EU868")
	viper.SetDefault("backend.basic_station.frequency_min", 863000000)
	viper.SetDefault("backend.basic_station.frequency_max", 870000000)

	viper.SetDefault("integration.marshaler", "protobuf")
	viper.SetDefault("integration.mqtt.auth.type", "generic")

	viper.SetDefault("integration.mqtt.event_topic_template", "gateway/{{ .GatewayID }}/event/{{ .EventType }}")
	viper.SetDefault("integration.mqtt.command_topic_template", "gateway/{{ .GatewayID }}/command/#")

	viper.SetDefault("integration.mqtt.auth.generic.server", "tcp://127.0.0.1:1883")
	viper.SetDefault("integration.mqtt.auth.generic.clean_session", true)
	viper.SetDefault("integration.mqtt.auth.generic.max_reconnect_interval", 10*time.Minute)

	viper.SetDefault("integration.mqtt.auth.gcp_cloud_iot_core.server", "ssl://mqtt.googleapis.com:8883")
	viper.SetDefault("integration.mqtt.auth.gcp_cloud_iot_core.jwt_expiration", time.Hour*24)

	viper.SetDefault("integration.mqtt.auth.azure_iot_hub.sas_token_expiration", 24*time.Hour)

	viper.SetDefault("meta_data.dynamic.execution_interval", time.Minute)
	viper.SetDefault("meta_data.dynamic.max_execution_duration", time.Second)

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
			default:
				log.WithError(err).Fatal("read configuration file error")
			}
		}
	}

	viperBindEnvs(config.C)

	if err := viper.Unmarshal(&config.C); err != nil {
		log.WithError(err).Fatal("unmarshal config error")
	}
}

func viperBindEnvs(iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			tv = strings.ToLower(t.Name)
		}
		if tv == "-" {
			continue
		}

		switch v.Kind() {
		case reflect.Struct:
			viperBindEnvs(v.Interface(), append(parts, tv)...)
		default:
			key := strings.Join(append(parts, tv), ".")
			viper.BindEnv(key)
		}
	}
}
