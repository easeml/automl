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
export NBASE="node-v$VERSION-linux-x64"
wget https://nodejs.org/dist/v$VERSION/$NBASE.tar.gz
tar xzf $NBASE.tar.gz -C .
mkdir -p $HOME/temp_install
mv $NBASE/* $HOME/temp_install
echo "Cleaning"
rm $NBASE.tar.gz
echo "Installing go"
curl -O https://storage.googleapis.com/golang/go1.14.4.linux-amd64.tar.gz
sudo tar -C $HOME/temp_install/ -xzf go1.14.4.linux-amd64.tar.gz
mkdir -p ~/go
export GOROOT=$HOME/temp_install/go
export GOPATH=$HOME/go
export PATH=$PATH:$HOME/go/bin:$HOME/temp_install/go/bin
echo "Getting Packr2"
go get -u github.com/gobuffalo/packr/v2/...
