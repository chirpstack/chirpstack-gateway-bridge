# Getting started

## Strategies

There are multiple ways that you can deploy the LoRa Gateway Bridge:

### Single instance

The most basic strategy is to connect all your gateways to a single instance
of the LoRa Gateway Bridge.

### Multiple instances

To make the LoRa Gateway Bridge HA, you can run a cluster of instances
(connecting to the same MQTT broker).
**Important:** make sure that each gateway connection is always routed to the
same instance!

### On each gateway

Depending on the capabilities of your gateway, you can deploy the LoRa Gateway
Bridge on each of your gateways. This enables you to encrypt all traffic from
your gateway by connecting to the MQTT broker over SSL/TLS.


## Requirements

Before you install the LoRa Gateway Bridge, make sure you've installed the
following requirements:

### MQTT broker

LoRa Gateway Brige makes use of MQTT for communication with the gateways and
applications. [Mosquitto](http://mosquitto.org/) is a popular open-source MQTT
server. Make sure you install a **recent** version of Mosquitto (the Mosquitto
project provides [repositories for various Linux distributions](http://mosquitto.org/download/)).

### LoRa Gateway with packet_forwarder

The [packet_forwarder](https://github.com/Lora-net/packet_forwarder/) is an
application which runs on your gateway. It's responsibility is to:

* forward received uplink packets (over UDP)
* forward statistics (over UDP)
* enqueue and transmit downlink packets (received over UDP)

See the **Gateways** section in the menu for instructions about how to setup the
packet_forwarder on your gateway. Is your gateway not in the list? Please 
consider contributing to this documentation by documenting the steps needed
to set your gateway up and create a pull-request!

## Installing the LoRa Gateway Brige

### Download

Download and unpack a pre-compiled binary from the
[releases](https://github.com/brocaar/lora-gateway-bridge/releases) page.
Alternatively, build the code from source (not covered).

### Configuration

All configuration is done by either environment variables or command-line
arguments. Arguments and environment variables can be mixed. 

To get a list (and explanation) of all the available arguments, execute:

``` bash
$ ./lora-gateway-bridge --help
```

### Starting LoRa Gateway Bridge

Assuming you have a MQTT broker running on the same host without authentication,
starting LoRa Gateway Bridge is as simple as:

``` bash
$ ./lora-gateway-bridge \
    --udp-bind 0.0.0.0:1700             \  # this is the port you must use in the packet_forwarder
    --mqtt-server tcp://127.0.0.1:1883
```

### LoRa gateway configuration

Now you have the LoRa Gateway Bridge running, it is time to configure the
packet_forwarder on your gateway. Edit the file ``local_config.json``:

``` json
{
    "gateway_conf": {
        "gateway_ID": "...",      /* you must pick a unique 64b number for each gateway (represented by an hex string) */
        "server_address": "...",  /* the IP address on which the LoRa Gateway Bridge is running */
        "serv_port_up": 1700,     /* 1700 is the default LoRa Gateway Bridge port for up and down */
        "serv_port_down": 1700
    }
}
```

### Verify data is coming in

After changing the LoRa gateway configuration (and restarting the
packet_forwarder!), you should see received packets in the logs. Example:

```
INFO[0000] starting LoRa Gateway Bridge                  docs=https://docs.loraserver.io/lora-gateway-bridge/ version=2.1.0
INFO[0000] backend: connecting to mqtt broker            server=tcp://127.0.0.1:1883
INFO[0000] gateway: starting gateway udp listener        addr=0.0.0.0:1700
INFO[0000] backend: connected to mqtt broker
INFO[0001] gateway: received udp packet from gateway     addr=192.168.1.10:51013 protocol_version=2 type=PullData
INFO[0001] backend: subscribing to topic                 topic=gateway/1dee08d0b691d149/tx
INFO[0001] gateway: sending udp packet to gateway        addr=192.168.1.10:51013 protocol_version=2 type=PullACK
INFO[0007] gateway: received udp packet from gateway     addr=192.168.1.10:42125 protocol_version=2 type=PushData
INFO[0007] gateway: stat packet received                 addr=192.168.1.10:42125 mac=1dee08d0b691d149
INFO[0007] backend: publishing packet                    topic=gateway/1dee08d0b691d149/stats
INFO[0007] gateway: sending udp packet to gateway        addr=192.168.1.10:42125 protocol_version=2 type=PushACK
INFO[0011] gateway: received udp packet from gateway     addr=192.168.1.10:51013 protocol_version=2 type=PullData
INFO[0011] gateway: sending udp packet to gateway        addr=192.168.1.10:51013 protocol_version=2 type=PullACK
INFO[0021] gateway: received udp packet from gateway     addr=192.168.1.10:51013 protocol_version=2 type=PullData
INFO[0021] gateway: sending udp packet to gateway        addr=192.168.1.10:51013 protocol_version=2 type=PullACK
```

For an explanation of the different types of data you can receive from and
send to the LoRa Gateway Bridge see [topics](topics.md).

## Setup LoRa Server

Now you have your LoRa Gateway bridge instance up and running, it is time to
setup [LoRa Server](https://github.com/brocaar/loraserver).
