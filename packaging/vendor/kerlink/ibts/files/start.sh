#!/bin/sh -e

# Open firewall ports
iptables_accept() {
	[ -n "${1}" ] || exit 1
	local RULE="OUTPUT -t filter -p tcp --dport ${1} -j ACCEPT"
	iptables -C ${RULE} 2> /dev/null || iptables -I ${RULE}
	local RULE="INPUT -t filter -p tcp --sport ${1} -j ACCEPT"
	iptables -C ${RULE} 2> /dev/null || iptables -I ${RULE}
}

iptables_accept 1883
iptables_accept 8883

(
	# Lock on pid file
	exec 8> /var/run/lora-gateway-bridge.pid
	flock -n -x 8
	trap "rm -f /var/run/lora-gateway-bridge.pid" EXIT

	# Start LoRa Gateway Bridge
	/user/lora-gateway-bridge/lora-gateway-bridge -c /user/lora-gateway-bridge/lora-gateway-bridge.toml 2>&1 | logger -p local1.notice -t lora-gateway-bridge &
	trap "killall -q lora-gateway-bridge" INT QUIT TERM

	# Update pid and wait for it
	pidof lora-gateway-bridge >&8
	wait
) &

