---
status: in-progress
date: 2026-10-04
associated-madr: "0004-MADR-go-module-architecture.md"
---
# Implement the gobble module architecture: scaffold, contracts, boundaries, toolchain, release

Associated MADR: [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md)

Also executes the identity decisions in
[0003-MADR-gobble-product-identity.md](0003-MADR-gobble-product-identity.md)
(names, directories, build info). The Pi bridge is not built here.

## Goal

A buildable, testable, releasable Go 1.27.1 module whose package tree,
contracts, import boundaries, and toolchain gates match 0004-MADR. At the
end it produces a `gobble` binary that answers `version`, `completion`, and
`config path`, and it contains no agent behaviour.
[0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md)
Phase 1 starts from this tree.

The CLI library is Kong. Phase 2 builds `internal/cli` with Kong, not
another command library. gobble has two modes: a native terminal CLI
mode, which is the default, and an enhanced terminal TUI mode. Core TUI
is go-tui-lib. This plan does not import it and does not reimplement
core TUI. Where a core behaviour is not in the library yet, gobble waits
rather than copying it. Self-update is go-selfupdate-lib. This plan does
not reimplement self-update. As much code as possible is canonical:
sibling libraries and the standard library rather than local copies.
The module is Go 1.27.1, idiomatic, and modular.

## Scope

In:

* `go.mod`, `tool` directives, Makefile, `scripts/go-precheck.sh`,
  `.golangci.yml`, ~~`.editorconfig`~~ *(struck 2026-10-04: no fleet repository has one; 0007-MADR D8)*, `LICENSE`, `NOTICE`
* One `doc.go` per package in the 0004-MADR package map, stating the tier
* Contract types and interfaces for `llm`, `tool`, `session`, `hook`,
  `permission` and `gobble` (compiling, documented, no behaviour)
* `internal/archtest`, `internal/buildinfo`, `internal/appdirs`,
  `internal/logging`, `internal/cli` root (Kong)
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

### Phase D — current sibling-source documentation (before Phase 0) (complete)

This phase may be approved and committed separately from the Go scaffold.
It changes documentation only. It does not add a module requirement or claim
that gobble code exists.

1. Reconcile `README.md`, `docs/README.md` and `docs/architecture.md` with
   the 2026-10-02 amendments of this MADR and
   [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md).
   `architecture.md` keeps describing the current documentation-only gobble
   tree; it identifies sibling code as a planned dependency, with version
   evidence and no invented package in gobble.
   Refresh the 0004/0005 MADR and PLAN source inventories if sibling commits
   or tags have changed since those amendments; keep released APIs separate
   from development HEAD.
2. Retain earlier MADR and PLAN paragraphs as dated history. Add clear
   current-state pointers where a reader would otherwise mistake the old
   `v1.1.0`, pre-S8 SDK or no-TUI-library assertions for today's API.
   Use 0005-PLAN's 2026-10-02 F4, F9 and F10 amendment as the future
   implementation boundary. Do not modify any sibling repository.
3. Check the three sibling commit/tag ids and exported APIs against their
   source, then check every newly added relative Markdown link resolves.
   Search the updated docs for the old current-state claims and ensure each
   is either labelled as a dated observation or superseded by a nearby
   current-state pointer. Run `git diff --check`.

**Accept:** the entry pages identify the three sibling repos and their
actual present surfaces; none says gobble imports them yet; links resolve;
`git diff --check` is clean. Commit this documentation phase after its
checks. Phases 0–5 retain their separate approval and execution order.

### Phase 0 — module and toolchain (complete)

1. `go mod init github.com/maccavelli/gobble-cli`. Set `go 1.27.1`.
2. Add `tool` directives:
   * `golang.org/x/vuln/cmd/govulncheck`
   * `honnef.co/go/tools/cmd/staticcheck`, pinned to the version
     magic-cli-remote pins (MADR 0170: staticcheck 0.8.1; re-resolve)
   * `golang.org/x/exp/cmd/apidiff`
   * `golang.org/x/lint/golint`

   Record the resolved versions in this PLAN (dated note).
3. ACP pin, chosen 2026-09-30 (0004-MADR recommendation, owner directed
   alignment with current facts): `require github.com/coder/acp-go-sdk v0.13.5`
   and
   `replace github.com/coder/acp-go-sdk => github.com/maccavelli/acp-go-sdk v0.13.6-mcr.1`,
   so the agent side has the same overflow policy as mcremote (magic-cli-remote
   MADR 0167). Do not add a relative `replace`.
4. Do **not** `require` `go-llmprovider-sdk` or `go-core-lib` in this
   phase. Nothing in the scaffold imports them. Record the pins later
   phases will use (re-resolved when the first import lands):
   * `github.com/maccavelli/go-core-lib v1.1.0`
     (`96b30961180671ab3697585951219001ecbb1c90`) — first imported in
     [0005-PLAN](0005-PLAN-v1-feature-scope.md) F10 (`gobble update`) and
     used as a workflow pin in this plan's Phase 4;
   * `github.com/maccavelli/go-llmprovider-sdk` — no tag. First imported
     in [0002-PLAN](0002-PLAN-cli-acp-headless-mcp-v1.md) Phase 3 at a
     commit pseudo-version of that module's `origin/main` (probed
     `3d4aff5f2be9363877aae0987755e6781f1cae98` on 2026-09-30). Re-resolve
     at 0005-PLAN F4. Do not import the pre-S8 `Generate*` API.
