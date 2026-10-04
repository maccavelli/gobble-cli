# gobble-cli toolchain (0004-PLAN phase 0).
# Target names follow magic-cli-remote. Gates that need later phases
# (archtest, apidiff, verify-build-metadata) are present and no-op until
# the tree those phases add exists.
MODULE := github.com/maccavelli/gobble-cli
SHELL := C:/PROGRA~1/Git/usr/bin/bash.exe
.SHELLFLAGS := -eu -o pipefail -c
# Git bash does not put its coreutils on PATH when make starts it from
# PowerShell. Recipes need grep, diff, cp, and mv.
export PATH := C:/PROGRA~1/Git/usr/bin:$(PATH)

CGO_ENABLED ?= 0

UNAME_S := $(shell uname -s 2>/dev/null || echo unknown)
UNAME_M := $(shell uname -m 2>/dev/null || echo unknown)

ifeq ($(UNAME_S),Linux)
  HOST_GOOS := linux
  GOOS ?= linux
  USER_BIN_DIR ?= $(HOME)/.local/bin
else ifeq ($(UNAME_S),Darwin)
  HOST_GOOS := darwin
  GOOS ?= darwin
  USER_BIN_DIR ?= $(HOME)/.local/bin
else ifneq (,$(findstring MINGW,$(UNAME_S)))
  HOST_GOOS := windows
  GOOS ?= windows
  USER_BIN_DIR ?= $(HOME)/.local/bin
else ifneq (,$(findstring MSYS,$(UNAME_S)))
  HOST_GOOS := windows
  GOOS ?= windows
  USER_BIN_DIR ?= $(HOME)/.local/bin
else ifneq (,$(findstring CYGWIN,$(UNAME_S)))
  HOST_GOOS := windows
  GOOS ?= windows
  USER_BIN_DIR ?= $(HOME)/.local/bin
else
  HOST_GOOS := $(shell go env GOOS 2>/dev/null || echo linux)
  GOOS ?= $(shell go env GOOS 2>/dev/null || echo linux)
  USER_BIN_DIR ?= $(HOME)/.local/bin
endif

ifeq ($(UNAME_M),x86_64)
  HOST_GOARCH := amd64
  GOARCH ?= amd64
else ifeq ($(UNAME_M),amd64)
  HOST_GOARCH := amd64
  GOARCH ?= amd64
else ifeq ($(UNAME_M),aarch64)
  HOST_GOARCH := arm64
  GOARCH ?= arm64
else ifeq ($(UNAME_M),arm64)
  HOST_GOARCH := arm64
  GOARCH ?= arm64
else
  HOST_GOARCH := $(shell go env GOARCH 2>/dev/null || echo amd64)
  GOARCH ?= $(shell go env GOARCH 2>/dev/null || echo amd64)
endif

ifeq ($(GOOS),windows)
  BIN_EXT := .exe
else
  BIN_EXT :=
endif

BIN := bin/gobble$(BIN_EXT)

.PHONY: build install test race vet fmt lint staticcheck vulncheck tidy clean \
	pre-add-check preflight check-cgo-off verify-build-metadata fix-check archtest apidiff

# Shipped binaries are pure Go. check-cgo-off refuses CGO_ENABLED other than 0
# and any `import "C"` in a .go file (0004-PLAN phase 0 accept).
build: check-cgo-off
	CGO_ENABLED=$(CGO_ENABLED) go build ./...

install: check-cgo-off
	@mkdir -p "$(USER_BIN_DIR)"
	CGO_ENABLED=$(CGO_ENABLED) go install ./...

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

# staticcheck is pinned by the tool directive to the version magic-cli-remote
# pins (v0.8.1). It is built once for THIS machine from that requirement.
# `go tool staticcheck` cannot do the per-GOOS pass: a foreign GOOS rebuilds
# the tool itself and the cross-compiled binary will not run (magic-cli-remote
# MADR 0170). The host binary is then run for linux, darwin, and windows,
# because some files build for one platform only.
STATICCHECK_BIN := bin/tools/staticcheck$(if $(filter windows,$(HOST_GOOS)),.exe,)

$(STATICCHECK_BIN): go.mod go.sum
	@mkdir -p bin/tools
	GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) CGO_ENABLED=0 go build -o $(STATICCHECK_BIN) honnef.co/go/tools/cmd/staticcheck

