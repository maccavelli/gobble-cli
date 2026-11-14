---
status: proposed
date: 2026-11-14
associated-madr: "0057-MADR-cache-visibility-and-one-hour-pricing.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement showing the prompt cache's reads and writes, counting one-hour cache writes, and marking their cost approximate

Associated MADR: [0057-MADR-cache-visibility-and-one-hour-pricing.md](0057-MADR-cache-visibility-and-one-hour-pricing.md)
(accepted 2026-11-14). ~~The MADR's Q1 to Q3 are open. The steps below are written to the recommended answers, Q1 (a),
Q2 (a) and Q3 (a). Any other answer rewrites the phases it touches before approval.~~

*(2026-11-14, the owner's rulings: Q1 "--stats, plus /usage and /session split", (a); Q2 "Same in every mode", (a); Q3
"Catalog rate, flagged approximate", (b). This plan was rewritten to them before any approval of its execution. The
draft's P2 priced one-hour writes at two times the input, and its P0 checked that rate against Anthropic's price page.
Both are gone. P2 now pins the catalog's one write rate and adds the approximate marker. The plan stays `proposed`,
and its code waits for `v0.7.0`.)*

## Goal

When this plan is complete:

* a one-hour cache write is counted from the SDK to the session file (MADR D3);
* every write, one-hour writes included, is priced at the catalog's one write rate. A cost that includes one-hour
  writes is marked approximate in `/usage` and `/session` (D4);
* `/usage` and `/session` show cache reads, cache writes and the input that was neither, and the writes' one-hour part
  when there is one (D2);
* the terminal shows a turn's cache reads and writes under `--stats`, in every view mode, from the prompt response it
  already receives (D1, D5);
* the ACP stream is unchanged, `usage_update`'s cost included.

## Prerequisite

**Nothing here starts before gobble `v0.7.0` is tagged** (the owner's ruling of 2026-11-14;
[0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md)).

**Order against the other plans** (MADR D6):

* [0037-PLAN-resume-prefix-and-failed-turn-history.md](0037-PLAN-resume-prefix-and-failed-turn-history.md) lands
  first. It is ruled "now", and it changes `acpserver/agent.go`, `acpserver/history.go` and `agent/agent.go`, which P1
  here also changes.
* [0056-PLAN-curate-the-terminal-session-view-with-view-modes.md](0056-PLAN-curate-the-terminal-session-view-with-view-modes.md)'s
  C6 requires `git diff --exit-code <base> -- acpserver agent tool acpext llm` to be empty after each of its phases.
  P1 to P3 here change `llm`, `agent` and `acpserver`, so they land before 0056-PLAN's P0 or after its P4, never
  between. **Recommended: before its P0**, since they are small and independent of the view.
* P4 here draws on 0056's `TurnEnd` and its line renderer (`internal/curate`, `internal/cli`), so it lands after
  0056-PLAN's P4.

## Scope

