/*
NAME:
   semtech-bridge - Semtech UDP protocol speaking gateway <-> MQTT

USAGE:
   main [global options] command [command options] [arguments...]
   
COMMANDS:
   help, h	Shows a list of commands or help for one command
   
GLOBAL OPTIONS:
   --udp-bind "0.0.0.0:1700"		ip:port to bind the UDP listener to [$UDP_BIND]
   --mqtt-server "tcp://127.0.0.1:1883"	MQTT server [$MQTT_SERVER]
   --mqtt-username 			MQTT username [$MQTT_USERNAME]
   --mqtt-password 			MQTT password [$MQTT_PASSWORD]
   --help, -h				show help
   --version, -v			print the version
   

*/
package main
