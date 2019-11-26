---
title: Configuration
menu:
    main:
        parent: install
        weight: 5
description: Instructions and examples how to configure the ChirpStack Gateway Bridge service.
---

# Configuration

The `chirpstack-gateway-bridge` has the following command-line flags:

{{<highlight text>}}
ChirpStack Gateway Bridge abstracts the packet_forwarder protocol into Protobuf or JSON over MQTT
        > documentation & support: https://www.chirpstack.io/gateway-bridge/
        > source & copyright information: https://github.com/brocaar/chirpstack-gateway-bridge

Usage:
  chirpstack-gateway-bridge [flags]
  chirpstack-gateway-bridge [command]

Available Commands:
  configfile  Print the ChirpStack Gateway Bridge configuration file
  help        Help about any command
  version     Print the ChirpStack Gateway Bridge version

Flags:
  -c, --config string   path to configuration file (optional)
  -h, --help            help for chirpstack-gateway-bridge
      --log-level int   debug=5, info=4, error=2, fatal=1, panic=0 (default 4)

Use "chirpstack-gateway-bridge [command] --help" for more information about a command.
{{< /highlight >}}

## Configuration file

By default `chirpstack-gateway-bridge` will look in the following order for a
configuration at the following paths when `--config` / `-c` is not set:

* `chirpstack-gateway-bridge.toml` (current working directory)
* `$HOME/.config/chirpstack-gateway-bridge/chirpstack-gateway-bridge.toml`
* `/etc/chirpstack-gateway-bridge/chirpstack-gateway-bridge.toml`

To load configuration from a different location, use the `--config` flag.

To generate a new configuration file `chirpstack-gateway-bridge.toml`, execute the following command:

{{<highlight bash>}}
chirpstack-gateway-bridge configfile > chirpstack-gateway-bridge.toml
{{< /highlight >}}

Note that this configuration file will be pre-filled with the current configuration
(either loaded from the paths mentioned above, or by using the `--config` flag).
This makes it possible when new fields get added to upgrade your configuration file
while preserving your old configuration. Example:

{{<highlight bash>}}
chirpstack-gateway-bridge configfile --config chirpstack-gateway-bridge-old.toml > chirpstack-gateway-bridge-new.toml
{{< /highlight >}}

Example configuration file:

{{<highlight toml>}}
[general]
# debug=5, info=4, warning=3, error=2, fatal=1, panic=0
log_level = 4


