FROM ubuntu:17.10

MAINTAINER Patsura Dmitry <talk@dmtry.me>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

ADD . /go/src/github.com/interpals/websocketerd
WORKDIR /go/src/github.com/interpals/websocketerd

RUN apt-get update \
    && apt-get -y upgrade \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        golang-go \
        git \
        curl \
    && mkdir -p /go/bin \
    && curl https://glide.sh/get | sh \
    && glide install \
    && go install github.com/interpals/websocketerd \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

ENTRYPOINT /go/bin/websocketerd

EXPOSE 8484
