package config

import "time"

// Config defines the configuration structure.
type Config struct {
	General struct {
		LogLevel int `mapstructure:"log_level"`
	}

	Backend struct {
		SemtechUDP struct {
			UDPBind       string `mapstructure:"udp_bind"`
			SkipCRCCheck  bool   `mapstructure:"skip_crc_check"`
			Configuration []struct {
				GatewayID      string `mapstructure:"gateway_id"`
				BaseFile       string `mapstructure:"base_file"`
				OutputFile     string `mapstructure:"output_file"`
				RestartCommand string `mapstructure:"restart_command"`
			} `mapstructure:"configuration"`
		} `mapstructure:"semtech_udp"`
	} `mapstructure:"backend"`

	Integration struct {
		Marshaler string `mapstructure:"marshaler"`

		MQTT struct {
			EventTopicTemplate   string `mapstructure:"event_topic_template"`
			CommandTopicTemplate string `mapstructure:"command_topic_template"`

			Auth struct {
				Type string `mapstructure:"type"`

				Generic struct {
					Server               string        `mapstructure:"server"`
					Username             string        `mapstructure:"username"`
					Password             string        `mapstrucure:"password"`
					CACert               string        `mapstructure:"ca_cert"`
					TLSCert              string        `mapstructure:"tls_cert"`
					TLSKey               string        `mapstructure:"tls_key"`
					QOS                  uint8         `mapstructure:"qos"`
					CleanSession         bool          `mapstructure:"clean_session"`
					ClientID             string        `mapstructure:"client_id"`
					MaxReconnectInterval time.Duration `mapstructure:"max_reconnect_interval"`
				} `mapstructure:"generic"`

				GCPCloudIoTCore struct {
					Server        string        `mapstructure:"server"`
					DeviceID      string        `mapstructure:"device_id"`
					ProjectID     string        `mapstructure:"project_id"`
					CloudRegion   string        `mapstructure:"cloud_region"`
					RegistryID    string        `mapstructure:"registry_id"`
					JWTExpiration time.Duration `mapstructure:"jwt_expiration"`
					JWTKeyFile    string        `mapstructure:"jwt_key_file"`
				} `mapstructure:"gcp_cloud_iot_core"`
			} `mapstructure:"auth"`
		} `mapstructure:"mqtt"`
	} `mapstructure:"integration"`

	Metrics struct {
		Prometheus struct {
			EndpointEnabled bool   `mapstructure:"endpoint_enabled"`
			Bind            string `mapstructure:"bind"`
		}
	}
}

// C holds the global configuration.
var C Config
