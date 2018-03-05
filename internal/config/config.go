package config

import "github.com/brocaar/lora-gateway-bridge/internal/backend/mqttpubsub"

// Config defines the configuration structure.
type Config struct {
	General struct {
		LogLevel int `mapstructure:"log_level"`
	}

	PacketForwarder struct {
		UDPBind      string `mapstructure:"udp_bind"`
		SkipCRCCheck bool   `mapstructure:"skip_crc_check"`
	} `mapstructure:"packet_forwarder"`

	Backend struct {
		MQTT mqttpubsub.BackendConfig
	}
}

// C holds the global configuration.
var C Config
