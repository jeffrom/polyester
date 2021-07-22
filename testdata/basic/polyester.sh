#!/bin/sh
set -eu

# P manifest basic 0.1.0

polyester plan gitty
polyester plan touchy

polyester noop

    # --ignore \.git \
polyester sh \
    --on-change /tmp/tunk/tunk \
    "echo tunk changed!"
