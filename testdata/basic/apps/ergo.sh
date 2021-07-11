#!/bin/sh
set -eux

poly pkg_required git

poly touch --mode 0600 /tmp/hello
poly useradd --create-home --shell /bin/sh appuser

poly git_repo ~appuser/repos/ergo \
    --upstream stable --ref stable \
    https://github.com/ergochat/ergo.git

# depends on the last repo that was declared. If there are multiple builds to
# do in one file, do them one after the other.
poly make --dir ~appuser/repos/ergo
poly make --dir ~appuser/repos/ergo install

poly pcopy ergo.yaml /etc/ergo/

poly systemd_unit ergo.service \
    --workdir ~appuser/apps/ergo \
    --service-template systemd/unit.service
    # --exec-start-template systemd/unit-start

# poly shell

mkdir -p /tmp/sup
printf "sup" > /tmp/sup/ok

# poly shell-end
