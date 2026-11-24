---
status: proposed
date: 2026-11-24
associated-madr: "0072-MADR-keep-a-refreshed-models-dev-copy-on-disk.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement a Refreshed Copy of the models.dev Document on Disk, Fetched by gobble, with No Built-in Model Data

Associated MADR: [0072-MADR-keep-a-refreshed-models-dev-copy-on-disk.md](0072-MADR-keep-a-refreshed-models-dev-copy-on-disk.md)

*(2026-11-24, approved by the owner, for the MADR and PLAN together: "Accept; approve; queue after 0070 (Recommended)". The build is queued after 0070; the status stays `proposed` until it starts.)*

*(2026-11-24, re-ground at 61569d0, where 0071's Phase 1b is committed. Moved citations: none. The changes since ba8f623 are `acpserver/agent.go` (line 836, edited in place, no line moved), `acpserver/outputlimit.go` and `acpserver/prompt.go` (lines added after 75 and 273; neither file is cited here), `internal/cli/line.go` (line 401 on; not cited here), `go.mod` (line 20, `acpext v0.5.0` to `v0.6.0`, in place; the SDK pin cited at `go.mod:16` has not moved) and docs. Every line cited in `acpserver/agent.go` (:457-464, :701, :1102-1124, :1145-1152) is at its old line. No row is rewritten.)*

*(2026-11-24, first draft, with the MADR's Q1 to Q3 open.)*

*(2026-11-24, second draft, with the MADR's second draft: every question answered, every row decided. The base is ba8f623,
where 0071's Phase 1 is committed. Phase 3, the removal, no longer waits for the SDK; Phase 4 is gobble's pin of
go-llmprovider-sdk `v1.33.0`.)*

*(2026-11-24, third draft, with the MADR's third draft: Phase 4 also hands each session's document to every `provider.New`
for that session, rebuilds a session that began with none once when the first download lands, and pins the cost of a refresh
(M62, M71 to M73).)*

## Goal

After Phases 1 and 2, gobble downloads `https://models.dev/api.json` itself, keeps the nine sections in its cache directory,
refreshes them in the background before a model call or in a `models` command when they are due, never makes a prompt, a tool
call or a start-up wait for it, guards against a bad or shrunken download, keeps each session's facts from its start, shows
the data's age, and answers `gobble models refresh [--force]`. After Phase 3, the binary carries no model data. After Phase 4,
on SDK `v1.33.0`, the SDK's listings and lookups read gobble's document, handed in. Every decided row of the MADR has a test,
red at the base where it changes behaviour, and a plant that turns it red.

Nothing starts before the MADR is `accepted`, this PLAN is approved, and the owner says `proceed`. The agent stages nothing
and commits nothing: each phase ends with its files ready, its gates recorded, and the coordinator stages; the owner commits.

## Dependencies and order

* **Base.** ba8f623: 0071's Phase 1 is committed
  ([0071-PLAN-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-PLAN-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md)).
  M37 and M50 build on its output-limit line and its `capSourced` (`llm/provider/model.go:126-140`).
* **0070.** [0070-PLAN-keep-a-failed-partial-reply-and-log-stream-health-per-call.md](0070-PLAN-keep-a-failed-partial-reply-and-log-stream-health-per-call.md)
  is also based on 0071's Phase 1 and edits `internal/config/keys.go`, the schema golden and `llm/provider/provider.go`, as
  Phases 1 and 2 here do. The two are built one after the other, never in one tree at once: 0070 first, then this PLAN's
  Phases 1 to 3, unless the owner orders otherwise, in which case the second re-grounds on the first.
* **Phases 1 to 3** run on `v1.32.0`. Phase 3 depends only on Phase 2.
* **Phase 4** is the change that moves gobble's pin to go-llmprovider-sdk `v1.33.0`. That tag carries the SDK's
  `0093-MADR-take-model-metadata-from-the-caller.md` (the owner's "Move it before v1.33.0"), which removes
  `WithModelMetadataURL`, so gobble does not compile on `v1.33.0` without Phase 4. The order is: this PLAN's Phases 1 to 3;
  the SDK's 0093 build and its `v1.33.0` tag; gobble's pin with Phase 4 in the same change (as 0065-PLAN's Phase 2 begins the
  pin); then 0071's Phase 2 (and the 0066-PLAN phases it waits for), which reads `CachedLimits` over the handed document.

## Scope

### In scope

* `llm/provider`: `FetchModelData`, `ModelData`, `ParseModelData`, `CurrentModelData`, `SetCurrentModelData`,
  `Options.ModelData`; the atomic holder in place of `snapshot`; a `TestMain` with the network guard; the listing's URL
  (Phases 2) and the hand-in (Phase 4).
* `internal/cli`: the store, `attempt.json`, the engine, the trigger in `modelProvider`'s provider and the `models` commands,
  the load at `runEnv`'s start, the exits, the warnings, `gobble models refresh [--force]`, the `data` line, `ModelDataLine`;
  `TestMain`'s seams.
* `acpserver`: `Options.ModelData`, `Options.ModelDataLine`, the session's document, its line in `cmdContext`;
  `acpserver/binary_test.go`'s proxy and no-copy check.
* `cmd/gobble/conpty_windows_test.go`'s no-copy check.
* `internal/config`: the two keys with bounds, the template block, the schema golden.
* Phase 3: `llm/catalog` (`data/`, `Snapshot`, `Source`), `scripts/gen_catalog.py`, the test data move, `NOTICE`.
* Docs and records (each phase's last step).

### Out of scope

* Any SDK change (the SDK's 0093 is its own record). A source or mirror setting (M57).

## Rules for Every Phase

* **Re-ground first.** Re-read every `path:line` the MADR cites at the phase's start commit and, for the SDK, at the pinned
  tag; record each moved line. A changed fact (F1 to F12) is a stop.
* **Red, then green.** A test of existing behaviour fails at the phase's base on its assertion, never on a compile error. A
  test of a new name takes its red from its plant. Guards pass at the base and are held by their plants.
* **Plants.** One production change each, by hand, on a scratch clone of the phase's tree, never in the tree, each turning its
  own row's test red; the execution record quotes the failing line. gobble has no `tools/plant.py`; plants are made as 0070's
  and 0071's PLANs make them, one recorded edit each. A plant its test does not catch is a deviation.
* **Seams.** `llm/provider`: `Options.HTTPClient` an in-memory `RoundTripper` that records each request and serves canned
  documents built from the 2026-10-14 copy, every URL under `.invalid`; its new `TestMain` runs the tests behind 0046 D4's
  guard. `internal/cli`: `fetchModelData` and `loadModelData` replaced per test; a clock seam; a write seam that can pause
  between the temporary file and the rename; store paths under `t.TempDir()`; the run's owner as 0046's tests use it.
  `acpserver`: `Options.ModelData` and `Options.ModelDataLine` set by the test.
* **No network.** No live call and no download of models.dev. Every document is a test double.
* **Hosts and heavy runs.** A single package's `go test` without `-race` on the session host. `make pre-add-check FILES="…"`
  through `~/.agents/skills/available-dev-hosts/scripts/run-heavy --`. `make preflight` through
  `~/.agents/skills/available-dev-hosts/scripts/run-on-host` on this Mac, the Linux machine and the Windows laptop, one at a
  time per host, after `pick-host`; `make race` on the Linux machine with `run-on-host --race`, never WSL; `make
  check-test-isolation` through `run-on-host` on one host. A fallback to this Mac does not count for another host. Long output
  to a scratch file, the exit code captured before any filter.
* **Stops**, each a stop and a prompt with evidence and three resolutions: a test outside the MADR's rows turns red; a row
  cannot be met as written; an exported name not in D21; a lint finding fixable only by a suppression; the same step failing
  twice; a third build pass of a phase.
* **Exit criterion of every phase:** no blocker, every row of the phase tested, plants caught, gates recorded, docs rows
  returned.

## Implementation Steps

### Phase 0: re-ground and confirm (records and a scratch clone only)

1. Re-read the MADR's citations (F1 to F12, D1 to D21) at ba8f623 and the SDK's `v1.32.0`; record each moved line.
2. On a scratch clone, by a throwaway test there, confirm: a `jsontext.Value` field written by `jsonx.MarshalStrict` compact
   and read back by `jsonx.UnmarshalStrict` keeps `document`'s bytes exactly (M2), else D2's sha256 is taken over the
   re-encoded form and the MADR gets a dated note; `Encode` of a document with one malformed section drops it (M15, F5); Go's
   client follows a 302 to another host and `resp.Request.URL` differs from the URL asked for (M58).
3. List every test that runs gobble's binary or a whole run without the `internal/cli` seams: at least
   `acpserver/binary_test.go:87-93` and `cmd/gobble/conpty_windows_test.go:1091-1096` (M64); any other is added to M64's
   test list by a dated note, or is a stop if it cannot take a closed-port proxy.
4. List every test that reads facts from the embedded copy (`gcatalog.Snapshot`, `provider.KnownModels`, `ModelInfo*` with a
   real model, goldens of `models list` and `models info`) for Phase 3 (M48), and every test whose output Phase 2's line, `data`
   line or warnings change. The MADR expects only the tests M37, M38 and M39 name; any other is a stop.
5. Confirm that `acpserver` may hold a `*provider.ModelData` (it imports `llm/provider` at `acpserver/auth.go`) and that
   `internal/archtest` allows each new import.
6. Grep `README.md`, `docs/guides/choosing-a-model.md`, `docs/architecture.md`, `docs/README.md`, `NOTICE`,
   `llm/catalog/doc.go` and `llm/provider/provider.go` for the snapshot, `gen_catalog`, `ModelMetadataURL` and the models
   commands; note the lines Phases 2 and 3 change.

Exit: citations re-grounded; the confirmations and lists recorded; nothing but the records changed.

### Phase 1: the store, the fetch and the engine (M1 to M17, M19, M23, M58, M59, M65, M66, M69)

| Rows | Test | Package | Red at the base because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| M1 | `TestModelDataPath_0072` (`GOBBLE_CACHE_DIR`, `GOBBLE_HOME`, XDG, the default, Windows' table) | `internal/cli` | new name | the path under `dirs.State`; then under `catalog/` |
| M2 | `TestModelDataRoundTrip_0072` (with and without `etag`, `lastModified`) | `internal/cli` | new name | sha256 over the whole object |
| M3 | `TestModelDataWriteKeepsOld_0072` | `internal/cli` | new name | `os.WriteFile` onto the path |
| M4 | `TestModelDataUnreadable_0072` (`not_json`, `truncated`, `version`, `sha256`, `refused`, `oversize`, `directory`) | `internal/cli` | new name | the sha256 check skipped |
| M5 | `TestModelDataConcurrentWriters_0072` (writer A paused by the write seam after its temporary file; writer B completes; a reader reads B whole; A resumes and renames; the reader reads A whole) | `internal/cli` | new name | the file truncated and written in place |
| M6 | `TestModelDataStaleTemps_0072` | `internal/cli` | new name | no removal; then every age removed |
| M7 | `TestModelDataHeldWindows_0072` | `internal/cli`, Windows | new name | `os.Rename` for `fsx.ReplaceFile` |
| M8, M23 | `TestFetchModelData_0072` (the constant is `https://models.dev/api.json`; requests go to a `.invalid` URL the test passes; one GET each call; the bytes `ParseMetadata` reads equal) | `llm/provider` | new name | a cached result returned on the second call; then the constant changed |
| M9 | `TestFetchModelDataConditional_0072` (`etag`, `last_modified`, `304_renews`, `304_unasked`) | `llm/provider`, `internal/cli` | new name | the validators not sent |
| M10 | `TestFetchModelDataUsesClient_0072`; `llm/provider`'s new `TestMain` guard | `llm/provider` | new name | `http.DefaultClient` in place of the run's (the guard then reports the `.invalid` host) |
| M11 to M13 | `TestModelDataRefreshFails_0072` (`offline`, `status`, `oversize`, `not_document`; the copy, the current document and `attempt.json`'s result checked) | `internal/cli` | new name | the copy cleared on a failure |
| M14 to M17 | `TestModelDataGuards_0072` (`empty`, `section_missing`, `section_malformed`, `shrink_below_half`, `shrink_half`, `first`, `other_source`) | `internal/cli` | new name | totals compared in place of sections; then the source not compared |
| M19 | `TestModelDataCurrent_0072` (the copy, else the built-in copy until Phase 3, never both) | `llm/provider` | new name | the built-in copy layered below the copy |
| M58 | `TestFetchModelDataRedirect_0072` | `llm/provider` | new name | the stamp taken from `resp.Request.URL` |
| M59 | `TestFetchModelDataDeadline_0072` (`synctest`: 30 s background, 120 s command) | `llm/provider`, `internal/cli` | new name | one deadline for both |
| M65 | `TestModelDataEngineNeverBlocks_0072` (`-race`: a fetch that blocks; a scripted prompt with a tool call completes) | `internal/cli` | new name | the trigger runs the engine inline |
| M66 | `TestModelDataAttemptFile_0072` (`shape`, `unreadable`, `two_writers`) | `internal/cli` | new name | the attempt written in place |
| M69 | `TestModelDataNotSaved_0072` | `internal/cli` | new name | the document dropped when the save fails |

Build:

1. `llm/provider`: `ModelData` (immutable: the parsed `catalog.Metadata` and its stamp), `ParseModelData`, the atomic holder
   (`CurrentModelData`, `SetCurrentModelData`) replacing `snapshot` for `modelInfoFrom` and `KnownModels`, with the built-in
   copy as the fallback until Phase 3; `FetchModelData` (D4); `TestMain` with the guard.
2. `internal/cli`: the store (path, file, write with D2's temporary file and `fsx.ReplaceFile`, read with
   `fsx.ReadRegularFile` and D3's checks, the sweep); D6's guards; `attempt.json` (D18); the engine goroutine; the seams
   `fetchModelData` and `loadModelData`, set in `TestMain` (D10).

Gates: the packages' tests; `make pre-add-check`; `make preflight` on the three hosts; `make race` on the Linux machine;
`make check-test-isolation`; `make check-records`; `make markdownlint`; `denyscan.py --changed`.

Docs: `llm/provider`'s package and `ModelInfo` comments (the current document); `docs/architecture.md:114` and the
`llm/provider` row.

### Phase 2: triggers, sessions, settings, display, the command, the listing and the records (M18, M20 to M22, M24 to M32, M34 to M46, M50 to M53, M60, M61, M63, M64, M67, M68, M70)

| Rows | Test | Package | Red at the base because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| M20 | `TestModelDataLoadedFirst_0072` (`--model` naming a model only the copy knows resolves with no custom-id warning; `initialize`, `defaultModel`, the picker, completion and `auth status --check` read it; the load counted once; no fetch) | `internal/cli` | `--model` resolves with the custom-id warning | the load after model resolution |
| M21 | `TestModelDataSwapForNewSessions_0072` (`-race`) | `acpserver` | new name | the session reads the current document per call |
| M22 | `TestListingUsesModelsDev_0072` (the listing's metadata GET goes to models.dev's URL, served in memory) | `llm/provider`, `internal/cli` | the listing names no URL in production | the URL left unset outside tests |
| M24, M25, M26, M27, M30, M31, M32, M67 | `TestModelDataRefreshTrigger_0072` (`model_call`, `models_cmds`, `initialize_no_fetch`, `context_no_fetch`, `one_in_flight`, `attempt_within_hour`, `other_process_attempt`, `recheck`, `pinned`, `clock_ahead`, `first_run`, `refused_waits`) | `internal/cli` | new name | the trigger in `modelFacts` (fetches on `initialize`); then no in-flight flag; then `attempt.json` not read; then no re-read; then 0 read as "always due"; then a refusal not recorded |
| M28 | `TestModelDataRunEnd_0072` (`acp`, `interactive`: cancelled at once, the exit not waiting; `headless`, `models`: a 1 s fetch and write finish, a blocked one cancelled at 5 s; `synctest`) | `internal/cli` | new name | every exit waits 5 s |
| M29 | `TestCompletionStartsNoRefresh_0072` | `internal/cli` | new name | completion through the triggering read |
| M34 | `TestNoBrowserOrSignIn` gains `fetchModelData` and `loadModelData` | `internal/cli` | red once the seams exist and `TestMain` does not set them | `TestMain` leaves `loadModelData` the real one |
| M35, M36 | `TestSchemaCommitted` and the config package's key tests, golden and template updated first | `internal/config` | the golden names the keys and the build does not | `Min: 1` in place of 0; then a default of 25 |
| M70 | `TestModelDataSettingBounds_0072` (8,760 and 3,650 accepted; 8,761 and 3,651 "must be at most N", the default used) | `internal/config` | new keys | `Max` unset |
| M37 | `TestModelDataLine_0072` (`internal/cli`: `fresh`, `past_warn`, `none`, `clock_ahead`); `TestContextShowsModelData_0072` (`acpserver`: the line after 0071's, for the session's data) | `internal/cli`, `acpserver` | new names | `>` for `>=` against `warnDays`; then the line before 0071's |
| M38 | `TestModelsInfoData_0072` | `internal/cli` | the `data` line is not printed | the line omitted for none |
| M39, M40, M63 | `TestModelDataWarning_0072` (`past`, `refused`, `failed`, `acp_log_once`, `off`, `list_stdout`, `clock_ahead`, `none_pinned`, `none_failed`, `none_untried`; before Phase 3 "none" is the built-in copy in use) | `internal/cli` | new name | the refusal's reason omitted; then the none warning before a first attempt |
| M18, M41 to M46, M68 | `TestModelsRefreshCmd_0072` (`saved`, `unchanged`, `not_modified`, `first`, `offline`, `status`, `timeout_120s`, `refused`, `not_saved`, `usage`, `interrupted`, `force`, `force_empty`; stdout, stderr and exit code compared exactly) | `internal/cli` | the command does not exist | the guards skipped without `--force`; then exit 0 on a refusal; then `--force` saving an empty document |
| M50 | `TestTurnCapFromModelData_0072` (the session's document's cap reaches the turn; a refresh does not change it) | `acpserver` | new name | the cap read from the current document |
| M51 | `TestModelsJSONAboveModelData_0072` | `llm/provider` | a guard | the models.json layer appended before the document |
| M60 | `TestSessionKeepsModelData_0072` (create, load, fork; a resume after a swap takes the new copy; a removed model gets the custom-id warning) | `acpserver` | new name | load takes the document saved at create |
| M61 | `TestSessionWithNoneGainsFirstCopy_0072` | `acpserver` | new name | none captured as a fixed empty document |
| M64 | `TestGobbleACPBinary` (every proxy variable a closed port; no `model-data/models-dev.json` under its `GOBBLE_HOME`; exit 0 within 2 s; no stderr) and the Windows conpty test's no-copy check | `acpserver`, `cmd/gobble` | the checks are new | the closed-port proxy removed from the binary's environment, with the trigger planted into `initialize` |
| M52, M53 | `make check-records`; the PLAN's order | records | | |

Build:

1. `internal/config`: the keys (D11) with `Max`, the template block, `Settings` fields, the golden.
2. `internal/cli`: the load at `runEnv`'s start (D7); the trigger in `modelProvider`'s provider and the `models` commands
   (D8); the attempt gate, the re-read, the pinned mode, the clock check; the exits (D20); the warnings (D12, D13);
   `ModelsRefreshCmd` with `--force` under `ModelsCmd`; the `data` line; `ModelDataLine`; `Options.ModelData` for
   `acpserver`.
3. `acpserver`: each session's `*provider.ModelData` at create, load and fork (D19), passed through `Options.ModelInfo`'s
   `provider.Options.ModelData`; `cmdContext`'s line after 0071's.
4. `llm/provider`: production sets `Options.ModelMetadataURL` to models.dev's URL for every listing (D17); `provider.go:53-55`'s
   "for tests" corrected.
5. The binary tests (M64).
6. Records: D16's dated notes in `0005-MADR-v1-feature-scope.md` (the "Catalog refresh" row),
   `0066-MADR-set-each-session-model-up-from-discovered-limits.md` (D3) and
   `0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md` (F4); `docs/README.md` index rows for 0072.

Gates: as Phase 1.

Docs: `README.md:99-101` (the commands, `models refresh [--force]`); `docs/guides/choosing-a-model.md:170-218` (where the
facts come from, their age, sessions keeping theirs, the command, the two settings, the pinned mode); the settings template's
block; `docs/architecture.md` (`internal/cli`'s row, the cache directory's files); `scripts/gen_catalog.py`'s docstring
(`:6-8`) until Phase 3.

### Phase 3: no built-in model data (M19's last case, M33, M47 to M49), on `v1.32.0`

| Rows | Test | Package | Red at the base because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| M33 | `TestNeverOnlineFirstRun_0072` (no copy, the fetch failing: no window, 16,384 `assumed`, no prices, levels off to xhigh, "-" in `models list`, the `/context` none line, D13's warning) | `internal/cli`, `acpserver` | the built-in copy still answers | the built-in copy kept as the fallback |
| M47 | `TestNoEmbeddedModelData_0072` (`llm/catalog` embeds no file, read from `go list -json`'s `EmbedFiles`) | `llm/catalog` | the package embeds two files | `data/models.json` restored |
| M48 | `TestModelDataTestdataInstalled_0072` in each test binary Phase 0 step 4 listed | each | new name | one binary's `TestMain` without the install |
| M49 | `TestNoticeNamesExistingFiles_0072` (every path `NOTICE` names exists) | `internal/archtest` (which reads the repository's files) | a guard: green at the base | `NOTICE` still naming `llm/catalog/data/LICENSE.models.dev` after the move |

Build: move the 2026-10-14 document to `llm/provider/testdata/models-dev-2026-10-14.json` and install it in each listed
`TestMain`; remove `llm/catalog/data/`, `Snapshot`, `Source`, their tests and `scripts/gen_catalog.py`; remove the fallback
from `llm/provider`; rename `SnapshotErr` to `ModelDataErr`; move the licence text to
`llm/provider/testdata/LICENSE.models.dev` and re-word `NOTICE:22-28` (D15). Record the build's size before and after on each
host.

Gates: as Phase 1.

Docs: `NOTICE`; `llm/catalog/doc.go:4-8`; `docs/architecture.md:114`, `:268`, `:282`; the MADR's F1 gets a dated note.

### Phase 4: gobble's pin of go-llmprovider-sdk `v1.33.0` (M54, M55, M62, M71 to M73)

Starts when the SDK tags `v1.33.0` with its 0093 built. It is the same change as the pin. First re-ground F4 at the tag: the
removed names and the hand-in's name and contract as the SDK's 0093-MADR and its build record them; a difference is a stop.

| Rows | Test | Package | Red at the base because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| M62 | `TestListingGetsModelData_0072` (no metadata request; the listing ranked by the handed document; none handed: ranked without one) | `llm/provider` | the listing names a URL (`v1.32.0`'s code does not compile on `v1.33.0`; the test's red is taken from its plant) | the document not handed to the listing |
| M62 | `TestProviderGetsSessionModelData_0072` (the turn's, compaction's, a switched and the title's `provider.New` each given the session's document; a new session the current one) | `acpserver`, `llm/provider` | as above | the title's provider built with the current document |
| M55 | `TestLookupGetsSessionModelData_0072` (a lookup reads the session's document, handed in, not as a layer) | `llm/provider` | as M62 | the current document handed in place of the session's |
| M71 | `TestSessionProviderRebuiltOnFirstCopy_0072` (a none-session with a prompt blocked mid-call when the first download lands: that call completes on the old provider; the next prompt's provider holds the document; a session with a document keeps its provider) | `acpserver` | new name | `s.provider` not cleared; then every session's cleared |
| M72 | `TestModelDataRefreshCost_0072` (a counting provider builder across a refresh: one build per none-session, zero otherwise; `weak.Pointer` to the replaced document reads nil after the last holding session ends and a collection) | `acpserver`, `internal/cli` | new name | every document kept in a process-wide slice |
| M73 | `TestOpenCodeRoutesFromSessionModelData_0072` (an OpenCode model whose route the session's document names: the request goes to that route; with none, the table's) | `llm/provider` | `v1.32.0` turns the document off in `New` | `WithoutModelMetadata` kept in `New` |
| M54 | 0071-PLAN's V1 test, with the document handed in | `acpserver` | 0071's Phase 2 | (0071's) |

Build: replace `WithModelMetadataURL` (`list.go:129-130`, `:159-160`) and the layer (`model.go:107-110`) with the SDK's
hand-in; remove `Options.ModelMetadataURL`; `provider.New` given `Options.ModelData` (the session's document) in place of
`WithoutModelMetadata` (`provider.go:215`), for every provider built for a session; D22's rebuild of a none-session's
provider when the first download lands. Then 0071's Phase 2 begins.

Gates: as Phase 1.

Docs: `llm/provider`'s comments; `docs/architecture.md`'s `llm/provider` row; the guide's line on the listing's source.

## Verification

* Every decided row has its test, red at the base or held by its plant, and green after its phase.
* Every plant turns its own test red, its failing line quoted in the execution record.
* `make preflight` passes on the three hosts, `make race` on the Linux machine and `make check-test-isolation` at the end of
  each build phase; `make check-records`, `make markdownlint` and `denyscan.py --changed` pass.
* No test reaches the network: both guards' reports are empty in every run, and no binary test leaves a copy.
* An independent review of each build phase finds no blocker.

## Rollout and Rollback

* Rollout: Phases 1 to 3 in one gobble release, a minor by
  `0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md`'s rule; Phase 4 in the release that pins `v1.33.0`.
  Tags are the owner's.
* Rollback: `modelData.refreshHours: 0` stops every automatic download at once. Deleting `<cache>/model-data/` makes the next
  model call download afresh. Reverting Phase 3 restores the built-in fallback; reverting Phases 1 and 2 restores the
  embedded copy alone, and the cache files are then ignored. Phase 4 cannot be reverted apart from the pin.

## Execution record

*(Empty: nothing is built before the owner's approval.)*
