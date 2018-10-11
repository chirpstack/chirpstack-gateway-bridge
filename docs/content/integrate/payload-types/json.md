---
title: JSON
menu:
    main:
        parent: payload-types
---

# JSON

This JSON serialization format uses the [JSON Mapping](https://developers.google.com/protocol-buffers/docs/proto3#json)
defined by the Protocol Buffers. Messages are larger in size than the [protobuf]({{<relref "protobuf.md">}})
type, but are easier when performing debugging.

## Notes

* The Protocol Buffers JSON Mapping defies that bytes must be encoded as base64
  strings. This also affects the `gatewayID` field. When re-encoding this filed
  to HEX encoding, you will find the expected gateway ID string.

## Payload examples

### Gateway statistics

{{<highlight json>}}
{
    "gatewayID": "cnb/AC4GLBg=",
    "ip": "192.168.1.5",
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
{{< /highlight >}}

### Uplink frames

{{<highlight json>}}
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
{{< /highlight >}}

### Downlink frames

{{<highlight json>}}
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
{{< /highlight >}}

### Downlink acknowledgements

{{<highlight json>}}
{
    "gatewayID": "cnb/AC4GLBg=",
    "token": 12345,
    "error": "GPS_UNLOCKED"
}
{{< /highlight >}}

### Gateway configuration

{{<highlight json>}}
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
{{< /highlight >}}