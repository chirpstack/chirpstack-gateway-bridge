# LoRa Gateway Bridge

[![Build Status](https://travis-ci.org/brocaar/lora-gateway-bridge.svg?branch=master)](https://travis-ci.org/brocaar/lora-gateway-bridge)

LoRa Gateway Bridge is a service which abstracts the 
[packet_forwarder UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT)
running on most LoRa gateways into JSON over MQTT. It enables you to use MQTT for
receiving data from and sending data to your gateways.
This project is part of the [LoRa Server](https://github.com/brocaar/loraserver)
project.

## Architecture

![architecture](https://docs.loraserver.io/img/architecture.png)

### Component links

* [LoRa Gateway Bridge](https://docs.loraserver.io/lora-gateway-bridge)
* [LoRa Gateway Config](https://docs.loraserver/lora-gateway-config)
* [LoRa Server](https://docs.loraserver.io/loraserver/)
* [LoRa App Server](https://docs.loraserver.io/lora-app-server/)

## Links

* [Downloads](https://docs.loraserver.io/lora-gateway-bridge/overview/downloads/)
* [Docker image](https://hub.docker.com/r/loraserver/lora-gateway-bridge/)
* [Documentation](https://docs.loraserver.io/lora-gateway-bridge/)
* [Building from source](https://docs.loraserver.io/lora-gateway-bridge/community/source/)
* [Contributing](https://docs.loraserver.io/lora-gateway-bridge/community/contribute/)
* Support
  * [Support forum](https://forum.loraserver.io)
  * [Bug or feature requests](https://github.com/brocaar/lora-gateway-bridge/issues)

## Sponsors

[![CableLabs](https://www.loraserver.io/img/sponsors/cablelabs.png)](https://www.cablelabs.com/)
[![SIDNFonds](https://www.loraserver.io/img/sponsors/sidn_fonds.png)](https://www.sidnfonds.nl/)
[![acklio](https://www.loraserver.io/img/sponsors/acklio.png)](http://www.ackl.io/)

## License

LoRa Gateway Bridge is distributed under the MIT license. See 
[LICENSE](https://github.com/brocaar/lora-gateway-bridge/blob/master/LICENSE).
