# LoRa Gateway Bridge documentation

LoRa Gateway Bridge is a service which abstracts the 
[packet_forwarder UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT)
running on most LoRa gateways into JSON over MQTT. It enables you to use MQTT for
receiving data from and sending data to your gateways.
This project is part of [LoRa Server](http://docs.loraserver.io/loraserver/).

## Compatibility

The table below shows the compatibility between LoRa Gateway Bridge and the
packet_forwarder UDP protocol versions:

| LoRa Gateway Bridge | packet_forwarder protocol version | Note                                                                |
|---------------------|-----------------------------------|---------------------------------------------------------------------|
| 1.x.x               | 1                                 |                                                                     |
| 2.0.x               | 2                                 | This protocol is used since version 3.0.0 of the `packet_forwarder` |
| >= 2.1.x            | 1 & 2 simultaneously              | Both protocol versions are supported and auto-detected              |

## Downloads

Pre-compiled binaries are available from the [releases](https://github.com/brocaar/lora-gateway-bridge/releases) page:

* Linux (including ARM / Raspberry Pi)
* OS X
* Windows

Source-code can be found at [https://github.com/brocaar/lora-gateway-bridge](https://github.com/brocaar/lora-gateway-bridge).

## Issues / feature-requests

Issues or feature-requests can be opened at [https://github.com/brocaar/lora-gateway-bridge/issues](https://github.com/brocaar/lora-gateway-bridge/issues).

## License

LoRa Gateway Bridge is distributed under the MIT license. See 
[LICENSE](https://github.com/brocaar/lora-gateway-bridge/blob/master/LICENSE).
