---
title: Tektelic
menu:
    main:
        parent: gateway
---

# Tektelic

## KONA Pico IoT Gateway

* [Product detail page](https://tektelic.com/iot/lorawan-gateways/)

The KONA Pico IoT Gateway is able to run different firmwares, each using a
different protocol. In order to work together with the LoRa Gateway Brige,
you must install the `semtech-vx.xx.bin` firmware. These steps have been tested
with version 1.2.1. The latest firmware can be [downloaded here](https://dl.loraserver.io/tektelic/konapico/).

1. After turning on the gateway, find the IP that has been assigned to it
   (eg. by listing the devices connected to your router). In the examples below
   it is assumed that the IP of the gateway is `192.168.1.10`.

2. Upload the channel configuration to the gateway. For this you need to use
   TFTP. The example command below shows how to do this from the command-line.
   The KONA Pico IoT Gateway configuration user-guide contains instructions how
   to do this using using a UI (Windows).
   ```bash
   $ tftp 192.168.1.10
   tftp> put lorawan_conf.json
   Sent 1412 bytes in 0.0 seconds
   ```
   Note: the tftp command is invoked from the same directory as where the
   `lorawan_conf.json` is stored.

3. Upload the configuration file containing the IP of the LoRa Gateway Bridge
   instance and the used ports.
   ```bash
   $ tftp 192.168.1.10
   tftp> put customer_conf.json
   Sent 121 bytes in 0.1 seconds
   ```
   Note: the tftp command is invoked from the same directory as where the
   `customer_conf.json` is stored.

### Example configuration files

#### lorawan_conf.json for EU868

This configuration file contains exactly the same radio configuration as
[global_conf.json.PCB_E336.EU868.basic](https://github.com/Lora-net/packet_forwarder/blob/master/lora_pkt_fwd/cfg/global_conf.json.PCB_E336.EU868.basic)

```json
{
    "public": true,
    "radio": [
        {
            "enable": true,
            "freq": 867500000
        },
        {
            "enable": true,
            "freq": 868500000
        }
    ],
    "lora_multi": [
        {
            "enable": true,
            "radio": 1,
            "offset": -400000
        },
        {
            "enable": true,
            "radio": 1,
            "offset": -200000
        },
        {
            "enable": true,
            "radio": 1,
            "offset": 0
        },
        {
            "enable": true,
            "radio": 0,
            "offset": -400000
        },
        {
            "enable": true,
            "radio": 0,
            "offset": -200000
        },
        {
            "enable": true,
            "radio": 0,
            "offset": 0
        },
        {
            "enable": true,
            "radio": 0,
            "offset": 200000
        },
        {
            "enable": true,
            "radio": 0,
            "offset": 400000
        }
    ],
    "lora_std": {
        "enable": true,
        "radio": 1,
        "offset": -200000,
        "bandwidth": "250kHz",
        "spread_factor": "SF7"
    },
    "fsk": {
        "enable": true,
        "radio": 1,
        "offset": 300000,
        "bandwidth": "125kHz",
        "datarate": 50000
    }
}
```

#### customer_conf.json

The IP or hostname of the `network_server` key must match the IP or hostname
of the machine on which LoRa Gateway Bridge is running.

```json
{
    "network_server": "192.168.1.5",
    "network_service_up_port": 1700,
    "network_service_down_port": 1700
}
```
