---
status: observation
date: 2026-10-11
subject: the current state of go-llmprovider-sdk, go-selfupdate-lib and acp-go-sdk2, when and in what order gobble takes each pinned to its current release, and whether to adopt them and start compatibility and integration work now
examines: "gobble-cli at 6a8b9d8 (main); go-llmprovider-sdk v1.7.1 (0adc791) and its main at f9834e0; go-selfupdate-lib v1.15.1 (fa74db4) and its main at d396052; acp-go-sdk2 v0.14.0 (3588f8a) and its main at 41b8840"
---

# REPORT 0015 — The three sibling libraries: current state, pins, order, and whether to adopt them now

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

This report records findings and a recommendation. It decides nothing. The owner asked, on 2026-10-11:

- for go-llmprovider-sdk, go-selfupdate-lib and acp-go-sdk2 to be assessed at their current versions: which gobble should implement, when, in what order, pinned to current, with every assertion proved;
- then for a recommendation on whether to adopt them now, and start compatibility assessments and code optimizations to integrate them.

How to read it:

- **Tags:** each finding is tagged **(R)** when read from source, a record or an API, and **(M)** when measured by running something.
- **The claims check:** every claim in the evidence index was checked by a script, and that script was seen failing with every claim inverted ("How the claims were checked").
- **When:** the measurements ran on 2026-10-11 between about 13:20 and 21:10 UTC, and the libraries moved throughout.
  - **Tags cut during that time:** go-llmprovider-sdk `v1.7.1` at 15:05, and go-selfupdate-lib `v1.15.0` at 15:15 and `v1.15.1` at 18:22.
  - **The branches** kept moving after those tags. This report freezes them at the commits named in its front matter, as seen at about 20:55 UTC. A later commit on any of them is not covered.

## Summary

| Library | Current tag | gobble pins | Default branch past the tag | What gobble uses it for | Where its work stands |
| :--- | :--- | :--- | :--- | :--- | :--- |
| go-llmprovider-sdk | `v1.7.1` | `v1.2.1` | 12 commits: forced tool choice fallbacks and `Response.Warnings`, additions only | every model call, in `llm/provider` | 0005-PLAN F4, the current phase |
| go-selfupdate-lib | `v1.15.1` | `v1.9.0` | 4 commits, documentation only | build identity and the release workflows; `gobble update` at 1.0 | identity and workflows done; `gobble update` is F10, the last 1.0 phase |
| acp-go-sdk2 | `v0.14.0`, its only tag | `v0.14.0` | 23 commits: stricter decoding and validation, and API breaks gobble does not use | the ACP server and client | done (0014-PLAN, F2, F3); current |

- **All three move to their current tags with no code change.** This was measured on scratch clones of gobble's `main`, with each library at its current tag and with all three together. (M)
  - `go mod tidy` changes the requirement lines and their `go.sum` hashes, and nothing else.
  - `go build`, `go vet` for linux, darwin and windows, and `go test -count=1 ./...` (29 packages) pass.
  - So does `govulncheck` on the combined set.
- **Each upgrade only adds API.** apidiff reports no incompatible change. go-llmprovider-sdk `v1.2.1`→`v1.7.1` has 15 compatible changes, and go-selfupdate-lib `v1.9.0`→`v1.15.1` has 30. The release workflows' inputs are unchanged. (M)
- **go-llmprovider-sdk `v1.7.1` changes what gobble offers:** four providers' default model, and the removal of ids OpenAI shuts down on 2026-10-23 (F2). (R, M)
- **acp-go-sdk2's next minor already breaks one gobble test.** At its default branch, an unscoped elicitation no longer encodes, and `acpclient`'s `TestElicitationNotFound` fails. The fix is in the test's data, not in gobble's code (F3). (M)
- **The integration survey found no code in gobble that duplicates a library.** It found three optimizations already planned, and two corrections to F4-3's entry before it is built (F5). (R)

**Recommendation (F7):** yes, adopt now, in this order. Start the compatibility assessments now, because they are cheap and two of them have already found defects. Limit the code changes to the items below.

1. **Now, as a pin refresh with no Go change:** go-selfupdate-lib `v1.15.1`.
2. **Now, as F4-1a:** go-llmprovider-sdk `v1.7.1`, with the error flag and a true `NativeStreaming`. Then 0012-PLAN P7, the strict flag.
3. **Now, as records only:** amend F4-3's entry for the two compatibility findings, and amend F4-2's for go-llmprovider-sdk `v1.8.0`.
4. **When tagged:** F4-2 on `v1.8.0`'s `ModelInfo`; F4-1b on a streaming `WithRetry`; acp-go-sdk2's next minor, with the elicitation test fixed and an interoperability check.
5. **At F10:** `gobble update`, against the newest go-selfupdate-lib.

**Not now:** speculative refactors toward unreleased APIs, or a gobble catalog resolver that `v1.8.0` would replace.

## Scope and method

- **Versions and dates:** `go list -m -versions` through the module proxy; the proxy's `.info` files; GitHub's git API, with each annotated tag peeled to its commit (`q400_peel.py`). (R)
- **Unreleased work:** GitHub's compare API from each current tag to the default branch (`q392_unreleased.py`), run at about 13:25, 15:00, 16:05, 20:50 and 20:55 UTC. Each run's output is kept. (R)
  - **Records:** the ones the runs found were fetched (`q402_new_records.py`).
  - **acp-go-sdk2's exported surface:** every function, method, type and struct-field type of its root package was compared between the tag and a ref (`q406_acp_surface.py`), as were the `types_gen.go` counts (`q394_acp_head_api.py`, `q395_acp_fields.py`).
  - **go-llmprovider-sdk's curated lists** were compared across versions (`q403_static_first.py`).
