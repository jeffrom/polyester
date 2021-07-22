#!/bin/sh
set -eu

testdir=/tmp/test/pcopy

polyester mkdir $testdir/f

P pcopy g $testdir/g
