#!/bin/sh
set -eu

# tmpdir=$(mktemp -d)
# trap 'rm -rf "$tmpdir"; trap - EXIT; exit' EXIT INT HUP TERM

tmpdir=/tmp/testrepo
mkdir -p $tmpdir

(
cd "$tmpdir"
if [ ! -e repo.git ]; then
    mkdir -p repo.git
    cd repo.git
    git init --bare

    cd "$tmpdir"
    git clone "$tmpdir/repo.git" repo
    cd repo
    echo "a" > a
    echo "b" > b
    echo "c" > c
    git add .
    git commit -am "initial commit"
    git push origin master
fi
)

polyester git-repo \
    "$tmpdir/repo.git" \
    /tmp/tunk

polyester sh \
    --dir /tmp/tunk \
    --target tunk \
    "grep '^echo tunk$' tunk || { echo 'echo tunk' > tunk && chmod 755 tunk; }"

polyester atomic-copy \
    --exclude "/**/.git" \
    --exclude "/**/.git/**" \
    /tmp/tunk /tmp/tunk2
