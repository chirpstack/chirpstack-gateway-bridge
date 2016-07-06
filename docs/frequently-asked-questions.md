# Frequently asked questions

## Packet-loss (packet_forwarder)

There are many issues that can cause packet-loss (data is not received or
transmitted by the gateway). Here are some hints:

* Compile the packet_forwarder with debugging enabled, it might give you a clue
  about what is going wrong. Before compiling the packet_forwarder, you can
  set the debug flags in `lora_gateway/libloragw/library.cfg`.

* Make sure the packet_forwarder is configured with the right configuration.
  Each region has different frequencies as documented by the LoRaWAN specs. See
  the [Lora-net/packet_forwarder](https://github.com/Lora-net/packet_forwarder/tree/master/lora_pkt_fwd/cfg)
  repository for examples.

* packet_forwarder v3.0.0 and up implements just-in-time scheduling, meaning
  it keeps a queue of packets to be transmitted. Prior versions of the
  packet_forwarder have a queue of size 1 which gets overwritten on each
  packet to be transmitted.

### `ERROR: Packet REJECTED, unsupported frequency - [FREQUENCY] (min:0,max:0)`

Make sure the `tx_freq_min` and `tx_freq_max` are present in your
configuration.

### `src/jitqueue.c:343:jit_enqueue(): ERROR: Packet (type=0) REJECTED, collision with packet already programmed at 423844236 (423843356)`

This error occurs when there is a collision between two packets being
scheduled by the packet_forwarder. Please note that a collision does not only
happen when two packets are scheduled at the same time, but also when they are
scheduled at around the same time. The packet_forwarder takes a pre-delay,
post-delay and a tx-margin-delay (which depending on the type of packets being
scheduled for transmission, can be > 100ms). This documented behaviour and can
be found in the [readme.md](https://github.com/Lora-net/packet_forwarder/blob/master/lora_pkt_fwd/readme.md)
of the packet_forwarder documentation (5.3. TX scheduling).

### `WARNING: [gps] GPS out of sync, keeping previous time reference`

To be investigated

## lora-gateway-bridge errors

### `gateway: could not handle packet: gateway: invalid protocol version`

packet_forwarder v3.0.0+ introduced a new protocol version. Please check
[Compatibility](index.md#compatibility).
