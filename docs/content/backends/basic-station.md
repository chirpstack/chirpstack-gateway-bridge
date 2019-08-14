---
title: Basic Station
description: Semtech Basic Station packet-forwarder backend.
menu:
  main:
    parent: backends
---

# Semtech Basic Station packet-forwarder

The [Semtech Basic Station](https://doc.sm.tc/station/) backend implements
the [LNS protocol](https://doc.sm.tc/station/tcproto.html). It exposes a
websocket handler to which Basic Station powered gateways can connect.

## Authentication modes

### No Authentication

The LoRa Gateway Bridge will not perform any authentication or authorization
and all connections are accepted.

### TLS Server Authentication

The `basic_station` backend [Configuration]({{<ref "install/config.md">}}) must
be configured with a `tls_cert` and `tls_key`. The CA certificate used to sign
the server TLS certificates must be provided to the Basic Station so that the
gateway is able to authenticate the LoRa Gateway Bridge.

### TLS Server and Client Authentication

Added to the _TLS Server Authentication_, the `basic_station` backend [Configuration]({{<relref "install/config.md">}})
must be configured with the `ca_cert` used to sign the used Basic Station
client certificates.

**Important:** The _Common Name (CN)_ must contain the _Gateway ID_ (64 bits)
of each gateway as a HEX encoded string, e.g. `0102030405060708`. 

## Channel-plan / `router_config`

When using Basic Station powered gateways, assigning a _Gateway Profile_ to a
gateway within LoRa (App) Server is a requirement. As part of the connection
handshake, LoRa Gateway Bridge must send the channel-plan configuration to the
Basic Station. The flow for this is:

* Basic Station sends a `version` message to the LoRa Gateway Bridge
* The LoRa Gateway Bridge sends a `stats` [Event]({{<ref "payloads/events.md">}})
* [LoRa Server](/loraserver/) responds with a `config` [Command]({{<ref "payloads/commands.md">}}), containing the channel-plan from the _Gateway Profile_
* The LoRa Gateway Bridge forwards this as `router_config` to the Basic Station

## Known issues

* The Basic Station does not send RX / TX stats

## Prometheus metrics

The Semtech Basic Station packet-forwarder backend exposes several [Prometheus](https://prometheus.io/)
metrics for monitoring.

### backend_basicstation_websocket_ping_pong_count

The number of WebSocket Ping/Pong requests sent and received (per event type).

### backend_basicstation_websocket_received_count

The number of WebSocket messages received by the backend (per msgtype).

### backend_basicstation_websocket_sent_count

The number of WebSocket messages sent by the backend (per msgtype).

### backend_basicstation_gateway_connect_count

The number of gateway connections received by the backend.

### backend_basicstation_gateway_disconnect_count

The number of gateways that disconnected from the backend.
