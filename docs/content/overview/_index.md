---
title: LoRa Gateway Bridge
menu:
    main:
        parent: overview
        weight: 1
---

# LoRa Gateway Bridge

LoRa Gateway Bridge is a service which abstracts the 
[packet-forwarder UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT)
into JSON over MQTT. The package-forwarder software is running on most
gateways and its purpose is to communicate with the LoRa chipset.

## Why?

### Visibility

Because LoRa Gateway Bridge publishes the content of the UDP packets as JSON
over MQTT, it becomes trivial to monitor the data that is sent and received
by each gateway.

### Routing

As each LoRa Gateway Bridge subscribes to the topics related to the gateways
that are connected to the instance, the MQTT broker will handle the routing
of which (downlink) frame must be sent to which LoRa Gateway Bridge
instance.

### Security

By running the LoRa Gateway Bridge on the gateway itself, it is possible to use
MQTT over TLS, meaning the transport between the gateway and server(s) is
secure.
