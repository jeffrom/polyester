#!/bin/sh
set -eu

testdir=/tmp/test/mkdir
P mkdir $testdir/a $testdir/b
P mkdir -m 0700 $testdir/c
