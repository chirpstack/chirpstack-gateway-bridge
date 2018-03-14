---
title: Send / receive data
menu:
    main:
        parent: use
        weight: 1
---

# MQTT topics

To receive data from your gateways, you need to subscribe to its MQTT topic(s).
For debugging, you can use a (command-line) tool like ``mosquitto_sub``
which is part of the [Mosquitto](http://mosquitto.org/) MQTT broker.

Use ``+`` for a single-level wildcard, ``#`` for a multi-level wildcard.
Examples:

```bash
mosquitto_sub -t "gateway/#" -v                   # show data from all gateways
mosquitto_sub -t "gateway/0101010101010101/+" -v  # show all data for the given gateway
```

**Note that the MQTT topics are lowercase.**

## gateway/[mac]/stats

Topic for gateway statistics. Example payload:

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

## gateway/[mac]/rx

Topic for received packets (from nodes). Example payload:

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

## gateway/[mac]/tx

Topic for publishing packets to be transmitted by the given gateway.
Example payload:

```json
{
    "token": 65535,  // random token (uint16), used for acknowledgements
    "phyPayload": "IKu70cumKom7BREUFrxlHtM=",
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

When `immediately` is set to `false`, either the `timestamp` **or** the
`timeSinceGPSEpoch` field must be present to tell the gateway at what (internal) time
the frame must be transmitted.

Optionally, the field `iPol` (type `bool`) can be used to control the
LoRa modulation polarization inversion. When left blank (`null`), the default
will be used (which is `true` for downlink LoRa modulation.

## gateway/[mac]/ack

Topic for received TX acknowledgements (or TX errors). Example payload:

```json
{
    "token": 65535,              // same token used during transmission
    "error": "COLLISION_PACKET"  // not set in case of acknowledgement
}
```

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
