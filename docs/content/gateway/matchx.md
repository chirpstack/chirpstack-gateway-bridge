---
title: MatchX
description: Onfigure the MatchX gateway to work with the ChirpStack Gateway Bridge service.
menu:
  main:
    parent: gateway
---

# MatchX

## Matchbox gateway

* [Product detail page](https://www.matchx.io/hardware/)

The MatchX Matchbox is an OpenWRT / Lede
base LoRa<sup>&reg;</sup> gateway, which by default sends its data to the MatchX hosted network-server
(which is based on the ChirpStack open-source LoRaWAN<sup>&reg;</sup> Network Server Stack).
By default it connects to the MatchX backend over a VPN connection.

To connect the gateway to your own environment:

* Power on the gateway and follow the MatchX provided manual to setup the
  wifi connection (in case needed).

* Create a MatchX account and register the gateway.

* On the page with the gateway details is a button **Global LoRa Config: [Edit Config]**.
  click on this button and scroll down to the bottom where you find the
  `server_address` setting. Change this to the IP or hostname on which
  your ChirpStack Gateway Bridge instance is listening for UDP packets.

**Note:** Running the ChirpStack Gateway Bridge on the gateway itself will probably
not work without re-compiling the kernel with FPU emulation. This has not been
tested. In order to get `root` access, you need to contact the MatchX support
for getting the password.
