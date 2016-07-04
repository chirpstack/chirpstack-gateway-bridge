# Airfy GateBoard AYTGB-137-8

This document describes how to setup the [packet_forwarder](https://github.com/Lora-net/packet_forwarder/)
for the Airfy GateBoard AYTGB-137-8 board. It assumes your Airfy GateBoard is
connected to a single-board computer like a Raspberry Pi running a Debian(like)
distribution.

## Requirements

Before you're able to compile the packet_forwarder, install a few requirements:

```bash
sudo apt-get install gcc make git
```

## Get the source

!!! warning "Compatibility"
    As the Airfy GateBoard has an FPGA v27 (based on the Semtech SX1301AP2 Ref
    Design), make sure you download 4.0.1+ of
    [lora_gateway](https://github.com/Lora-net/lora_gateway)!

```bash
# get the driver / hardware abstraction layer source
git clone https://github.com/Lora-net/lora_gateway.git

# get the packet_forwarder source
git clone https://github.com/Lora-net/packet_forwarder.git
```

## Building

``` bash
# first build the driver / hal library
cd lora_gateway
make all
cd ..

# secondly, build the packet_forwarder (and tools)
cd packet_forwarder
make all
```

## Configuration & usage

From here on, please follow the instructions documented in:

* [https://github.com/Lora-net/packet_forwarder/blob/master/lora_pkt_fwd/readme.md](https://github.com/Lora-net/packet_forwarder/blob/master/lora_pkt_fwd/readme.md#4-usage)
* [github.com/Lora-net/packet_forwarder/](https://github.com/Lora-net/packet_forwarder/)
* [Getting started](../getting-started.md)
