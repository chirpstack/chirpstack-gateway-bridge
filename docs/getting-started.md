## Strategies

There are multiple ways that you can deploy the LoRa Semtech Bridge:

### Single instance

The most basic strategy is to connect all your gateways to a single instance
of the LoRa Semtech Bridge.

### Multiple instances

To make the LoRa Semtech Bridge HA, you can run a cluster of instances
(connecting to the same MQTT broker).
**Important:** make sure that each gateway connection is always routed to the
same instance!

### On each gateway

Depending on the capabilities of your gateway, you can deploy the LoRa Semtech
Bridge on each of your gateways. This enables you to encrypt all traffic from
your gateway by connecting to the MQTT broker over SSL/TLS.




## Requirements

Before you install the LoRa Semtech Bridge, make sure you've installed the
following requirements:

### MQTT broker

LoRa Semtech Brige makes use of MQTT for communication with the gateways and
applications. [Mosquitto](http://mosquitto.org/) is a popular open-source MQTT
server. Make sure you install a **recent** version of Mosquitto (the Mosquitto
project provides [repositories for various Linux distributions](http://mosquitto.org/download/)).

### LoRa Gateway with packet_forwarder

See [packet_forwarder](packet-forwarder.md) for instructions about how to
setup the packet_forwarder on your gateway.

## Install LoRa Semtech Brige

!!! warning "Compatibility"
    Please check the [Compatibility](index.md#compatibility) to make sure you
    download the correct LoRa Semtech Bridge version.

### Download

Download and unpack a pre-compiled binary from the
[releases](https://github.com/brocaar/lora-semtech-bridge/releases) page.
Alternatively, build the code from source (not covered).

### Configuration

All configuration is done by either environment variables or command-line
arguments. Arguments and environment variables can be mixed. 

To get a list (and explanation) of all the available arguments, execute:

``` bash
$ ./semtech-bridge --help
```

### Starting LoRa Semtech Bridge

Assuming you have a MQTT broker running on the same host without authentication,
starting LoRa Semtech Bridge is as simple as:

``` bash
$ ./semtech-bridge \
    --udp-bind 0.0.0.0:1700             \  # this is the port you must use in the packet_forwarder
    --mqtt-server tcp://127.0.0.1:1883
```

### LoRa gateway configuration

Now you have the LoRa Semtech Bridge running, it is time to configure the
packet_forwarder on your gateway. Edit the file ``local_config.json``:

``` json
{
    "gateway_conf": {
        "gateway_ID": "...",      /* you must pick a unique 64b number for each gateway (represented by an hex string) */
        "server_address": "...",  /* the IP address on which the LoRa Semtech Bridge is running */
        "serv_port_up": 1700,     /* 1700 is the default LoRa Semtech Bridge port for up and down */
        "serv_port_down": 1700
    }
}
```

### Verify data is coming in

After changing the LoRa gateway configuration (and restarting the
packet_forwarder!), you should see received packets in the logs. Example:

```
INFO[0000] backend/mqttpubsub: connecting to mqtt broker  server=tcp://127.0.0.1:1883
INFO[0000] gateway: starting gateway udp listener        addr=0.0.0.0:1700
INFO[0000] backend/mqttpubsub: connected to mqtt broker
INFO[0006] gateway: received udp packet from gateway     addr=192.168.1.8:45082 type=PullData
INFO[0006] backend/mqttpubsub: subscribing to topic      topic=gateway/1dee08d0b691d149/tx
INFO[0006] gateway: sending udp packet to gateway        addr=192.168.1.8:45082 type=PullACK
INFO[0016] gateway: received udp packet from gateway     addr=192.168.1.8:45082 type=PullData
INFO[0016] gateway: sending udp packet to gateway        addr=192.168.1.8:45082 type=PullACK
INFO[0021] gateway: received udp packet from gateway     addr=192.168.1.8:45738 type=PushData
INFO[0021] gateway: stat packet received                 addr=192.168.1.8:45738 mac=1dee08d0b691d149
INFO[0021] gateway: sending udp packet to gateway        addr=192.168.1.8:45738 type=PushACK
INFO[0021] backend/mqttpubsub: publishing packet         topic=gateway/1dee08d0b691d149/stats
```

For an explanation of the different types of data you can receive from and
send to the LoRa Semtech Bridge see [topics](topics.md).

## Setup LoRa Server

Now you have your LoRa Semtech bridge instance up and running, it is time to
setup [LoRa Server](https://github.com/brocaar/loraserver).
