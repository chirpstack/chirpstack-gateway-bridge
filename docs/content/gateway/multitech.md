---
title: Multitech
description: Installation of the ChirpStack Gateway Bridge service on the Multitech Conduit gateway.
menu:
  main:
    parent: gateway
---

# Multitech

## Multitech Conduit

* [Product detail page](https://www.multitech.com/brands/multiconnect-conduit)
* [Product documentation page](http://www.multitech.net/developer/products/multiconnect-conduit-platform/)

After completing these steps, you will have a Multitech Conduit running both the
packet-forwarder and ChirpStack Gateway Bridge. The packet-forwarder will forward
the UDP data to `localhost:1700` and the ChirpStack Gateway Bridge will forward
this data over MQTT to a MQTT broker.

There are two different Multitech Conduit firmware versions: mLinux and AEP.
The AEP version comes with a web-interface and IBM Node-RED pre-installed.
The mLinux version provides an open Linux development environment and is
recommended when complete (firmware) control is preferred.

Please refer to [http://www.multitech.net/developer/products/multiconnect-conduit-platform/](http://www.multitech.net/developer/products/multiconnect-conduit-platform/)
for more documentation on on the Multitech Conduit.

**Note:** It is possible to install mLinux on an AEP Conduit version following the
steps below. This is recommended when you don't rely on any software provided by
the AEP firmware.

### Getting the IP address

Before continuing, you'll want to obtain the IP address of the Conduit.  This can 
be done using a serial connection from a computer using a USB-to-microUSB cable,
connecting to the plug behind the Multitech logo placard. Plug the device into 
your network, provide power, and let it boot until the "STATUS" light is 
blinking in a heartbeat pattern.  Connect to the device via a serial terminal 
program. Example (where `/dev/ttyACM0` should equal to the serial interface):

{{<highlight bash>}}
screen /dev/ttyACM0 115200
{{</highlight>}}

Once logged in, issue the command "ifconfig" to get the IP address of 
the eth0 connection.  Note that if the IP address is `192.168.2.1`, the device is
likely configured with a static IP.  In this case, edit the file 
`/etc/network/interfaces`, change the line that says, `iface eth0 inet static` to 
`iface eth0 inet dhcp`, and comment out the lines specifying the IP address and 
netmask by adding a `#` at the beginning of each line:

{{<highlight text>}}
# address 192.168.2.1
# netmask 255.255.255.0
{{< /highlight >}}

Then execute `/etc/init.d/networking restart`, and obtain the issued IP address
as outlined above.

### Upgrading / migrating from AEP to the latest mLinux

The suggested way to setup the packet-forwarder and ChirpStack Gateway Bridge on a
Multitech Conduit is by using the base mLinux firmware image
(`mlinux-base-*.jffs2`). This firmware
image installs the minimal amount of software needed to boot the Conduit,
but does not contain any other software which could conflict with your setup.
The latest firmware version can be downloaded from: [http://www.multitech.net/mlinux/images/mtcdt/](http://www.multitech.net/mlinux/images/mtcdt/).

The [Flashing mLinux Firmware](http://www.multitech.net/developer/software/mlinux/using-mlinux/flashing-mlinux-firmware-for-conduit/)
instructions cover both upgrading mLinux to a newer version and converting an
AEP model into a mLinux model. In both the AEP migrate and mLinux upgrade you
can use the **Using Auto-Flash During Reboot** steps. **Again, make sure to
use the `mlinux-base*.jffs2` image!**

**Important:** after flashing the device, you need to update the `opkg` cache
which can be done with the command: `opkg update`.

Example commands for upgrading / migrating:

{{<highlight bash>}}
mkdir /var/volatile/flash-upgrade
cd /var/volatile/flash-upgrade
wget -O uImage.bin http://www.multitech.net/mlinux/images/mtcdt/5.0.0/uImage--4.9.87-r9.2-mtcdt-20190618233259.bin
wget -O rootfs.jffs2 http://www.multitech.net/mlinux/images/mtcdt/5.0.0/mlinux-base-image-mtcdt-20190618233259.rootfs.jffs2
touch /var/volatile/do_flash_upgrade
reboot
{{< /highlight >}}

Then after the reboot update the `opkg` cache:

{{<highlight bash>}}
opkg update
{{< /highlight >}}

### Setting up the ChirpStack Gateway Bridge

1. Log in using SSH or use the USB to serial interface.

2. Download the latest `chirpstack-gateway-bridge` `.ipk` package from:
   [https://artifacts.chirpstack.io/vendor/multitech/conduit/](https://artifacts.chirpstack.io/vendor/multitech/conduit/).
   Example (assuming you want to install `chirpstack-gateway-bridge_3.1.0-r1_arm926ejste.ipk`):
   {{<highlight bash>}}
   wget https://artifacts.chirpstack.io/vendor/multitech/conduit/chirpstack-gateway-bridge_3.1.0-r1_arm926ejste.ipk
   {{< /highlight >}}

3. Now that the `.ipk` package is stored on the Conduit, you can install it
   using the `opkg` package-manager utility. Example (assuming the same
   `.ipk` file):
   {{<highlight bash>}}
   opkg install chirpstack-gateway-bridge_3.1.0-r1_arm926ejste.ipk
   {{< /highlight >}}

4. Update the MQTT connection details so that ChirpStack Gateway Bridge is able to
   connect to your MQTT broker. You will find the configuration file in the
   `/var/config/chirpstack-gateway-bridge` directory.

5. Start ChirpStack Gateway Bridge and ensure it will be started on boot.
   Example:
   {{<highlight bash>}}
   /etc/init.d/chirpstack-gateway-bridge start
   {{< /highlight >}}

### Setting up the packet-forwarder

The packages installed with the commands below will by default choose the US
or EU band configuration, based on the used hardware. Please refer to the
[Multitech documentation](http://www.multitech.net/developer/software/lora/aep-lora-packet-forwarder/)
for alternative configurations.

To find out if you should follow the `MTAC-LORA-H` or `MTAC-LORA`
instructions, you could use the following commands:

{{<highlight bash>}}
# mCard in ap1 slot
mts-io-sysfs show ap1/product-id

# mCard in ap2 slot
mts-io-sysfs show ap2/product-id

# detect first mCard
mts-io-sysfs show lora/product-id
{{< /highlight >}}

#### mLinux with MTAC-LORA-H-915 or MTAC-LORA-H-868

**Important:** Follow these steps only when you have a `MTAC-LORA-H` (v1.5)
card which uses the SPI interface.

1. Log in using SSH or use the USB to serial interface.

2. Download the latest `lora-packet-forwarder` `*.ipk` package
   from [https://artifacts.chirpstack.io/vendor/multitech/conduit/](https://artifacts.chirpstack.io/vendor/multitech/conduit/).
   Example:
   {{<highlight bash>}}
   wget https://artifacts.chirpstack.io/vendor/multitech/conduit/lora-packet-forwarder_4.0.1-r5.0_mtcdt.ipk
   {{< /highlight >}}

3. Now this `.ipk` package is stored on the Conduit, you can install it
   using the `opkg` package-manager utility. Example (assuming the same
   `.ipk` file):
   {{<highlight bash>}}
   opkg install lora-packet-forwarder_4.0.1-r5.0_mtcdt.ipk
   {{< /highlight >}}

4. As the package has the same name as the default package provided by Multitech
   you need to 'flag' the package with the status `hold` to make sure an
   `opkg upgrade` does not overwrite it:
   {{<highlight bash>}}
   opkg flag hold lora-packet-forwarder
   {{< /highlight >}}

5. Start the packet-forwarder and enable it to start on boot. Note that the
   `-ap1` or `-ap2` suffix refers to the slot in which your `MTAC-LORA-H` card
   is present. In case you have two `MTAC-LORA-H` cards, this allows you to start
   two packet-forwarder instances with each using their own configuration.
   Example:
   {{<highlight bash>}}
   /etc/init.d/lora-packet-forwarder-ap1 start
   update-rc.d lora-packet-forwarder-ap1 defaults
   {{< /highlight >}}

   **Note:** on the first start of the packet-forwarder it will detect for you
   the version of your `MTAC-LORA-H` cards (868 or 915) and if your Conduit
   has an onboard GPS. It will then automatically generate the correct
   configuration for you.

   Configuration is stored in `/var/config/lora-packet-forwarder-ap1` and
   `/var/config/lora-packet-forwarder-ap2` directories and can be modified after
   the first start.

   The build recipe of the `.ipk` package can be found at:
   [https://github.com/brocaar/chirpstack-yocto](https://github.com/brocaar/chirpstack-yocto).

#### mLinux with MTAC-LORA-915 or MTAC-LORA-868

**Important:** Follow these steps only when you have a `MTAC-LORA` (v1.0
card which uses the FTDI interface.

1. Log in using SSH or use the USB to serial interface.

2. Download the latest `lora-packet-forwarder-usb` `*.ipk` package
   from [https://artifacts.chirpstack.io/vendor/multitech/conduit/](https://artifacts.chirpstack.io/vendor/multitech/conduit/).
   Example:
   {{<highlight bash>}}
   wget https://artifacts.chirpstack.io/vendor/multitech/conduit/lora-packet-forwarder-usb_1.4.1-r2.0_arm926ejste.ipk
   {{< /highlight >}}

3. Now this `.ipk` package is stored on the Conduit, you can install it
   using the `opkg` package-manager utility. Example (assuming the same
   `.ipk` file):
   {{<highlight bash>}}
   opkg install lora-packet-forwarder-usb_1.4.1-r2.0_arm926ejste.ipk
   {{< /highlight >}}

4. As the package has the same name as the default package provided by Multitech
   you need to 'flag' the package with the status `hold` to make sure an
   `opkg upgrade` does not overwrite it:
   {{<highlight bash>}}
   opkg flag hold lora-packet-forwarder-usb
   {{< /highlight >}}

5. Start the packet-forwarder and enable it to start on boot. 
   Example:
   {{<highlight bash>}}
   /etc/init.d/lora-packet-forwarder-usb start
   update-rc.d lora-packet-forwarder-usb defaults
   {{< /highlight >}}

   **Note:** on the first start of the packet-forwarder it will detect for you
   the version of your `MTAC-LORA` cards (868 or 915). It will then automatically
   generate the correct configuration for you.

   Configuration is stored in `/var/config/lora-packet-forwarder-usb` directory
   and can be modified after the first start.

   The build recipe of the `.ipk` package can be found at:
   [https://github.com/brocaar/chirpstack-yocto](https://github.com/brocaar/chirpstack-yocto).

#### AEP: Setting up the packet-forwarder

Use the web interface to set up the Conduit's packet forwarder.  By default, 
the connection will not be “secure” over https because the device uses a self-
signed certificate.  Accept the certificate to proceed.

1. Log in to the web-interface.
2. On the home screen, you should be able to see information about the version of the LoRa card.  Find the corresponding section on the web page:  
  http://www.multitech.net/developer/software/lora/aep-lora-packet-forwarder/  
   This page has links to basic configuration for each card version which you 
   will need below.
3. You should see the “First-Time Setup Wizard” welcome screen.  If not, you
   can access it using the menu on the left side of the screen.  Once started,
   Click “Next”.
4. Set your password on the device for the admin account and click “Next”.
5. Set the date and time and click “Next”.
6. For the “Cellular PPP Configuration,” leave all fields blank and click 
   “Next”.
7. For the “Cellular PPP Authentication”, leave the Type as NONE, and click 
   “Next”.
8. For the “IP Setup - eth0”, set up as appropriate for your network and click 
   “Next”.
9. For the “Access Configuration”, set up as appropriate for your network and 
   click “Done”.
10. On the left menu, select “Setup”, and then “LoRa Network Server” from the 
    submenu.
11. In the “LoRa Configuration” window:
12. At the top of the left column, check “Enabled”.
13. At the top of the right column, set “Mode” to be “PACKET FORWARDER”.
14. In the “Config” text box, copy and paste the configuration data for your 
    MTAC LoRa card and region.  In addition, you will want to modify/add the 
    following configuration details in the gateway\_conf section.  Leave any 
    other settings in this section as they are.  The ref\_* fields should be set 
    for the gateway.  (Altitude is specified in meters.):
    
    {{<highlight javascript>}}
    {
        ...
        "gateway_conf": {
            ...
            "server_address": "localhost",
            "serv_port_up": 1700,
            "serv_port_down": 1700,
            "fake_gps": true,
            "ref_latitude": 39.9570133,
            "ref_longitude": -105.1603241,
            "ref_altitude": 1664
        }
    }
    {{< /highlight >}}
    Note that the serv_port_up and serv_port_down represent the ports used to 
    communicate with the chirpstack-gateway-bridge, usually on localhost (the 
    server_address parameter).  See the image above.
    
15. Select “Submit”.
16. Select the “Save and Restart” option on the left menu.

5. Be sure to add the gateway to the ChirpStack Application Server.  See [here](/application-server/use/gateways/).

6. Finally, restart the system to get everything running.
