---
title: Debian / Ubuntu
menu:
    main:
        parent: install
        weight: 4
---

# Debian / Ubuntu

These steps have been tested using:

* Debian Jessie
* Ubuntu Trusty (14.04)
* Ubuntu Xenial (16.04)

## LoRa Server Debian repository

The LoRa Server project provides pre-compiled binaries packaged as Debian (.deb)
packages. In order to activate this repository, execute the following
commands:

```bash
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 1CE2AFD36DBCCA00

export DISTRIB_ID=`lsb_release -si`
export DISTRIB_CODENAME=`lsb_release -sc`
sudo echo "deb https://repos.loraserver.io/${DISTRIB_ID,,} ${DISTRIB_CODENAME} testing" | sudo tee /etc/apt/sources.list.d/loraserver.list
sudo apt-get update
```

## Install LoRa Gateway Bridge

In order to install LoRa Gateway Bridge, execute the following command:

```bash
sudo apt-get install lora-gateway-bridge
```

This will setup an user and group, create start scripts for systemd or init.d
(this depends on your version of Debian / Ubuntu). The configuration file is
located at `/etc/lora-gateway-bridge/lora-gateway-bridge.toml`.

## Starting LoRa Gateway Bridge

How you need to (re)start and stop LoRa Gateway Bridge depends on if your
platform uses systemd or init.d.

### systemd

```bash
sudo systemctl [start|stop|restart|status] lora-gateway-bridge
```

### init.d

```bash
sudo /etc/init.d/lora-gateway-bridge [start|stop|restart|status]
```

## LoRa Gateway log output

Now you've setup LoRa Gateway Bridge and your gateway is configured to forward
it's data to it, it is a good time to verify that data is actually comming in.
This can be done by looking at the LoRa Gateway Bridge log output.

Like the previous step, which command you need to use for viewing the
log output depends on if your distribution uses init.d or systemd.

### init.d

All logs are written to `/var/log/lora-gateway-bridge/lora-gateway-bridge.log`.
To view and follow this logfile:

```bash
tail -f /var/log/lora-gateway-bridge/lora-gateway-bridge.log
```

### systemd

```bash
journalctl -u lora-gateway-bridge -f -n 50
```

Example output:

```
lora-gateway-bridge[9714]: time="2016-08-19T09:05:18+02:00" level=info msg="starting LoRa Gateway Bridge" docs="https://docs.loraserver.io/lora-gateway-bridge/" version=2.1.0
lora-gateway-bridge[9714]: time="2016-08-19T09:05:18+02:00" level=info msg="backend: connecting to mqtt broker" server="tcp://localhost:1883/"
lora-gateway-bridge[9714]: time="2016-08-19T09:05:18+02:00" level=info msg="gateway: starting gateway udp listener" addr=0.0.0.0:1700
lora-gateway-bridge[9714]: time="2016-08-19T09:05:18+02:00" level=info msg="backend: connected to mqtt broker"
lora-gateway-bridge[9714]: time="2016-08-19T09:05:23+02:00" level=info msg="gateway: received udp packet from gateway" addr=86.83.25.107:35368 protocol_version=2 type=PushData
lora-gateway-bridge[9714]: time="2016-08-19T09:05:23+02:00" level=info msg="gateway: stat packet received" addr=86.83.25.107:35368 mac=1dee08d0b691d149
lora-gateway-bridge[9714]: time="2016-08-19T09:05:23+02:00" level=info msg="backend: publishing packet" topic="gateway/1dee08d0b691d149/stats"
lora-gateway-bridge[9714]: time="2016-08-19T09:05:23+02:00" level=info msg="gateway: sending udp packet to gateway" addr=86.83.25.107:35368 protocol_version=2 type=PushACK
lora-gateway-bridge[9714]: time="2016-08-19T09:05:24+02:00" level=info msg="gateway: received udp packet from gateway" addr=86.83.25.107:45562 protocol_version=2 type=PullData
lora-gateway-bridge[9714]: time="2016-08-19T09:05:24+02:00" level=info msg="backend: subscribing to topic" topic="gateway/1dee08d0b691d149/tx"
lora-gateway-bridge[9714]: time="2016-08-19T09:05:24+02:00" level=info msg="gateway: sending udp packet to gateway" addr=86.83.25.107:45562 protocol_version=2 type=PullACK
lora-gateway-bridge[9714]: time="2016-08-19T09:05:34+02:00" level=info msg="gateway: received udp packet from gateway" addr=86.83.25.107:45562 protocol_version=2 type=PullData
lora-gateway-bridge[9714]: time="2016-08-19T09:05:34+02:00" level=info msg="gateway: sending udp packet to gateway" addr=86.83.25.107:45562 protocol_version=2 type=PullACK
```

For an explanation of the different types of data you can receive from and
send to the LoRa Gateway Bridge see [send / receive data]({{< ref "use/data.md" >}}).

## Install other components

A complete LoRa Server setup, requires the setup of the following components:


* LoRa Gateway Bridge
* [LoRa Server](/loraserver/)
* [LoRa App Server](/lora-app-server/)
