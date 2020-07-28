#!/bin/sh
export NODE_VERSION='12.8.0'
export NBASE="node-v$NODE_VERSION-linux-x64.tar.gz"
export GBASE="go1.14.4.linux-amd64.tar.gz"
export TOOL_INSTALL_PATH=/usr/local
export PATH=$PATH:$TOOL_INSTALL_PATH/bin
echo "Updating"
apt update
echo "Preparing Build Environment"
mkdir -p $TOOL_INSTALL_PATH
mkdir -p ~/go

if ! type make > /dev/null; then
    echo "Installing Make"
    apt install -y make
fi

if ! type git > /dev/null; then
    echo "Installing git"
    apt install -y git
fi

if ! type wget > /dev/null; then
    echo "Installing wget"
    apt install -y wget
fi

if ! type curl > /dev/null; then
    echo "Installing curl"
    apt install -y curl
fi

if ! type node > /dev/null; then  
    echo "Getting NodeJS"
    wget https://nodejs.org/dist/v$NODE_VERSION/$NBASE
    tar -C $TOOL_INSTALL_PATH/ -xzf $NBASE --strip-components=1
    rm $NBASE
fi

if ! type go > /dev/null; then
    echo "Installing go"
    curl -O https://storage.googleapis.com/golang/$GBASE
    tar -C $TOOL_INSTALL_PATH/ -xzf $GBASE
    rm $GBASE
fi

