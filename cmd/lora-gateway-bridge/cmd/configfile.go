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


# Configuration which relates to the packet-forwarder.
[packet_forwarder]
# ip:port to bind the UDP listener to
#
# Example: 0.0.0.0:1700 to listen on port 1700 for all network interfaces.
# This is the listeren to which the packet-forwarder forwards its data
# so make sure the 'serv_port_up' and 'serv_port_down' from your
# packet-forwarder matches this port.
udp_bind = "{{ .PacketForwarder.UDPBind }}"

# Skip the CRC status-check of received packets
#
# This is only has effect when the packet-forwarder is configured to forward
# LoRa frames with CRC errors.
skip_crc_check = {{ .PacketForwarder.SkipCRCCheck }}


  # # Managed packet-forwarder configuration.
  # #
  # # By configuring one or multiple managed packet-forwarder sections, the
  # # LoRa Gateway Bridge updates the configuration when the backend receives
  # # a configuration change, after which it will restart the packet-forwarder.
  # [[packet_forwarder.configuration]]
  # # Gateway MAC.
  # #
  # # The LoRa Gateway Bridge will only apply the configuration updates for this
  # # gateway MAC.
  # mac="0102030405060708"

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


# Configuration for the MQTT backend.
[backend.mqtt]
# MQTT topic templates for the different MQTT topics.
#
# The meaning of these topics are documented at:
# https://docs.loraserver.io/lora-gateway-bridge/use/data/
#
# The default values match the default expected configuration of the
# LoRa Server MQTT backend. Therefore only change these values when
# absolutely needed.
# Use "{{ "{{ .MAC }}" }}" as an substitution for the LoRa gateway MAC. 
#
# Note that some authentication types might overwrite these templates (e.g.
# in case of GCP Cloud IoT Core)!
uplink_topic_template="{{ .Backend.MQTT.UplinkTopicTemplate }}"
downlink_topic_template="{{ .Backend.MQTT.DownlinkTopicTemplate }}"
stats_topic_template="{{ .Backend.MQTT.StatsTopicTemplate }}"
ack_topic_template="{{ .Backend.MQTT.AckTopicTemplate }}"
config_topic_template="{{ .Backend.MQTT.ConfigTopicTemplate }}"

# Payload marshaler.
#
# This defines how the MQTT payloads are encoded. Valid options are:
# * v2_json:   The default LoRa Gateway Bridge v2 encoding (will be deprecated and removed in LoRa Gateway Bridge v3)
# * protobuf:  Protobuf encoding (this will become the LoRa Gateway Bridge v3 default)
# * json:      JSON encoding (easier for debugging, but less compact than 'protobuf')
marshaler="{{ .Backend.MQTT.Marshaler }}"

  # MQTT authentication.
  [backend.mqtt.auth]
  # Type defines the MQTT authentication type to use.
  #
  # Set this to the name of one of the sections below.
  # Note: when the 'v2_json marhaler' is configured, the generic backend will
  # always be used.
  type="{{ .Backend.MQTT.Auth.Type }}"

    # Generic MQTT authentication.
    [backend.mqtt.auth.generic]
    # MQTT server (e.g. scheme://host:port where scheme is tcp, ssl or ws)
    server="{{ .Backend.MQTT.Auth.Generic.Server }}"

    # Connect with the given username (optional)
    username="{{ .Backend.MQTT.Auth.Generic.Username }}"

    # Connect with the given password (optional)
    password="{{ .Backend.MQTT.Auth.Generic.Password }}"

    # Quality of service level
    #
    # 0: at most once
    # 1: at least once
    # 2: exactly once
    #
    # Note: an increase of this value will decrease the performance.
    # For more information: https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels
    qos={{ .Backend.MQTT.Auth.Generic.QOS }}

    # Clean session
    #
    # Set the "clean session" flag in the connect message when this client
    # connects to an MQTT broker. By setting this flag you are indicating
    # that no messages saved by the broker for this client should be delivered.
    clean_session={{ .Backend.MQTT.Auth.Generic.CleanSession }}

    # Client ID
    #
    # Set the client id to be used by this client when connecting to the MQTT
    # broker. A client id must be no longer than 23 characters. When left blank,
    # a random id will be generated. This requires clean_session=true.
    client_id="{{ .Backend.MQTT.Auth.Generic.ClientID }}"

    # CA certificate file (optional)
    #
    # Use this when setting up a secure connection (when server uses ssl://...)
    # but the certificate used by the server is not trusted by any CA certificate
    # on the server (e.g. when self generated).
    ca_cert="{{ .Backend.MQTT.Auth.Generic.CACert }}"

    # mqtt TLS certificate file (optional)
    tls_cert="{{ .Backend.MQTT.Auth.Generic.TLSCert }}"

    # mqtt TLS key file (optional)
    tls_key="{{ .Backend.MQTT.Auth.Generic.TLSKey }}"

    # Maximum interval that will be waited between reconnection attempts when connection is lost.
    # Valid units are 'ms', 's', 'm', 'h'. Note that these values can be combined, e.g. '24h30m15s'.
    max_reconnect_interval="{{ .Backend.MQTT.Auth.Generic.MaxReconnectInterval }}"


    # Google Cloud Platform Cloud IoT Core authentication.
    #
    # Please note that when using this authentication type, the MQTT topics
    # will be automatically set to match the MQTT topics as expected by
    # Cloud IoT Core.
    [backend.mqtt.auth.gcp_cloud_iot_core]
    # MQTT server.
    server="{{ .Backend.MQTT.Auth.GCPCloudIoTCore.Server }}"

    # Google Cloud IoT Core Device id.
    device_id="{{ .Backend.MQTT.Auth.GCPCloudIoTCore.DeviceID }}"

    # Google Cloud project id.
    project_id="{{ .Backend.MQTT.Auth.GCPCloudIoTCore.ProjectID }}"

    # Google Cloud region.
    cloud_region="{{ .Backend.MQTT.Auth.GCPCloudIoTCore.CloudRegion }}"

    # Google Cloud IoT registry id.
    registry_id="{{ .Backend.MQTT.Auth.GCPCloudIoTCore.RegistryID }}"

    # JWT token expiration time.
    jwt_expiration="{{ .Backend.MQTT.Auth.GCPCloudIoTCore.JWTExpiration }}"

    # JWT token key-file.
    #
    # Example command to generate a key-pair:
    #  $ ssh-keygen -t rsa -b 4096 -f private-key.pem
    #  $ openssl rsa -in private-key.pem -pubout -outform PEM -out public-key.pem
    #
    # Then point the setting below to the private-key.pem and associate the
    # public-key.pem with this device / gateway in Google Cloud IoT Core.
    jwt_key_file="{{ .Backend.MQTT.Auth.GCPCloudIoTCore.JWTKeyFile }}"


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
