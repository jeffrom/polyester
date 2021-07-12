#!/bin/sh
set -eux

polyester pkg_required git

polyester touch --mode 0600 /tmp/hello
polyester useradd --create-home --shell /bin/sh appuser

polyester git-repo \
    --upstream stable --ref stable \
    https://github.com/ergochat/ergo.git \
    ~appuser/repos/ergo

# depends on the last repo that was declared. If there are multiple builds to
# do in one file, do them one after the other.
polyester make --dir ~appuser/repos/ergo
polyester make --dir ~appuser/repos/ergo install

polyester pcopy ergo.yaml /etc/ergo/

polyester systemd-unit ergo.service \
    --workdir ~appuser/apps/ergo \
    --service-template systemd/unit.service
    # --exec-start-template systemd/unit-start

# polyester shell

mkdir -p /tmp/sup
printf "sup" > /tmp/sup/ok

# polyester shell-end
