---
title: Requirements
menu:
    main:
        parent: install
        weight: 1
---

# Requirements

## MQTT broker

LoRa Gateway Bridge makes use of MQTT for publishing and receivng application
payloads. [Mosquitto](http://mosquitto.org/) is a popular open-source MQTT
server, but any MQTT broker implementing MQTT 3.1.1 should work. 
In case you install Mosquitto, make sure you install a **recent** version.

### Install

#### Debian / Ubuntu

For Ubuntu Trusty (14.04), execute the following command in order to add the
Mosquitto Apt repository, for Ubuntu Xenial and Debian Jessie you can skip
this step:

```bash
sudo apt-add-repository ppa:mosquitto-dev/mosquitto-ppa
sudo apt-get update
```

In order to install Mosquitto, execute the following command:

```bash
sudo apt-get install mosquitto
```

#### Other platforms

Please refer to the [Mosquitto download](https://mosquitto.org/download/) page
for information about how to setup Mosquitto for your platform.
