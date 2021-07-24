#!/bin/sh
set -eu

# 1
testdir=/tmp/test/shell

# 2
P mkdir $testdir

# 3
P sh --dir $testdir --target a "grep 'a' a || { echo a > a; }"
# 4
P sh --dir $testdir --target b 'grep b b || { echo b > b; }'

# 5
P sh --dir $testdir --target c

# 6
grep c c || { echo c > c; }

# 7
echo nice
