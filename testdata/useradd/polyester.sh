#!/bin/sh
set -eux

polyester useradd --create-home --shell /bin/sh cooluser
