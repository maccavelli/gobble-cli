---
status: complete
date: 2026-11-08
associated-madr: "0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement the stable ACP surface gobble can honour: conformance, notices, compaction updates, and the records

Associated MADR: [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md)
(its "Refresh (2026-11-08)": the open items D6 to D11 and D13, the recommended options, and Q1 to Q7).

## Goal

* gobble's ACP goldens are checked by the SDK's conformance validator (D6).
* The agent asks the SDK's gate before a capability-gated update (D7).
* MCP and prompt diagnostics reach a client that offered `session.notices` as notices, and nothing new reaches one that
  did not (D8).
* Compaction reaches a client that offered `session.compaction` through the standard updates, with its summary, and
  through `_gobble/compaction` and today's text otherwise; no client receives one compaction twice (D9).
* gobble's own line client offers `session.compaction` and renders it with the SDK's fold (D10).
* The records that call stable surfaces unstable carry dated notes (D11), and D13's surfaces stay with their owners.

The plan is complete when the Verification holds. It carries the MADR's recommended set; an owner's answer that picks
another option amends this plan before the phase it touches runs.

## Scope

**Base.** `main` at `4970cca`, equal to `origin/main`, acp-go-sdk2 `v1.34.2`, acpext `v0.1.0`.

**In scope,** by phase:

* P0, the records: the MADR's answers recorded; D11's dated notes in `docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md`,
  `0004-MADR-go-module-architecture.md` and `0005-MADR-v1-feature-scope.md`; `docs/README.md`'s 0016 rows.
* ~~P1, conformance (D6): `acpclient/acptest` (a records adapter), `acpserver/agent_test.go` (`TestSession`),
  `acpclient/acptest/pair_test.go` (`TestPairInitialize`).~~ *(Q1 answered Option 2.)* P1, the SDK's recorder (D6):
  `acpclient/acptest` (`recorder.go` reduced to the host-path normalizer, `pair.go`), `acpclient/acptest/pair_test.go`,
  `acpserver/agent_test.go`, and the two ACP goldens, rewritten once:
  `acpserver/testdata/session.golden` and `acpclient/acptest/testdata/initialize.golden`.
