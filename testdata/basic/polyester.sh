#!/bin/sh
set -eux

polyester plan gitty
polyester plan touchy

polyester sh "echo nice shell!"
