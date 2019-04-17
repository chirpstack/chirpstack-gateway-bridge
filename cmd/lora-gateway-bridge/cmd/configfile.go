package cmd

import (
	"html/template"
	"os"

	"github.com/brocaar/lora-gateway-bridge/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// when updating this template, don't forget to update config.md!
const configTemplate = `[general]
# debug=5, info=4, warning=3, error=2, fatal=1, panic=0
log_level = {{ .General.LogLevel }}


# Gateway backend configuration.
[backend]

# Backend type.
#
# Valid options are:
#   * semtech_udp
#   * basic_station
type="{{ .Backend.Type }}"


  # Semtech UDP packet-forwarder backend.
  [backend.semtech_udp]

  # ip:port to bind the UDP listener to
  #
  # Example: 0.0.0.0:1700 to listen on port 1700 for all network interfaces.
  # This is the listeren to which the packet-forwarder forwards its data
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

    # # Managed packet-forwarder configuration.
    # #
    # # By configuring one or multiple managed packet-forwarder sections, the
    # # LoRa Gateway Bridge updates the configuration when the backend receives
    # # a configuration change, after which it will restart the packet-forwarder.
    # [[packet_forwarder.configuration]]
    # # Gateway ID.
    # #
    # # The LoRa Gateway Bridge will only apply the configuration updates for this
    # # gateway ID.
    # gateway_id="0102030405060708"

    # # Base configuration file.
    # #
    # # This file will be used as base-configuration and will not be overwritten on
    # # a configuration update. This file needs to exist and contains the base
    # # configuration and vendor specific
    # base_file="/etc/lora-packet-forwarder/global_conf.json"

    # # Output configuration file.
    # #
    # # This will be the final configuration for the packet-forwarder, containing
    # # a merged version of the base configuration + the requested configuration
    # # update.
    # # Warning: this file will be overwritten on a configuration update!
    # output_file="/etc/lora-packet-forwarder/local_conf.json"

    # # Restart command.
    # #
    # # This command is issued by the LoRa Gateway Bridge on a configuration
    # # change. Make sure the LoRa Gateway Bridge process has sufficient
    # # permissions to execute this command.
    # restart_command="/etc/init.d/lora-packet-forwarder restart"

  # Basic Station backend.
  [backend.basic_station]

  # ip:port to bind the Websocket listener to.
  bind="{{ .Backend.BasicStation.Bind }}"

  # TLS certificate and key files.
  #
  # When set, the websocket listener will use TLS to secure the connections
  # between the gateways and LoRa Gateway Bridge (optional).
  tls_cert="{{ .Backend.BasicStation.TLSCert }}"
  tls_key="{{ .Backend.BasicStation.TLSKey }}"

  # TLS CA certificate.
  #
  # When configured, LoRa Gateway Bridge will validate that the client
  # certificate of the gateway has been signed by this CA certificate.
  ca_cert="{{ .Backend.BasicStation.CACert }}"

  # Ping interval.
  ping_interval="{{ .Backend.BasicStation.PingInterval }}"

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

    # Filters.
    [backend.basic_station.filters]

    # NetIDs to filter on when receiving uplinks.
    net_ids=[{{ range $index, $elm := .Backend.BasicStation.Filters.NetIDs }}
      "{{ $elm }}",{{ end }}
    ]

    # JoinEUIs to filter on when receiving join-requests.
    join_euis=[{{ range $index, $elm := .Backend.BasicStation.Filters.JoinEUIs }}
      ["{{ index $elm 0 }}", "{{ index $elm 1 }}"],{{ end }}
    ]


# Integration configuration.
[integration]
# Payload marshaler.
#
# This defines how the MQTT payloads are encoded. Valid options are:
# * protobuf:  Protobuf encoding (this will become the LoRa Gateway Bridge v3 default)
# * json:      JSON encoding (easier for debugging, but less compact than 'protobuf')
marshaler="{{ .Integration.Marshaler }}"

  # MQTT integration configuration.
  [integration.mqtt]
  # Event topic template.
  event_topic_template="{{ .Integration.MQTT.EventTopicTemplate }}"

  # Command topic template.
  command_topic_template="{{ .Integration.MQTT.CommandTopicTemplate }}"


  # MQTT authentication.
  [integration.mqtt.auth]
  # Type defines the MQTT authentication type to use.
  #
  # Set this to the name of one of the sections below.
  type="{{ .Integration.MQTT.Auth.Type }}"

    # Generic MQTT authentication.
    [integration.mqtt.auth.generic]
    # MQTT server (e.g. scheme://host:port where scheme is tcp, ssl or ws)
    server="{{ .Integration.MQTT.Auth.Generic.Server }}"

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

    # Maximum interval that will be waited between reconnection attempts when connection is lost.
    # Valid units are 'ms', 's', 'm', 'h'. Note that these values can be combined, e.g. '24h30m15s'.
    max_reconnect_interval="{{ .Integration.MQTT.Auth.Generic.MaxReconnectInterval }}"


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


# Metrics configuration.
[metrics]

  # Metrics stored in Prometheus.
  #
  # These metrics expose information about the state of the LoRa Gateway Bridge
  # instance like number of messages processed, number of function calls, etc.
  [metrics.prometheus]
  # Expose Prometheus metrics endpoint.
  endpoint_enabled={{ .Metrics.Prometheus.EndpointEnabled }}

  # The ip:port to bind the Prometheus metrics server to for serving the
  # metrics endpoint.
  bind="{{ .Metrics.Prometheus.Bind }}"
`

var configCmd = &cobra.Command{
	Use:   "configfile",
	Short: "Print the LoRa Gateway Bridge configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		t := template.Must(template.New("config").Parse(configTemplate))
		err := t.Execute(os.Stdout, config.C)
		if err != nil {
			return errors.Wrap(err, "execute config template error")
		}
		return nil
	},
}