- **API compatibility:** gobble's own `apidiff` tool (`go tool apidiff -m`), over export data written in each module's cache directory (`q396_apidiff.py`). (M)
- **Workflows:** the `workflow_call` inputs of go-selfupdate-lib's two reusable workflows at `v1.9.0`, `v1.14.0`, `v1.15.0` and `v1.15.1`, against the `with:` keys of gobble's `ci.yml` (`q397_workflow_inputs.py`). (R)
- **Build and test:** fresh clones of gobble's `main` in the session scratchpad, one per requirement set (`q398_experiments.py`). The real tree was not touched. (M) Each clone runs:
  - `go get`, then `go mod tidy`, and a check that the resolved version is the one asked for;
  - `go build ./...`;
  - `go vet ./...` for linux/amd64, darwin/arm64 and windows/amd64, with cgo off;
  - `go test -count=1 ./...`;
  - for the combined sets, `go tool govulncheck ./...`.
- **The integration survey:** gobble's code at each library's seam, read against the library's exported API (`q407_acp_helper_use.py` counts acp-go-sdk2's helpers in use). (R)
- **Records:** gobble's 0004, 0005 and 0012 pairs and 0014-REPORT, and each library's own decision records. (R)
- **The owner's local clone of acp-go-sdk2,** made during the assessment, read with read-only git commands and not fetched (`q410_clone_evidence.py`). (R)
  - It is on `main` at `f94f253`, with no uncommitted change.
  - Its one commit past this report's snapshot, `41b8840`, touches `.github/workflows/ci.yml` and a PLAN only, so the snapshot's findings hold for it.

The scripts named here live in the session scratchpad and are not committed.

## What exists

### go-llmprovider-sdk (R, M)

- **Tags:** 13, `v1.0.0` to `v1.7.1`. Five, `v1.4.0` to `v1.7.1`, are dated 2026-10-09 to 2026-10-11 by the proxy.
  - `v1.7.1` peels to `0adc791d8ce376d81251d2943ddcc34073095ac5`, tagged 2026-10-11T15:05:00Z.
  - `v1.7.0` peels to `93f3b5331c62662cfd5a93b632da89a05573248e`, tagged 13:13:50Z the same day.
- **Module:** `go 1.27.2`; requires `golang.org/x/term v0.43.0`, and `golang.org/x/sys v0.47.0` indirectly; Apache-2.0.
- **In gobble:**
  - pinned `v1.2.1` (`go.mod:16`) and imported by `llm/provider` alone (`docs/architecture.md:90`);
  - it turns the metadata fetch off (`llm/provider/provider.go:44`);
  - it takes each provider's default model as `StaticModels[0]`, and lists `StaticModels` as its model list (`llm/provider/ambient.go:33-34`, `:74`).
- **`v1.2.1`→`v1.7.1`:**
  - 15 additions, among them `Tool.Strict`, `FunctionCallOutputItem.IsError`, `catalog.LookupMetadataWith` and `CachedMetadataWith`, and `llmtest` fields for streams and the two flags;
  - `v1.7.0`→`v1.7.1` changes no exported API;
  - the curated lists are the same at `v1.2.1` and `v1.7.0`, and `v1.7.1` rebuilds them (F2).
- **Still missing at `v1.7.1`:**
  - model limits and cost: the catalog package has no `ContextWindow`, `MaxOutputTokens` or cost field, and `Metadata` exports only `ReasoningEfforts`, `InterleavedField` and `NPM`;
  - a keyring, image input, and a cache-write count in `Usage`;
  - a stream that retries.
- **Default branch at `f9834e0`, 12 commits past `v1.7.1`:**
  - forced tool choice: a refusing Claude model degrades a forced choice, learns the refusal per model, and can route the forced tool through structured outputs;
  - `Response.Warnings` reports the degradation.
  - From `v1.7.1` to the commit before `f9834e0`, apidiff lists 12 additions and nothing incompatible: `Response.Warnings`, `Warning`, `WarningKind`, the `ForcedChoice…` values, and `WithForcedChoiceFallback` and `WithModelForcedChoice`.
  - gobble at `f9834e0` builds, vets and passes its 29 test packages.
- **Recorded next:** `0033-MADR-public-model-metadata-and-current-catalogs.md`, accepted 2026-10-11, released with `v1.7.1`. Its catalog-refresh PLAN shipped as `v1.7.1`; its metadata PLAN, for `v1.8.0`, is `proposed`.

### go-selfupdate-lib (R, M)

- **Tags:** 24, `v1.0.0` to `v1.15.1`.
  - `v1.0.0` declares its module path as `github.com/maccavelli/go-core-lib`, the name before the 2026-10-02 rename, so `go get` of it under the new path fails.
  - Ten tags, `v1.10.0` to `v1.15.1`, are dated 2026-10-07 to 2026-10-11 by the proxy.
  - `v1.15.1` peels to `fa74db4b292baaa14df6e057886102cc712bbeeb`, tagged 2026-10-11T18:22:21Z. The default branch, at `d396052`, is 4 documentation commits past it.
