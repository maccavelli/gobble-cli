---
status: proposed
date: 2026-09-29
associated-madr: "0004-MADR-go-module-architecture.md"
---
# Implement the pigo module architecture: scaffold, contracts, boundaries, toolchain, release

Associated MADR: [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md)

Also executes the identity decisions in
[0003-MADR-pigo-product-identity.md](0003-MADR-pigo-product-identity.md)
(names, directories, build info). The Pi bridge is not built here.

## Goal

A buildable, testable, releasable Go 1.27.1 module whose package tree,
contracts, import boundaries, and toolchain gates match 0004-MADR. At the
end it produces a `pigo` binary that answers `version`, `completion`, and
`config path`, and it contains no agent behaviour.
[0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md)
Phase 1 starts from this tree.

## Scope

In:

* `go.mod`, `tool` directives, Makefile, `scripts/go-precheck.sh`,
  `.golangci.yml`, `.editorconfig`, `LICENSE`, `NOTICE`
* One `doc.go` per package in the 0004-MADR package map, stating the tier
* Contract types and interfaces for `llm`, `tool`, `session`, `hook`,
  `permission` and `pigo` (compiling, documented, no behaviour)
* `internal/archtest`, `internal/buildinfo`, `internal/appdirs`,
  `internal/logging`, `internal/cli` root (fang)
* Test harness packages `llm/llmtest` and `acpclient/acptest` (skeletons
  that later phases fill)
* CI (three operating systems) and a tag-driven release workflow, dry-run
  only
* `docs/architecture.md` rewritten to describe the tree that now exists

Out:

* Any agent loop, tool, provider, or ACP method behaviour. That is
  0002-PLAN from Phase 1 onwards, then
  [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md).
* Pushing a tag or publishing a release. A tag needs an explicit ask in
  the same turn, per the global rule.
* Renaming the repository (0003-MADR, the option not chosen).

## Implementation Steps

### Phase 0 — module and toolchain

1. `go mod init github.com/maccavelli/pi-go`. Set `go 1.27.1`.
2. Add `tool` directives:
   * `golang.org/x/vuln/cmd/govulncheck`
   * `honnef.co/go/tools/cmd/staticcheck`, pinned to the version
     magic-cli-remote pins
   * `golang.org/x/exp/cmd/apidiff`
   * `golang.org/x/lint/golint`

   Record the resolved versions in this PLAN (dated note).
3. Decide the ACP pin: `github.com/coder/acp-go-sdk v0.13.5`, and whether
   to add `replace github.com/coder/acp-go-sdk => github.com/maccavelli/acp-go-sdk v0.13.6-mcr.1`.
   0004-MADR recommends the replace. Record the owner's choice here before
   the `require` lands.
4. Create the Makefile. Target names follow magic-cli-remote:
   * `build`, `install`, `test`, `race`, `vet`, `fmt`, `lint`
     (golangci-lint v2), `staticcheck` (for linux, darwin, windows),
     `vulncheck`, `tidy`, `clean`
   * `pre-add-check` (runs `scripts/go-precheck.sh`)
   * `preflight` (all gates)
   * `check-cgo-off`
   * `verify-build-metadata`
   * `fix-check` (`go fix -diff ./...` must be empty)
   * `archtest`
   * `apidiff` (no-op until the first tag)
5. Write `scripts/go-precheck.sh`: gofmt (not gofumpt), per-file golint,
   and govulncheck (`GO_PRECHECK_SKIP_VULN=1` skips it). This is the same
   contract as the fleet script and the global pre-add hook.
6. Write `.golangci.yml` (v2) with mcplib's linter set: bodyclose,
   errcheck, errorlint, gocognit, goconst, gocritic, gocyclo, gosec, govet,
   ineffassign, makezero, misspell, nakedret, nilerr, revive, staticcheck,
   unconvert, unparam.
7. Add `LICENSE` (MIT) and `NOTICE`, which credits Pi's MIT copyright line
   (0003-MADR).

**Accept:**

* `go build ./...` and `go test ./...` pass.
* `make preflight` passes.
* `go tool govulncheck ./...` runs from the tool directive, with no global
  install.
