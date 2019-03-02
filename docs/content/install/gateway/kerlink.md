---
title: Kerlink
menu:
    main:
        parent: gateway
description: Installation of the LoRa Gateway Bridge on Kerlink gateways.
---

# Kerlink

## Kerlink IOT station

* [Product detail page](https://www.kerlink.com/product/wirnet-station/)

The Kerlink IOT station has a meganism to start "custom" application on boot.
These steps will install the LoRa Gateway Bridge ARM build on the Kerlink.

1. Create the the directories needed:
{{<highlight bash>}}
mkdir -p /mnt/fsuser-1/lora-gateway-bridge/bin
{{< /highlight >}}

2. Download and extract the LoRa Gateway Bridge ARMv5 binary into the above
   directory. See [downloads]({{< ref "/overview/downloads.md" >}}).
   Make sure the binary is marked as executable.

3. Save the following content as `/mnt/fsuser-1/lora-gateway-bridge/start.sh`:
{{<highlight bash>}}
#!/bin/bash

LOGGER="logger -p local1.notice"

# mosquitto
iptables -A INPUT -p tcp --sport 1883 -j ACCEPT

/mnt/fsuser-1/lora-gateway-bridge/bin/lora-gateway-bridge --mqtt-server tcp://YOURSERVER:1883  2>&1 | $LOGGER &
{{< /highlight >}}

    Make sure to replace `YOURSERVER` with the hostname / IP of your MQTT
    broker. Also make sure the file is marked as executable.

4. Save the following content as `/mnt/fsuser-1/lora-gateway-bridge/manifest.xml`:
{{<highlight xml>}}
<?xml version="1.0"?>
<manifest>
	<app name="lora-gateway-bridge" appid="1" binary="start.sh" >
		<start param="" autostart="y"/>
		<stop kill="9"/>
	</app>
</manifest>
{{< /highlight >}}

## Kerlink iBTS

* [Product detail page: iBTS](https://www.kerlink.com/product/wirnet-ibts/)

### SSH into the gateway

The first step is to login into the gateway using ssh:

{{<highlight bash>}}
sh root@GATEWAY-IP-ADDRESS
{{</highlight>}}

Please refer to the [Kerlink iBTS wiki](http://wikikerlink.fr/wirnet-ibts/)
for login instructions.

### Install IPK package

Find the latest package at https://artifacts.loraserver.io/vendor/kerlink/ibts/
and copy the URL to your clipboard. Then on the gateway use `wget` to download
the package into a folder named `/user/.updates`. Example for `lora-gateway-bridge_2.7.0-r1_klk_lpbs.ipk`:

{{<highlight bash>}}
mkdir -p /user/.updates
cd /user/.updates
wget https://artifacts.loraserver.io/vendor/kerlink/ibts/lora-gateway-bridge_2.7.0-r1_klk_lpbs.ipk
{{</highlight>}}

To trigger the iBTS gateway to install / update the package, run the following commands:

{{<highlight bash>}}
kerosd -u
reboot
{{</highlight>}}

Please refer to the [Kerlink iBTS wiki](http://wikikerlink.fr/wirnet-ibts/)
for more information about installing and updating packages.

### Edit the LoRa Gateway Bridge configuration

To connect the LoRa Gateway Bridge with your MQTT broker, you must update
the LoRa Gateway Bridge configuration file, which is located at:
`/user/lora-gateway-bridge/lora-gateway-bridge.toml`.

### (Re)start and stop commands

Use the following commands to (re)start and stop the LoRa Gateway Bridge Service:

{{<highlight bash>}}
# status
monit status lora-gateway-bridge

# start
monit start lora-gateway-bridge

# stop
monit stop lora-gateway-bridge

# restart
monit restart lora-gateway-bridge
{{</highlight>}}

### Configure packet-forwarder

You must configure the packet-forwarder on the gateway to forward its data to
`127.0.0.1` at port `1700`. The file `/user/spf2/etc/config.json` must contain the
following lines:

{{<highlight text>}}
"server_address": "127.0.0.1",
"serv_port_up": 1700,
"serv_port_down": 1700,
{{</highlight>}}

## Kerlink iFemtoCell

* [Product detail page](https://www.kerlink.com/product/wirnet-ifemtocell/)

The installation steps to install the LoRa Gateway Bridge component are exactly
the same as for the iBTS gateway. The only difference is that the packages
are located at https://artifacts.loraserver.io/vendor/kerlink/ifemtocell/.

