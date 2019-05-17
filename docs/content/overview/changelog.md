---
title: Changelog
menu:
    main:
        parent: overview
        weight: 2
toc: false
description: Lists the changes per LoRa App Server release, including steps how to upgrade.
---

# Changelog

## v3.0.0

### Features

#### Basic Station packet-forwarder support

The LoRa Gateway Bridge has been refactored to support multiple packet-forwarder
backends. Next to the Semtech UDP packet-forwarder, support has been added to
support the Basic Station packet-forwarder. This backend implements the 
[Basic Station LNS protocol](https://doc.sm.tc/station/tcproto.html).

#### Updated payload formats

The uplink payload contains a `context` field, used to store gateway
specific context data (like the internal counter).

The downlink frame contains a `timing` field which can be either
`IMMEDIATELY`, `DELAY` or `GPS_EPOCH`. Based on the `timing` value, an
additional object must be given with the additional timing information.
Refer to [Commands](https://www.loraserver.io/lora-gateway-bridge/payloads/commands/)
for more details.

#### Azure IoT Hub integration

LoRa Gateway Bridge now has support to connect the [Azure IoT Hub](https://azure.microsoft.com/en-us/services/iot-hub/)
MQTT interface. Like with the Google Cloud Platform IoT Core integration this
makes it easier to scale up and manage gateway credentials using the IoT Hub
console (or API).

#### Fake RX time

In case the gateway does not have a GPS module, the RX time would would be
left blank. This feature makes it possible to use the system time as a fallback.
([#109](https://github.com/brocaar/lora-gateway-bridge/pull/109))

#### Custom gateway meta-data

This feature makes it possible to expose additional meta-data in the gateway
stats. Meta-data can either be static or dynamic (executing external commands).
The latter option allows to for example read and expose the gateway temperature,
humidity, ...

### Upgrading

LoRa Gateway Bridge v3.0.0 include a couple of changes that are not backwards
compatible. You need to re-generate the configuration file and update it where
needed. LoRa Gateway Bridge v3 is compatible with LoRa Server v2.7+.
Below a summary:

#### MQTT topics

The MQTT topic configuration has been updated from:

{{<highlight toml>}}
uplink_topic_template="gateway/{{ .MAC }}/rx"
downlink_topic_template="gateway/{{ .MAC }}/tx"
stats_topic_template="gateway/{{ .MAC }}/stats"
ack_topic_template="gateway/{{ .MAC }}/ack"
config_topic_template="gateway/{{ .MAC }}/config"
{{</highlight>}}

To:

{{<highlight toml>}}
event_topic_template="gateway/{{ .GatewayID }}/event/{{ .EventType }}"
command_topic_template="gateway/{{ .GatewayID }}/command/#"
{{</highlight>}}

Event types are: `up`, `stats` and `ack`.
Commands are: `down` and `config`.

Please note that LoRa Gateway Bridge v3 is compatible with LoRa Server v2.7+,
but you will need to update the MQTT topics in your `loraserver.toml` file.
Example snippet:

{{<highlight toml>}}
uplink_topic_template="gateway/+/event/up"
stats_topic_template="gateway/+/event/stats"
ack_topic_template="gateway/+/event/ack"
downlink_topic_template="gateway/{{ .MAC }}/command/down"
config_topic_template="gateway/{{ .MAC }}/command/config"
{{</highlight>}}

#### Backends

With LoRa Gateway Bridge v2 you would configure the MQTT backend under the
`[backend...]` section. This has changed and the `[backend...]` section is
now used for selecting and configuring the packet-forwarder backends.

#### Integrations

The MQTT integration configuration has moved to the new `[integration...]`
section. This allows for adding new integrations in the future besides MQTT.

#### Prometheus metrics

The Prometheus metrics have been updated / cleaned up.

## v2.7.1

### Bugfixes

* Fix `rxpk.brd` parsing for some packet-forwarders. ([#104](https://github.com/brocaar/lora-gateway-bridge/issues/104))

## v2.7.0

### Features

* Environment variable based [configuration](https://www.loraserver.io/lora-gateway-bridge/install/config/) has been re-implemented.

### Improvements

* Remove QoS > 0 bottleneck with non-blocking MQTT publish ([#101](https://github.com/brocaar/lora-gateway-bridge/pull/101))

### Bugfixes

* Fix potential deadlock on MQTT re-connect ([#103](https://github.com/brocaar/lora-gateway-bridge/issues/103))
* Fix logrotate issue (init based systems) ([#98](https://github.com/brocaar/lora-gateway-bridge/pull/98))

## v2.6.2

### Bugfixes

* Fix QoS backend setting regression. [#95](https://github.com/brocaar/lora-gateway-bridge/pull/95)

## v2.6.1

### Bugfixes

* Fix corrupted gateway statistics for downlink packets. [#94](https://github.com/brocaar/lora-gateway-bridge/pull/94)

## v2.6.0

### Features

#### Google Cloud Platform integration

LoRa Gateway Bridge is able to integrate with [Cloud IoT Core](https://cloud.google.com/iot-core/).
This makes it easier to scale up, but also makes it possible to manage gateway
authentication using the web-based Cloud IoT console (or APIs).

#### Gateway IP address

The gateway IP address is exposed through the gateway stats.

## v2.5.1

### Bugfixes

Fix configuration parse error (some keys were not loaded because of configuration alias).

## v2.5.0

### Upgrade notes

When using the `v2_json` marshaler (default), this version is fully compatible
with previous versions of LoRa Server. When changing the marshaler to `protobuf`
or `json`, you first need to upgrade to LoRa Server v2.1.0. When
LoRa Server v2.1.0 is installed, it is recommended to use either the `protobuf`
or `json` marshaler as it provides better compatibility (e.g. with the iBTS gateway).

### Features

#### Protocol Buffer data serialization

To save on bandwith between the gateway and the MQTT broker (e.g. when the
gateway uses a cellular connection), this update makes it possible to
configure the `marshaler` for encoding / decoding the data (in
`lora-gateway-bridge.toml`) using [Protocol Buffers](https://developers.google.com/protocol-buffers/).

This will become the default option in LoRa Gateway Bridge v3.

#### New JSON format

The new JSON structure re-uses the messages defined for
[Protocol Buffers](https://developers.google.com/protocol-buffers/docs/proto3#json)
based serialization. For backwards compatibility, the default `marshaler` for
LoRa Gateway Bridge v2.x will be `v2_json`. 

#### Kerlink iBTS fine-timestamp

When using the `protobuf` or `json` `marshaler`, LoRa Gateway Bridge will
expose the fine-timestamp fields when using Kerlink iBTS gateways.

#### Prometheus metrics

For monitoring the health and throughput of each LoRa Gateway Bridge instance,
LoRa Gateway Bridge is able to expose various metrics that can be scraped by
[Prometheus](https://prometheus.io/).

### Changes

#### Configuration structure of MQTT backend

The structure of the `[backend.mqtt]` configuration section has been updated.
These changes are fully backwards compatible.

### Improvements

All vendored dependencies have been updated.

## 2.4.1

**Bugfixes:**

* Only log in case of UDP read / write errors (instead of breaking out of loop).

## 2.4.0

**Features:**

* LoRa Gateway Bridge can now manage the packet-forwarder configuration (channels).
  See [Send / receive data](https://www.loraserver.io/lora-gateway-bridge/use/data/) for more information.

## 2.3.2

**Improvements:**

* Expose the following MQTT options for the MQTT gateway backend:
  * QoS (quality of service)
  * Client ID
  * Clean session on connect

**Bugfixes:**

* Use topic from configuration file on re-connect (this was still hardcoded).

## 2.3.1

**Improvements:**

* MQTT topics are now configurable through the configuration file.
  See [Configuration](https://docs.loraserver.io/lora-gateway-bridge/install/config/).

## 2.3.0

**Features:**

* LoRa Gateway Bridge uses a new configuration file format.
  See [configuration](https://docs.loraserver.io/lora-gateway-bridge/install/config/) for more information.

* Support MQTT client certificate authentication ([#74](https://github.com/brocaar/lora-gateway-bridge/pull/74)).

**Upgrade notes:**

When upgrading using the `.deb` package / using `apt` or `apt-get`, your
configuration will be automatically migrated for you. In any other case,
please see [configuration](https://docs.loraserver.io/lora-gateway-bridge/install/config/).

## 2.2.0

**Features:**

* LoRa Gateway Bridge now publishes TX acknowledgement messages over MQTT.
  See [MQTT topics](https://docs.loraserver.io/lora-gateway-bridge/use/data/).

* TX (GPS) `timeSinceGPSEpoch` field is now exposed to transmit at given
  time since GPS epoch (1980-01-06, only possible when the gateway
  has a GPS time source).

* RX (GPS) `timeSinceGPSEpoch` field is now exposed, containing the time
  since GPS epoch (1980-01-06, only available when the gateway has a GPS
  time source).

**Bugfixes:**

* Without GPS time source, the gateway would use `0001-01-01T00:00:00Z`
  as RX `time`. The `time` field is now omitted when unavailable.


## 2.1.6

**Features:**

* Add Kerlink iBTS compatibility (multi antenna and multi board).

## 2.1.5

**Improvements:**

* `--mqtt-ca-cert` / `MQTT_CA_CERT` configuration flag was added to
  specify an optional CA certificate
  (thanks [@minggi](https://github.com/minggi)).

**Bugfixes:**

* MQTT client library update which fixes an issue where during a failed
  re-connect the protocol version would be downgraded
  ([paho.mqtt.golang#116](https://github.com/eclipse/paho.mqtt.golang/issues/116)).

## 2.1.4

* Retry connecting to MQTT broker on startup (thanks @jcampanell-cablelabs).
* Make latitude, longitude and altitude optional as this data is not always
  provided by the packet_forwarder and would else incorrectly be set to `0`.

## 2.1.3

* Provide `TXPacketsReceived` and `TXPacketsEmitted` in stats.
* Implement `--skip-crc-check` / `SKIP_CRC_CHECK` config flag to disable CRC
  checks by LoRa Gateway Bridge.

## 2.1.2

* Add optional `iPol` field to `txInfo` struct in JSON to override the default
  behaviour (which is `iPol=true` when using LoRa modulation)

## 2.1.1

* Do not unmarshal and marshal PHYPayload on receiving / sending
* Set FDev field when using FSK modulation ([#16](https://github.com/brocaar/lora-gateway-bridge/issues/16))
* Omit empty time field ([#15](https://github.com/brocaar/lora-gateway-bridge/issues/16))

## 2.1.0

* Support protocol v1 & v2 simultaneously.

## 2.0.2

* Rename from `lora-semtech-bridge` to `lora-gateway-bridge`

## 2.0.1

* Update `lorawan` vendor to fix a mac command related marshaling issue.

## 2.0.0

* Update to Semtech UDP protocol v2. This is the protocol version used
  since `packet_forwarder` 3.0.0 (which implements just-in-time scheduling).

## 1.1.3

* Minor log related improvements.

## 1.1.2

* Provide binaries for multiple platforms.

## 1.1.1

* Rename `DataRate` to `BitRate` (FSK modulation).

## 1.1.0

* Change from [GOB](https://golang.org/pkg/encoding/gob/) to JSON.

## 1.0.1

* Update MQTT vendor to fix various connection issues.

## 1.0.0

Initial release.