# Filters.
#
# These can be used to filter LoRaWAN frames to reduce bandwith usage between
# the gateway and ChirpStack Gateway Bride. Depending the used backend, filtering
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
net_ids=[
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
join_euis=[
]


# Gateway backend configuration.
[backend]

# Backend type.
#
# Valid options are:
#   * semtech_udp
#   * basic_station
type="semtech_udp"


  # Semtech UDP packet-forwarder backend.
  [backend.semtech_udp]

  # ip:port to bind the UDP listener to
  #
  # Example: 0.0.0.0:1700 to listen on port 1700 for all network interfaces.
  # This is the listener to which the packet-forwarder forwards its data
  # so make sure the 'serv_port_up' and 'serv_port_down' from your
  # packet-forwarder matches this port.
  udp_bind = "0.0.0.0:1700"

  # Skip the CRC status-check of received packets
  #
  # This is only has effect when the packet-forwarder is configured to forward
  # LoRa frames with CRC errors.
  skip_crc_check = false

  # Fake RX timestamp.
  #
  # Fake the RX time when the gateway does not have GPS, in which case
  # the time would otherwise be unset.
  fake_rx_time=false



  # Basic Station backend.
  [backend.basic_station]

  # ip:port to bind the Websocket listener to.
  bind=":3001"

  # TLS certificate and key files.
  #
  # When set, the websocket listener will use TLS to secure the connections
  # between the gateways and ChirpStack Gateway Bridge (optional).
  tls_cert=""
  tls_key=""

  # TLS CA certificate.
  #
  # When configured, ChirpStack Gateway Bridge will validate that the client
  # certificate of the gateway has been signed by this CA certificate.
  ca_cert=""

  # Ping interval.
  ping_interval="1m0s"

  # Read timeout.
  #
  # This interval must be greater than the configured ping interval.
  read_timeout="1m5s"

  # Write timeout.
  write_timeout="1s"

  # Region.
  #
  # Please refer to the LoRaWAN Regional Parameters specification
  # for the complete list of common region names.
  region="EU868"

  # Minimal frequency (Hz).
  frequency_min=863000000

  # Maximum frequency (Hz).
  frequency_max=870000000

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


# Integration configuration.
[integration]
# Payload marshaler.
#
# This defines how the MQTT payloads are encoded. Valid options are:
# * protobuf:  Protobuf encoding
# * json:      JSON encoding (easier for debugging, but less compact than 'protobuf')
marshaler="protobuf"

  # MQTT integration configuration.
  [integration.mqtt]
  # Event topic template.
  event_topic_template="gateway/{{ .GatewayID }}/event/{{ .EventType }}"

  # Command topic template.
  command_topic_template="gateway/{{ .GatewayID }}/command/#"

  # Maximum interval that will be waited between reconnection attempts when connection is lost.
  # Valid units are 'ms', 's', 'm', 'h'. Note that these values can be combined, e.g. '24h30m15s'.
  max_reconnect_interval="10m0s"


  # MQTT authentication.
  [integration.mqtt.auth]
  # Type defines the MQTT authentication type to use.
  #
  # Set this to the name of one of the sections below.
  type="generic"

    # Generic MQTT authentication.
    [integration.mqtt.auth.generic]
    # MQTT servers.
    #
    # Configure one or multiple MQTT server to connect to. Each item must be in
    # the following format: scheme://host:port where scheme is tcp, ssl or ws.
    servers=[
      "tcp://127.0.0.1:1883",
    ]

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


    # Google Cloud Platform Cloud IoT Core authentication.
    #
    # Please note that when using this authentication type, the MQTT topics
    # will be automatically set to match the MQTT topics as expected by
    # Cloud IoT Core.
    [integration.mqtt.auth.gcp_cloud_iot_core]
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


    # Azure IoT Hub
    #
    # This setting will preset uplink and downlink topics that will only
    # work with Azure IoT Hub service.
    [integration.mqtt.auth.azure_iot_hub]

    # Device connection string (symmetric key authentication).
    #
    # This connection string can be retrieved from the Azure IoT Hub device
    # details when using the symmetric key authentication type.
    device_connection_string=""

    # Token expiration (symmetric key authentication).
    #
    # ChirpStack Gateway Bridge will generate a SAS token with the given expiration.
    # After the token has expired, it will generate a new one and trigger a
    # re-connect (only for symmetric key authentication).
    sas_token_expiration="24h0m0s"

    # Device ID (X.509 authentication).
    #
    # This will be automatically set when a device connection string is given.
    # It must be set for X.509 authentication.
    device_id=""

    # IoT Hub hostname (X.509 authentication).
    #
    # This will be automatically set when a device connection string is given.
    # It must be set for X.509 authentication.
    # Example: iot-hub-name.azure-devices.net
    hostname=""

    # Client certificates (X.509 authentication).
    #
    # Configure the tls_cert (certificate file) and tls_key (private-key file)
    # when the device is configured with X.509 authentication.
    tls_cert=""
    tls_key=""


# Metrics configuration.
[metrics]

  # Metrics stored in Prometheus.
  #
  # These metrics expose information about the state of the ChirpStack Gateway Bridge
  # instance like number of messages processed, number of function calls, etc.
  [metrics.prometheus]
  # Expose Prometheus metrics endpoint.
  endpoint_enabled=false

  # The ip:port to bind the Prometheus metrics server to for serving the
  # metrics endpoint.
  bind=""


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



  # Dynamic meta-data.
  #
  # Dynamic meta-data is retrieved by executing external commands.
  # This makes it possible to for example execute an external command to
  # read the gateway temperature.
  [meta_data.dynamic]

  # Execution interval of the commands.
  execution_interval="1m0s"

  # Max. execution duration.
  max_execution_duration="1s"

  # Commands to execute.
  #
  # The value of the stdout will be used as the key value (string).
  # In case the command failed, it is ignored. In case the same key is defined
  # both as static and dynamic, the dynamic value has priority (as long as the)
  # command does not fail.
  [meta_data.dynamic.commands]
  # Example:
  # temperature="/opt/gateway-temperature/gateway-temperature.sh"


# Executable commands.
#
# The configured commands can be triggered by sending a message to the
# ChirpStack Gateway Bridge.
[commands]
  # Example:
  # [commands.commands.reboot]
  # max_execution_duration="1s"
  # command="/usr/bin/reboot"
{{</highlight>}}

## Environment variables

Although using the configuration file is recommended, it is also possible
to use environment variables to set configuration variables.

Example:

{{<highlight toml>}}
[backend.semtech_udp]
udp_bind="0.0.0.0:1700"
{{</highlight>}}

Can be set using the environment variable:

{{<highlight toml>}}
BACKEND.SEMTECH_UDP.UDP_BIND="0.0.0.0:1700"
{{</highlight>}}

