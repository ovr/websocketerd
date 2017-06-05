#!/bin/bash

set -e

cd /go/src/github.com/interpals/websocketerd

/usr/local/bin/confd -onetime -backend env -config-file /etc/confd/conf.d/websocketerd.toml

cat /etc/interpals/websocketerd.json

/go/bin/websocketerd --config=/etc/interpals/websocketerd.json
