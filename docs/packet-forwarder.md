The [packet_forwarder](https://github.com/Lora-net/packet_forwarder/) is an
application which runs on your gateway. It's responsibility is to:

* forward received uplink packets (over UDP)
* forward statistics (over UDP)
* enqueue and transmit downlink packets (received over UDP)

## Building from source

This guide assumes you're on the same platform as where the packet_forwarder
will run eventually and it assumes you have all tools installed to compile
C source-code.

### Get the source

``` bash
# get the driver / hardware abstraction layer source
git clone https://github.com/Lora-net/lora_gateway.git

# get the packet_forwarder source
git clone https://github.com/Lora-net/packet_forwarder.git
```

### Debugging

For testing and debugging, it is advised to enable debugging as it will give
you more feedback when for example packets get rejected. You can find the
debug flags in `lora_gateway/libloragw/library.cfg`.

### Target-specific configuration

!!! note "please contribute"
    This list of target-specific changes is far from complete. Please create
    a pull-request to add your target to this list.

#### BeagleBone Black

When compiling the packet_forwarder for a BeagleBone Black, you need to set the
`SPI_DEV_PATH` to `/dev/spidev1.0`. This can be changed in
`lora_gateway/libloragw/src/loragw_spi.native.c`.


#### Kerlink

When compiling the packet_forwarder for a Kerlink, you need to set the
`SPI_DEV_PATH` to `/dev/spidev32766.0`. This can be changed in
`lora_gateway/libloragw/src/loragw_spi.native.c`.
In addtion, `lora_gateway/libloragw/tst/test_loragw_gps.c` needs to be
modified to point to the correct GPS device: 
`i = lgw_gps_enable("/dev/nmea", NULL, 0, &gps_tty_dev);`

Cross compiler needs to be installed and configured in: `lora_gateway\Makefile` and in
`packet_forwarder\Makefile`:
```
ARCH ?=arm
CROSS_COMPILE ?=/opt/toolchains/arm-2011.03-wirma2/bin/arm-none-linux-gnueabi-
```

### Building

``` bash
# first build the driver / hal library
cd lora_gateway
make all
cd ..

# secondly, build the packet_forwarder (and tools)
cd packet_forwarder
make all
```

### Configuration & usage

From here on, please follow the instructions documented in:

* [https://github.com/Lora-net/packet_forwarder/blob/master/lora_pkt_fwd/readme.md](https://github.com/Lora-net/packet_forwarder/blob/master/lora_pkt_fwd/readme.md#4-usage)
* [github.com/Lora-net/packet_forwarder/](https://github.com/Lora-net/packet_forwarder/)
