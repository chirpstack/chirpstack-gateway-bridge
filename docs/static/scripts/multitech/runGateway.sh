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
# the chirpstack-gateway-bridge package.
export MQTT_SERVER="ssl://some.server.net"
export MQTT_USERNAME="username"
export MQTT_PASSWORD="password"

# Start the log file by identifying the device.  Just print it out, too, 
# so we can find it in the system startup output.
export LORA_CARD_MAC=$(mts-io-sysfs show lora/eui)
export LORA_CARD_MAC=${LORA_CARD_MAC//:/}
echo "ChirpStack Gateway Bridge running on device with MAC $LORA_CARD_MAC"
echo "ChirpStack Gateway Bridge running on device with MAC $LORA_CARD_MAC" > /var/log/chirpstack-gateway-bridge.log

/opt/chirpstack-gateway-bridge/bin/chirpstack-gateway-bridge --log-level 5 >> /var/log/chirpstack-gateway-bridge.log 2>&1 &
PID=$!
if [[ $PIDFILE != "" ]]; then
    echo $PID > $PIDFILE
fi
