---
title: V2 JSON
menu:
    main:
        parent: payload-types
---

# V2 JSON

This is the default JSON format for LoRa Gateway Bridge v2.x. It will be
removed in the next major LoRa Gateway Bridge release in favor of
the [json]({{<relref "json.md">}}) and [protobuf]({{<relref "protobuf.md">}})
format.

## Payload examples

### Gateway statistics

{{<highlight json>}}
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
{{< /highlight >}}

### Uplink frames

{{<highlight json>}}
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
{{< /highlight >}}

### Downlink frames

{{<highlight json>}}
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
{{< /highlight >}}

Optionally, the field `iPol` (type `bool`) can be used to control the
LoRa modulation polarization inversion. When left blank (`null`), the default
will be used (which is `true` for downlink LoRa modulation).

### Downlink acknowledgements

{{<highlight json>}}
{
    "token": 65535,              // same token as used in downlink
    "error": "COLLISION_PACKET"  // not set in case of acknowledgement
}
{{< /highlight >}}

### Gateway configuration

{{<highlight json>}}
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
{{< /highlight >}}
