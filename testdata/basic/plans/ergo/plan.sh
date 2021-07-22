#!/bin/sh
set -eu

P noop

# polyester apt-install git

# polyester useradd --create-home --shell /bin/sh appuser

# polyester git-repo \
#     --ref stable \
#     https://github.com/ergochat/ergo.git \
#     ~appuser/repos/ergo

# depends on the last repo that was declared. If there are multiple builds to
# do in one file, do them one after the other.
# polyester make --dir ~appuser/repos/ergo
# polyester make --dir ~appuser/repos/ergo install

# polyester copy ergo.yaml /etc/ergo/

# polyester systemd-unit ergo.service \
#     --workdir ~appuser/apps/ergo \
#     --template systemd/unit.service
    # --exec-start-template systemd/unit-start

# polyester shell

# mkdir -p /tmp/sup
# printf "sup" > /tmp/sup/ok

# polyester shell-end
