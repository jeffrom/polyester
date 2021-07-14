#!/bin/sh
set -eux


testdir=/tmp/test/pcopy

polyester mkdir $testdir/f

polyester pcopy files/a $testdir/a
polyester pcopy files/a files/d $testdir/f
polyester pcopy "files/{b,c}" $testdir/
