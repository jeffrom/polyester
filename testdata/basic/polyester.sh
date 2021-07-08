#!/bin/sh
set -eux

ptouch --mode 0600 /tmp/hello
puseradd --create-home --shell /bin/sh appuser
