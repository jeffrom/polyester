#!/bin/sh
set -eu

testdir=/tmp/test/shell

P mkdir $testdir

P sh --dir $testdir --target a "grep 'a' a || { echo a > a; }"
P sh --dir $testdir --target b 'grep b b || { echo b > b; }'

# P sh --dir $testdir --target c

# grep c c || { echo c > c; }

# P sh-eof
# # EOPSH
