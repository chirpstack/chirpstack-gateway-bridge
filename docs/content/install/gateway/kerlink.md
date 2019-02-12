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

* [Product detail page](https://www.kerlink.com/product/wirnet-ibts/)

The following steps describe how to install the LoRa Gateway Bridge component
on the Kerlink iBTS gateway. These steps must be executed on the gateway.

1. Create a path for the LoRa Gateway Bridge binary:
{{<highlight bash>}}
mkdir /user/lora-gateway-bridge
cd /user/lora-gateway-bridge
{{< /highlight >}}

2. Download the latest LoRa Gateway Bridge ARMv5 binary from the
   [Downloads](https://www.loraserver.io/lora-gateway-bridge/overview/downloads/)
   page and extract it:
{{<highlight bash>}}
# replace VERSION with the version to download
wget https://artifacts.loraserver.io/downloads/lora-gateway-bridge/lora-gateway-bridge_VERSION_linux_armv5.tar.gz

# extract the .tar.gz file
tar zxf lora-gateway-bridge_VERSION_linux_armv5.tar.gz
{{</highlight>}}

3. Create the LoRa Gateway Bridge configuration file and modify it:
{{<highlight bash>}}
# create the lora-gateway-bridge.toml configuration file
./lora-gateway-bridge configfile > lora-gateway-bridge.toml

# modify it using vim
vim lora-gateway-bridge.toml
{{</highlight>}}

4. In the `/user/lora-gateway-bridge` directory create a file named
`execute_lgb.sh` file with the following content:
{{<highlight bash>}}
#!/bin/sh -e

# Open firewall ports
iptables_accept() {
	[ -n "${1}" ] || exit 1
	local RULE="OUTPUT -t filter -p tcp --dport ${1} -j ACCEPT"
	iptables -C ${RULE} 2> /dev/null || iptables -I ${RULE}
	local RULE="INPUT -t filter -p tcp --sport ${1} -j ACCEPT"
	iptables -C ${RULE} 2> /dev/null || iptables -I ${RULE}
}

iptables_accept 1883
iptables_accept 8883

(
	# Lock on pid file
	exec 8> /var/run/lora-gateway-bridge.pid
	flock -n -x 8
	trap "rm -f /var/run/lora-gateway-bridge.pid" EXIT

	# Start LoRa Gateway Bridge
	/user/lora-gateway-bridge/lora-gateway-bridge -c /user/lora-gateway-bridge/lora-gateway-bridge.toml 2>&1 | logger -p local1.notice -t lora-gateway-bridge &
	trap "killall -q lora-gateway-bridge" INT QUIT TERM

	# Update pid and wait for it
	pidof lora-gateway-bridge >&8
	wait
) &
{{</highlight>}}

5. Make sure this file is executable:
{{<highlight bash>}}
chmod +x execute_lgb.sh
{{</highlight>}}

6. Create the LoRa Gateway Bridge [Monit](https://mmonit.com/monit/documentation/monit.html)
   configuration named `/etc/monit.d/lora-gateway-bridge`:
{{<highlight text>}}
check process lora-gateway-bridge pidfile /var/run/lora-gateway-bridge.pid
	start program = "/user/lora-gateway-bridge/execute_lgb.sh"
	stop program = "/usr/bin/killall -q lora-gateway-bridge"
{{</highlight>}}

7. Finally reload the Monit daemon and start the LoRa Gateway Bridge service:
{{<highlight bash>}}
monit reload
monit start lora-gateway-bridge
{{</highlight>}}
