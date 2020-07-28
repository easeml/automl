#!/bin/sh

echo "Updating"
apt update
echo "Preparing Build Environment"
echo "Installing Make"
apt install -y make
echo "Installing git"
apt install -y git
apt install -y wget
apt install -y curl
echo "Getting NodeJS"
export VERSION='12.8.0'
export NBASE="node-v$VERSION-linux-x64.tar.gz"
export GBASE="go1.14.4.linux-amd64.tar.gz"
#export TOOL_INSTALL_PATH=$HOME/temp_install
export TOOL_INSTALL_PATH=/usr/local
mkdir -p $TOOL_INSTALL_PATH
wget https://nodejs.org/dist/v$VERSION/$NBASE
tar -C $TOOL_INSTALL_PATH/ -xzf $NBASE --strip-components=1
echo "Installing go"
curl -O https://storage.googleapis.com/golang/$GBASE
tar -C $TOOL_INSTALL_PATH/ -xzf $GBASE
mkdir -p ~/go
echo "Cleaning"
rm $NBASE
rm $GBASE

