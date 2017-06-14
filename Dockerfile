FROM golang:1.8.3

MAINTAINER Patsura Dmitry <talk@dmtry.me>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN mkdir -p /etc/confd/{conf.d,templates}
RUN mkdir -p /etc/interpals

COPY conf.d /etc/confd/conf.d
COPY templates /etc/confd/templates

ADD https://github.com/kelseyhightower/confd/releases/download/v0.11.0/confd-0.11.0-linux-amd64 /usr/local/bin/confd
RUN chmod +x /usr/local/bin/confd

ADD . /go/src/github.com/interpals/websocketerd
WORKDIR /go/src/github.com/interpals/websocketerd

RUN apt-get update \
    && apt-get -y upgrade \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        git \
        curl \
    && mkdir -p /go/bin \
    && curl https://glide.sh/get | sh \
    && glide install \
    && go install github.com/interpals/websocketerd \
    && apt-get remove -y git curl \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

ENTRYPOINT /bin/bash start.sh

EXPOSE 8484
