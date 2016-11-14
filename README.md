# LoRa Gateway Bridge

[![Build Status](https://travis-ci.org/brocaar/lora-gateway-bridge.svg?branch=master)](https://travis-ci.org/brocaar/lora-gateway-bridge)
[![GoDoc](https://godoc.org/github.com/brocaar/lora-gateway-bridge/cmd/gateway-bridge?status.svg)](https://godoc.org/github.com/brocaar/lora-gateway-bridge/cmd/lora-gateway-bridge)
[![Gitter chat](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/loraserver/lora-gateway-bridge)

LoRa Gateway Bridge is a service which abstracts the 
[packet_forwarder UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT)
running on most LoRa gateways into JSON over MQTT. It enables you to use MQTT for
receiving data from and sending data to your gateways.
This project is part of [LoRa Server](https://github.com/brocaar/loraserver).

## Documentation

See [http://docs.loraserver.io/lora-gateway-bridge](http://docs.loraserver.io/lora-gateway-bridge)
for documentation about setting up LoRa Gateway Bridge.

## Compatibility

The table below shows the compatibility between LoRa Gateway Bridge and the
packet_forwarder UDP protocol versions:

| LoRa Gateway Bridge | packet_forwarder protocol version | Note                                                                |
|---------------------|-----------------------------------|---------------------------------------------------------------------|
| 1.x.x               | 1                                 |                                                                     |
| 2.0.x               | 2                                 | This protocol is used since version 3.0.0 of the `packet_forwarder` |
| >= 2.1.x            | 1 & 2 simultaneously              | Both protocol versions are supported and auto-detected              |

## Downloads

Pre-compiled binaries are available from the [releases](https://github.com/brocaar/lora-gateway-bridge/releases) page:

* Linux (including ARM / Raspberry Pi)
* OS X
* Windows

Source-code can be found at [https://github.com/brocaar/lora-gateway-bridge](https://github.com/brocaar/lora-gateway-bridge).

## Building from source

The easiest way to get started is by using the provided 
[docker-compose](https://docs.docker.com/compose/) environment. To start a bash
shell within the docker-compose environment, execute the following command from
the root of this project:

```bash
docker-compose run --rm gatewaybridge bash
```

A few example commands that you can run:

```bash
# run the tests
make test

# compile
make build

# cross-compile for Linux ARM
GOOS=linux GOARCH=arm make build

# cross-compile for Windows AMD64
GOOS=windows BINEXT=.exe GOARCH=amd64 make build

# build the .tar.gz file
make package

# build the .tar.gz file for Linux ARM
GOOS=linux GOARCH=arm make build

# build the .tar.gz file for Windows AMD64
GOOS=windows BINEXT=.exe GOARCH=amd64 make build
```

Alternatively, you can run the same commands from any working
[Go](https://golang.org/) environment. As all requirements are vendored,
there is no need to `go get` these, but make sure vendoring is enabled for
your Go environment or that you have Go 1.6+ installed.

## Issues / feature-requests

Issues or feature-requests can be opened at [https://github.com/brocaar/lora-gateway-bridge/issues](https://github.com/brocaar/lora-gateway-bridge/issues).

## License

LoRa Gateway Bridge is distributed under the MIT license. See 
[LICENSE](https://github.com/brocaar/lora-gateway-bridge/blob/master/LICENSE).
