# ChirpStack Gateway Bridge

[![CircleCI](https://circleci.com/gh/brocaar/chirpstack-gateway-bridge.svg?style=svg)](https://circleci.com/gh/brocaar/chirpstack-gateway-bridge)

ChirpStack Gateway Bridge is a service which converts LoRa<sup>&reg;</sup> Packet Forwarder protocols
into a ChirpStack Network Server [common data-format](https://github.com/brocaar/chirpstack-network-server/blob/master/api/gw/gw.proto) (JSON and Protobuf).
This component is part of the ChirpStack open-source LoRaWAN<sup>&reg;</sup> Network Server stack.

## Backends

The following packet-forwarder backends are provided:

* [Semtech UDP packet-forwarder](https://github.com/Lora-net/packet_forwarder)
* [Basic Station packet-forwarder](https://github.com/lorabasics/basicstation)

## Integrations

The following integrations are provided:

* Generic MQTT broker
* [GCP Cloud IoT Core MQTT Bridge](https://cloud.google.com/iot-core/)

## Architecture

![architecture](https://www.chirpstack.io/img/graphs/architecture.png)

### Component links

* [ChirpStack Gateway Bridge](https://www.chirpstack.io/gateway-bridge/)
* [ChirpStack Network Server](https://www.chirpstack.io/network-server/)
* [ChirpStack Application Server](https://www.chirpstack.io/application-server/)

## Links

* [Downloads](https://www.chirpstack.io/gateway-bridge/overview/downloads/)
* [Docker image](https://hub.docker.com/r/chirpstack/chirpstack-gateway-bridge/)
* [Documentation](https://www.chirpstack.io/gateway-bridge/)
* [Building from source](https://www.chirpstack.io/gateway-bridge/community/source/)
* [Contributing](https://www.chirpstack.io/gateway-bridge/community/contribute/)
* Support
  * [Community forum](https://forum.chirpstack.io)
  * [Bug or feature requests](https://github.com/brocaar/chirpstack-gateway-bridge/issues)

## Sponsors

[![CableLabs](https://www.chirpstack.io/img/sponsors/cablelabs.png)](https://www.cablelabs.com/)
[![SIDNFonds](https://www.chirpstack.io/img/sponsors/sidn_fonds.png)](https://www.sidnfonds.nl/)
[![acklio](https://www.chirpstack.io/img/sponsors/acklio.png)](http://www.ackl.io/)

## License

ChirpStack Gateway Bridge is distributed under the MIT license. See 
[LICENSE](https://github.com/brocaar/chirpstack-gateway-bridge/blob/master/LICENSE).
