---
title: Basic Station
description: Semtech Basic Station Packet Forwarder backend.
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

The ChirpStack Gateway Bridge will not perform any authentication or authorization
and all connections are accepted.

### TLS Server Authentication

The `basic_station` backend [Configuration]({{<ref "/install/config.md">}}) must
be configured with a `tls_cert` and `tls_key`. The CA certificate used to sign
the server TLS certificates must be provided to the Basic Station so that the
gateway is able to authenticate the ChirpStack Gateway Bridge.

### TLS Server and Client Authentication

Added to the _TLS Server Authentication_, the `basic_station` backend [Configuration]({{<ref "/install/config.md">}})
must be configured with the `ca_cert` used to sign the used Basic Station
client certificates.

**Important:** The _Common Name (CN)_ must contain the _Gateway ID_ (64 bits)
of each gateway as a HEX encoded string, e.g. `0102030405060708`. 

## Channel-plan / `router_config`

You must configure the gateway channel-plan in the ChirpStack Gateway Bridge
[Configuration]({{<ref "/install/config.md">}}) file.

**Note:** In previous versions of the ChirpStack Gateway Bridge, you had to configure
a _Gateway Profile_. This has been deprecated if favor of directly configuring
the channels in the configuration file.

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
