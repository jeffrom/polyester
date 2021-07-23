#!/bin/sh
set -e

testdir=/tmp/test/repo

if [ -z "$REPO_URL" ]; then
    echo "skipping as \$REPO_URL isn't set"
    exit 0
fi
set -u

P git-repo \
    "$REPO_URL" \
    "$testdir/a"
