# Developing gobble

How to build gobble, run its tests and gates, add a package, and write a live test. What the
packages are and how they may import each other is in [architecture.md](../architecture.md). Why
they are that way is in the [decision records](../README.md).

## What you need

- **Go 1.27.1.** `go.mod` names it, and `go version` should print `go version go1.27.1 …`.
- **GNU make and bash.** On Windows, run `make` from Git Bash or from PowerShell. The Makefile uses
  Git's bash either way. In PowerShell a bare `bash` is WSL, not Git Bash.
- **Python 3**, for the records and build-metadata checks. The Makefile runs `python3`; set
  `PYTHON=python` where the interpreter has that name.
- **golangci-lint v2.14.0** on `PATH`:
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0`.
- **Node**, for `npx markdownlint-cli2`.

`golint`, `govulncheck`, `staticcheck` and `apidiff` come from the `tool` directives in `go.mod`, so
they need no install.

## Build and run

```sh
go build ./...          # every package
make bin/gobble.exe     # the stamped binary; bin/gobble on Linux and macOS
```

`make bin/gobble` stamps the build with `git describe` and the kind `local`, with cgo off. Shipped
binaries are pure Go, and `make check-cgo-off` refuses a build with cgo on.

```console
$ ./bin/gobble.exe version
gobble 0.0.0-dev+2513638e2cea.dirty
commit: 2513638e2ceaedf58285bc692dc9e69038ebe310
date:   2026-10-07T19:52:04Z
go:     1.27.1 windows/amd64
```

A local build reports its version as `0.0.0-dev`, plus the commit. `version --json` prints the same
fields as one JSON object. `version --identity` prints the line the release workflow checks.

`gobble` with no command is the line session, `gobble -p "…"` answers one prompt, and `gobble acp`
serves the agent on stdio. A prompt needs a model credential in the environment.

## Test

```sh
make test               # go test ./...
make race               # the race detector, with cgo on
```

The race detector needs cgo and a C toolchain. On Windows, run `make race` in WSL.

### Golden transcripts

`acpclient/acptest` records ACP frames, and tests compare them with files under `testdata/`. When a
transcript changes on purpose, rewrite the goldens and review the diff:

```sh
ACPTEST_UPDATE=1 go test ./...
git diff -- '*.golden'
```

A mismatch without the variable fails the test and leaves the actual transcript as an artifact;
run `go test -artifacts -outputdir <dir>` to keep it.

### Suites outside the default run

| Target | What it runs | Where |
| :--- | :--- | :--- |
| `make completion-shells` | Completion scripts in real shells. A missing shell fails the suite. | Windows: Git Bash, `pwsh`, `powershell.exe`. Linux: bash, fish. macOS: zsh, bash, `/bin/bash`. |
| `make probe-conhost` | The line editor in a real Console Host window. It borrows the clipboard, and saves and restores it. | Windows only |

Neither is part of `make preflight` or CI.

## The gates

Before you stage a Go file, run the pre-add check on it:

```console
$ make pre-add-check FILES="internal/archtest/check.go internal/archtest/check_test.go"
…
go-precheck: 2 file(s) clean (gofmt, golint, govulncheck).
```

With no `FILES` it checks every tracked Go file. `GO_PRECHECK_SKIP_VULN=1` skips govulncheck when
you are offline.

Before you commit, run every gate:

```sh
make preflight
```

It ends with `preflight passed`. In order, it runs:

1. cgo off, gofmt drift, and `go mod tidy` drift;
2. the pre-add check on every Go file;
3. `go vet`, then `staticcheck` for linux, darwin and windows;
4. `golangci-lint`, `govulncheck`, and `go fix -diff`;
5. `archtest`, the import rules;
6. `apidiff`, a no-op below `v1.0.0`;
7. `verify-build-metadata`, which builds `cmd/gobble` stamped and unstamped;
8. `check-records`, the records' names, pairs and links;
9. `markdownlint`.

Any one of them fails the run. Each is also a target of its own (`make archtest`, `make lint`, …),
for a quicker loop.

## Add a package

1. **Choose its tier.**
   - A public package starts as `beta`. The `v1.0.0` release gate promotes packages to `stable`
     one by one.
   - A package only gobble uses goes under `internal/`.
   - Unstable work goes under `exp/`. Only `cmd/gobble` may import it.
2. **Write its `doc.go`.** The package comment's first sentence says what it is. The next line
   states the tier:

   ```go
   // Package widget is an example package from the developing guide.
   // Stability: beta
   package widget
   ```

3. **Check the import rules.** `make archtest` enforces them; architecture.md lists them. The
   protocol SDKs, the provider SDK, go-selfupdate-lib, Kong, Charm and go-tui-lib each have one
   permitted home, so a new package that imports one fails:

   ```text
   --- FAIL: TestImportRules/rule_9_module (0.00s)
       check_test.go:139: unexpected edge:
           rule 9: github.com/maccavelli/gobble-cli/widget imports github.com/alecthomas/kong
   ```

   A new permitted edge means amending
   [0004-MADR](../decisions/0004-MADR-go-module-architecture.md) and its rule in
   `internal/archtest/check.go` together, never editing the test alone.
4. **Add its row** to the package table in architecture.md, with the same tier as its `doc.go`.
5. **A new module dependency needs a decision record.** No module is required unless a MADR in
   this repository names it. `go.mod` and `go.sum` change in the commit that adds the first import.

## Live tests

A test that calls a real model provider is a live test. None exists yet, and default CI runs none.
The convention, from `llm/llmtest`:

- the file carries the build tag `live_<provider>`, such as `live_anthropic`;
- the test reads its credential with `llmtest.LiveCredential`, which skips the test when the
  provider's usual variable is unset:

  ```go
  //go:build live_anthropic

  key := llmtest.LiveCredential(t, "ANTHROPIC_API_KEY")
  ```

- you run it with `go test -tags live_anthropic ./...`.

Never commit a credential or put one on a command line. Export it in your shell.

## CI and the release rehearsal

`.github/workflows/ci.yml` runs on every push to `main`, on every pull request, and on `v*` tags:

- `validate` runs `go test`, `make preflight` and `make archtest` on Linux, macOS and Windows, and
  `make race` on Linux and macOS;
- `sbom` saves the dependency graph's SPDX SBOM;
- `build` builds the five release binaries and runs `gobble version --identity` on each platform.
  Without a tag this is a rehearsal, stamped `rehearsal-<commit>`;
- `release` publishes a GitHub release, only on a `v*` tag at `main`'s head.

## Decision records

Mutating work follows a MADR and its PLAN in `docs/decisions/`. `AGENTS.md` has the workflow.

- `python3 scripts/check_records.py --next` prints the next free number.
- A follow-up to existing work amends that record with a dated entry; it does not rewrite the
  record's history.
- [docs/README.md](../README.md) indexes every record by hand. Update it in the same change.
