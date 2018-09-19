---
title: Requirements
menu:
    main:
        parent: install
        weight: 1
description: Instructions how to setup the LoRa Gateway Bridge requirements.
---

# Requirements

## MQTT broker

LoRa Gateway Bridge makes use of MQTT for publishing and receivng application
payloads. [Mosquitto](http://mosquitto.org/) is a popular open-source MQTT
server, but any MQTT broker implementing MQTT 3.1.1 should work. 
In case you install Mosquitto, make sure you install a **recent** version.

### Install

#### Debian / Ubuntu

In order to install Mosquitto, execute the following command:

{{<highlight bash>}}
sudo apt-get install mosquitto
{{< /highlight >}}

#### Other platforms

Please refer to the [Mosquitto download](https://mosquitto.org/download/) page
for information about how to setup Mosquitto for your platform.
