# gobble-cli toolchain (0004-PLAN phase 0).
# Target names follow magic-cli-remote. apidiff is present and a no-op
# below v1.0.0.
MODULE := github.com/maccavelli/gobble-cli
# The modules each Go gate runs over: the root, and the nested acpext, the
# ACP extension contract (0042-MADR D6). Each runs in its own directory,
# with the root's tools.
MODULES := . acpext
# A loop over them keeps each module's failure in rc: macOS's GNU Make 3.81
# ignores .SHELLFLAGS, so a recipe does not stop at a failing command.
# Recipes need bash (for pipefail) and grep, diff, cp and mv. On Windows
# that is Git's bash, and Git's usr/bin goes on PATH because Git bash does
# not put its coreutils there when make starts it from PowerShell. Set
# GIT_BASH if Git is installed elsewhere. Every other host uses /bin/bash
# (0007-MADR D5).
ifeq ($(OS),Windows_NT)
  GIT_BASH ?= C:/PROGRA~1/Git/usr/bin/bash.exe
  SHELL := $(GIT_BASH)
  export PATH := $(dir $(GIT_BASH)):$(PATH)
else
  SHELL := /bin/bash
endif
.SHELLFLAGS := -eu -o pipefail -c

# Shipped builds are pure Go. override beats an ambient or command-line
# CGO_ENABLED, but only on these targets. Do not export it: local race
# testing keeps CGO_ENABLED=1, and the race target turns cgo on.
build install check-cgo-off: override CGO_ENABLED := 0

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

# The Python the records and build-metadata checks run with; the Windows CI
# runner names it python (0002-PLAN Phase 8).
PYTHON ?= python3

# Build identity is go-selfupdate-lib buildinfo's two stamps; commit and date
# come from the checkout, which go build records (0004-MADR amendment of
# 2026-10-05). A release build sets VERSION to its tag and BUILD_KIND=release.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)
BUILD_KIND ?= local
BUILDINFO := github.com/maccavelli/go-selfupdate-lib/buildinfo
LDFLAGS := -X $(BUILDINFO).version=$(VERSION) -X $(BUILDINFO).kind=$(BUILD_KIND)

.PHONY: build install test race vet fmt lint staticcheck vulncheck tidy clean \
	pre-add-check preflight check-cgo-off verify-build-metadata fix-check archtest apidiff check-acpext-deps \
	check-records check-icon check-libs markdownlint probe-conhost completion-shells perf $(BIN)

# Shipped binaries are pure Go. check-cgo-off refuses CGO_ENABLED other than 0
# and any `import "C"` in a .go file (0004-PLAN phase 0 accept).
build: check-cgo-off
	CGO_ENABLED=$(CGO_ENABLED) go build ./...

# The gobble binary, stamped.
$(BIN): override CGO_ENABLED := 0
$(BIN): check-cgo-off
	@mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/gobble

install: check-cgo-off
	@mkdir -p "$(USER_BIN_DIR)"
	CGO_ENABLED=$(CGO_ENABLED) go install ./...

test:
	@rc=0; for m in $(MODULES); do echo "go test ($$m)"; go -C $$m test ./... || rc=1; done; exit $$rc

race:
	@rc=0; for m in $(MODULES); do echo "go test -race ($$m)"; CGO_ENABLED=1 go -C $$m test -race ./... || rc=1; done; exit $$rc

# The CLI mode's timings against their baselines, at most three times each
# (0030-MADR D5). Not part of preflight: timing varies by host and by load.
perf:
	go test -tags perf -run '^TestPerfBudget$$' -count=1 -v ./internal/cli/ ./internal/cli/editor/

vet:
	@rc=0; for m in $(MODULES); do echo "go vet ($$m)"; go -C $$m vet ./... || rc=1; done; exit $$rc

fmt:
	gofmt -w .

# golangci-lint v2.14.0 is the fleet pin (0007-MADR D5).
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0" >&2; \
		exit 2; \
	}
	@rc=0; for m in $(MODULES); do echo "golangci-lint ($$m)"; (cd $$m && golangci-lint run --config "$(CURDIR)/.golangci.yml" ./...) || rc=1; done; exit $$rc

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
		for m in $(MODULES); do \
			(cd $$m && GOOS=$$goos CGO_ENABLED=0 "$(CURDIR)/$(STATICCHECK_BIN)" ./...) || rc=1; \
		done; \
		GOOS=$$goos CGO_ENABLED=0 $(PYTHON) scripts/check_staticcheck_ignored.py ./$(STATICCHECK_BIN) || rc=1; \
	done; \
	exit $$rc

vulncheck:
	@rc=0; for m in $(MODULES); do echo "govulncheck ($$m)"; go tool govulncheck -C $$m ./... || rc=1; done; exit $$rc

tidy:
	@rc=0; for m in $(MODULES); do go -C $$m mod tidy || rc=1; done; exit $$rc

clean:
	rm -rf bin

