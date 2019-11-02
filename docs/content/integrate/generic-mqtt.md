---
title: Generic MQTT
menu:
    main:
        parent: integrate
        weight: 2
description: Setting up the ChirpStack Gateway Bridge using a generic MQTT broker.
---

# Generic MQTT authentication

The Generic MQTT authentication type provides a generic MQTT client where most
of the connection parameters can be configured using the
[Configuration file]({{<ref "/install/config.md">}}). This is the
recommended authentication type for most MQTT brokers.

## Consuming data

To receive events from your gateways, you need to subscribe to its MQTT topic(s).
For debugging, you can use a (command-line) tool like `mosquitto_sub`
which is part of the [Mosquitto](http://mosquitto.org/) MQTT broker.

Use ``+`` for a single-level wildcard, ``#`` for a multi-level wildcard.
Examples:

{{<highlight bash>}}
# show data from all gateways 
mosquitto_sub -t "gateway/#" -v

# show all events and commands for the given gateway ID
mosquitto_sub -t "gateway/0101010101010101/#" -v

# show all events for the given gateway ID
mosquitto_sub -t "gateway/0101010101010101/event/+" -v

# show all commands for the given gateway ID
mosquitto_sub -t "gateway/0101010101010101/command/+" -v
{{< /highlight >}}
