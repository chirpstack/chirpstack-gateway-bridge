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
# MQTT server (e.g. scheme://host:port where scheme is tcp, ssl or ws)
server="{{ .Backend.MQTT.Server }}"

# Connect with the given username (optional)
username="{{ .Backend.MQTT.Username }}"

# Connect with the given password (optional)
password="{{ .Backend.MQTT.Password }}"

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
