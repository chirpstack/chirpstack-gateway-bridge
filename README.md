# LoRa Gateway Bridge

[![Build Status](https://travis-ci.org/brocaar/lora-gateway-bridge.svg?branch=master)](https://travis-ci.org/brocaar/lora-gateway-bridge)
[![GoDoc](https://godoc.org/github.com/brocaar/lora-gateway-bridge/cmd/gateway-bridge?status.svg)](https://godoc.org/github.com/brocaar/lora-gateway-bridge/cmd/lora-gateway-bridge)

LoRa Gateway Bridge is a service which abstracts the 
[packet_forwarder UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT)
running on most LoRa gateways into JSON over MQTT. It enables you to use MQTT for
receiving data from and sending data to your gateways.
This project is part of [LoRa Server](https://github.com/brocaar/loraserver).

## Documentation

See [http://docs.loraserver.io/lora-gateway-bridge](http://docs.loraserver.io/lora-gateway-bridge)
for documentation about setting up LoRa Gateway Bridge.

## Compatibility

The table below shows the compatibility between LoRa Gateway Bridge and the
packet_forwarder UDP protocol versions:

| LoRa Gateway Bridge | packet_forwarder protocol version | Note                                                                |
|---------------------|-----------------------------------|---------------------------------------------------------------------|
| 1.x.x               | 1                                 |                                                                     |
| 2.x.x               | 2                                 | This protocol is used since version 3.0.0 of the `packet_forwarder` |


LoRa Gateway Bridge v1.x.x will be update with bugfixes for the near future as
not all gateways are using packet_forwarder v3.0.0+ yet.

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
