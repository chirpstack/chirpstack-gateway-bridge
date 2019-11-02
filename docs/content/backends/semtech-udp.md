---
title: Semtech UDP
description: Semtech UDP packet-forwarder backend.
menu:
  main:
    parent: backends
---

# Semtech UDP packet-forwarder backend

The [Semtech UDP packet-forwarder](https://github.com/lora-net/packet_forwarder)
backend abstracts the [UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT).

It is compatible with:

* v1 of the protocol
* v2 of the protocol
* Modified format used by the [Kerlink iBTS](https://www.kerlink.com/product/wirnet-ibts/)
  containing the (encrypted) fine-timestamp

## Configuration

When the `semtech_udp` backend has been enabled, make sure your packet-forwarder
`global_conf.json` or `local_conf.json` is configured correctly, under `gateway_conf`,
the `server_address`, `serv_port_up` and `serv_port_down` must be configured so
that data is forwarded to the ChirpStack Gateway Bridge instance.

{{<highlight text>}}
"gateway_conf": {
	"gateway_ID": "AA555A0000000000",
	"server_address": "localhost",
	"serv_port_up": 1700,
	"serv_port_down": 1700,
	...
}
{{</highlight>}}

## Deployment

The ChirpStack Gateway Bridge can be deployed either on the gateway (recommended)
and "in the cloud". In the latter case, multiple gateways can connect to the
same ChirpStack Gateway Bridge instance.

When the ChirpStack Gateway Bridge is deployed on the gateway, you will benefit from
the MQTT authentication / authorization layer and optional TLS.

## Prometheus metrics

The Semtech UDP packet-forwarder backend exposes several [Prometheus](https://prometheus.io/)
metrics for monitoring.

### backend_semtechudp_udp_sent_count

The number of UDP packets sent by the backend (per packet_type).


### backend_semtechudp_udp_received_count

The number of UDP packets received by the backend (per packet_type).

### backend_semtechudp_gateway_connect_count

The number of gateway connections received by the backend.

### backend_semtechudp_gateway_disconnect_count

The number of gateways that disconnected from the backend.
