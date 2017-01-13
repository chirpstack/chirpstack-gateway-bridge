#!/usr/bin/env bash

NAME=lora-gateway-bridge
BIN_DIR=/usr/bin
SCRIPT_DIR=/usr/lib/$NAME/scripts
LOG_DIR=/var/log/$NAME
DAEMON_USER=gatewaybridge
DAEMON_GROUP=gatewaybridge

function install_init {
	cp -f $SCRIPT_DIR/init.sh /etc/init.d/$NAME
	chmod +x /etc/init.d/$NAME
	update-rc.d $NAME defaults
}

function install_systemd {
	cp -f $SCRIPT_DIR/$NAME.service /lib/systemd/system/$NAME.service
	systemctl daemon-reload
	systemctl enable $NAME
}

function restart_gatewaybridge {
	echo "Restarting LoRa Gateway Bridge"
	which systemctl &>/dev/null
	if [[ $? -eq 0 ]]; then
		systemctl daemon-reload
		systemctl restart $NAME
	else
		/etc/init.d/$NAME restart || true
	fi	
}

# create gatewaybridge user
id $DAEMON_USER &>/dev/null
if [[ $? -ne 0 ]]; then
	useradd --system -U -M $DAEMON_USER -d /bin/false
fi

mkdir -p "$LOG_DIR"
chown $DAEMON_USER:$DAEMON_GROUP "$LOG_DIR"

# add defaults file, if it doesn't exist
if [[ ! -f /etc/default/$NAME ]]; then
	cp $SCRIPT_DIR/default /etc/default/$NAME
	chown $DAEMON_USER:$DAEMON_GROUP /etc/default/$NAME
	chmod 640 /etc/default/$NAME
	echo -e "\n\n\n"
	echo "-----------------------------------------------------------------------------------------"
	echo "A sample configuration file has been copied to: /etc/default/$NAME"
	echo "After setting the correct values, run the following command to start LoRa Gateway Bridge:"
	echo ""
	which systemctl &>/dev/null
	if [[ $? -eq 0 ]]; then
		echo "$ sudo systemctl start $NAME"
	else
		echo "$ sudo /etc/init.d/$NAME start"
	fi
	echo "-----------------------------------------------------------------------------------------"
	echo -e "\n\n\n"
fi

# add start script
which systemctl &>/dev/null
if [[ $? -eq 0 ]]; then
	install_systemd
else
	install_init
fi

if [[ -n $2 ]]; then
	restart_gatewaybridge
fi
