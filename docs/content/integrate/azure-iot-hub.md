---
title: Azure IoT Hub
menu:
  main:
    parent: integrate
    weight: 3
description: Setting up the ChirpStack Gateway Bridge using the Azure IoT Hub MQTT protocol.
---

# Azure IoT Hub

The Azure [IoT Hub](https://azure.microsoft.com/en-us/services/iot-hub/)
authentication thype must be used when connecting with the
[IoT Hub MQTT interface](https://docs.microsoft.com/en-us/azure/iot-hub/iot-hub-mqtt-support).

## Limitations

* As you need to setup the device ID (in this case the device is the gateway)
  when provisioning the device (LoRa<sup>&reg;</sup> Gateway) in Cloud IoT Core,
  this does not allow to connect multiple LoRa Gateways to a single ChirpStack Gateway
  Bridge instance.

## Conventions

### Device ID naming

The IoT Hub Device ID must match the Gateway ID (e.g. `0102030405060708`).
It must be entered in lowercase (the IoT Hub Device ID is case-sensitive).

### MQTT topics

When the Azure IoT Hub authentication type has been configured, ChirpStack Gateway
Bridge will use MQTT topics which are expected by Azure IoT Hub and will
ignore the MQTT topic configuration from the `chirpstack-gateway-bridge.toml`
configuration file.

#### Uplink topics

* `devices/[GATEWAY_ID]/messages/events/up`: uplink frame
* `devices/[GATEWAY_ID]/messages/events/stats`: gateway statistics
* `devices/[GATEWAY_ID]/messages/events/ack`: downlink frame acknowledgements (scheduling)

#### Downlink topics

* `devices/[GATEWAY_ID]/messages/devicebound/down`: scheduling downlink frame transmission
* `devices/[GATEWAY_ID]/messages/devicebound/config`: gateway configuration

