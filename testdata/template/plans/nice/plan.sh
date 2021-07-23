#!/bin/sh
set -eu

testdir=/tmp/test/templates

P mkdir $testdir

P template cool $testdir/nice-cool
