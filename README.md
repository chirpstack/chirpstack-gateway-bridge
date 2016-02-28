# LoRa Semtech Bridge

[![Build Status](https://travis-ci.org/brocaar/lora-semtech-bridge.svg?branch=master)](https://travis-ci.org/brocaar/lora-semtech-bridge)
[![GoDoc](https://godoc.org/github.com/brocaar/lora-semtech-bridge/cmd/semtech-bridge?status.svg)](https://godoc.org/github.com/brocaar/lora-semtech-bridge/cmd/semtech-bridge)

*LoRa Semtech Bridge* is a service to handle LoRa gateway (Semtech UDP protocol)
connections and communication with the ``loraserver`` (https://github.com/brocaar/loraserver).

## Installation


## Getting started

1. First install ``semtech-bridge``:

``` bash
go get github.com/brocaar/lora-semtech-bridge/...
```

2. Make sure you have a MQTT server running. Mosquitto is a good option:
   http://mosquitto.org/.

3. Start the ``semtech-bridge`` service. The ``--help`` argument will show
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

4. Now setup the ``loraserver`` service! https://github.com/brocaar/loraserver

## License

This package is licensed under the MIT license which can be found in ``LICENSE``.
