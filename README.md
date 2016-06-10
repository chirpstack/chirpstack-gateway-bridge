# LoRa Semtech Bridge

[![Build Status](https://travis-ci.org/brocaar/lora-semtech-bridge.svg?branch=master)](https://travis-ci.org/brocaar/lora-semtech-bridge)
[![Documentation Status](https://readthedocs.org/projects/lora-semtech-bridge/badge/?version=latest)](http://lora-semtech-bridge.readthedocs.org/en/latest/?badge=latest)
[![Documentation Status](https://readthedocs.org/projects/lora-semtech-bridge/badge/?version=stable)](http://lora-semtech-bridge.readthedocs.org/en/stable/?badge=stable)
[![GoDoc](https://godoc.org/github.com/brocaar/lora-semtech-bridge/cmd/semtech-bridge?status.svg)](https://godoc.org/github.com/brocaar/lora-semtech-bridge/cmd/semtech-bridge)

LoRa Semtech Bridge is a service which abstracts the 
[Semtech protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT)
into JSON over MQTT. This project is part of [LoRa Server](https://github.com/brocaar/loraserver).


## Documentation

See the [http://lora-semtech-bridge.readthedocs.io/](http://lora-semtech-bridge.readthedocs.io/)
for documentation about setting up LoRa Semtech Bridge.

## Compatibility

The table below shows the compatibility between LoRa Semtech Bridge and the
available Semtech UDP protocol versions:

| LoRa Semtech Bridge | Semtech UDP protocol version | Note                                                                |
|---------------------|------------------------------|---------------------------------------------------------------------|
| 1.x.x               | 1                            |                                                                     |
| 2.x.x               | 2                            | This protocol is used since version 3.0.0 of the `packet_forwarder` |

## Downloads

Pre-compiled binaries are available for:

* Linux (and ARM build for e.g. Raspberry Pi)
* OS X
* Windows

See [releases](https://github.com/brocaar/lora-semtech-bridge/releases).

## License

LoRa Server is licensed under the MIT license. See ``LICENSE``.
