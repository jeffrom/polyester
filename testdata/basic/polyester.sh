#!/bin/sh
set -eux

polyester plan gitty
polyester plan touchy

polyester noop

    # --ignore \.git \
polyester sh \
    --on-change /tmp/tunk/tunk \
    "echo tunk changed!"
