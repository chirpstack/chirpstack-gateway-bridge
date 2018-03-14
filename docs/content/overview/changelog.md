---
title: Changelog
menu:
    main:
        parent: overview
        weight: 2
---

# Changelog

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
