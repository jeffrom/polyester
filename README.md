# polyester

Bootstrap servers, apps, and secrets. Run applications with secrets mapped to environment variables. Template configuration files and scripts. Continuously deploy. Made for version control.

## status

Pretty unstable!

## usage

### plans

Since we're an ops tool, we need a magic directory structure:

```
.
├── files
├── plans
│   ├── myapp
│   │   ├── files
│   │   └── templates
│   └── myapp2
├── templates
│   └── systemd
└── vars
```

### agent

run a collection of plans (default usage):

```bash
$ make && ./polyester apply testdata/basic
```

run a single plan in a filesystem sandbox, w/ an overridden state directory:

```bash
$ make && ./polyester apply --dir-root /tmp/polytest --state-dir /tmp/polystate testdata/basic/plans/touchy/install.sh
```

Normal usage entails running `polyester apply` on remote servers. To continuously update a git repository containing the polyester manifest, a cron could periodically run a script such as:

```
$ cd ~/repos/cluster && git pull && polyester apply
```

## how does it work

The key concepts are "plans" and "operators". Plans are sequences of operations, used to execute commands on an environment. The primary domain language is POSIX shell (though others could be supported without a huge amount of effort). Shell scripts are evaluated to generate the execution plan by outputting it to an intermediate format in the local filesystem.

Operators idempotently execute operations and track state. In many cases they extend common linux tools. Some example operators:

- useradd
- touch
- copy

See the `testdata/` directory for example plans.

## development

build it:

```
$ make
```

run docker tests:

```
$ go test ./cmd/polyester -run TestDocker
```

Operators can be partially "sandboxed" using the --dir-root flag, which can be useful for debugging. For example, to override the state directory and use a filesystem sandbox:

```
$ ./polyester apply --dir-root /tmp/mysandbox --state-dir /tmp/mystate testdata/basic
```

## todo

* secret management (maybe w/ age and / or sops)
* templating (maybe w/ gomplate)
* validate operator calls in shell scripts pre-execution
* systemd operators to reenable, reinstall, restart units on state changes (maybe shell is enough).

## errata

The dsl can be posix shell, just inject a bunch of functions that write the plan to an intermediate format. Can separate out the statements that are real shell statements and covert them into "shell script" operations.

maybe can use this to parse out the normal shell from the special dsl shell: https://github.com/mvdan/sh
