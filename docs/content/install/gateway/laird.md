---
title: Laird
menu:
    main:
      parent: gateway
---

# Laird

## Sentrius RG1XX LoRa gateway

* [Product detail page](https://www.lairdtech.com/products/rg1xx-lora-gateway)

**Note:** The LoRa Gateway Bridge component comes pre-installed since firmware version
[93.7.2.9](https://assets.lairdtech.com/home/brandworld/files/CONN-RN-RG1xx-laird-93_7_2_9.pdf).
If your gateway is running an older version, please update it first.

The pre-installed LoRa Gateway Bridge can be configured through the gateway
web-interface using the following steps:

* Login into the gateway web-interface
* In the top navigation bar, click on **LoRa**
* In the left navigation menu, click **Forwarder**
* From the *Mode* dropdown, select **MQTT Forwarder**
* Enter your MQTT **Server Address** and optionally the MQTT **Username** and **Password**

Please see [configuration]({{<ref "install/config.md">}}) for valid
configuration options. The fields map to the following configuration options:

* Server Address:  `--mqtt-server`
* Username: `--mqtt-username`
* Password: `--mqtt-password`
