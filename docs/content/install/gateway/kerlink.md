---
title: Kerlink
menu:
    main:
        parent: gateway
---

# Kerlink

## Kerlink IOT station

* [Product detail page](https://www.kerlink.com/product/wirnet-station/)

The Kerlink IOT station has a meganism to start "custom" application on boot.
These steps will install the LoRa Gateway Bridge ARM build on the Kerlink.

1. Create the the directories needed:

    `mkdir -p /mnt/fsuser-1/lora-gateway-bridge/bin`

2. Download and extract the LoRa Gateway Bridge ARM binary into the above
   directory. See [downloads]({{< ref "overview/downloads.md" >}}).
   Make sure the binary is marked as executable.

3. Save the following content as `/mnt/fsuser-1/lora-gateway-bridge/start.sh`:

    ```bash
    #!/bin/bash

    LOGGER="logger -p local1.notice"

    # mosquitto
    iptables -A INPUT -p tcp --sport 1883 -j ACCEPT

    /mnt/fsuser-1/lora-gateway-bridge/bin/lora-gateway-bridge --mqtt-server tcp://YOURSERVER:1883  2>&1 | $LOGGER &
    ```

    Make sure to replace `YOURSERVER` with the hostname / IP of your MQTT
    broker. Also make sure the file is marked as executable.

4. Save the following content as `/mnt/fsuser-1/lora-gateway-bridge/manifest.xml`:

    ```xml
    <?xml version="1.0"?>
    <manifest>
    <app name="lora-gateway-bridge" appid="1" binary="start.sh" >
    <start param="" autostart="y"/>
    <stop kill="9"/>
    </app>
    </manifest>
    ```
