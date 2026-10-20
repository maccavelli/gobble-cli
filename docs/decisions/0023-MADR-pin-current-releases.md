---
status: accepted
date: 2026-10-19
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Pin the current releases: acp-go-sdk2 v1.16.0, go-llmprovider-sdk v1.15.0, the x modules and the tool modules

## Context and Problem Statement

The owner asked on 2026-10-19 to "assess the codebase and current docs. evaluate the acp-go-sdk2, go-llmprovider-sdk,
go-selfupdate-lib, dependencies and ensure we are pinning the current releases." Each fact below was measured that
day against `main` `30b8a4b`, by the command or file named.

**What gobble pins, and what is current** (`go.mod`; `GOFLAGS=-mod=mod go list -m -u all`; `go list -m -versions`):

| Module | Pinned | Current | Role |
| :--- | :--- | :--- | :--- |
| `github.com/maccavelli/acp-go-sdk2` | `v1.14.0` | `v1.16.0` | direct |
| `github.com/maccavelli/go-llmprovider-sdk` | `v1.13.0` | `v1.15.0` | direct |
| `github.com/maccavelli/go-selfupdate-lib` | `v1.15.2` | `v1.15.2` | direct, and CI's release workflows at `436a7f59` |
| `github.com/alecthomas/kong` | `v1.16.1` | `v1.16.1` | direct |
| `github.com/modelcontextprotocol/go-sdk` | `v1.8.0` | `v1.8.0` | direct |
| `github.com/google/jsonschema-go` | `v0.4.3` | `v0.4.3` | direct |
| `go.yaml.in/yaml/v3` | `v3.0.5` | `v3.0.5` | direct |
| `golang.org/x/sys`, `x/term`, `x/text` | `v0.48.0`, `v0.46.0`, `v0.42.0` | `v0.49.0`, `v0.47.0`, `v0.43.0` | direct |
| `golang.org/x/vuln` (govulncheck) | `v1.8.0` | `v1.9.0` | `tool` directive |
| `honnef.co/go/tools` (staticcheck) | `v0.8.1` | `v0.8.2` | `tool` directive |
| `golang.org/x/exp` (apidiff) | `v0.0.0-20260908205506-85c1c2202aba` | `v0.0.0-20261014143641-f0251eb4aa3e` | `tool` directive |
| `golang.org/x/lint` (golint) | `v0.0.0-20241112194109-818c5a804067` | the same | `tool` directive |
| golangci-lint | `v2.14.0` (`Makefile:121-124`, `.github/workflows/ci.yml:53`) | `v2.15.0` | installed by `go install` |
| Go | `go1.27.2` (`go.mod:3`) | `go1.27.2` | toolchain |
| `actions/checkout`, `actions/setup-go` | `v7.0.1`, `v7.0.0` | the same | CI, by commit |
| `actions/upload-artifact` | `v7.0.1` (`ci.yml:64`, `:136`) | `v7.0.2` | CI, by commit |

