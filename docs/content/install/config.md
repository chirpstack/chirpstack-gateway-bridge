---
title: Configuration
menu:
    main:
        parent: install
        weight: 5
description: Instructions and examples how to configure the LoRa Gateway Bridge service.
---

# Configuration

## Gateway

Modify the [packet-forwarder](https://github.com/lora-net/packet_forwarder)
of your gateway so that it will send its data to the LoRa Gateway Bridge.
You will need to change the following configuration keys:

* `server_address` to the IP address / hostname of the LoRa Gateway Bridge
* `serv_port_up` to `1700` (the default port that LoRa Gateway Bridge is using)
* `serv_port_down` to `1700` (same)

## LoRa Gateway Bridge

The `lora-gateway-bridge` has the following command-line flags:

{{<highlight text>}}
LoRa Gateway Bridge abstracts the packet_forwarder protocol into JSON over MQTT
        > documentation & support: https://docs.loraserver.io/lora-gateway-bridge
        > source & copyright information: https://github.com/brocaar/lora-gateway-bridge

Usage:
  lora-gateway-bridge [flags]
  lora-gateway-bridge [command]

Available Commands:
  configfile  Print the LoRa Gateway configuration file
  help        Help about any command
  version     Print the LoRa Gateway Bridge version

Flags:
  -c, --config string   path to configuration file (optional)
  -h, --help            help for lora-gateway-bridge
      --log-level int   debug=5, info=4, error=2, fatal=1, panic=0 (default 4)

Use "lora-gateway-bridge [command] --help" for more information about a command.
{{< /highlight >}}

### Configuration file

By default `lora-gateway-bridge` will look in the following order for a
configuration at the following paths when `--config` / `-c` is not set:

* `lora-gateway-bridge.toml` (current working directory)
* `$HOME/.config/lora-gateway-bridge/lora-gateway-bridge.toml`
* `/etc/lora-gateway-bridge/lora-gateway-bridge.toml`

To load configuration from a different location, use the `--config` flag.

To generate a new configuration file `lora-gateway-bridge.toml`, execute the following command:

{{<highlight bash>}}
lora-gateway-bridge configfile > lora-gateway-bridge.toml
{{< /highlight >}}

Note that this configuration file will be pre-filled with the current configuration
(either loaded from the paths mentioned above, or by using the `--config` flag).
This makes it possible when new fields get added to upgrade your configuration file
while preserving your old configuration. Example:

{{<highlight bash>}}
lora-gateway-bridge configfile --config lora-gateway-bridge-old.toml > lora-gateway-bridge-new.toml
{{< /highlight >}}

Example configuration file:

{{<highlight toml>}}
[general]
# debug=5, info=4, warning=3, error=2, fatal=1, panic=0
log_level = 4


# Configuration which relates to the packet-forwarder.
[packet_forwarder]
# ip:port to bind the UDP listener to
#
# Example: 0.0.0.0:1700 to listen on port 1700 for all network interfaces.
# This is the listeren to which the packet-forwarder forwards its data
# so make sure the 'serv_port_up' and 'serv_port_down' from your
# packet-forwarder matches this port.
udp_bind = "0.0.0.0:1700"

# Skip the CRC status-check of received packets
#
# This is only has effect when the packet-forwarder is configured to forward
# LoRa frames with CRC errors.
skip_crc_check = false


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
# Use "{{ .MAC }}" as an substitution for the LoRa gateway MAC.
#
# Note that some authentication types might overwrite these templates (e.g.
# in case of GCP Cloud IoT Core)!
uplink_topic_template="gateway/{{ .MAC }}/rx"
downlink_topic_template="gateway/{{ .MAC }}/tx"
stats_topic_template="gateway/{{ .MAC }}/stats"
ack_topic_template="gateway/{{ .MAC }}/ack"
config_topic_template="gateway/{{ .MAC }}/config"

# Payload marshaler.
#
# This defines how the MQTT payloads are encoded. Valid options are:
# * v2_json:   The default LoRa Gateway Bridge v2 encoding (will be deprecated and removed in LoRa Gateway Bridge v3)
# * protobuf:  Protobuf encoding (this will become the LoRa Gateway Bridge v3 default)
# * json:      JSON encoding (easier for debugging, but less compact than 'protobuf')
marshaler="v2_json"

  # MQTT authentication.
  [backend.mqtt.auth]
  # Type defines the MQTT authentication type to use.
  #
  # Set this to the name of one of the sections below.
  # Note: when the 'v2_json marhaler' is configured, the generic backend will
  # always be used.
  type="generic"

    # Generic MQTT authentication.
    [backend.mqtt.auth.generic]
    # MQTT server (e.g. scheme://host:port where scheme is tcp, ssl or ws)
    server="tcp://127.0.0.1:1883"

    # Connect with the given username (optional)
    username=""

    # Connect with the given password (optional)
    password=""

    # Quality of service level
    #
    # 0: at most once
    # 1: at least once
    # 2: exactly once
    #
    # Note: an increase of this value will decrease the performance.
    # For more information: https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels
    qos=0

    # Clean session
    #
    # Set the "clean session" flag in the connect message when this client
    # connects to an MQTT broker. By setting this flag you are indicating
    # that no messages saved by the broker for this client should be delivered.
    clean_session=true

    # Client ID
    #
    # Set the client id to be used by this client when connecting to the MQTT
    # broker. A client id must be no longer than 23 characters. When left blank,
    # a random id will be generated. This requires clean_session=true.
    client_id=""

    # CA certificate file (optional)
    #
    # Use this when setting up a secure connection (when server uses ssl://...)
    # but the certificate used by the server is not trusted by any CA certificate
    # on the server (e.g. when self generated).
    ca_cert=""

    # mqtt TLS certificate file (optional)
    tls_cert=""

    # mqtt TLS key file (optional)
    tls_key=""

    # Maximum interval that will be waited between reconnection attempts when connection is lost.
    # Valid units are 'ms', 's', 'm', 'h'. Note that these values can be combined, e.g. '24h30m15s'.
    max_reconnect_interval="10m0s"


    # Google Cloud Platform Cloud IoT Core authentication.
    #
    # Please note that when using this authentication type, the MQTT topics
    # will be automatically set to match the MQTT topics as expected by
    # Cloud IoT Core.
    [backend.mqtt.auth.gcp_cloud_iot_core]
    # MQTT server.
    server="ssl://mqtt.googleapis.com:8883"

    # Google Cloud IoT Core Device id.
    device_id=""

    # Google Cloud project id.
    project_id=""

    # Google Cloud region.
    cloud_region=""

    # Google Cloud IoT registry id.
    registry_id=""

    # JWT token expiration time.
    jwt_expiration="24h0m0s"

    # JWT token key-file.
    #
    # Example command to generate a key-pair:
    #  $ ssh-keygen -t rsa -b 4096 -f private-key.pem
    #  $ openssl rsa -in private-key.pem -pubout -outform PEM -out public-key.pem
    #
    # Then point the setting below to the private-key.pem and associate the
    # public-key.pem with this device / gateway in Google Cloud IoT Core.
    jwt_key_file=""


# Metrics configuration.
[metrics]

  # Metrics stored in Prometheus.
  #
  # These metrics expose information about the state of the LoRa Gateway Bridge
  # instance like number of messages processed, number of function calls, etc.
  [metrics.prometheus]
  # Expose Prometheus metrics endpoint.
  endpoint_enabled=false

  # The ip:port to bind the Prometheus metrics server to for serving the
  # metrics endpoint.
  bind=""
{{< /highlight >}}

### Warning: deprecation warning! update your configuration

When you see this warning, you need to update your configuration!
Before LoRa Gateway Bridge 2.3.0 environment variables were used for setting
configuration flags. Since LoRa Gateway Bridge 2.3.0 the configuration format
has changed.

The `.deb` installer will automatically migrate your configuration. For non
`.deb` installations, you can migrate your configuration in the following way:

{{<highlight bash>}}
# Export your environment variables, in this case from a file, but anything
# that sets your environment variables will work.
set -a
source /etc/default/lora-gateway-bridge

# Create the configuration directory.
mkdir /etc/lora-gateway-bridge

# Generate new configuration file, pre-filled with the configuration set
# through the environment variables.
lora-gateway-bridge configfile > /etc/lora-gateway-bridge/lora-gateway-bridge.toml

# "Remove" the old configuration (in you were using a file).
mv /etc/default/lora-gateway-bridge /etc/default/lora-gateway-bridge.old
{{< /highlight >}}
