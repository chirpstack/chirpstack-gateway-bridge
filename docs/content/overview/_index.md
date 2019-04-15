---
title: LoRa Gateway Bridge
menu:
    main:
        parent: overview
        weight: 1
---

# LoRa Gateway Bridge

LoRa Gateway Bridge is a service which converts LoRa packet-forwarder protocols
into a LoRa Server [common protocol](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto) (JSON and Protobuf).
This project is part of the [LoRa Server](https://github.com/brocaar/loraserver)
project.

## Backends

The following packet-forwarder backends are provided:

* [Semtech UDP packet-forwarder](https://github.com/Lora-net/packet_forwarder)
* [Basic Station packet-forwarder](https://github.com/lorabasics/basicstation)

## Integrations

The following integrations are provided:

* Generic MQTT broker
* [GCP Cloud IoT Core MQTT Bridge](https://cloud.google.com/iot-core/)
