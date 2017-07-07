/*
NAME:
   lora-gateway-bridge - abstracts the packet_forwarder protocol into JSON over MQTT

USAGE:
   main.exe [global options] command [command options] [arguments...]

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --udp-bind value       ip:port to bind the UDP listener to (default: "0.0.0.0:1700") [%UDP_BIND%]
   --mqtt-server value    mqtt server (e.g. scheme://host:port where scheme is tcp, ssl or ws) (default: "tcp://127.0.0.1:1883") [%MQTT_SERVER%]
   --mqtt-username value  mqtt server username (optional) [%MQTT_USERNAME%]
   --mqtt-password value  mqtt server password (optional) [%MQTT_PASSWORD%]
   --mqtt-ca-cert value   mqtt CA certificate file (optional) [%MQTT_CA_CERT%]
   --skip-crc-check       skip the CRC status-check of received packets [%SKIP_CRC_CHECK%]
   --log-level value      debug=5, info=4, warning=3, error=2, fatal=1, panic=0 (default: 4) [%LOG_LEVEL%]
   --help, -h             show help
   --version, -v          print the version

COPYRIGHT:
   See http://github.com/brocaar/lora-gateway-bridge for copyright information

*/
package main
