#!/bin/sh
set -eu

polyester dependency first-touch

polyester touch --mode 0600 /tmp/hello
