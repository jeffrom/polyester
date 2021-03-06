#!/bin/sh
set -eu

testdir=/tmp/test/templates

P mkdir $testdir

P plan nice

P template cool $testdir/cool
P template cool $testdir/cool2
P template \
    --data extra.yaml \
    cool $testdir/extracool

# ls -alF /src/testdata/template/plans/nice/secrets/
