---
title: Payload types
menu:
    main:
        parent: integrate
        identifier: payload-types
        weight: 1
---

# Payload types

As of LoRa Gateaway Bridge v2.5.0 it is possible to configure the data
serialization format to use for the messages sent and received over MQTT.
This can be configured by the `marshaler` option in the [configuration]({{<ref "/install/config.md">}})
file. The following marshalers are available:

* [V2 JSON]({{<relref "v2-json.md">}}) (will be removed in the next major release)
* [JSON]({{<relref "json.md">}})
* [Protocol Buffers]({{<relref "protobuf.md">}})

## Payload types

### Gateway statistics

Messages are published to the topic defined by the
`stats_topic_template` setting (see [configuration]({{<ref "/install/config.md">}})).

#### Uplink frames

Messages are published to the topic defined by the
`uplink_topic_template` setting (see [configuration]({{<ref "/install/config.md">}})).

#### Downlink frames

This topic is used when you would like to send a LoRaWAN frame using a given
LoRa Gateway. Messages are expected on the topic defined by the
`downlink_topic_template` setting (see [configuration]({{<ref "/install/config.md">}})).
When `immediately` is set to `false`, either the `timestamp` **or** the
`timeSinceGPSEpoch` field must be present to tell the gateway at what (internal) time
the frame must be transmitted.

#### Downlink acknowledgements

Messages are published to the topic defined by the
`ack_topic_template` setting (see [configuration]({{<ref "/install/config.md">}})).
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

#### Gateway configuration

This message can be used when you would like to re-configure the packet-forwarder
channel-plan. 
The LoRa Gateway Bridge assumes this configuation will be applied to an 8-channel
gateway (+ 1 single-SF LoRa and 1 FSK channel). Make sure that the channel-plan
fits within the bandwidth of both radios of the gateway.
Messages are expected on the topic defined by the
`config_topic_template` setting (see [configuration]({{<ref "/install/config.md">}})).

**Note:** in order to configure the packet-forwarder over MQTT, don't forget
to configure at least one `[[packet_forwarder.configuration]]` section in the
[configuration]({{<ref "/install/config.md">}}) file.