* `make check-cgo-off` fails on a scratch copy that adds `import "C"`.
  Quote that failure in the handoff.

### Phase 1 — package skeleton and contracts

1. Create each directory in the 0004-MADR package map. Each gets a
   `doc.go` with a package comment whose first line after the summary is
   `Stability: stable|beta|internal|experimental`.
2. Write the contract types:
   * `llm`: `Provider`, `Request`, sealed `Event` set, `Message`,
     `Content`, `Usage`, `Cost`, `Model`, typed errors;
   * `tool`: `Tool`, `Spec`, `Call`, `Env`, `Result`, `Annotations`,
     `Kind`, `Exposure`, generic `New[In, Out]`;
   * `session`: `Store`, `Log`, `Entry` union, `Header`, `ID`;
   * `hook`: `Hook`, `Event`, `Outcome`;
   * `permission`: `Policy`, `Rule`, `Decision`, `Mode`;
   * `pigo`: `New`, `Option`, `Runtime` (methods return
     `errors.ErrUnsupported` until later phases).

   Every wire struct uses `encoding/json/v2` tags. It has `omitzero` where
   a field is optional and a `json:",unknown"` `jsontext.Value` where the
   MADR requires forward compatibility.
3. Write `internal/archtest`. It runs `go list -deps -json ./...` and
   enforces the seven import-boundary rules in 0004-MADR, with a table
   test per rule.
4. Write the `Example` functions for `tool.New` and `pigo.New`. They
   compile, and their output is `// Output:`-checked where there is output.

**Accept:**

* `archtest` is green on the tree.
* On a **scratch clone** (never the working tree), plant
  `internal/cli → agent` and then `agent → charm.land/lipgloss/v2`. Each
  run fails and names the edge. Quote both failures in the handoff.
* `go doc ./tool` shows `New` with its generic signature.
* A round-trip test for a `session.Entry` with an unknown field is
  written, and fails against a copy that drops the `unknown` tag.

### Phase 2 — process edge and identity

1. `cmd/pigo/main.go`: `signal.NotifyContext(ctx, os.Interrupt,
   syscall.SIGTERM)`, then `os.Exit(cli.Main(ctx, os.Args[1:], os.Stdin,
   os.Stdout, os.Stderr))`.
2. `internal/buildinfo`:
   * `-X` variables `version`, `commit`, `date`, `buildKind`;
   * a `debug.ReadBuildInfo` fallback (module version and `vcs.revision`)
     so that `go install …@vX` still reports a version;
   * a User-Agent builder that returns the 0003-MADR format.
3. `internal/appdirs`: the 0003-MADR table, including `PIGO_HOME` and the
   per-role overrides.
4. `internal/logging`:
   * `slog` JSON or text output;
   * a rolling file sink in the state directory (the magic-cli-remote MADR
     0157 pattern, dependency-free);
   * `slog.NewMultiHandler` fan-out;
   * a redacting `ReplaceAttr` for keys matching `*key`, `*token`,
     `*secret`, and `authorization`;
   * **no stderr handler when the process is `pigo acp`**, because stdio
     is the protocol.
5. `internal/cli` root through `fang.Execute`. Subcommands:
   * `version [--json]`;
   * `completion` (fang or Cobra);
   * `config path`, which prints the four role directories;
   * `acp`, which returns "not yet implemented" and exit code 2 until
     0002-PLAN Phase 1.
6. Exit-code table in `internal/cli/exit.go`: 0 ok, 1 runtime failure, 2
   usage, 3 authentication, 4 cancelled (130 for SIGINT is kept when a
   signal ended the process).

**Accept:**

* `pigo version --json` prints `{name:"pigo", version, commit, date, go,
  os, arch}`.
* The `appdirs` table test passes for three GOOS values, with a fake
  `$HOME`, `$XDG_*`, and `%APPDATA%`.
* A test starts `pigo acp` with a closed stdin and asserts that nothing is
  written to stdout except JSON-RPC. It fails on a copy that logs a
  start-up banner to stdout.

### Phase 3 — test harnesses

