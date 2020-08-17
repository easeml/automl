FROM ubuntu:18.04

RUN useradd --create-home easeml

WORKDIR /home/easeml
ENV PATH "$PATH:/home/easeml/go/bin"
RUN apt-get update 
RUN  apt-get install sudo -y \
 &&  apt install -y git \
 &&  apt install -y git-lfs
 
RUN echo "\neaseml ALL=(ALL) NOPASSWD: ALL\n" | tee -a /etc/sudoers

USER easeml

COPY easeml go/bin/

ENTRYPOINT ["/home/easeml/go/bin/easeml"]
