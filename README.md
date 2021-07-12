# polyester

Bootstrap servers, apps, and secrets. Run applications with secrets mapped to environment variables. Template configuration files and scripts. Continuously deploy. Made for version control.

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

## stuff

* age / saltpack ?

The dsl can be posix shell, just inject a bunch of functions that write the plan to an intermediate format. Can separate out the statements that are real shell statements and covert them into "shell script" operations.

maybe can use this to parse out the normal shell from the special dsl shell: https://github.com/mvdan/sh
