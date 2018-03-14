---
title: Debugging
menu:
    main:
        parent: install
        weight: 6
---

# Debugging

The follwing steps can be used to debug any LoRa Gateway Bridge related issues.

## Receiving MQTT messages

To validate that the LoRa Gateway Bridge is publishing LoRa frames to the
MQTT broker, you can subscribe to the `gateway/+/rx` MQTT topic. When using
the `mosquitto_sub` utility, you can execute the following command:

```bash
mosquitto_sub -v -t "gateway/+/rx"
```

Of course you will only see data when the gateway is receiving LoRa frames.
Make sure that your node is sending some data.

When you don't see any frames appearing, this could be caused by the following issues:

* LoRa Gateway Bridge does not receive data from the packet-forwarder
* The MQTT credentials / authorizations are invalid (the user is not authorized
  to subscribe to the MQTT topic)

## Is LoRa Gateway Bridge receiving data

To validate that the LoRa Gateway Bridge is receiving, you can take a look
at the LoRa Gateway Bridge logs. How to retrieve the logs depends on how you
installed the LoRa Gateway Bridge. When you followed the [Debian / Ubuntu]({{<ref "install/debian.md">}})
installation steps, this is either:

* `journalctl -f -n 100 -u lora-gateway-bridge`
* `tail -f -n 100 /var/log/lora-gateway-bridge/lora-gateway-bridge.log`

With an interval of a couple of seconds, the following lines should appear (`PullData` / `PullACK`):

```text
level=info msg="gateway: received udp packet from gateway" addr=127.0.0.1:34268 protocol_version=2 type=PullData
level=info msg="gateway: sending udp packet to gateway" addr=127.0.0.1:34268 protocol_version=2 type=PullACK
level=info msg="gateway: received udp packet from gateway" addr=127.0.0.1:34268 protocol_version=2 type=PullData
level=info msg="gateway: sending udp packet to gateway" addr=127.0.0.1:34268 protocol_version=2 type=PullACK
level=info msg="gateway: received udp packet from gateway" addr=127.0.0.1:43827 protocol_version=2 type=PushData
level=info msg="gateway: sending udp packet to gateway" addr=127.0.0.1:43827 protocol_version=2 type=PushACK
```

The packet-forwarder sends every couple of seconds a ping (`PullData`) to which
the LoRa Gateway Bridge must respond (`PullACK`).

When you don't see these logs, this could be caused by the following issues:

* The packet-forwarder is not running
* The packet-forwarder is forwarding to a different host / port
* Firewall / NAT issue (in case the packet-forwarder and LoRa Gateway Bridge are not installed on the same device)

When you do see these logs, but you don't receive any MQTT messages, this could
be caused by:

* The MQTT user is not authorized to publish data to the MQTT topic or, the MQTT
  user for subscribing is not authorized to subscribe to the MQTT topic.
* The node is sending on a frequency that the packet-forwarder is not
  configured for.

## Is the packet-forwarder running

To validate that the packet-forwarder is running, you could execute the
following command on your gateway:

```bash
# when the LoRa Gateway Bridge is running on the same device, add -i lo
sudo tcpdump -AUq -i lo port 1700

# when the LoRa Gateway Bridge is running somewhere else
sudo tcpdump -AUq port 1700
```

This should output something like:

```text
11:42:00.114726 IP localhost.34268 > localhost.1700: UDP, length 12
E..(..@.@."................'.....UZ.....
11:42:00.130292 IP localhost.1700 > localhost.34268: UDP, length 4
E.. ..@.@.".....................
11:42:10.204723 IP localhost.34268 > localhost.1700: UDP, length 12
E..(.&@.@..................'.x...UZ.....
11:42:10.206503 IP localhost.1700 > localhost.34268: UDP, length 4
E.. .'@.@....................x..
11:42:10.968420 IP localhost.43827 > localhost.1700: UDP, length 113
E....h@.@............3...y.......UZ.....{"stat":{"time":"2017-09-11 11:42:10 GMT","rxnb":0,"rxok":0,"rxfw":0,"ackr":100.0,"dwnb":0,"txnb":0}}
11:42:10.970702 IP localhost.1700 > localhost.43827: UDP, length 4
E.. .i@.@..b...........3........
11:42:20.284752 IP localhost.34268 > localhost.1700: UDP, length 12
E..(..@.@..................'.....UZ.....
11:42:20.289256 IP localhost.1700 > localhost.34268: UDP, length 4
E.. ..@.@.......................
11:42:30.364780 IP localhost.34268 > localhost.1700: UDP, length 12
E..( .@.@..................'.S7..UZ.....
11:42:30.366310 IP localhost.1700 > localhost.34268: UDP, length 4
E..  .@.@....................S7.
```

What we see in this log:

* `localhost.34268 > localhost.1700`: packet sent from the packet-forwarder to the LoRa Gateway Bridge
* `localhost.1700 > localhost.34268`: packet sent from the LoRa Gateway Bridge to the packet-forwarder

A log example where the packet-forwarder is sending data, but not receiving
any data (caused by a firewall / NAT issue?):

```text
11:47:22.724729 IP localhost.34268 > localhost.1700: UDP, length 12
E..([.@.@..................'.....UZ.....
11:47:32.804724 IP localhost.34268 > localhost.1700: UDP, length 12
E..(].@.@..................'.!...UZ.....
11:47:40.964220 IP localhost.43827 > localhost.1700: UDP, length 113
E...`/@.@............3...y...s...UZ.....{"stat":{"time":"2017-09-11 11:47:40 GMT","rxnb":0,"rxok":0,"rxfw":0,"ackr":100.0,"dwnb":0,"txnb":0}}
11:47:42.884727 IP localhost.34268 > localhost.1700: UDP, length 12
E..(`f@.@..\...............'.-C..UZ.....
```

When you don't see any of these, this could be caused by the following issues:

* The packet-forwarder is not running
* The packet-forwarder is forwarding to a different host / port

## Where is the packet-forwarder sending data to

Inspect the `local_conf.json` of the packet-forwarder running on your gateway.
You might need to refer to your gateway manual to find out where you can locate
this file. This file could contain the following content:

```json
{
    "gateway_conf": {
        ...
        "serv_port_down": 1700,
        "serv_port_up": 1700,
        "server_address": "localhost",
        ...
    }
}
```

What we learn from this file is that:

* It uses port `1700` (default port used by LoRa Gateway Bridge)
* It sends data to `localhost` (when LoRa Gateway Bridge is installed on the same device)

Make sure the ports and `server_address` are correct. In case LoRa Gateway
Bridge is not running on the same device, you need to replace this with the
correct hostname or IP of your LoRa Gateway Bridge instance. After making any
changes, don't forget to restart the packet-forwarder.

See [https://github.com/Lora-net/packet_forwarder/tree/master/lora_pkt_fwd](https://github.com/Lora-net/packet_forwarder/tree/master/lora_pkt_fwd)
for more information about the packet-forwarder.