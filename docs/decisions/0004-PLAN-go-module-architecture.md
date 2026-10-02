---
status: in-progress
date: 2026-10-02
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

### Phase D — current sibling-source documentation (before Phase 0)

This phase may be approved and committed separately from the Go scaffold.
It changes documentation only. It does not add a module requirement or claim
that pigo code exists.

1. Reconcile `README.md`, `docs/README.md` and `docs/architecture.md` with
   the 2026-10-02 amendments of this MADR and
   [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md).
   `architecture.md` keeps describing the current documentation-only pigo
   tree; it identifies sibling code as a planned dependency, with version
   evidence and no invented package in pigo.
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
actual present surfaces; none says pigo imports them yet; links resolve;
`git diff --check` is clean. Commit this documentation phase after its
checks. Phases 0–5 retain their separate approval and execution order.

### Phase 0 — module and toolchain

1. `go mod init github.com/maccavelli/pi-go`. Set `go 1.27.1`.
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
     [0005-PLAN](0005-PLAN-v1-feature-scope.md) F10 (`pigo update`) and
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
   unconvert, unparam.
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

### Phase 1 — package skeleton and contracts

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
   * `pigo`: `New`, `Option`, `Runtime` (methods return
     `errors.ErrUnsupported` until later phases).

   Every wire struct uses `encoding/json/v2` tags. It has `omitzero` where
   a field is optional and a `json:",unknown"` `jsontext.Value` where the
   MADR requires forward compatibility.
3. Write `internal/archtest`. It runs `go list -deps -json ./...` and
   enforces the eight import-boundary rules in 0004-MADR, with a table
   test per rule. Rules 4 and 8 have no edges to fail on until the
   first import of `go-llmprovider-sdk` (0002-PLAN Phase 3) and
   `go-core-lib` (0005-PLAN F10); the table still names them.
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
   * Do **not** import `llmprovider/llmtest` here. That package's `Run`
     and `Fake` exercise the SDK contract; pigo's `Script` exercises the
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
binaries** named `ExactAssetName("pigo", platform)` =
`pigo-<goos>-<goarch>` with `.exe` on windows. It does not select
`.tar.gz` / `.zip` archives. The reusable workflow refuses any tag
that is not a strict `vMAJOR.MINOR.PATCH` (so `v1.0.0-rc.1` cannot
publish through it).

1. `.github/workflows/ci.yml`:
   * matrix ubuntu, macos, windows;
   * `actions/setup-go` with `go-version-file: go.mod`;
   * steps `make preflight`, `make race` (not on windows arm), and
     `make archtest`;
   * golden-transcript artifacts are uploaded on failure.
2. A **build** job on tags matching `v*` produces the staged set into
   an artifact. A **publish** job runs only on a strict stable tag and
   calls the reusable workflow. Record the pin in the workflow file:

   ```yaml
   uses: maccavelli/go-core-lib/.github/workflows/publish-selfupdate-release.yml@96b30961180671ab3697585951219001ecbb1c90 # go-core-lib v1.1.0
   with:
     artifact-name: pigo-release
     products-json: '["pigo"]'
     platforms-json: '[{"os":"darwin","arch":"amd64"},{"os":"darwin","arch":"arm64"},{"os":"linux","arch":"amd64"},{"os":"linux","arch":"arm64"},{"os":"windows","arch":"amd64"},{"os":"windows","arch":"arm64"}]'
     extra-assets-json: '["pigo.spdx.json"]'
   ```

   The staged artifact contains exactly:
   * `pigo-darwin-amd64`, `pigo-darwin-arm64`, `pigo-linux-amd64`,
     `pigo-linux-arm64`, `pigo-windows-amd64.exe`,
     `pigo-windows-arm64.exe`;
   * `SHA256SUMS` (the same parser the client uses;
     `selfupdate/testdata/manifest-parity/` is the fixture set);
   * `pigo.spdx.json` as the extra asset.
   Each binary is `CGO_ENABLED=0 -trimpath -tags netgo,osusergo` with
   the 0003-MADR ldflags. The reusable workflow attests and publishes;
   pigo does not call `softprops/action-gh-release` itself.
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
  planted `pigo-linux-amd64.tar.gz` in a scratch `dist/` is **not**
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
./dist/pigo-$(go env GOOS)-$(go env GOARCH) version --json   # append .exe on windows
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
`git diff --check` passed. Pigo has no `go.mod`, so no Go test was run.
The documentation phase is committed with this record. Phases 0–5 remain
proposed and require separate approval.

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

`pigo update` stays 0005-PLAN F10. Native token streaming stays an SDK
capability pigo consumes when it exists.

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
  * go-llmprovider-sdk: development pin `940fee0`. No pigo tag ships on a pseudo-version (MADR decision 1).
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
* **Phase 2, step 6.** The exit-code table adds `10` for "update available" (`pigo update --check` only). Every other selfupdate error maps to `1`, except two:
  * contradictory update flags are `2` (usage);
  * a cancelled update is `4`, or `130` on SIGINT.
* **Phase 4.** Read the original steps with these additions.
  * **Precondition, recorded with evidence before the first tag** (an owner action, not this plan's): on `maccavelli/pi-go`,
    * "immutable releases" is enabled;
    * a tag ruleset restricts `v*` to the owner, with 2FA.

    Without immutable releases the reusable workflow publishes, waits 120 s, fails, and leaves a mutable release that every client refuses (`ErrMutableRelease`).
  * **The calling job declares `permissions: {contents: write, id-token: write, attestations: write}`**, which the snippet in step 2 omitted.
  * **The SBOM generator is named, and pinned by version and checksum, in this PLAN before step 2 lands.** `pigo.spdx.json` is an extra asset. It is **not** in `SHA256SUMS`. Optionally it is attested with `actions/attest-sbom`.
  * **Tags.** The build job runs on strict `vX.Y.Z` tags only, with no release-candidate tags until go-core-lib `v1.3.0` channels are adopted by a later amendment. `make verify-build-metadata` asserts `buildKind=release` on tag builds, and is shown failing on a scratch build stamped `local`.
  * **`make release-dry-run` runs go-core-lib's `scripts/verify-selfupdate-release.sh`** fetched at the pinned SHA, instead of an inlined copy of the name rule. The planted-archive accept stays.
  * **Actions in pigo's own workflows are pinned by commit SHA**, as go-core-lib's are.
  * **Users verify releases** with `gh release verify` and `gh attestation verify`. `docs/guides/developing.md` (Phase 5) documents the commands.

**2026-10-02 — sibling-source refresh and shared TUI workspace.** Adds
documentation-only Phase D, available before Phase 0. The current evidence
is go-llmprovider-sdk `67fc56e` (no tag, with an uncommitted PLAN edit),
go-core-lib `v1.3.0` at `f97c681`, and go-tui-lib `v0.1.0` at `5c57806`.
Phase 0 still imports none of them. At its later Phase 4, resolve the
go-core-lib workflow to the then-selected tag's peeled commit; `v1.3.0`
currently supplies channels, while pigo's release workflow remains
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
those record corrections. No sibling repository or pigo source code is
changed by this phase.
