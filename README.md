# Ease.ml - A Scalable Auto-ML System

![Logo](doc/img/logo-big.png?raw=true "Logo")

Ease.ml is a declarative machine learning service platform. It enables users to upload their datasets and start model selection and tuning jobs. Given the schema of the dataset, ease.ml does an automatic search for applicable models and performs training, prediction and evaluation. All models are stored as Docker images which allows greater portability and reproducibility.

For more details, check out out recent publications:

T Li, J Zhong, J Liu, W Wu, C Zhang. Ease.ml: Towards Multi-tenant Resource Sharing for Machine Learning Workloads. VLDB 2018. [[PDF]](http://www.vldb.org/pvldb/vol11/p607-li.pdf)

Bojan Karlas, Ji Liu, Wentao Wu, Ce Zhang. Ease.ml in Action: Towards Multi-tenant Declarative Learning Services. VLDB (Demo) 2018. [[PDF]](http://www.vldb.org/pvldb/vol11/p2054-karlas.pdf)

The project is being developed by the [DS3 Lab](https://ds3lab.org/) at ETH Zurich and is still in its early stages. Stay tuned for more updates.

## Build ease.ml from source (Linux)

### Install go

```bash
sudo snap install --classic go
```

Make sure the go binary directory is in PATH (add this to the  `~/.profile` file):

```bash
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
```

Install packr which we will be using to bundle the web UI files into the go binary:

```bash
go get -u github.com/gobuffalo/packr/...
```

### Install node and npm

```bash
curl -sL https://deb.nodesource.com/setup_10.x | sudo -E bash -
sudo apt-get install -y nodejs
sudo apt-get install -y npm
```

### Get the code

```bash
go get github.com/ds3lab/easeml
```

### Initialize the web directory and build the web UI

```bash
cd $GOPATH/src/github.com/ds3lab/easeml/web
npm install
npm run build
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

### Build ease.ml

```bash
packr install github.com/ds3lab/easeml
```

### Run ease.ml

```bash
easeml start --browser
```
