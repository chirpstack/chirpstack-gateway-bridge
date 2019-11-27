---
title: Dragino
description: Installation of the ChirpStack Gateway Bridge on the Dragino gateway.
menu:
  main:
    parent: gateway
---

# Dragino

## LG308 Indoor LoRaWAN Gateway

* [Product detail page](https://www.dragino.com/products/lora-lorawan-gateway/item/140-lg308.html)
* [Product documentation](https://www.dragino.com/downloads/index.php?dir=LoRa_Gateway/LG308-LG301/)

### Configure Packet Forwarder

In the Dragino LG-308 web-interface, you need to configure the Packet Forwarder
so that it forwards its data to `localhost` on port `1700`.

* In the **Service** menu, click on **LoRaWAN Gateway**
* Make sure the following settings are set:
  * **IoT Service:** _LoRaWAN / RAW Forwarder
  * **Service Provider:** _--custom--_
  * **LoRaWAN Server Address:** _localhost_
  * **Server port for upstream:** _1700_
  * **Server port for downstream:** _1700_

Click **Save & Apply**.

### Install ChirpStack Gateway Bridge

#### SSH into the gateway

The first step is to login into the gateway using ssh:

{{<highlight bash>}}
ssh root@GATEWAY-IP-ADDRESS
{{</highlight>}}

The default password is _dragino_.

#### Download IPK

Find the latest package at https://artifacts.chirpstack.io/vendor/dragino/LG308/
and copy the URL to your clipboard. Then on the gateway use `wget` to download
the IPK package. It is important you download the package to `/tmp`. Example for
`chirpstack-gateway-bridge_3.5.0-r1_mips_24kc.ipk`:

{{<highlight bash>}}
cd /tmp
wget https://artifacts.chirpstack.io/vendor/dragino/LG308/chirpstack-gateway-bridge_3.5.0-r1_mips_24kc.ipk
{{</highlight>}}

#### Install IPK

Use the `opkg` package-manager to install the downloaded package. Example:

{{<highlight bash>}}
opkg install chirpstack-gateway-bridge_3.5.0-r1_mips_24kc.ipk
{{</highlight>}}

**Note:** In case of an upgrade, it is recommeded to first uninstall the
`chirpstack-gateway-bridge` package using `opkg remove ...` and then install the
new version using `opkg install ...`. Configuration files will be maintained.

#### Configuration

To connect the ChirpStack Gateway Bridge with your MQTT broker, you must update
the ChirpStack Gateway Bridge configuration file, which is located at:
`/etc/chirpstack-gateway-bridge/chirpstack-gateway-bridge.toml`.

#### (Re)start and stop commands

Use the following commands to (re)start and stop the ChirpStack Gateway Bridge Service:

{{<highlight bash>}}
# start
/etc/init.d/chirpstack-gateway-bridge start

# stop
/etc/init.d/chirpstack-gateway-bridge stop

# restart
/etc/init.d/chirpstack-gateway-bridge restart
{{</highlight>}}
