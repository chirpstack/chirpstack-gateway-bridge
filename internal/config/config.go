package config

import (
	"time"
)

// Config defines the configuration structure.
type Config struct {
	General struct {
		LogLevel    int  `mapstructure:"log_level"`
		LogToSyslog bool `mapstructure:"log_to_syslog"`
	} `mapstructure:"general"`

	Filters struct {
		NetIDs   []string    `mapstructure:"net_ids"`
		JoinEUIs [][2]string `mapstructure:"join_euis"`
	} `mapstructure:"filters"`

	Backend struct {
		Type string `mapstructure:"type"`

		SemtechUDP struct {
			UDPBind      string `mapstructure:"udp_bind"`
			SkipCRCCheck bool   `mapstructure:"skip_crc_check"`
			FakeRxTime   bool   `mapstructure:"fake_rx_time"`
		} `mapstructure:"semtech_udp"`

		BasicStation struct {
			Bind             string        `mapstructure:"bind"`
			TLSCert          string        `mapstructure:"tls_cert"`
			TLSKey           string        `mapstructure:"tls_key"`
			CACert           string        `mapstructure:"ca_cert"`
			StatsInterval    time.Duration `mapstructure:"stats_interval"`
			PingInterval     time.Duration `mapstructure:"ping_interval"`
			TimesyncInterval time.Duration `mapstructure:"timesync_interval"`
			ReadTimeout      time.Duration `mapstructure:"read_timeout"`
			WriteTimeout     time.Duration `mapstructure:"write_timeout"`
			// TODO: remove Filters in the next major release, use global filters instead
			Filters struct {
				NetIDs   []string    `mapstructure:"net_ids"`
				JoinEUIs [][2]string `mapstructure:"join_euis"`
			} `mapstructure:"filters"`
			Region        string                     `mapstructure:"region"`
			FrequencyMin  uint32                     `mapstructure:"frequency_min"`
			FrequencyMax  uint32                     `mapstructure:"frequency_max"`
			Concentrators []BasicStationConcentrator `mapstructure:"concentrators"`
		} `mapstructure:"basic_station"`

		Concentratord struct {
			EventURL   string `mapstructure:"event_url"`
			CommandURL string `mapstructure:"command_url"`
			CRCCheck   bool   `mapstructure:"crc_check"`
		} `mapstructure:"concentratord"`
	} `mapstructure:"backend"`

	Integration struct {
		Marshaler string `mapstructure:"marshaler"`

		MQTT struct {
			EventTopicTemplate      string        `mapstructure:"event_topic_template"`
			CommandTopicTemplate    string        `mapstructure:"command_topic_template"`
			StateTopicTemplate      string        `mapstructure:"state_topic_template"`
			StateRetained           bool          `mapstructure:"state_retained"`
			KeepAlive               time.Duration `mapstructure:"keep_alive"`
			MaxReconnectInterval    time.Duration `mapstructure:"max_reconnect_interval"`
			TerminateOnConnectError bool          `mapstructure:"terminate_on_connect_error"`
			MaxTokenWait            time.Duration `mapstructure:"max_token_wait"`

			Auth struct {
				Type string `mapstructure:"type"`

				Generic struct {
					Server       string   `mapstructure:"server"`
					Servers      []string `mapstructure:"servers"`
					Username     string   `mapstructure:"username"`
					Password     string   `mapstrucure:"password"`
					CACert       string   `mapstructure:"ca_cert"`
					TLSCert      string   `mapstructure:"tls_cert"`
					TLSKey       string   `mapstructure:"tls_key"`
					QOS          uint8    `mapstructure:"qos"`
					CleanSession bool     `mapstructure:"clean_session"`
					ClientID     string   `mapstructure:"client_id"`
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

				AzureIoTHub struct {
					DeviceConnectionString string        `mapstructure:"device_connection_string"`
					DeviceID               string        `mapstructure:"device_id"`
					Hostname               string        `mapstructure:"hostname"`
					DeviceKey              string        `mapstructure:"-"`
					SASTokenExpiration     time.Duration `mapstructure:"sas_token_expiration"`
					TLSCert                string        `mapstructure:"tls_cert"`
					TLSKey                 string        `mapstructure:"tls_key"`
				} `mapstructure:"azure_iot_hub"`
			} `mapstructure:"auth"`
		} `mapstructure:"mqtt"`
	} `mapstructure:"integration"`

	Metrics struct {
		Prometheus struct {
			EndpointEnabled bool   `mapstructure:"endpoint_enabled"`
			Bind            string `mapstructure:"bind"`
		} `mapstructure:"prometheus"`
	} `mapstructure:"metrics"`

	MetaData struct {
		Static  map[string]string `mapstructure:"static"`
		Dynamic struct {
			ExecutionInterval    time.Duration     `mapstructure:"execution_interval"`
			MaxExecutionDuration time.Duration     `mapstructure:"max_execution_duration"`
			SplitDelimiter       string            `mapstructure:"split_delimiter"`
			Commands             map[string]string `mapstructure:"commands"`
		} `mapstructure:"dynamic"`
	} `mapstructure:"meta_data"`

	Commands struct {
		Commands map[string]struct {
			MaxExecutionDuration time.Duration `mapstructure:"max_execution_duration"`
			Command              string        `mapstructure:"command"`
		} `mapstructure:"commands"`
	} `mapstructure:"commands"`
}

// BasicStationConcentrator holds the configuration for a BasicStation concentrator.
type BasicStationConcentrator struct {
	MultiSF BasicStationConcentratorMultiSF `mapstructure:"multi_sf"`
	LoRaSTD BasicStationConcentratorLoRaSTD `mapstructure:"lora_std"`
	FSK     BasicStationConcentratorFSK     `mapstructure:"fsk"`
}

// BasicStationConcentratorMultiSF holds the multi-SF channels.
type BasicStationConcentratorMultiSF struct {
	Frequencies []uint32 `mapstructure:"frequencies"`
}

// BasicStationConcentratorLoRaSTD holds the LoRa STD config.
type BasicStationConcentratorLoRaSTD struct {
	Frequency       uint32 `mapstructure:"frequency"`
	Bandwidth       uint32 `mapstrcuture:"bandwidth"`
	SpreadingFactor uint32 `mapstructure:"spreading_factor"`
}

// BasicStationConcentratorFSK holds the FSK config.
type BasicStationConcentratorFSK struct {
	Frequency uint32 `mapstructure:"frequency"`
}

// C holds the global configuration.
var C Config