* **In:** the files each phase names, and nothing else. A file a phase needs and does not name is a deviation (C6).
* **Out:**
  * any new ACP field, `_meta` member or `usage_update` field (MADR D1);
  * a one-hour rate, a multiplier or a provider-keyed price rule (D4, Q3's ruling);
  * an approximate marker on the status line (D4);
  * recomputing costs already stored in session entries (D4);
  * cache warming (0021-MADR D7's other half);
  * pushing and tagging, which need the owner's ask in the turn they happen (`AGENTS.md`).

## Conventions every phase follows

* **C1. Start state.** The tree is clean apart from this plan's own records, after P0; `HEAD` is the previous phase's
  commit on `main`; the hooks directory is the global one. Each phase re-reads its files at its base and records any
  line that moved since `512db99`.
* **C2. Scratch work.** Every planted break is made in a scratch copy of the tree (`<scratch>/break-<phase>-<n>/`),
  never in the working tree, as an exact replacement asserted present once by a stdlib-only Python script. The record
  names the break and quotes the failure's first lines.
* **C3. Long output.** Every gate writes its whole output to `<scratch>/<phase>-<gate>.log`, and its exit status is
  taken before any filter.
* **C4. Heavy runs.** `make preflight`, `make race`, `make perf` and `go test ./...` run one at a time per host. Before
  each, `~/.agents/skills/available-dev-hosts/scripts/pick-host` names the host; the run goes there through
  `~/.agents/skills/available-dev-hosts/scripts/run-on-host --need <goos> -- <command>`, or through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>` when it names the session host. A WSL host
  serves `--need linux` when the Linux machine is busy. A single-package `go test` while iterating may stay on the
  session host.
* **C5. Phase gates:** `make pre-add-check FILES="<every changed .go file>"`; `make preflight` (C4); `make race` (C4);
  `go test -count=1 ./...` with `--need linux` and `--need windows` (C4). P4, which changes `internal/cli`, adds
  `make perf` (C4) and `make probe-conhost`.
* **C6. Deviations** stop the phase: evidence (file, line, exact failure, whether pre-existing), resolutions with the
  recommended first and each one's cost, and what doing nothing means. No workaround is offered. A dated entry here,
  and in the MADR when a decision or a fact changes, before work continues.
* **C7. Seen failing.** Every new or changed test fails on a planted break before it is trusted (C2).
* **C8. Commit.** One commit per phase, `git add` of exactly its files, `git commit --no-edit`, on `main`, after C5
  and on the owner's word in the turn it happens. The phase's record and this plan's `docs/README.md` row go with it.
* **C9. Records' paths.** No hostname, account name or real machine path in a committed file, quoted output included.

## Implementation Steps

### P0 — preconditions

1. `v0.7.0` is tagged; record the tag's commit. 0037-PLAN-resume-prefix-and-failed-turn-history.md is complete.
2. ~~The owner answers Q1 to Q3.~~ *(2026-11-14: done; the rulings are in the header and the MADR.)*
3. ~~Under Q3 (a): check the one-hour write rate against Anthropic's published price page.~~ *(2026-11-14: dropped by
   Q3's ruling; no rate is priced.)*
4. Re-read the pins: go-llmprovider-sdk still has `Usage.CacheWrite1hTokens` with the doc quoted in MADR F5, and
   acp-go-sdk2's `acp.Usage` still has `CachedReadTokens` and `CachedWriteTokens` and no one-hour field. If the catalog's
   snapshot has gained a one-hour rate by then, that is a fact the MADR records (C6) before P2, since D4 is written to
   its absence.
5. ~~The MADR becomes `accepted`, and the dated notes of MADR D7 are written.~~ *(2026-11-14: done. The MADR is
   `accepted`, and the notes in 0021-MADR D7 and F13 and 0056-MADR D2 are written.)*
6. The owner approves this plan's execution, whole or by phase.

### P1 — count one-hour writes (MADR D3)

**Files:** `llm/event.go`, `llm/provider/provider.go`, `llm/provider/provider_test.go`, `agent/agent.go`,
`agent/agent_test.go`, `acpserver/history.go`, `acpserver/commands.go`, `acpserver/usage_test.go`, this plan,
`docs/README.md`.

1. `llm.Usage` gains `CacheWrite1hTokens int64` with `json:"cacheWrite1hTokens,omitzero"`.
2. `usageOf` maps `u.CacheWrite1hTokens`. The all-zero test's condition is unchanged, since the field is a part of
   `CacheWriteTokens`.
3. `addUsage` sums it.
4. `piUsage` adds `cacheWrite1h` to the usage's unknown members when non-zero, beside `estimated`, with both kept when
   both are set. `totals` gains `cacheWrite1h`, and `sessionTotals` reads it.
5. **Tests:**

   | Test | What it holds | Planted break (C7) |
   | :--- | :--- | :--- |
   | `TestUsageOfOneHour` (`llm/provider`) | a response with `CacheWriteTokens` 10 and `CacheWrite1hTokens` 7 gives `llm.Usage` 10 and 7 | the mapping removed |
   | `TestAddUsageOneHour` (`agent`) | two calls' one-hour counts are summed | the sum left out |
   | `TestOneHourWritesStored` (`acpserver/usage_test.go`) | an assistant entry written with 7 one-hour writes holds `"cacheWrite1h":7` in its usage, and `sessionTotals` over the reloaded entries gives 7; with `Estimated` also set, both members are kept | `estimated`'s member written over the one-hour one |
6. **Unchanged.** A session with no one-hour writes writes byte-identical entries: the goldens and
   `session/jsonl/testdata/` are unchanged.
7. C5. Commit (C8).

### P2 — one-hour writes at the catalog's write rate, and the cost marked approximate (MADR D4; Q3 (b))

*(2026-11-14: rewritten to Q3's ruling. The draft's P2 changed `catalog.Price`'s signature and priced one-hour writes
at two times the input.)*

**Files:** `llm/catalog/catalog_test.go`, `acpserver/commands.go`, `acpserver/usage_test.go`,
`acpserver/catalog_test.go`, this plan, `docs/README.md`.

1. **The price stays.** `catalog.Price` (`llm/catalog/catalog.go:132-147`) and `pricedUsage`
   (`acpserver/history.go:124-129`) are not changed: every write, one-hour writes included, is priced at `CacheWrite`.
   A test pins that, so a later change to it is a decision, not an accident.
2. **Approximate.** `totals` gains `approximate`, true when `cacheWrite1h` is greater than 0 over the summed entries.
   Nothing new is stored: an entry's `cacheWrite1h` member (P1) is its mark, since Pi's `cost` has no place for one
   (`session/session.go:221-228`).
3. **The marker.** When the session's model is priced and `approximate` is true, `usageText`'s and `cmdContext`'s
   `Cost` line reads `Cost: $0.123 (approximate: one-hour cache writes priced at the write rate)`. Its digits are as
   today. Otherwise the line is as today. `_gobble/usage`'s text, which is `usageText`'s, carries it too.
4. **No marker elsewhere.** The status line, `--stats`, `usage_update` and `_gobble/get_session_stats` are unchanged
   (MADR D4: neither the prompt response nor `usage_update` carries a one-hour count).
5. **Tests:**

   | Test | What it holds | Planted break (C7) |
   | :--- | :--- | :--- |
   | `TestPriceOneHourAtWriteRate` (`llm/catalog`) | a model with `Input` 4 and `CacheWrite` 5, and the same above a tier: a million writes, of which 400,000 are one-hour, cost 5.0, the write rate for all | `Price` charging the one-hour part at two times the input: 6.2 |
   | `TestPricedUsageOneHourAtWriteRate` (`acpserver/catalog_test.go`) | an assistant entry's stored cost for a turn with one-hour writes equals the cost of the same writes without them | a stored cost that differs |
   | `TestCostMarkedApproximate` (`acpserver/usage_test.go`) | `/usage` and `/session` over a session holding one-hour writes show the marker; over one with writes but no one-hour writes, they do not; over an unpriced model, no cost line, as today | the marker keyed on `cacheWrite` instead of `cacheWrite1h`; the marker dropped |
6. C5. Commit (C8).

### P3 — `/usage` and `/session` split writes out (MADR D2)

**Files:** `acpserver/commands.go`, `acpserver/usage_test.go`, this plan, `docs/README.md`.

1. `tokenLines` writes, under `Input`: `  Cache read: N`, `  Cache write: N`, and, when non-zero, `    of which 1h: N`,
   then `  Uncached: N`, where Uncached is `input` alone. With no cache use, the lines are as today: `Input` alone.
2. **Test:** `TestTokenLinesSplitWrites` pins the lines for a session with reads, writes and one-hour writes, and for
   one with none. Planted break: Uncached computed as `input + cacheWrite`, as at `512db99`.
3. `_gobble/get_session_stats` is unchanged: its `CacheWrite` already counts writes apart (`acpserver/extensions.go:512-513`).
4. C5. Commit (C8).

### P4 — the terminal's figures (MADR D1, D5; Q1 (a), Q2 (a))

**Files** (as 0056-PLAN leaves them; re-read at the phase's base, C1): `internal/cli/render/status.go`,
`internal/cli/render/status_test.go`, the line renderer's `TurnEnd` drawing in `internal/cli` (at `512db99`,
`statusLine` in `internal/cli/view.go:473-494`), `internal/cli/print.go`, their tests and goldens, this plan,
`docs/README.md`.

1. `render.Stats` gains the turn's cache read and write counts, drawn after tokens per second as `cache <read> read ·
   <written> written`, in the same short token form as the status line, only when either is non-zero.
2. The status line under `--stats` passes them from `Result.Usage.CachedReadTokens` and `CachedWriteTokens`, in the
   line session and in print mode. No mode branches on it (0056-MADR D13: neither renderer branches on the mode). No
   approximate marker is drawn (MADR D4).
3. **Tests:** `render.Stats`'s table gains rows with reads, writes, both and neither; the line session's `--stats`
   golden and print mode's gain a turn that used the cache. Planted break: read and written swapped.
4. **The stream unchanged:** 0056-MADR D1's test, every side's bytes equal across the four modes, passes unchanged, and
   `git diff --exit-code <base> -- acpserver agent tool acpext llm` is empty for this phase.
5. C5, with `make perf` and `make probe-conhost`. Commit (C8).

### P5 — close-out

1. `docs/architecture.md`, where it describes `/usage` or usage, and `docs/guides/choosing-a-model.md`, where it
   describes `cacheRetention: long`'s cost, are corrected if either says otherwise. The guide states that a `long`
   session's cost is marked approximate, and why. Each document is named in the record, or the record says neither
   needed a change.
2. The disclosure guard, over the outgoing commits, is clean.
3. This plan becomes `complete` when every Goal item is met.
4. Commit (C8).

## Verification

* `go test -count=1 -run 'TestUsageOfOneHour' ./llm/provider/`; `-run 'TestAddUsageOneHour' ./agent/`;
  `-run 'TestPriceOneHourAtWriteRate' ./llm/catalog/`;
  `-run 'TestOneHourWritesStored|TestPricedUsageOneHourAtWriteRate|TestCostMarkedApproximate|TestTokenLinesSplitWrites' ./acpserver/`
  pass on all three hosts;
* every new test has its quoted failure on its planted break (C7);
* the session goldens and `session/jsonl/testdata/` are unchanged by P1 to P3;
* 0056-MADR D1's byte-equality test passes unchanged after P4.

## Rollout and Rollback

* Each phase is one commit and reverts on its own. Reverting P1 needs P2 and P3 reverted first, since both read the
  count P1 adds.
* **What users see change:**
  * P1: nothing.
  * P2: `/usage` and `/session` mark a cost approximate when the session holds one-hour writes. The figure itself does
    not change.
  * P3: `/usage` and `/session` show writes apart.
  * P4: `--stats` shows the turn's cache figures.
* A session file written by this plan's code is read by any earlier gobble and by Pi. The one new member,
  `cacheWrite1h`, is an unknown member to them.

## Execution record

Not started. The plan is `proposed`. Q1 to Q3 are ruled (2026-11-14). It waits for `v0.7.0` and for the owner's
approval of its execution.
