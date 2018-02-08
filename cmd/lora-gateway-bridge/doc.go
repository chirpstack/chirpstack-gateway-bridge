/*
LoRa Gateway Bridge abstracts the packet_forwarder protocol into JSON over MQTT
	> documentation & support: https://docs.loraserver.io/lora-gateway-bridge
	> source & copyright information: https://github.com/brocaar/lora-gateway-bridge

Usage:
  lora-gateway-bridge [flags]
  lora-gateway-bridge [command]

Available Commands:
  configfile  Print the LoRa Gateway Bridge configuration file
  help        Help about any command
  version     Print the LoRa Gateway Bridge version

Flags:
  -c, --config string   path to configuration file (optional)
  -h, --help            help for lora-gateway-bridge
      --log-level int   debug=5, info=4, error=2, fatal=1, panic=0 (default 4)

Use "lora-gateway-bridge [command] --help" for more information about a command.

*/
package main