FILES ?=
pre-add-check:
	bash scripts/go-precheck.sh $(FILES)

# Every gate. apidiff is a no-op below v1.0.0.
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
	if ! go -C acpext mod tidy -diff; then \
		echo "acpext/go.mod not tidy - commit the result of 'make tidy'"; rc=1; \
	fi; \
	exit $$rc
	@echo "==> pre-add-check"; bash scripts/go-precheck.sh $(shell git ls-files '*.go'; git ls-files --others --exclude-standard '*.go')
	@echo "==> go vet"; $(MAKE) --no-print-directory vet
	@echo "==> staticcheck"; $(MAKE) --no-print-directory staticcheck
	@echo "==> golangci-lint"; $(MAKE) --no-print-directory lint
	@echo "==> govulncheck"; $(MAKE) --no-print-directory vulncheck
	@echo "==> check-acpext-deps"; $(MAKE) --no-print-directory check-acpext-deps
	@echo "==> fix-check"; $(MAKE) --no-print-directory fix-check
	@echo "==> archtest"; $(MAKE) --no-print-directory archtest
	@echo "==> apidiff"; $(MAKE) --no-print-directory apidiff
	@echo "==> verify-build-metadata"; $(MAKE) --no-print-directory verify-build-metadata
	@echo "==> check-records"; $(MAKE) --no-print-directory check-records
	@echo "==> check-icon"; $(MAKE) --no-print-directory check-icon
	@echo "==> markdownlint"; $(MAKE) --no-print-directory markdownlint
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

# Builds cmd/gobble stamped and unstamped and checks `version --json`
# (0008-PLAN P4; the logic is in Python, deviation 2).
verify-build-metadata:
	$(PYTHON) scripts/verify_build_metadata.py

fix-check:
	@for m in $(MODULES); do \
		out="$$(go -C $$m fix -diff ./...)" || { echo "fix-check: go fix -diff failed in $$m" >&2; exit 1; }; \
		if [ -n "$$out" ]; then \
			printf '%s\n' "$$out"; \
			echo "fix-check: go fix -diff ./... is not empty in $$m" >&2; \
			exit 1; \
		fi; \
	done

# acpext imports only the standard library (0042-MADR D6, A1): every
# package it and its tests build from is standard, or acpext's own.
check-acpext-deps:
	@deps="$$(go -C acpext list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./...)" || { \
		echo "check-acpext-deps: go list failed: acpext does not build from the standard library alone" >&2; \
		exit 1; \
	}; \
	bad="$$(printf '%s\n' "$$deps" | grep -v -E '^(github\.com/maccavelli/gobble-cli/acpext([._ ].*)?)?$$' || true)"; \
	if [ -n "$$bad" ]; then \
		echo "check-acpext-deps: acpext imports outside the standard library:" >&2; \
		printf '%s\n' "$$bad" >&2; \
		exit 1; \
	fi; \
	echo "check-acpext-deps: ok"

# Records and docs links (0007-MADR D6).
check-records:
	$(PYTHON) scripts/check_records.py --check-all

# assets/gobble.svg is what scripts/gen_icon.py writes, byte for byte, on
# every platform (0044-MADR D3).
check-icon:
	$(PYTHON) scripts/gen_icon.py --check

# The sibling libraries against their latest tags (0019-MADR D14); needs the network, so not in preflight.
check-libs:
	$(PYTHON) scripts/check_libs.py

# The fleet markdownlint config; records are excluded by its own globs
# (0007-MADR D2).
markdownlint:
	npx --yes markdownlint-cli2@0.23.2

# The live Console Host suite (0008-PLAN P3). Windows only: it opens a
# conhost.exe window for a few seconds and borrows the clipboard, which it
# saves and restores. Not part of preflight.
probe-conhost:
	go test -tags conhost -run '^TestConhost$$' -count=1 -v ./internal/cli/editor/

# Real-shell completion tests (0008-PLAN P5). Each host runs the shells the
# plan assigns it, and a missing one fails. Not part of preflight.
completion-shells:
	go test -tags shells -run '^TestShells$$' -count=1 -v ./internal/cli/complete/

# No-op until internal/archtest lands in phase 1.
archtest:
	@if [ -d internal/archtest ]; then \
		go test ./internal/archtest/...; \
	else \
		echo "archtest: no-op until phase 1"; \
	fi

# A no-op below v1.0.0: 0004-MADR covers stable packages from v1.0.0. From
# then it fails until 0005-PLAN F10 step 6 writes the comparison.
apidiff:
	@stable="$$(git tag -l 'v*' | grep -E '^v[1-9][0-9]*\.[0-9]+\.[0-9]+$$' || true)"; \
	if [ -n "$$stable" ]; then \
		echo "apidiff: tags at or above v1.0.0 exist ($$(echo $$stable)); the comparison is not written yet (0005-PLAN F10 step 6)" >&2; \
		exit 1; \
	fi; \
	echo "apidiff: no-op below v1.0.0"
