## LoRa Semtech Bridge documentation

LoRa Semtech Bridge is a service which abstracts the 
[Semtech protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT)
into JSON over MQTT. This project is part of [LoRa Server](https://github.com/brocaar/loraserver).

## Features

### Connection handling

LoRa Semtech Bridge will handle all the gateway pings / acks.

### JSON

All (uplink) packets are published as JSON messages. Downlink packets
can be published to MQTT and will be transformed back to the Semtech
protocol by the LoRa Semtech Bridge.

### HA setup

Multiple LoRa Semtech Bridge instances can run at the same time. Since
it will subscribe to gateway topics for which it has open connections,
downlink packets are directed to the correct LoRa Semtech
Bridge instance.
**Important:** one gateway connection should always be load-balanced the
same LoRa Semtech Bridge instance!

## Downloads

Pre-compiled binaries are available for:

* Linux (including ARM / Raspberry Pi)
* OS X
* Windows

See [https://github.com/brocaar/lora-semtech-bridge/releases](https://github.com/brocaar/lora-semtech-bridge/releases)
for downloads. Source-code can be found at
[https://github.com/brocaar/lora-semtech-bridge](https://github.com/brocaar/lora-semtech-bridge).

## License

LoRa Semtech Bridge is distributed under the MIT license. See also
[LICENSE](https://github.com/brocaar/lora-semtech-bridge/blob/master/LICENSE).

