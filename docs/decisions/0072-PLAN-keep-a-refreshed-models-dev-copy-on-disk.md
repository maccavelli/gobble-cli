---
status: in-progress
date: 2026-11-25
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

*(2026-11-25, deviation, the owner's "gobble: v1.33.0 pin first, then 0072 (Recommended)": gobble pins go-llmprovider-sdk
`v1.33.0` in 0071-PLAN's Phase 2 before this PLAN starts, so every phase here runs on `v1.33.0`; M22's step is struck; Phase 4
is built as Phase 2b, between Phases 2 and 3, and is no longer the pin. See "Deviation (2026-11-25): pin first" under the
execution record. Steps and rows are struck or annotated in place, not deleted.)*

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
* *(2026-11-25, superseding the three bullets above by "gobble: v1.33.0 pin first, then 0072 (Recommended)".)* **Base:** the
  commit that ends 0071-PLAN's Phase 2c, on `v1.33.0` (0070 is built, in 073fd1f). **Order:** 0071-PLAN's Phases 2.0, 2a (the
  pin), 2b and 2c; then this PLAN's Phases 0, 1, 2, 2b (was 4) and 3, each on `v1.33.0`. gobble compiles at every step: the pin
  removes `WithModelMetadataURL` and `WithoutModelMetadata` itself (0071 P2). Phase 3 depends on Phase 2b; Phase 2b on Phase 2.

## Scope

### In scope

