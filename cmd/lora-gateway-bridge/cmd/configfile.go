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
uplink_topic_template="{{ .Backend.MQTT.UplinkTopicTemplate }}"
downlink_topic_template="{{ .Backend.MQTT.DownlinkTopicTemplate }}"
stats_topic_template="{{ .Backend.MQTT.StatsTopicTemplate }}"
ack_topic_template="{{ .Backend.MQTT.AckTopicTemplate }}"

# MQTT server (e.g. scheme://host:port where scheme is tcp, ssl or ws)
server="{{ .Backend.MQTT.Server }}"

# Connect with the given username (optional)
username="{{ .Backend.MQTT.Username }}"

# Connect with the given password (optional)
password="{{ .Backend.MQTT.Password }}"

# Quality of service level
#
# 0: at most once
# 1: at least once
# 2: exactly once
#
# Note: an increase of this value will decrease the performance.
# For more information: https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels
qos={{ .Backend.MQTT.QOS }}

# Clean session
#
# Set the "clean session" flag in the connect message when this client
# connects to an MQTT broker. By setting this flag you are indicating
# that no messages saved by the broker for this client should be delivered.
clean_session={{ .Backend.MQTT.CleanSession }}

# Client ID
#
# Set the client id to be used by this client when connecting to the MQTT
# broker. A client id must be no longer than 23 characters. When left blank,
# a random id will be generated. This requires clean_session=true.
client_id="{{ .Backend.MQTT.ClientID }}"

# CA certificate file (optional)
#
# Use this when setting up a secure connection (when server uses ssl://...)
# but the certificate used by the server is not trusted by any CA certificate
# on the server (e.g. when self generated).
ca_cert="{{ .Backend.MQTT.CACert }}"

# mqtt TLS certificate file (optional)
tls_cert="{{ .Backend.MQTT.TLSCert }}"

# mqtt TLS key file (optional)
tls_key="{{ .Backend.MQTT.TLSKey }}"
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
