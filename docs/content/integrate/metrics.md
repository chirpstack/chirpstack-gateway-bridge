---
title: Prometheus metrics
description: Prometheus metrics exposed by the MQTT backend.
menu:
  main:
    parent: integrate
    weight: 4
---

# Prometheus metrics

Independent from the chosen MQTT authentication type, the MQTT integration
expses the following [Prometheus](https://prometheus.io/) metrics for monitoring.

### integration_mqtt_event_count

The number of gateway events published by the MQTT integration (per event).

### integration_mqtt_command_count

The number of commands received by the MQTT integration (per command).

### integration_mqtt_connect_count

The number of times the integration connected to the MQTT broker.

### integration_mqtt_disconnect_count

The number of times the integration disconnected from the MQTT broker.

### integration_mqtt_reconnect_count

The number of times the integration reconnected to the MQTT broker (this also increments the disconnect and connect counters).

