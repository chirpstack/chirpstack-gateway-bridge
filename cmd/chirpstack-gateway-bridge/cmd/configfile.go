package cmd

import (
	"html/template"
	"os"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// when updating this template, don't forget to update config.md!
const configTemplate = `[general]
# debug=5, info=4, warning=3, error=2, fatal=1, panic=0
log_level={{ .General.LogLevel }}

# Log to syslog.
#
# When set to true, log messages are being written to syslog.
log_to_syslog={{ .General.LogToSyslog }}


# Filters.
#
# These can be used to filter LoRaWAN frames to reduce bandwith usage between
# the gateway and ChirpStack Gateway Bridge. Depending the used backend, filtering
# will be performed by the Packet Forwarder or ChirpStack Gateway Bridge.
[filters]

# NetIDs filters.
#
# The configured NetIDs will be used to filter uplink data frames.
# When left blank, no filtering will be performed on NetIDs.
#
# Example:
# net_ids=[
#   "000000",
#   "000001",
# ]
net_ids=[{{ range $index, $elm := .Filters.NetIDs }}
  "{{ $elm }}",{{ end }}
]

# JoinEUI filters.
#
# The configured JoinEUI ranges will be used to filter join-requests.
# When left blank, no filtering will be performed on JoinEUIs.
#
# Example:
# join_euis=[
#   ["0000000000000000", "00000000000000ff"],
#   ["000000000000ff00", "000000000000ffff"],
# ]
join_euis=[{{ range $index, $elm := .Filters.JoinEUIs }}
  ["{{ index $elm 0 }}", "{{ index $elm 1 }}"],{{ end }}
]


# Gateway backend configuration.
[backend]

# Backend type.
#
# Valid options are:
#   * semtech_udp
#   * concentratord
#   * basic_station
type="{{ .Backend.Type }}"


  # Semtech UDP packet-forwarder backend.
  [backend.semtech_udp]

  # ip:port to bind the UDP listener to
  #
  # Example: 0.0.0.0:1700 to listen on port 1700 for all network interfaces.
  # This is the listener to which the packet-forwarder forwards its data
  # so make sure the 'serv_port_up' and 'serv_port_down' from your
  # packet-forwarder matches this port.
  udp_bind = "{{ .Backend.SemtechUDP.UDPBind }}"

  # Skip the CRC status-check of received packets
  #
  # This is only has effect when the packet-forwarder is configured to forward
  # LoRa frames with CRC errors.
  skip_crc_check = {{ .Backend.SemtechUDP.SkipCRCCheck }}

  # Fake RX timestamp.
  #
  # Fake the RX time when the gateway does not have GPS, in which case
  # the time would otherwise be unset.
  fake_rx_time={{ .Backend.SemtechUDP.FakeRxTime }}


  # ChirpStack Concentratord backend.
  [backend.concentratord]

  # Check for CRC OK.
  crc_check={{ .Backend.Concentratord.CRCCheck }}

  # Event API URL.
  event_url="{{ .Backend.Concentratord.EventURL }}"

  # Command API URL.
  command_url="{{ .Backend.Concentratord.CommandURL }}"


  # Basic Station backend.
  [backend.basic_station]

  # ip:port to bind the Websocket listener to.
  bind="{{ .Backend.BasicStation.Bind }}"

  # TLS certificate and key files.
  #
  # When set, the websocket listener will use TLS to secure the connections
  # between the gateways and ChirpStack Gateway Bridge (optional).
  tls_cert="{{ .Backend.BasicStation.TLSCert }}"
  tls_key="{{ .Backend.BasicStation.TLSKey }}"

  # TLS CA certificate.
  #
  # When configured, ChirpStack Gateway Bridge will validate that the client
  # certificate of the gateway has been signed by this CA certificate.
  ca_cert="{{ .Backend.BasicStation.CACert }}"

  # Stats interval.
  #
  # This defines the interval in which the ChirpStack Gateway Bridge forwards
  # the uplink / downlink statistics.
  stats_interval="{{ .Backend.BasicStation.StatsInterval }}"

  # Ping interval.
  ping_interval="{{ .Backend.BasicStation.PingInterval }}"

  # Timesync interval.
  #
  # This defines the interval in which the ChirpStack Gateway Bridge sends
  # a timesync request to the gateway. Setting this to 0 disables sending
  # timesync requests.
  timesync_interval="{{ .Backend.BasicStation.TimesyncInterval }}"

  # Read timeout.
  #
  # This interval must be greater than the configured ping interval.
  read_timeout="{{ .Backend.BasicStation.ReadTimeout }}"

  # Write timeout.
  write_timeout="{{ .Backend.BasicStation.WriteTimeout }}"

  # Region.
  #
  # Please refer to the LoRaWAN Regional Parameters specification
  # for the complete list of common region names.
  region="{{ .Backend.BasicStation.Region }}"

  # Minimal frequency (Hz).
  frequency_min={{ .Backend.BasicStation.FrequencyMin }}

  # Maximum frequency (Hz).
  frequency_max={{ .Backend.BasicStation.FrequencyMax }}

  # Concentrator configuration.
  #
  # This section contains the configuration for the SX1301 concentrator chips.
  # Example:
  # [[backend.basic_station.concentrators]]
  #
  #   # Multi-SF channel configuration.
  #   [backend.basic_station.concentrators.multi_sf]
  #
  #   # Frequencies (Hz).
  #   frequencies=[
  #     868100000,
  #     868300000,
  #     868500000,
  #     867100000,
  #     867300000,
  #     867500000,
  #     867700000,
  #     867900000,
  #   ]
  #
  #   # LoRa STD channel.
  #   [backend.basic_station.concentrators.lora_std]
  #
  #   # Frequency (Hz).
  #   frequency=868300000
  #
  #   # Bandwidth (Hz).
  #   bandwidth=250000
  #
  #   # Spreading factor.
  #   spreading_factor=7
  #
  #   # FSK channel.
  #   [backend.basic_station.concentrators.fsk]
  #
  #   # Frequency (Hz).
  #   frequency=868800000
{{ range $i, $concentrator := .Backend.BasicStation.Concentrators }}
    [[backend.basic_station.concentrators]]
      [backend.basic_station.concentrators.multi_sf]
      frequencies=[{{ range $index, $elm := $concentrator.MultiSF.Frequencies }}
		{{ $elm }},{{ end }}
      ]

      [backend.basic_station.concentrators.lora_std]
      frequency={{ $concentrator.LoRaSTD.Frequency }}
      bandwidth={{ $concentrator.LoRaSTD.Bandwidth }}
      spreading_factor={{ $concentrator.LoRaSTD.SpreadingFactor }}

      [backend.basic_station.concentrators.fsk]
      frequency={{ $concentrator.FSK.Frequency }}
{{ end }}

# Integration configuration.
[integration]
# Payload marshaler.
#
# This defines how the MQTT payloads are encoded. Valid options are:
# * protobuf:  Protobuf encoding
# * json:      JSON encoding (easier for debugging, but less compact than 'protobuf')
marshaler="{{ .Integration.Marshaler }}"

  # MQTT integration configuration.
  [integration.mqtt]
  # Event topic template.
  event_topic_template="{{ .Integration.MQTT.EventTopicTemplate }}"

  # State topic template.
  #
  # States are sent by the gateway as retained MQTT messages (by default)
  # so that the last message will be stored by the MQTT broker. When set to
  # a blank string, this feature will be disabled. This feature is only
  # supported when using the generic authentication type.
  state_topic_template="{{ .Integration.MQTT.StateTopicTemplate }}"

  # Command topic template.
  command_topic_template="{{ .Integration.MQTT.CommandTopicTemplate }}"

  # State retained.
  #
  # By default this value is set to true and states are published as retained
  # MQTT messages. Setting this to false means that states will not be retained
  # by the MQTT broker.
  state_retained={{ .Integration.MQTT.StateRetained }}

  # Keep alive will set the amount of time (in seconds) that the client should
  # wait before sending a PING request to the broker. This will allow the client
  # to know that a connection has not been lost with the server.
  # Valid units are 'ms', 's', 'm', 'h'. Note that these values can be combined, e.g. '24h30m15s'.
  keep_alive="{{ .Integration.MQTT.KeepAlive }}"

  # Maximum interval that will be waited between reconnection attempts when connection is lost.
  # Valid units are 'ms', 's', 'm', 'h'. Note that these values can be combined, e.g. '24h30m15s'.
  max_reconnect_interval="{{ .Integration.MQTT.MaxReconnectInterval }}"

  # Terminate on connect error.
  #
  # When set to true, instead of re-trying to connect, the ChirpStack Gateway Bridge
  # process will be terminated on a connection error.
  terminate_on_connect_error={{ .Integration.MQTT.TerminateOnConnectError }}


  # MQTT authentication.
  [integration.mqtt.auth]
  # Type defines the MQTT authentication type to use.
  #
  # Set this to the name of one of the sections below.
  type="{{ .Integration.MQTT.Auth.Type }}"

    # Generic MQTT authentication.
    [integration.mqtt.auth.generic]
    # MQTT servers.
    #
    # Configure one or multiple MQTT server to connect to. Each item must be in
    # the following format: scheme://host:port where scheme is tcp, ssl or ws.
    servers=[{{ range $index, $elm := .Integration.MQTT.Auth.Generic.Servers }}
      "{{ $elm }}",{{ end }}
    ]

    # Connect with the given username (optional)
    username="{{ .Integration.MQTT.Auth.Generic.Username }}"

    # Connect with the given password (optional)
    password="{{ .Integration.MQTT.Auth.Generic.Password }}"

    # Quality of service level
    #
    # 0: at most once
    # 1: at least once
    # 2: exactly once
    #
    # Note: an increase of this value will decrease the performance.
    # For more information: https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels
    qos={{ .Integration.MQTT.Auth.Generic.QOS }}

    # Clean session
    #
    # Set the "clean session" flag in the connect message when this client
    # connects to an MQTT broker. By setting this flag you are indicating
    # that no messages saved by the broker for this client should be delivered.
    clean_session={{ .Integration.MQTT.Auth.Generic.CleanSession }}

    # Client ID
    #
    # Set the client id to be used by this client when connecting to the MQTT
    # broker. A client id must be no longer than 23 characters. When left blank,
    # a random id will be generated. This requires clean_session=true.
    client_id="{{ .Integration.MQTT.Auth.Generic.ClientID }}"

    # CA certificate file (optional)
    #
    # Use this when setting up a secure connection (when server uses ssl://...)
    # but the certificate used by the server is not trusted by any CA certificate
    # on the server (e.g. when self generated).
    ca_cert="{{ .Integration.MQTT.Auth.Generic.CACert }}"

    # mqtt TLS certificate file (optional)
    tls_cert="{{ .Integration.MQTT.Auth.Generic.TLSCert }}"

    # mqtt TLS key file (optional)
    tls_key="{{ .Integration.MQTT.Auth.Generic.TLSKey }}"


    # Google Cloud Platform Cloud IoT Core authentication.
    #
    # Please note that when using this authentication type, the MQTT topics
    # will be automatically set to match the MQTT topics as expected by
    # Cloud IoT Core.
    [integration.mqtt.auth.gcp_cloud_iot_core]
    # MQTT server.
    server="{{ .Integration.MQTT.Auth.GCPCloudIoTCore.Server }}"

    # Google Cloud IoT Core Device id.
    device_id="{{ .Integration.MQTT.Auth.GCPCloudIoTCore.DeviceID }}"

    # Google Cloud project id.
    project_id="{{ .Integration.MQTT.Auth.GCPCloudIoTCore.ProjectID }}"

    # Google Cloud region.
    cloud_region="{{ .Integration.MQTT.Auth.GCPCloudIoTCore.CloudRegion }}"

    # Google Cloud IoT registry id.
    registry_id="{{ .Integration.MQTT.Auth.GCPCloudIoTCore.RegistryID }}"

    # JWT token expiration time.
    jwt_expiration="{{ .Integration.MQTT.Auth.GCPCloudIoTCore.JWTExpiration }}"

    # JWT token key-file.
    #
    # Example command to generate a key-pair:
    #  $ ssh-keygen -t rsa -b 4096 -f private-key.pem
    #  $ openssl rsa -in private-key.pem -pubout -outform PEM -out public-key.pem
    #
    # Then point the setting below to the private-key.pem and associate the
    # public-key.pem with this device / gateway in Google Cloud IoT Core.
    jwt_key_file="{{ .Integration.MQTT.Auth.GCPCloudIoTCore.JWTKeyFile }}"


    # Azure IoT Hub
    #
    # This setting will preset uplink and downlink topics that will only
    # work with Azure IoT Hub service.
    [integration.mqtt.auth.azure_iot_hub]

    # Device connection string (symmetric key authentication).
    #
    # This connection string can be retrieved from the Azure IoT Hub device
    # details when using the symmetric key authentication type.
    device_connection_string="{{ .Integration.MQTT.Auth.AzureIoTHub.DeviceConnectionString }}"

    # Token expiration (symmetric key authentication).
    #
    # ChirpStack Gateway Bridge will generate a SAS token with the given expiration.
    # After the token has expired, it will generate a new one and trigger a
    # re-connect (only for symmetric key authentication).
    sas_token_expiration="{{ .Integration.MQTT.Auth.AzureIoTHub.SASTokenExpiration }}"

    # Device ID (X.509 authentication).
    #
    # This will be automatically set when a device connection string is given.
    # It must be set for X.509 authentication.
    device_id="{{ .Integration.MQTT.Auth.AzureIoTHub.DeviceID }}"

    # IoT Hub hostname (X.509 authentication).
    #
    # This will be automatically set when a device connection string is given.
    # It must be set for X.509 authentication.
    # Example: iot-hub-name.azure-devices.net
    hostname="{{ .Integration.MQTT.Auth.AzureIoTHub.Hostname }}"

    # Client certificates (X.509 authentication).
    #
    # Configure the tls_cert (certificate file) and tls_key (private-key file)
    # when the device is configured with X.509 authentication.
    tls_cert="{{ .Integration.MQTT.Auth.AzureIoTHub.TLSCert }}"
    tls_key="{{ .Integration.MQTT.Auth.AzureIoTHub.TLSKey }}"


# Metrics configuration.
[metrics]

  # Metrics stored in Prometheus.
  #
  # These metrics expose information about the state of the ChirpStack Gateway Bridge
  # instance like number of messages processed, number of function calls, etc.
  [metrics.prometheus]
  # Expose Prometheus metrics endpoint.
  endpoint_enabled={{ .Metrics.Prometheus.EndpointEnabled }}

  # The ip:port to bind the Prometheus metrics server to for serving the
  # metrics endpoint.
  bind="{{ .Metrics.Prometheus.Bind }}"


# Gateway meta-data.
#
# The meta-data will be added to every stats message sent by the ChirpStack Gateway
# Bridge.
[meta_data]

  # Static.
  #
  # Static key (string) / value (string) meta-data.
  [meta_data.static]
  # Example:
  # serial_number="A1B21234"
  {{ range $k, $v := .MetaData.Static }}
  {{ $k }}="{{ $v }}"
  {{ end }}


  # Dynamic meta-data.
  #
  # Dynamic meta-data is retrieved by executing external commands.
  # This makes it possible to for example execute an external command to
  # read the gateway temperature.
  [meta_data.dynamic]

  # Execution interval of the commands.
  execution_interval="{{ .MetaData.Dynamic.ExecutionInterval }}"

  # Max. execution duration.
  max_execution_duration="{{ .MetaData.Dynamic.MaxExecutionDuration }}"

  # Split delimiter.
  #
  # When the output of a command returns multiple lines, ChirpStack Gateway Bridge
  # assumes multiple values are returned. In this case it will split by the given delimiter
  # to obtain the key / value of each row. The key will be prefixed with the name of the
  # configured command.
  split_delimiter="{{ .MetaData.Dynamic.SplitDelimiter }}"


  # Commands to execute.
  #
  # The value of the stdout will be used as the key value (string).
  # In case the command failed, it is ignored. In case the same key is defined
  # both as static and dynamic, the dynamic value has priority (as long as the)
  # command does not fail.
  [meta_data.dynamic.commands]
  # Example:
  # temperature="/opt/gateway-temperature/gateway-temperature.sh"
  {{ range $k, $v := .MetaData.Dynamic.Commands }}
  {{ $k }}="{{ $v }}"
  {{ end }}

# Executable commands.
#
# The configured commands can be triggered by sending a message to the
# ChirpStack Gateway Bridge.
[commands]
  # Example:
  # [commands.commands.reboot]
  # max_execution_duration="1s"
  # command="/usr/bin/reboot"
{{ range $k, $v := .Commands.Commands }}
  [commands.commands.{{ $k }}]
  max_execution_duration="{{ $v.MaxExecutionDuration }}"
  command="{{ $v.Command }}"
{{ end }}
`

var configCmd = &cobra.Command{
	Use:   "configfile",
	Short: "Print the ChirpStack Gateway Bridge configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		t := template.Must(template.New("config").Parse(configTemplate))
		err := t.Execute(os.Stdout, config.C)
		if err != nil {
			return errors.Wrap(err, "execute config template error")
		}
		return nil
	},
}
