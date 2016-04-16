/*
NAME:
   semtech-bridge - communicate with a LoRa gateway (Semtech UDP protocol) over MQTT

USAGE:
   main [global options] command [command options] [arguments...]
   
COMMANDS:
   help, h	Shows a list of commands or help for one command
   
GLOBAL OPTIONS:
   --udp-bind "0.0.0.0:1700"		ip:port to bind the UDP listener to [$UDP_BIND]
   --mqtt-server "tcp://127.0.0.1:1883"	MQTT server [$MQTT_SERVER]
   --mqtt-username 			MQTT username [$MQTT_USERNAME]
   --mqtt-password 			MQTT password [$MQTT_PASSWORD]
   --log-level "4"			debug=5, info=4, warning=3, error=2, fatal=1, panic=0 [$LOG_LEVEL]
   --help, -h				show help
   --version, -v			print the version
   
COPYRIGHT:
   See http://github.com/brocaar/lora-semtech-bridge for copyright information
   

*/
package main