* P2, notices (D7, D8): `acpserver` (a gate helper; `mcp.go`, `sysprompt.go`, `agent.go`'s call to `mcpTools`), and
  their tests.
* ~~P3, compaction (D9): `acpserver/compact.go`, `extensions.go` (`_gobble/compact`), `autocompact.go`, `history.go`
  (replay), and their tests.~~ *(2026-11-08, the MADR's "P3 re-assessment": P3 is three phases.)*
  * P3a, the contract: `acpext/types.go` (`CompactionMeta`), `acpext/doc.go`, `acpext/roundtrip_test.go`,
    `acpext/testdata/compaction.meta.*.json`; then the tag `acpext/v0.2.0`, which the owner pushes.
  * P3b, the id: `session/tree.go` (a reservation `Next` honours), `session/tree_test.go`.
  * P3c, the reporter: `go.mod` and `go.sum` (`acpext v0.2.0`), `acpserver/compact.go`, `extensions.go`
    (`_gobble/compact`), `autocompact.go`, `history.go` (replay), `agent.go` (`liveSession.append`), and their tests.
* P4, gobble's client (D10): `acpclient/conn.go`, `acpclient/update.go`, `internal/cli` (line mode's options and the
  view), and their tests.
* P5, acceptance: the goldens, the docs (`docs/architecture.md`'s `acpserver` and `acpclient` rows), the records.

**Out of scope:**

* D13's surfaces: `session/delete`, elicitation, images and audio, boolean config options, `session/list` pagination.
* ~~Any change to `acpext`~~ *(2026-11-08, P3-Q3: P3a adds `CompactionMeta`)*. Any other change to `acpext`:
  `_gobble/compaction` and `CompactionNotice` stay as published at `v0.1.0`.
* magic-cli-remote: its 0179 D10 needs an amendment in that repository (MADR R11); telling Magic is the owner's.
* A gobble release.

## Conventions every phase follows

* **C1. Start state.** Each phase re-reads `git status`, `HEAD` and `origin/main`; a difference from this plan is a
  deviation.
* **C2. Seen failing.** Every new test or check is seen failing first on a scratch copy, never in the working tree:
  against the code before the phase, and against a planted break of the new code. The record quotes each failure.
* **C3. No real effects.** Every scratch run sets `HTTPS_PROXY` and `HTTP_PROXY` to a closed loopback port, and `HOME`
  and `XDG_*` into the scratch directory; `internal/cli`'s guard stays on.
* **C4. The wire.** A golden whose client offers neither capability stays byte-identical (`git diff --quiet`). A golden
  that changes is a deviation unless the phase names it. P1 names one: the two ACP goldens are rewritten once into the
  SDK's record format, with the same frames (P1 step 5); from P1's commit on, they are compared with that commit's
  files.
* **C5. Gates.** `make pre-add-check FILES=…` on every changed Go file, `make lint`, `make preflight`, `make race`; the
  tests and race on the Linux machine and the Windows laptop, the tree shipped as before; the identifier scan.
* **C6. Commits.** The agent stages; the owner commits on `main` with `git commit --no-edit`. Pushing and tags are the
  owner's.
* **C7. Deviations stop the phase,** and are recorded here, and in the MADR when a decision or fact changes, before
  work goes on.

## Implementation Steps

### P0 — the records (D11, Q1 to Q7)

**Gate:** the owner's answers to Q1 to Q7.

1. The MADR's refresh records each answer, dated, and the recommended set becomes the decision where the answer is the
   recommendation; any other answer amends the phase it touches here.
2. **D11's notes** (Option 1), each dated 2026-11-08 and citing schema 1.25.0's stable lists (MADR R6), at:
   * 0004-MADR `:45-47` (delete and elicitation are stable);
   * 0005-MADR `:148` (`session/delete`), `:192` and `:629` (elicitation);
   * 0002-MADR `:977` (`usage_update` is standard) and `:1030` (`session/delete`).
3. `docs/README.md`: the 0016 MADR row, and a PLAN row.
4. `make check-records markdownlint` passes. Staged; the owner commits.

### P1 — the SDK's recorder and conformance under the goldens (D6, ~~Option 1~~ Option 2, Q1)

**Gate:** P0 committed.

~~1. **The adapter,** in `acpclient/acptest` … `GoldenRecords(path)` … 2. **The tests:** `TestSession` and
`TestPairInitialize` call `acptest.ValidateGolden(t, records)` after their byte comparison.~~ *(Q1 answered Option 2;
the steps below replace them.)*

1. **Read first, at acp-go-sdk2 `v1.34.2`:** `acptest.Recorder` (`acptest/recorder.go:39-87`: `Writer`, `Reader`,
   `Records`, `Wait`), `Record` and `Direction` (`Sent`, `Received`, `:13-28`), `Golden` (`acptest/golden.go:91`),
   `ValidateGolden` (`:41`, its direction tables at `:263`), and `WriteRecords` / `ReadRecords` (`records.go:17`, `:44`).
   Record how `ValidateGolden` reads a direction against the side that recorded, since gobble's pair records from the
   client.
2. **The pair records with the SDK's recorder.** `acpclient/acptest`'s `Pair` (`pair.go:21-46`) wraps the client's
   pipes with `acptest.Recorder`'s `Writer` (client→agent, `Sent`) and `Reader` (agent→client, `Received`), and returns
   it in place of gobble's `*Recorder`. `Wait` keeps its role in the tests that wait for frames.
3. **The one gobble piece kept:** `hostIndependent` (`acpclient/acptest/recorder.go:165-189`, 0019-MADR D4) becomes a
   function over `[]acptest.Record`, rewriting each path under `Abs("/")` in slash form, so that a golden recorded on
   macOS matches on Windows. It is applied before `acptest.Golden`. Every other part of gobble's `Recorder`
   (`Frame`, `Direction`, `Transcript`, `Golden`, `firstDiff`) is deleted.
4. **The tests:** `TestSession` (`acpserver/agent_test.go:127`) and `TestPairInitialize` (`pair_test.go:103`) call
   `acptest.Golden(t, path, hostIndependent(rec.Records()))`, then `acptest.ValidateGolden(t, rec.Records())`.
   `pair_test.go`'s tests of gobble's own golden mismatch and missing file (`:192`, `:201`) are removed: they tested the
   deleted code, and the SDK tests its own `Golden`.
5. **The goldens, rewritten once** with `ACPTEST_UPDATE=1`, into the SDK's record format. **Proof the frames are the
   same:** a scratch script decodes the old files' `client→agent` / `agent→client` lines and the new files' records,
   canonicalizes each frame (RFC 8785), and compares the two sequences, direction and frame, one for one; the record
   quotes the counts (13 and 2 frames at `4970cca`) and the result.
6. **Seen failing** (C2), on scratch copies:
   * a new golden with a `session/update` frame lacking `sessionId`: `ValidateGolden` fails, "missing required property
     \"sessionId\"";
   * a string `protocolVersion`: `ValidateGolden` fails;
   * a changed frame: `acptest.Golden` fails, naming the first differing record;
   * `hostIndependent` removed, on the Windows laptop: `TestSession` fails on a path; restored, it passes.
   An unbroken copy passes on all three platforms.
7. Gates (C5), the Windows laptop among them for step 6's host check. **Commit.**

### P2 — the gate helper and the notices (D7 and D8, Option 1 each)

**Gate:** P1 committed.

1. **The tests first,** each on `acptest`'s pair, with a client that offers `session.notices` and one that does not:
   * `TestNoticesWhenOffered`: an MCP server that fails, an MCP config error, an ignored exposure, a server still
     connecting, and a prompt file not used each send one `notice` with D8's severity (`warning`, `warning`,
     `warning`, `info`, `warning`), the log message as its title and the detail as its description; each keeps its
     once-per-session rule (a second prompt sends none again).
   * `TestNoNoticeWithoutTheCapability`: the same events send no `notice` and no agent text; the log records are as
     today.
   * `TestNoticesAreNotReplayed`: a `session/load` of that session replays no `notice`.
2. **The code:**
   * `acpserver`: `func (a *Agent) offers(su acp.SessionUpdate) bool` (D7), over `a.conn.Negotiated()` and
     `Negotiation.AllowsValue(acp.ClientMethodSessionUpdate, acp.SessionNotification{Update: su, …})`; false before
     initialize.
   * `mcpTools` takes the session (`acpserver/agent.go:675`), and the four MCP diagnostics and `note`
     (`acpserver/sysprompt.go:223`) send `acp.UpdateNotice(severity, title)` with its description when `offers` says
     yes, beside the unchanged log record.
3. **Seen failing** (C2): each test against the code before step 2; then the planted breaks: the gate inverted (a
   notice sent without the capability, which the SDK refuses with `*CapabilityError`, so the test sees an error or no
   frame); a severity swapped; the once-per-session rule removed.
4. Gates (C5). `session.golden` is byte-identical (C4). **Commit.**

### ~~P3 — compaction through the standard updates (D9, Option 1)~~

*(2026-11-08: replaced by P3a, P3b and P3c below, from the MADR's "P3 re-assessment", P3-Q1 to P3-Q5. The first
version's chunk, its `Options.NewID` id, and its untyped `_meta` are withdrawn.)*

### P3a — the contract: `acpext.CompactionMeta` (D9; P3-Q3)

**Gate:** P2 committed.

1. **The type,** in `acpext/types.go`, standard library only (`check-acpext-deps`):

   ```go
   // CompactionMeta is gobble's own facts about a compaction, sent under
   // _meta[MetaKey] of ACP's compaction_update when the client offered
   // session.compaction (0016-MADR D9, P3-Q3). _meta is replaced whole by
   // each update that carries it, so each carries all of it.
   type CompactionMeta struct {
       Reason       CompactionReason `json:"reason,omitzero"`
       TokensBefore int64            `json:"tokensBefore,omitzero"`
       Paused       CompactionPause  `json:"paused,omitzero"`
   }
   ```

   It reuses `CompactionReason` and `CompactionPause`; `CompactionNotice` and every `v0.1.0` name are unchanged.
   `doc.go` says where it is sent and that `_gobble/compaction` remains for a client without the capability.
2. **Tests:** `roundtrip_test.go` decodes three new goldens, `testdata/compaction.meta.{start,end,paused}.json`, with
   unknown members refused, and re-encodes them byte for byte; `apidiff` reports one compatible addition.
3. **Seen failing** (C2): a golden with a misspelt member (refused), and one with `tokensBefore` as a string.
4. Gates (C5), `make check-acpext-deps` among them. **Commit.** The owner tags `acpext/v0.2.0` on that commit and
   pushes the tag; the module proxy serving it is P3c's gate (0042-MADR, the tag before the root uses it).

### P3b — the id: a reservation in `session.Tree` (D9; P3-Q1)

**Gate:** P3a committed.

1. **The tests first** (`session/tree_test.go`):
   * `TestReserveIsHonoured`: `Reserve()` returns an id not in the tree; an entry appended with that id keeps it, with
     its parent the leaf, as `Next` gives any entry;
   * `TestReserveIsNotReissued`: while reserved, no `Next` hands the id out;
   * `TestReleaseFreesTheId`: `Release(id)` returns an unused reservation; a reservation is never written by itself.
2. **The code:** `Tree.Reserve() string` and `Tree.Release(id string)`, under the tree's mutex; `Next` keeps an entry's
   id when it is a live reservation, and consumes it; `newID` skips reserved ids. No session file change.
3. **Seen failing** (C2): each test against the code before step 2; then the breaks: `Next` overwriting a reserved id;
   `newID` not skipping reservations.
4. Gates (C5). **Commit.**

### P3c — the reporter: one channel per connection (D9; P3-Q1 to P3-Q5)

**Gate:** P3b committed; `acpext/v0.2.0` served by the module proxy; and the owner's word that the gobble release
Magic captures from may carry it, with magic-cli-remote's 0179 amended or scheduled (MADR M6, R12).

1. **The tests first,** for each path (`/compact`, `_gobble/compact`, automatic at the threshold, on overflow), with a
   client that offers `session.compaction` (and `session.notices` where named) and one that does not:
   * **offered, success:** `compaction_update` `in_progress` with `_meta.gobble {reason}` once the run is prepared, then
     `completed` with the summary in its `summary` field and `_meta.gobble {reason, tokensBefore}`; one id, the
     compaction entry's (`entries[len-1].ID` equals the id); then `usage_update` as today; **no `_gobble/compaction`**
     and, for `/compact`, no compaction agent text;
   * **offered, failure:** `failed`, its `error` the first line, `_meta.gobble {reason}`; on an automatic failure that
     pauses, `_meta.gobble {reason, paused}` and, with `session.notices`, one `warning` notice saying automatic
     compaction is paused and how it resumes (D8's path; none without `session.notices`);
   * **offered, cancel:** before the entry is written, `cancelled`; after it, `completed` (P3-Q4);
   * **offered, refusal** ("Nothing to compact …", "Already compacted"): no compaction update; `/compact` answers with
     its one-line reply, `_gobble/compact` with -32600 as today;
   * **not offered:** today's frames, `_gobble/compaction` and agent text, byte for byte; `TestExtensionCompact`,
     `TestNotifyFramesUnchanged`, `TestAcpextNotificationsAreSent` and the autocompact tests pass unchanged;
   * **replay:** a `session/load` sends, for an offering client, one `compaction_update` `completed` with the entry's
     summary, the entry's id, and `_meta.gobble {tokensBefore}`; for any other, today's "Compacted from N tokens:" text;
     the replayed id equals the id the live run sent;
   * **conformance:** the offered frames pass `acptest.ValidateGolden`.
2. **The code:**
   * `go get github.com/maccavelli/gobble-cli/acpext@v0.2.0`, then `go mod tidy`.
   * `runCompaction` reserves the compaction entry's id before `compaction.Compact` (`Tree.Reserve`), passes it to the
     reporter, writes the entry with it, and releases it on any path that writes nothing.
   * A small reporter interface, chosen once per call by D7's `offers`, before anything is sent (MADR G1):
     * the standard reporter sends `acp.UpdateCompaction(id, status)` with `Summary` set by `acp.PatchValue` of one
       text block, `Error` by `acp.PatchValue` of the first line, and `Meta` by `acp.PatchValue` of
       `{"gobble": CompactionMeta{…}}` encoded through `jsonx`; it sends the pause notice through D8's helper;
     * the extension reporter is today's `out.notify(acpext.MethodCompaction, …)` calls and text, unchanged, so
       `TestAcpextNotificationsAreSent` still finds its literal `notify` calls.
   * `/compact` and `_gobble/compact` choose `completed` over `cancelled` when the entry was written.
   * `replayMessage` (`acpserver/history.go:359-360`) chooses the same way, with the entry's id.
3. **Seen failing** (C2), each break a planted change on a scratch copy:
   * both reporters at once: a `_gobble/compaction` reaches the offering client;
   * the reporter chosen after sending: an ungated `compaction_update` fails the turn (`*CapabilityError`, MADR G1);
   * the summary as a chunk instead of the field; `_meta` sent without `reason` on the terminal update (S2);
   * a fresh id instead of the reservation: the live and replayed ids differ;
   * a written-then-cancelled run reported `cancelled`;
   * a pause without its notice when `session.notices` was offered;
   * replay ungated: the SDK refuses it.
4. Gates (C5). `session.golden` and `TestNotifyFramesUnchanged`'s frames byte-identical (C4). **Commit.**
5. **For the owner, not posted:** a note for magic-cli-remote listing 0179's lines to amend (MADR M6).

### P4 — gobble's own client (D10, Option 1)

**Gate:** P3 committed.

1. **The tests first:**
   * `acpclient`: `TestTranslateStandardCompaction`: with `Options.Compaction`, `initialize` offers
     `session.compaction`, and `compaction_update` and `compaction_summary_chunk` fold, through the SDK's
     `CompactionTimeline`, into `KindCompaction` updates carrying the status, the summary and `_meta.gobble`'s reason
     and tokens; without it, nothing is offered and the variants stay `KindOther` as today.
   * `internal/cli`: line mode offers it, and draws an automatic compaction with today's words ("compacting the
     conversation ({reason})", "compaction failed: {error}" and its paused forms); print mode offers nothing, and
     `print-json.golden` and `print-stream-json.golden` are byte-identical.
   * *(2026-11-08, deviation 2)* `internal/cli`: line mode draws a **manual** compaction from its rows, in `/compact`'s
     words and the session's voice, as reply text: `in_progress` gives "Compacting the conversation…"; `completed`
     gives "Compacted from {tokens} tokens." and the summary; `failed` gives "Compaction failed: {error}". Its stdout
     equals, byte for byte, what the same compaction drew as `/compact`'s agent text.
     `acptest.ScriptAgent` records each `initialize`'s client capabilities, so the tests can show what each mode
     offered.
2. **The code:** `acpclient.Options.Compaction`; `initialize` sets `ClientCapabilities.Session.Compaction`; `translate`
   keeps one `CompactionTimeline` per session; line mode sets the option. Notices are not offered (MADR D10).
   *(2026-11-08, deviation 2: the line view draws a manual compaction from its rows, since P3c sends a client that
   offers `session.compaction` no `/compact` text.)*
3. **Seen failing** (C2): each test against the code before step 2; then the planted breaks: the option not reaching
   `initialize`; a chunk dropped from the fold; print mode offering it. *(Deviation 2 adds a fourth: the view drawing
   nothing for a manual row.)*
4. Gates (C5), with `make perf`, since the view changes. **Commit.**

### P5 — acceptance

**Gate:** P4 committed.

1. `acpserver/testdata/session.golden` and `acpclient/acptest/testdata/initialize.golden` are byte-identical to P1's
   commit (their one rewrite), and `internal/cli/testdata`'s goldens to `4970cca`'s; each ACP golden passes
   `ValidateGolden`.
2. A scratch run of `gobble acp` driven by `acptest` with a client that offers both capabilities, through a failing MCP
   server and a `/compact`, records the notice and the compaction updates; the frames pass `ValidateGolden`.
3. **Records:** the MADR's refresh marks its items done with their evidence; `docs/architecture.md`'s `acpserver` and
   `acpclient` rows; this plan `complete`; `docs/README.md`'s rows. A note for magic-cli-remote's 0179 D10 is drafted
   for the owner, not posted.

## Verification

| Item | Where |
| :--- | :--- |
| D6: goldens conform | P1 step 2; P5 step 1 |
| D7: the SDK's gate decides | P2 step 2; its gate-inverted break |
| D8: notices when offered, nothing new otherwise, not replayed | P2 step 1 |
| D9: one channel per connection; replay gated; terminal status | ~~P3 step 1~~ P3c step 1 |
| D9: `CompactionMeta` published and decoded strictly | P3a step 2 |
| D9: one id live and on replay | P3b step 1; P3c step 1 (replay) |
| D10: the line client offers compaction; print mode unchanged | P4 step 1 |
| D11: the dated notes | P0 step 2 |
| The wire without the capabilities | C4 in every phase; P5 step 1 |

## Rollout and Rollback

* **Rollout.** Each phase is one commit on `main`. P3a's commit is tagged `acpext/v0.2.0` by the owner before P3c
  requires it. P3c changes what Magic receives from gobble, so it lands only with the owner's word (its gate), ahead
  of or with the gobble release Magic captures from.
* **Rollback.** P1, P2 and P4 each revert alone. P3c reverts alone, restoring `_gobble/compaction` and text for every
  client; P4 reverts first if P3c is reverted. P3b reverts alone once P3c is reverted. P3a's tag is never moved or
  deleted: a reverted P3c simply stops using `CompactionMeta`, which stays in `acpext/v0.2.0`.

## Execution record

~~Not started. The plan is `proposed` until the owner approves it.~~ *Approved 2026-11-08 by the owner's "committed,
proceed", after the records commit `ea4c179`. The plan is `in-progress`.*

### Answers (2026-11-08)

* **Q1 to Q7 answered** by the owner (the MADR's table): D6 Option 2, "Adopt SDK recorder"; D7, D8, D9, D10, D11 and
  D13 Option 1. P1 is rewritten for Option 2 above, its first steps struck through; the other phases stand as written.
* **Found while rewriting P1:** gobble's `Recorder.Golden` normalizes host paths (`hostIndependent`, 0019-MADR D4),
  which the SDK's `acptest.Golden` does not. P1 keeps that normalizer, over the SDK's records. The MADR records it.
* **Files added to scope:** P1's `acpclient/acptest/recorder.go`, `pair.go` and `pair_test.go`, and the two ACP
  goldens, rewritten once.

### P3 re-assessment (2026-11-08)

* **Found,** on the owner's request to research P3 more deeply across magic-cli-remote and this repository: the MADR's
  "P3 re-assessment" records M1 to M6, S1 to S6 and G1 to G6, read at gobble-cli `df5c21e`, acp-go-sdk2 `v1.34.2` and
  magic-cli-remote `c87d22e4`.
* **Decided,** by the owner's answers to P3-Q1 to P3-Q5, each the recommended option: the id is the compaction entry's,
  reserved before the run; the summary travels in the `completed` update's `summary` field; gobble's facts are a typed
  `acpext.CompactionMeta` under `_meta.gobble`, published as `acpext/v0.2.0` first; `/compact` sends the row only, and
  a run written before a cancel reports `completed`; a pause is `failed`, `_meta.gobble.paused`, and a `warning`
  notice when notices are offered.
* **Changed here:** P3 becomes P3a, P3b and P3c (the first version struck through); Scope gains acpext and
  `session/tree.go`; Rollout orders the tag before P3c.

### P0 record (2026-11-08)

The base is `ea4c179`, equal to the tree (C1: `git status` clean; `origin/main` at `4970cca`, the records commits
`df5c21e` and `ea4c179` ahead of it). No deviation.

1. **The answers** (step 1) are recorded in the MADR's refresh (Q1 to Q7) and its P3 re-assessment (P3-Q1 to P3-Q5),
   both committed before this phase (`df5c21e`, `ea4c179`).
2. **D11's notes** (step 2), each dated 2026-11-08, citing 0016-MADR D11 and schema 1.25.0 through acp-go-sdk2
   `v1.34.2`, each placed after the claim it corrects, the claim left as written. A script asserted each anchor occurs
   once before writing; `grep -c '2026-11-08, 0016-MADR D11'` then counted 1, 3 and 2:
   * `docs/decisions/0004-MADR-go-module-architecture.md` `:45-47`: delete and elicitation are stable;
   * `docs/decisions/0005-MADR-v1-feature-scope.md` `:148` (`session/delete`, gated on `sessionCapabilities.delete`),
     `:192` (elicitation, gated on `elicitation.form` and `.url`) and `:629` (free-text answers need not go to
     `exp/`);
   * `docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md` `:977` (`usage_update` is stable) and `:1030`
     (`session/delete` is stable; gobble still does not advertise it, since `session.Store` has no `Delete`).
3. **`docs/README.md`** (step 3): the 0016 PLAN row says `in-progress`, P0 complete.
4. **Gates** (step 4): `make check-records markdownlint` passed, 0 issues; the identifier scan of the diff found
   nothing. Staged for the owner's commit.

### Deviation 1 (2026-11-08): P1, `acpclient/acptest` is a stable package

* **Found** at P1's start (`HEAD` `5d482d9`, the tree clean), before any code: `acpclient/acptest` declares
  "Stability: stable" (`acpclient/acptest/doc.go`), and 0004-MADR's package map lists it as stable, "In-memory
  agent↔client pair and frame recorder for embedders' tests" (`docs/decisions/0004-MADR-go-module-architecture.md:251`).
  Adopting the SDK's recorder (Q1) removes its exported recorder names and changes `Conn.Recorder`'s and `Pair`'s types.
  `apidiff` is a no-op below `v1.0.0` (`Makefile:177`, `:294-300`), and no repository in the owner's workspace imports
  `gobble-cli/acpclient` (magic-cli-remote does not require gobble-cli). Also found: replacing gobble's `Connect` with the
  SDK's `ConnectRecorded` would change the cleanup 37 tests in 30 files rely on (a 10 s bound, and a failure for any end
  but a close), so the pair keeps gobble's pipes and cleanup and records with the SDK's `Recorder.Writer` and `Reader` on
  the client's side, as `ConnectRecorded` records — step 2 as written.
* **Decision,** the owner's ("Replace, amend 0004"): the stable package's recorder API is replaced, and the change is
  named in 0004-MADR (a dated note after its package map) and in the package doc. Not chosen: deprecated wrappers kept
  to `v1.0.0`; reverting Q1.
* **Files added to scope:** `docs/decisions/0004-MADR-go-module-architecture.md` (the note), `acpclient/acptest/doc.go`,
  `acpserver/commands_test.go` (`assertFramesBefore` reads records), and a new `acpclient/acptest/hostindependent_test.go`
  for `HostIndependent`, now exported, which had no test of its own.

### P1 record (2026-11-08)

The base is `5d482d9`, the tree clean (C1). Deviation 1 above.

1. **Read first** (step 1), at acp-go-sdk2 `v1.34.2`: `acptest.Recorder` (`acptest/recorder.go`: `Writer` records
   `Sent`, `Reader` records `Received`, in canonical encoding/json v1 form), `Record` (`direction`, `frame`), `Golden`
   (`acptest/golden.go:91`, the `ACPTEST_UPDATE` and artifact behaviour kept), `ValidateGolden` (`:41`, which reads `Sent`
   as client→agent and matches a response to its request wherever it is recorded), and `ConnectRecorded`
   (`acptest/pair.go`), which records the client's side with `Writer` and `Reader`.
2. **The pair** (step 2): `Connect` keeps gobble's pipes, logger and cleanup, and records with
   `rec.Writer(toAgentW)` and `rec.Reader(toClientR)` on the client's side; `Conn.Recorder` and `Pair` return
   `*sdkacptest.Recorder`.
3. **The kept piece** (step 3): `recorder.go` holds only `HostIndependent(records)` and its helpers, the internal
   `slashRecords(records, root)` taking the root so a test can run the Windows case anywhere. `filepath.ToSlash` is
   followed by a backslash replacement, which changes nothing on Windows and lets that test run off it. Removed:
   `Direction`, `ClientToAgent`, `AgentToClient`, `Frame`, `Recorder`, its `tap`, `observe`, `Frames`, `Wait`,
   `Transcript`, `Golden` and `firstDiff`. `doc.go` says what the package now does.
4. **The tests** (step 4): `TestSession` (`acpserver/agent_test.go:128`) and `TestPairInitialize`
   (`acpclient/acptest/pair_test.go:104`) call `sdkacptest.Golden(t, path, acptest.HostIndependent(records))`, then
   `sdkacptest.ValidateGolden(t, records)`. `TestConnectRecordsNotifications` looks for a `Received` `session/update`
   and a `Sent` `session/prompt` among the records. `assertFramesBefore` (`acpserver/commands_test.go`) reads records.
   Removed: `TestRecorderJoinsSplitLines` and `TestGoldenWritesArtifact` with its `fakeTB`, which tested the deleted code.
   Added: `TestHostIndependent` (`acpclient/acptest/hostindependent_test.go`).
5. **The goldens, rewritten once** (step 5) with `ACPTEST_UPDATE=1`: `acpserver/testdata/session.golden` (14 lines) and
   `acpclient/acptest/testdata/initialize.golden` (2), each line now `{"direction":…,"frame":…}`. A scratch script read
   `HEAD`'s files and the new ones, encoded each frame canonically, and compared them one for one, direction and frame:
   "old 14 frames, new 14 frames — identical" and "old 2 frames, new 2 frames — identical", exit 0 each. The order held
   under the SDK's read-time recording: `go test -count=50` of `TestSession` and `TestStartFramesPrecedeTheResponse`, and
   of `TestPairInitialize`, `TestConnectRecordsNotifications` and `TestHostIndependent`, passed; `-race -count=10` of the
   first two passed; `-count=20` passed on the Linux machine and the Windows laptop.
6. **Seen failing** (step 6), each break on a scratch copy with the proxy variables at a closed port; the control passed:
   * a `session/update` without `sessionId` among the records: `pair_test.go:106: acptest: record 2, received:
     session/update params: SessionNotification: missing required property "sessionId"`;
   * a string `protocolVersion`: `pair_test.go:106: acptest: record 0, sent: initialize params: InitializeRequest: json:
     cannot unmarshal string into Go struct field .protocolVersion of type acp.ProtocolVersion`;
   * a changed frame in the golden: `pair_test.go:104: acptest: record 0 differs from the golden
     testdata/initialize.golden`;
   * `HostIndependent` rewriting nothing: `hostindependent_test.go:23: record 0 = sent {"cwd":"C:\\work\\a.go",…}, want
     sent {"cwd":"/work/a.go",…}`;
   * **on the Windows laptop**, a copy with `HostIndependent` disabled: `agent_test.go:128: acptest: record 2 differs
     from the golden testdata\session.golden` (`session/new`'s `cwd`); the unbroken copy passed there.
7. **Found by the gates and fixed in the phase:** `gofmt` on `pair_test.go` (blank lines left by the removal), and
   `revive`'s confusing-naming on `hostIndependent` beside `HostIndependent`, renamed `slashRecords`.
8. **Gates** (step 7): `make pre-add-check` on the seven Go files, "7 file(s) clean (gofmt, golint, govulncheck)";
   `make lint` 0 issues; `make preflight` "preflight passed", its 15 gates; `make race` 39 ok; `go test ./...` 38 ok. The
   Linux machine and the Windows laptop, the tree shipped (898 files, no `._*`): `go test` and race 38 ok each, `make
   check-records markdownlint` 0 issues, preflight stopped at `verify-build-metadata` as on every copy.

### P2 record (2026-11-08)

The base is `a233e3f` (P1's commit), the tree clean (C1). No deviation; the delivery mechanism below is how step 2's
"send … when `offers` says yes" is built, with the decision unchanged.

1. **The tests first** (step 1), `acpserver/notices_test.go`, each agent offering what its test names at initialize:
   * `TestNoticesWhenOffered/mcp`: an `mcp.json` holding a config error (`autoEnableCodemode` not a boolean), a server
     that fails (`down`, `127.0.0.1:1`), and a slow stdio server with an exposure set (`slow`, 1500 ms against a 200 ms
     start-up wait). The first prompt delivers four notices: `warning` "mcp: config error", `warning` "mcp: exposure and
     toolExposure apply …" (`slow`), `warning` "mcp: server needs attention" (`down: failed: …`) and `info` "mcp: server
     still connecting …" (`slow`); a second prompt adds none.
   * `TestNoticesWhenOffered/prompt_file`: a config directory whose `prompts/help.md` a built-in shadows; `session/new`
     delivers one `warning` "prompt: a file was not used", its description the file and `name "/help" collision …`.
   * `TestNoNoticeWithoutTheCapability`: both setups, the client offering nothing: no notice, the agent text only the
     reply, and all five log records as before.
   * `TestNoticesAreNotReplayed`: after a prompt that sent notices, a `session/load` on a second connection offering
     notices sends none, and the session file holds no `"notice"`.
   * **Against the code before step 2:** `notices_test.go:124: 0 notices, want 4`, `:150: notices [], want one warning
     …`, `:196: no notice was sent live`; `TestNoNoticeWithoutTheCapability` passed, as today's code sends nothing.
2. **The code** (step 2):
   * `acpserver/notice.go`: `offers(sessionID, update)` (D7), `conn.Negotiated()` and `Negotiation.AllowsValue(
     acp.ClientMethodSessionUpdate, …)`, false before initialize; `queueNotice(s, severity, title, description)`, which
     keeps `acp.UpdateNotice` with its description on the session only when `offers` says yes; `sendNotices(out, s)`.
   * **Delivery:** `note` (`acpserver/sysprompt.go`) runs from `sections`, `skills` and `templates`, which have neither a
     context nor the connection, and `contextcheck` refuses a fresh `context.Background()` there (P5 of
     0046-PLAN met the same rule). So the diagnostics are kept on the session (`liveSession.notices`, under the agent's
     `mu`) and sent through the turn's or the start's own `updates`: from `startFrames` (`acpserver/commands.go`), so a
     diagnostic found when a session starts goes out with its start frames, and from `Prompt` before the agent runs
     (`acpserver/agent.go`), so the MCP diagnostics go out in the prompt that found them. A diagnostic found later in a
     turn goes out with the next prompt. The once-per-session rules are the existing ones (`l.reported`, `first`,
     `s.noted`).
   * `mcpTools` takes the session (`acpserver/agent.go:675`); its four diagnostics and `note`'s keep their log records,
     each beside its `queueNotice`.
3. **Seen failing** (step 3), each break on a scratch copy, the proxy variables at a closed port; the control passed:
   * **the gate inverted:** the three offered cases see no notice (`:124`, `:150`, `:196`), and the client that offered
     nothing has its prompt fail: `{"code":-32603,…,"data":{"error":"session/update: capability session.notices not
     advertised: variant notice"}}`, the SDK's refusal of an ungated update (the MADR's G1);
   * **still connecting sent as a warning:** `:134: no info notice "mcp: server still connecting …"`;
   * **the once-per-session rule removed:** `:141: 5 notices after a second prompt, want still 4`;
   * **no notices sent from `Prompt`:** `:124: 0 notices, want 4`;
   * **no notices sent from the start frames:** `:150: notices [], want one warning`;
   * **a notice sent on load:** `:206: session/load sent notices […], want none`.
4. **Gates** (step 4): `make pre-add-check` on the six Go files, "6 file(s) clean (gofmt, golint, govulncheck)"; `make
   lint` 0 issues; `make preflight` "preflight passed", its 15 gates; `make race` 39 ok; `go test ./...` 38 ok;
   `acpserver/testdata/session.golden` byte-identical (`git diff --quiet`, C4). The Linux machine and the Windows laptop,
   the tree shipped (900 files, no `._*`): `go test` and race 38 ok each, the three notice tests `-count=5` ok on each,
   `make check-records markdownlint` 0 issues, preflight stopped at `verify-build-metadata` as on every copy.

### P3a record (2026-11-08)

The base is `feee808` (P2's commit), the tree clean (C1). No deviation.

1. **The type** (step 1): `acpext.CompactionMeta{Reason CompactionReason, TokensBefore int64, Paused CompactionPause}`,
   each `omitzero` (`acpext/types.go`), reusing `v0.1.0`'s `CompactionReason` and `CompactionPause`; its doc says the
   protocol replaces an update's `_meta` whole, so each update carries all of it. `acpext/doc.go` gains "Compaction on
   the standard updates": a client that offered `session.compaction` gets `compaction_update` with this type under
   `_meta.gobble`, read with `acp.DecodeMeta[acpext.CompactionMeta](meta, acpext.MetaKey)`, and no `MethodCompaction`;
   any other gets `MethodCompaction` as before. No `v0.1.0` name changed.
2. **Tests** (step 2): three goldens, `acpext/testdata/compaction.meta.start.json` (`reason` `manual`), `.end.json`
   (`threshold`, `tokensBefore` 110101) and `.paused.json` (`threshold`, `paused` `turn`), each an entry in
   `roundtrip_test.go`'s `frames`, so `TestFramesRoundTrip` decodes it with unknown members refused under
   `encoding/json` and `encoding/json/v2` and encodes it back canonically equal, and `TestFramesCoverEveryMethod` counts
   it. **`apidiff`**, run with the module's tool on export data of `acpext/v0.1.0` and of the change: "Compatible
   changes: - CompactionMeta: added", and nothing incompatible.
3. **Seen failing** (step 3), each on a scratch copy of the module, the control passing:
   * a misspelt member (`tokensBefor`): `roundtrip_test.go:125: compaction.meta.end.json, encoding/json: decode: json:
     unknown field "tokensBefor"`, and under v2 `unknown object member name "tokensBefor"`;
   * `tokensBefore` as a string: `cannot unmarshal string into Go struct field CompactionMeta.tokensBefore of type
     int64`, and under v2 `cannot unmarshal JSON string into Go int64 within "/tokensBefore"`;
   * the code before P3a: `undefined: acpext.CompactionMeta` at the three `frames` entries.
4. **Gates** (step 4): `make pre-add-check` on `acpext/types.go`, `doc.go` and `roundtrip_test.go`, "3 file(s) clean";
   `make check-acpext-deps` "ok" (the standard library only); `make lint` 0 issues; `make preflight` "preflight
   passed", its 15 gates; `make race` 39 ok. The Linux machine and the Windows laptop, the tree shipped (903 files, no
   `._*`): `go test` and race 38 ok each, acpext's own tests ok, `make check-records markdownlint` 0 issues, preflight
   stopped at `verify-build-metadata` as on every copy.
5. **Not done here, the owner's:** the annotated tag `acpext/v0.2.0` on P3a's commit, in `acpext/v0.1.0`'s form (message
   `acpext/v0.2.0`), and its push, under 0043-MADR's ruleset for `acpext/*` tags; then the module proxy serving it,
   which is P3c's gate. The disclosure guard runs over the outgoing commits before the push.
6. **Committed, tagged and pushed** by the owner: P3a is `253e366`. Before the push, the disclosure guard over the six
   outgoing commits (`df5c21e` to `253e366`) exited 0, and a scan of their diff for the identifiers the repository
   forbids found nothing. The annotated tag `acpext/v0.2.0` (tag object `cb6ab1f`, message `acpext/v0.2.0`) is on
   `253e366`, locally and on `origin`; `origin/main` is `253e366`. **The module proxy** served it from an empty module
   cache: `go list -m -json github.com/maccavelli/gobble-cli/acpext@v0.2.0` gave `v0.2.0`, `Time`
   2026-11-08T18:44:37Z, `Origin` `253e366f…` at `refs/tags/acpext/v0.2.0`. acpext at `253e366`, from `git archive`:
   `go mod tidy -diff` clean, `go vet` and `go test` ok, standard library only, 47 files. **CI** on `253e366` (run
   `42511886550`): success, `validate` on macos-15, ubuntu-24.04 and windows-2025 and every `build / identity` job.
   * **A correction:** during that check a run of `make markdownlint` reported MD010 in
     `docs/reports/0047-REPORT-selfupdate-adoption.md:283`. The committed file holds no tab (`git show HEAD:… | grep -c
     $'\t'` gives 0; it is 276 lines, and the linted copy was 292); the failure came from another session's uncommitted
     edit to that file in the working tree, made between the status check and the lint. The pushed commit was clean, as
     the Docs workflow's success on `253e366` agrees.

### P3b record (2026-11-08)

The base is `253e366` (P3a's commit, tagged `acpext/v0.2.0`), the tree clean (C1). No deviation.

1. **The tests first** (step 1), in `session/tree_test.go`:
   * `TestReserveIsHonoured`: an entry `Next` stamps with a reserved id keeps it, its parent the leaf, and becomes the
     leaf when added; the spent reservation is not honoured again; an id nobody reserved is replaced, as before.
   * `TestReserveIsNotReissued`: a reserved id is taken (`inUse`, the predicate `newID` asks), so no new id can be it,
     and `Reserve` does not give it twice. The check is the predicate, since entry ids are random and a loop could not
     show it.
   * `TestReleaseFreesTheId`: a reservation writes no entry; a released id is no longer taken and is not honoured.
   * **Against the code before step 2:** `tr.Reserve undefined`, `tr.inUse undefined`, `tr.Release undefined`.
2. **The code** (step 2), `session/tree.go`: `Tree.reserved`, a set under the tree's mutex; `Reserve()` takes a
   `newID` and holds it; `Release(id)` drops it; `Next` keeps an entry's id when it is a live reservation, and spends it,
   and otherwise gives a new id as before; `newID` asks `inUse`, an entry's id or a reservation. No session-file change;
   `session` is a stable package (`session/doc.go`), and both methods are additions.
3. **Seen failing** (step 3), each break on a scratch copy, the control passing:
   * `Next` overwriting a reserved id: `tree_test.go:259: Next kept id "…", parent …; want "…" under "…"`;
   * `newID` not skipping reservations: `tree_test.go:285: the reserved id "…" is not taken`;
   * `Next` not spending the reservation: `tree_test.go:268: the spent reservation "…" was honoured twice`;
   * `Release` doing nothing: `tree_test.go:305: the released id "…" is still taken`.
4. **Gates** (step 4): `make pre-add-check` on the two files, "2 file(s) clean (gofmt, golint, govulncheck)"; `make
   lint` 0 issues; `make preflight` "preflight passed", its 15 gates; `make race` 39 ok; `session` and `acpserver`
   tests ok. The Linux machine and the Windows laptop, the tree shipped (904 files, no `._*`): `go test` and race 38 ok
   each, `make check-records markdownlint` 0 issues, preflight stopped at `verify-build-metadata` as on every copy.

### P3c record (2026-11-08)

The base is `f555311`, P3b's commit. `origin/main` is at `253e366`, so P3b is committed but not yet pushed. The tree
was clean (C1). No deviation.

The gate:

* P3b is committed.
* The module proxy serves `acpext/v0.2.0` (P3a record).
* The owner gave their word on 2026-11-08: "committed, proceed", in reply to the request to say "proceed with P3c" when
  ready, a request that named this last condition.
* magic-cli-remote's 0179 is unchanged at `254f996b` since `c87d22e4`. Amending it is the owner's job, with step 5's
  note.

1. **The tests first** (step 1), in `acpserver/compactupdate_test.go`. Each client offers `session.compaction`, or
   `session.compaction` and `session.notices`.
   * **`TestCompactionUpdateManual`:**
     * `/compact` on an empty session sends no update, only its one-line reply.
     * After two prompts, `/compact` sends `in_progress` with `{reason: manual}`, then `completed` with
       `{manual, tokensBefore}` and the summary in its `summary` field. Both share one id: the compaction entry's id
       (`_gobble/get_entries`). The summary is the entry's. `usage_update` comes last. There is no agent text and no
       `_gobble/compaction`.
     * A second `/compact` gives the "Already compacted" reply and no update.
     * `session/close` then `session/load` replays one `completed` update with the live id, the entry's summary and
       `{tokensBefore}` alone, and no "Compacted from" text.
     * `ValidateGolden` passes on every frame of the connection.
   * **`TestCompactionUpdateManualFailed`:** a summary that fails offline sends `in_progress`, then `failed` with
     "Summarization failed: llm: offline" and `{manual}`. There is no agent text and no entry.
   * **`TestCompactionUpdateExtension`:**
     * `_gobble/compact` on an empty session returns -32600 "Nothing to compact (session too small)" and sends no
       update.
     * After two prompts it sends the same pair. The result's `summary` and `tokensBefore` match the update's.
       `usage_update` comes last.
   * **`TestCompactionUpdateAutomatic`:** at the threshold and on overflow, the pair carries its reason, and no notice
     is sent.
   * **`TestCompactionUpdatePaused`:** a summary that fails offline at the threshold sends `failed` with
     `{threshold, paused: turn}`. With `session.notices` offered, one `warning` notice follows, titled "compaction:
     automatic compaction paused", its description carrying the error. Without `session.notices`, none.
   * **`TestCompactionUpdateCancelled`:** a provider in front of the script holds the summary request until the cancel.
     * `/compact` cancelled before the write: `cancelled`, no entry, and the turn ends `cancelled`.
     * `/compact` cancelled after the write: `completed` with the entry's id and summary, then `usage_update`. The turn
       still ends `cancelled`.
     * `_gobble/compact` cancelled after the write: it returns its result, and the row is `completed`.
     * Automatic, cancelled before the write: `cancelled` with `{threshold}`.
   * **Not offered:** the existing tests pass unchanged: `TestCompactShortensContext`, `TestCompactionLoads`,
     `TestExtensionCompact`, `TestManualCompactionNotice`, the `TestAutoCompact*` tests, `TestNotifyFramesUnchanged`
     and `TestAcpextNotificationsAreSent`.
   * **Test helpers:**
     * The compaction harness gained `harnessConfig` (the client's capabilities, and a provider in front of the
       script). `newCompactHarness` passes the zero config, so its initialize request is unchanged.
     * `session_test.go`'s two `replayUpdates` calls pass `rows` false.
   * **Against the code before step 2:** the run was in the working tree with acpext `v0.2.0` required. Every new test
     failed, each with "0 compaction_updates [], want 2". `_gobble/compact` written then cancelled answered
     `{"code":-32603,…,"data":{"error":"Compaction cancelled"}}`.
2. **The code** (step 2):
   * **The module:** `go.mod` moves acpext from `v0.1.0` to `v0.2.0`, `go.sum` follows, and `go mod tidy` is clean.
   * **`acpserver/compactreport.go`:**
     * The interface `compactionReport` has three methods: `rows`, `started(id)`, and `ended(res, err, cancelled,
       paused) (shown bool)`.
     * `newCompactionReport` chooses one implementation per compaction, by `offersCompaction` (D7's gate), before
       anything is sent.
     * `compactionNotices` holds today's two literal `notify(acpext.MethodCompaction, …)` calls. A refusal is still a
       start and an end.
     * `compactionUpdates` sends `in_progress`, then one terminal update:
       * `completed`, whenever the entry was written: the summary as one text block by `acp.PatchValue`;
       * `cancelled`;
       * `failed`: the error's first line, by the existing `firstLine`.

       Each update carries its whole `_meta.gobble`, an `acpext.CompactionMeta` encoded by `jsonx.SetMember`. A pause
       also goes through D8's `queueNotice` and `sendNotices`.
   * **`compact.go`:**
     * `runCompaction` reserves the entry's id (`Tree.Reserve`) before asking for the summary, and reports the start
       with it. It writes the entry with that id, and releases it with a `defer`, which does nothing once the
       reservation is spent.
     * `compactSession` takes the report.
     * `manualEnd` is removed. Its cases are now `compactionNotices.ended`.
     * `cmdCompact` says the start text only when the report is not `rows`. After a run the client was shown:
       * written: it sends `usage_update` and returns `context.Cause`, which is nil without a cancel;
       * failed: it sends nothing more.

       Refusals and the notices channel are unchanged.
   * **`extensions.go`:** `extCompact` returns its result for a written run the client was shown, even after a cancel.
   * **`autocompact.go`:** the report is chosen after the flush. It reports a cancel, a failure with its pause, and a
     success.
   * **`history.go`:**
     * `replayUpdates` and `replayHistory` take `rows`.
     * `replayCompaction` is a compaction as `completed`, with the entry's id, its summary and `{tokensBefore}`.
     * `LoadSession` (`agent.go`) passes `offersCompaction`.
   * **The frames:**
     * The manual paths now prepare before the notices channel's start, which `runCompaction` sends. Prepare sends
       nothing, so the frames are the same.
     * `git diff --quiet` passes on every golden and testdata file, and `TestNotifyFramesUnchanged` passes (C4).
3. **Seen failing** (step 3), each break planted on a scratch copy. Each run used `HTTPS_PROXY`/`HTTP_PROXY` set to
   `127.0.0.1:9`, and `HOME` and `XDG_*` in the scratch directory (C3). The control passed.
   * **Both reporters at once:**
     `compactupdate_test.go:195: _gobble/compaction sent to a client that offered session.compaction: [map[phase:start reason:manual]]`,
     and the same in every new test.
   * **The reporter chosen without the gate:** each existing compaction test, including `TestCompactShortensContext`,
     `TestExtensionCompact` and the `TestAutoCompact*` tests, failed with
     `{"code":-32603,…,"data":{"error":"session/update: capability session.compaction not advertised: variant compaction_update"}}`.
   * **The summary as a chunk:**
     * `a compaction_summary_chunk was sent; the summary belongs in the completed update`;
     * `compaction_update 1, completed: summary []`.
   * **`_meta` without `reason`:**
     `compaction_update 1: completed "" {Reason: TokensBefore:10 Paused:}, want completed "" {Reason:manual …}`, and
     the same on `failed` and `cancelled`.
   * **A fresh id instead of the reservation:**
     `completed "1f948ebe" with summary "SUMMARY"; the entry is "0208dd02" with summary "SUMMARY"`.
   * **A written-then-cancelled run reported `cancelled`:**
     `compaction_update 1: cancelled "" {Reason:manual …}, want completed …`, for both `/compact` and
     `_gobble/compact`.
   * **A pause without its notice:** `compactupdate_test.go:329: 0 notices [], want 1`.
   * **Replay ungated:** `TestCompactionLoads`, whose client offers nothing, failed with
     `compact_test.go:128: {"code":-32603,…,"data":{"error":"session/update: capability session.compaction not advertised: variant compaction_update"}}`.
4. **Gates** (step 4):
   * `make pre-add-check` on the nine Go files: "9 file(s) clean (gofmt, golint, govulncheck)".
   * `make lint`: 0 issues in both modules.
   * `make preflight`: "preflight passed".
   * `make race`: 39 ok.
   * The new tests and the existing compaction tests: 30 runs ok.
   * The identifier scan of the diff and the new files found nothing.
   * **The full suite and C3:** the full suite, run with C3's environment, fails two packages P3c does not touch:
     * `llm/provider` `TestWireOffline`: the probe dialed the proxy, `127.0.0.1:9`;
     * `tool/builtin` `TestBash*`: the host's bash startup file resolves through `HOME`.

     Both pass in the normal environment, where the full suite and race run. C3's environment is for the scratch runs,
     which touch only `acpserver`.
   * The Linux machine and the Windows laptop, with the tree shipped (906 files, no `._*`):
     * `go test` and race: 38 ok each.
     * `make check-records markdownlint`: 0 issues.
     * preflight stopped at `verify-build-metadata`, as it does on every copy, since the copy has no commit.
5. **For the owner, not posted** (step 5): a note for magic-cli-remote, kept outside the tree. It lists the 0179 lines to
   amend:
   * F7 G3, lines 297–298: the compaction half is closed.
   * D10, lines 515–516: `_gobble/compaction` is no longer sent to Magic. The note recommends removing the handler,
     which needs a stated minimum gobble release, or else keeping it only for releases before P3c.
   * D11, line 524: `notify.go`.
   * The live-run list, line 558: the fixtures should record the standard updates.

   Decoding `CompactionMeta` is offered as optional.

### Deviation 2 (2026-11-08): P4, line mode's `/compact` would print nothing

* **Found** at P4's start (`HEAD` `3d6fb65`, the tree clean), before any code:
  * P3c sends a client that offers `session.compaction` only the compaction's rows for `/compact` (MADR P3-Q4,
    `acpserver/compact.go`). That client gets no "Compacting the conversation…" text, no "Compacted from N tokens."
    text and no summary as agent text.
  * P4 makes line mode offer `session.compaction`.
  * The line view's `compactionLine` (`internal/cli/view.go`) draws nothing for a manual compaction, "since /compact's
    reply says so".
  * P4's step 1 named only the automatic lines. MADR D10 (`:237`) and its open question (`:383`) already said that line
    mode prints the progress and the summary "in the words `/compact` uses today".
  * Line mode resumes with no replay (`acpclient/sessions.go:145`), so replay needs nothing here.
* **Decision,** the owner's ("View draws it"):
  * The line view draws a manual compaction from its rows, in `/compact`'s words (`personality.CompactStart`,
    `CompactDone`, `CompactFailed`) and the session's voice, through the same reply path as agent text. A test checks
    that its stdout matches what `/compact`'s agent text drew.
  * One difference: a failure shows the error's first line, which is all the row carries.
  * Not chosen: having the agent send `/compact`'s text to clients that offer `session.compaction` too (this would
    revisit P3-Q4, and Magic would show both); D10 Option 3 (this would reverse Q5).
* **Files added to scope:**
  * `acpclient/acptest/script.go`: `ScriptAgent` records each `initialize`'s client capabilities. This is one new
    method on a stable package, a compatible addition (`apidiff` itself checks nothing below `v1.0.0`).
  * `internal/cli/view.go`'s manual drawing, with its test.

  The MADR needs no amendment: D10 already states this behaviour.

### P4 record (2026-11-08)

The base is `3d6fb65`, P3c's commit. `origin/main` is at `253e366`, with P3b and P3c not yet pushed. The tree was clean
(C1). The gate: P3 is committed. One deviation, 2 above, was found before any code.

1. **The tests first** (step 1, with deviation 2):
   * **`acpclient/standardcompaction_test.go`, `TestTranslateStandardCompaction`:** with `Options.Compaction`, the
     scripted agent sees `session.compaction` and no `session.notices`. The agent sends three compactions:
     * a manual one, whose summary arrives in two chunks;
     * an automatic one that fails and pauses;
     * one that is cancelled.

     They reach the callback as eight `KindCompaction` updates, each carrying the ID, the status, the summary so far,
     and `_meta.gobble`'s reason, tokens and pause. An `in_progress` update is a `start`; a terminal update is an
     `end`, a cancelled one with the error "cancelled"; a chunk is neither. Without the option, the scripted agent
     sees no session capabilities.
   * **`acpclient/compaction_test.go`, `TestStandardCompactionNotOffered`:** without a timeline, both variants stay
     `KindOther`, by their variant.
   * **`internal/cli/compactionview_test.go`:**
     * `TestLineCompactionRows`: line mode offers `session.compaction` alone. A manual compaction's rows draw the same
       stdout as `/compact`'s agent text, for a success and for a failure. An automatic compaction's rows draw today's
       stderr lines: "compacting the conversation (threshold)", and the failure with "(automatic compaction paused for
       this turn)". A manual compaction draws nothing on stderr.
     * `TestLineCompactThroughTheAgent`: `/compact` in line mode, against gobble's own agent, draws its progress, its
       count and its summary once each, in the session's voice.
     * `TestPrintOffersNoCompaction`: print mode's scripted agent sees no session capabilities.
   * **Not offered:** `print-json.golden`, `print-stream-json.golden` and every other golden are unchanged
     (`git diff --quiet`).
   * **Against the code before step 2:**
     * Both new test files fail to compile:
       * `agent.ClientCapabilities undefined`;
       * `unknown field Compaction in struct literal of type acpclient.Options`.
     * With the `ScriptAgent` recorder alone added (a scratch copy with HEAD's `acpclient` and `internal/cli` code),
       `TestLineCompactionRows` failed. Each turn timed out with "condition not met within 5 s", since the scripted
       agent's SDK refused rows the client had not offered, and the test ended with
       `client capabilities [{… Session:<nil> …}], want session.compaction alone`.
2. **The code** (step 2):
   * **`acpclient/acptest/script.go`:** `ScriptAgent.ClientCapabilities()`, recorded at `initialize` (deviation 2).
   * **`acpclient/conn.go`:**
     * `Options.Compaction` makes `initialize` send `ClientCapabilities.Session.Compaction`, and nothing else.
     * The handler keeps one `acp.CompactionTimeline` per session, created on first use and dropped with the session,
       and only when the option is set.
   * **`acpclient/update.go`:**
     * `translate` takes the session's timeline. With it, a `compaction_update` or `compaction_summary_chunk` is
       applied and becomes `KindCompaction`.
     * `compactionFrom` reads the entity: the status, the summary's text, the error, and `acpext.CompactionMeta` by
       `acp.DecodeMeta`.
     * `Compaction` gains `ID`, `Status` and `Summary`. These are additions, and the struct stays comparable.
   * **`internal/cli`:**
     * `startAgentWith` takes `compaction`. Line mode passes true; print mode and `models_test.go` pass false.
     * `view.go`'s agent-text drawing moves into `reply`, unchanged.
     * `manualCompaction` gives `/compact`'s words for a manual row, by `personality.CompactStart`, `CompactDone` and
       `CompactFailed`, in the view's voice, and draws them through `reply`.
     * `compactionLine` and `ambient.note` are unchanged, and read the folded rows as they read `_gobble/compaction`.
   * Notices are not offered (MADR D10).
3. **Seen failing** (step 3), each break planted on a scratch copy with C3's environment. The control passed.
   * **The option not reaching `initialize`:**
     * `standardcompaction_test.go:39: session/prompt: {"code":-32603,…,"data":{"error":"session/update: capability session.compaction not advertised: variant compaction_update"}}`;
     * in `internal/cli`, `TestLineCompactionRows` timed out on each turn, then reported
       `client capabilities [{… Session:<nil> …}]`.
   * **A chunk dropped from the fold:**
     * `standardcompaction_test.go:61: update 1: other {…}, want compaction {… Summary:## Goal …}`;
     * `update 3: compaction {Phase:end … Summary:}, want … Summary:## Goal …`.
   * **Print mode offering it:** `compactionview_test.go:119: print mode's client capabilities [{… Session:0x… …}], want none`.
   * **The view drawing nothing for a manual row:**
     * `compactionview_test.go:66: "SUMMARY-ONE" not drawn after "Compacting the conversation…"`;
     * `TestLineCompactThroughTheAgent`: `condition not met within 5 s`, then `line mode's /compact drew …` without the
       summary.
4. **Gates** (step 4):
   * `make pre-add-check` on the eleven Go files: "11 file(s) clean (gofmt, golint, govulncheck)". `view.go` was checked
     again after the lint fix.
   * `make lint`: the first run found one `goconst` finding, a third `"start"` literal in `view.go`. The new line now
     uses `acpext.PhaseStart`, and lint finds 0 issues in both modules.
   * `make preflight`: "preflight passed". Its `apidiff` step reports "no-op below v1.0.0".
   * `make race`: 39 ok.
   * `make perf`: within budget:
     * ReplyRender 21.9 µs (baseline 30 µs);
     * Reprint 5.86 ms (9.4 ms);
     * LineSessionStart 984 µs (830 µs);
     * KeyDraw 1.52 µs (1.5 µs).
   * The new tests and their neighbours: 20 runs ok.
   * The identifier scan of the diff and the new files found nothing.
   * The Linux machine and the Windows laptop, with the tree shipped (908 files, no `._*`):
     * `go test` and race: 38 ok each.
     * `make check-records markdownlint`: 0 issues.
     * Preflight was not run there this phase. On every copy it stops at `verify-build-metadata`, and C5 asks the hosts
       for the tests and race.

### P5 record (2026-11-08)

The base is `315cbe4`, P4's commit. `origin/main` is at `253e366`, with P3b, P3c and P4 not yet pushed. The tree was
clean (C1). The gate: P4 is committed. No deviation.

1. **The goldens** (step 1):
   * `git diff --quiet a233e3f HEAD -- acpserver/testdata/session.golden acpclient/acptest/testdata/initialize.golden`:
     exit 0, byte-identical to P1's commit, their one rewrite.
   * `git diff --quiet 4970cca HEAD -- internal/cli/testdata`: exit 0, all twelve files byte-identical to the base.
   * `TestSession` and `TestPairInitialize`, which compare those goldens and check them with `ValidateGolden`: PASS.
     `go test ./internal/cli/`: ok.
2. **The scratch run** (step 2), on a copy of the tree with C3's environment, never in the working tree:
   * **Set-up:** `gobble acp` built from the tree, with `GOBBLE_HOME` in the scratch directory and no model credential
     in its environment. A `models.json` points `huggingface` at a local fake Chat Completions server, with a
     placeholder key. `gobble config set` makes `huggingface` the default and `compaction.keepRecentTokens` 1. An
     `mcp.json` names a server, `down`, at a closed loopback port.
   * **The client:** the SDK's `ClientSideConnection` over the child's stdio, every frame recorded by the SDK's
     `acptest.Recorder`, offering `session.notices` and `session.compaction`. It sent `initialize`, `session/new`, the
     prompts "hello", "and again" and "/compact" (each `end_turn`), then `session/close`, and closed stdin; the agent
     exited 0 with nothing on stderr.
   * **Received:**

     ```text
     P5 notice: warning "mcp: server needs attention"
     P5 compaction_update: id 45dbdddb status in_progress meta {"gobble":{"reason":"manual"}}
     P5 compaction_update: id 45dbdddb status completed meta {"gobble":{"reason":"manual","tokensBefore":160}}
     P5 frames 24; notices 1; compaction_update 2; summary chunks 0; agent text chunks 2; /compact text or _gobble/compaction 0
     P5 ValidateGolden: passed
     ```

     The `completed` frame's `summary` field holds the fake model's summary, with the split turn's context after it.
     The two agent text chunks are the two replies. The model server saw `GET /models` and four `POST
     /chat/completions`, all on loopback.
   * **Seen failing** (C2), each break on its own copy; the unbroken run above is the control:
     * **The agent never offering compaction rows** (`offersCompaction` returns false): `compaction_update 0; … /compact
       text or _gobble/compaction 4`, then `p5scratch_test.go:165: want notices, the compaction's rows, and no legacy
       channel`.
     * **No notice queued at all:** `notices 0`, then the same failure at `:165`.
     * **A recorded `completed` frame corrupted before validation** (`"jsonrpc":"1.0"`): `p5scratch_test.go:172:
       acptest: record 19, received: "jsonrpc" is not "2.0": {…"sessionUpdate":"compaction_update","status":"completed"…}`.
     * **Two breaks the run did not fail on, recorded as findings, not gaps in gobble:**
       * A `completed` row with the status `"done"` passed `ValidateGolden`. In acp-go-sdk2 `v1.34.2`,
         `SessionCompactionUpdate.validate` checks only the summary's blocks: the status is an open set. gobble's own
         tests (`TestCompactionUpdate*`) hold its four statuses.
       * Notices queued without the gate still passed, as expected: this client offered notices. Without the
         capability, `TestNoNoticeWithoutTheCapability` (P2) holds it.
     * **A break the SDK stops before the wire:** a summary block with `annotations.audience` `"robot"` failed the
       `/compact` prompt with `… audience[0]: \"robot\" is not a Role`, the SDK's check on send.
3. **Records** (step 3):
   * `0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md`: a "Completion (2026-11-08)" section. Each item gets its
     status, its commit and its evidence, plus the acceptance run and the two SDK findings. The refresh's status table
     is left as read at `4970cca`.
   * `docs/architecture.md`: the `acpserver` row (one channel per compaction, the notices, the gate) and the
     `acpclient` row (`Options.Compaction`, the fold, line mode and print mode).
   * This plan: `complete`. `docs/README.md`: the MADR's and this plan's rows.
   * The note for magic-cli-remote's `docs/decisions/0179-MADR-gobble-native-acp-provider.md` D10 was re-read against
     magic-cli-remote `31e21118`: 0179 is unchanged since `c87d22e4`, and the cited lines still hold. It is handed to
     the owner, not posted, and not committed.
4. **Gates:**
   * No Go file changed, so there was no pre-add check or lint run of new code.
   * `go test ./...`: 38 ok. `make race`: 39 ok.
   * `make preflight`: "preflight passed"; its `markdownlint` linted 58 files and found 0 issues.
   * The identifier scan of the diff found nothing. It looked for the local account name, the hostname domain, the
     hosts' names and real-machine paths.
   * The Linux machine and the Windows laptop were not run this phase. The code is P4's, which both passed at P4 with
     the tree shipped, and P5 changes only Markdown. Preflight's `check-records` checks the records' names and links,
     and its `markdownlint` checks `docs/architecture.md` and `docs/README.md`; its globs skip MADR and PLAN files.

### Correction (2026-11-09): `docs/architecture.md`'s acpext row

P3c (`3d6fb65`) moved `go.mod` from acpext `v0.1.0` to `v0.2.0` (P3c record, step 2) and left
`docs/architecture.md`'s table of direct requirements at `v0.1.0`. Found while 0048-PLAN P4 edited the same table; the
row was put right on the owner's word ("fix the acpext version in architecture.md"). `go.mod:20` requires
`github.com/maccavelli/gobble-cli/acpext v0.2.0`; no other document outside the records names acpext's version.
