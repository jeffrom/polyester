#!/bin/sh
set -eu

testdir=/tmp/test/templates

P template cool $testdir/
# P template cool $testdir/cool2
# P template --data extra.yaml cool $testdir/extracool
