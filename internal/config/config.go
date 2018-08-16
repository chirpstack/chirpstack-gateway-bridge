package config

import (
	"github.com/brocaar/lora-gateway-bridge/internal/backend/mqtt"
	"github.com/brocaar/lora-gateway-bridge/internal/gateway/semtech"
)

// Config defines the configuration structure.
type Config struct {
	General struct {
		LogLevel int `mapstructure:"log_level"`
	}

	PacketForwarder struct {
		UDPBind      string `mapstructure:"udp_bind"`
		SkipCRCCheck bool   `mapstructure:"skip_crc_check"`

		Configuration []semtech.PFConfiguration `mapstructure:"configuration"`
	} `mapstructure:"packet_forwarder"`

	Backend struct {
		MQTT mqtt.BackendConfig
	}
	Metrics struct {
		Prometheus struct {
			EndpointEnabled bool `mapstructure:"endpoint_enabled"`
			Bind            string
		}
	}
}

// C holds the global configuration.
var C Config
