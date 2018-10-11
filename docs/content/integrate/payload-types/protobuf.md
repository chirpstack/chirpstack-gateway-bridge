---
title: Protocol Buffers
menu:
    main:
        parent: payload-types
---

# Protocol Buffers

The [Protocol Buffers](https://developers.google.com/protocol-buffers/) serialization
format is a binary format which is more bandwidth efficient. Using the
Protocol Buffers tooling, it is possible to generate code for various
programming languages. This is planned to become the default serialization
format in the future.

### Gateway statistics

This message is defined by the [GatewayStats](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.

### Uplink frames

This message is defined by the [UplinkFrame](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.

### Downlink frames

This message is defined by the [DownlinkFrame](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.

### Downlink acknowledgements

This message is defined by the [DownlinkTXAck](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.

### Gateway configuration

This message is defined by the [GatewayConfiguration](https://github.com/brocaar/loraserver/blob/master/api/gw/gw.proto)
Protocol Buffers message.
