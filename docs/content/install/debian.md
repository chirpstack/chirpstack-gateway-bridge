---
title: Debian / Ubuntu
menu:
    main:
        parent: install
        weight: 4
description: Instructions how to install ChirpStack Gateway Bridge on a Debian or Ubuntu based Linux installation.
---

# Debian / Ubuntu

These steps have been tested using:

* Ubuntu 18.04 (LTS)
* Debian 10 (Buster)

## ChirpStack Debian repository

ChirpStack pre-compiled binaries packaged as Debian (.deb)
packages. In order to activate this repository, execute the following
commands:

{{<highlight bash>}}
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 1CE2AFD36DBCCA00

sudo echo "deb https://artifacts.chirpstack.io/packages/3.x/deb stable main" | sudo tee /etc/apt/sources.list.d/chirpstack.list
sudo apt-get update
{{< /highlight >}}

## Install ChirpStack Gateway Bridge

In order to install ChirpStack Gateway Bridge, execute the following command:

{{<highlight bash>}}
sudo apt-get install chirpstack-gateway-bridge
{{< /highlight >}}

This will setup an user and group, create start scripts for systemd or init.d
(this depends on your version of Debian / Ubuntu). The configuration file is
located at `/etc/chirpstack-gateway-bridge/chirpstack-gateway-bridge.toml`.

## Starting ChirpStack Gateway Bridge

How you need to (re)start and stop ChirpStack Gateway Bridge depends on if your
platform uses systemd or init.d.

### systemd

{{<highlight bash>}}
sudo systemctl [start|stop|restart|status] chirpstack-gateway-bridge
{{< /highlight >}}

### init.d

{{<highlight bash>}}
sudo /etc/init.d/chirpstack-gateway-bridge [start|stop|restart|status]
{{< /highlight >}}

## ChirpStack Gateway Bridge log output

Now you've setup ChirpStack Gateway Bridge and your gateway is configured to forward
it's data to it, it is a good time to verify that data is actually comming in.
This can be done by looking at the ChirpStack Gateway Bridge log output.

Like the previous step, which command you need to use for viewing the
log output depends on if your distribution uses init.d or systemd.

### init.d

All logs are written to `/var/log/chirpstack-gateway-bridge/chirpstack-gateway-bridge.log`.
To view and follow this logfile:

{{<highlight bash>}}
tail -f /var/log/chirpstack-gateway-bridge/chirpstack-gateway-bridge.log
{{< /highlight >}}

### systemd

{{<highlight bash>}}
journalctl -u chirpstack-gateway-bridge -f -n 50
{{< /highlight >}}
