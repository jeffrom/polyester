#!/bin/sh
set -eux

# polyester apt-install git

polyester git-repo \
    https://github.com/jeffrom/tunk.git \
    /tmp/tunk

polyester sh \
    --dir /tmp/tunk \
    --target tunk \
    "make build"
