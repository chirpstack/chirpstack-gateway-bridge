---
title: Configuration
menu:
    main:
        parent: install
        weight: 5
---

## Configuration

### Gateway

Modify the [packet-forwarder](https://github.com/lora-net/packet_forwarder)
of your gateway so that it will send its data to the LoRa Gateway Bridge.
You will need to change the following configuration keys:

* `server_address` to the IP address / hostname of the LoRa Gateway Bridge
* `serv_port_up` to `1700` (the default port that LoRa Gateway Bridge is using)
* `serv_port_down` to `1700` (same)

### LoRa Gateway Bridge

To list all configuration options, start `lora-gateway-bridge` with the
`--help` flag. This will display:

```
GLOBAL OPTIONS:
   --udp-bind value       ip:port to bind the UDP listener to (default: "0.0.0.0:1700") [$UDP_BIND]
   --mqtt-server value    mqtt server (e.g. scheme://host:port where scheme is tcp, ssl or ws) (default: "tcp://127.0.0.1:1883") [$MQTT_SERVER]
   --mqtt-username value  mqtt server username (optional) [$MQTT_USERNAME]
   --mqtt-password value  mqtt server password (optional) [$MQTT_PASSWORD]
   --mqtt-ca-cert value   mqtt CA certificate file (optional) [$MQTT_CA_CERT]
   --skip-crc-check       skip the CRC status-check of received packets [$SKIP_CRC_CHECK]
   --log-level value      debug=5, info=4, warning=3, error=2, fatal=1, panic=0 (default: 4) [$LOG_LEVEL]
   --help, -h             show help
   --version, -v          print the version
```

Both cli arguments and environment-variables can be used to pass configuration
options.
