---
title: Deployment strategies
menu:
    main:
        parent: install
        weight: 2
description: Information on different ChirpStack Gateway Bridge deployment strategies.
---

# Deployment strategies

There are multiple ways that you can deploy the ChirpStack Gateway Bridge:

## Single instance

The most basic strategy is to connect all your gateways to a single instance
of the ChirpStack Gateway Bridge. This is the easiest option, as installing the
ChirpStack Gateway Bridge on the gateway might involve some additional steps.
Please note that from a security perspective, it is the least secure option.
The UDP protocol implemented by most gateways don't support any
form of authorization and checks that the received data is authentic. It is
however an easy way to get started.

## Multiple instances

For performance and to make the ChirpStack Gateway Bridge highly available, you
can run ChirpStack Gateway Bridge on multiple servers, each connecting to the same
MQTT broker.

**Important:** In case you put a load-balancer in front of the ChirpStack Gateway
Bridge cluster, make sure that each gateway connection is always routed to the
same instance!

## On each gateway

Depending on the capabilities of your gateway, you can deploy the ChirpStack Gateway
Bridge on each of your gateways. This might require a few additional steps in
the setup, but has the following advantages:

### MQTT (using TCP) over UDP

By using MQTT (which uses TCP) over UDP when using the Semtech UDP packet-forwarder
backend, the connection becomes more reliable in case packetloss is common.

### Authentication

It is possible to setup credentials for each gateway, so that only gateways with
valid credentials are able to ingest data.

### SSL/TLS

The MQTT protocol supports SSL/TLS meaning that you are able to setup a secure
connection between your gateways and your MQTT broker. This not only means that
other people are not able to intercept any data, it also means nobody is able
to tamper with your data.
