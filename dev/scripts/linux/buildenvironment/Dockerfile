FROM snapcore/snapcraft

USER root

RUN  apt update \
 &&  apt install -y software-properties-common \
 &&  apt-add-repository -y ppa:git-core/ppa 

RUN  apt update \
 &&  apt install -y make \
 &&  apt install -y python \
 &&  apt install -y python-pip \
 &&  apt install -y curl \
 &&  apt install -y git \
 &&  apt install -y socat \
 &&  apt install -y libssl-dev zlib1g-dev libbz2-dev libreadline-dev libsqlite3-dev \
 &&  python -m pip install --upgrade pip \
 &&  python -m pip install pipenv

ENV PYENV_ROOT=/usr/local/pyenv
RUN curl https://pyenv.run | bash
RUN $PYENV_ROOT/bin/pyenv install 3.6.10

RUN echo "\neaseml ALL=(ALL) NOPASSWD: ALL\n" | tee -a /etc/sudoers

COPY prepare-temp-build-env-engine.sh .
RUN ./prepare-temp-build-env-engine.sh

RUN useradd --create-home easeml

USER easeml

WORKDIR /home/easeml
ENV EASEML_HOME=/home/easeml

RUN echo "export GOROOT=/usr/local/go" > $EASEML_HOME/.env.base
RUN echo "export GOPATH=\$HOME/go" >> $EASEML_HOME/.env.base
RUN echo "export PATH=$PATH:/usr/local/go/bin:\$PYENV_ROOT/bin:\$HOME/go/bin" >> $EASEML_HOME/.env.base
RUN echo "source $EASEML_HOME/.env.base" >> $EASEML_HOME/.bashrc
RUN echo "//registry.npmjs.org/:_authToken=${NODE_AUTH_TOKEN}" > ~/.npmrc

ENV GOROOT "/usr/local/go"
ENV GOPATH "$EASEML_HOME/go"
ENV PATH "$PATH:/usr/local/go/bin:$PYENV_ROOT/bin:$EASEML_HOME/go/bin"

RUN mkdir -p $GOPATH
RUN find $EASEML_HOME \( -type d -exec sudo chmod -v u+rwx,g+rwx,o+rwx {} \; -o -executable -type f -exec sudo chmod -v u+rwx,g+rwx,o+rwx {} \; -o ! -executable -type f -exec sudo chmod -v u+rw,g+rw,o+rw {} \; \)
RUN find /usr/local \( -type d -exec sudo chmod -v u+rwx,g+rwx,o+rwx {} \; -o -executable -type f -exec sudo chmod -v u+rwx,g+rwx,o+rwx {} \; -o ! -executable -type f -exec sudo chmod -v u+rw,g+rw,o+rw {} \; \)

ENTRYPOINT /bin/bash