- **Module:** `go 1.27.1` with `toolchain go1.27.2`; requires `golang.org/x/mod v0.40.0`, `golang.org/x/sys v0.47.0` and `golang.org/x/term v0.43.0`; Apache-2.0.
- **In gobble:**
  - pinned `v1.9.0` (`go.mod:17`);
  - `internal/buildinfo` imports its `buildinfo` (`internal/buildinfo/buildinfo.go:9`);
  - the tests import `selfupdate`, `selfupdatetest` and `releasespec`, and `TestReleaseSpec` parses gobble's release spec with `releasespec.Parse` (`internal/cli/release_test.go:9`, `:22`);
  - `ci.yml` calls the library's build and publish workflows at `39b12945fad311174252038745f5a88f71c4c66c # v1.9.0` (`.github/workflows/ci.yml:145`, `:157`).
- **`v1.9.0`→`v1.15.1`:** eight decision records, 0014 to 0022, and 30 API additions, among them `releasespec.Spec.Installer`, `KeptBackups`, `ErrNoRelease`, `ErrBatchFile` and two new packages. The records cover:
  - shared installer templates;
  - a third debugging pass, whose two High findings are on the install path: a lost backup after a failed restore, and a service left stopped after a failed stop;
  - opt-in build-provenance checks (`selfupdate/verify/ghattest`);
  - Go 1.27.2 as toolchain, and `go fix` modernisers;
  - a replace-before-stop option, and a Task Scheduler backend;
  - a documentation pass (0022), whose chosen D3-A exports `selfupdate.ErrBatchFile` and released `v1.15.0`.

  `v1.14.0`→`v1.15.0` adds that error and `taskscheduler.ErrTaskDisabled`. `v1.15.0`→`v1.15.1` changes no exported API: it fixes how Windows detects an exited process.

### acp-go-sdk2 (R, M)

- **Tags:** one, `v0.14.0`, peeling to `3588f8a3ca1ec0ee72be194eaa06ca32b6558f94`, tagged 2026-10-10T23:43:57Z. gobble pins it (`go.mod:15`), as 0014-PLAN set.
- **Module:** `go 1.27.2`; no requirements; Apache-2.0.
- **Default branch at `41b8840`, 23 commits past the tag** (all of 2026-10-11). Against the tag, its root package:
  - **adds** 133 methods (`UnmarshalJSON` and `Validate` among them) and 7 struct fields, among them `CreateElicitationUrl.Session` and `.Request`;
  - **removes** 21 items: the nine `ErrorCode…` variant types, `ErrorCode`'s nine fields, and its `MarshalJSON`, `UnmarshalJSON` and `Validate`;
  - **changes** four: `ErrorCode` from a struct to `int32`, and `ExtNotification`, `ExtRequest` and `ExtResponse` from `any` to `= json.RawMessage`.
- **Already absorbed:** the type changes of its 0012 record (sized integers, `json.RawMessage` raw fields, typed maps) were in `v0.14.0`, and 0014-PLAN absorbed them.

## F1 — go-selfupdate-lib can move to `v1.15.1` now, with no code change (R, M)

- **Compatible:**
  - apidiff `v1.9.0`→`v1.15.1` reports no incompatible change and 30 compatible ones.
  - `v1.15.0`→`v1.15.1` reports none of either.
  - Run the other way, `v1.14.0`→`v1.9.0` reports 29 incompatible changes, which shows the tool flags a break when there is one.
- **Same workflow interface:**
  - The build workflow takes `spec-path` (required), `module-dir`, `extras-artifact-name`, `artifact-name` and `retention-days` at `v1.9.0`, `v1.14.0`, `v1.15.0` and `v1.15.1`.
  - The publish workflow takes `artifact-name`, `products-json`, `platforms-json`, `extra-assets-json` (required) and `prerelease-channels-json` at all four.
  - Neither declares a secret. gobble passes no input `v1.15.1` lacks, and omits none it requires.
