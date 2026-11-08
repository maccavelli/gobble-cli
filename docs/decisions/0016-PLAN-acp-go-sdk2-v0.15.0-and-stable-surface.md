---
status: proposed
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
* P3, compaction (D9): `acpserver/compact.go`, `extensions.go` (`_gobble/compact`), `autocompact.go`, `history.go`
  (replay), and their tests.
* P4, gobble's client (D10): `acpclient/conn.go`, `acpclient/update.go`, `internal/cli` (line mode's options and the
  view), and their tests.
* P5, acceptance: the goldens, the docs (`docs/architecture.md`'s `acpserver` and `acpclient` rows), the records.

**Out of scope:**

* D13's surfaces: `session/delete`, elicitation, images and audio, boolean config options, `session/list` pagination.
* Any change to `acpext`: `_gobble/compaction` and `CompactionNotice` stay as published at `v0.1.0`.
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

### P3 — compaction through the standard updates (D9, Option 1)

**Gate:** P2 committed, and the owner's word that the release Magic captures from may carry it (MADR R11, R12).

1. **The tests first,** for each path (`/compact`, `_gobble/compact`, automatic at the threshold, on overflow), with a
   client that offers `session.compaction` and one that does not:
   * offered: `compaction_update` `in_progress`, one `compaction_summary_chunk` with the summary, then `completed`
     (with `_meta.gobble` `reason` and `tokensBefore`), or `failed` with the error's first line (and `paused` on an
     automatic failure), or `cancelled`; one id across the run; **no `_gobble/compaction`**, and for `/compact` none
     of its two agent-text lines;
   * not offered: exactly today's frames, `_gobble/compaction` and agent text, byte for byte;
   * a refusal ("Nothing to compact", "Already compacted") sends no compaction update in either case;
   * replay: a `session/load` sends, for an offering client, one `compaction_update` `completed` with the entry's
     summary and the entry's id; for any other, today's "Compacted from N tokens:" text.
2. **The code:** one reporter the four paths share, chosen per connection by `offers`:
   * the standard reporter sends `acp.UpdateCompaction` / `UpdateCompactionSummaryText` with an id from
     `Options.NewID`, and `_meta.gobble` built with the SDK's `MetaObject`;
   * the extension reporter is today's `_gobble/compaction` notices and text, unchanged;
   * `replayMessage` (`acpserver/history.go:359-360`) chooses the same way.
3. **Seen failing** (C2): each test against the code before step 2; then the planted breaks: both reporters at once (a
   duplicate `_gobble/compaction`); no terminal status on failure; a fresh id per update; replay ungated (the SDK refuses
   it, `*CapabilityError`).
4. Gates (C5). `session.golden` is byte-identical (C4). **Commit.**

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
2. **The code:** `acpclient.Options.Compaction`; `initialize` sets `ClientCapabilities.Session.Compaction`; `translate`
   keeps one `CompactionTimeline` per session; line mode sets the option. Notices are not offered (MADR D10).
3. **Seen failing** (C2): each test against the code before step 2; then the planted breaks: the option not reaching
   `initialize`; a chunk dropped from the fold; print mode offering it.
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
| D9: one channel per connection; replay gated; terminal status | P3 step 1 |
| D10: the line client offers compaction; print mode unchanged | P4 step 1 |
| D11: the dated notes | P0 step 2 |
| The wire without the capabilities | C4 in every phase; P5 step 1 |

## Rollout and Rollback

* **Rollout.** Each phase is one commit on `main`. P3 changes what Magic receives from gobble, so it lands only with
  the owner's word (its gate), ahead of or with the gobble release Magic captures from.
* **Rollback.** P1, P2 and P4 each revert alone. P3 reverts alone, restoring `_gobble/compaction` and text for every
  client; P4 reverts first if P3 is reverted.

## Execution record

Not started. The plan is `proposed` until the owner approves it.

### Answers (2026-11-08)

* **Q1 to Q7 answered** by the owner (the MADR's table): D6 Option 2, "Adopt SDK recorder"; D7, D8, D9, D10, D11 and
  D13 Option 1. P1 is rewritten for Option 2 above, its first steps struck through; the other phases stand as written.
* **Found while rewriting P1:** gobble's `Recorder.Golden` normalizes host paths (`hostIndependent`, 0019-MADR D4),
  which the SDK's `acptest.Golden` does not. P1 keeps that normalizer, over the SDK's records. The MADR records it.
* **Files added to scope:** P1's `acpclient/acptest/recorder.go`, `pair.go` and `pair_test.go`, and the two ACP
  goldens, rewritten once.
