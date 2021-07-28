#!/bin/sh
set -eu

P plan cool

testdir=/tmp/test/pcopy

P mkdir $testdir/f

P pcopy a $testdir/a
P pcopy a d $testdir/f
P pcopy "{b,c}" $testdir/
# P pcopy "plans/cool/h" $testdir/h