* `llm/provider`: `FetchModelData`, `ModelData`, `ParseModelData`, `CurrentModelData`, `SetCurrentModelData`,
  `Options.ModelData`; the atomic holder in place of `snapshot`; a `TestMain` with the network guard; the listing's URL
  (Phases 2) and the hand-in (Phase 4). *(2026-11-25: ~~the listing's URL~~ is not built; `ModelData`, `ParseModelData` and
  `Options.ModelData` exist from 0071's pin (its P3), and this PLAN adds `Stamp()`, `Sections()` and the rest of D21; the
  hand-in is Phase 2b.)*
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
   *(2026-11-25: at this PLAN's base and the SDK's `v1.33.0`; the MADR's amendment of that date records the moves to 073fd1f.)*
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
   commands; note the lines Phases 2 and 3 change. *(2026-11-25: `ModelMetadataURL` is gone with 0071's pin; grep instead for
   `Options.ModelData`, the built-in copy's use in the listings (0071 P6) and `WithMetadata`.)*

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
   copy as the fallback until Phase 3; `FetchModelData` (D4); `TestMain` with the guard. *(2026-11-25: `ModelData` and
   `ParseModelData` exist (0071 P3); this step adds `Stamp()`, `Sections()` and the holder. Where 0071's pin hands the
   listings the built-in copy (its P6 under Q5 (1)), `internal/cli` hands `CurrentModelData()` instead; M19's test gains a
   `listing` case: a listing's `Options.ModelData` is the current document. Under Q5 (2) the listings start reading a document
   here; under (3) `New` and the lookups too, until Phase 2b hands each session its own.)*
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
| ~~M22~~ | ~~`TestListingUsesModelsDev_0072` (the listing's metadata GET goes to models.dev's URL, served in memory)~~ *(2026-11-25: struck with M22; no URL exists at `v1.33.0`)* | `llm/provider`, `internal/cli` | the listing names no URL in production | the URL left unset outside tests |
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
4. ~~`llm/provider`: production sets `Options.ModelMetadataURL` to models.dev's URL for every listing (D17); `provider.go:53-55`'s
   "for tests" corrected.~~ *(2026-11-25: struck; the field is gone with 0071's pin, and the listing's document is Phase 1's.)*
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

*(2026-11-25: on `v1.33.0`, after Phase 2b. The fallback it removes is the holder's built-in copy; where 0071's pin handed it
to the listings or the lookups (0071 Q5), the current copy has replaced it since Phases 1 and 2b.)*

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

*(2026-11-25: built as **Phase 2b, the hand-in**, after Phase 2 and before Phase 3, on `v1.33.0`, which 0071's pin already
holds; it is not the pin. F4 is re-grounded at the tag in the MADR's amendment of this date: the hand-in is
`catalog.WithMetadata`; a difference found at this phase's start is a stop. The rows' "red at the base" is at Phase 2's end:
on its assertion where the code exists (the lookups hold the built-in copy as a layer, or none; `New` none), by the plant
where a name is new. M54's row is `TestTurnCapFromLimits_0071` with a `session_document` case.)*

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

*(2026-11-25, Phase 2b's build: ~~replace `WithModelMetadataURL`~~, ~~remove `Options.ModelMetadataURL`~~ and ~~in place of
`WithoutModelMetadata`~~ are done by 0071's pin. Phase 2b replaces the built-in copy's layer in the lookups and the limits
calls (`model.go:107-111`) with `catalog.WithMetadata` of the session's document; hands the session's document to `New` for
every provider built for a session, where the pin handed none (or, under 0071 Q5 (3), the built-in copy); D22's rebuild. ~~Then
0071's Phase 2 begins~~: it is already built. Docs: `llm/provider`'s comments, the `llm/provider` row of `docs/architecture.md`,
the guide's line on what the listing reads.)*

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
* *(2026-11-25: Phases 1, 2, 2b and 3 ship in one gobble release on `v1.33.0`, after the release that carries 0071's Phase 2.
  Phase 2b reverts on its own: the lookups return to the built-in copy's layer, or to none (0071 Q5), and `New` to none.)*

## Execution record

*(Empty: nothing is built before the owner's approval.)*

### Deviation (2026-11-25): pin first

The owner's ruling, relayed by the coordinator, quoted exactly: **"gobble: v1.33.0 pin first, then 0072 (Recommended)"**.
Found: this PLAN ran Phases 1 to 3 on `v1.32.0` and made Phase 4 the pin; the ruling pins `v1.33.0` first, in
[0071-PLAN-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-PLAN-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md)'s
Phase 2, with 0065's and 0066's adoption amendments. Decision, recorded in the MADR's amendment of this date:

* **Order:** 0071-PLAN's Phases 2.0, 2a (the pin), 2b, 2c; then this PLAN's Phase 0, Phase 1, Phase 2, Phase 2b (was Phase 4,
  the hand-in) and Phase 3, all on `v1.33.0`. The base is the commit that ends 0071's Phase 2c.
* **Struck, not deleted:** M22's row in Phase 2's table and Phase 2's build step 4 (the `v1.32.0` listing URL); the Scope's
  "the listing's URL"; Phase 4's "same change as the pin", its removals and its "Then 0071's Phase 2 begins".
* **Annotated:** "Dependencies and order"; Phase 0's steps 1 and 6; Phase 1's build step 1 (M19's `listing` case); Phase 3's
  heading; Phase 4's heading and build; "Rollout and Rollback".
* **Moved to 0071's pin (its P2 to P5):** the removal of `WithModelMetadataURL`, `Options.ModelMetadataURL` and
  `WithoutModelMetadata` at `llm/provider/list.go:129-130`, `:159-160`, `provider.go:53-55`, `:215`, `model.go:176`,
  `signin.go:344`; `ModelData`, `ParseModelData`, `Options.ModelData` (D21's first three, nil handing none); the tests that
  served a document by URL.
* **Re-pointed:** M54 (0071's Phase 2 first; `TestTurnCapFromLimits_0071/session_document` in Phase 2b), M55 and M62 (built
  in Phase 2b, their reds at Phase 2's end), M71 to M73 (Phase 2b).
* **Files added to any phase's scope:** none beyond the rows' own.
* **Waits on:** 0071's Q5, which sets what Phase 1 and Phase 2b replace. No row here is open.

### Deviation (2026-11-25, second): 0071's Q5 and Q6 answered

0071's Q5, "Built-in copy to listings only (Recommended)", and Q6, "Hand them in as a caller layer (Recommended)", relayed by
the coordinator, are answered; nothing waits. The MADR's second addition adds M74 to M76.

| Phase | Rows | Test | Red at the phase's base because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| 1 | M74 | `TestModelDataCurrent_0072/listing` (a listing's `Options.ModelData` is the current document, a copy that differs from the built-in one); 0071's `TestListingRanksByBuiltInCopy_0071` is re-pointed to it, its name in the execution record | the pin hands the built-in copy | the listing keeps the built-in copy |
| 2b | M75 | `TestLookupGetsSessionModelData_0072`; 0066's `TestLimits_BuiltInCopyBelowListing_0066` retired, its name in the execution record | the lookups hold the built-in copy's layer | M55's plant, as written |
| 2b | M76 | `TestProviderGetsSessionModelData_0072/saved_caps` (each `New` for a session gets the saved caps' layer below models.json beside the session's document) | a guard: the pin hands the layer | the layer left off `New` when the document is handed |

Phase 1's build step 1 and Phase 2b's build are as annotated; the struck `Q5 (2)` and `Q5 (3)` clauses in them are void.

### Phases 0 and 1 execution entry (2026-11-25)

The owner approved this PLAN with the MADR, relayed by the coordinator, quoted exactly: **"Accept; approve; queue after 0070
(Recommended)"**; and ordered it after the pin, quoted exactly: **"gobble: v1.33.0 pin first, then 0072 (Recommended)"**. 0070
and the pin (0071-PLAN's Phases 2.0 to 2c) have landed. This pass builds Phases 0 and 1 as the first deviation of this date
orders them, from the base `927c7dd` (the commit that ends 0071's Phase 2c, go-llmprovider-sdk `v1.33.0` at `go.mod:16`); the
tree was clean there (`git status --porcelain` empty). Phase 2 is not started. The agent stages nothing and commits nothing.

### Phase 0, pass 1 (2026-11-25, at `927c7dd` and go-llmprovider-sdk `v1.33.0`)

Scratch: `<scratch>/b0072/` (the scripts `cites.py` and `guard_probe.py`, the throwaway test `p0_confirm_test.go`, the clone
`p0` of `927c7dd`; logs in `logs/`). The SDK was read in the module cache at its tag.

* **Step 1, re-ground.** Every range the MADR and this PLAN cite was printed at `927c7dd` (`cites.py`, `logs/cites.txt`) and
  read. F1 to F12 hold; no fact changed. Moved since the MADR's amendment (at 073fd1f), by 0071's Phases 2a to 2c:
  * `acpserver/agent.go`: `Initialize`'s "makes no network call" (`:457-458`) is `:517-518`, its image check `:524`
    (`promptImage`, `:1186-1193`, is `:1320-1337`); `sessionProvider` (`:1141-1165`) is `:1270-1294`; a prompt takes its
    provider at `:766` (was `:701`). `acpserver/compact.go:81` is `:85`; `acpserver/config.go:320-336` (`buildProvider`) is
    `:348-364`.
  * `llm/provider/model.go`: the built-in copy's layer (`:107-111`) is `lookupLayers`, `:157-173` (the copy at `:163-165`),
    which `modelInfoFrom` (`:190-191`), `Discover` (`:127`) and 0071's `CachedLimits` (`:139-151`) read; `KnownModels`
    (`:142-151`) is `:270-279`; `capSourced` (`:126-140`) is `:213-241`; `:176`'s `WithoutModelMetadata` is gone (0071 P2).
    `llm/provider/provider.go:53-55` is `HTTPClient` now and `Options.ModelData` is `:54-58` (0071 P3); `list.go`'s URL sites
    are `o.ModelData.options()` at `:130` and `:158`.
  * `internal/cli/agent.go`: `SnapshotErr` is logged at `:117` (was `:87`); the `ModelInfo` hook is `:141-143` and the
    `CachedLimits` hook `:148-153` (0071 P46). `internal/cli/models.go:21-34`, `builtInModelData`, is the built-in copy the
    listings get (`:323`; 0071 P6). `internal/cli/completion.go:129` reads `offlineModels` (`modelflags.go:42-55`), which reads
    `KnownModels` and the lookups. `internal/cli/testmain_test.go`: `TestNoBrowserOrSignIn` is `:70-84`, the seams `:142-146`,
    `TestMain` and the guard `:105-157` and `:178-324`.
  * Unchanged: F1 (`llm/catalog/catalog.go:13-23`, `source.json:1-16`, `NOTICE:22-28`, `scripts/gen_catalog.py:6-8`, `:31`,
    `:36-40`, `:59-78`), F6, F7, F8's binary tests, F9's 0008 line, F10, F12 (`internal/config/keys.go:78-145` as amended),
    `docs/architecture.md:114`, `:268`, `:282`, `llm/catalog/doc.go:4-8`.
  * **0071's `CachedLimits`.** `provider.CachedLimits` (`model.go:139-151`) and the hook `acpserver.Options.CachedLimits`
    (`acpserver/agent.go:137`, read by `describeModel`, `acpserver/config.go:243-262`) read `lookupLayers`, so from Phase 1 they
    read the current document, as every lookup does (M19). M55 and M62 give a session's lookups and limits calls its document
    in Phase 2b; the hook, like `Options.ModelInfo`, takes a provider and a model and no session (Q-D below).
* **Step 2, the confirmations** (`p0_confirm_test.go` on the clone, `logs/p0-confirm.log`; the redirect served by an in-memory
  transport between two `.invalid` hosts):
  * **M2: not byte-exact.** `Encode` of the 2026-10-14 copy is 515,680 bytes; a `jsontext.Value` field written by
    `jsonx.MarshalStrict` and read back by `jsonx.UnmarshalStrict` holds 515,648: the marshal compacts the value (its final
    newline and white space dropped; a pretty-printed 94-byte document reads back as 67 bytes). The read-back form is stable: a
    second round trip is byte-identical. So, as this step decides, D2's `sha256` is taken over the re-encoded (compact) form,
    which the copy then holds byte for byte; the MADR's third addition records it. The strict policy also refuses a document
    whose kept sections repeat a name (`duplicate object member name "openai" within "/document"`) or hold invalid UTF-8
    (`invalid UTF-8 within "/document/openai/models/a/name"`), both of which `ParseMetadata` accepts: such a download cannot be
    saved, which is M69's row (used in memory for the run, `not_saved`).
  * **M15: not as the step expects.** A section whose bytes do not decode makes `ParseMetadata` refuse the whole document
    (`model metadata: decode openai: json: cannot unmarshal JSON number into Go map[string]catalog.modelMetadata within
    "/models"`), as F5 states, so `Encode` never meets it. A section present without `models`, or `null`, parses, and `Encode`
    drops it; `{"models":{}}` is kept, with no model (Q-B below).
  * **M58: confirmed.** A 302 from `https://models.invalid/api.json` to `https://mirror.invalid/doc.json` is followed by Go's
    client, and `resp.Request.URL` is the mirror's, not the URL asked for.
* **Step 3, tests that run gobble's binary or a whole run without the seams:** `acpserver/binary_test.go:87-93` and
  `cmd/gobble/conpty_windows_test.go:1091-1096`, which M64 names, and `internal/cli/complete/binary_test.go:85`
  (`TestBinaryCompletionMode`: it builds `./cmd/gobble` and runs it in completion mode under `isolatedEnv`, `:56-68`). That one
  can take a closed-port proxy (`isolatedEnv`'s `kv`), so it joins M64's list by the MADR's third addition. The pty and conpty
  tests in `internal/cli`, `update_test.go` and `editor/conhost_windows_test.go` run the test binary itself, behind its
  `TestMain`.
* **Step 4.** Tests that read the built-in copy, for Phase 3 (M48), at least: `acpserver/discover_test.go`; in
  `internal/cli`, `auth_test.go`, `choice_test.go`, `limit_test.go`, `modelflags_test.go`, `oauthsessions_test.go`,
  `observation_test.go`, `providerscmd_test.go`, `sdkcache_test.go`, `testmain_test.go`; `llm/catalog/catalog_test.go`; in
  `llm/provider`, `discover_test.go`, `login_test.go`, `model_test.go`, `savedcaps_test.go`, `sdkcache_test.go`,
  `sessions_test.go` (and the live `live_glm_cap_test.go`). Tests whose output Phase 2 changes: the `/context` and `/session`
  tests (`acpserver/catalog_test.go`, `commands_test.go`, `outputlimit_test.go`, `replies_test.go`) assert with
  `strings.Contains` or `HasPrefix`, so a line after 0071's changes none of them; of the `gobble models info` tests only
  `TestModelsInfo` (`internal/cli/modelscmd_test.go:112-126`) compares stdout exactly, so M38's `data` line changes its
  expected output, and M37 to M39 do not name it (Q-E below).
* **Step 5.** `acpserver` imports `llm/provider` (`acpserver/auth.go:14`), so it may hold a `*provider.ModelData`.
  `internal/archtest`'s rules allow Phase 1's imports (only `llm/provider` imports the SDK; `internal/cli` already imports
  `llm/provider`, `internal/fsx`, `internal/appdirs` and `jsonx`); rule 11 reads `[]byte` fields, and the copy's `document` is
  a `jsontext.Value`.
* **Step 6, docs.** Phase 1: `docs/architecture.md:113` (`llm/provider`: "the lookups keep the embedded snapshot as their
  layer"), `llm/provider/provider.go:54-57` (`Options.ModelData`: "Only List reads it") and `internal/cli/models.go:21-27`
  (`builtInModelData`). Phase 2: `README.md:99-101` and `docs/guides/choosing-a-model.md` (no line names the facts' age or a
  refresh). Phase 3: `docs/architecture.md:114`, `:268`, `:282`, `NOTICE:22-28`, `llm/catalog/doc.go:4-8`,
  `scripts/gen_catalog.py:6-8`. `docs/README.md:240` (the SDK's listing fetching models.dev, as of `v1.15.0`) is a run's
  history and stays.
* **D10's guard in `llm/provider`, probed.** `internal/cli`'s guard, copied as is behind a `TestMain` on the clone
  (`guard_probe.py`, `logs/p0-guard-run.log`): 203 of the package's tests pass, but `TestWireOffline/unknown_host` fails
  (`offline_test.go:67: Probe dialed ["tcp 127.0.0.1:56925"], want tcp gobble-offline.invalid:80 alone`: `Probe` dials the
  proxy the environment names) and `TestProbePort` skips (`offline_test.go:106: the environment names a proxy for
  https://api.example.test/v1/responses …`). Neither is a row's test (Q-C below).

Exit of Phase 0: citations re-grounded; the confirmations and lists recorded; nothing but the two records changed. Two
confirmations and two lists raise cases the record does not decide, and D21 cannot carry Phase 1's build: Phase 1 stops
before it.

### Phase 1, pass 1 (2026-11-25, from `927c7dd`): stopped before the build

No source or test file changed. The cases below go to the owner in one prompt, each with three resolutions, the recommended
first; none is a workaround. No row is changed until they are answered.

* **Q-A. The exported surface Phase 1 needs (D21).** D21 lists `FetchModelData`, `ModelData` with `Stamp()` and `Sections()`,
  `ParseModelData`, `CurrentModelData` and `SetCurrentModelData`, and this PLAN's rules make "an exported name not in D21" a
  stop. `internal/cli`, which may not import the SDK (F7), needs from `llm/provider`: the stamp's fields (source, fetched,
  ETag, Last-Modified) as `Stamp()`'s result; a way to stamp a document read from the copy (0071's `ParseModelData(r)` takes a
  reader alone); `Encode`'s bytes, for the copy's `document` and its sha256; whether a reply was a 304; and M23's constant,
  which `TestFetchModelData_0072` asserts in `llm/provider` and `internal/cli` passes. None has a name in D21.
  1. *(Recommended)* D21 gains four names: `ModelDataStamp` (`Source`, `Fetched`, `ETag`, `LastModified`), `Stamp()`'s type;
     `(*ModelData).Stamped(ModelDataStamp) *ModelData`, the same parsed document under another stamp (a copy read from disk, a
     304's renewal); `(*ModelData).Document() []byte`, `Encode`'s bytes, compact, the caller's own; and `ModelDataURL`, the
     constant. `FetchModelData(ctx, url, o) (*ModelData, bool, error)` sends the validators of `o.ModelData` when its stamp's
     source is `url`, and the bool is a 304. Cost: D21 amended; one parse per download, as D23 says.
  2. The copy's JSON lives in `llm/provider`: `ModelDataStamp`, `ReadModelDataCopy([]byte)` and `(*ModelData).CopyJSON()`,
     which write D2's whole object, and `ModelDataURL`; `internal/cli` keeps the path, the temporary file and rename, the sweep,
     the guards and `attempt.json`. Cost: four names as well, and M2's and M4's tests move to `llm/provider`.
  3. No new name: `Stamp()` returns `(source string, fetched time.Time, etag, lastModified string)`; `ParseModelData` takes
     those four as arguments (0071's two callers change); `FetchModelData` returns `(doc []byte, d *ModelData, notModified bool,
     err error)`; the URL is a constant in `internal/cli`, and M23's assertion moves there. Cost: 0071's exported
     `ParseModelData` changes, and two calls carry four and five values.
  * Doing nothing: Phase 1 cannot be built within D21.
* **Q-B. M15's `section_malformed`** (step 2's measurement).
  1. *(Recommended)* As measured: a download with a section that does not decode is M13's `not_document` failure
     (`attempt.json` `failed`; the copy kept; `--force` cannot save what the SDK refuses), and M15's `section_malformed` case is
     a section the SDK reads as absent (no `models`, or `null`), refused as missing. A dated note on D6 and M15. Cost: such a
     download is reported as a failed fetch, not a refusal.
  2. gobble reads a `ParseMetadata` error that names a section (`model metadata: decode <key>:`) as the guard's refusal, so
     M15 holds as written. Cost: it matches the SDK's error text, which no typed error carries at `v1.33.0`; `--force` still
     cannot save it.
  3. gobble decodes the download's sections itself and drops one that does not decode before `ParseMetadata`. Cost: a second
     parse of every download (D23 says one) and gobble's own copy of the SDK's section decoding.
  * Doing nothing: `TestModelDataGuards_0072/section_malformed` cannot be written as M15 states it.
* **Q-C. D10's network guard in `llm/provider`** (the probe above).
  1. *(Recommended)* `llm/provider`'s `TestMain` runs the guard as `internal/cli`'s does, the tests in a guarded child, and
     runs `TestWireOffline` and `TestProbePort`, whose assertions need a dial with no proxy (a refused loopback port, a
     `.invalid` name), in a second child without the proxy variables; the guarded child leaves those two out
     (`-test.skip`). No assertion changes. Cost: about 40 lines of `TestMain`; those two tests stay unguarded, as today.
  2. As 1, with the guard moved to a test-support package both `TestMain`s import. Cost: as 1, a refactor of
     `internal/cli/testmain_test.go` and a new internal package for `internal/archtest`'s rules.
  3. No proxy guard in `llm/provider`: M10 is held by the in-memory transport alone (a fetch on another client fails its
     test, since the transport records no request); D10 amended. Cost: an escape from any other `llm/provider` test goes
     unseen.
  * Doing nothing: the guard as D10 states it turns `TestWireOffline` red and `TestProbePort` to a skip.
* **Q-D. The session's document and `acpserver`'s two hooks (Phases 2 and 2b; 0071's `CachedLimits`).**
  `acpserver.Options.ModelInfo` and `Options.CachedLimits` (`acpserver/agent.go:137`; 0071 P46) are `func(provider, model
  string)`. D19 has a session's calls, `/context` and compaction read its document, and M55 and M62 every lookup and limits
  call for a session; Phase 2's build step 3 passes the document "through `Options.ModelInfo`'s `provider.Options.ModelData`",
  which the hook's signature cannot carry. Eleven literals in nine `acpserver` test files set the two hooks.
  1. *(Recommended)* Both hooks take the session's `*provider.ModelData` first (nil: the current); `acpserver` passes the
     session's; `internal/cli` sets it as `lookupOptions`' `Options.ModelData`, which the lookups and `provider.CachedLimits`
     read (Phase 2 for `ModelInfo`, Phase 2b for `CachedLimits`). Cost: two exported signatures change, added to D21; the
     eleven literals take the argument, no assertion changes.
  2. As 1 for `ModelInfo` only; the model option's descriptions (`CachedLimits`) read the current document. Cost: a session's
     menu may describe a model by a newer document than its calls use (no request either way).
  3. Two new hooks that take the document, the old ones kept. Cost: four hooks for two jobs, against the no-compatibility
     rule.
  * Doing nothing: M50 (Phase 2) and M55 (Phase 2b) cannot reach the session's document.
* **Q-E. `TestModelsInfo` and M38 (Phase 2)** (step 4).
  1. *(Recommended)* `TestModelsInfo` takes the `data` line into its expected output in Phase 2, as M38's consequence, by a
     dated note (as 0071's P45 changed it). Cost: one expected string.
  2. `TestModelsInfo` is retired into `TestModelsInfoData_0072`, which then asserts the whole output; its name in the
     execution record. Cost: one test fewer, the same assertions.
  3. `gobble models info` prints the `data` line only while a copy is in use. Cost: D12 changes, and the "none" line is not
     shown there.
  * Doing nothing: Phase 2 stops at M38.

Pass count: Phase 0, one pass; Phase 1, one pass, stopped before the build. Gates: the records' alone (nothing else changed):
`make check-records`, `make markdownlint`, `denyscan.py --changed` and the disclosure redact dry run. No test, pre-add or
preflight gate ran: no source or test file changed. Exit codes taken before any filter (`<scratch>/b0072/localgates.py`; logs
`<scratch>/b0072/logs/gate-p0stop-*.log`).

| Command | Where | Exit |
| :--- | :--- | ---: |
| `make check-records` | this Mac | 0 |
| `make markdownlint` | this Mac | 0, 0 issues in 59 files |
| `denyscan.py --root <gobble> --changed` | this Mac | 0, 0 hits in 3 files |
| `github-disclosure.py redact` (dry run) | this Mac | 0, would edit 0 files |

### Deviation (2026-11-25): the answers to Phase 1's stop, pass 1 continuing

Relayed by the coordinator, quoted exactly: Q-A **"Add the four names (Recommended)"**; Q-B **"As a failed download
(Recommended)"**; Q-D **"Hooks take the session's document (Recommended)"**. Q-C and Q-E are the coordinator's calls on the
owner's ruling for the same kind of case, 0071's Q-H **"Expect the new output (Recommended)"**: Q-C, "`offline_test.go:67`
and `TestProbePort` run in a second child process without proxy variables. No assertion changes."; Q-E, `TestModelsInfo`'s
expected string gains M38's `data` line by a dated note. The MADR's fourth addition records them as M77 to M81 and annotates
D10, D21 and M15.

| Phase | Rows | Test | Red at the phase's base because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| 1 | M77 | `TestModelDataStamp_0072` (`llm/provider`: `Stamped` keeps the parse and leaves the original's stamp; `Document()` is compact, equals `Encode`'s bytes compacted, and is the caller's own copy; `ModelDataURL`) | new names | `Stamped` changing the original's stamp in place |
| 1 | M78 | `TestModelDataRefreshFails_0072/section_undecodable` (copy, current document and `attempt.json`: `failed`, `not_document`); `TestModelDataGuards_0072/section_malformed` (a section present without `models`: refused) | new names | an error naming a section's decode read as the guard's refusal |
| 1 | M79 | `llm/provider`'s `TestMain`: the unguarded child checks that `net/http` names no proxy; `TestWireOffline`, `TestProbePort` run there, unchanged | the guard, as probed in Phase 0, turns `TestWireOffline` red and `TestProbePort` to a skip | the proxy variables left in the unguarded child's environment |
| 2 | M81 | `TestModelsInfo` | M38's `data` line | (M38's) |
| 2, 2b | M80 | M50's and M55's tests | as those rows | as those rows |

* **Files added to Phase 1's scope:** `llm/provider/testmain_test.go` (new: `TestMain`, a copy of `internal/cli`'s guard, and
  the unguarded child). M74's re-pointing of `TestListingRanksByBuiltInCopy_0071` (`internal/cli/limit_test.go:286`) is the
  second deviation's, unchanged.
* **Phase 2 and 2b gain:** the hooks' signatures (M80) and `TestModelsInfo`'s expected output (M81); not built here.

### Phase 1, pass 1 continued (2026-11-25): built

Built in the tree (unstaged), each edit a recorded script in `<scratch>/b0072/` (`spec*.json` through `subst.py`, which asserts
each anchor once; `make_testmain.py`, `repoint_m74.py`, `lintfix1.py`).

* **`llm/provider`.** `modeldata.go`: `ModelDataURL`; `ModelDataStamp`; `ModelData` holds one shared parse
  (`parsedModelData`) and its stamp; `Stamp()`, `Stamped()`, `Document()` (the SDK's `Encode`, written once through
  `jsonx.MarshalStrict` to its compact form; a document the strict form refuses keeps `Encode`'s bytes, which then cannot be
  saved, M69) and `Sections()` (models per section key, sections with a model); the holder `CurrentModelData` /
  `SetCurrentModelData` (an atomic pointer; the built-in copy, parsed once with the zero stamp, until one is set), replacing
  `snapshot` in `lookupLayers` and `KnownModels`, `SnapshotErr` reading the built-in parse's error; `FetchModelData(ctx, url,
  o) (*ModelData, bool, error)` (M77). A failed fetch's error is ctx's when ctx ended it, wraps `llm.ErrOffline` when the
  request or its body could not be carried, is an `*llm.APIError` for a status other than 200 or 304 (and for a 304 to a request
  that sent no validator), and is otherwise "not a models.dev document"; `internal/cli` names these `timeout`, `canceled`,
  `offline`, `status` and `not_document` in `attempt.json` (M11 to M13 name no kind beyond "as M11"; a body over the limit is
  `not_document`). `testmain_test.go` (new): M79's `TestMain` with `internal/cli`'s guard copied as is.
* **`internal/cli`.** `modeldata.go` (new): the paths (D1); the copy's object and its encode, decode (D3's checks), read
  (`fsx.ReadRegularFile`, 32 MiB) and write (`os.CreateTemp` in the private directory, a pause seam, `fsx.ReplaceFile`, the
  temporary file removed on a failure, the sweep after an accepted write); D6's guards; `attempt.json` (D18); the engine:
  `startModelDataRefresh` runs one refresh on the run's listing group (owned and ended by the run, 0046 D2) and returns at once,
  and `refreshModelData` marks the attempt started, reads the copy again, fetches under the mode's deadline (30 s, 120 s) with
  the copy as `Options.ModelData`, guards, makes the download current, saves, and marks the result; warnings are logged once
  per run each (`warnModelData`, as "model data" with the text as its `warning`). Seams: `fetchModelData`, `loadModelData`
  (set by `TestMain` to an offline fetch and a reader of no file, D10), `modelDataSource`, `modelDataNow`,
  `modelDataBeforeRename`, `modelDataReplace`. `models.go`: `builtInModelData` removed; the listing's `Options.ModelData` is
  `provider.CurrentModelData()` (M74). `main.go`: `runEnv.modelDataWarned`. Nothing calls the engine in production yet: the
  trigger and the load at the run's start are Phase 2's (D7, D8).
* **Existing tests changed.** `TestListingRanksByBuiltInCopy_0071` (`internal/cli/limit_test.go:286`) is retired into
  `TestModelDataCurrent_0072/listing` (M74, the second deviation). `TestLimits_BuiltInCopyBelowListing_0066`
  (`llm/provider/discover_test.go:359`) read the removed `snapshot()`: it reads the built-in parse (`builtIn()`), no assertion
  changed (the build step's "replacing `snapshot`"; the test is M75's to retire in Phase 2b). `internal/cli`'s `TestMain` sets
  the two seams; `TestNoBrowserOrSignIn`'s identity check of them is M34's, Phase 2.
* **Rows to tests** (all green after the build; `logs/cli-green1.log`, `logs/green2.log`):

| Rows | Test | Package | Red |
| :--- | :--- | :--- | :--- |
| M1 | `TestModelDataPath_0072` (`GOBBLE_CACHE_DIR`, `GOBBLE_HOME`, `XDG`, `default`, `windows`) | `internal/cli` | new name: its plants |
| M2 | `TestModelDataRoundTrip_0072` (`validators`, `no_validators`) | `internal/cli` | new name |
| M3 | `TestModelDataWriteKeepsOld_0072` (`temporary_file`, `rename`) | `internal/cli` | new name |
| M4 | `TestModelDataUnreadable_0072` (`not_json`, `truncated`, `version`, `sha256`, `refused`, `oversize`, `directory`) | `internal/cli` | new name |
| M5 | `TestModelDataConcurrentWriters_0072` | `internal/cli` | new name |
| M6 | `TestModelDataStaleTemps_0072` | `internal/cli` | new name |
| M7 | `TestModelDataHeldWindows_0072` (`replace`, `read`) | `internal/cli`, Windows | new name |
| M8, M23 | `TestFetchModelData_0072` | `llm/provider` | new name |
| M9 | `TestFetchModelDataConditional_0072` (`etag`, `last_modified`, `other_source`, `304_renews`, `304_unasked`; and the engine's: the copy's ETag sent, a 304 renewing the copy) | `llm/provider`, `internal/cli` | new name |
| M10 | `TestFetchModelDataUsesClient_0072`; `llm/provider`'s guard | `llm/provider` | new name |
| M11 to M13, M78 | `TestModelDataRefreshFails_0072` (`offline`, `status`, `oversize`, `not_document`, `section_undecodable`); `TestFetchModelDataFailures_0072` (the same five, as `FetchModelData` reports them) | `internal/cli`, `llm/provider` | new name |
| M14 to M17, M15/M78 | `TestModelDataGuards_0072` (`empty`, `empty_force`, `section_missing`, `section_malformed`, `shrink_below_half`, `shrink_half`, `first`, `other_source`, `force`) | `internal/cli` | new name |
| M19, M74 | `TestModelDataCurrent_0072` (`llm/provider`: `built_in`, `copy`, `never_both`; `internal/cli`: `listing`) | both | new name |
| M58 | `TestFetchModelDataRedirect_0072` | `llm/provider` | new name |
| M59 | `TestFetchModelDataDeadline_0072` (`synctest`; `llm/provider`: the caller's deadline; `internal/cli`: 30 s and 120 s) | both | new name |
| M65 | `TestModelDataEngineNeverBlocks_0072` (a scripted `-p` turn with a `read` tool call completes while the fetch blocks) | `internal/cli` | new name |
| M66 | `TestModelDataAttemptFile_0072` (`shape`, `unreadable`, `two_writers`) | `internal/cli` | new name |
| M69 | `TestModelDataNotSaved_0072` | `internal/cli` | new name |
| M77 | `TestModelDataStamp_0072`; M9's `304_renews` | `llm/provider` | new name |
| M79 | `TestWireOffline`, `TestProbePort` in the unguarded child | `llm/provider` | at the base, as Phase 0 probed: the guard turns `TestWireOffline/unknown_host` red (`offline_test.go:67`) and `TestProbePort` to a skip (`:106`); both pass in the unguarded child |

* **Plants** (`plants.py`, 29 and the control, each one change on a scratch copy of the final tree, `plantwork`, synced by
  `sync_tree.py`; every Phase 1 row test selected in both packages; `logs/final-plants.out`, `logs/plants-f2/`): every one
  caught, the control not. M7's runs on the Windows laptop (`plants_windows.py`, `logs/plant-M7-windows.log`).

| Plant | Row | Caught by | Failing line |
| :--- | :--- | :--- | :--- |
| M1a the state role | M1 | `TestModelDataPath_0072` (four cases) | `modeldata_test.go:203: copy …/h/model-data/models-dev.json, want …/c/…` |
| M1b `catalog/` | M1 | `TestModelDataPath_0072` (five cases) | `modeldata_test.go:203: copy …/c/catalog/models-dev.json, want …` |
| M2 `lastModified` not written | M2 | `…RoundTrip_0072/validators` | `modeldata_test.go:239: etag and lastModified written {"version":1,…,"etag":"W/\"abc\"","sha256":…` |
| M3 temporary file kept after a failed rename | M3 | `…WriteKeepsOld_0072/rename` | `modeldata_test.go:280: temporary files left: [….models-dev.json.833925206.tmp]` |
| M4 sha256 not checked | M4 | `…Unreadable_0072/sha256` | `modeldata_test.go:330: read {Source:https://models-dev.invalid/api.json …} as a copy` |
| M5 the copy's temporary file at one name | M5 | `…ConcurrentWriters_0072` | `modeldata_test.go:376: writer A: rename ….models-dev.json.fixed.tmp …` |
| M6a no removal | M6 | `…StaleTemps_0072` | `modeldata_test.go:410: left [fresh hour stale young], want [fresh young]` |
| M6b an hour-old file kept | M6 | `…StaleTemps_0072` | `modeldata_test.go:410: left [fresh hour young], want [fresh young]` |
| M7 `os.Rename` for `fsx.ReplaceFile` (Windows) | M7 | `…HeldWindows_0072/replace` | `modeldata_windows_test.go:31: the replace under a reader held for 200 ms: rename … The process cannot access the file because it is being used by another process.` |
| M8a a cached result | M8 | `TestFetchModelData_0072` | `modeldata_test.go:169: 1 requests, want two: one per call` |
| M23 the constant changed | M23 | `TestModelDataStamp_0072`, `TestFetchModelData_0072` | `modeldata_test.go:108: ModelDataURL "https://models.dev/api.json?v=2"` |
| M9 validators not sent | M9 | `…Conditional_0072/etag`, `/last_modified`, `/304_renews` | `modeldata_test.go:200: If-None-Match "", If-Modified-Since ""` |
| M10 `http.DefaultClient` for the run's | M10 | `…UsesClient_0072`, and every fetch test, which all send through the run's client | `modeldata_test.go:158: … llm: offline: Get "https://models-dev.invalid/api.json": Forbidden` (the guard's 403) |
| M11 the current document cleared on a failure | M11 to M13 | `…RefreshFails_0072` (all five cases) | `modeldata_test.go:462: a failed fetch changed the copy or the current document` |
| M78 a section's decode read as a refusal | M78 | `…RefreshFails_0072/section_undecodable` | `modeldata_test.go:459: result refused not_document, want failed not_document` |
| M14a totals compared, not sections | M14 to M17 | `…Guards_0072/section_missing`, `/section_malformed`, `/shrink_below_half` | `modeldata_test.go:503: result ok (), want refused, the copy kept` |
| M14b the source not compared | M17 | `…Guards_0072/other_source` | `modeldata_test.go:501: result refused (its openai section has 1 models, under half the copy's 10), want ok` |
| M19 the built-in copy layered below the copy | M19 | `…Current_0072/never_both` | `modeldata_test.go:367: claude-opus-4-6 reads "Claude Opus 4.6", window 1000000, from the built-in copy beside the current document` |
| M74 the listing hands no document | M74 | `…Current_0072/listing` | `modeldata_test.go:710: the listing's Options carry 0x0, want the current document …` |
| M58 the stamp from where the redirect went | M58 | `…Redirect_0072` | `modeldata_test.go:263: stamped "https://mirror.invalid/doc.json", want the URL asked for` |
| M59 one deadline for both | M59 | `…Deadline_0072` (`internal/cli`) | `modeldata_test.go:524: mode 1: timeout after 30s; want a timeout at 2m0s` |
| M65 the engine run inline | M65 | `…EngineNeverBlocks_0072` | `modeldata_test.go:555: starting the engine waited for its fetch` |
| M66 attempt.json written in place | M66 | `…AttemptFile_0072/shape`, `/two_writers` | `modeldata_test.go:638: the first writer never paused before its rename` |
| M69 a download not saved is dropped | M69 | `…NotSaved_0072` | `modeldata_test.go:672: result not_saved, current …, data …; want not_saved and the download current` |
| M77 `Stamped` changes the original | M77 | `TestModelDataStamp_0072`, M9's `304_renews` | `modeldata_test.go:118: stamped {…}, original {…}` |
| M79 the unguarded tests left in the guarded child | M79 | `TestWireOffline/unknown_host` | `offline_test.go:67: Probe dialed ["tcp 127.0.0.1:63421"], want tcp gobble-offline.invalid:80 alone` |

* **Deviation (2026-11-25), plants re-pointed** (first run `logs/plants1.out`): the PLAN's M2 plant (sha256 over the whole
  object) makes every seeded copy unreadable, so the guard and conditional tests caught it too; M2's is now `lastModified` not
  written. M3's (`os.WriteFile` onto the path) also stopped M5's writer pausing; M3's is now the temporary file kept after a
  failed rename, and M3's test makes the temporary file vanish before its rename in place of a creation seam. M5's is the
  copy's temporary file at one name (the PLAN's "truncated and written in place" was caught by M3 too). M6's second ("every age
  removed") also broke M5's paused writer; it is now an hour-old file kept. M8's cache is keyed by URL and client. M74's
  ("the listing keeps the built-in copy") cannot be made in `internal/cli` without the holder; it is the listing handed no
  document. M79's first form (the proxy kept in the unguarded child's environment) was not caught, since the first process
  carries no proxy variable; it is the two tests left in the guarded child. M10's plant fails every fetch test, as each sends
  through the run's client: recorded, not narrowed. No assertion changed.
* **Lint.** `make pre-add-check` found `contextcheck` (the warnings took no context), `errorlint`, `goconst`, `modernize`,
  `sloglint` (a non-constant message: the text is now the `warning` attribute of "model data") and `staticcheck` ST1008; fixed
  without a suppression, re-run: exit 0. The first preflights on the Linux machine and the Windows laptop found `errcheck`
  (`internal/cli/modeldata.go:410:5: Error return value of errors.AsType is not checked`) that this Mac's lint did not report;
  fixed (`ae, ok := …; ok && ae != nil`), and every gate ran again on the final tree.
* **Docs.** `docs/architecture.md:113` (`llm/provider`: the current document, `FetchModelData`, the stamp, the guard's carve-out)
  and `:135` (`internal/cli`: the store, `attempt.json`, the engine, the seams) — fixed; `llm/provider/provider.go:54-58`
  (`Options.ModelData`), `llm/provider/model.go`'s `ModelInfo`, `Discover` and `modelsJSONLayer` comments, `llm/limit.go:23`
  (`CapCatalog`) — fixed; `internal/cli/models.go`'s listing comment — fixed. Open, by phase: `README.md`,
  `docs/guides/choosing-a-model.md` (the age, the command, the settings: Phase 2); `docs/architecture.md:114`, `:268`, `:282`,
  `NOTICE`, `llm/catalog/doc.go` (the built-in copy: Phase 3).

#### Phase 1 gates (2026-11-25, the final tree)

Exit codes taken before any filter (`<scratch>/b0072/final_run.py`, `preadd.py`, `gates.py`, `windows_pkgs.py`,
`localgates.py`); logs `<scratch>/b0072/logs/`; no log has a fallback line; every heavy run through `run-heavy` or
`run-on-host`.

| Command | Where | Exit |
| :--- | :--- | ---: |
| `make pre-add-check FILES=…` (the 14 changed `.go` files), `run-heavy` | this Mac | 2 (six linters), then 0 |
| `go test -count=1 ./...`, `run-heavy` | this Mac | 0 |
| `make check-test-isolation`, `run-heavy` | this Mac | 0; 0 files written above the temp roots |
| `make preflight`, `run-on-host --need darwin` | this Mac | 0, "preflight passed" |
| `make preflight race`, `run-on-host --need linux --race` | the Linux machine (linux/amd64, not WSL) | 2 (errcheck), then 0, "preflight passed" |
| `make preflight`, `run-on-host --need windows` | the Windows laptop (windows/amd64) | 2 (errcheck), then 0, "preflight passed" |
| `go test -count=1 -v -run '_0072\|^(TestWireOffline\|TestProbePort)$' ./internal/cli/ ./llm/provider/`, `run-on-host --need windows` (Windows preflight runs no tests) | the Windows laptop | 0; `TestModelDataHeldWindows_0072` and its two cases pass |
| `make check-records`, `make markdownlint` | this Mac | 0; 0, 0 issues in 59 files |
| `denyscan.py --root <gobble> --changed` | this Mac | 0, 0 hits in 18 files |
| `github-disclosure.py redact` (dry run) | this Mac | 0, would edit 0 files |

Exit of Phase 1 (pass 1): every row of Phase 1 (M1 to M17, M19, M23, M58, M59, M65, M66, M69, M74, M77 to M79) tested, red
by its plant (M79 red at the base, as Phase 0 probed); 29 plants caught and the control not; gates recorded; docs rows
returned. One build pass. An independent review is the coordinator's.

### Deviation (2026-11-25): Phase 1's review, and pass 2

The coordinator relayed `code-reviewer`'s findings on Phase 1 and the owner's ruling on one, quoted exactly: **"Follow any,
but refuse https→http (Recommended)"** (the MADR's fifth addition, M82). Each finding is closed in **Phase 1, pass 2**, a new
pass, with a test red without its fix and a plant caught by it.

| Item | Finding | Fix | Test (red without the fix) | Plant |
| :--- | :--- | :--- | :--- | :--- |
| R1 (M82) | a redirect from `https` to `http` was followed | `FetchModelData`'s client copy refuses it in `CheckRedirect` (other redirects as Go's, up to ten); the error wraps `llm.ErrNotPermitted`, which `internal/cli` names `redirect_downgrade` | `TestFetchModelDataRedirect_0072/https_to_http`; `TestModelDataRefreshFails_0072/redirect_downgrade` | the downgrade check removed |
| R2 (M79) | the unguarded child inherited `-test.testlogfile` and `-test.coverprofile`, truncating the parent's files (a stale test cache, a wrong profile) | the child writes its own (the parent's path plus `.unguarded`), appended to the parent's after it ends without its header line; its CPU, memory, block, mutex and trace profiles go to their own `.unguarded` files | `TestUnguardedChildKeepsTestLog_0072` (`llm/provider`: the binary run with `-test.testlogfile` holds one header and both children's lines); the cache and the profile shown by `testlog_check.py` | the child's log path not given its own |
| R3 (M13) | the oversize cases could not fail on size: a string-valued `openai` is refused as undecodable | the padding sits in a key the SDK does not keep (`{"xai":…,"pad":"<33 MiB>"}`); `FetchModelData` enforces D4's 32 MiB itself, its own words ("the reply is over 32 MiB") | `TestFetchModelDataFailures_0072/oversize`, `TestModelDataRefreshFails_0072/oversize` assert those words | gobble's limit removed |
| R4 (M6) | the sweep's `filepath.Glob` breaks on `[`, `*`, `?` in the cache path | `os.ReadDir` and a prefix and suffix test | `TestModelDataStaleTemps_0072` under a directory named with `[` | `filepath.Glob` again |
| R5 (M6) | `attempt.json`'s temporary files were not swept | the sweep covers `.attempt.json.*.tmp` | `TestModelDataStaleTemps_0072` (a stale attempt temporary file) | the attempt prefix left out |
| R6 (M2, M3, M5, M66) | the copy and `attempt.json` were written without `Sync` or a directory sync | both through `fsx.WriteFileMode` (synced, the directory synced; 0o600 new; Windows' rename retries kept); its rename step gains a test seam, `fsx.SetRename`, as `fsx.SetOpenFile` serves the reads (an internal package's test seam) | M3, M5, M66 and M69's tests, through the seam | M3: the temporary file kept after a failed rename (in `internal/fsx`); M5: a fixed temporary name |
| R7a (M65) | `TestModelDataEngineNeverBlocks_0072`'s prompt half could not fail: the prompt read nothing the engine held | the engine's fetch is `FetchModelData` itself on a transport that blocks, and the prompt's turn reads the model facts (`CurrentModelData`) while it blocks | that test, its prompt bounded at 20 s | the holder locked for the length of a fetch |
| R7b (M14 to M17) | the guards' test did not assert the current document on a refusal | each refused case asserts `CurrentModelData()` unchanged | `TestModelDataGuards_0072` | a refused download made current |

* **Files added to Phase 1's scope:** `internal/fsx/write.go` (the rename seam, `SetRename`).
* **Follow-up, recorded, not built:** `writePrivate` (`internal/cli/choice.go:272`), the listing cache's writer, syncs neither
  its file nor its directory; it predates this record.
* **Phase 2 notes (no code now):**
  * The engine runs on the run's listing group, and `endListings` waits for that group up to `endListingsWait`, 6 s
    (`internal/cli/models.go:51`). D20 and M28 say `gobble acp` and the interactive CLI cancel a refresh at their exit at once
    and do not wait for a write, and `gobble acp` exits within 2 s: Phase 2 must give the engine an owner those exits do not
    wait on for longer than that (or end it before `endListings` waits), and M28's test must show it.
  * The hour's gate (D8, D18, M26) reads `attempt.json`'s `started` before an automatic refresh. `refreshModelData` writes
    `started` first (`internal/cli/modeldata.go`, at its first `recordModelDataAttempt`), so Phase 2's trigger must read
    `attempt.json` before that write, or the gate would see the process's own attempt.

### Phase 1, pass 2 (2026-11-25): the review's findings closed

Built in the tree (unstaged), each edit a recorded script in `<scratch>/b0072/` (`madr5.json`, `fsx_seam.json`,
`pass2_apply_tests.py`, `pass2_prov.json`, `pass2_testmain.json`, `pass2_fix_prov.json`, `pass2_fix_cli.json`).

* **Built.** `llm/provider/modeldata.go`: `refusingDowngrade` (a copy of the run's client whose `CheckRedirect` stops a redirect
  from `https` to `http`, wrapping `llm.ErrNotPermitted`, and otherwise defers to the client's own or follows up to ten, R1);
  `readErrors` reads at most `modelDataBodyLimit` (32 MiB) and reports "the reply is over 32 MiB" (R3). `internal/cli/modeldata.go`:
  `writeReplacing` is `fsx.WriteFileMode` after `appdirs.EnsurePrivateDir` (R6; the pause and rename seams are gone); the sweep
  reads the directory with `os.ReadDir` and matches `.models-dev.json.` and `.attempt.json.` prefixes with a `.tmp` suffix (R4,
  R5); `modelDataFailure` names `llm.ErrNotPermitted` `redirect_downgrade` (R1). `internal/fsx/write.go`: `osFS.Rename` goes
  through `SetRename`'s function when a test sets one (R6). `llm/provider/testmain_test.go`: `childOutputs` and
  `mergeChildOutputs` (R2), `TestLogProbe_0072` and `TestUnguardedChildKeepsTestLog_0072`.
* **Reds** before the fixes, on a scratch copy of the tree with the new tests and R2's fix taken out (`pass2_red.py`,
  `logs/pass2-red.log`): R1 `modeldata_test.go:278: &{…}, false, <nil>; want the download refused, not permitted` and
  `modeldata_test.go:496: result ok , want failed redirect_downgrade`; R2 `testmain_test.go:184: the test log holds 1 headers and
  the guarded child's input false`; R3 `modeldata_test.go:335: … model metadata: the reply is larger than 32 MiB; want gobble's
  own limit` and `:499`; R4 and R5 `modeldata_test.go:433: left [.attempt.json.stale.tmp … .models-dev.json.stale.tmp …]` (a
  directory named `cache [1]` swept nothing); R6 `modeldata_test.go:300: renamed [".models-dev.json.2522478237.tmp"
  ".attempt.json.1120543577.tmp"]; want the copy and attempt.json each through internal/fsx's atomic writer`. R7a and R7b are
  tests strengthened on unchanged code; their reds are their plants'. After the fixes every test passes
  (`logs/pass2-green.log`).
* **R2, the cache and the profile** (`testlog_check.py`, `logs/r2/`, `logs/r2-unfixed/`). With the fix: a run of
  `TestLogProbe_0072`, `TestProbePort` and `TestWireOffline` is not cached the first time, cached the second, and run again when
  `GOBBLE_TEST_LOGPROBE_A`, an input only the guarded child reads, changes; a `-coverprofile` run holds one `mode:` line, the
  guarded child's counts (`ModelData.Stamped`) and the unguarded child's (`adapter.Probe`). Without it: the third run was served
  from the cache (`cache input_changed: … cached True, want cached False`), the stale cache the review named. The unguarded
  child's own tests read no input the log records (the proxy variables are read by `TestMain`'s check before the log starts),
  so what the cache records is the guarded child's inputs, kept. On this toolchain the coverage profile was whole without the
  fix too (the go command merges each process's counters from `-test.gocoverdir`); the child is given its own profile path all
  the same, and the profile, appended, still holds one `mode:` line.
* **Rows to tests.** M82: `TestFetchModelDataRedirect_0072/https_to_http` (one request, the `https` one; `llm.ErrNotPermitted`),
  `TestModelDataRefreshFails_0072/redirect_downgrade` (failed, `redirect_downgrade`, the copy and the current document kept, one
  request). M79: `TestUnguardedChildKeepsTestLog_0072`. M13: the two `oversize` cases, the padding in a key the SDK does not keep,
  asserting "the reply is over 32 MiB". M6: `TestModelDataStaleTemps_0072` in `cache [1]`, with `attempt.json`'s temporary
  files. D2, M3, M5, M66, M69: `TestModelDataWriteKeepsOld_0072/house_writer` and the tests that fail or pause fsx's rename
  through `fsx.SetRename`. M65: `TestModelDataEngineNeverBlocks_0072`, its fetch `FetchModelData` on a transport that blocks
  mid-request, its `-p` turn reading the model facts, bounded at 20 s. M14 to M17: `TestModelDataGuards_0072` asserts the current
  document unchanged on each refusal and the download current on each acceptance.
* **Plants** (`plants.py`, 34 and the control, on a scratch copy of the final tree; `logs/plants-p2a.out`, `logs/plants-p2b.out`;
  M7 on the Windows laptop, `logs/plant-M7-windows-p2.log`): every one caught, the control not. New and re-pointed:

| Plant | Row | Caught by | Failing line |
| :--- | :--- | :--- | :--- |
| R1 the downgrade check removed | M82 | `…Redirect_0072/https_to_http`, `…RefreshFails_0072/redirect_downgrade` | `modeldata_test.go:278: &{…}, false, <nil>; want the download refused, not permitted` |
| R2 the child's log at the parent's path | M79 | `TestUnguardedChildKeepsTestLog_0072` | `testmain_test.go:183: open …/test.log: no such file or directory` |
| R3 gobble's 32 MiB limit removed | M13 | both `oversize` cases | `modeldata_test.go:335: … the reply is larger than 32 MiB; want gobble's own limit, "the reply is over 32 MiB"` |
| R4 the sweep by glob | M6 | `…StaleTemps_0072` | `modeldata_test.go:433: left [.attempt.json.stale.tmp … .models-dev.json.stale.tmp …]` |
| R5 `attempt.json`'s temporary files not swept | M6 | `…StaleTemps_0072` | `modeldata_test.go:433: left [.attempt.json.stale.tmp .attempt.json.young.tmp …]` |
| R6 written without internal/fsx's writer | D2 | `…WriteKeepsOld_0072/house_writer`, and the tests that fail or pause the rename | `modeldata_test.go:300: renamed []; want the copy and attempt.json each through internal/fsx's atomic writer` |
| R7a the holder locked for a fetch | M65 | `…EngineNeverBlocks_0072` | `modeldata_test.go:627: the prompt waited for the engine's fetch` |
| R7b a refused download made current | M14 to M17 | `…Guards_0072` (five refused cases) | `modeldata_test.go:551: a refused download changed the current document` |
| M3 the temporary file kept after a failed rename (now in `internal/fsx`) | M3 | `…WriteKeepsOld_0072/rename` | `modeldata_test.go:283: temporary files left: [….models-dev.json.gobble-dc60bbf6b666.tmp]` |
| M5 a fixed temporary name (now in `internal/fsx`) | M5 | `…ConcurrentWriters_0072`, M66's `two_writers` (one writer serves both) | `modeldata_test.go:377: open ….models-dev.json.gobble-fixed.tmp …` |
| M7 fsx's rename retries cut to none (Windows) | M7 | `…HeldWindows_0072/replace` | `modeldata_windows_test.go:31: the replace under a reader held for 200 ms: rename … The process cannot access the file because it is being used by another process.` |

  The other 23 caught as in pass 1, at their moved lines; M66's is caught by `house_writer` too, and M11's by the new
  `redirect_downgrade` case.
* **Docs.** `docs/architecture.md:113` and `:135` gain the downgrade refusal, gobble's limit, the house writer and the sweep —
  fixed; `llm/provider/modeldata.go`'s `FetchModelData` comment — fixed. Open as before: Phase 2's and Phase 3's lines.
* **Lint and a platform's path.** The first gate round found `errcheck` on the limit reader's one-byte probe
  (`llm/provider/modeldata.go:305:9: Error return value of b.r.Read is not checked`), on every host; fixed without a
  suppression (the probe's error is kept as a read error). The Windows laptop then refused the sweep test's directory
  `cache [1]*?` (`The filename, directory name, or volume label syntax is incorrect`): `*` and `?` are not legal there, so the
  directory is `cache [1]`, a glob's class, legal everywhere, which still defeats `filepath.Glob` (R4's plant is caught as
  before). Every gate then ran again on the final tree.
* **Correction (2026-11-25) of pass 1's count.** Pass 1's record and hand-back said 29 plants; pass 1 ran 26 (25 on this Mac
  and M7 on the Windows laptop) and the control, each caught as its table shows. Pass 2 adds eight (R1 to R7b, R2).
* **Follow-up, recorded, not built:** `writePrivate` (`internal/cli/choice.go:272`) syncs neither file nor directory.

#### Phase 1 gates, pass 2 (2026-11-25, the final tree)

Exit codes taken before any filter (`<scratch>/b0072/final_run2.py`); logs `<scratch>/b0072/logs/`; no log has a fallback line;
every heavy run through `run-heavy` or `run-on-host`.

| Command | Where | Exit |
| :--- | :--- | ---: |
| `make pre-add-check FILES=…` (the 15 changed `.go` files), `run-heavy` | this Mac | 2 (errcheck), then 0 |
| `go test -count=1 ./...`, `run-heavy` | this Mac | 0 |
| `make check-test-isolation`, `run-heavy` | this Mac | 0; 0 files written above the temp roots |
| `make preflight`, `run-on-host --need darwin` | this Mac | 2 (errcheck), then 0, "preflight passed" |
| `make preflight race`, `run-on-host --need linux --race` | the Linux machine (linux/amd64, not WSL) | 2 (errcheck), then 0, "preflight passed" |
| `make preflight`, `run-on-host --need windows` | the Windows laptop (windows/amd64) | 2 (errcheck), then 0, "preflight passed" |
| the changed packages' row tests (`internal/cli`, `llm/provider`), `run-on-host --need windows` | the Windows laptop | 1 (the sweep test's directory name), then 0; 27 top-level passes, M7's two cases among them |
| every plant (`plants.py`, `plants_windows.py`) | this Mac; M7 on the Windows laptop | 0; 33 caught and the control not; M7 caught |
| `make check-records`, `make markdownlint` | this Mac | 0; 0 issues in 59 files |
| `denyscan.py --root <gobble> --changed` | this Mac | 0, 0 hits in 19 files |
| `github-disclosure.py redact` (dry run) | this Mac | 0, would edit 0 files |

Exit of Phase 1 (pass 2): every finding closed, each test red without its fix or held by its plant; 34 plants caught and the
control not; gates recorded; docs rows returned. Two build passes of Phase 1.