`go get` of the two SDKs, the three x modules and the three tool modules also moves the indirect `golang.org/x/mod`
`v0.41.0` → `v0.42.0`, `x/sync` `v0.23.0` → `v0.24.0`, `x/tools` `v0.50.0` → `v0.52.0` and `x/telemetry` to
`v0.0.0-20261010193011-f71dc92da45e`, and adds no module (a scratch clone's `go.mod` diff: 12 lines replaced).

**What the checks say** (0019-MADR D14's `make check-libs`, on a scratch clone at `30b8a4b`):

- **acp-go-sdk2 `v1.16.0`:** get and tidy, build, vet for linux, darwin and windows, test, apidiff, govulncheck and the
  advisories all `ok`.
- **go-llmprovider-sdk `v1.15.0`:** every check `ok` but the tests: `TestListFullCatalog`
  (`llm/provider/list_test.go:90-104`) gets OpenAI's curated listing with `gpt-5-mini` before `gpt-5-nano`, where it
  pins `gpt-5-nano` first. Every other package passes.
- **go-selfupdate-lib:** pinned at its latest tag; every check `ok`.

With every pin above moved in a scratch clone, `go vet ./...` is clean, `make preflight` passes on macOS
(golangci-lint `0 issues.`, staticcheck for three systems, govulncheck "No vulnerabilities found"), and `go test
./...` fails only `TestListFullCatalog`, as above.

**What changes in the SDKs** (their own release notes and READMEs, in the module cache):

- **acp-go-sdk2** `v1.15.0`, `v1.15.1`, `v1.15.2` and `v1.16.0` (`docs/reports/0019-REPORT-release-notes-v1-15-*.md`,
  `-v1-16-0.md`): the ACP schema stays `1.25.0`; the exported API grows by three entries and loses none.
  - `v1.15.2`: a connection whose input ends cleanly answers the requests it already read, for up to five seconds,
    before `Done()` closes (its issue 8, which gobble reported, `0019-REPORT-release-notes-v1-15-2.md`). gobble's
    `Serve` waits on `conn.Done()` (`acpserver/serve.go:39-41`), so `gobble acp` then answers a request piped to it and
    followed by end of file, which it drops today. Subagent updates are gated on the `subagents` capability, which
    gobble neither sends nor reads (no `Subagent` or `SessionMessage` in `acpserver` or `acpclient`). `Prompt` no longer
    cancels a prompt it never sent; gobble's `acpclient` calls it at `acpclient/conn.go:282`.
  - `v1.15.0`: `UnstableMcpError.Data` keeps an explicit `null`; gobble uses no `McpError`.
  - `v1.16.0`: fewer allocations, the same bytes on the wire (`TestEncodingsUnchanged`), and a failed cancel of a
    prompt logged at Debug.
- **go-llmprovider-sdk** `v1.14.0` and `v1.15.0` (`README.md:286-330`): `v1.14.0` decides each model's thinking,
  effort and output limit from its cached facts and ranks every provider's band from them (its 0041-MADR); Ollama
  leaves out models that cannot write text (0042-MADR); the exported API only grows. `v1.15.0` adds OpenAI's ChatGPT-plan
  sign-in (0043-MADR) and `WithRetryNotify` (0044-MADR), closing its issues 2 and 3. Its retry still buffers a stream
  (`llmprovider/retry.go:79-80`).

**What else holds a version:** `docs/architecture.md`, `README.md` and `docs/README.md` name the pins; 0020-MADR D4
starts a new session on the live listing's first model, so a provider's order is gobble's default model.

## Decision Drivers

- The owner's ask: current releases, pinned.
- AGENTS.md: a requirement changes with the code that needs it; `go mod tidy` clean at every commit; 0019-MADR D14's
  checks before each pin.
- An upgrade that changes behaviour is recorded and tested, never taken silently.
- 0007-MADR D5 names golangci-lint `v2.14.0` "the fleet's current pin": a change there is the fleet's as much as
  gobble's.

## Considered Options

* **A. Pin every current release in one change,** with the tests the upgrade needs.
* **B. Pin the two SDKs only,** and leave the x and tool modules and CI's actions.
* **C. Pin nothing now,** and take each when a feature needs it.

## Decision Outcome

Chosen option: "A. Pin every current release in one change", because every pin above passes the checks with one test
to bring up to date, and pinning them apart would run the same gates several times for the same result. *Proposed:
the owner decides at review.* *(2026-10-19: accepted by the owner, "Approve 0023", with D3 decided: golangci-lint stays at the fleet's `v2.14.0`; 0023 runs before 0005-PLAN F6-2.)*

- **D1. The pins:** acp-go-sdk2 `v1.16.0`, go-llmprovider-sdk `v1.15.0`, `golang.org/x/sys` `v0.49.0`, `x/term`
  `v0.47.0`, `x/text` `v0.43.0`; the tool modules `golang.org/x/vuln` `v1.9.0`, `honnef.co/go/tools` `v0.8.2` and
  `golang.org/x/exp` at its latest pseudo-version, and the indirect moves `go mod tidy` makes. go-selfupdate-lib,
  Kong, the MCP SDK, jsonschema-go, yaml, golint and Go stay, being current.
- **D2. CI's `actions/upload-artifact` moves to `v7.0.2`,** by its commit, in both places.
- **D3. golangci-lint `v2.15.0` is proposed separately, for the owner's choice:** 0007-MADR D5 makes `v2.14.0` the
  fleet's pin, so moving it here amends that record and leaves gobble ahead of the fleet. This record moves it only
  if the owner says so at review; otherwise it stays, named. *(2026-10-19, the owner's choice: it stays at `v2.14.0`, with the fleet.)*
- **D4. `gobble acp` answers a request piped before end of file** (acp-go-sdk2 issue 8, fixed in `v1.15.2`): a test
  runs the agent over a pipe that closes after one request and reads its answer. No gobble code changes, since the fix
  is the SDK's default; the test fails at `v1.14.0`.
- *(2026-10-19, 0023-PLAN deviation 1, the owner's choice: from go-llmprovider-sdk `v1.14.0` every provider's listing but Kilo's and Ollama's also fetches the model metadata document, `models.opencode.ai/api.json`, with no credential, cached in-process for 10 minutes; gobble takes it, as 0020-MADR's "model metadata on, once per process" covers, and `TestListFullCatalog` requires the document to carry no `Authorization`. The facts above, which said one test failed on its order, missed this second check, which the order failure hid.)*
- **D5. The listing's order is the SDK's:** `TestListFullCatalog` takes `v1.15.0`'s order once the PLAN shows, from the
  SDK's source, which change makes it. 0020-MADR D4 is unchanged: a new session starts on the listing's first model,
  which `v1.14.0`'s ranking may now choose differently for a provider; that is said in the README's notes for the
  change.
- **D6. Nothing else of the SDKs is adopted here.** The ChatGPT-plan sign-in is 0005-PLAN F4-3d's; the retry notices
  are the F5-1 sub-phase its decision 5 left for the SDK's callback; native deferral (go-llmprovider-sdk 0040-MADR)
  and the cached facts are later records'. Each is made executable on the owner's word.
- **D7. The records and docs follow:** 0019-MADR and 0020-MADR get dated notes that the pins moved; `docs/architecture.md`,
  `README.md` and `docs/README.md` name the new versions; the memory of issue 8 is closed by D4's test.

### Consequences

* Good, because every direct requirement and tool is at its current release, and `make check-libs` reports nothing
  to take.
* Good, because `gobble acp` answers a piped request, which a script or a test harness depends on.
* Neutral, because acp-go-sdk2's encoder changes wholesale (3754 lines added and 2314 removed in six of its files, nearly all in `types_gen.go`, by its `v1.16.0` notes), with
  the wire bytes pinned by its own golden test; gobble's ACP goldens check them again.
* Bad, because a new session's default model can change for a provider whose ranking `v1.14.0` re-orders, without a
  gobble change to show for it.
* Bad, because golangci-lint stays one minor behind unless D3 is taken.

### Confirmation

The PLAN's gates on macOS, the Linux machine and the Windows laptop: `make check-libs` with nothing to take; D4's test
seen failing at `v1.14.0` and passing at `v1.16.0`; `TestListFullCatalog` failing on the old order with the new SDK;
`go test ./...`, the race detector and `make preflight`.

## Pros and Cons of the Options

### A. Pin every current release in one change

* Good, because one set of gates covers every move, and the scratch clone already passes them but one test.
* Bad, because a regression is harder to bisect to one module; the PLAN's steps keep each module's move in its own
  check before the commit.

### B. Pin the two SDKs only

* Good, because the change is smaller.
* Bad, because the x and tool modules stay behind for no measured reason, and the owner asked for current releases.

### C. Pin nothing now

* Good, because nothing moves.
* Bad, because `gobble acp` keeps dropping a piped request, which the SDK now fixes, and every later feature's record
  carries the upgrade as well.

## More Information

- [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md) D14: the checks.
- [0020-MADR-full-model-catalog-listing.md](0020-MADR-full-model-catalog-listing.md) D4: the default model.
- [0007-MADR-repository-scaffolding-to-fleet-standard.md](0007-MADR-repository-scaffolding-to-fleet-standard.md) D5: golangci-lint's fleet pin.
- [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F4-3d, F5-1 decision 5, and the entry "The docs brought up to date".
- acp-go-sdk2 issues 7, 8 and 9; go-llmprovider-sdk issues 2 and 3, all closed.

## Amendment (2026-10-19): acp-go-sdk2 `v1.16.3`, go-llmprovider-sdk `v1.17.0`, go-selfupdate-lib `v1.17.0`

*Proposed: the owner decides at review.* The owner asked on 2026-10-19, after `fa28838`, to "ensure we are pinning the
most updated releases of acp-go-sdk2, go-llmprovider-sdk and go-selfupdate-lib". Each fact below was measured that
day against `main` `fa28838`.

### Context

**What is pinned, and what is newest** (`go.mod:15-17`; `GOPROXY=direct go list -m -versions`; `gh release list`;
`gh run list` on each library):

| Module | Pinned | Newest tag | The tag's state |
| :--- | :--- | :--- | :--- |
| `acp-go-sdk2` | `v1.16.0` | `v1.16.3` (also `v1.16.1`, `v1.16.2`) | GitHub release "Latest"; the tag's CI passed |
| `go-llmprovider-sdk` | `v1.15.0` | `v1.17.0` (also `v1.16.0`) | tagged 01:46 UTC on 2026-10-20; the tag's CI run in progress at measurement and no GitHub release yet, "Latest" being `v1.16.0` |
| `go-selfupdate-lib` | `v1.15.2` | `v1.17.0` (also `v1.16.0`), at `adc2c757d7e2fa9ed7e5eb5ff33a076eba34f4bd` | tags only, as at `v1.15.2`; the tag's CI passed |

CI's release jobs call go-selfupdate-lib's `build-selfupdate-release.yml` and `publish-selfupdate-release.yml` at
`436a7f5976a6815491c7384527fef90869574a1b # v1.15.2` (`.github/workflows/ci.yml:145`, `:157`), pinned with `go.mod`
since 0019-PLAN P2.

**What the checks say** (0019-MADR D14's `scripts/check_libs.py`, which clones the committed repository):
acp-go-sdk2 `v1.16.3` and go-llmprovider-sdk `v1.17.0`: get and tidy, version, modules, build, vet for linux, darwin
and windows, test, api, govulncheck and advisories all `ok`. go-selfupdate-lib `v1.17.0`, run alone (`--lib`): every
check `ok`, "0 failing checks". Run together, the script stopped at go-selfupdate-lib's api step with
`FileExistsError: … api-v1.17.0`: `export_api` names its scratch module and its export by version alone
(`scripts/check_libs.py:182-190`), so two libraries checked at the same version collide. No run met the case before.

**What changes** (each library's release notes, CHANGELOG and exported API at the tags, in the module cache):

- **acp-go-sdk2** (ACP schema still `1.25.0`; the exported API the same): `v1.16.1` (its 0023-MADR) ends running
  notification handlers' contexts at `Close()`, with cause `ErrConnectionClosed`, where they ended up to 5 s later;
  gobble's notification handlers do not read their context (`acpclient/conn.go:373`, `acpserver/agent.go:906`), and
  `acpclient` closes input before `Close()` (`acpclient/conn.go:173-191`). `v1.16.2` lowers its Go floor to 1.26.9
  with `toolchain go1.27.2`; gobble stays at `go 1.27.2`. `v1.16.3` changes documentation only.
- **go-llmprovider-sdk** (no new module; `go 1.27.2`): `v1.16.0` (its 0048-MADR) adds `catalog.Cache`,
  `catalog.WithCache` and `providers.DefaultWith`, the default of one cache a process unchanged; gobble uses
  `catalog.List` and `providers.Default()` (`llm/provider/list.go:54`, `ambient.go`). `v1.17.0` (its 0049-MADR) gives
  its errors sentinels: an OAuth `access_denied` or an expired device code matches `ErrAuthFailure`; a provider it does
  not serve, `ErrUnsupported`; a nil `*StaticToken` or a foreign `DeviceLogin.Wait`, `ErrInvalidRequest` in place of a
  panic. gobble's logins wrap the SDK's error in `mapError` (`llm/provider/login.go:106`, `:119`, `:132`), whose table
  maps `ErrAuthFailure` to `llm.ErrAuth` (`llm/provider/provider.go:460-469`); gobble's tests already require a
  denied login to be `llm.ErrAuth` (`llm/provider/login_test.go:242`, `:333`), and pass at `v1.17.0`. A denial's
  message may now carry the SDK's "authentication failed" as well. Retry still buffers a stream
  (`llmprovider/retry.go:82`), so 0005-PLAN F4-1b still waits.
- **go-selfupdate-lib** (no new module; Go floor lowered to 1.26.9; `v1.0.0`–`v1.4.1` retracted): `v1.16.0` (its
  0025-MADR) adds `NewGitHubUpdater`, `HTTPError`, `service.CommandError`, `Clock`, `NewVersionProberWith`,
  `NewSlogReporter`, `selfupdatetest.VerifyReleaseSource` and an opt-in `verify/signednote`; redacts `Credential` and
  `GitHubOptions` when printed; fixes a Task Scheduler data race; reserves the asset name `SHA256SUMS.note`. `v1.17.0`
  (its 0026-MADR) adds the `before_uninstall` hook and the installers' `--force`. gobble imports `buildinfo`
  (`internal/buildinfo/buildinfo.go:9`) and, in a test, `selfupdate` and `releasespec`
  (`internal/cli/release_test.go:8-9`); its release spec declares no hook. Of the two workflows gobble calls,
  `build-selfupdate-release.yml` is unchanged from `436a7f59` to `adc2c757`; `publish-selfupdate-release.yml` gains an
  opt-in `sign` job (input default false, `environment: selfupdate-signing`), and the publish job now `needs: sign`,
  running when it succeeded or was skipped without `sign`. gobble passes no `sign`. The publish job runs only on a
  tag, so `main`'s rehearsal does not reach it.

### Decision

Chosen: pin the three newest tags, with the workflows, and fix the check that found the collision.

- **A1. The pins:** acp-go-sdk2 `v1.16.3`, go-llmprovider-sdk `v1.17.0`, go-selfupdate-lib `v1.17.0`, and the
  indirect moves `go mod tidy` makes; nothing else of D1 moves.
- **A2. The workflows follow the library:** both `uses:` lines at `adc2c757d7e2fa9ed7e5eb5ff33a076eba34f4bd # v1.17.0`.
- **A3. go-llmprovider-sdk `v1.17.0` only once its tag's CI has passed and its GitHub release is published;** until
  then the plan stops and asks, the alternative being `v1.16.0`, its "Latest" release.
- **A4. `check-libs` names its API exports by library and version,** so two libraries at one version do not collide;
  its `--self-test` gains the case, seen failing with today's names.
- **A5. Nothing new is adopted** (D6 holds): not go-selfupdate-lib's signing, hook or updater, nor the provider SDK's
  `catalog.Cache`. The login errors' new sentinels are taken as they come, `mapError` unchanged.
- **A6. The records and docs follow** as D7 says: `docs/architecture.md`, `README.md`, `docs/README.md`, and dated notes
  in 0019-MADR D2 and D14.

### Consequences

* Good, because the three libraries are at their newest tags and `check-libs` checks all three in one run again.
* Good, because a denied or expired provider login is classified by the SDK, not by gobble's guess at its message.
* Neutral, because the publish workflow's new `sign` job is first exercised by gobble's next tag; it is skipped there,
  and the job it gates ran as before in go-selfupdate-lib's own CI.
* Bad, because go-llmprovider-sdk `v1.17.0` is hours old; A3 holds the pin until its release is published.

### Options not chosen

* **Pin acp-go-sdk2 and go-selfupdate-lib now and go-llmprovider-sdk `v1.16.0`,** its published release: good, because
  nothing waits; bad, because the provider SDK stays one tag behind, which the owner asked against. Kept as A3's
  fallback.
* **Leave `check-libs` and check the libraries one at a time:** bad, because `make check-libs`, the documented command,
  fails on the next run.
