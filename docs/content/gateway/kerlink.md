---
title: Kerlink
description: Installation of the ChirpStack Gateway Bridge on Kerlink gateways.
menu:
  main:
    parent: gateway
---

# Kerlink

## Kerlink IOT station

* [Product detail page](https://www.kerlink.com/product/wirnet-station/)

The Kerlink IOT station has a mechanism to start "custom" application on boot.
These steps will install the LoRa Gateway Bridge ARM build on the Kerlink.

1. Create the the directories needed:
{{<highlight bash>}}
mkdir -p /mnt/fsuser-1/chirpstack-gateway-bridge/bin
{{< /highlight >}}

2. Download and extract the ChirpStack Gateway Bridge ARMv5 binary into the above
   directory. See [downloads]({{< ref "/overview/downloads.md" >}}).
   Make sure the binary is marked as executable.

3. Save the following content as `/mnt/fsuser-1/chirpstack-gateway-bridge/start.sh`:
{{<highlight bash>}}
#!/bin/bash

LOGGER="logger -p local1.notice"

# mosquitto
iptables -A INPUT -p tcp --sport 1883 -j ACCEPT

/mnt/fsuser-1/chirpstack-gateway-bridge/bin/chirpstack-gateway-bridge --mqtt-server tcp://YOURSERVER:1883  2>&1 | $LOGGER &
{{< /highlight >}}

    Make sure to replace `YOURSERVER` with the hostname / IP of your MQTT
    broker. Also make sure the file is marked as executable.

4. Save the following content as `/mnt/fsuser-1/chirpstack-gateway-bridge/manifest.xml`:
{{<highlight xml>}}
<?xml version="1.0"?>
<manifest>
	<app name="chirpstack-gateway-bridge" appid="1" binary="start.sh" >
		<start param="" autostart="y"/>
		<stop kill="9"/>
	</app>
</manifest>
{{< /highlight >}}

## Kerlink iBTS

* [Product detail page: iBTS](https://www.kerlink.com/product/wirnet-ibts/)

**Note:** These steps have been tested using the _KerOS firmware v4.1.6_.
Please make sure you have this version or later installed. You must also
install the Kerlink Common Packet Forwarder.

### SSH into the gateway

The first step is to login into the gateway using ssh:

{{<highlight bash>}}
ssh root@GATEWAY-IP-ADDRESS
{{</highlight>}}

Please refer to the [Kerlink wiki](http://wikikerlink.fr/wirnet-productline)
for login instructions.

### Install IPK package

Find the latest package at https://artifacts.chirpstack.io/vendor/kerlink/ibts/
and copy the URL to your clipboard. Then on the gateway use `wget` to download
the package into a folder named `/user/.updates`. Example for `chirpstack-gateway-bridge_3.4.0-r1_klk_lpbs.ipk`:

{{<highlight bash>}}
mkdir -p /user/.updates
cd /user/.updates
wget https://artifacts.chirpstack.io/vendor/kerlink/ibts/chirpstack-gateway-bridge_3.4.0-r1_klk_lpbs.ipk
{{</highlight>}}

To trigger the iBTS gateway to install / update the package, run the following commands:

{{<highlight bash>}}
sync
kerosd -u
reboot
{{</highlight>}}

Please refer to the [Kerlink wiki](http://wikikerlink.fr/wirnet-productline)
for more information about installing and updating packages.

### Edit the ChirpStack Gateway Bridge configuration

To connect the ChirpStack Gateway Bridge with your MQTT broker, you must update
the ChirpStack Gateway Bridge configuration file, which is located at:
`/user/etc/chirpstack-gateway-bridge/chirpstack-gateway-bridge.toml`.

### (Re)start and stop commands

Use the following commands to (re)start and stop the ChirpStack Gateway Bridge Service:

{{<highlight bash>}}
# status
monit status chirpstack-gateway-bridge

# start
monit start chirpstack-gateway-bridge

# stop
monit stop chirpstack-gateway-bridge

# restart
monit restart chirpstack-gateway-bridge
{{</highlight>}}

### Configure packet-forwarder

You must configure the packet-forwarder on the gateway to forward its data to
`127.0.0.1` at port `1700`. The file `/user/etc/lorafwd/lorafwd.toml` must contain the
following lines under the `[ gwmp ]` section:

{{<highlight toml>}}
node = "127.0.0.1"
service.uplink = 1700
service.downlink = 1700
{{</highlight>}}

After updating this configuration file, make sure to restart the `lorafwd` service:

{{<highlight bash>}}
monit restart lorafwd
{{</highlight>}}

## Kerlink iFemtoCell

* [Product detail page](https://www.kerlink.com/product/wirnet-ifemtocell/)

**Note:** These steps have been tested using the _KerOS firmware v4.1.6_.
Please make sure you have this version or later installed. You must also
install the Kerlink Common Packet Forwarder.

### SSH into the gateway

The first step is to login into the gateway using ssh:

{{<highlight bash>}}
ssh root@GATEWAY-IP-ADDRESS
{{</highlight>}}

Please refer to the [Kerlink wiki](http://wikikerlink.fr/wirnet-productline)
for login instructions.

### Install IPK package

Find the latest package at https://artifacts.chirpstack.io/vendor/kerlink/ifemtocell/
and copy the URL to your clipboard. Then on the gateway use `wget` to download
the package into a folder named `/user/.updates`. Example for `chirpstack-gateway-bridge_3.4.0-r1_klk_wifc.ipk`:

{{<highlight bash>}}
mkdir -p /user/.updates
cd /user/.updates
wget https://artifacts.chirpstack.io/vendor/kerlink/ifemtocell/chirpstack-gateway-bridge_3.4.0-r1_klk_wifc.ipk
{{</highlight>}}

To trigger the iFemtoCell gateway to install / update the package, run the following commands:

{{<highlight bash>}}
sync
kerosd -u
reboot
{{</highlight>}}

Please refer to the [Kerlink wiki](http://wikikerlink.fr/wirnet-productline)
for more information about installing and updating packages.

### Edit the ChirpStack Gateway Bridge configuration

To connect the ChirpStack Gateway Bridge with your MQTT broker, you must update
the ChirpStack Gateway Bridge configuration file, which is located at:
`/user/etc/chirpstack-gateway-bridge/chirpstack-gateway-bridge.toml`.

### (Re)start and stop commands

Use the following commands to (re)start and stop the ChirpStack Gateway Bridge Service:

{{<highlight bash>}}
# status
monit status chirpstack-gateway-bridge

# start
monit start chirpstack-gateway-bridge

# stop
monit stop chirpstack-gateway-bridge

# restart
monit restart chirpstack-gateway-bridge
{{</highlight>}}

### Configure packet-forwarder

You must configure the packet-forwarder on the gateway to forward its data to
`127.0.0.1` at port `1700`. The file `/user/etc/lorafwd/lorafwd.toml` must contain the
following lines under the `[ gwmp ]` section:

{{<highlight toml>}}
node = "127.0.0.1"
service.uplink = 1700
service.downlink = 1700
{{</highlight>}}

After updating this configuration file, make sure to restart the `lorafwd` service:

{{<highlight bash>}}
monit restart lorafwd
{{</highlight>}}
