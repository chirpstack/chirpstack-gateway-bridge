---
title: Send / receive data
menu:
    main:
        parent: use
        weight: 1
---

# MQTT topics

## Using MQTT

To receive data from your gateways, you need to subscribe to its MQTT topic(s).
For debugging, you can use a (command-line) tool like `mosquitto_sub`
which is part of the [Mosquitto](http://mosquitto.org/) MQTT broker.

Use ``+`` for a single-level wildcard, ``#`` for a multi-level wildcard.
Examples:

```bash
# show data from all gateways 
mosquitto_sub -t "gateway/#" -v

# show all data for the given gateway
mosquitto_sub -t "gateway/0101010101010101/+" -v
```

## Data encodings

As of LoRa Gateaway Bridge v2.5.0 it is possible to configure the data
serialization format to use for the messages sent and received over MQTT.
This can be configured by the `marshaler` option in the [configuration]({{<ref "install/config.md">}})
file.

#### `v2_json`

The default JSON format for LoRa Gateway Bridge v2.x.

#### `protobuf`

[Protocol Buffers](https://developers.google.com/protocol-buffers/) serialization
format. This is a binary format which is more bandwidth efficient. Using the
Protocol Buffers tooling, it is possible to generate code for various
programming languages. This is planned to become the default serialization
format in the future.

#### `json`

The JSON serialization format uses the [JSON Mapping](https://developers.google.com/protocol-buffers/docs/proto3#json)
defined by the Protocol Buffers. Messages are larger in size than the `protobuf`
option, but are easier when performing debugging.

**Notes:**

* `v2_json` and `json` are not compatible.
* The Protocol Buffer JSON Mapping defines that bytes fields must be base64
  encoded. This means that also the `gatewayID` will be base64 encoded.

### Messages

#### Gateway statistics

Messages are published to the topic defined by the
`stats_topic_template` setting (see [configuration]({{<ref "install/config.md">}})).

##### v2_json

Example:

```json
{
    "altitude": 10,                 // only available when gateway has gps
    "latitude": 52.3740364,         // only available when gateway has gps
    "longitude": 4.9144401,         // only available when gateway has gps
    "mac": "1dee08d0b691d149",
    "rxPacketsReceived": 20,
    "rxPacketsReceivedOK": 15,
    "txPacketsReceived": 10,
    "txPacketsEmitted": 9,
    "time": "2016-06-10T14:04:53Z"
}
```


##### protobuf

This message is defined by the [GatewayStats](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.

##### json

Example:

```json
{
    "gatewayID": "cnb/AC4GLBg=",
    "time": "2018-07-26T13:36:31Z",
    "location": {                   // only set when gateway has GPS
        "latitude": 1.12345,
        "longitude": 2.12345,
        "altitude": 10,
        "source": "GPS",
    },
    "configVersion": "1.2.3",       // maps to the 'Gateway configuration' message version
    "rxPacketsReceived": 4,
    "rxPacketsReceivedOK": 1,
    "txPacketsReceived": 0,
    "txPacketsEmitted": 1
}
```

#### Uplink frames

Messages are published to the topic defined by the
`uplink_topic_template` setting (see [configuration]({{<ref "install/config.md">}})).

##### v2_json

Example:

```json
{
    "phyPayload": "AAEBAQEBAQEBAgICAgICAgJpNbxrAh8=",  // base64 encoded LoRaWAN packet
    "rxInfo": {
        "board": 0,
        "antenna": 0,
        "channel": 1,
        "codeRate": "4/5",
        "crcStatus": 1,
        "dataRate": {
            "bandwidth": 125,
            "modulation": "LORA",
            "spreadFactor": 7
        },
        "frequency": 868300000,
        "loRaSNR": 7,
        "mac": "1dee08d0b691d149",
        "rfChain": 1,
        "rssi": -57,
        "size": 23,
        "time": "2017-12-12T15:28:53.222434Z",         // timestamp (only set when the gateway has a GPS time source)
        "timeSinceGPSEpoch": "332535h29m12.222s",      // time since GPS epoch (only set when the gateway has a GPS time source)
        "timestamp": 2074240683                        // gateway internal timestamp (32 bit) with microsecond precision
    }
}
```

##### protobuf

This message is defined by the [UplinkFrame](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.

##### json

Example:

```json
{
    "phyPayload": "AAEBAQEBAQEBAQEBAQEBAQGXFgzLPxI=",  // base64 encoded LoRaWAN frame
    "txInfo": {
        "frequency": 868300000,
        "modulation": "LORA",
        "loRaModulationInfo": {
            "bandwidth": 125,
            "spreadingFactor": 11,
            "codeRate": "4/5",
            "polarizationInversion": false
        }
    },
    "rxInfo": {
        "gatewayID": "cnb/AC4GLBg=",
        "time": "2018-07-26T15:15:58.599497Z",         // only set when the gateway has a GPS time source
        "timestamp": 58692860,                         // gateway internal timestamp (23 bit)
        "rssi": -55,
        "loRaSNR": 15,
        "channel": 2,
        "rfChain": 0,
        "board": 0,
        "antenna": 0,
        "fineTimestampType": "ENCRYPTED",
        "encryptedFineTimestamp": {
            "aesKeyIndex": 0,
            "encryptedNS": "d2YFe51PraE3EpnrZJV4aw=="  // encrypted nanosecond part of the time
        }
    }
}
```

#### Downlink frames

This topic is used when you would like to send a LoRaWAN frame using a given
LoRa Gateway.

Messages are expected on the topic defined by the
`downlink_topic_template` setting (see [configuration]({{<ref "install/config.md">}})).

When `immediately` is set to `false`, either the `timestamp` **or** the
`timeSinceGPSEpoch` field must be present to tell the gateway at what (internal) time
the frame must be transmitted.

##### v2_json

Example:

```json
{
    "token": 65535,                                // random token (uint16), used for acknowledgements
    "phyPayload": "IKu70cumKom7BREUFrxlHtM=",      // base64 encoded LoRaWAN frame
    "txInfo": {
        "board": 0,
        "antenna": 0,
        "codeRate": "4/5",
        "dataRate": {
            "bandwidth": 125,
            "modulation": "LORA",
            "spreadFactor": 7
        },
        "frequency": 868300000,
        "immediately": false,
        "mac": "1dee08d0b691d149",
        "power": 14,
        "timestamp": 2079240683,                  // gateway internal timestamp for transmission -OR-
        "timeSinceGPSEpoch": "332535h29m12.222s"  // time since GPS epoch (only when the gateway has a GPS time source)
    }
}
```

Optionally, the field `iPol` (type `bool`) can be used to control the
LoRa modulation polarization inversion. When left blank (`null`), the default
will be used (which is `true` for downlink LoRa modulation).

##### protobuf

This message is defined by the [DownlinkFrame](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.

##### json

Example:

```json
{
    "phyPayload": "IHN792Ld0vEHetyVv9+llJnnmz88Up6pFz8UiUdJMnUc",
    "txInfo": {
        "gatewayID": "cnb/AC4GLBg=",
        "immediately": false,
        "timeSinceGPSEpoch": null,
        "timestamp": 3240216372,
        "frequency": 868500000,
        "power": 14,
        "modulation": "LORA",
        "loRaModulationInfo": {
            "bandwidth": 125,
            "spreadingFactor": 11,
            "codeRate": "4/5",
            "polarizationInversion": true
        },
        "board": 0,
        "antenna": 0
    },
    "token": 38150
}
```


#### Downlink acknowledgements

Messages are published to the topic defined by the
`ack_topic_template` setting (see [configuration]({{<ref "install/config.md">}})).

**Note:** this topic is only available for gateways implementing v2 of
the packet-forwarder [PROTOCOL.txt](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT).

Possible error values are:

* `TOO_LATE`: Rejected because it was already too late to program this packet for downlink
* `TOO_EARLY`: Rejected because downlink packet timestamp is too much in advance
* `COLLISION_PACKET`: Rejected because there was already a packet programmed in requested timeframe
* `COLLISION_BEACON`: Rejected because there was already a beacon planned in requested timeframe
* `TX_FREQ`: Rejected because requested frequency is not supported by TX RF chain
* `TX_POWER`: Rejected because requested power is not supported by gateway
* `GPS_UNLOCKED`: Rejected because GPS is unlocked, so GPS timestamp cannot be used

##### v2_json

Example:

```json
{
    "token": 65535,              // same token as used in downlink
    "error": "COLLISION_PACKET"  // not set in case of acknowledgement
}
```

#### protobuf

This message is defined by the [DownlinkTXAck](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.


#### json

Example:

```json
{
    "gatewayID": "cnb/AC4GLBg=",
    "token": 12345,
    "error": "GPS_UNLOCKED"
}
```

#### Gateway configuration

This message can be used when you would like to re-configure the packet-forwarder
channel-plan. 

The LoRa Gateway Bridge assumes this configuation will be applied to an 8-channel
gateway (+ 1 single-SF LoRa and 1 FSK channel). Make sure that the channel-plan
fits within the bandwidth of both radios of the gateway.

**Note:** in order to configure the packet-forwarder over MQTT, don't forget
to configure at least one `[[packet_forwarder.configuration]]` section in the
[configuration]({{<ref "install/config.md">}}) file.

Messages are expected on the topic defined by the
`config_topic_template` setting (see [configuration]({{<ref "install/config.md">}})).

##### v2_json

Example:

```json
{
    "mac": "1dee08d0b691d149",
    "version": "1.2.3",
    "channels": [
        {
            "modulation": "LORA",
            "frequency": 868100000,
            "bandwidth": 125,
            "spreadingfactors": [7, 8, 9, 10, 11, 12]
        },
        {
            "modulation": "LORA",
            "frequency": 868300000,
            "bandwidth": 125,
            "spreadingfactors": [7, 8, 9, 10, 11, 12]
        },
        {
            "modulation": "LORA",
            "frequency": 868500000,
            "bandwidth": 125,
            "spreadingfactors": [7, 8, 9, 10, 11, 12]
        },
        {
            "modulation": "LORA",
            "frequency": 867100000,
            "bandwidth": 125,
            "spreadingfactors": [7, 8, 9, 10, 11, 12]
        },
        {
            "modulation": "LORA",
            "frequency": 867300000,
            "bandwidth": 125,
            "spreadingfactors": [7, 8, 9, 10, 11, 12]
        },
        {
            "modulation": "LORA",
            "frequency": 867500000,
            "bandwidth": 125,
            "spreadingfactors": [7, 8, 9, 10, 11, 12]
        },
        {
            "modulation": "LORA",
            "frequency": 867700000,
            "bandwidth": 125,
            "spreadingfactors": [7, 8, 9, 10, 11, 12]
        },
        {
            "modulation": "LORA",
            "frequency": 867900000,
            "bandwidth": 125,
            "spreadingfactors": [7, 8, 9, 10, 11, 12]
        },
        {
            "modulation": "LORA",
            "frequency": 868300000,
            "bandwidth": 250,
            "spreadingfactors": [7]
        },
        {
            "modulation": "FSK",
            "frequency": 868800000,
            "bandwidth": 125,
            "bitrate": 50000
        }
    ]
}
```

##### protobuf

This message is defined by the [GatewayConfiguration](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.

##### json

Example:

```json
{
    "gatewayID": "cnb/AC4GLBg=",
    "version": "1.2.3",
    "channels": [
        {
            "frequency": 868100000,
            "modulation": "LORA",
            "loRaModulationConfig": {
                "bandwidth": 125,
                "spreadingFactors": [7, 8, 9, 10, 11, 12]
            }
        },
        {
            "frequency": 868300000,
            "modulation": "LORA",
            "loRaModulationConfig": {
                "bandwidth": 125,
                "spreadingFactors": [7, 8, 9, 10, 11, 12]
            }
        },
        {
            "frequency": 868500000,
            "modulation": "LORA",
            "loRaModulationConfig": {
                "bandwidth": 125,
                "spreadingFactors": [7, 8, 9, 10, 11, 12]
            }
        },
        {
            "frequency": 867100000,
            "modulation": "LORA",
            "loRaModulationConfig": {
                "bandwidth": 125,
                "spreadingFactors": [7, 8, 9, 10, 11, 12]
            }
        },
        {
            "frequency": 867300000,
            "modulation": "LORA",
            "loRaModulationConfig": {
                "bandwidth": 125,
                "spreadingFactors": [7, 8, 9, 10, 11, 12]
            }
        },
        {
            "frequency": 867500000,
            "modulation": "LORA",
            "loRaModulationConfig": {
                "bandwidth": 125,
                "spreadingFactors": [7, 8, 9, 10, 11, 12]
            }
        },
        {
            "frequency": 867700000,
            "modulation": "LORA",
            "loRaModulationConfig": {
                "bandwidth": 125,
                "spreadingFactors": [7, 8, 9, 10, 11, 12]
            }
        },
        {
            "frequency": 867900000,
            "modulation": "LORA",
            "loRaModulationConfig": {
                "bandwidth": 125,
                "spreadingFactors": [7, 8, 9, 10, 11, 12]
            }
        },
        {
            "frequency": 868300000,
            "modulation": "LORA",
            "loRaModulationConfig": {
                "bandwidth": 250,
                "spreadingFactors": [
                    7
                ]
            }
        },
        {
            "frequency": 868800000,
            "modulation": "FSK",
            "fskModulationConfig": {
                "bandwidth": 125,
                "bitrate": 50000
            }
        }
    ]
}
```
