---
title: Laird
description: Configuring the pre-installed ChirpStack Gateway Bridge service on Laird gateways.
menu:
  main:
    parent: gateway
---

# Laird

## Sentrius RG1XX LoRa<sup>&reg;</sup> gateway

* [Product detail page](https://www.lairdtech.com/products/rg1xx-lora-gateway)

The Laird gateway can be configured in two ways:

* MQTT Forwarder
* Semtech Forwarder

**Note:** The first option is not (yet) compatible with ChirpStack Network Server v3 as
the Laird gateway comes with ChirpStack Gateway Bridge v2. In case you would like to
use this gateway with ChirpStack Network Server v3, use the second option.

### MQTT Forwarder

**Note:** The ChirpStack Gateway Bridge v2 component comes pre-installed since firmware version
[93.7.2.9](https://assets.lairdtech.com/home/brandworld/files/CONN-RN-RG1xx-laird-93_7_2_9.pdf).
If your gateway is running an older version, please update it first.

The pre-installed ChirpStack Gateway Bridge can be configured through the gateway
web-interface using the following steps:

* Login into the gateway web-interface.
* In the top navigation bar, click on **LoRa**.
* In the left navigation menu, click **Forwarder**.
* From the *Mode* dropdown, select **MQTT Forwarder**.
* Enter your MQTT **Server Address** and optionally the MQTT **Username** and **Password**.

### Semtech Forwarder

For this option you need to install the ChirpStack Gateway Bridge component first on
a server. See the [Install Documentation](/gateway-bridge/install/) for
more information.

The Semtech (UDP) Forwarder can be configured through the gateway
web-interface using the following steps:

* Login into the gateway web-interface.
* In the top navigation bar, click on **LoRa**.
* In the left navigation menu, click **Forwarder**.
* Fromt he *Mode* dropdown, select **Semtech Forwarder**.
* As **Network Server Address** enter the address of your ChirpStack Gateway Bridge instance.
