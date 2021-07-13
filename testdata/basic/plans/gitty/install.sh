#!/bin/sh
set -eux

polyester git-repo \
    https://github.com/jeffrom/tunk.git \
    /tmp/tunk
