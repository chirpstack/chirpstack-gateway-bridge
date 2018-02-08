package config

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
		MQTT struct {
			Server   string
			Username string
			Password string
			CACert   string `mapstructure:"ca_cert"`
			TLSCert  string `mapstructure:"tls_cert"`
			TLSKey   string `mapstructure:"tls_key"`
		}
	}
}

// C holds the global configuration.
var C Config
