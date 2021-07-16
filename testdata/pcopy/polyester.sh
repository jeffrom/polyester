#!/bin/sh
set -eu

P plan cool

testdir=/tmp/test/pcopy

polyester mkdir $testdir/f

polyester pcopy a $testdir/a
polyester pcopy a d $testdir/f
polyester pcopy "{b,c}" $testdir/
polyester pcopy "plans/cool/h" $testdir/h