5. Create the Makefile. Target names follow magic-cli-remote:
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
6. Write `scripts/go-precheck.sh`: gofmt (not gofumpt), per-file golint,
   and govulncheck (`GO_PRECHECK_SKIP_VULN=1` skips it). This is the same
   contract as the fleet script and the global pre-add hook.
7. Write `.golangci.yml` (v2) with mcplib's linter set: bodyclose,
   errcheck, errorlint, gocognit, goconst, gocritic, gocyclo, gosec, govet,
   ineffassign, makezero, misspell, nakedret, nilerr, revive, staticcheck,
   unconvert, unparam. *(2026-10-04: mcplib's set is these plus `unused` and `whitespace`, with the `gofmt` and `goimports` formatters; 0007-PLAN P5 adds them.)*
8. ~~Add `LICENSE` (MIT) and `NOTICE`. `NOTICE` credits Pi's MIT copyright
   line (0003-MADR) and `go-core-lib` (Apache-2.0) for the self-update
   library and reusable workflow this plan will call. Do not invent a
   license line for `go-llmprovider-sdk`; that tree has no LICENSE file
   as of 2026-09-30.~~ *(Struck 2026-10-01: the owner chose Apache-2.0.
   `LICENSE` and `NOTICE` land in that change, ahead of this phase.
   Phase 0 does not recreate them.)* `LICENSE` is already the Apache
   License 2.0, the fleet copy. `NOTICE` already credits Pi (MIT),
   `go-core-lib` (Apache-2.0), and `go-llmprovider-sdk` (Apache-2.0).

**Accept:**

* `go build ./...` and `go test ./...` pass.
* `make preflight` passes.
* `go tool govulncheck ./...` runs from the tool directive, with no global
  install.
* `make check-cgo-off` fails on a scratch copy that adds `import "C"`.
  Quote that failure in the handoff.

### Phase 1 — package skeleton and contracts (complete)

1. Create each directory in the 0004-MADR package map. Each gets a
   `doc.go` with a package comment whose first line after the summary is
   `Stability: stable|beta|internal|experimental`.
2. Write the contract types:
   * `llm`: `Provider`, `Request`, sealed `Event` set, `Message`,
     `Content`, `Usage`, `Cost`, `Model`, typed errors. This is the
     agent-facing facade. It does **not** import
     `go-llmprovider-sdk` in this phase. Shape it so F4's adapter is
     mechanical, against the SDK contract probed on 2026-09-30
     (`Provider.ID/Capabilities/Generate`, `Stream` as
     `iter.Seq2[Event, error]`, `APIError` + the sentinels listed in
     0004-MADR). Include a short mapping comment in `llm/doc.go`:
     `text_delta`/`reasoning_delta`/`item`/`done` → the sealed set;
     `ErrRateLimited`/`ErrQuotaExhausted`/`ErrAuthFailure`/
     `ErrContextOverflow` → `llm.ErrRateLimited`/`ErrQuota`/`ErrAuth`/
     `ErrContextOverflow`.
   * `llm/provider`: `doc.go` only in this phase (Stability: beta). No
     adapter behaviour yet. 0002-PLAN Phase 3 fills it.
   * `tool`: `Tool`, `Spec`, `Call`, `Env`, `Result`, `Annotations`,
     `Kind`, `Exposure`, generic `New[In, Out]`;
   * `session`: `Store`, `Log`, `Entry` union, `Header`, `ID`;
   * `hook`: `Hook`, `Event`, `Outcome`;
   * `permission`: `Policy`, `Rule`, `Decision`, `Mode`;
   * `gobble`: `New`, `Option`, `Runtime` (methods return
     `errors.ErrUnsupported` until later phases).

   Every wire struct uses `encoding/json/v2` tags. It has `omitzero` where
   a field is optional and a `json:",unknown"` `jsontext.Value` where the
   MADR requires forward compatibility.
3. Write `internal/archtest`. It runs `go list -deps -json ./...` and
   enforces the eight import-boundary rules in 0004-MADR, with a table
   test per rule. Rules 4 and 8 have no edges to fail on until the
   first import of `go-llmprovider-sdk` (0002-PLAN Phase 3) and
   `go-core-lib` (0005-PLAN F10); the table still names them.
4. Write the `Example` functions for `tool.New` and `gobble.New`. They
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

Steps 7–13 were added on 2026-10-04 after
[0006-MADR-goose-cli-port-candidates.md](0006-MADR-goose-cli-port-candidates.md)
was accepted. They were not in this phase when it was first written.
Config implementation stays unspecified. The config surface is undecided
and will be either a native Kong facility or a surface we write.

1. `cmd/gobble/main.go`: `signal.NotifyContext(ctx, os.Interrupt,
   syscall.SIGTERM)`, then `os.Exit(cli.Main(ctx, os.Args[1:], os.Stdin,
   os.Stdout, os.Stderr))`.
2. `internal/buildinfo`:
   * `-X` variables `version`, `commit`, `date`, `buildKind`;
   * a `debug.ReadBuildInfo` fallback (module version and `vcs.revision`)
     so that `go install …@vX` still reports a version;
   * a User-Agent builder that returns the 0003-MADR format.
3. `internal/appdirs`: the 0003-MADR table, including `GOBBLE_HOME` and the
   per-role overrides.
4. `internal/logging`:
   * `slog` JSON or text output;
   * a rolling file sink in the state directory (the magic-cli-remote MADR
     0157 pattern, dependency-free);
   * `slog.NewMultiHandler` fan-out;
   * a redacting `ReplaceAttr` for keys matching `*key`, `*token`,
     `*secret`, and `authorization`;
   * **no stderr handler when the process is `gobble acp`**, because stdio
     is the protocol.
5. `internal/cli` root through Kong. Subcommands:
   * `version [--json]`;
   * `completion` (Kong), which prints a script for bash, zsh, fish,
     and powershell only. Not nu or elvish. Do not call clap_complete;
   * `config path`, which prints the four role directories;
   * `acp`, which returns "not yet implemented" and exit code 2 until
     0002-PLAN Phase 1. This stub is not the product `acp` command;
   * a session command and a one-shot command that share flags
     (steps 8 and 9);
   * session list, resume, and rename only (step 10);
   * `configure`, only on a TTY (step 12);
   * `info --check`, non-zero on provider failure (step 13).
6. Exit-code table in `internal/cli/exit.go`: 0 ok, 1 runtime failure, 2
   usage, 3 authentication, 4 cancelled (130 for SIGINT is kept when a
   signal ended the process).
7. In `cmd/gobble/main.go`, enable Windows virtual-terminal processing
   once on stdout and stderr before any UI, in addition to step 1.
   Use the standard library or an existing sibling. Do not depend on
   the Rust console crate. Do not copy an 8 MiB stack workaround
   (0006 D8).
8. Shared flags on the session command and the one-shot command, on the
   Kong definition in `internal/cli` (0006 D9).
9. One-shot input is exclusive among text, file, and stdin. Output is
   text or json only. Do not copy a stdin panic. Do not drop `--system`
   when the input is a file path (0006 D10).
10. Session subcommands are list, resume, and rename only (0006 D11).
11. A Go line editor with history. Ctrl+C clears the line. A Windows
   paste is not submitted once per line. Do not implement the editor
   with go-tui-lib. TUI stays go-tui-lib (0006 D12).
12. `configure` runs only on a TTY (0006 D13).
13. `info --check` exits non-zero on provider failure (0006 D14).

These commands are the Kong surface accepted in 0006. They do not
add agent-loop behaviour. That stays out of this plan (Scope).

Out of this phase, named by the 2026-10-04 acceptance in 0006 and not
added here: bare session (goose-as-default-session); the skills
command; a second parser (clap, Cobra, or fang), because this phase
stays on Kong; term aliases; man pages; hidden commands; Sigstore
update (self-update stays go-selfupdate-lib); a path-root override
(keep step 3); date log folders and 14-day cleanup (keep step 4);
product commands (real acp, serve, roam, mcp, gateway, schedule,
recipe, review, local-models, plugin); and a slash registry (one
registry outside the CLI binary, unknown slash is an error), which is
not a Phase 2 command. The `acp` line in step 5 remains the
not-yet-implemented stub. It is not the product command. Config
implementation stays unspecified.

**Accept:**

* `gobble version --json` prints `{name:"gobble", version, commit, date, go,
  os, arch}`.
* The `appdirs` table test passes for three GOOS values, with a fake
  `$HOME`, `$XDG_*`, and `%APPDATA%`.
* A test starts `gobble acp` with a closed stdin and asserts that nothing is
  written to stdout except JSON-RPC. It fails on a copy that logs a
  start-up banner to stdout.
* `gobble completion bash` and `gobble completion powershell` print a
  script on stdout and exit 0. nu and elvish are not required.
* Before any UI, virtual-terminal processing is enabled on stdout and
  stderr. The module graph does not depend on the Rust console crate,
  and there is no 8 MiB stack workaround.
* The session command and the one-shot command share one flag set.
* One-shot input is exactly one of text, file, or stdin, and the output
  is text or json. A file path keeps `--system`. A failed stdin read is
  an error, not a panic.
* Session offers list, resume, and rename, and no other session
  subcommand.
* The line editor keeps history, Ctrl+C clears the line, and a
  multi-line Windows paste is not submitted line by line.
* `configure` fails when stdin is not a terminal.
* `info --check` exits non-zero when the provider check fails.

### Phase 3 — test harnesses

1. `llm/llmtest`:
   * a `Script` of turns (text deltas, tool calls, usage, errors,
     `Retry-After`), replayed as an `llm.Provider`;
   * assertion helpers for requests (system prompt sections, tool list,
     message tail).
   * Do **not** import `llmprovider/llmtest` here. That package's `Run`
     and `Fake` exercise the SDK contract; gobble's `Script` exercises the
     facade. F4's adapter tests will sit between them.
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

Grounded in `go-core-lib` `v1.1.0` `selfupdate` and
`.github/workflows/publish-selfupdate-release.yml` at
`96b30961180671ab3697585951219001ecbb1c90`. That client selects **raw
binaries** named `ExactAssetName("gobble", platform)` =
`gobble-<goos>-<goarch>` with `.exe` on windows. It does not select
`.tar.gz` / `.zip` archives. The reusable workflow refuses any tag
that is not a strict `vMAJOR.MINOR.PATCH` (so `v1.0.0-rc.1` cannot
publish through it).

1. `.github/workflows/ci.yml`:
   * matrix ubuntu, macos, windows;
   * `actions/setup-go` with `go-version-file: go.mod`;
   * steps `make preflight`, `make race` (not on windows arm), and
     `make archtest`;
   * golden-transcript artifacts are uploaded on failure.
   * *(2026-10-04, 0007-MADR D8: see the amendment of that date for the CI hygiene this job also carries.)*
2. A **build** job on tags matching `v*` produces the staged set into
   an artifact. A **publish** job runs only on a strict stable tag and
   calls the reusable workflow. Record the pin in the workflow file:

   ```yaml
   uses: maccavelli/go-core-lib/.github/workflows/publish-selfupdate-release.yml@96b30961180671ab3697585951219001ecbb1c90 # go-core-lib v1.1.0
   with:
     artifact-name: gobble-release
     products-json: '["gobble"]'
     platforms-json: '[{"os":"darwin","arch":"amd64"},{"os":"darwin","arch":"arm64"},{"os":"linux","arch":"amd64"},{"os":"linux","arch":"arm64"},{"os":"windows","arch":"amd64"},{"os":"windows","arch":"arm64"}]'
     extra-assets-json: '["gobble.spdx.json"]'
   ```

   The staged artifact contains exactly:
   * `gobble-darwin-amd64`, `gobble-darwin-arm64`, `gobble-linux-amd64`,
     `gobble-linux-arm64`, `gobble-windows-amd64.exe`,
     `gobble-windows-arm64.exe`;
   * `SHA256SUMS` (the same parser the client uses;
     `selfupdate/testdata/manifest-parity/` is the fixture set);
   * `gobble.spdx.json` as the extra asset.
   Each binary is `CGO_ENABLED=0 -trimpath -tags netgo,osusergo` with
   the 0003-MADR ldflags. The reusable workflow attests and publishes;
   gobble does not call `softprops/action-gh-release` itself.
3. `make release-dry-run`: builds every target locally into `dist/`
   (git-ignored), writes `SHA256SUMS`, checks each name equals
   `ExactAssetName` (inline the name rule; do not import `go-core-lib`
   yet), and verifies the host-platform binary's `version --json`.
4. No dependabot configuration (fleet policy).

**Accept:**

* The CI run is green on all three operating systems on a pushed branch.
  Pushing needs the owner's ask in that turn; otherwise the evidence is
  `act` or a local `make preflight` on each available OS, and the gap is
  recorded here.
* `make release-dry-run` produces the six named binaries plus
  `SHA256SUMS`, and `SHA256SUMS` verifies with `shasum -a 256 -c`. A
  planted `gobble-linux-amd64.tar.gz` in a scratch `dist/` is **not**
  listed in `SHA256SUMS` and is not accepted by the verifier script.
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
./dist/gobble-$(go env GOOS)-$(go env GOARCH) version --json   # append .exe on windows
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

**Where execution stands (2026-10-04).** Phase D, Phase 0, and Phase 1
have run. Phase 0 is `925ef0abf83e475c33b3ffae14685617b035639e`
(`build: scaffold the gobble module (0004-PLAN phase 0)`). The same-day
commits `c4658ae2a64dd7787bbd1257865e4a760676701a`,
`9bc93114ad15fa8231ade96a9381c3e450f7d425`, and
`7ea8634daac4d1b73ef23a132cdd8aa65f5bffc3` keep ship builds CGO off and
`make race` CGO on. Phase 1 is
`661b14e768407264a38687e2db99201cae2a04a1`
(`build: add phase 1 package map and compile-only contracts (0004-PLAN)`).
Phases 2–5 have not run. There is no `cmd/gobble` process entry yet.
`git ls-remote origin refs/heads/main` on this date returned
`8e016ab53a5765b2c912fab338d873918b8eed36`. That tip's parent is
`917d03d9bbbbf09b47e356ad02558e389258f8a2`, whose parent is
`ddf7d97cfc21603224f583732820b1c95fb5a12d`. `ddf7d97` is an ancestor,
not the current origin tip.

**Phase D, 2026-10-02 — complete.** The owner approved this documentation
phase and then approved refreshing the records when sibling HEADs changed.
The phase updated the four 0004/0005 records, `README.md`, `docs/README.md`
and `docs/architecture.md`; it changed no Go source or sibling repository.
Source checks observed SDK commit `efd9c61` (F4-facing contracts unchanged
since `67fc56e`), go-core-lib tag `v1.3.1` at `351bd6a` and development
commit `bf7221a`, and go-tui-lib tag `v0.1.0` at `5c57806` with later
documentation-only commit `7907590`. A relative-link check resolved
94 links in the seven edited files. Its deliberate bad input
`[broken](missing.md)` failed with `broken relative link: missing.md`.
`git diff --check` passed. On that date gobble had no `go.mod`, so no Go test was run.
The documentation phase is committed with this record. On that date Phases 0–5
had not been approved.

**Phase D refresh, 2026-10-03 — complete.** The owner approved the
2026-10-03 amendment below ("Update docs. Proceed"). The refresh added the
source corrections of that date to 0004-MADR and 0005-MADR, and the
amendments to this PLAN and 0005-PLAN. It updated `README.md`'s Self-update
line, `docs/architecture.md`'s sibling row, `NOTICE`, and three status cells
in `docs/README.md`. It changed no Go source and no sibling repository.
Source checks: `git ls-remote` on `maccavelli/go-selfupdate-lib` peels
`v1.4.0` to `4d7b053`, `v1.4.1` to `58411f1` and `v1.5.0` to `6deaa52`.
The tag trees have no `buildinfo` or `selfupdate/cli` at `v1.3.1`, and have
both at `v1.4.0` and `v1.5.0`. `selfupdate/e2e_running_test.go` exists at
`v1.5.0`. A relative-link check, anchors included, resolved 95 links in the
seven edited Markdown files. On a scratch copy, a planted
`[broken](missing.md)` in `docs/README.md` failed with `broken relative
link: missing.md`. Over `README.md`, `NOTICE` and `docs/architecture.md`,
the go-core-lib scan found only the "formerly" and "the old path" mentions.
A stray mention planted in a copy of `README.md` failed it. markdownlint-cli2,
with its default configuration, reports the same 23 issues (22 MD013, one
MD040) at the same lines before and after. `git diff --check` passed.

**Phase 1, 2026-10-04 — complete.** The package map and compile-only contracts are in the tree. No command library is imported. `make preflight` exited 0. `go test ./internal/archtest` was green (`ok github.com/maccavelli/gobble-cli/internal/archtest`). On a deleted scratch copy, planting `internal/cli` importing `agent` failed archtest with `rule 6: github.com/maccavelli/gobble-cli/internal/cli imports github.com/maccavelli/gobble-cli/agent`. Planting `agent` importing `charm.land/lipgloss/v2` failed with `rule 1: github.com/maccavelli/gobble-cli/agent imports charm.land/lipgloss/v2` and the same edge under rule 5. `session.Entry` keeps unknown JSON members with `json:",embed"` because Go 1.27.1 ignores `json:",unknown"`. Replacing that tag with `json:"unknown"` failed `TestEntryUnknownFieldRoundTrip`: unknown fields not preserved, got `{"type":"message","id":"abc","unknown":null}`.

## Amendments

**2026-09-30 — shared libraries as they exist.** 0004-MADR's amendment of
this date changes D10 and the toolchain half of D9. This PLAN:

* records the ACP `replace` as the Phase 0 choice;
* records the go-core-lib `v1.1.0` and go-llmprovider-sdk (no tag,
  `3d4aff5`) pins, and keeps both out of Phase 0 `go.mod`;
* adds `llm/provider` as a `doc.go` in Phase 1;
* shapes the `llm` contract as a facade over the probed SDK API;
* replaces Phase 4 archives + `softprops/action-gh-release` with
  `ExactAssetName` raw binaries and the go-core-lib reusable workflow;
* names `go-core-lib` (Apache-2.0) in `NOTICE`.

`gobble update` stays 0005-PLAN F10. Native token streaming stays an SDK
capability gobble consumes when it exists.

**2026-10-01 — Apache-2.0 licence, ahead of Phase 0.** The owner
directed Apache-2.0 for this repository, `go-llmprovider-sdk`, and
`go-core-lib`. 0003-MADR's licence paragraph is Apache-2.0.
`LICENSE` (fleet copy) and `NOTICE` are added in this change, so
Phase 0 step 8 is struck. `go-llmprovider-sdk` is licensed by that
repository's `0018-MADR-apache-2-license.md`; `NOTICE` names it.

**2026-10-01 — shared libraries measured in code (MADR second amendment of this date).** No new phases. Changes to existing phases:

* **Phase 0, step 3.**
  * The `replace` stays.
  * Add to the step: `acpserver` constructs `AgentSideConnection` with the default overflow policy; `acpclient` passes `WithNotificationOverflowPolicy(OverflowDropNewest)` and a drop handler.
* **Phase 0, step 4.** The pins to record change:
  * go-core-lib `v1.2.0` (`cfc95c883220b013c21705e1ebe3a268bdd63b56`) is the newest tag. F10 re-resolves to the newest `v1.x` and uses the same peeled SHA in `go.mod` and in the workflow `uses:`.
  * go-llmprovider-sdk: development pin `940fee0`. No gobble tag ships on a pseudo-version (MADR decision 1).
* **Phase 1, step 2.** The `llm` contract takes the wider shape from MADR decision 2:
  * the `ToolCallStart` and `ToolCallDelta` events, and `Done{StopReason}`;
  * `Usage` with cache read/write and `Estimated`;
  * image `Content`;
  * opaque `ProviderData`;
  * the four extra sentinels.

  The mapping comment in `llm/doc.go` also records:
  * the error-mapping order;
  * the `*RateLimitError` / `*IncompleteError` handling until S8;
  * the thinking-level map.
* **Phase 1, step 3.** In archtest rule 8, `_test.go` files may import `selfupdate/selfupdatetest`. The rule's table test proves that the exemption does not leak to non-test files: it is shown failing on a scratch file that imports it outside a test.
* **Phase 2, step 2.** `internal/buildinfo` follows MADR "`internal/buildinfo` follows go-core-lib's planned rule": `ReleaseBuild` only for `buildKind=release` plus a strict tag. The date stamp is the commit time (`SOURCE_DATE_EPOCH`), not the wall clock, so builds are reproducible.
* **Phase 2, step 6.** The exit-code table adds `10` for "update available" (`gobble update --check` only). Every other selfupdate error maps to `1`, except two:
  * contradictory update flags are `2` (usage);
  * a cancelled update is `4`, or `130` on SIGINT.
* **Phase 4.** Read the original steps with these additions.
  * **Precondition, recorded with evidence before the first tag** (an owner action, not this plan's): on `maccavelli/gobble-cli`,
    * "immutable releases" is enabled;
    * a tag ruleset restricts `v*` to the owner, with 2FA.

    Without immutable releases the reusable workflow publishes, waits 120 s, fails, and leaves a mutable release that every client refuses (`ErrMutableRelease`).
  * **The calling job declares `permissions: {contents: write, id-token: write, attestations: write}`**, which the snippet in step 2 omitted.
  * **The SBOM generator is named, and pinned by version and checksum, in this PLAN before step 2 lands.** `gobble.spdx.json` is an extra asset. It is **not** in `SHA256SUMS`. Optionally it is attested with `actions/attest-sbom`.
  * **Tags.** The build job runs on strict `vX.Y.Z` tags only, with no release-candidate tags until go-core-lib `v1.3.0` channels are adopted by a later amendment. `make verify-build-metadata` asserts `buildKind=release` on tag builds, and is shown failing on a scratch build stamped `local`.
  * **`make release-dry-run` runs go-core-lib's `scripts/verify-selfupdate-release.sh`** fetched at the pinned SHA, instead of an inlined copy of the name rule. The planted-archive accept stays.
  * **Actions in gobble's own workflows are pinned by commit SHA**, as go-core-lib's are.
  * **Users verify releases** with `gh release verify` and `gh attestation verify`. `docs/guides/developing.md` (Phase 5) documents the commands.

**2026-10-02 — sibling-source refresh and shared TUI workspace.** Adds
documentation-only Phase D, available before Phase 0. The current evidence
is go-llmprovider-sdk `67fc56e` (no tag, with an uncommitted PLAN edit),
go-core-lib `v1.3.0` at `f97c681`, and go-tui-lib `v0.1.0` at `5c57806`.
Phase 0 still imports none of them. At its later Phase 4, resolve the
go-core-lib workflow to the then-selected tag's peeled commit; `v1.3.0`
currently supplies channels, while gobble's release workflow remains
stable-only. Phase 1's `llm` facade keeps `Estimated`, but its adapter later
prefers the SDK's now-decoded `Response.Usage`. F9 in 0005-PLAN selects a
corrected tested go-tui-lib tag for workspace use; Phase 1 does not import
it. The old version pins and pre-S8 statements above remain dated history.

**2026-10-02 — Phase D source-inventory deviation.** After Phase D approval,
go-core-lib had gained a `v1.3.1` tag (`351bd6a`) and development HEAD
`bf7221a` added `buildinfo`; the 2026-10-02 MADR inventory at `v1.3.0`
therefore no longer described the current sibling tree. The provider SDK's
`d4ca925` first changed only its `0015-PLAN`; later `efd9c61` added catalog
and wizard work without changing the F4-facing contract. The TUI library's
later `7907590` changed documentation only. The owner approved refreshing
the 0004/0005 records and continuing Phase D. Phase D step 1 now includes
those record corrections. No sibling repository or gobble source code is
changed by this phase.

**2026-10-03 — go-core-lib renamed go-selfupdate-lib; Phase D refresh
(approved 2026-10-03).** The owner asked, after go-core-lib's rename: "update docs".
The source corrections of this date in 0004-MADR and
[0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) record the
rename and the released surface at `v1.5.0`. Phase D runs again for this
one sibling, with documentation only, and no sibling repository changes:

1. **Step 1, for go-selfupdate-lib.** `README.md`'s Self-update line and
   `docs/architecture.md`'s sibling row name go-selfupdate-lib, formerly
   go-core-lib, with its released surface at `v1.5.0`. `docs/README.md`'s
   status cells say "amended through 2026-10-03". `NOTICE` names
   `github.com/maccavelli/go-selfupdate-lib`.
2. **Later phases** read go-core-lib as go-selfupdate-lib, at the tag F10
   selects (`v1.5.0` or later). That covers Phase 0 step 4's
   "do not `require`", Phase 4's `uses:` line, and `NOTICE`. Later phases require `github.com/maccavelli/go-selfupdate-lib` at the tag F10 selects (`v1.5.0` or later), not `go-core-lib`.
3. **Step 3's checks,** as written: the tag, commit and exported packages
   against go-selfupdate-lib's source; every relative link in the changed
   files, with the planted bad link seen failing; `git diff --check`. In
   addition, `git grep go-core-lib` over `README.md`, `NOTICE` and
   `docs/architecture.md` shows only "formerly" mentions.

The execution record gains a dated Phase D entry. On 2026-10-03 this PLAN stayed
`in-progress`. Phase 0 and Phase 1 have since run; see Where execution stands.

**2026-10-04 — Phase 0 tool versions resolved.** `go` is `1.27.1`.
`go mod tidy` recorded these tool modules:

* `honnef.co/go/tools v0.8.1` — staticcheck 2026.2.1 (0.8.1). This is the
  magic-cli-remote pin (MADR 0170). Re-resolved on this date; the version
  is unchanged.
* `golang.org/x/vuln v1.8.0` — govulncheck@v1.8.0.
* `golang.org/x/exp v0.0.0-20260908205506-85c1c2202aba` — `cmd/apidiff`.
* `golang.org/x/lint v0.0.0-20241112194109-818c5a804067` — golint.

The ACP require is `github.com/coder/acp-go-sdk v0.13.5`, replaced by
`github.com/maccavelli/acp-go-sdk v0.13.6-mcr.1`. The replace is not
relative. `go-llmprovider-sdk`, `go-core-lib`, and `go-selfupdate-lib`
are not required. Root `tools.go` blank-imports the ACP module so
`go mod tidy` keeps that requirement; it is not a package from the
package map, and it adds no package directory.

`make check-cgo-off` failed on an uncommitted scratch file
`scratch_cgo_phase0.go` whose third line was `import "C"`:

```text
check-cgo-off: cgo import "C" is not allowed:
./scratch_cgo_phase0.go:3:import "C"
mingw32-make: *** [makefile:159: check-cgo-off] Error 1
```

The scratch file was deleted. On the restored tree, `make check-cgo-off`
exited 0.

**2026-10-04 — ship builds stay CGO off; `make race` stays CGO on.**
The owner approved this the same day. The shell on the build machine
keeps `CGO_ENABLED=1` for local race testing. That ambient value must
not turn cgo on for shipped builds, and a Makefile override must not
force it off for `make race`.

`CGO_ENABLED ?= 0` does not override a value already in the
environment. At `925ef0ab`, exact `make preflight` exited 2:
`check-cgo-off` refused on `CGO_ENABLED=1` before it looked for
`import "C"`. A file-level `override` plus `export` then made
preflight exit 0, and it also forced the race recipe off. The race
detector needs cgo.

`9bc93114` limits the override to `build`, `install`, and
`check-cgo-off`. It is not exported. `race` runs
`CGO_ENABLED=1 go test -race ./...`. `staticcheck` already passes
`CGO_ENABLED=0` on its tool build and on each GOOS run. Other targets
leave the ambient value alone. The phase 0 wording above is unchanged:
shipped binaries stay pure Go, and `check-cgo-off` still fails on
`import "C"`.

With `CGO_ENABLED=1` in the environment, `make -n build` prints
`CGO_ENABLED=0 go build ./...` and `make -n race` prints
`CGO_ENABLED=1 go test -race ./...`. `make preflight` exited 0.
`make check-cgo-off` exited 0. `make race` exited 0
(`? github.com/maccavelli/gobble-cli [no test files]`). A scratch file
`cgoscratch_test.go`, whose third line was `import "C"`, failed
`make check-cgo-off`:

```text
check-cgo-off: cgo import "C" is not allowed:
./cgoscratch_test.go:3:import "C"
mingw32-make: *** [makefile:162: check-cgo-off] Error 1
```

mingw32-make exited 2. The scratch file was deleted and was not
committed.

**2026-10-04 — Phase 2 terminal scope accepted from 0006.** The owner
accepted a wider terminal scope than the narrow proposal recorded in
`5353bdb897040723c63af69b24fe1ded3d1a743b`. This plan was already
`in-progress`. It is not completed. Phase 2 steps 5 and 7–13, the
out-of-phase list under Phase 2, and the Phase 2 accept checks are the
change. Step 5's `completion` bullet now names bash, zsh, fish, and
powershell only. It previously said only `` `completion` (Kong) ``.
Steps 1–4 and 6 are unchanged. Still out of Phase 2: bare session
(goose-as-default-session), the skills command, a second parser (clap,
Cobra, or fang), term aliases, man pages, hidden commands, Sigstore
update, a path-root override, date log folders and 14-day cleanup,
product commands (real acp, serve, roam, mcp, gateway, schedule,
recipe, review, local-models, plugin), and a slash registry, which is
not a Phase 2 command. Config implementation stays unspecified. The
config surface is undecided and will be either a native Kong facility
or a surface we write. The CLI library in Phase 2 remains Kong.
Go 1.27.1. TUI stays go-tui-lib. Self-update stays go-selfupdate-lib.
The Goal section is unchanged.

**2026-10-04 — repository scaffold (0007-MADR D8).** [0007-MADR-repository-scaffolding-to-fleet-standard.md](0007-MADR-repository-scaffolding-to-fleet-standard.md) brought the repository up to the fleet scaffold. This PLAN changes as follows:

* **Scope.** `.editorconfig` is struck. No fleet repository has one; gofmt, `.gitattributes` and markdownlint cover its job.
* **Phase 0 step 7.** The linter set is mcplib's full set: the 18 listed plus `unused` and `whitespace`, with the `gofmt` and `goimports` formatters. 0007-PLAN P5 added them.
* **Phase 4, CI hygiene,** in addition to the steps above:
  * the workflow declares `permissions: contents: read` at the top level; the publish job keeps its own wider block;
  * `concurrency: { group: ci-${{ github.ref }}, cancel-in-progress: true }`;
  * `actions/checkout` runs with `persist-credentials: false`;
  * on Linux, golangci-lint is installed at `v2.14.0`, and `go mod tidy -diff` runs;
  * on Linux, shellcheck `v0.11.0` is downloaded with its SHA-256 checked, and runs over `scripts/*.sh`;
  * on Linux, actionlint `v1.7.12` runs over `.github/workflows/`.

  `make preflight` already runs markdownlint and the records check, so Phase 4 adds no separate step for them.
* **Phase 4, Makefile.** `make preflight` runs on Linux and macOS since 0007-PLAN P2, which this phase's CI depends on.

**2026-10-04 — Phase 2 split with 0008-PLAN (0008-MADR D14, D16).**
[0008-PLAN-native-cli-mode.md](0008-PLAN-native-cli-mode.md) executes Phase 2 steps 1, 5 (the root and the `version`, `completion`, `config path`, `acp`-stub and default commands), 6, 7, 8, 9 and 11. Steps 2, 3 and 4 run here first, because they are 0008-PLAN P4's preconditions. Steps 10, 12 and 13 run here after 0008-PLAN P4 has created the root.

Changes to the steps as written:

* **Step 1.** `cmd/gobble/main.go` calls `cli.Main(context.Background(), …)`. `cli.Main` owns signal handling: `signal.Notify` with the per-OS list (Interrupt, SIGTERM and SIGHUP on Unix; Interrupt only on Windows), and `context.WithCancelCause` with a typed cause, so the exit code can be 128 plus the signal (0008-MADR D2, D10). SIGTERM is not registered on Windows, which never delivers it.
* **Step 4.** The rolling file sink is written from magic-cli-remote 0157-MADR's design: 10 MiB per file, 5 backups, 28 days. That record has no code to copy (0008-MADR F17).
* **Step 5.** The one-shot and session commands are one default command, `chat`, with Pi's mode resolution and `-p/--print` (0008-MADR D1). Kong is used with `kong.Name`, `kong.Writers` and an `Exit` that panics a sentinel. `kong.Parse` and `FatalIfErrorf` are never called (0008-MADR D14).
* **Step 6.** The exit table adds 128 plus the signal: 130, 129 and 143 (0008-MADR D10).
* **Step 11.** The editor is `golang.org/x/term` behind gobble's adapter (0008-MADR D6).
* **Phase 4.** `make verify-build-metadata` is implemented by 0008-PLAN P4, because creating `cmd/gobble/main.go` trips the Phase 0 placeholder (`Makefile`, `verify-build-metadata`). Phase 4 keeps the CI wiring.
* **Phase 0 step 7.** The lint set gains 0008-MADR D18's rules (sloglint, depguard, forbidigo, modernize, intrange, copyloopvar, usestdlibvars, perfsprint, usetesting, noctx, contextcheck, fatcontext).
* **archtest rule 9.** Only `internal/cli/...` may import Kong. Added by 0008-PLAN P1.
