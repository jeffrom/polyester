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

The key concepts are "plans" and "operators". Plans are sequences of operations, used to execute commands on an environment. Operations are run in order, by plan (plans will probably run concurrently in the future). There is a caching strategy similar to the docker layer cache, where, if an operation's state changes, it and every subsequent operation is executed.

The primary domain language is POSIX shell (though others could be supported without a huge amount of effort). Shell scripts are evaluated to generate the execution plan by outputting it to an intermediate format in the local filesystem. This means variable scope and other behavior may not be what you expect because the script doesn't immediately execute, but rather constructs an intermediate plan.

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

* operators
  - secret management (age, sops, ?)
  - templating (maybe w/ gomplate)
  - systemd operators to reenable, reinstall, restart units on state changes (maybe shell is enough).
* shell plan script improvements
  - validate operator calls in shell scripts pre-execution
  - handling variables / scope in shell script plans
  - maybe a special annotation to embed sh operator scripts directly in plan files (currently you pass a string, ie `polyester sh "echo hi"`).
* planner improvements
  - concurrent plan execution, accounting for dependencies
  - improve dryrun contract -- ie planner passes --dry-run to commands that have it otherwise execution is skipped.
  - use remote, versioned git repo / tar / exported plans
* output formats
  - nice readable apply summaries, what changed etc
  - json log for debugging, introspection, integration testing
  - operations interface (maybe just stringer) to control its argument formatting
* testing
  - parameterize base docker image to run against various distros & oses
  - generic table test for operation idempotency
  - replace github repo in docker basic test w/ a proper test fixture repo on the local fs
* docs
  - write some
  - templatized operation docs using go generate / gomplate / operation.Info
  - example repo / article
