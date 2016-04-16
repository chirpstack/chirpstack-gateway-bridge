# LoRa Semtech Bridge

[![Build Status](https://travis-ci.org/brocaar/lora-semtech-bridge.svg?branch=master)](https://travis-ci.org/brocaar/lora-semtech-bridge)
[![GoDoc](https://godoc.org/github.com/brocaar/lora-semtech-bridge/cmd/semtech-bridge?status.svg)](https://godoc.org/github.com/brocaar/lora-semtech-bridge/cmd/semtech-bridge)

*LoRa Semtech Bridge* is a service to enable LoRa gateway communication over MQTT.
All incoming UDP traffic (Semtech UDP protocol) is published to a MQTT broker and
all messages received from the MQTT broker are sent to the gateway using UDP.

This project is part of [LoRa Server](https://github.com/brocaar/loraserver).

## Requirements

#### MQTT broker

[Mosquitto](http://mosquitto.org/) is a popular open-source MQTT server.

#### LoRa gateway

Your gateway must be setup so that it sends UDP packets (Semtech UDP protocol).
Depending upon your LoRa Gateway type, you might need to install the
[packet_forwarder](https://github.com/TheThingsNetwork/packet_forwarder).


## Getting started

* Download and unpack a pre-compiled binary from the [releases](https://github.com/brocaar/lora-semtech-bridge/releases)
  page. Alternatively, build the code from source (when you have a Go development environment
  ``make build`` should be sufficient).

* Start the ``semtech-bridge`` service. The ``--help`` argument will show
   you all the available config options. When everything is setup correctly
   (your gateway is configured to send data to ``semtech-bridge``), you
   should see ``PullData`` packets coming in. E.g.

``` bash
$ ./bin/semtech-bridge
INFO[0000] backend/mqttpubsub: connecting to mqtt server  server=tcp://127.0.0.1:1883
INFO[0000] starting gateway udp listener                 addr=0.0.0.0:1700
INFO[0006] incoming gateway packet                       addr=192.168.1.4:54993 type=PullData
INFO[0006] backend/mqttpubsub: subscribing to topic      topic=gateway/1dee08d0b691d149/tx
INFO[0006] outgoing gateway packet                       addr=192.168.1.4:54993 type=PullACK
INFO[0006] incoming gateway packet                       addr=192.168.1.4:51926 type=PushData
INFO[0006] stat packet received                          addr=192.168.1.4:51926 mac=1dee08d0b691d149
INFO[0006] outgoing gateway packet                       addr=192.168.1.4:51926 type=PushACK
INFO[0006] backend/mqttpubsub: publishing message        topic=gateway/1dee08d0b691d149/stats
```

* Now it is time to setup the [LoRa Server](https://github.com/brocaar/loraserver)!

## Debugging

To view all incoming and outgoing MQTT messages, you can use the ``mosquitto_sub``
tool.

```bash
$ mosquitto_sub -t "#" -v
gateway/1dee08d0b691d149/stats {"mac":"1dee08d0b691d149","time":"2016-04-16T10:08:11Z","latitude":0,"longitude":0,"altitude":0,"rxPacketsReceived":0,"rxPacketsReceivedOK":0}
gateway/1dee08d0b691d149/rx {"rxInfo":{"mac":"1dee08d0b691d149","time":"2016-04-16T10:08:30.005418Z","timestamp":1623489499,"frequency":868300000,"channel":1,"rfChain":1,"crcStatus":1,"codeRate":"4/5","rssi":-48,"loRaSNR":10,"size":23,"dataRate":{"modulation":"LORA","spreadFactor":7,"bandwidth":125}},"phyPayload":"AAEBAQEBAQEBAgICAgICAgJzTIBGXXg="}
gateway/1dee08d0b691d149/stats {"mac":"1dee08d0b691d149","time":"2016-04-16T10:08:41Z","latitude":0,"longitude":0,"altitude":0,"rxPacketsReceived":1,"rxPacketsReceivedOK":1}
```

## Data model

The data model is defined in the [LoRa Server](https://github.com/brocaar/loraserver)
project. JSON is used as marshaling format. See:

* [RXPacket](https://godoc.org/github.com/brocaar/loraserver/models/#RXPacket) (packet received by the gateway)
* [TXPacket](https://godoc.org/github.com/brocaar/loraserver/models/#TXPacket) (packet to be sent by the gateway)
* [GatewayStatsPacket](https://godoc.org/github.com/brocaar/loraserver/models/#GatewayStatsPacket) (gateway stats)


## License

This package is licensed under the MIT license. See ``LICENSE``.