1. `llm/llmtest`:
   * a `Script` of turns (text deltas, tool calls, usage, errors,
     `Retry-After`), replayed as an `llm.Provider`;
   * assertion helpers for requests (system prompt sections, tool list,
     message tail).
2. `acpclient/acptest`:
   * `Pair(t, agent acp.Agent)`, which returns a connected client and a
     recorder of every JSON-RPC frame in both directions, in order;
   * golden-file comparison that writes the actual transcript to
     `t.ArtifactDir()` on mismatch.
3. A `synctest.Test` example in `internal/…`: a retry back-off of 2 s,
   4 s, then 8 s completes in virtual time. It proves the pattern for
   0005-PLAN's retry phase.
4. Live-test convention: a build tag `live` per provider (`live_anthropic`,
   `live_openai`, …), skipped without credentials, never run in default CI.

**Accept:**

* `acptest.Pair` round-trips `initialize` against a trivial stub agent.
* The golden helper writes an artifact on a deliberately wrong golden
  file, in a scratch test run.
* The `synctest` example finishes in under 100 ms of wall time.

### Phase 4 — CI and release pipeline

1. `.github/workflows/ci.yml`:
   * matrix ubuntu, macos, windows;
   * `actions/setup-go` with `go-version-file: go.mod`;
   * steps `make preflight`, `make race` (not on windows arm), and
     `make archtest`;
   * golden-transcript artifacts are uploaded on failure.
2. Release job on `v*` tags, in the same workflow, following the fleet
   `ci-server.yml` template:
   * build `CGO_ENABLED=0 -trimpath -tags netgo,osusergo` for
     darwin/{amd64,arm64}, linux/{amd64,arm64}, windows/{amd64,arm64};
   * archives, `SHA256SUMS`, and an SPDX SBOM;
   * `actions/attest-build-provenance` for each archive;
   * `softprops/action-gh-release`.
3. `make release-dry-run`: builds every target locally into `dist/`
   (git-ignored), writes `SHA256SUMS`, and verifies each binary's
   `version --json` for the host platform.
4. No dependabot configuration (fleet policy).

**Accept:**

* The CI run is green on all three operating systems on a pushed branch.
  Pushing needs the owner's ask in that turn; otherwise the evidence is
  `act` or a local `make preflight` on each available OS, and the gap is
  recorded here.
* `make release-dry-run` produces six archives, and `SHA256SUMS` verifies
  with `shasum -a 256 -c`.
* `make verify-build-metadata` fails on a scratch build without
  `-ldflags`, then passes on the real build.

### Phase 5 — documentation

1. Rewrite [architecture.md](../architecture.md) to list every package
   that now exists, its tier, and its one-line role, plus the
   import-boundary rules as they are enforced. No rationale.
2. Add `docs/guides/developing.md`: how to build, run gates, add a
   package (tier plus doc.go plus archtest rule), and run live tests.
3. Update `docs/README.md` "I want to…" rows for those two documents.

**Accept:** every package directory is named in architecture.md, checked
by a throwaway script run in the scratchpad that diffs `go list ./...`
against the document. The script is shown failing on a copy of the
document with one package deleted.

## Verification

After Phase 5, from a clean clone:

```text
make preflight            # gofmt, golint, vet, staticcheck×3, golangci-lint, govulncheck, fix-check, archtest
go test ./...
go test -race ./...
make release-dry-run
./dist/pigo_<host>/pigo version --json
```

Capture exit status before any filter (global rule). Each gate added in
this plan (check-cgo-off, archtest, the stdout-purity test, the round-trip
test, verify-build-metadata, the architecture.md diff) must be recorded
here as having been seen failing on a planted defect. Record where the
defect was planted (scratch copy or scratch clone) and the failure text.

## Rollout and Rollback

* No users. Each phase is one commit after its pre-add checks pass.
* Rollback is `git revert` of the phase commit. Phases 1–3 depend on
  Phase 0. Phase 4 is independent of Phases 1–3.
* When this plan completes, 0002-PLAN's Phase 0 is satisfied, and
  0002-PLAN Phase 1 begins.

## Execution record

None. This plan is proposed and has not been approved.
