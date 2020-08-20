# Containerized easeml server

## Run the container 

Clear files from previous runs from the host
```bash
rm -rf ${HOME}/.easeml/*
```

Create db folder
```bash
mkdir -p ${HOME}/.easeml/shared/db
```

Create network
```bash
docker network create --label easeml_local local_easeml_network
```
Run mongo container
```bash
docker run \
    --name mongo3423 \
    --label easeml_local \
    -v ${HOME}/.easeml/shared/db:/home/mongo/db  \
    --network local_easeml_network \
    --network-alias mongodb \
    --user $(id -u):$(id -g) \
    -p 27018:27017  \
    --rm \
mongo:3.4.23 --dbpath /home/mongo/db
```
Run easeml container
```bash
docker run \
     --label easeml_local \
     -v /var/run/docker.sock:/var/run/docker.sock \
     -v $(which docker):/usr/bin/docker:ro \
     -v /etc/passwd:/etc/passwd:ro \
     -v ${HOME}/.easeml:${HOME}/.easeml \
     --user $(id -u):$(getent group docker | cut -d: -f3) \
     --network local_easeml_network \
     --network-alias easemlserver \
     --rm \
     -p 8080:8080 \
     -t -i \
easeml/server start --database-address mongodb:27017 --server-address :8080 --login
```

# Run Client for Demo

## Get demo data and move to the directory
```bash
git clone https://renkulab.io/gitlab/leonel.aguilar.m/easeml_renku_demo.git
```

```bash
cd easeml_renku_demo/
```

## Run client container
```bash
docker run \
     --label easeml_local \
     -v ${HOME}/.easeml:/home/jovyan/.easeml \
     -v ${PWD}:/home/jovyan/demo/ \
     -w /home/jovyan/demo/ \
     --user $(id -u):$(id -g)\
     --network local_easeml_network \
     --network-alias easemlclient \
     --rm \
     -p 8888:8888 \
     -t -i \
     --entrypoint /bin/bash \
registry.renkulab.io/leonel.aguilar.m/easeml_renku_demo:aa71000
```

## Add Host address
```bash
echo "host: http://easemlserver:8080" >> ${HOME}/.easeml/config.yaml
```
## Add jupyterlab configuration, to access the webui from the host (http://localhost:8888 when starting jupyter lab from the container)
```bash
mkdir -p ~/.jupyter/lab/user-settings/@easeml/jupyterlab_easeml
echo '{ "easemlConfig": {"easemlServer": "http://localhost:8080"}}' > ~/.jupyter/lab/user-settings/@easeml/jupyterlab_easeml/plugin.jupyterlab-settings
```

## Now you can start the Demo
see README.md from https://renkulab.io/gitlab/leonel.aguilar.m/easeml_renku_demo.git

# Remove the network when finished
```bash
docker network rm local_easeml_network
```
# Build Container instead of using Dockerhub's version

Make sure that the easeml binary is at ~/go/bin/easeml
```bash
cd ../../../engine/ && make install && cd -
```

Create the container
```bash
docker build -t easeml/server -f Dockerfile ~/go/bin
```



