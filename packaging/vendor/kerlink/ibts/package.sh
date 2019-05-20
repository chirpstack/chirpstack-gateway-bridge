#!/bin/env bash

PACKAGE_NAME="lora-gateway-bridge"
PACKAGE_VERSION="3.0.0"
REV="r1"


PACKAGE_URL="https://artifacts.loraserver.io/downloads/lora-gateway-bridge/lora-gateway-bridge_${PACKAGE_VERSION}_linux_armv5.tar.gz"
DIR=`dirname $0`
PACKAGE_DIR="${DIR}/package"
PACKAGE_USER_DIR="${PACKAGE_DIR}/user/${PACKAGE_NAME}"
PACKAGE_MONIT_DIR="${PACKAGE_DIR}/user/rootfs_rw/etc/monit.d"

# Cleanup
rm -rf $PACKAGE_DIR

# CONTROL
mkdir -p $PACKAGE_DIR/CONTROL
cat > $PACKAGE_DIR/CONTROL/control << EOF
Package: $PACKAGE_NAME
Version: $PACKAGE_VERSION-$REV
Architecture: klk_lpbs
Maintainer: Orne Brocaar <info@brocaar.com>
Priority: optional
Section: network
Source: N/A
Description: LoRa Gateway Bridge
EOF

cat > $PACKAGE_DIR/CONTROL/conffiles << EOF
/user/lora-gateway-bridge/lora-gateway-bridge.toml
EOF

# Files
mkdir -p $PACKAGE_MONIT_DIR
mkdir -p $PACKAGE_USER_DIR

cp files/$PACKAGE_NAME.toml $PACKAGE_USER_DIR
cp files/$PACKAGE_NAME.monit $PACKAGE_MONIT_DIR/lora-gateway-bridge
cp files/start.sh $PACKAGE_USER_DIR
wget -P $PACKAGE_USER_DIR $PACKAGE_URL
tar zxf $PACKAGE_USER_DIR/*.tar.gz -C $PACKAGE_USER_DIR
rm $PACKAGE_USER_DIR/*.tar.gz

# Package
opkg-build -o root -g root $PACKAGE_DIR

# Cleanup
rm -rf $PACKAGE_DIR
