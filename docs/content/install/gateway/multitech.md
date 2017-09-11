---
title: Multitech
menu:
    main:
        parent: gateway
---

## Multitech

### Multitech Conduit

There are Multitech Conduit models that differ in the operating system and 
versions of software.  In general, the goal is to set up the Conduits so that 
they are running in a packet forwarding mode, forwarding packets to the 
lora-gateway-bridge.  The difference in how this is done can be significant.

Two of the Multitech Conduit platforms are the AEP and the mLinux varieties.
The mLinux version comes in a case suitable for use outdoors.  But once the 
box is removed from the case, the two versions look identical.  Here are a few
ways to tell them apart:

1. When logging in via the serial port behind the Multitech logo cover, they
   display the type of box they are.
2. The AEP model supports a web interface for settings.  The mLinux version
   does not.
3. Because of the difference in step 2, startup scripts in /etc/init.d get
   settings from the web server using curl.  In particular, if curl is used in
   /etc/init.d/lora-network-server, then you are on an AEP device.
3. The AEP model ships with the default login/password as admin/admin.  The 
   mLinux version uses root/root.

In either case, you'll want to obtain the IP address for the device.  This can 
be done using a serial connection from a computer using a USB-to-microUSB cable,
connecting to the plug behind the Multitech logo placard.  Plug the device into 
your network, provide power, and let it boot until the "STATUS" light is 
blinking in aheartbeat pattern.  Connect to the device via a serial terminal 
program.  Once logged in, issue the command "ifconfig" to get the IP address of 
the eth0 connection.  Note that is the IP address is 192.168.2.1, the device is
likely configured to be a DHCP server.  In this case, edit the file 
/etc/network/interfaces, change the line that says, “iface eth0 inet static” to 
“iface eth0 inet dhcp”, and comment out the lines specifying the IP Address and 
netmask by adding a “#” at the beginning of each line:
			#address 192.168.2.1
			#netmask 255.255.255.0
Then reboot, and obtain the issued IP address as outlined above.

The basic setup steps are outlined below for the packet forwarder for each 
device, followed by `Install lora-gateway-bridge` below.

#### Multitech Conduit AEP

Use the web interface to set up the Conduit's packet forwarder.  By default, 
the connection will not be “secure” over https because the device uses a self-
signed certificate.  Accept the certificate to proceed.
 
1. Log in to the interface.
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
    
    ```javascript
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
    ```
    
15. Select “Submit”.
16. Select the “Save and Restart” option on the left menu.

#### Multitech Conduit mLinux

The latest Conduit mLinux version makes setting up the device pretty straight 
forward.  Start by disabling the lora-network-server and enabling the 
lora-packet-forwarder.  This is done by:

1. Create the file /var/config/lora/global_conf.json and create the serrings by 
   referencing the information at  
   http://www.multitech.net/developer/software/lora/conduit-mlinux-convert-to-basic-packet-forwarder/  
   and be sure to update the settings as described in step 14 for the
   Conduit AEP instructions above.

2. Edit /etc/defaults/lora-network-server and change ENABLED="true" to
   ENABLED="false".
3. Edit /etc/defaults/lora-packet-forwarder and change ENABLED="false" to
   ENABLED="true".
4. Ensure that the lora-packet-forwarder will run after reboot by issuing the 
   command:
   
       `update-rc.d lora-packet-forwarder defaults`

#### Multitech Conduit - Install lora-gateway-bridge

Now you will want to set up the lora-gateway-bridge on the device.  The 
following are suggested files and locations:

1. Download the arm build of the lora-gateway-bridge (see `Downloads` on the 
   left), and extract it to `/opt/lora-gateway-bridge/bin/`

2. Create a script to run the application with the appropriate settings for your
   installation in /opt/lora-gateway-bridge/bin/runGateway.sh:
   
   ```
    #!/bin/bash
    # Starts the gateway code

    # Accept a single parameter of where to put the pid of the actual lora-app-server 
    # process.
    if [ "$#" -ge 1 ]; then
        PIDFILE=$1
    else
        PIDFILE=
    fi

    # These placeholder values will be replaced by the install script at startup.
    # A change in these values will require a manual change here or a reinstall of
    # the lora-gateway-bridge package.
    export MQTT_SERVER="ssl://some.server.net"
    export MQTT_USERNAME="username"
    export MQTT_PASSWORD="password"

    # Start the log file by identifying the device.  Just print it out, too, 
    # so we can find it in the system startup output.
    export LORA_CARD_MAC=$(mts-io-sysfs show lora/eui)
    export LORA_CARD_MAC=${LORA_CARD_MAC//:/}
    echo "Lora Gateway Bridge running on device with MAC $LORA_CARD_MAC"
    echo "Lora Gateway Bridge running on device with MAC $LORA_CARD_MAC" > /var/log/lora-gateway-bridge.log

    /opt/lora-gateway-bridge/bin/lora-gateway-bridge --log-level 5 >> /var/log/lora-gateway-bridge.log 2>&1 &
    PID=$!
    if [[ $PIDFILE != "" ]]; then
        echo $PID > $PIDFILE
    fi```
    
3. Then create a startup script in /etc/init.d/lora-gateway-bridge:

   ```
    #!/bin/bash
    #
    # A SysV init script for the lora-gateway-bridge
    #
    ### BEGIN INIT INFO
    # Provides:             lora-gateway-bridge
    # Required-Start:       $syslog $network
    # Required-Stop:        $syslog $network
    # Should-Start:         $local_fs
    # Should-Stop:          $local_fs
    # Default-Start:        2 3 4 5
    # Default-Stop:         0 1 6
    # Short-Description:    lora-gateway-bridge
    # Description:          Sends lora messages from the gateway to the open source server..
    ### END INIT INFO
    #
    PIDFILE=/var/run/lora-gateway-bridge.pid
    NAME=lora-gateway-bridge
    STARTSCRIPT=/opt/lora-gateway-bridge/bin/runGateway.sh
    RETVAL=0

    start() {
        if [ -f $PIDFILE ]; then
            echo "$NAME is already running"
        else
            echo "Starting $NAME"
            # Start the server, passing in the file to get the PID
            $STARTSCRIPT $PIDFILE
            RETVAL=$?
        fi
    }
     
    stop() {
        if [ -f $PIDFILE ]; then
            echo "Shutting down $NAME"
            kill `cat $PIDFILE`
            # Get rid of the pidfile so we don't assume it's running any more.
            rm -f $PIDFILE
            RETVAL=$?
        else
            echo "$NAME is not running."
        fi
    }
     
    restart() {
        echo "Restarting $NAME"
        stop
        start
    }
     
    status() {
        if [ -f $PIDFILE ]; then
            echo "Status for $NAME:"
            ps -ef `cat $PIDFILE`
            RETVAL=$?
        else
            echo "$NAME is not running"
        fi
    }
     
    case "$1" in
        start)
            start
            ;;
        stop)
            stop
            ;;
        status)
            status
            ;;
        restart)
            restart
            ;;
        *)
            echo "Usage: {start|stop|status|restart}"
            exit 1
            ;;
    esac
    exit $RETVAL```
    
4. Make sure the script start on reboot:

    `update-rc.d lora-gateway-bridge defaults`
    
5. Finally, restart the system to get everything running.