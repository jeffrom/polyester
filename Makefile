.POSIX:
.SUFFIXES:
.SUFFIXES: .1 .5 .7 .1.scd .5.scd .7.scd

override undefine VERSION # don't allow local overrides, we want our version
_git_version=$(shell git describe --long --tags --dirty 2>/dev/null | sed 's/-/.r/;s/-/./')
ifeq ($(strip $(_git_version)),)
VERSION=0.1.0
else
VERSION=$(_git_version)
endif

VPATH=doc
PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin
SHAREDIR?=$(PREFIX)/share/polyester
MANDIR?=$(PREFIX)/share/man
GO?=go
GOFLAGS?=

# end git.sr.git/sircmpwn/aerc stealing, header portion at least :)

TMPDIR := $(if $(TMPDIR),$(TMPDIR),"/tmp/")
GOPATH := $(shell go env GOPATH)

bin := polyester
gofiles := $(wildcard go.mod go.sum *.go **/*.go **/**/*.go **/**/**/*.go)

gocoverutil := $(GOPATH)/bin/gocoverutil
staticcheck := $(GOPATH)/bin/staticcheck
gomodoutdated := $(GOPATH)/bin/go-mod-outdated

all: build doc

build: $(bin) go.mod go.sum

$(bin): $(gofiles)
	$(GO) build $(GOFLAGS) \
		-ldflags "-X main.Prefix=$(PREFIX) \
		-X main.Version=$(VERSION)" \
		-o $@ \
		./cmd/polyester

DOCS := \
	polyester.1 \
	# polyester-config.5 \
	# polyester-ci.7

.1.scd.1:
	scdoc < $< > $@

.5.scd.5:
	scdoc < $< > $@

.7.scd.7:
	scdoc < $< > $@

doc: $(DOCS)

# Exists in GNUMake but not in NetBSD make and others.
RM?=rm -f

.PHONY: clean
clean:
	$(RM) -r $(TMPDIR)/polyester*
	$(RM) $(DOCS) polyester

.PHONY: install
install: all
	mkdir -m755 -p $(DESTDIR)$(BINDIR) $(DESTDIR)$(MANDIR)/man1 $(DESTDIR)$(MANDIR)/man5 $(DESTDIR)$(MANDIR)/man7
	install -m755 $(bin) $(DESTDIR)$(BINDIR)/polyester
	install -m644 polyester.1 $(DESTDIR)$(MANDIR)/man1/polyester.1
	# install -m644 polyester-ci.7 $(DESTDIR)$(MANDIR)/man7/polyester-ci.7
	# install -m644 polyester-config.5 $(DESTDIR)$(MANDIR)/man5/polyester-config.5

RMDIR_IF_EMPTY:=sh -c '\
if test -d $$0 && ! ls -1qA $$0 | grep -q . ; then \
	rmdir $$0; \
fi'

.PHONY: uninstall
uninstall:
	$(RM) $(DESTDIR)$(BINDIR)/polyester
	$(RM) $(DESTDIR)$(MANDIR)/man1/polyester.1
	# $(RM) $(DESTDIR)$(MANDIR)/man7/polyester-ci.7
	# $(RM) $(DESTDIR)$(MANDIR)/man5/polyester-config.5
	$(RM) -r $(DESTDIR)$(SHAREDIR)
	${RMDIR_IF_EMPTY} $(DESTDIR)$(BINDIR)
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man1
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man5
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man7
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)

.PHONY: ci
ci: build doc test.cover test.lint

.PHONY: test
test: build
	GO111MODULE=on TESTBIN=$$(pwd)/polyester go test -short -cover -count 1 ./...

.PHONY: test.race
test.race: build
	GO111MODULE=on TESTBIN=$$(pwd)/polyester go test -race ./...

.PHONY: test.lint
test.lint: $(staticcheck)
	GO111MODULE=on $(staticcheck) -checks all ./...
	go vet ./...
	semgrep --error -c r/dgryski.semgrep-go -c p/gosec -c p/golang

.PHONY: test.cover
test.cover: SHELL:=/bin/bash
test.cover: $(gocoverutil)
	set -eo pipefail; TESTBIN=$$(pwd)/polyester $(gocoverutil) -coverprofile=cov.out test -covermode=count ./... \
		2> >(grep -v "no packages being tested depend on matches for pattern" 1>&2) \
		| sed -e 's/of statements in .*/of statements/'
	@echo -n "total: "; go tool cover -func=cov.out | tail -n 1 | sed -e 's/\((statements)\|total:\)//g' | tr -s "[:space:]"

.PHONY: test.outdated
test.outdated: $(gomodoutdated)
	GO111MODULE=on go list -u -m -json all | go-mod-outdated -direct

.PHONY: release.dryrun
release.dryrun:
	goreleaser --snapshot --skip-publish --rm-dist

.PHONY: release
release:
	goreleaser --rm-dist

$(gocoverutil):
	GO111MODULE=off go get github.com/AlekSi/gocoverutil

$(staticcheck):
	cd $(TMPDIR) && GO111MODULE=on go get honnef.co/go/tools/cmd/staticcheck@2019.2.3

$(gomodoutdated):
	GO111MODULE=off go get github.com/psampaz/go-mod-outdated
