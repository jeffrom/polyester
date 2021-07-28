#!/bin/sh
set -eu

testdir=/tmp/test/pcopy

P mkdir $testdir/f

P pcopy g $testdir/g
