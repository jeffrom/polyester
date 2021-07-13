#!/bin/sh
set -eux

polyester dependency first-touch

polyester touch --mode 0600 /tmp/hello
