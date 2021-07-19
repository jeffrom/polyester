#!/bin/sh
set -eu

testdir=/tmp/test/templates

P mkdir $testdir

P template cool $testdir/cool
P template cool $testdir/cool2
P template \
    --data extra.yaml \
    cool $testdir/extracool
