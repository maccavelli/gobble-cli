---
status: accepted
date: 2026-11-24
decision-makers: repository owner
consulted: "the owner's rulings of 2026-11-24, relayed by the coordinator: \"go with option 1 for models.dev, draft the record\"; Q1 to Q3's answers; the answers to the first draft's review (eleven cases); this repository at bf816c1 (first draft) and ba8f623 (second draft, 0071's Phase 1 committed), read only; go-llmprovider-sdk at its tag v1.32.0 (gobble's pin, go.mod:16) and at 9e86748 (its main), read only, and the draft of its 0093-MADR, read only; the read-only checkouts opencode at 4ce137b39e and kilocode at 58072bd696"
informed: go-llmprovider-sdk (its 0093-MADR takes the document from its caller from v1.33.0; this record is that caller)
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Keep a Refreshed Copy of the models.dev Document on Disk, Fetched by gobble and Read with the SDK's ParseMetadata, and Ship No Built-in Model Data

Associated PLAN: [0072-PLAN-keep-a-refreshed-models-dev-copy-on-disk.md](0072-PLAN-keep-a-refreshed-models-dev-copy-on-disk.md)

*(2026-11-24, accepted by the owner, for the MADR and PLAN together: "Accept; approve; queue after 0070 (Recommended)".)*

