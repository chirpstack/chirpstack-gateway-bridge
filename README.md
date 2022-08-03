# ChirpStack Gateway Bridge

![Tests](https://github.com/brocaar/chirpstack-gateway-bridge/actions/workflows/main.yml/badge.svg?branch=master)

ChirpStack Gateway Bridge is a service which converts LoRa<sup>&reg;</sup> Packet Forwarder protocols
into a ChirpStack [common data-format](https://github.com/chirpstack/chirpstack/blob/master/api/proto/gw/gw.proto) (JSON and Protobuf).
This component is part of the ChirpStack open-source LoRaWAN<sup>&reg;</sup> Network Server project.

## Backends

The following packet-forwarder backends are provided:

* [Semtech UDP packet-forwarder](https://github.com/Lora-net/packet_forwarder)
* [Basic Station packet-forwarder](https://github.com/lorabasics/basicstation)
* [ChirpStack Concentratord](https://github.com/brocaar/chirpstack-concentratord/)

## Integrations

The following integrations are provided:

* Generic MQTT broker
* [GCP Cloud IoT Core MQTT Bridge](https://cloud.google.com/iot-core/)
* [Azure IoT Hub](https://azure.microsoft.com/en-us/services/iot-hub/)

## Documentation

Please refer to the [ChirpStack documentation](https://www.chirpstack.io/) for
more information.

## License

ChirpStack Gateway Bridge is distributed under the MIT license. See 
[LICENSE](https://github.com/brocaar/chirpstack-gateway-bridge/blob/master/LICENSE).
