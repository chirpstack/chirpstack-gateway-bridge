---
title: ChirpStack Gateway Bridge
menu:
    main:
        parent: overview
        weight: 1
listPages: false
---

# ChirpStack Gateway Bridge

ChirpStack Gateway Bridge is a service which converts LoRa<sup>&reg;</sup> Packet Forwarder protocols
into a ChirpStack Network Server [common data-format](https://github.com/brocaar/chirpstack-network-server/blob/master/api/gw/gw.proto) (JSON and Protobuf).
This component is part of the ChirpStack open-source LoRaWAN<sup>&reg;</sup> Network Server stack.

## Backends

The following Packet Forwarder backends are provided:

* [Semtech UDP Packet Forwarder](https://github.com/Lora-net/packet_forwarder)
* [Basic Station Packet Forwarder](https://github.com/lorabasics/basicstation)

## Integrations

The following integrations are provided:

* Generic MQTT broker
* [GCP Cloud IoT Core MQTT Bridge](https://cloud.google.com/iot-core/)
* [Azure IoT Hub MQTT Bridge](https://docs.microsoft.com/en-us/azure/iot-hub/iot-hub-mqtt-support)
