#!/usr/bin/env bash

set -e

PACKAGE_NAME="chirpstack-gateway-bridge"
PACKAGE_VERSION=$1
REV="r1"

PACKAGE_URL="https://artifacts.chirpstack.io/downloads/chirpstack-gateway-bridge/chirpstack-gateway-bridge_${PACKAGE_VERSION}_linux_armv5.tar.gz"
DIR=`dirname $0`
FILES_DIR="${DIR}/files"
KEY_DIR="${DIR}/key"
PACKAGE_DIR="${DIR}/package"
TMP_DIR="${DIR}/temp"

# Cleanup
rm -rf $PACKAGE_DIR
rm -rf $TMP_DIR

if [ ! -d $KEY_DIR ]; then
	echo "Key-pair does not yet exist, creating one."
	mkdir -p $KEY_DIR
	openssl genrsa -out $KEY_DIR/private.key 2048
	openssl rsa -pubout -in $KEY_DIR/private.key > $KEY_DIR/public.key
fi

mkdir -p $PACKAGE_DIR
mkdir -p $TMP_DIR/package

# Copy package files
cp -R $FILES_DIR/* $PACKAGE_DIR

# ChirpStack Gateway Bridge binary
mkdir -p $PACKAGE_DIR/opt/$PACKAGE_NAME
wget -P $PACKAGE_DIR/opt/$PACKAGE_NAME $PACKAGE_URL
tar zxf $PACKAGE_DIR/opt/$PACKAGE_NAME/*.tar.gz -C $PACKAGE_DIR/opt/$PACKAGE_NAME
rm $PACKAGE_DIR/opt/$PACKAGE_NAME/*.tar.gz

echo "Tarring"
tar cvfz $TMP_DIR/files.pkg.tar.gz -C $PACKAGE_DIR .

echo "Create MD5 check sum file"
cd $TMP_DIR
md5sum files.pkg.tar.gz > cpkg.hash

echo "Packaging files and cpkg.hash file"
tar cvf files.hashedpkg.tar.gz files.pkg.tar.gz cpkg.hash
cd ..

echo "Signing"
openssl dgst -sha1 -sign $KEY_DIR/private.key -out $TMP_DIR/files.sig $TMP_DIR/files.pkg.tar.gz


cat $TMP_DIR/files.pkg.tar.gz $TMP_DIR/files.sig > $TMP_DIR/package/${PACKAGE_NAME}_${PACKAGE_VERSION}_${REV}.cpkg
cp $KEY_DIR/public.key $TMP_DIR/package/

tar cvfz ${PACKAGE_NAME}_${PACKAGE_VERSION}_${REV}.tar.gz -C $TMP_DIR/package .
