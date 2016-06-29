LoRa Semtech Bridge is a service which abstracts the 
[Semtech UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT)
used by the [packet_forwarder](https://github.com/Lora-net/packet_forwarder/)
which is running on most LoRa gateways. It enables you to use MQTT for
receiving data from and sending data to your gateways (JSON encoded).
This project is part of [LoRa Server](https://github.com/brocaar/loraserver).

## Compatibility

The table below shows the compatibility between LoRa Semtech Bridge and the
available Semtech UDP protocol versions:

| LoRa Semtech Bridge | Semtech UDP protocol version | packet_forwarder version  |
|---------------------|------------------------------|---------------------------|
| 1.x.x               | 1                            | < 3.0.0                   |
| 2.x.x               | 2                            | >= 3.0.0                  |

LoRa Semtech Bridge v1.x.x will be update with bugfixes for the near future as
not all gateways are using packet_forwarder v3.0.0+ yet.

## Downloads

Pre-compiled binaries are available from the [releases](https://github.com/brocaar/lora-semtech-bridge/releases) page:

* Linux (including ARM / Raspberry Pi)
* OS X
* Windows

Source-code can be found at [https://github.com/brocaar/lora-semtech-bridge](https://github.com/brocaar/lora-semtech-bridge).

## Issues / feature-requests

Issues or feature-requests can be opened at [https://github.com/brocaar/lora-semtech-bridge/issues](https://github.com/brocaar/lora-semtech-bridge/issues).

## License

LoRa Semtech Bridge is distributed under the MIT license. See 
[LICENSE](https://github.com/brocaar/lora-semtech-bridge/blob/master/LICENSE).

