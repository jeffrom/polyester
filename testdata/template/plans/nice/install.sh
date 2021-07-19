#!/bin/sh
set -ex

testdir=/tmp/test/templates

P mkdir $testdir

P template cool $testdir/nice-cool
