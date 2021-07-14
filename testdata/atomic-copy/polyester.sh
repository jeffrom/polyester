#!/bin/sh
set -eu

testdir=/tmp/test/atomic-copy

polyester mkdir $testdir/a $testdir/b
polyester mkdir -m 0700 $testdir/c

polyester touch $testdir/b/bfile

polyester atomic-copy $testdir/a $testdir/d
polyester atomic-copy $testdir/b $testdir/e

polyester atomic-copy $testdir/a $testdir/b $testdir/f
