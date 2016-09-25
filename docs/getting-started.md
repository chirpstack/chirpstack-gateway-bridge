# Getting started

This getting started document describes the steps needed to setup LoRa Gateway
bridge and all its requirements on Ubuntu 16.04 LTS. When using an other Linux
distribution, you might need to adapt these steps slightly!

!!! warning
    This getting started guide does not cover setting up firewall rules! After
    setting up LoRa Server and its requirements, don't forget to configure
    your firewall rules.

## LoRa Gateway with packet_forwarder

The [packet_forwarder](https://github.com/Lora-net/packet_forwarder/) is an
application which runs on your gateway. It's responsibility is to:

* forward received uplink packets (over UDP)
* forward statistics (over UDP)
* enqueue and transmit downlink packets (received over UDP)

See the **Gateways** section in the menu for instructions about how to setup the
packet_forwarder on your gateway. Is your gateway not in the list? Please 
consider contributing to this documentation by documenting the steps needed
to set your gateway up and create a pull-request!

## MQTT broker

LoRa Gateway Brige makes use of MQTT for communication with the gateways and
applications. [Mosquitto](http://mosquitto.org/) is a popular open-source MQTT
server. Make sure you install a **recent** version of Mosquitto (the Mosquitto
project provides [repositories for various Linux distributions](http://mosquitto.org/download/)).
Ubuntu 16.04 LTS already includes a recent version which can be installed with:

```bash
sudo apt-get install mosquitto
```


## Install LoRa Gateway Brige

Create a system user for `gatewaybridge`:

```bash
sudo useradd -M -r -s /bin/false gatewaybridge
```

Download and unpack a pre-compiled binary from the [releases](https://github.com/brocaar/lora-gateway-bridge/releases)
page:

```bash
# replace VERSION with the latest version or the version you want to install

# download
wget https://github.com/brocaar/lora-gateway-bridge/releases/download/VERSION/lora_gateway_bridge_VERSION_linux_amd64.tar.gz

# unpack
tar zxf lora_gateway_bridge_VERSION_linux_amd64.tar.gz

# move the binary to /opt/lora-gateway-bridge/bin
sudo mkdir -p /opt/lora-gateway-bridge/bin
sudo mv lora-gateway-bridge /opt/lora-gateway-bridge/bin
```

In order to start LoRa Gateway Bridge as a service, create the file
`/etc/systemd/system/lora-gateway-bridge.service` with as content:

```
[Unit]
Description=lora-gateway-bridge
After=mosquitto.service

[Service]
User=gatewaybridge
Group=gatewaybridge
ExecStart=/opt/lora-gateway-bridge/bin/lora-gateway-bridge
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

The default LoRa Gateway Bridge settings will work, but in case you would like
to use different settings, create a directory named
`/etc/systemd/system/lora-gateway-bridge.service.d`:

```bash
sudo mkdir /etc/systemd/system/lora-gateway-bridge.service.d
```

Inside this directory, put a file named `lora-gateway-bridge.conf`:

```
[Service]
Environment="SETTING_A=value_a"
Environment="SETTING_B=value_b"
```

(note that the above content is an example and the values must be replaced by
actual config keys)

## Starting LoRa Gateway Bridge

In order to (re)start and stop LoRa Gateway Bridge:

```bash
# start
sudo systemctl start lora-gateway-bridge

# restart
sudo systemctl restart lora-gateway-bridge

# stop
sudo systemctl stop lora-gateway-bridge
```

Verify that LoRa Gateway Bridge is up-and-running by looking at it's log output:

```bash
journalctl -u lora-gateway-bridge -f -n 50
```

### Configuration

For all the configuration options, run `lora-gateway-bridge --help` for an
overview of all available options. Note that configuration variables can be
passed as cli arguments and / or environment variables
(which we did in the above example).

### Verify data is coming in

After setting up LoRa Gateway Bridge and configuring your gateway so that it
forwards data to your Lora Gateway Bridge instance, you should see data coming
in:

```bash
journalctl -u lora-gateway-bridge -f -n 50
```

Example:

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