*(2026-11-24, re-ground at 61569d0, where 0071's Phase 1b is committed. Moved citations: none. The changes since ba8f623 are `acpserver/agent.go` (line 836, edited in place, no line moved), `acpserver/outputlimit.go` and `acpserver/prompt.go` (lines added after 75 and 273; neither file is cited here), `internal/cli/line.go` (line 401 on; not cited here), `go.mod` (line 20, `acpext v0.5.0` to `v0.6.0`, in place; the SDK pin cited at `go.mod:16` has not moved) and docs. Every line cited in `acpserver/agent.go` (:457-464, :701, :1102-1124, :1145-1152) is at its old line. No row is rewritten.)*

*(2026-11-24, first draft. `python3 scripts/check_records.py --next` printed `0072` at bf816c1 with 0070's and 0071's records
in the tree, so this record takes `0072`.)*

*(2026-11-24, second draft, after the owner answered Q1 to Q3 and the eleven cases of an independent review, relayed by the
coordinator. Changed: gobble fetches `https://models.dev/api.json` itself (D4, M8 to M10, M58, M59); the listing reads the
same document, by URL on `v1.32.0` and handed in on `v1.33.0` (M22, M62); the built-in copy and its generator are removed in
this record's Phase 3, not at the SDK pin (M47); a session keeps the facts it began with (M21, M60); only model calls and the
`models` commands start a refresh (M24); the attempt file is shared by every process (M26, M66); `--force` (M68); bounds on
the settings (M70); the review's other findings (M5, M10, M37, M49, the citations, M64). No row is open; the record stays
`proposed` until the owner accepts it.)*

*(2026-11-24, third draft, after the owner answered the second draft's four calls, relayed by the coordinator: calls 1 to 3
confirmed (M61, M68, M26); call 4 changed: gobble hands the document to `provider.New` too, so each session's provider is
built with the session's document and OpenCode routes from it (D17, D19, D22, D23, M62, M71 to M73). No row is open.)*

*(2026-11-25, amendment: the owner's "gobble: v1.33.0 pin first, then 0072 (Recommended)". gobble pins go-llmprovider-sdk
`v1.33.0` before this record is built (0071-PLAN's Phase 2a), so every phase here runs on `v1.33.0`: M22 is struck; D14, D17,
D21, M54, M55 and M62 are re-pointed; Phase 4's hand-in becomes Phase 2b, between Phases 2 and 3. Re-grounded at 073fd1f. See
"Amendment of 2026-11-25: built on v1.33.0" at the end. No row of this record is open.)*

*(2026-11-25, third addition at the end: re-grounded at 927c7dd by the PLAN's Phase 0; D2's sha256 is over the compact form;
M64 gains `TestBinaryCompletionMode`; five cases put to the owner by the PLAN's Phase 1 stop.)*

*(2026-11-25, fourth addition at the end: the answers to those five cases, M77 to M81.)*

*(2026-11-25, fifth addition at the end: Phase 1's review, and the owner's "Follow any, but refuse https→http
(Recommended)", M82.)*

## Context and Problem Statement

Every fact gobble knows of a model without asking its provider (its name, window, output cap, input kinds, reasoning and
efforts, prices) comes from one copy of the models.dev document embedded in the binary, fetched once, on 2026-10-14. Since
0071, that copy also sets each turn's output cap. A model released after that date is unknown to gobble until someone re-runs
a script and ships a release; a price or cap corrected upstream stays wrong in gobble the same way. The model list a provider's
listing ranks reads another document, OpenCode's copy, which the SDK downloads itself.

The owner ruled that gobble keeps its own disk copy, refreshed in the background, that models.dev is the one source for gobble
and the SDK, that the SDK stops fetching it (go-llmprovider-sdk's 0093-MADR, drafted in parallel), and that gobble ships no
built-in model data. This record decides every case of that store.

### Owner rulings

| Ruling, as given | Label | Rows |
| :--- | :--- | :--- |
| A gobble-side disk copy of the models.dev document, read with the SDK's `ParseMetadata`, refreshed in the background past a set age, its time and source recorded, `gobble models refresh`, its age shown | "go with option 1 for models.dev, draft the record" | M1 to M70 |
| Q1. The source | "make models.dev the source-of-truth for both the SDK and gobble." | M8, M22, M23, M55, M62 |
| Q1, the SDK's side (go-llmprovider-sdk 0093-MADR) | "scope the models.dev fetch/data/presentation out of the SDK. this is better suited to the agent harness CLI than the LLM provider. thoughts?"; then "Move it before v1.33.0" | M8, M54, M55, M56, M62 |
| Q2. The ages | "Refresh 24h, warn past 7 days (Recommended)" | M36 |
| Q3. Built-in data | "None; remove in 0072 (Recommended)", superseding "Keep until the v1.33.0 pin (Recommended)" | M19, M33, M47 to M49 |
| Q3, the owner's ask behind it | "why not integrate buntdb to hold persistent stored models.dev data, which would be read if the fetch failed, but would be refreshed upon the next successful fetch? or an alternative mechanism to persist so i do not have to hard-code anything." | M2 |
| Q3, the persistence | "JSON file, atomic write (Recommended)" | M2, M3, M5 |
| Case 1. What starts a refresh | "Only model calls and models cmds (Recommended)" | M24, M29 |
| Case 2. A refresh at exit | "acp/interactive cancel at once (Recommended)" | M28 |
| Case 3. Reads before any model call | "Load the copy first (Recommended)" | M20 |
| Case 4. `refreshHours: 0` | "Never fetch; warn once (Recommended)" | M30, M63 |
| Case 5. A refused download | "Add a --force flag" | M67, M68 |
| Case 6. A download that cannot be saved | "Use it this run, log, wait (Recommended)" | M69 |
| Case 7. The engine: "the refresh/models.dev fetch/load/save engine needs to invoke it's own goroutine so it never blocks the other processes and can work synchronously while the gobble harness is doing other mutli-process concurrent work. assess this." | "Goroutine + shared attempt time (Recommended)" | M26, M65, M66 |
| Case 8. Timeouts | "Background 30 s, refresh cmd 120 s (Recommended)" | M59 |
| Case 9. A refresh during a session | "Keep each session's facts from its start (Recommended)" | M21, M60, M61 |
| Case 10. Setting bounds | "Max 8760 h and 3650 days (Recommended)" | M70 |
| Case 11. Redirects | "Follow any, record the asked URL" | M58 |
| Call 1. A session that began with no data | "Pick up the first download (Recommended)" | M61, M71 |
| Call 2. `--force` and an empty document | "Still refuse an empty document (Recommended)" | M14, M68 |
| Call 3. What gates the hour | "Any attempt gates an hour (Recommended)" | M26, M67 |
| Call 4. The document and `provider.New` (the coordinator's answer to the question in it: not hard-coded; gobble passes the downloaded or disk document to `New` at run time) | "for both 0093 and here, i want to hand to provider.New, but does that mean it is hard coded?" | M62, M71 to M73 |
| The SDK's 0093 Q4 | "Requests read only New's document (Recommended)" | M73 |
| *(2026-11-25)* gobble pins `v1.33.0` first (0071's Phase 2), then builds this record on it; its `v1.32.0`-only steps are dropped | "gobble: v1.33.0 pin first, then 0072 (Recommended)" | M22, M54, M55, M62; D14, D17, D21 |

### Facts read at bf816c1, re-grounded at ba8f623, and at the SDK's v1.32.0

*(2026-11-24, re-ground at ba8f623, where 0071's Phase 1 is committed: `KnownModels` moved to `llm/provider/model.go:142-151`
and the lookup's `WithoutModelMetadata` to `:176`; `cmdContext` is `acpserver/commands.go:301-339`. Every other line below
was re-read there and has not moved.)*

* **F1. The embedded copy.** `llm/catalog/catalog.go:13-23` embeds `data/models.json` and `data/source.json`; `Snapshot()`
  returns a copy. `source.json` names `https://models.dev/api.json`, fetched `2026-10-14T02:22Z`, with its sha256 and 782
  models in nine sections (`llm/catalog/data/source.json:1-16`). `models.json` is 515,680 bytes; `LICENSE.models.dev` sits
  beside it; `NOTICE:22-28` credits it. `scripts/gen_catalog.py` makes both (`:31` the URL, `:36-40` the fetch, `:59-78` the
  write); its docstring says nothing in gobble fetches the document at run time (`:6-8`).
* **F2. Who reads it.** `llm/provider/model.go:18-26` parses it once (`sync.OnceValues`, `catalog.ParseMetadata`);
  `modelInfoFrom` hands it to the SDK as a layer (`:107-110`), models.json above it (`:115`), with the SDK's own fetch off
  (`:176`); `KnownModels` lists its tool models (`:142-151`); `SnapshotErr` (`:47-53`) is logged at
  `internal/cli/agent.go:87`. Per-turn facts, 0071's cap included, come through `a.modelOf` (`acpserver/config.go:48`), which
  is `modelFacts` (`internal/cli/agent.go:80`, `:117`). Before any model call they are also read by ACP `initialize`'s image
  check (`acpserver/agent.go:457-464`, `:1145-1152`), `--model` resolution (`internal/cli/modelflags.go:28`, `:45-51`), the
  `defaultModel` check (`:74`), the provider picker (`internal/cli/choice.go:104`), shell completion
  (`internal/cli/completion.go:129`) and `gobble auth status --check`'s model (`internal/cli/auth.go:764`). A model id the
  facts do not know resolves with "Model %q not found for provider %q. Using custom model id." (`llm/catalog/resolve.go:200`).
  Each model call goes through the provider `modelProvider` builds (`internal/cli/agent.go:35`).
* **F3. The listing reads another document.** A listing passes `WithModelMetadataURL` only when `Options.ModelMetadataURL` is
  set (`llm/provider/list.go:129-130`, `:159-160`), which only tests set (`llm/provider/provider.go:53-55`), so a listing
  ranks with the SDK's default, `https://models.opencode.ai/api.json` (go-llmprovider-sdk `llmprovider/catalog/model_metadata.go:28`
  at `v1.32.0`), cached in the process for 10 minutes (`:31`). `provider.New` turns the fetch off (`provider.go:212-216`).
* **F4. The SDK leaves the fetch in v1.33.0.** go-llmprovider-sdk's 0093-MADR (draft, `proposed`, on the owner's rulings
  above) removes `LookupMetadata`, `LookupMetadataWith`, `CachedMetadataWith`, `WithModelMetadataURL` and its embedded
  `limits_snapshot.json` (86,356 bytes, dated 2026-10-12, on its main only: `limits_reads.go:331-336`), and adds a way for a
  caller to hand in a parsed `catalog.Metadata` (its D1; this record cites that record for the name). `ParseMetadata`,
  `Metadata` and `Encode` stay. At `v1.32.0` none of that has happened: `WithModelMetadataURL` exists and the extract does not.
  The owner answered 0093's Q4 "Requests read only New's document (Recommended)": a provider's requests (OpenCode's route,
  reasoning efforts, the replay field) read exactly the document its `New` was given.
* **F4a. A session's provider is built once and kept.** `sessionProvider` returns `s.provider` when set, else builds it under
  `a.mu` and keeps it (`acpserver/agent.go:1102-1124`); a prompt takes it once at its start (`:701`), as compaction does
  (`acpserver/compact.go:27`, `:81`); `buildProvider` builds a switched one (`acpserver/config.go:320-336`).
* **F5. `Encode` and `ParseMetadata` at `v1.32.0`.** `ParseMetadata` keeps the nine sections the SDK reads, decodes each now and
  refuses one that does not decode, a body over 32 MiB, or a non-document (`catalog/metadata_document.go:18-32`). `Encode`
  writes each section's bytes as read (`:34-67`), only the sections that decode (`:41-47`).
* **F6. gobble's directories and files.** `internal/appdirs/dirs.go:42-56` resolves the cache role: `GOBBLE_CACHE_DIR`, else
  `GOBBLE_HOME`, else `$XDG_CACHE_HOME/gobble` (default `~/.cache/gobble`) on Unix and macOS (`:149-154`), and
  `%LOCALAPPDATA%\gobble\cache` on Windows (`:121-127`). gobble keeps each provider's last listing at
  `<cache>/catalog/<id>.json` (`internal/cli/choice.go:217-224`), written through a temporary file and a rename (`:270-288`); a
  Chat Completions server's id is the user's models.json key, so a file in `catalog/` could share its name. On Windows a read
  meeting a replace retries its open for up to 2 s (`internal/fsx/regular.go:41`, `:76-87`) and a replace meeting a reader
  retries for up to 2 s (`internal/fsx/write.go:191`, `:198-217`) (0067-MADR D1, D4).
* **F7. Only `llm/provider` imports the SDK** (`internal/archtest/check_test.go:28-31`, `:51-54`, `:66-69`).
* **F8. Tests and the network.** `internal/cli`'s `TestMain` replaces the lister and the model facts with offline ones
  (`internal/cli/testmain_test.go:134-135`), checks them by identity (`:64-76`), and runs behind a proxy that fails the binary
  when a test reaches a host (`:108-148`; 0046-MADR D4); background work on a `runEnv` has an owner (0046-MADR D2). Not every
  `internal/cli` test sets `GOBBLE_HOME`. `llm/provider` has no `TestMain` and no guard. Two binary tests run gobble itself:
  `acpserver/binary_test.go:87-93` (`gobble acp` with a temporary `GOBBLE_HOME`, no proxy set, which must exit 0 within 2 s
  of its input closing and write nothing to stderr, `:123-129`) and `cmd/gobble/conpty_windows_test.go:1091-1096` (proxies set
  to a closed port).
* **F9. 0008-MADR D5's start-up rule:** "No network call before the first prompt" (`docs/decisions/0008-MADR-native-cli-mode.md:419`);
  `initialize` "makes no network call" (`acpserver/agent.go:457-458`).
* **F10. Sizes and timing.** models.dev's document: 5,377,832 bytes on 2026-10-10 and 5,621,182 on 2026-10-14
  (`docs/decisions/0005-PLAN-v1-feature-scope.md:3246`, `:3638`). The nine sections are 515,680 bytes (F1). This Mac's
  installed gobble build is 23,152,898 bytes, so the embedded copy is 2.2 % of it.
* **F11. Other harnesses.** opencode keeps the body with its digest and ETag, revalidates with `If-None-Match`, boots on a
  bundled snapshot and refreshes every 5 minutes (`<opencode>/packages/core/src/models-dev.ts:494-497`, `:551`, `:571-584`,
  `:604-606`, `:660-692`). kilocode reads `https://models.dev/api.json` into `<cache>/models.json`, writes a temporary file
  and renames it, locks across processes, refreshes every 60 minutes, and can pin a file or turn the fetch off
  (`<kilocode>/packages/core/src/models-dev.ts:171-176`, `:195`, `:207-241`, `:268-271`).
* **F12. Displays, commands, settings.** `cmdContext` (`acpserver/commands.go:301-339`) gains 0071's output-limit line.
  `gobble models` has `list` and `info` (`internal/cli/modelscmd.go:16-21`); exit codes are `internal/cli/exit.go:9-17`, an
  interrupt 128 plus the signal (`:21-26`). Keys are `internal/config/keys.go:78-143`; a value over a key's `Max` is reported
  "must be at most N" (`internal/config/merge.go:159`). 0005-MADR deferred "Catalog refresh: `gobble models refresh` into the
  cache directory" (`docs/decisions/0005-MADR-v1-feature-scope.md:178`).

## Decision Drivers

* The owner's rulings.
* No start-up, prompt or tool call waits for the network; no network before the first prompt; no test reaches it.
* A bad, partial or shrunken download never replaces a good copy; nothing hard-coded.
* One source, models.dev, for gobble and the SDK; a session's facts do not change under it.
* Every case decided up front, each with a test and a plant.

## Considered Options

* A. gobble fetches models.dev itself, keeps it in a JSON file, and hands it to the SDK (the rulings)
* B. As A, through the SDK's `LookupMetadataWith` (the first draft)
* C. An embedded key-value store (the owner's buntdb question)
* D. Keep the embedded copy, refreshed at release

## Decision Outcome

Chosen option: "A", because it is the owner's rulings: the SDK leaves the fetch (0093), so gobble owns it; a JSON file written
whole by a rename is the persistence the owner chose. B relies on names v1.33.0 removes. C adds a dependency and a file format
for one document a rename already writes atomically. D hard-codes data.

### The decisions

* **D1. Where.** `<cache>/model-data/models-dev.json` and, beside it, `attempt.json` (D18), `<cache>` the cache role of
  `appdirs.Resolve` (F6). Not `os.UserCacheDir`. A directory of its own, so no models.json server id names the same file.
* **D2. The copy's file.** One compact JSON object: `version` (1), `source` (the URL asked for, D4), `fetched` (UTC, RFC 3339
  to the second), `etag` and `lastModified` (when the reply carried them), `sha256` (hex, of `document`'s bytes), `sections`
  (models per section), and `document`, `Encode`'s bytes, verbatim. Written to `.models-dev.json.*.tmp` in the same directory
  (`os.CreateTemp`, owner-only; the directory by `appdirs.EnsurePrivateDir`) and moved with `fsx.ReplaceFile`; a failure
  removes the temporary file and leaves the old one as it was. No lock: each process renames a whole file; the last wins.
  After an accepted write, temporary files of that pattern older than an hour are removed. Reads through
  `fsx.ReadRegularFile`, limited to 32 MiB. *(2026-11-25, the review of Phase 1: written by `internal/fsx`'s atomic writer,
  which syncs the file and its directory; the sweep covers `attempt.json`'s temporary files too. See the fifth addition.)*
* **D3. A file that is not a copy** (not JSON, truncated, another `version`, a wrong `sha256`, a refused `document`, over 32
  MiB, not a regular file) is no copy: one log warning with the path and the reason; due at once; replaced by the next accepted
  download, never removed before.
* **D4. The fetch is gobble's.** `provider.FetchModelData(ctx, url, o)`: one GET of `https://models.dev/api.json` with the
  run's client (`Options.HTTPClient`, else the SDK's `llmprovider.NewHTTPClient`, which reads the proxy variables), gobble's
  user agent, `If-None-Match` and `If-Modified-Since` from a copy of the same source when it holds them; a body limited to
  32 MiB; then `ParseMetadata` and `Encode`, so only the nine sections are kept and each decodes. A 304 renews the copy's
  `fetched` and writes nothing else. Redirects are followed as Go's client follows them (up to ten, to any host), and the copy
  is stamped with the URL asked for. *(2026-11-25, "Follow any, but refuse https→http (Recommended)": a redirect from `https`
  to `http` fails the download, M82.)* The fetch's own deadline: 30 s in the background, 120 s for `gobble models refresh`. It
  lives in `llm/provider` (F7) and keeps no cache: two calls are two requests.
* **D5. A failed fetch** (offline, a status other than 200 or 304, the deadline, a body past the limit or not a document)
  changes nothing on disk or in memory; it is recorded in `attempt.json`.
* **D6. The guards.** A downloaded document is refused, and the copy kept, when it holds none of the nine sections with a
  model (always); or, against a copy from the same source, a section the copy has is missing (a section that does not decode is
  missing: F5) or has fewer than half the copy's models. With no copy or a copy from another source, any document past the
  first test is accepted. An identical document is saved with the new time. `--force` (D9) skips the comparisons, not the first
  test.
* **D7. Load.** `runEnv` reads the copy once, at its start, before any read of model facts (F2), whatever the command; loading
  never fetches. The process's current document is held by `llm/provider` behind an atomic pointer (in place of `snapshot`'s
  `sync.OnceValues`) as an immutable `provider.ModelData`; with none, models are unknown (`SnapshotErr`'s path today).
* **D8. When a refresh starts.** Only before a model call (the provider `modelProvider` builds, F2) and in `gobble models list`,
  `info` and `refresh`, and only when the copy is due: none, unreadable, from another source, fetched after now by this clock,
  or as old as `modelData.refreshHours`. ACP `initialize`, `/context`, the picker, completion, `--model`, `defaultModel` and
  `auth status --check` load and never fetch, so F9 stays true. The refresh never makes the call wait. One per process at a
  time. Before fetching, the file is read again: a copy another process wrote, not due, becomes current and nothing is
  fetched; and `attempt.json` is read: an attempt by any process that started within the last hour skips the automatic
  refresh. `refreshHours: 0` never fetches on its own.
* **D9. `gobble models refresh [--force]`.** Fetches now, whatever the age, the setting or the hour's wait; obeys D6 unless
  `--force`; saves; prints one line on stdout. Exit codes: 0 saved or not modified; 1 not fetched, refused or not saved; 2 a
  usage error; an interrupt's code (F12), nothing written.
* **D10. Tests.** `internal/cli`'s `TestMain` sets `fetchModelData` to one that fails offline and `loadModelData` to one that
  reads no file; `TestNoBrowserOrSignIn` checks both by identity. `llm/provider` gains a `TestMain` with 0046 D4's network
  guard, and its tests use in-memory transports and URLs under `.invalid`. The two binary tests (F8) set every proxy variable
  to a closed port and check that no copy was written under their `GOBBLE_HOME`. `make check-test-isolation` runs.
  *(2026-11-25, the coordinator's call on Q-C: `llm/provider`'s `TestMain` runs `TestWireOffline` and `TestProbePort` in a
  second child without the proxy variables (M79).)*
* **D11. Settings.** `modelData.refreshHours` (default 24, at most 8,760) and `modelData.warnDays` (default 7, at most
  3,650): integers, at least 0, global only, read at process start; 0 turns each off; a larger value is "must be at most N"
  (F12), and the default is used.
* **D12. Showing the age.** `/context` and `/session` print one line after 0071's output-limit line, for the session's data
  (D19): "Model data: models.dev, fetched YYYY-MM-DD HH:MM UTC (AGE ago)"; past `warnDays` followed by "; gobble models
  refresh fetches it again"; "Model data: none; models are unknown until gobble models refresh downloads them"; or "Model data:
  models.dev, fetched … (ahead of this clock)". `gobble models info` prints the same words as a last `data` line. A copy past
  `warnDays` warns once per run on stderr (interactive start, headless, `models list` and `models info`): "model data was
  fetched N days ago from models.dev; gobble models refresh fetches it again", adding "; the last download was refused:
  REASON; gobble models refresh --force accepts it" or "; the last download failed: KIND" from `attempt.json`. `gobble acp`
  logs it at warn once per process and writes nothing to stderr. `gobble models list`'s stdout is unchanged.
* **D13. No data when none is in use** (a never-online first run, a pinned run with no copy): no window (no automatic
  compaction until the service refuses), 0071's assumed 16,384 cap, no prices, thinking levels off to `xhigh`, curated ids
  only, "-" for every fact in `gobble models list`. One warning per run on stderr (`gobble acp`: the log), when the run is
  pinned or its or the last attempt failed: "no model data: models are unknown; gobble models refresh downloads them", with the
  failure's kind. None while a first download has not yet been tried.
* **D14. Removal.** In Phase 3, on `v1.32.0`: `llm/catalog/data/models.json`, `source.json`, `catalog.Snapshot`,
  `catalog.Source`, `scripts/gen_catalog.py` and their tests go; `SnapshotErr` becomes `ModelDataErr`. The 2026-10-14
  document becomes test data (`llm/provider/testdata/models-dev-2026-10-14.json`), installed by each test binary that reads
  model facts, so no expected value changes. The build loses 515,680 bytes. *(2026-11-25: on `v1.33.0`, which gobble then
  pins; the built-in copy that 0071's pin hands the listings, or none (0071 Q5), is what Phase 3 removes.)*
* **D15. The licence.** After Phase 3 the binary carries no models.dev data (and from `v1.33.0` neither does the SDK, F4).
  `NOTICE`'s paragraph then says gobble downloads model data from models.dev at run time, and that its test data holds a copy of
  models.dev's document under the MIT licence, whose text moves to `llm/provider/testdata/LICENSE.models.dev`, beside it.
* **D16. 0071, models.json, the records.** Each call's cap and window read the session's document (D19); with none, 0071 L5's
  assumed 16,384 and L4's unknown window. models.json stays the layer above the document. Dated notes: 0005-MADR's "Catalog
  refresh" row; 0066-MADR D3 (superseded by Q1: the copy is handed to the SDK from `v1.33.0`, D17); 0071-MADR F4. 0070 shares
  no behaviour; both edit `internal/config/keys.go`, the schema golden and `llm/provider`, so they are built one after the
  other.
* **D17. The SDK.** On `v1.32.0` (Phases 1 to 3), production sets `Options.ModelMetadataURL` to `https://models.dev/api.json`
  for every listing, so the SDK's listing reads models.dev too (a live read of the same source, in the SDK's 10-minute cache).
  In Phase 4, which is the same change as gobble's pin of `v1.33.0`, because that tag removes `WithModelMetadataURL`: gobble
  hands its current document to every listing, and its session's document to every lookup and to every `provider.New` it calls
  for that session (the turn's, compaction's, a switched provider's, the title's), through the hand-in 0093-MADR adds (cited
  there, its D1), in place of the URL, the layer at `model.go:107-110` and `New`'s `WithoutModelMetadata` (`provider.go:215`).
  Nothing is hard-coded: the document handed is the one downloaded or read from disk at run time (call 4). Under 0093's Q4
  answer a provider's requests read exactly that document, so OpenCode routes from it first. With no document, the SDK ranks,
  looks up and routes without one (0093's "none is off"; OpenCode by its table). New sessions build from the current copy.
  0071's Phase 2 follows Phase 4, so `CachedLimits` reads the handed document.
  *(2026-11-25, re-pointed by "gobble: v1.33.0 pin first, then 0072 (Recommended)": the first sentence is struck, as M22 is:
  no `v1.32.0` phase is built. The removal of `WithModelMetadataURL`, `Options.ModelMetadataURL` and `New`'s
  `WithoutModelMetadata` is 0071's pin (its P2), with no document handed until this record exists, or the built-in copy to
  the listings (0071 Q5). The rest of this decision stands and is built as Phase 2b (was Phase 4), on `v1.33.0`, no longer
  the same change as the pin. 0071's Phase 2 now comes before this record: in Phase 2b `CachedLimits` reads the session's
  document, handed in (M54).)*
* **D22. Rebuilding a session's provider** (Phase 4). Only a session that began with no data is rebuilt, once, when the first
  download lands (call 1): under `a.mu`, `s.provider` is cleared and the session takes the new document; the next
  `sessionProvider` (its next prompt, or a compaction) builds with it. A call under way keeps the provider its prompt took
  (F4a) and finishes on it; nothing is cancelled or resent. A session that holds a document is never rebuilt by a refresh.
* **D23. The cost of one refresh.** One GET (the document gzipped, about half a megabyte, or a 304), one `ParseMetadata` of the
  nine sections (the parse start-up does today), one copy write and one `attempt.json` write. No provider is rebuilt but those
  of sessions that began with none, once each. A document stays in memory while a session holds it and is released when the
  last such session ends, so a process holds at most one document per distinct copy its open sessions began with, plus the
  current one.
* **D18. The engine and `attempt.json`.** The load, fetch, guard and save run in one goroutine per process, started by D8's
  trigger and owned by the run (0046 D2); no prompt, tool call or other process waits on it; readers take the current document
  from the atomic pointer. `attempt.json` is `{version: 1, source, started, finished, result, reason}`, `result` one of
  `started`, `ok`, `not_modified`, `failed`, `refused`, `not_saved`; written at the start and at the end, each time to a
  temporary file and `fsx.ReplaceFile`; the last writer wins. An unreadable `attempt.json` is no attempt: logged once, a
  refresh allowed, replaced by the next write. Its `started` time gates every process's automatic refresh for an hour,
  whatever the result; an attempt left `started` by a process that died gates no longer than that hour.
* **D19. A session's facts.** Each ACP session takes the process's current `provider.ModelData` when it is created, loaded or
  forked, and keeps it until it ends; its calls, `/context`, compaction and (from Phase 4) its providers read it. A session
  that began with none takes the first download when it lands and keeps that one from then on (call 1; D22). The
  snapshot is the session's in memory only: a resumed session takes the copy current at its resume, and a model that copy lacks
  is unknown, with the usual custom-id warning. A refresh's new document is current for sessions that start after it.
* **D20. Exits.** `gobble acp` and the interactive CLI cancel an automatic refresh at their exit at once and do not wait for a
  write under way (at worst a temporary file is left, removed by D2's sweep). A headless run and the `models` commands wait for
  it up to 5 s, the write and `fsx`'s Windows retries (2 s each, F6) included, then cancel.
* **D21. Exported names** (gobble is below `v1.0.0`): `llm/provider`: `FetchModelData`, `ModelData` (with `Stamp()` and
  `Sections()`), `ParseModelData`, `CurrentModelData`, `SetCurrentModelData`, `Options.ModelData` (the session's; nil is the
  current), `ModelDataErr` (for `SnapshotErr`, Phase 3); `acpserver`: `Options.ModelData func() *provider.ModelData` and
  `Options.ModelDataLine func(*provider.ModelData) string`; `internal/config`: the two keys' `Settings` fields; removed in
  Phase 3: `catalog.Snapshot`, `catalog.Source`. *(2026-11-25: `ModelData`, `ParseModelData` and `Options.ModelData` are
  built by 0071's pin (its P3), with nil handing no document; "nil is the current" is re-pointed: nil hands none, and
  `internal/cli` and `acpserver` pass the current or the session's document explicitly. This record adds the others.)*
  *(2026-11-25, "Add the four names (Recommended)" and "Hooks take the session's document (Recommended)": D21 also lists
  `ModelDataStamp`, `(*ModelData).Stamped`, `(*ModelData).Document()` and `ModelDataURL` (M77), and the session's
  `*provider.ModelData` as the first argument of `acpserver.Options.ModelInfo` and `Options.CachedLimits` (M80). See the
  fourth addition.)*

### Consequences

* Good, because new models, prices and caps reach gobble within a day of use, with no release and nothing hard-coded.
* Good, because gobble and the SDK read one source, and from `v1.33.0` one document.
* Good, because a bad, empty, shrunken or torn download never replaces a good copy, and `--force` is the deliberate way past
  a guard.
* Good, because a session's facts do not change under it.
* Neutral, because the copy is about 0.5 MB on disk, the same parse as today's.
* Bad, because a never-online first run knows no model (D13).
* Bad, because a resumed session may see facts other than those it began with (D19: the snapshot is not persisted).
* Bad, because until `v1.33.0` the listing reads models.dev live while the facts read the copy: one source, two times.
  *(2026-11-25: lapses; no phase runs on `v1.32.0`.)*
* Bad, because a headless run whose copy is due may exit up to 5 s later (D20).

### Confirmation

* The PLAN's tests, one per decided row, red at the base where the row changes behaviour, each held by a plant on a scratch
  clone.
* `make preflight` on this Mac, the Linux machine and the Windows laptop through `run-on-host`; `make race` on the Linux
  machine; `make check-test-isolation`; an independent review of each build phase finds no blocker.

### State table

Each decided row's plant is in the PLAN, one to one. "Copy" is the disk copy; "current" is D7's; "the session's" is D19's.

| Row | Input or state | Status | Outcome | Test | Decided by |
| :--- | :--- | :--- | :--- | :--- | :--- |
| M1 | Where the files live, per platform and variable | decided | `<cache>/model-data/`, `<cache>` from `appdirs.Resolve` | `TestModelDataPath_0072` | D1 |
| M2 | The copy's shape | decided | D2's object, `etag` and `lastModified` when sent; written then read, `document` byte-identical, its sha256 matching | `TestModelDataRoundTrip_0072` | D2; Q3 |
| M3 | A write that fails (temporary file, rename) | decided | the old file byte-identical; no temporary file left | `TestModelDataWriteKeepsOld_0072` | D2 |
| M4 | A copy not JSON, truncated, of another `version`, with a wrong sha256, a refused `document`, over 32 MiB, or a directory | decided | no copy; one log warning; due at once; kept until replaced | `TestModelDataUnreadable_0072` (one subtest each) | D3 |
| M5 | Two writers at once | decided | a writer paused between its temporary file and its rename while the other completes: a reader reads the other's whole file, then, after the rename, the first's; never a mix | `TestModelDataConcurrentWriters_0072` (a write seam that pauses mid-write) | D2 |
| M6 | Temporary files a killed process left | decided | removed after an accepted write when older than an hour; younger ones kept | `TestModelDataStaleTemps_0072` | D2 |
| M7 | Windows: another process holds the file | decided | reads and the replace retry the sharing violation (0067) | `TestModelDataHeldWindows_0072` (Windows only) | D2 |
| M8 | Who fetches, and from where | decided | `FetchModelData`: gobble's GET of `https://models.dev/api.json`, `ParseMetadata`, `Encode`; two calls, two requests | `TestFetchModelData_0072` | D4; Q1 |
| M9 | A copy of the same source holding an ETag or a date | decided | sent as `If-None-Match`, `If-Modified-Since`; a 304 renews `fetched` only; a 304 to a request that sent neither is a failure | `TestFetchModelDataConditional_0072` | D4 |
| M10 | Proxies | decided | the run's client carries the request; test URLs under `.invalid`; `llm/provider`'s new network guard fails any escape | `TestFetchModelDataUsesClient_0072`; `llm/provider`'s `TestMain` | D4, D10 |
| M11 | Offline | decided | a failure; nothing changes; `attempt.json` `failed`, `offline` | `TestModelDataRefreshFails_0072/offline` | D5 |
| M12 | A status other than 200 or 304 | decided | as M11, `status` | `…/status` | D5 |
| M13 | A body over 32 MiB, or not a document | decided | as M11 | `…/oversize`, `…/not_document` | D5 |
| M14 | A document with none of the nine sections | decided | refused, `--force` or not | `TestModelDataGuards_0072/empty` | D6 |
| M15 | Same source: a section the copy has is missing, or does not decode | decided | refused; the copy kept *(2026-11-25, "As a failed download (Recommended)": "does not decode" is a section the SDK reads as absent; one whose bytes fail the SDK's decode is M13's failure, M78)* | `…/section_missing`, `…/section_malformed` | D6; F5 |
| M16 | Same source: a section with fewer models | decided | under half the copy's: refused; half or more: accepted | `…/shrink_below_half`, `…/shrink_half` | D6 |
| M17 | No copy, or a copy from another source | decided | any document past M14 accepted | `…/first`, `…/other_source` | D6 |
| M18 | A document identical to the copy | decided | saved with the new time | `TestModelsRefreshCmd_0072/unchanged` | D6 |
| M19 | Which document is current | decided | the copy, else none; never two; no built-in one after Phase 3 | `TestModelDataCurrent_0072` | D7; Q3 |
| M20 | Reads before any model call (`initialize`, `--model`, `defaultModel`, picker, completion, `auth status --check`) | decided | the copy loaded first, once; nothing fetched; `--model` naming a model only the copy knows resolves without the custom-id warning | `TestModelDataLoadedFirst_0072` | D7; case 3 |
| M21 | A refresh lands while sessions are open | decided | current for sessions that start after it; open sessions keep theirs | `TestModelDataSwapForNewSessions_0072` (`-race`) | D19; case 9 |
| M22 | The listing on `v1.32.0` | ~~decided~~ struck *(2026-11-25)* | ~~production sets `ModelMetadataURL` to models.dev's URL for every listing~~ *(2026-11-25, "gobble: v1.33.0 pin first, then 0072 (Recommended)": no phase runs on `v1.32.0`. Before this record the listing gets the built-in copy or none (0071 P6, Q5); from Phase 1 the current copy by `Options.ModelData`, M19's test with a listing case)* | ~~`TestListingUsesModelsDev_0072`~~ | D17; Q1 |
| M23 | The source URL | decided | `https://models.dev/api.json`, a constant; no setting | `TestFetchModelData_0072` asserts it | D4; Q1 |
| M24 | What starts a refresh | decided | a due copy before a model call, or in `models list`, `info`, `refresh`; nothing else | `TestModelDataRefreshTrigger_0072/model_call`, `…/models_cmds`, `…/initialize_no_fetch`, `…/context_no_fetch` | D8; case 1 |
| M25 | Several model calls while one refresh runs | decided | one refresh per process | `…/one_in_flight` | D8 |
| M26 | An attempt by any process started within the hour | decided | no automatic refresh in any process; the command ignores it | `…/attempt_within_hour`, `…/other_process_attempt` | D8, D18; case 7 |
| M27 | Another process wrote a copy not due | decided | read first, made current, nothing fetched | `…/recheck` | D8 |
| M28 | The run ends with a refresh under way | decided | `acp` and interactive cancel at once and do not wait; headless and the `models` commands wait up to 5 s with the write, then cancel; `gobble acp` still exits 0 within 2 s, no stderr | `TestModelDataRunEnd_0072` (`acp`, `interactive`, `headless`, `models`); `TestGobbleACPBinary` | D20; case 2 |
| M29 | Shell completion | decided | loads, never fetches | `TestCompletionStartsNoRefresh_0072` | D8; case 1 |
| M30 | `refreshHours: 0` | decided | never fetches on its own; the command still fetches | `TestModelDataRefreshTrigger_0072/pinned` | D8; case 4 |
| M31 | The copy's time is after now | decided | due; "ahead of this clock"; no age warning | `…/clock_ahead`; `TestModelDataWarning_0072/clock_ahead` | D8, D12 |
| M32 | First run, online | decided | the first model call starts a download; that session reads it once it lands (M61) | `TestModelDataRefreshTrigger_0072/first_run` | D8, D19 |
| M33 | First run, never online | decided | D13: no window, assumed 16,384, no prices, levels off to xhigh, "-" in `models list`, `/context`'s "none" line, D13's warning after the failed attempt | `TestNeverOnlineFirstRun_0072` | D13; Q3 |
| M34 | Tests, the network and the host's cache | decided | `TestMain`'s seams fail offline and read no file, checked by identity; the guard fails any escape | `TestNoBrowserOrSignIn` (gains `fetchModelData`, `loadModelData`); `make check-test-isolation` | D10 |
| M35 | The settings' keys | decided | `modelData.refreshHours`, `modelData.warnDays`: integers, at least 0, global only, at process start; template, schema, golden | `TestSchemaCommitted` and the package's key tests | D11 |
| M36 | Their defaults | decided | 24 and 7 | the template and golden | D11; Q2 |
| M37 | `/context` and `/session` | decided | D12's line for the session's data, after 0071's, in each form; `internal/cli` builds each form | `TestContextShowsModelData_0072` (`acpserver`); `TestModelDataLine_0072` (`internal/cli`: `fresh`, `past_warn`, `none`, `clock_ahead`) | D12 |
| M38 | `gobble models info` | decided | a last `data` line, D12's words | `TestModelsInfoData_0072` | D12 |
| M39 | The copy is older than `warnDays` | decided | one stderr warning per run, with the last attempt's refusal and `--force`, or its failure kind; `gobble acp` logs it once | `TestModelDataWarning_0072/past`, `…/refused`, `…/failed`, `…/acp_log_once` | D12; case 5 |
| M40 | `warnDays: 0`; `gobble models list`'s stdout | decided | no age warning; stdout unchanged | `…/off`, `…/list_stdout` | D12 |
| M41 | `gobble models refresh` saves | decided | exit 0; "Model data: fetched from models.dev at TIME UTC: N models in K sections; it replaces the copy fetched TIME." (or "; unchanged since the copy fetched TIME.", "; not modified since TIME.", "; there was no copy.") | `TestModelsRefreshCmd_0072/saved`, `…/unchanged`, `…/not_modified`, `…/first` | D9 |
| M42 | It cannot fetch | decided | exit 1; "Error: model data: could not fetch URL: REASON; the copy fetched TIME is kept." (or "there is no copy.") | `…/offline`, `…/status`, `…/timeout_120s` | D9 |
| M43 | A guard refuses | decided | exit 1; "Error: model data: the document from URL was not saved: REASON; the copy fetched TIME is kept. gobble models refresh --force saves it." | `…/refused` | D9 |
| M44 | It cannot save | decided | exit 1; "Error: model data: fetched from URL but not saved: ERROR." | `…/not_saved` | D9 |
| M45 | An extra argument | decided | exit 2, the usage error | `…/usage` | D9 |
| M46 | Interrupted | decided | nothing written; the interrupt's exit code (130 for SIGINT) | `…/interrupted` | D9 |
| M47 | The built-in copy, `Snapshot`, `Source`, `gen_catalog.py` | decided | removed in Phase 3, on `v1.32.0`; the build 515,680 bytes smaller | `TestNoEmbeddedModelData_0072` (`llm/catalog` embeds no file) | D14; Q3 |
| M48 | Existing tests that read model facts, after removal | decided | read the 2026-10-14 document from test data; no expected value changes | `TestModelDataTestdataInstalled_0072` in each such test binary | D14 |
| M49 | `NOTICE` | decided | every file it names exists (a guard, green at the base); it names the test data's licence after Phase 3 | `TestNoticeNamesExistingFiles_0072` | D15 |
| M50 | 0071's cap and window | decided | read from the session's document; with none, 0071 L5 and L4 | `TestTurnCapFromModelData_0072` | D16 |
| M51 | models.json | decided | the layer above the document | `TestModelsJSONAboveModelData_0072` | D16 |
| M52 | 0005, 0066 and 0071's records | decided | D16's dated notes | `make check-records` | D16 |
| M53 | 0070 | decided | no shared behaviour; built one after the other | the PLAN's order | D16 |
| M54 | 0071's Phase 2 on `v1.33.0` | decided | after Phase 4: `CachedLimits` reads the handed document; with none, 0071's assumed cap *(2026-11-25: 0071's Phase 2 now comes first; in Phase 2b (was 4) `CachedLimits` reads the session's document handed in)* | 0071-PLAN's V1 test, with the document handed in *(2026-11-25: `TestTurnCapFromLimits_0071` gains a `session_document` case in Phase 2b)* | D17 |
| M55 | The document and the SDK's limits on `v1.33.0` | decided | handed in by 0093's hand-in, not as a layer (supersedes 0066 D3's "not handed") *(2026-11-25: the hand-in is `catalog.WithMetadata`, the SDK's `llmprovider/catalog/model_metadata.go:242-260` at `v1.33.0`; it replaces the built-in copy's layer that 0071's pin keeps (0066 S42); built in Phase 2b)* | `TestLookupGetsSessionModelData_0072` | D17; Q1 |
| M56 | An SDK-side persisted store | out of scope | ruled out by Q1's SDK side: the SDK holds no document of its own | | Q1 |
| M57 | A source or mirror setting, or a pinned file path | out of scope | a follow-up; `refreshHours: 0` is the pinned mode | | |
| M58 | A redirect, to another host | decided | followed (Go's client, up to ten); the copy stamped with the URL asked for *(2026-11-25, "Follow any, but refuse https→http (Recommended)": but a redirect from `https` to `http` fails the download, M82)* | `TestFetchModelDataRedirect_0072` (302 to another `.invalid` host) | D4; case 11 |
| M59 | The fetch's deadline | decided | 30 s in the background, 120 s for the command | `TestFetchModelDataDeadline_0072` (`synctest`) | D4; case 8 |
| M60 | A session's facts during its life | decided | taken at create, load or fork; kept until it ends; in memory only; a resume takes the current copy; a model it lacks gets the custom-id warning | `TestSessionKeepsModelData_0072` | D19; case 9 |
| M61 | A session that began with none | decided | takes the first download when it lands and keeps it | `TestSessionWithNoneGainsFirstCopy_0072` | D19; call 1 |
| M62 | Phase 4, gobble's pin of `v1.33.0` *(2026-11-25: Phase 2b, on `v1.33.0`, which 0071's pin already holds)* | decided | the current document handed to every listing; the session's to every lookup and every `provider.New` for that session, by 0093's hand-in; `ModelMetadataURL`, the layer and `New`'s `WithoutModelMetadata` gone *(2026-11-25: `ModelMetadataURL` and `WithoutModelMetadata` are gone with 0071's pin (its P2); the listing's current document is handed from Phase 1; Phase 2b takes the layer out and hands the session's document to the lookups and to `New`)* | `TestListingGetsModelData_0072` (no metadata request; ranked by the handed document); `TestProviderGetsSessionModelData_0072` (turn, compaction, switched, title) | D17; Q1; call 4 |
| M63 | No data in use (pinned with no copy; the last attempt failed) | decided | D13's warning once per run, naming `gobble models refresh`; `gobble acp`: the log; none before a first attempt | `TestModelDataWarning_0072/none_pinned`, `…/none_failed`, `…/none_untried` | D13; case 4 |
| M64 | The binary tests (F8) | decided | every proxy variable a closed port; after the run, no `model-data/models-dev.json` under their `GOBBLE_HOME` | `TestGobbleACPBinary`; the Windows conpty test | D10 |
| M65 | The engine and the rest of the run | decided | its own goroutine; a prompt and a tool call complete while a fetch blocks; readers never lock | `TestModelDataEngineNeverBlocks_0072` (`-race`) | D18; case 7 |
| M66 | `attempt.json`: its shape, an unreadable one, two writers | decided | D18's object, written at start and end by rename, last writer wins; unreadable is no attempt, logged once, replaced | `TestModelDataAttemptFile_0072` (`shape`, `unreadable`, `two_writers`) | D18; case 7 |
| M67 | An automatic download refused | decided | counts as an attempt (the hour's wait); logged once with its reason; named, with `--force`, in the age warning | `TestModelDataRefreshTrigger_0072/refused_waits`; `TestModelDataWarning_0072/refused` | D8, D12, D18; case 5 |
| M68 | `gobble models refresh --force` | decided | skips the section and shrink guards; still refuses an empty document | `TestModelsRefreshCmd_0072/force`, `…/force_empty` | D6, D9; case 5 |
| M69 | A download that cannot be saved | decided | current for this run, in memory; logged once; `attempt.json` `not_saved` where writable; the hour's wait applies | `TestModelDataNotSaved_0072` | D18; case 6 |
| M70 | `refreshHours` 8,760 and 8,761; `warnDays` 3,650 and 3,651 | decided | the bound accepted; one past it "must be at most N", the default used | `TestModelDataSettingBounds_0072` | D11; case 10 |
| M71 | The first download lands for a session that began with none, a prompt under way | decided | the prompt's calls finish on the provider it took; its next prompt or compaction builds with the document; a session holding a document is not rebuilt | `TestSessionProviderRebuiltOnFirstCopy_0072` | D22; calls 1 and 4 |
| M72 | The cost of a refresh | decided | providers built across a refresh: one per none-session, none for the rest; a document no session holds is released (a weak pointer to it reads nil after a collection) | `TestModelDataRefreshCost_0072` | D23 |
| M73 | A provider's requests on `v1.33.0` | decided | read exactly `New`'s document: OpenCode routes a model by the session's document; with none, by its table | `TestOpenCodeRoutesFromSessionModelData_0072` | D17; 0093 Q4 |

Counts: 71 decided, 2 out of scope, 0 open; 73 rows. *(First draft: 49 decided, 4 out of scope, 4 open; 57 rows. Second draft: 68 decided, 2 out of scope, 0 open; 70 rows.)* *(2026-11-25, "gobble: v1.33.0 pin first, then 0072 (Recommended)": M22 struck; 70 decided, 2 out of scope, 1 struck, 0 open; 73 rows.)*

## Pros and Cons of the Options

### A. gobble's fetch, a JSON file, handed to the SDK

* Good, because it is the owner's rulings, and it survives the SDK's v1.33.0.
* Good, because gobble keeps the ETag and date, so an unchanged document costs a 304.
* Bad, because gobble carries an HTTP fetch, its limit, its deadline and its redirects.

### B. Through the SDK's `LookupMetadataWith`

* Good, because it reuses the SDK's fetch on `v1.32.0`.
* Bad, because v1.33.0 removes it (F4).

### C. An embedded key-value store

* Good, because it could hold more than one document and attempt history.
* Bad, because one document written whole by a rename needs neither; a new dependency and file format for no case this
  record has.

### D. The embedded copy, refreshed at release

* Good, because nothing runs at run time.
* Bad, because data is hard-coded and as old as the release.

## Owner Questions

All answered; no question is open. The answers are in "Owner rulings" above, each with its rows.

| Question | Answer, as given | Rows |
| :--- | :--- | :--- |
| Q1. The source of the model data | "make models.dev the source-of-truth for both the SDK and gobble." | M8, M22, M23, M55, M62 |
| Q2. Refresh and warning ages | "Refresh 24h, warn past 7 days (Recommended)" | M36 |
| Q3. The built-in copy | "None; remove in 0072 (Recommended)" (superseding "Keep until the v1.33.0 pin (Recommended)"); persistence "JSON file, atomic write (Recommended)" | M2, M19, M33, M47 |
| Call 1. Does a session that began with no data take the first download? | "Pick up the first download (Recommended)" | M61, M71 |
| Call 2. Does `--force` save an empty document? | "Still refuse an empty document (Recommended)" | M14, M68 |
| Call 3. Which attempts start the hour's wait? | "Any attempt gates an hour (Recommended)" | M26, M67 |
| Call 4. Is the document handed to `provider.New`? | "for both 0093 and here, i want to hand to provider.New, but does that mean it is hard coded?" (not hard-coded: the run-time document is handed) | M62, M71 to M73 |

## More Information

### Relation to other records

* go-llmprovider-sdk `0093-MADR-take-model-metadata-from-the-caller.md` (draft): the hand-in Phase 4 uses, and the removals
  that make Phase 4 part of gobble's pin of `v1.33.0`.
* `0005-MADR-v1-feature-scope.md`'s "Catalog refresh" row: built by this record (D16).
* `0066-MADR-set-each-session-model-up-from-discovered-limits.md`: D3's "not handed to the SDK for limits" superseded by Q1 from
  Phase 4 (M55).
* `0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md`: its cap reads the session's document (D16); its
  Phase 2 follows this record's Phase 4 (M54).
* `0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md`: no shared behaviour (M53).
* `0046-MADR-own-background-listings-and-keep-tests-hermetic.md`, `0067-MADR-windows-sharing-violation-when-reading-auth-json.md`,
  `0008-MADR-native-cli-mode.md` (D5's start-up rule).

### What could not be established offline

* Whether models.dev sends an ETag or `Last-Modified`, how it redirects, and the document's size today: no download was made.
  M9 and M58 are decided either way.
* The name of 0093's hand-in: that record is a draft; Phase 4 re-grounds it at the tag. *(2026-11-25: settled below.)*

## Amendment of 2026-11-25: built on v1.33.0

Additions only. The owner's ruling of 2026-11-25, relayed by the coordinator, quoted exactly: **"gobble: v1.33.0 pin first,
then 0072 (Recommended)"**. gobble pins go-llmprovider-sdk `v1.33.0` (tagged at `138d52f`) with
[0071-PLAN-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-PLAN-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md)'s
Phase 2, then builds this record on it. Nothing here exists only for `v1.32.0` any more.

### Re-grounded at 073fd1f and at the SDK's tag

* **gobble.** Since 61569d0, 0070's build moved two of this record's citations: F4a's `acpserver/agent.go:1102-1124` is
  `:1141-1165` (`sessionProvider`), and F2's initialize image check `:1145-1152` is `:1186-1193` (`promptImage`);
  `internal/config/keys.go:78-143` is `:78-145` (a key added after `:124`). Re-read and unchanged: `acpserver/agent.go:457-464`,
  `:701`; `acpserver/compact.go:27`, `:81`; `llm/provider/provider.go:53-55`, `:212-216`, `:215`; `llm/provider/model.go:18-26`,
  `:47-53`, `:107-115`, `:142-151`, `:176`; `llm/provider/list.go:129-130`, `:159-160`; `internal/cli/agent.go:35`, `:80`, `:87`,
  `:117`; `go.mod:16` (`v1.32.0`, until 0071's pin).
* **F4, at the tag.** The SDK's 0093 is built and tagged: the hand-in is `catalog.WithMetadata(m Metadata)`
  (`llmprovider/catalog/model_metadata.go:242-260`), a shared option; the eight names its D10 lists are removed (the SDK's
  CHANGELOG `[v1.33.0]`, "Removed"); a provider's requests read only the document given to its `New` (0093 Q4); with no
  document every reader behaves as with none, and a Claude or OpenCode messages-route request that names no limit, with no other
  source, is refused before sending (0093 D3, D8), which no gobble call meets (0071 F21). `ParseMetadata` and `Encode` are kept
  unchanged (0093 MH43). F3 is history: no phase runs where the SDK fetched.

### What changes here

* **M22** is struck (its row). The listing's document before this record is 0071's P6, under 0071's Q5; from Phase 1, the
  current copy, through `Options.ModelData`.
* **D17**'s `v1.32.0` sentence is struck, and its Phase 4 is Phase 2b, built after Phase 2 and before Phase 3 (D17's
  annotation). 0071's Phase 2 comes before this record (M54).
* **D21**'s first three names are 0071's P3, nil handing none (D21's annotation).
* **D14 and M47**: the removal is on `v1.33.0`.
* **M55, M62, M71 to M73** are built in Phase 2b. Their tests' "red at the base" is no longer "`v1.32.0`'s code does not
  compile on `v1.33.0`": at Phase 2's end the lookups hold the built-in copy as a layer (or none, 0071 Q5) and `New` none, so
  each test is red on its assertion, or by its plant where it names a new name.
* **What moved to 0071's pin:** the removal of `WithModelMetadataURL` (`list.go:129-130`, `:159-160`), `Options.ModelMetadataURL`
  (`provider.go:53-55`) and `New`'s and the lookups' `WithoutModelMetadata` (`provider.go:215`, `model.go:176`); `ModelData`,
  `ParseModelData`, `Options.ModelData`; the tests that served a document by URL.

No row of this record is open. 0071's Q5 decides what the listings, lookups and providers read until Phases 1 and 2b replace it.

### Second addition (2026-11-25): 0071's Q5 and Q6 answered

The owner answered 0071's Q5, "Built-in copy to listings only (Recommended)", and Q6, "Hand them in as a caller layer
(Recommended)", relayed by the coordinator. So before this record the listings get the built-in copy, the lookups and limits
calls keep it as a "models.dev" layer, and `New` gets none (0071 P6); every lookup, limits call, discovery and `New` also gets
the provider's saved caps as a "service-stated" caller layer below models.json (0071 P24). What that means here, as rows:

| Row | Input or state | Status | Outcome | Test | Decided by |
| :--- | :--- | :--- | :--- | :--- | :--- |
| M74 | 0071's `TestListingRanksByBuiltInCopy_0071` once Phase 1 hands the listings the current copy | decided | re-pointed: it becomes M19's `listing` case, the listing's `Options.ModelData` the current document (the built-in copy until Phase 3, as M19 says) | `TestModelDataCurrent_0072/listing` | 0071 Q5; M19 |
| M75 | 0066's `TestLimits_BuiltInCopyBelowListing_0066` once Phase 2b hands the session's document | decided | retired: the built-in copy's layer is gone, and M55's `TestLookupGetsSessionModelData_0072` holds the document's place below the listing | `TestLookupGetsSessionModelData_0072`; the retired test's name in the execution record | 0071 Q5; M55 |
| M76 | The saved caps' layer beside the session's document (Phase 2b) | decided | unchanged by the hand-in: every lookup, limits call and `New` built for a session gets the session's document by `WithMetadata` and the saved caps' layer by `WithModelLayer`, below models.json | `TestProviderGetsSessionModelData_0072` gains a `saved_caps` case | 0071 Q6, P24; M62 |

Counts after this addition: 73 decided, 2 out of scope, 1 struck, 0 open; 76 rows.

### Third addition (2026-11-25): re-grounded at 927c7dd; Phase 0's measurements

The PLAN's Phase 0 ran at `927c7dd` (0071's Phase 2c ended, go-llmprovider-sdk `v1.33.0`); its execution record lists each
moved line. No fact F1 to F12 changed. What Phase 0 measured, and what it adds here:

* **D2, M2: the copy's `sha256` is over the re-encoded form.** A `jsontext.Value` written by `jsonx.MarshalStrict` and read
  back by `jsonx.UnmarshalStrict` is compacted (the 2026-10-14 copy's `Encode` bytes, 515,680, read back as 515,648; the final
  newline and white space dropped), and that form is stable on a second round trip. As the PLAN's Phase 0 step 2 decides, D2's
  `document` is held, and its `sha256` taken, in the compact form, which M2's round trip then keeps byte for byte.
* **M69 covers two more downloads.** The strict policy refuses a `document` whose kept sections repeat a name or hold invalid
  UTF-8, both of which `ParseMetadata` accepts; such a download cannot be saved, which is M69's row.
* **M58: confirmed.** Go's client follows a 302 to another host, and `resp.Request.URL` differs from the URL asked for.
* **M64 gains a test.** `internal/cli/complete/binary_test.go`'s `TestBinaryCompletionMode` builds and runs gobble's binary
  without `internal/cli`'s seams (in completion mode, which M29 says loads and never fetches); it can take a closed-port proxy,
  so M64's list adds it: every proxy variable a closed port, and no `model-data/models-dev.json` under its home after the run.
* **Put to the owner** (the PLAN's Phase 1 stop, Q-A to Q-E): the names Phase 1 needs beyond D21; M15's `section_malformed`,
  since `ParseMetadata` refuses a document with a section that does not decode (F5) before D6's guards meet it; D10's guard in
  `llm/provider`, which turns two existing tests red or skipped; how a session's document reaches `acpserver`'s `ModelInfo`
  and `CachedLimits` hooks (0071 P46), which take no session; and `TestModelsInfo`, whose exact output M38's `data` line
  changes. No row is changed until they are answered.

### Fourth addition (2026-11-25): the answers to Phase 1's stop

The owner answered Q-A, Q-B and Q-D, relayed by the coordinator, each quoted exactly; Q-C and Q-E are the coordinator's
calls, made on the owner's earlier ruling for the same kind of case (0071's Q-H, **"Expect the new output (Recommended)"**),
recorded as such.

| Ruling, as given | Label | Rows |
| :--- | :--- | :--- |
| Q-A. The exported surface Phase 1 needs | "Add the four names (Recommended)" | M77; D21 |
| Q-B. A download whose section the SDK cannot decode | "As a failed download (Recommended)" | M78; M13, M15 |
| Q-C. D10's guard in `llm/provider`: "`offline_test.go:67` and `TestProbePort` run in a second child process without proxy variables. No assertion changes." | the coordinator's call, on 0071 Q-H's precedent | M79; D10, M10 |
| Q-D. The session's document and `acpserver`'s `ModelInfo` and `CachedLimits` hooks | "Hooks take the session's document (Recommended)" | M80; D19, D21 |
| Q-E. `TestModelsInfo` and M38's `data` line | the coordinator's call, on 0071 Q-H's "Expect the new output (Recommended)" | M81; M38 |

| Row | Input or state | Status | Outcome | Test | Decided by |
| :--- | :--- | :--- | :--- | :--- | :--- |
| M77 | D21's names for the stamp, the copy's bytes, the 304 and the URL | decided | `ModelDataStamp` (`Source`, `Fetched`, `ETag`, `LastModified`) is `Stamp()`'s type; `(*ModelData).Stamped(s)` is the same parsed document under stamp `s`, the original unchanged; `(*ModelData).Document()` is `Encode`'s bytes in their compact form, the caller's own copy; `ModelDataURL` is `https://models.dev/api.json`; `FetchModelData(ctx, url, o) (*ModelData, bool, error)` sends the validators of `o.ModelData` when its stamp's source is `url`, and the bool reports a 304, which carries no body and returns `o.ModelData`'s document stamped anew | `TestModelDataStamp_0072` (`llm/provider`); M9's `TestFetchModelDataConditional_0072` | Q-A |
| M78 | A download with a section the SDK cannot decode | decided | `ParseMetadata` refuses the document (F5), so it is M13's failure: nothing changes on disk or in memory; `attempt.json` `failed`, `not_document`; `--force` cannot save it. M15's `section_malformed` is a section the SDK reads as absent (present without `models`, or `null`), which `Encode` drops and D6 refuses as missing | `TestModelDataRefreshFails_0072/section_undecodable`; `TestModelDataGuards_0072/section_malformed` | Q-B; F5 |
| M79 | `llm/provider`'s network guard and the two tests whose dials need no proxy | decided | `TestMain` runs the package's tests behind 0046 D4's guard in one child, and `TestWireOffline` and `TestProbePort` in a second child with no proxy variable, which first checks that `net/http` names no proxy; the guarded child leaves those two out; no assertion changes | the unguarded child's check in `TestMain`; `TestWireOffline`, `TestProbePort` | Q-C |
| M80 | The session's document and `acpserver`'s `ModelInfo` and `CachedLimits` hooks (Phases 2 and 2b) | decided | both hooks take the session's `*provider.ModelData` first (nil: the current); `acpserver` passes the session's; `internal/cli` hands it to the lookups and to `provider.CachedLimits` as `Options.ModelData`; the test literals that set the hooks take the argument, no assertion changes | M50's `TestTurnCapFromModelData_0072` (Phase 2); M55's `TestLookupGetsSessionModelData_0072` (Phase 2b) | Q-D |
| M81 | `TestModelsInfo`, whose stdout M38's `data` line changes (Phase 2) | decided | its expected output gains the `data` line, by a dated note in Phase 2 | `TestModelsInfo` | Q-E; M38 |

What these change above, as annotations: D21 gains M77's four names and M80's two hook signatures; D10's guard is M79's;
M13 and M15 read as M78 says. Counts after this addition: 78 decided, 2 out of scope, 1 struck, 0 open; 81 rows.

### Fifth addition (2026-11-25): the review of Phase 1 and a redirect downgrade

An independent review (`code-reviewer`) of Phase 1, relayed by the coordinator, and the owner's ruling on one case it raised,
quoted exactly. The other findings are fixes inside rows already decided; the PLAN's deviation of this date lists them.

| Ruling, as given | Label | Rows |
| :--- | :--- | :--- |
| A redirect from `https` to `http` | "Follow any, but refuse https→http (Recommended)" | M58, M82; D4 |

| Row | Input or state | Status | Outcome | Test | Decided by |
| :--- | :--- | :--- | :--- | :--- | :--- |
| M82 | A redirect from an `https` URL to an `http` one | decided | the download fails before the `http` request is sent: nothing changes on disk or in memory; `attempt.json` `failed`, `redirect_downgrade` (`FetchModelData`'s error wraps `llm.ErrNotPermitted`); a redirect to any other host over `https` is still followed (M58) | `TestFetchModelDataRedirect_0072/https_to_http`; `TestModelDataRefreshFails_0072/redirect_downgrade` | the owner's ruling above; M58 |

What the review changes above, as annotations: D2's write is `internal/fsx`'s atomic writer, which syncs the file and its
directory (its temporary file is `.<name>.gobble-<hex>.tmp`, and a file with more than one hard link is written in place, as
that writer does for every file); the sweep reads the directory with `os.ReadDir` and also removes `attempt.json`'s temporary
files; D4's 32 MiB body limit is gobble's own, ahead of the SDK's. Counts after this addition: 79 decided, 2 out of scope, 1
struck, 0 open; 82 rows.