- **Same build toolchain:** at each of those versions, the build workflow installs Go from the caller's `go.mod` (`go-version-file`), with `GOTOOLCHAIN: local`.
- **gobble on `v1.15.1`:** `go mod tidy` changes one `go.mod` line and two `go.sum` lines. Build, vet on three systems and the 29 test packages pass, `TestReleaseSpec` among them.
- **What changes in the tree:**
  - `go.mod:17` and `go.sum:25-26`;
  - `ci.yml:145` and `:157`, to `fa74db4b292baaa14df6e057886102cc712bbeeb # v1.15.1`, so the workflows run at the tag `go.mod` requires (0005-PLAN's F10 notes);
  - `docs/architecture.md:107` and `README.md:86`.
- **Precedent:** 0004-PLAN's entry of 2026-10-06 refreshed this library and go-llmprovider-sdk as "an execution fact, not an architectural choice". It recorded the refresh in that PLAN, with a scratch-copy check and a negative test (`docs/decisions/0004-PLAN-go-module-architecture.md:874`).
- **Why now and not at F10:**
  - the gap has grown by six minor versions since 2026-10-06;
  - every push's CI rehearsal runs the pinned workflows;
  - F10 re-resolves the newest tag anyway (0005-PLAN, amendment of 2026-10-03).
- **What it does not buy:** the High fixes and the Windows process fixes are on the install and service paths. gobble calls them only from `gobble update`, in F10.

## F2 — go-llmprovider-sdk: `v1.7.1` in F4-1a, and F4-2 on `v1.8.0` (R, M)

- **F4-1a's entry names `v1.7.0`.** 0005-PLAN's entry "F4-1 re-made for go-llmprovider-sdk `v1.7.0`" (committed in `ed49e76`) pins it, with the error flag, a true `NativeStreaming` and the wire tests. P7 follows, then F4-2.
  - `v1.7.1` has the same exported API, and passes gobble's tests unchanged, alone and with the other two at their current tags.
  - That entry's decision 1 already takes the latest tag at each later sub-phase. Taking `v1.7.1` in F4-1a itself is a change to the entry, and the owner's.
- **`v1.7.1` changes what gobble offers.** `models_catalog.go` rebuilds the curated lists from the providers' official pages (0033-MADR D4). That changes four default models, since gobble takes `StaticModels[0]`:
  - OpenAI `gpt-4.1-mini` → `gpt-5.6-luna`;
  - Claude `claude-haiku-4-5` → `claude-haiku-5-5`;
  - Gemini `gemini-3.7-flash` → `gemini-3.6-flash`;
  - Grok `grok-3-mini-fast` → `grok-4.7`.

  It also removes ids gobble lists today, among them OpenAI's `gpt-4.1-nano` and `o4-mini`. 0033-MADR records those two as shut down on 2026-10-23, so on `v1.7.0` or earlier gobble's model list offers two ids that stop answering on that date.
  - The other five providers' first ids do not change.
  - No gobble test depends on a default model. gobble's one hard-coded id, `gpt-4.1-mini` in the strict probe's live test (`llm/provider/strict_live_test.go:62`), stays in `v1.7.1`'s list.
- **`v1.8.0` publishes the metadata F4-2 was to build.** 0033-MADR, accepted 2026-10-11, lists gobble-cli's 0005-MADR, 0004-MADR and 0014-REPORT as consulted. It decides:
  - D1: a `catalog.ModelInfo` with context, input and output limits, modalities, tools, reasoning and efforts, costs with cache read and write, and status and retirement, each with a presence set;
  - D2: layers, highest first: the caller's own, the provider's listing, then models.dev;
  - D3: one `Lookup` entry point. The caller persists the document, and "gobble-cli seeds its cache with its own embedded copy, under its own `NOTICE`. It passes the parsed copy as a layer, with `WithoutModelMetadata` to stay offline";
  - D5: `v1.7.1` for the refresh, and `v1.8.0` for the metadata, additions only.

  Its metadata PLAN is `proposed`, so `v1.8.0` is not near.
- **What that means for F4-2.** 0005-MADR's amendment of 2026-10-10 chose a gobble-side models.dev snapshot in Pi's `Model` shape, with `models.json` overlaid by gobble. 0033's design keeps the embedded copy and the `NOTICE` credit, and moves the merging into the SDK, with `models.json` as one caller layer.
  - Built before `v1.8.0`, F4-2 writes a resolver that `v1.8.0` replaces.
  - Built after it, F4-2 is a seed file, a `models.json` layer, and the Pi-shaped view over `ModelInfo`.
- **0014-REPORT's catalog question is answered.** It left unverified "whether SDK `catalog` metadata supplies every field 0005-MADR's catalog row names". At `v1.7.1` it does not; at `v1.8.0`, by 0033-MADR D1, it will.
- **F4-1b's dependency is unmet.** It needs `WithRetry` to implement `Streamer` and retry before the first event. At `v1.7.1` the wrapper reports `NativeStreaming: Unsupported`, and documents that a caller wanting live text streams the provider itself. None of the 12 commits past `v1.7.1` is about streaming.
- **The forced-choice work does not reach gobble today:** gobble sets no tool choice, so neither the degradation nor `Response.Warnings` fires. If gobble ever forces a tool, `Response.Warnings` is where the degradation shows, and gobble should surface it.

## F3 — acp-go-sdk2 is current; its next minor breaks one gobble test and changes decoding (R, M)

- **Current:** gobble pins the only tag, so there is nothing to bump today.
- **The API breaks on its default branch do not touch gobble.**
  - Nothing in gobble names `acp.ErrorCode…`, `acp.ExtNotification`, `acp.ExtRequest` or `acp.ExtResponse`. Every import binds the package as `acp`: 44 by an alias, and two by its package name, `acp`.
  - gobble at `41b8840` builds and vets on three systems.
- **One gobble test fails at its default branch.** At `0e32255` and at `41b8840`, `acpclient`'s `TestElicitationNotFound` fails:

  ```text
  elicitation_test.go:75: elicitation/create: {"code":-32602,…"CreateElicitationUrl: no member set"}, want method not found
  ```

  - The test builds an elicitation with no scope (`acpclient/elicitation_test.go:71-72`). At the default branch, `CreateElicitationUrl` gains `Session` and `Request` scopes, and its new `MarshalJSON` refuses an elicitation with neither.
  - At `b70e692`, before that change, the same test passed.
  - gobble's own code builds no elicitation: its client answers `elicitation/create` with -32601, method not found (`acpclient/conn.go:454-455`).
  - So the fix, at the bump, is the test's request: give it a scope. The other 28 packages pass.
  - This break adds fields, which apidiff calls compatible. Only running gobble's tests showed it.
  - **The SDK made the same fix to its own tests.** The change is commit `0e32255`, which also turns `ErrorCode` into `int32`. Its record, `0011-PLAN-engine-tests.md`, says: "Three root tests built or decoded a form elicitation with no scope, which the type could not hold before. … Each now carries a session scope."
  - **Measured from a different baseline.** That record also lists `Validate` on `Error`, `ErrorStopReason` and `IdleStateUpdateError` as removed. Those three were added after `v0.14.0` (they are at `b70e692`) and removed at `0e32255`, so from the tag there is no net change, as the surface diff reports.
- **The next minor's behaviour changes,** recorded in acp-go-sdk2's `0012-PLAN-schema-fidelity-framework.md` ("No tag; a push waits for the owner's ask. The next minor version's release notes list every changed field type and signature … and the behaviour changes"):
  - a message without a required key fails to decode, and a request gets `-32602`;
  - JSON `null` for a struct with a required key fails the same way;
  - malformed values of lenient fields fall back, and invalid items of lenient arrays are dropped;
  - a number outside its type's range fails, unless the field is lenient;
  - `Validate` no longer rejects a present empty string, and checks nested values, union variants and `minLength`.
- **What the empty-string change means for gobble.** gobble's ACP server checks no required ID for `""` itself; it relies on the SDK. After the bump, an empty `sessionId` reaches gobble's session lookup rather than being refused as `-32602`. gobble's tests pass at heads that carry the `Validate` change, so none of them depends on it.
- **What the tests cannot show:** a peer that omits a required key, and gobble's answer once the bump lands. That needs a real client, such as magic-cli-remote or an editor.
- **A pseudo-version is not a pin.** 0014-REPORT cites 0004-MADR's rule of no gobble tag on a pseudo-version. The measurements above are evidence, not a proposal.

## F4 — `gobble update` belongs in F10, and F10's step 4 must be re-made executable then (R)

- **Nothing to update to yet:** gobble has no git tag and no GitHub release. `gobble update` resolves the newest release of `maccavelli/gobble-cli`, and the first one is F10's own release gate.
- **The surface exists at `v1.14.0`, and later tags only add to it:**
  - `selfupdate/cli` documents `<prog> update [--check] [--yes|-y] [--force] [--dry-run] [--json] [--version vX.Y.Z] [--channel NAME]`, with exit 0 when current, declined or installed, 10 when `--check` finds an update, and 1 on error;
  - a program that parses its own flags "binds Flags on its flag set and calls Flags.Request, Run and Exit itself". `Flags` is a plain struct (`Check`, `Yes`, `Force`, `DryRun`, `JSON`, `Version`, `Channel`), so Kong can fill it;
  - `NewStandaloneInstaller`, `NewExactAssetSelector`, `NewGitHubSource`, `NewTextReporter`, `NewTerminalConfirmer`, `NewVersionProber`, `NewImageVerifier`, `Updater.RunWith`, `Start`, `Checker.Check` and `StandaloneInstaller.CleanupPending` exist, as 0005-MADR's self-update rows name them.
- **F10's step 4 is stale as written.** It names `go-core-lib/selfupdate` `v1.1.0` at a fixed commit (0005-PLAN F10). Its amendment of 2026-10-03 already says to re-resolve go-selfupdate-lib's newest `v1.x`, and to compare `selfupdate/cli` and `buildinfo` before binding. So F10 starts with a "made executable" amendment of its own, as F1 to F4 did.
- **F10's other dependencies:**
  - the default permission rule `bash(gobble update*)` → ask is 0005-PLAN F6's;
  - `gobble doctor`, which also checks for updates, is F10 step 3.

## F5 — The integration survey: what gobble could use, and what it should not change (R)

Each seam was read for code that duplicates a library, or that leaves a library feature unused that gobble needs.

### go-llmprovider-sdk

- **No duplication.**
  - `llm/provider/ambient.go` reads the SDK's registry: each descriptor's `EnvVar`, `RequiresAPIKey` and `StaticModels`. The SDK has no chooser of its own; its registry offers `Register`, `Descriptor`, `Descriptors` and `New`.
  - `mapError`'s sentinel table exists because gobble's agent-facing `llm` facade has its own errors (0004-MADR).
- **Three optimizations are already planned, in F4-1a and P7:**
  - send `IsError`;
  - report the `WithRetry` result's `NativeStreaming`;
  - set `Tool.Strict`.
- **Two corrections to F4-3's entry, before it is built:**
  - **`FedRAMP` must be kept.**
    - F4-3 keeps "the SDK session's own fields (`issuer`, `clientId`, `accountId`, `tokenUrl`) beside Pi's" (0005-PLAN, F4-3).
    - `auth.OAuthSession` at `v1.7.1` also has `FedRAMP`, which "sends X-OpenAI-Fedramp", and the SDK's own `FileTokenStore` saves it as `fedramp`.
    - An `auth.json` without it loses the flag on reload, and a FedRAMP ChatGPT account then stops sending the header.
  - **Reuse gobble's own owner-only file code.**
    - F4-3 plans the 0600 file and the Windows DACL "through `golang.org/x/sys/windows`", and a lock file for every write.
    - gobble already has both: `internal/appdirs` (`EnsurePrivateDir`, `FileIsOwnerOnly`, and on Windows `SecurePrivateFile`), and `session/jsonl`'s exclusive lock (`flock` on Unix, `LockFileEx` on Windows).
    - The lock is unexported (`session/jsonl/lock.go:21`). Moving it to an internal package both use is the optimization; a third copy is what F4-3 would otherwise write.
    - The SDK's own equivalents (`internal/ownerperm`, `internal/filelock`) are internal to the SDK, so gobble cannot import them.
  - The SDK's `TokenStore` holds `*OAuthSession` only (`Load`, `Save`, `Delete` by provider). API keys stay gobble's, in the same file, as F4-3 plans.
- **To assess at F4-3:** `wizard.ConfigureLLM` and `wizard.Logout`, over a `Prompter` (`Select`, `MultiSelect`, `Confirm`, …), against F4-3's own `gobble auth login`.

### go-selfupdate-lib

- **No duplication.**
  - `internal/buildinfo` adds one rule over the library's `Info`: a version that is not a clean SemVer tag becomes `0.0.0-dev+<revision>` (0008-MADR D19). The library has no such rule.
  - Its `buildinfo` is unchanged from `v1.9.0` to `v1.15.1`.
- **Nothing to change before F10.** The workflows take the same inputs, and `gobble update`'s binding is F10's.

### acp-go-sdk2

- **No duplication.** gobble calls 19 of the 32 helper constructors in `helpers.go`. Its non-test code builds five `SessionUpdate` variants by hand, and the SDK has no helper for any of them:
  - `ConfigOptionUpdate` (`acpserver/commands.go:330`);
  - `SessionInfoUpdate` (`:387`);
  - `AvailableCommandsUpdate` (`:516`);
  - `CurrentModeUpdate` (`acpserver/modes.go:50`);
  - `UsageUpdate` (`acpserver/prompt.go:304`).

  The helpers build only `AgentMessageChunk`, `AgentThoughtChunk`, `Plan`, `ToolCall`, `ToolCallUpdate` and `UserMessageChunk`.
- **To prepare for the next minor:** the elicitation test's scope, and an interoperability check (F3).

## F6 — Order and timing (R)

| Order | Change | Library and pin | When | Blocked by |
| ---: | :--- | :--- | :--- | :--- |
| 1 | Pin refresh, no Go change | go-selfupdate-lib `v1.15.1` | now, before or beside F4-1a | nothing |
| 2 | 0005-PLAN F4-1a | go-llmprovider-sdk `v1.7.1` | now; already the "Next" line, at `v1.7.0` | the owner's choice of tag |
| 3 | Records: F4-3's two corrections, F4-2 on `v1.8.0` | none | now, with F4-1a's records or before | nothing |
| 4 | 0012-PLAN P7, made executable first | go-llmprovider-sdk, the latest tag | after F4-1a | F4-1a's pin |
| 5 | F4-3, credentials | go-llmprovider-sdk, the latest tag | after P7 | nothing outside gobble |
| 6 | F4-2, re-made on `ModelInfo` | go-llmprovider-sdk `v1.8.0` | when tagged | the SDK's tag |
| 7 | F4-1b, live streaming | the first tag with a streaming `WithRetry` | when tagged | the SDK |
| 8 | acp-go-sdk2's next minor | that tag | when tagged | the SDK; the elicitation test; an interoperability check |
| 9 | F10 `gobble update` | go-selfupdate-lib, the latest tag at F10 | F10, after F5 to F9 | F6's permission rule; the first gobble release |

- **Rows 1 and 2 are independent.** Each changes only its own `go.mod` line, and the clone with both, plus acp-go-sdk2 at its tag, passes as each alone does. The refresh can also ride with F4-1a's `go.mod` change, as 0014-REPORT suggests for a pin-only bump. Kept apart, F4-1a stays provider-scoped, and the refresh keeps its own record, as 2026-10-06's did.
- **If F4-1a stays on `v1.7.0`,** a `v1.7.1` bump must still land before 2026-10-23, for the shut-down ids.
- **Rows 5 and 6 swap the plan's F4-2 and F4-3,** and row 3 amends both entries. That is the owner's to decide. Rows 6 and 7 can come in either order, depending on which SDK tag lands first.

## F7 — Recommendation: adopt now, assess now, optimize only where the evidence points

**Adopt now: yes.** All three are already imported, and each current tag works with gobble unchanged (Summary).

- **go-selfupdate-lib.** Pin `v1.15.1` now, in its own refresh (F1).
- **go-llmprovider-sdk.** Its features are the current phase's (F4), and `v1.7.1` carries a model-list fix with a date on it (F2). Take it in F4-1a.
- **acp-go-sdk2.** It is already current; adopting it is done (F3).

**Start the compatibility assessments now: yes.** Each is read-only or runs on scratch clones, so it changes nothing in gobble until an approved entry does. Two have already found defects:

- the elicitation test against acp-go-sdk2's next minor (F3);
- `FedRAMP`, missing from F4-3's field list (F5).

What to keep running, at the start of each sub-phase and at each library tag, as 0005-PLAN's rule of 2026-10-11 already says for F4:

- `go get` the latest tag on a scratch clone, then build, vet on three systems and test;
- apidiff from the pinned tag;
- for acp-go-sdk2, a root-package surface diff and gobble's tests at its default branch, since its next minor is the one with behaviour changes;
- the recorded facts' claims check.

Whether those scripts move from the session scratchpad into `scripts/`, as a gate any session can run, is the owner's choice. Their logic is in this report's method.

**Start code optimizations now: only these.**

- **F4-1a and P7's three items** (F5): the error flag, a true `NativeStreaming`, and the strict flag. Each has an approved entry, or one whose amendment is written.
- **Before F4-3 is built,** amend its entry: keep `FedRAMP`, and reuse `internal/appdirs` and a shared form of `session/jsonl`'s lock (F5).

**Not now:**

- **F4-2's catalog resolver,** which go-llmprovider-sdk `v1.8.0` replaces (F2). F4-2 waits for that tag, or the owner accepts the rework.
- **Any change for an untagged library commit.** acp-go-sdk2 has 23 untagged commits, and go-llmprovider-sdk 12. Code written against them is written against a moving target, and against a pseudo-version, which gobble does not pin (F3).
- **Refactors at the seams.** The survey found nothing duplicated to remove (F5).

## F8 — Two records-hygiene findings (R)

- **Two records claim number 0014 for different subjects.**
  - `docs/reports/0014-REPORT-llmprovider-sdk-integration.md` (commit `ad6ea57`, merged in `6a8b9d8`) examines go-llmprovider-sdk.
  - `0014-MADR-move-to-acp-go-sdk2.md` and its PLAN are the ACP move.
  - The REPORT's base, `051fead`, predates `d42d24a`, which added 0014-MADR, and has no 0014 record. So each session took 0014 from `check_records.py --next` in its own checkout.
  - The repository's rules let a REPORT share a number only with a record it examines; this one examines none of 0014's.
  - `scripts/check_records.py` (`check_structure`, `:87-105`) warns on two MADRs with one number, a PLAN without a MADR, and a GATES without a PLAN, and on nothing else, so it passes.
- **0014-REPORT is not in `docs/README.md`.** The index has rows for 0014-MADR and 0014-PLAN only, and the records check does not check the index.
- **The remedies are the owner's,** since 0014-REPORT is another session's committed work: renumber it (a number is never reused, so it would take the next free one), or record the collision where it stands. Separately, the records check could warn on a REPORT that shares a number with an unrelated pair, and on a record missing from the index. This report changes neither.

## How the claims were checked

- **The claims check:** `q401_claims_0015.py` holds every factual claim above as a file, an optional line range and a pattern. It reads them from gobble's tree, the module cache, and the evidence files the other scripts saved. Its count and its inverted run are in "Confirmation".
- **Negative tests of the instruments:**
  - **apidiff:** run backwards, `v1.14.0`→`v1.9.0`, it reported 29 incompatible changes.
  - **The surface diff:** run on the tag against itself, fetched from GitHub, it reported no change in 2,067 items. Run on the default branch, it found the 21 removals.
  - **The experiment harness, go-llmprovider-sdk `v1.0.0`:** the build fails (`llm/provider/provider.go:138`: `it.Signature undefined`).
  - **The experiment harness, go-selfupdate-lib `v1.5.0`:** with `go mod tidy` it is reported as `PIN DRIFT … asked v1.5.0 got v1.14.0`, because tidy fetched the newest release with `selfupdate/releasespec`, which `v1.5.0` lacks. Without tidy it fails: `no required module provides package …/selfupdate/releasespec`.
  - **The experiment harness, at acp-go-sdk2's default branch:** it reported the `TestElicitationNotFound` failure, a real one.
  - **The workflow comparison:** run over a copy of `v1.14.0`'s build workflow with `spec-path` removed, it reported the removal, and `spec-path` as an input gobble passes that the copy lacks.
- **What the method got wrong first:**
  - **cgo in the vet steps.** `go vet` for linux from this Windows host failed in every clone, the unmodified baseline among them, because it compiled `runtime/cgo` with the host's C compiler. gobble's shipped builds run with cgo off (`make check-cgo-off`), so the vet steps were re-run with `CGO_ENABLED=0`, and passed.
  - **The first selfupdate negative** named `v1.0.0`, which `go get` refuses for its old module path. The harness showed `get=1`, and the negative was re-made with `v1.5.0`. The clone left by that run then made the next clone fail: removing a tree with read-only git objects had failed silently. The harness now clears the flag and asserts the tree is gone.
  - **The helper count first counted slice literals** of helper results (`[]acp.SessionUpdate{acp.UpdateAgentMessageText(…)}`) as hand-built values: 13, not 5. It now counts struct literals only.
  - **The curated-list script read the compare file before its version arguments,** so it failed once the branch had no commits past the tag. It now compares two cached versions without the network.
  - **The libraries moved while this ran.** The first compare, at about 13:25 UTC, was printed and not saved. Every later run's output is kept under its time.

## Not verified

- How a real ACP client fares against gobble on acp-go-sdk2's next minor (F3).
- The exact shape of go-llmprovider-sdk `v1.8.0`'s API: 0033-MADR's D1 to D3 are its plan, and the metadata PLAN is `proposed`.
- Whether `wizard.ConfigureLLM` fits `gobble auth login` (F5): named as an assessment, not made.
- Whether `actions/setup-go` in the publish workflow honours `toolchain go1.27.2` in go-selfupdate-lib's own `go.mod`. That workflow builds the library's release tool, not gobble.
- `go test -race` and the Linux race gate on the combined pins: the experiments ran `go test` without the race detector, on this Windows host alone.
- Whether the four new default models answer gobble's requests: the SDK's refresh PLAN measured each candidate live, and gobble did not.
- Any library commit after the front matter's.
- go-tui-lib, the fourth sibling library, which the owner did not name. It is not in `go.mod` before 0005-PLAN F9.

## Confirmation

Run on 2026-10-11:

```text
python -I q401_claims_0015.py             -> 293 claims; 0 failed
python -I q401_claims_0015.py --invert    -> inverted: 293 claims, 293 failed
python -I q398_experiments.py pinset3     -> tidy=0; build=0; vet linux=0; vet darwin=0; vet windows=0; test=0; govulncheck=0
python -I q398_experiments.py acp-head3   -> test=1: acpclient TestElicitationNotFound, 28 packages ok
python -I q396_apidiff.py                 -> incompatible 0 in every pair
python -I q397_workflow_inputs.py v1.15.1 -> added [], removed [], newly required [] for both workflows
python -I q406_acp_surface.py v0.14.0     -> added 0; removed 0; changed 0
python -I q397_negative.py                -> the planted removal of spec-path is reported
```

## Evidence index

| Claim | Source |
| :--- | :--- |
| go-llmprovider-sdk has 13 tags, the last `v1.7.1`, peeling to `0adc791…`; `v1.7.0` to `93f3b53…` | `versions-go-llmprovider-sdk.txt`; `peel.json`; `peel-1500.json` |
| its default branch, 12 commits past `v1.7.1`, adds 12 API items, and gobble passes at `f9834e0` | `out-q392-2100.txt`; the `v1.7.1`→`502a5d4` apidiff report; `exp_summary4.txt` |
| 0033-MADR's D1 to D6, its two PLANs | `docs/decisions/0033-*` at `v1.7.1` |
| four default models change; `gpt-4.1-nano` and `o4-mini` go, shut down 2026-10-23 | `out-q403-171.txt`; 0033-MADR; `llm/provider/ambient.go:33-34` |
| `v1.2.1` and `v1.7.0` have the same curated lists | `out-q403-121.txt` |
| gobble's one hard-coded model id, and no tool choice | `llm/provider/strict_live_test.go:62`; a search of gobble's non-test Go files |
| go-selfupdate-lib has 24 tags, the last `v1.15.1`, peeling to `fa74db4…`; 4 documentation commits past it | `versions-go-selfupdate-lib.txt`; `peel.json`; `out-q392-2100.txt` |
| `v1.0.0` declares `go-core-lib` | the `neg-su-1.0.0` clone's `go get` log |
| 0022-MADR's D3-A and `v1.15.0` | `docs/decisions/0022-MADR-…` at `v1.15.0` |
| acp-go-sdk2 has one tag, peeling to `3588f8a…`; 23 commits past it | `versions-acp-go-sdk2.txt`; `peel.json`; `out-q392-2100.txt` |
| its root package at `41b8840`: 140 added, 21 removed, 4 changed | `out-q406-41b.txt`; `out-q406-self.txt` |
| `TestElicitationNotFound` fails at `0e32255` and `41b8840`, passes at `b70e692`; the scope fields | `exp_summary3.txt`, `exp_summary4.txt`, `exp_summary.txt`; the clones' test logs; `types_gen.go` at `0e32255` |
| gobble names none of the changed ACP types | a search of gobble's Go files |
| the local clone's head; `0e32255` brings the scope and `ErrorCode`; the three `Validate` methods by ref; the SDK's own fixture fix | `clone-evidence.txt`; `docs/decisions/0011-PLAN-engine-tests.md` in the clone |
| the next minor's behaviour changes | acp-go-sdk2 `0012-PLAN-schema-fidelity-framework.md` at `b70e692` |
| gobble pins `v0.14.0`, `v1.2.1`, `v1.9.0` | `go.mod:15-17` |
| apidiff: 0 incompatible in every pair; 15 and 30 compatible to the current tags; 29 incompatible reversed | `apidiff/` reports; `out-q396-now.txt` |
| the workflow inputs are unchanged through `v1.15.1`, and gobble passes none missing | `out-q397.txt`; `out-q397-1500.txt`; `out-q397-1515.txt`; `out-q397neg.txt` |
| the build workflows use the caller's `go.mod` and `GOTOOLCHAIN: local` | `build-selfupdate-release.yml:105-114` at each version |
| the requirement sets pass build, vet and test, and govulncheck on the combined ones | `exp_summary*.txt`, `exp_revet.txt` and the clones' logs |
| gobble has no tag and no release | `gobble-tags.txt`; `gobble-releases.json` |
| `selfupdate/cli`'s surface, `Flags`, and the binding rule | `selfupdate/cli/doc.go`, `flags.go:26-40`, `:116`, `run.go:120`, `:288` at `v1.14.0` |
| F10 step 4 names go-core-lib `v1.1.0`; F4-3's field list and its DACL step | `docs/decisions/0005-PLAN-v1-feature-scope.md`, F10 and F4-3 |
| `OAuthSession.FedRAMP`, and the file store saving it; `TokenStore`'s methods; the wizard's functions | `llmprovider/auth/tokenstore.go`, `tokenstore_file.go`, `wizard/*.go` at `v1.7.1` |
| gobble's owner-only and lock code | `internal/appdirs/*.go`; `session/jsonl/lock*.go` |
| 19 of 32 helpers used; 5 hand-built variants, none with a helper | `out-q407.txt` |
| the SDK's metadata exports three methods, and its catalog has no limits or cost, at `v1.7.1` | `catalog/model_metadata.go` and `catalog/*.go` at `v1.7.1` |
| 0014's two subjects; 0014-REPORT unindexed; the checker's three warnings | `docs/reports/`, `docs/README.md`, `scripts/check_records.py:87-105`; `git-evidence.txt` |

The evidence files are in the session scratchpad, beside the scripts.

## Bears on

- [0004-MADR-go-module-architecture.md](../decisions/0004-MADR-go-module-architecture.md) and [0004-PLAN-go-module-architecture.md](../decisions/0004-PLAN-go-module-architecture.md): the dependency rule, and the pin-refresh precedent (F1).
- [0005-MADR-v1-feature-scope.md](../decisions/0005-MADR-v1-feature-scope.md) and [0005-PLAN-v1-feature-scope.md](../decisions/0005-PLAN-v1-feature-scope.md): F4's pin, catalog, credentials and order, and F10 (F2, F4, F5, F6).
- [0012-PLAN-tool-schemas-descriptions-and-loading.md](../decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md): P7 (F6).
- [0014-MADR-move-to-acp-go-sdk2.md](../decisions/0014-MADR-move-to-acp-go-sdk2.md) and [0014-PLAN-move-to-acp-go-sdk2.md](../decisions/0014-PLAN-move-to-acp-go-sdk2.md): the ACP pin and the elicitation answer (F3).
- [0014-REPORT-llmprovider-sdk-integration.md](0014-REPORT-llmprovider-sdk-integration.md): its catalog question (F2), and its number (F8).
