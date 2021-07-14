#!/bin/sh
set -eu

testdir=/tmp/test/copy

P mkdir $testdir/a $testdir/b
P mkdir -m 0700 $testdir/c

P touch $testdir/b/bfile

P copy $testdir/a $testdir/d
P copy $testdir/b $testdir/e

P copy $testdir/a $testdir/b $testdir/f
