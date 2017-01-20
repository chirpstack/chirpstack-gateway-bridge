# Getting started

This getting started document describes the steps needed to setup LoRa Gateway
Bridge using the provided Debian package repository. Please note that LoRa
Gateway Bridge is not limited to Debian / Ubuntu only! General purpose binaries
can be downloaded from the 
[releases](https://github.com/brocaar/lora-gateway-bridge/releases) page.

!!! info
	An alternative way to setup all the components is by using the
	[loraserver-setup](https://github.com/brocaar/loraserver-setup) Ansible
	scripts. It automates the steps below and can also be used in combination
	with [Vagrant](https://www.vagrantup.com/).

!!! warning
    This getting started guide does not cover setting up firewall rules! After
    setting up LoRa Server and its requirements, don't forget to configure
    your firewall rules.

## Setting up LoRa Gateway Bridge

These steps have been tested with:

* Debian Jessie
* Ubuntu Trusty (14.04)
* Ubuntu Xenial (16.06)

### LoRa Gateway

First you need to setup the
[packet_forwarder](https://github.com/Lora-net/packet_forwarder/)
on your LoRa Gateway. The packet_forwarder is an application which
communicates with your gateway hardware. It's responsibility is to:

* forward received uplink packets (over UDP)
* forward statistics (over UDP)
* enqueue and transmit downlink packets (received over UDP)

See the **Gateways** section in the menu for instructions about how to setup the
packet_forwarder on your gateway. Is your gateway not in the list? Please 
consider contributing to this documentation by documenting the steps needed
to set your gateway up and create a pull-request!

### LoRa Server Debian repository

The LoRa Server project provides pre-compiled binaries packaged as Debian (.deb)
packages. In order to activate this repository, execute the following
commands:

```bash
source /etc/lsb-release
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 1CE2AFD36DBCCA00
sudo echo "deb https://repos.loraserver.io/${DISTRIB_ID,,} ${DISTRIB_CODENAME} testing" | sudo tee /etc/apt/sources.list.d/loraserver.list
sudo apt-get update
```

### MQTT broker

LoRa Gateway Brige makes use of MQTT for communication with the gateways and
applications. [Mosquitto](http://mosquitto.org/) is a popular open-source MQTT
server. Make sure you install a **recent** version of Mosquitto.

For Ubuntu Trusty (14.04), execute the following command in order to add the
Mosquitto Apt repository:

```bash
sudo apt-add-repository ppa:mosquitto-dev/mosquitto-ppa
sudo apt-get update
```

In order to install Mosquitto, execute the following command:

```bash
sudo apt-get install mosquitto
```

### Install LoRa Gateway Brige

In order to install LoRa Gateway Bridge, execute the following command:

```bash
sudo apt-get install lora-gateway-bridge
```

This will setup an user and group, create start scripts for systemd or init.d
(this depends on your version of Debian / Ubuntu).

The configuration file is located at `/etc/default/lora-gateway-bridge`.
For this getting started, you can keep all the default values.

### Starting LoRa Gateway Bridge

How you need to (re)start and stop LoRa Gateway Bridge depends on if your
distribution uses init.d or systemd.

#### init.d

```bash
sudo /etc/init.d/lora-gateway-bridge [start|stop|restart|status]
```

#### systemd

```bash
sudo systemctl [start|stop|restart|status] lora-gateway-bridge
```

### LoRa Gateway log output

Now you've setup LoRa Gateway Bridge and your gateway is configured to forward
it's data to it, it is a good time to verify that data is actually comming in.
This can be done by looking at the LoRa Gateway Bridge log output.

Like the previous step, which command you need to use for viewing the
log output depends on if your distribution uses init.d or systemd.

#### init.d

All logs are written to `/var/log/lora-gateway-bridge/lora-gateway-bridge.log`.
To view and follow this logfile:

```bash
tail -f /var/log/lora-gateway-bridge/lora-gateway-bridge.log
```

#### systemd

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
send to the LoRa Gateway Bridge see [topics](topics.md).

## Setup LoRa Server

Now you have your LoRa Gateway bridge instance up and running, it is time to
setup [LoRa Server](https://docs.loraserver.io/loraserver/).
