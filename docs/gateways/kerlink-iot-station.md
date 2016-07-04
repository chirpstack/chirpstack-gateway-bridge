# Kerlink LoRa IoT Station

This document describes how to setup the [packet_forwarder](https://github.com/Lora-net/packet_forwarder/)
for the Kerlink LoRa IoT Station.

## Requirements

Since you can't compile the packet_forwarder on the gateway itself, the
following steps describe how to cross-compile the packet_forwarder. The
following commands have been tested on Ubuntu 16.04 LTS. When running them
on a different distribution, you might need to change these commands.

Install the requirements needed to compile the packet_forwarder:

```bash
sudo apt-get install gcc make git
```

!!! note
    You also need the arm cross-compile toolchain from Kerlink in able to
    cross-compile the binaries for ARM. In the following examples it is assumed
    they are installed under /opt/toolchains/arm-2011.03-wirma2.

## Get the source

```bash
# get the driver / hardware abstraction layer source
git clone https://github.com/Lora-net/lora_gateway.git

# get the packet_forwarder source
git clone https://github.com/Lora-net/packet_forwarder.git
```

## Configuration

When compiling the packet_forwarder for a Kerlink, you need to set the
`SPI_DEV_PATH` to `/dev/spidev32766.0`. This can be changed in
`lora_gateway/libloragw/src/loragw_spi.native.c`.
In addtion, `lora_gateway/libloragw/tst/test_loragw_gps.c` needs to be
modified to point to the correct GPS device: 
`i = lgw_gps_enable("/dev/nmea", NULL, 0, &gps_tty_dev);`


## Building

``` bash
# first build the driver / hal library
cd lora_gateway
ARCH=arm CROSS_COMPILE=/opt/toolchains/arm-2011.03-wirma2/bin/arm-none-linux-gnueabi- make all
cd ..

# secondly, build the packet_forwarder (and tools)
cd packet_forwarder
ARCH=arm CROSS_COMPILE=/opt/toolchains/arm-2011.03-wirma2/bin/arm-none-linux-gnueabi- make all
```

## Configuration & usage

From here on, please follow the instructions documented in:

* [https://github.com/Lora-net/packet_forwarder/blob/master/lora_pkt_fwd/readme.md](https://github.com/Lora-net/packet_forwarder/blob/master/lora_pkt_fwd/readme.md#4-usage)
* [github.com/Lora-net/packet_forwarder/](https://github.com/Lora-net/packet_forwarder/)
* [Getting started](../getting-started.md)
