---
title: Prometheus
menu:
  main:
    parent: metrics
    weight: 1
description: Read metrics from the Prometheus metrics endpoint.
---

# Prometheus metrics

ChirpStack Gateway Bridge provides a [Prometheus](https://prometheus.io/) metrics endpoint
for monitoring the performance of the ChirpStack Gateway Bridge service. Please refer to
the [Prometheus](https://prometheus.io/) website for more information on
setting up and using Prometheus.

## Configuration

Please refer to the [Configuration documentation]({{<ref "install/config.md">}}).

## Metrics

### Go runtime metrics

These metrics are prefixed with `go_` and provide general information about
the process like:

* Garbage-collector statistics
* Memory usage
* Go go-routines

### MQTT integration metrics

These metrics are prefixed with `integration_mqtt_` and provide:

* The number of gateway events published by the MQTT integration
* The nubmer of commands received by the MQTT integration
* The number of times the integration connected to the MQTT broker
* The number of times the integration disconnected from the MQTT broker
* The number of times the integration reconnected to the MQTT broker

### Backends

Please refer to [Backends](/gateway-bridge/backends/) for the provided metrics per backend.
