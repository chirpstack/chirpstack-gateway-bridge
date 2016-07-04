# IMST iC880A

This document describes how to setup the [packet_forwarder](https://github.com/Lora-net/packet_forwarder/)
for the [IMST iC88A](http://www.wireless-solutions.de/products/radiomodules/ic880a.html) board.
It assumes the IMST iC880A is connected to a single-board compulter like
a BeagleBone Black or Raspberry PI) running a Debian(like) distribution.

## Requirements

Before you're able to compile the packet_forwarder, install a few requirements:

```bash
sudo apt-get install gcc make git
```

## Get the source

```bash
# get the driver / hardware abstraction layer source
git clone https://github.com/Lora-net/lora_gateway.git

# get the packet_forwarder source
git clone https://github.com/Lora-net/packet_forwarder.git
```

## Target specific changes

### BeagleBone Black

When compiling the packet_forwarder for a BeagleBone Black, you need to set the
`SPI_DEV_PATH` to `/dev/spidev1.0`. This can be changed in
`lora_gateway/libloragw/src/loragw_spi.native.c`.

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
