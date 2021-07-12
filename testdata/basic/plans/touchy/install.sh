#!/bin/sh
set -eux

polyester touch --mode 0600 /tmp/hello

polyester git-repo \
    https://github.com/jeffrom/tunk.git \
    /tmp/tunk