staticcheck: $(STATICCHECK_BIN)
	@set -e; rc=0; \
	ver="$$(./$(STATICCHECK_BIN) -version)"; \
	for goos in linux darwin windows; do \
		echo "staticcheck $$ver GOOS=$$goos"; \
		GOOS=$$goos CGO_ENABLED=0 ./$(STATICCHECK_BIN) ./... || rc=1; \
	done; \
	exit $$rc

vulncheck:
	go tool govulncheck ./...

tidy:
	go mod tidy

clean:
	rm -rf bin

FILES ?=
pre-add-check:
	bash scripts/go-precheck.sh $(FILES)

# All phase-0 gates. archtest, apidiff, and verify-build-metadata are no-ops
# until the phases that give them something to check.
preflight: check-cgo-off
	@echo "==> gofmt"; \
	unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt drift (run 'make fmt'):"; echo "$$unformatted"; exit 1; \
	fi
	@echo "==> go mod tidy"; \
	cp go.mod .go.mod.pre && cp go.sum .go.sum.pre; \
	go mod tidy; \
	rc=0; \
	if ! diff -q go.mod .go.mod.pre >/dev/null || ! diff -q go.sum .go.sum.pre >/dev/null; then \
		echo "go.mod/go.sum not tidy - commit the result of 'make tidy'"; rc=1; \
	fi; \
	mv -f .go.mod.pre go.mod; mv -f .go.sum.pre go.sum; \
	exit $$rc
	@echo "==> pre-add-check"; bash scripts/go-precheck.sh $(shell git ls-files '*.go'; git ls-files --others --exclude-standard '*.go')
	@echo "==> go vet"; go vet ./...
	@echo "==> staticcheck"; $(MAKE) --no-print-directory staticcheck
	@echo "==> golangci-lint"; golangci-lint run ./...
	@echo "==> govulncheck"; go tool govulncheck ./...
	@echo "==> fix-check"; $(MAKE) --no-print-directory fix-check
	@echo "==> archtest"; $(MAKE) --no-print-directory archtest
	@echo "==> apidiff"; $(MAKE) --no-print-directory apidiff
	@echo "==> verify-build-metadata"; $(MAKE) --no-print-directory verify-build-metadata
	@echo "preflight passed"

check-cgo-off:
	@if [ "$(CGO_ENABLED)" != "0" ]; then \
		echo "check-cgo-off: refusing to build with CGO_ENABLED=$(CGO_ENABLED)." >&2; \
		echo "Shipped binaries are pure Go." >&2; \
		exit 1; \
	fi
	@hits=$$(grep -R -n -E '(^|[[:space:]])import[[:space:]]+"C"|(^|[[:space:]])"C"[[:space:]]*(//.*)?$$' \
		--include='*.go' --exclude-dir=.git --exclude-dir=vendor --exclude-dir=bin . || true); \
	if [ -n "$$hits" ]; then \
		echo "check-cgo-off: cgo import \"C\" is not allowed:" >&2; \
		printf '%s\n' "$$hits" >&2; \
		exit 1; \
	fi

# No-op until a cmd/gobble binary exists (phase 2) and release ldflags are
# stamped (phase 4). The target name is part of the phase 0 Makefile.
verify-build-metadata:
	@if [ ! -d cmd/gobble ]; then \
		echo "verify-build-metadata: no-op (cmd/gobble is not in this phase)"; \
		exit 0; \
	fi; \
	echo "verify-build-metadata: cmd/gobble exists; phase 4 owns the ldflags check" >&2; \
	exit 1

fix-check:
	@out="$$(go fix -diff ./...)"; \
	if [ -n "$$out" ]; then \
		printf '%s\n' "$$out"; \
		echo "fix-check: go fix -diff ./... is not empty" >&2; \
		exit 1; \
	fi

# No-op until internal/archtest lands in phase 1.
archtest:
	@if [ -d internal/archtest ]; then \
		go test ./internal/archtest/...; \
	else \
		echo "archtest: no-op until phase 1"; \
	fi

# No-op until the first tag (0004-PLAN phase 0).
apidiff:
	@if git describe --tags --exact-match >/dev/null 2>&1 || git tag -l 'v*' | grep -q .; then \
		echo "apidiff: a tag exists; phase 0 leaves the comparison unimplemented" >&2; \
		exit 1; \
	fi; \
	echo "apidiff: no-op until the first tag"
