#!/bin/sh
set -eux

testdir=/tmp/test/copy

polyester mkdir $testdir/a $testdir/b
polyester mkdir -m 0700 $testdir/c

polyester touch $testdir/b/bfile

polyester copy $testdir/a $testdir/d
polyester copy $testdir/b $testdir/e

polyester copy $testdir/a $testdir/b $testdir/f
