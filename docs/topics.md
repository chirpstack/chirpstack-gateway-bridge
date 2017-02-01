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

!!! info
	Note that the MQTT topics are lowercase

## gateway/[mac]/stats

Topic for gateway statistics. Example payload:

```json
{
    "altitude": 0,
    "latitude": 0,
    "longitude": 0,
    "mac": "1dee08d0b691d149",
    "rxPacketsReceived": 2,
    "rxPacketsReceivedOK": 1,
    "time": "2016-06-10T14:04:53Z"
}
```

## gateway/[mac]/rx

Topic for received packets (from nodes). Example payload:

```json
{
    "phyPayload": "AAEBAQEBAQEBAgICAgICAgJpNbxrAh8=",  // base64 encoded LoRaWAN packet
    "rxInfo": {
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
        "time": "0001-01-01T00:00:00Z",
        "timestamp": 2074240683                        // gateway internal timestamp (32 bit) with microsecond precision
    }
}
```

## gateway/[mac]/tx

Topic for publishing packets to be transmitted by the given gateway.
Example payload:

```json
{
    "phyPayload": "IKu70cumKom7BREUFrxlHtM=",
    "txInfo": {
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
        "timestamp": 2079240683
    }
}
```

Optionally, the field `iPol` (type `bool`) can be used to control the
LoRa modulation polarization inversion. When left blank (`null`), the default
will be used (which is `true` for downlink LoRa modulation.
