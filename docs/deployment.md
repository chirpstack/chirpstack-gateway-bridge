# Deployment strategies

There are multiple ways that you can deploy the LoRa Gateway Bridge:

## Single instance

The most basic strategy is to connect all your gateways to a single instance
of the LoRa Gateway Bridge.

## Multiple instances

To make the LoRa Gateway Bridge HA, you can run a cluster of instances
(connecting to the same MQTT broker).
**Important:** make sure that each gateway connection is always routed to the
same instance!

## On each gateway

Depending on the capabilities of your gateway, you can deploy the LoRa Gateway
Bridge on each of your gateways. This enables you to encrypt all traffic from
your gateway by connecting to the MQTT broker over SSL/TLS.
