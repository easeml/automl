# Ease.ml - A Scalable Auto-ML System

![Logo](docs/img/logo-big.png?raw=true "Logo")

Ease.ml is a declarative machine learning service platform. It enables users to upload their datasets and start model selection and tuning jobs. Given the schema of the dataset, ease.ml does an automatic search for applicable models and performs training, prediction and evaluation. All models are stored as Docker images which allows greater portability and reproducibility. TEST

For more details, check out out recent publications:

T Li, J Zhong, J Liu, W Wu, C Zhang. Ease.ml: Towards Multi-tenant Resource Sharing for Machine Learning Workloads. VLDB 2018. [[PDF]](http://www.vldb.org/pvldb/vol11/p607-li.pdf)

Bojan Karlas, Ji Liu, Wentao Wu, Ce Zhang. Ease.ml in Action: Towards Multi-tenant Declarative Learning Services. VLDB (Demo) 2018. [[PDF]](http://www.vldb.org/pvldb/vol11/p2054-karlas.pdf)

The project is being developed by the [DS3 Lab](https://ds3lab.org/) at ETH Zurich and is still in its early stages. Stay tuned for more updates.

## Build ease.ml from source (Linux)

## Prerequisites
- go and packr2
- nodejs and npm
- Mongo DB
- Docker

### Install go

```bash
sudo snap install --classic go
```

Make sure the go binary directory is in PATH (add this to the  `~/.profile` file):

```bash
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
```

Install packr2 which we will be using to bundle the web UI files into the go binary:

```bash
go get -v -u github.com/gobuffalo/packr/v2/...
```

### Install node and npm

```bash
curl -sL https://deb.nodesource.com/setup_10.x | sudo -E bash -
sudo apt-get install -y nodejs
sudo apt-get install -y npm
```

### Install Mongo DB

See [this page](https://docs.mongodb.com/manual/tutorial/install-mongodb-on-ubuntu/#run-mongodb-community-edition).

```bash
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 9DA31620334BD75D9DCB49F368818C72E52529D4

echo "deb [ arch=amd64 ] https://repo.mongodb.org/apt/ubuntu trusty/mongodb-org/4.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-4.0.list

sudo apt-get update

sudo apt-get install -y mongodb-org
```

If you would like to set up a different database directory

```bash
mongod --dbpath <your_new_db_path>
```

### Install Docker

See [this page](https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-on-ubuntu-18-04).

```bash
sudo apt install apt-transport-https ca-certificates curl software-properties-common

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu bionic stable"

sudo apt update

sudo apt install docker-ce
```

To enable execution of docker commands without sudo, add the current user to the `docker` group:

```bash
sudo usermod -aG docker ${USER}

su - ${USER}
```


## Ease.ml

### Get the source code

```bash
go get -v github.com/ds3lab/easeml
```

### Build and Install binary from source

```bash
cd $GOPATH/src/github.com/ds3lab/easeml/engine

#Install in GOPATH/bin
make install
```
\# Alternative:
```bash
# Install in ALT_PATH
# ALT_PATH should be added to PATH
make install INSTALL_PATH=ALT_PATH
```

<!---
### Initialize the web directory and build the web UI

```bash
cd $GOPATH/src/github.com/ds3lab/easeml/web
npm install
npm run build
```

### Build and install Ease.ml

```bash
cd $GOPATH/src/github.com/ds3lab/easeml/engine

packr2 -v

go install
```
-->

### Run ease.ml

```bash
easeml start --browser
```
