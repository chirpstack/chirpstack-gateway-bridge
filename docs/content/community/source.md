---
title: Source
menu:
    main:
        parent: community
        weight: 3
description: How to get the ChirpStack Gateway Bridge source and how to compile this into an executable binary.
---

# ChirpStack Gateway Bridge source

Source-code can be found at [https://github.com/brocaar/chirpstack-gateway-bridge](https://github.com/brocaar/chirpstack-gateway-bridge).

## Building

### With Docker

The easiest way to get started is by using the provided 
[docker-compose](https://docs.docker.com/compose/) environment. To start a bash
shell within the docker-compose environment, execute the following command from
the root of this project:

{{<highlight bash>}}
docker-compose run --rm chirpstack-gateway-bridge bash
{{< /highlight >}}

### Without Docker

It is possible to build ChirpStack Gateway Bridge without Docker. However this requires
to install a couple of dependencies (depending your platform, there might be
pre-compiled packages available):

#### Go

Make sure you have [Go](https://golang.org/) installed (1.11+). As
ChirpStack Gateway Bridge, the repository must be cloned outside the `$GOPATH`.

### Example commands

A few example commands that you can run:

{{<highlight bash>}}
# install development requirements
make dev-requirements

# run the tests
make test

# compile
make build

# compile snapshot for supported architectures (using goreleaser)
make snapshot
{{< /highlight >}}
