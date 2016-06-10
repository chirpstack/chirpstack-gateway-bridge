# Getting started

## Requirements

Before you install the LoRa Semtech Bridge, make sure you've installed the following requirements:

#### MQTT server

LoRa Semtech Brige makes use of MQTT for communication with the gateways and applications.
[Mosquitto](http://mosquitto.org/) is a popular open-source MQTT server.
Make sure you install a **recent** version of Mosquitto (the Mosquitto project provides
repositories for various Linux distributions).

## Install LoRa Semtech Brige

#### Download

Download and unpack a pre-compiled binary from the
[releases](https://github.com/brocaar/lora-semtech-bridge/releases) page. Alternatively,
build the code from source.

#### Configuration

All configuration is done by either environment variables or command-line
arguments. Arguments and environment variables can be mixed. 

Run ``./semtech-bridge --help`` for a list of available arguments.

#### Starting LoRa Semtech Bridge

Assuming you have a MQTT broker running on the same host without authentication,
starting LoRa Semtech Bridge is as simple as:

```bash
./semtech-bridge
```

#### LoRa gateway configuration

Now you have the LoRa Semtech Bridge running, it is time to configure the gateway.
Assuming you have the [``packet_forwarder``](https://github.com/Lora-net/packet_forwarder/)
already setup, edit the file ``local_config.json``:

```json
{
/* Put there parameters that are different for each gateway (eg. pointing one gateway to a test server while the others stay in production) */
/* Settings defined in global_conf will be overwritten by those in local_conf */
    "gateway_conf": {
        "gateway_ID": "...",      /* you must pick a unique 64b number for each gateway (represented by an hex string) */
        "server_address": "...",  /* the IP address on which the LoRa Semtech Bridge is running */
        "serv_port_up": 1700,     /* 1700 is the default LoRa Semtech Bridge port for up and down */
        "serv_port_down": 1700
    }
}
```

#### Verify data is coming in

After changing the LoRa gateway configuration (and restarting the `packet_forwarder`!),
you should see received packets in the logs. Example:

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

When using a MQTT client, you should be able to see all data sent by the
gateways by subscribing to the topic `gateway/#`.

## Setup LoRa Server

Now you have your LoRa Semtech bridge instance up and running, it is time to
setup [LoRa Server](https://github.com/brocaar/loraserver)!
