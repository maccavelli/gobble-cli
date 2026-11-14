---
status: accepted
date: 2026-11-14
decision-makers: repository owner
consulted: "a read-only analysis of gobble's prompt caching, 2026-11-14, with throwaway probes on a scratch clone of main at 512db99; this repository at 512db99 (main, clean), read for every fact below; go-llmprovider-sdk v1.29.0 and acp-go-sdk2 v1.36.0, read from the module cache; the owner's rulings of 2026-11-14, relayed by the coordinator"
informed: "magic-cli-remote, whose ACP stream this record leaves unchanged (under Q3's ruling the usage_update cost does not change either); `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md`, whose TurnEnd and status line D5 draws on"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Show the prompt cache's reads and writes, count one-hour cache writes, and mark their cost approximate until the catalog prices them

*(2026-11-14: retitled by Q3's ruling. As drafted, the title ended "and price one-hour cache writes at their own
rate".)*

## Context and Problem Statement

[0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md) has every agent turn ask the
provider to cache its prompt, and counts the cached reads and writes. Its D5 has the ACP prompt response report both.
Its D7 left one-hour writes unpriced: "Warming and one-hour pricing wait for the catalog … `llm.Usage` gains no
one-hour field until F4-2's cost work has a use for it." The catalog now prices every model (`catalog.Price`).

On 2026-11-14 a read-only analysis of gobble's caching found that a person at the terminal can barely see what the
cache does, and that one-hour writes are priced at the five-minute rate. The owner ruled that day: draft "Cache
visibility + 1-hour pricing" now, with its code "after v0.7.0". The same analysis's first record is the 2026-11-14
amendment of [0037-MADR-hardening-integrity-and-performance.md](0037-MADR-hardening-integrity-and-performance.md), on
a resumed session's prefix, whose code is built now.

The question: where, and in which view modes, should the terminal show the cache's reads and writes, and how should a
one-hour write be counted and priced, without changing the ACP stream that
[0056-MADR-curate-the-terminal-session-view-with-view-modes.md](0056-MADR-curate-the-terminal-session-view-with-view-modes.md)
D1 keeps raw?

### What the code does at `512db99` (read 2026-11-14; facts, no conclusions)

* **F1. The status line and `--stats` show no cache figure.**
  * `statusLine` (`internal/cli/view.go:473-494`) passes the last `usage_update`'s used tokens, window and cost to
    `render.StatusLine` (`internal/cli/render/status.go:25-38`): elapsed, a bar, percent, tokens, cost.
  * Under `--stats` it adds `render.Stats` (`:43-49`): time to first token and tokens per second, from the response's
    `OutputTokens` alone. Print mode writes the same line under `--stats` (`internal/cli/print.go:232-237`).
* **F2. The ACP prompt response already carries the turn's cache counts, and the CLI drops them.**
  * `acpUsage` (`acpserver/prompt.go:312-325`) sets `CachedReadTokens` and `CachedWriteTokens` when non-zero, summed over
    the turn's model calls (`agent/agent.go:1015-1024`).
  * acpclient decodes both into `Result.Usage` (`acpclient/conn.go:299-306`, `:344-346`).
  * Nothing under `internal/cli` reads either field: a search for `CachedReadTokens` and `CachedWriteTokens` finds them
    only in `acpclient`.
* **F3. `usage_update` carries no cache count.** It is used tokens, window size and the session's cost
  (`acpserver/prompt.go:330-338`; acpclient's `ContextUsage`, `acpclient/update.go:250-253`). A session's cache totals
  reach an ACP client only as `_gobble/get_session_stats`'s token stats (`acpserver/extensions.go:512-513`) and as
  `/usage` and `/session` text.
* **F4. `/usage` and `/session` count cache writes as uncached.**
  * `tokenLines` writes `fmt.Fprintf(b, "  Cached: %s\n  Uncached: %s\n", thousands(t.cacheRead),
    thousands(t.input+t.cacheWrite))` (`acpserver/commands.go:263`).
  * Both `/usage` (`usageText`, `:281`) and `/session` and `/context` (`cmdContext`, `:329`) call it.
  * No test pins either line: "Uncached" appears in no test file.
  * A write is billed at its own rate, not the input rate, so "Uncached" mixes two prices.
* **F5. One-hour writes are counted by the SDK and dropped by gobble.**
  * go-llmprovider-sdk `v1.29.0` (`go.mod:17`): "CacheWrite1hTokens is the part of CacheWriteTokens written with a
    one-hour lifetime. Only Anthropic's Messages reports it." (`llmprovider/contract.go:201-203`).
  * `usageOf` (`llm/provider/provider.go:442-452`) maps `CachedTokens` and `CacheWriteTokens`, not
    `CacheWrite1hTokens`. `llm.Usage` (`llm/event.go:79-86`) has no field for it. Pi's stored usage has none either
    (`session/session.go:210-219`), but keeps unknown members, as `piUsage` already does for `estimated`
    (`acpserver/history.go:144-146`).
* **F6. Every write is priced at one rate.**
  * `catalog.Price` (`llm/catalog/catalog.go:132-147`) prices `cacheWrite` at the model's, or the tier's, one
    `CacheWrite` rate. `llm.ModelCost` and `llm.CostTier` (`llm/llm.go:180-195`) have one `CacheWrite` each.
  * The snapshot has no one-hour rate. Across the 768 priced models of `llm/catalog/data/models.json`, the cost keys are
    `input`, `output`, `cache_read`, `cache_write`, `tiers`, `context_over_200k`, `reasoning`, `input_audio` and
    `output_audio`. All 17 Anthropic models carry `cache_write`, at 1.25 times `input` (`claude-opus-5-5`: 4 and 5).
  * 0021-MADR F13 puts a one-hour write at "2 times rather than 1.25 times" the input.
  * One-hour writes happen only under `cacheRetention: long` on Anthropic's routes (0021-MADR D3; the default is
    `short`, `internal/config/keys.go:84`). There every write is a one-hour write and is priced at 62.5% of its charge.
    The cost is stored in each assistant entry when it is written (`pricedUsage`, `acpserver/history.go:124-129`), and
    every total is summed from the entries (`sessionTotals`, `acpserver/commands.go:194-204`), including the
    `usage_update` cost (`acpserver/prompt.go:334-336`).
* **F7. What 0056 fixes, and what it rules out.**
  * 0056-MADR D13's `TurnEnd` item carries "the last usage and the `acpclient.Result`" to the footer, which the line
    renderer draws as the status line on stderr. The `Result` is where F2's counts already are.
  * D2: "The status line and `--stats` (D7) are the same in every mode." D7 keeps `--stats`.
  * D1: no view mode changes the ACP stream. A count the terminal shows must be one the stream already carries, which
    F2's two are and F5's one-hour count is not: `acp.Usage` has no such field (acp-go-sdk2 `v1.36.0`,
    `types_gen.go:24548-24567`).
  * 0056-PLAN's C6 requires `git diff --exit-code <base> -- acpserver agent tool acpext llm` to be empty after each of
    its phases.

## Decision Drivers

* **The ACP stream stays as it is** (0056-MADR D1). The terminal shows what the prompt response already carries. No new
  `_meta` member, no new `usage_update` field.
* **Every figure is priced at what it costs.** A one-hour write is not a five-minute write.
  *(2026-11-14: Q3's ruling keeps the catalog's one write rate until the catalog has a one-hour rate. Until then this
  driver is met in part: the cost is labelled approximate where it is read in full.)*
* **A label says what it counts.** "Uncached" counts no write.
* **The quiet default stays quiet.** 0056 curates the default view, and a new figure earns its place there.
* **The owner's timing.** The code lands after `v0.7.0`, and outside 0056-PLAN's phases (its C6).
* **The owner is the only user.** Stored sessions are not rewritten, and no compatibility shim is kept.

## Considered Options

* **A. A record of its own:** the terminal shows the turn's cache counts from the prompt response, `/usage` and
  `/session` split writes out, and one-hour writes are counted and priced. 0021-MADR D7 and 0056-MADR D2 and D13 get
  dated notes on acceptance.
* **B. Amend the two records instead:** a 0056 decision and phase for the terminal's figures, and a 0021 plan for
  one-hour pricing.
* **C. As A, with the session's cache totals sent on `usage_update`'s `_meta`,** so that the status line can show the
  session's figures, not only the turn's.
* **D. Change nothing:** `/usage` already shows a cached count.

## Decision Outcome

Chosen option: "A. A record of its own" (accepted 2026-11-14 by the owner, with Q1 to Q3 ruled below), because:

* the work crosses packages 0056 keeps out of its scope (`llm`, `llm/provider`, `llm/catalog`, `agent`,
  `acpserver`), and 0056-PLAN's C6 forbids those changes during its phases;
* 0021's plan is complete, and its D7 deferred exactly this work to a later record;
* the terminal half needs only what the prompt response carries (F2), so 0056's D1 holds unamended.

C sends the phone bytes it does not need, for figures `/usage` already gives. D leaves F4's misleading line and F6's
under-price.

~~The owner has ruled the timing: the code lands after `v0.7.0`. The rest is proposed. Three questions are open: where
the counts show (Q1), how they behave per view mode (Q2), and where a one-hour rate comes from (Q3).~~

*(2026-11-14, later: the owner ruled on all three. Q1: "--stats, plus /usage and /session split", (a). Q2: "Same in
every mode", (a). Q3: "Catalog rate, flagged approximate", (b), which was not the recommended option. D4 is reworked
to Q3's ruling, and D5 is written to Q1 and Q2. The code still lands after `v0.7.0`, and the plan stays `proposed`.)*

### D1. The terminal's cache figures come from the prompt response

The CLI reads `Result.Usage.CachedReadTokens` and `CachedWriteTokens`, which acpclient already decodes (F2). No ACP
field, `_meta` member or `usage_update` change is made. The figures are the turn's, summed over its model calls. Where
they show is Q1, and per mode Q2.

### D2. `/usage` and `/session` split cache writes out of "Uncached"

`tokenLines` writes input as three parts: cache reads, cache writes, and input neither read nor written. The total
`Input` line is unchanged. When the session holds one-hour writes, a line under the writes gives their part (D3). The
words stay the agent's own, as today, and are not personality keys. A test pins the lines for the first time.
*(2026-11-14, by D4 as reworked: the same commands' `Cost` line carries the approximate marker.)*

### D3. One-hour writes are counted

* `llm.Usage` gains `CacheWrite1hTokens` (`json:"cacheWrite1hTokens,omitzero"`). This lifts 0021-MADR D7's deferral.
* `usageOf` maps the SDK's field, and `agent`'s `addUsage` sums it.
* `piUsage` stores it in the usage's unknown members as `cacheWrite1h` when non-zero, as it stores `estimated`.
  `sessionTotals` reads it.
* Pi's own fields are unchanged, and a file without the member reads as zero one-hour writes.

### D4. One-hour writes are priced at the catalog's write rate, and a cost that includes them is marked approximate

*(2026-11-14, reworked by Q3's ruling, "Catalog rate, flagged approximate". The draft read as follows, struck:)*

~~* `catalog.Price` takes the one-hour count as a part of `cacheWrite`: that part at the one-hour rate, the rest at
  `CacheWrite`, per tier as today. `pricedUsage` passes it.~~
~~* The rate comes from Q3's answer.~~
~~* Costs already stored in old entries are not recomputed.~~

* **The price.** `catalog.Price` is unchanged. Every write, one-hour writes included, is priced at the model's or the
  tier's one `CacheWrite` rate. This stays so until the catalog's snapshot carries a one-hour rate, which a later record
  adopts. No multiplier and no provider-keyed rule is added. A test pins it: one-hour writes cost what other writes
  cost.
* **What counts as approximate.**
  * A turn's or an entry's cost is approximate when its usage has one-hour writes: `cacheWrite1h` greater than 0 (D3).
  * A session's cost is approximate when any entry summed into it is (`sessionTotals`).
* **The marker, and where it shows:**
  * **`/usage` and `/session`:** the `Cost` line, when the session's cost is approximate, reads
    `Cost: $0.123 (approximate: one-hour cache writes priced at the write rate)`. Its digits are as today. The
    `of which 1h: N` line under the writes (D2) says how many there were. `_gobble/usage`, which returns `/usage`'s
    text in boring's voice (`usageText`, `acpserver/commands.go:274-277`), carries the same marker, because it is the
    same text.
  * **`--stats` and the status line: no marker.** The status line's cost is `usage_update`'s session cost, and the
    turn's figures are the prompt response's. Neither carries a one-hour count (`acp.Usage`, F7), and D1 adds no field.
    A marker there would need a stream change, which 0056-MADR D1 rules out, or a guess from the session's retention
    and provider. This record takes neither. `/usage` and `/session` are where the cost is qualified.
  * **The session file: no member of its own.** Pi's `cost` object has no place for one: `session.Cost` is five
    numbers, with no unknown members (`session/session.go:221-228`). The usage's `cacheWrite1h` member (D3) is the
    mark: an entry that holds it has a cost priced at the write rate.
* Costs already stored in old entries are not recomputed. An old entry has no `cacheWrite1h`, so it is never marked,
  even if it was a one-hour write.

### D5. Where and when the figures show (Q1 (a), Q2 (a))

~~These are written as decisions once the owner answers.~~

*(2026-11-14, written to the rulings "--stats, plus /usage and /session split" and "Same in every mode".)*

* Under `--stats`, the status line gains the turn's cache figures after tokens per second: `cache 12.3k read · 1.1k
  written`, in the status line's short token form, only when either is non-zero. This holds in print mode and in the
  line session.
* Without `--stats`, the status line is unchanged. The figures are in `/usage` and `/session` (D2).
* The figures are the same in every view mode, as 0056-MADR D2 has the status line and `--stats`. 0056's D2 sentence
  and D13's `TurnEnd` row are unchanged: `TurnEnd` already carries the `acpclient.Result` the counts are in.

### D6. Timing and order

* The owner's ruling: nothing is built before `v0.7.0`.
* D2 to D4 change `llm`, `llm/provider`, `llm/catalog`, `agent` and `acpserver`. Under 0056-PLAN's C6, they land before
  0056-PLAN's P0 or after its P4, never between.
* D1 and D5 draw on 0056's `TurnEnd` and its line renderer, so they land after 0056-PLAN's P3. Before that, D1 would be
  built into `statusLine` and then rebuilt.
* The 0037 amendment's plan,
  [0037-PLAN-resume-prefix-and-failed-turn-history.md](0037-PLAN-resume-prefix-and-failed-turn-history.md), lands
  before any of this.

### D7. The records this amends on acceptance

Each gets a dated note beside the text it changes, written in the plan's P0:

* 0021-MADR D7 and F13, whose deferral D3 and D4 lift;
* 0056-MADR D2's "The status line and `--stats` (D7) are the same in every mode", and D13's `TurnEnd` row, only as Q1
  and Q2 require.

*(2026-11-14: no note depends on an open question any more, so all are written now, on acceptance:*

* *[0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md) D7 and F13. D3 lifts the
  field's deferral. The pricing half stays at the write rate, marked approximate (D4).*
* *[0056-MADR-curate-the-terminal-session-view-with-view-modes.md](0056-MADR-curate-the-terminal-session-view-with-view-modes.md)
  D2's sentence. It gets a note that `--stats` also shows the turn's cache figures, in every mode, and that the
  sentence holds. D13's `TurnEnd` row needs no change, and its note says so.)*

### Consequences

* Good, because a person at the terminal sees what the cache did, from figures the agent already sends.
* Good, because `/usage` and `/session` stop counting writes as uncached input.
* ~~Good, because a `long` retention session on Anthropic's routes reports its true cost. Today that cost is
  understated by 37.5% on writes, by 0021-MADR F13's rate.~~ *(2026-11-14, struck by Q3's ruling.)*
* Bad, because the under-price stays. *(2026-11-14, by Q3's ruling.)*
  * Under `cacheRetention: long` on Anthropic's routes, every write is a one-hour write. It is priced at 62.5% of its
    charge: 1.25 times the input against the 2 times of 0021-MADR F13. So the write cost is understated by 37.5%.
  * `/usage` and `/session` say so, with the marker. The status line's cost does not (D4).
* Good, because the under-price is now visible where the cost is read in full, and the count it needs for a later
  rate is already stored (D3). *(2026-11-14.)*
* ~~Neutral, because the phone's stream is unchanged except for `usage_update`'s cost, which D4 makes correct.~~
  Neutral, because the phone's stream is unchanged: under Q3's ruling, `usage_update`'s cost is computed as today.
  *(2026-11-14.)*
* Neutral, because old entries keep the costs they were written with.
* ~~Bad, because under Q3 (a) a multiplier stands in for a catalog rate until the snapshot carries one.~~
  *(2026-11-14: struck; Q3 (b) adds no multiplier.)*
* Bad, because the terminal half waits for 0056-PLAN's P3.

### Confirmation

* `usageOf` maps a response's `CacheWrite1hTokens`. A planted break, the mapping removed, fails its test.
* ~~`catalog.Price` prices a million one-hour tokens at Q3's rate. A planted break, the one-hour part priced at
  `CacheWrite`, fails its test.~~ *(2026-11-14, by Q3's ruling:)* `catalog.Price` prices one-hour writes at
  `CacheWrite`. A session with one-hour writes shows `/usage`'s and `/session`'s approximate marker, and one without
  them does not. Planted breaks: the marker shown with none; the marker missing with some.
* `piUsage` and `sessionTotals` carry `cacheWrite1h` through a write and a read.
* `/usage` and `/session` pin the three input lines. A planted break, writes folded back into "Uncached", fails.
* The terminal's figures, per Q1 and Q2, are pinned by the line renderer's goldens. A planted break, reads and writes
  swapped, fails them.
* 0056-MADR D1's test, the bytes each side wrote equal across the four modes, passes unchanged.
* Each new test is seen failing on a scratch copy before its change is trusted.

## Pros and Cons of the Options

### A. A record of its own

* Good, because its scope is its own: the pricing half lands without waiting for 0056, and the terminal half after it.
* Good, because 0021's deferral and 0056's sentence each get one dated note, not a new decision buried in an accepted
  record.
* Bad, because one more record to read beside 0021 and 0056.

### B. Amend 0056 and 0021

* Good, because no new number.
* Bad, because 0056's D1 keeps `llm`, `agent` and `acpserver` out of its scope, so the pricing half cannot be one of
  its phases. 0021's plan is complete, and a new plan under it would carry a decision 0021 deferred to the catalog.

### C. Session totals on `usage_update`'s `_meta`

* Good, because the status line could show the session's figures, not only the turn's.
* Bad, because it changes every client's stream for a figure `/usage` gives on request.

### D. Change nothing

* Good, because no work.
* Bad, because "Uncached" stays wrong, and a `long` session's cost stays under-reported.

## Questions for the owner (2026-11-14)

Each has three options, the recommended one first. *(2026-11-14, later: all three are ruled. Each ruling follows its
options, which stand as written. Until then this heading read "open".)*

* **Q1. Where the turn's cache figures show (D1, D5).** D2's split of `/usage` and `/session` applies under every
  option.
  * **(a) Recommended. Under `--stats` only,** after tokens per second: `cache 12.3k read · 1.1k written`, in print mode
    and the line session alike. Cost: someone who never passes `--stats` sees them only in `/usage` and `/session`; the
    default line is unchanged. `--stats` is already the flag for diagnostic figures.
  * **(b) The status line always shows a hit ratio when the turn used the cache,** `cache 92%`, with `--stats` adding
    the counts. Cost: one more part on every line, which can crowd a narrow terminal. 0056's D13 `TurnEnd` row changes.
  * **(c) No terminal line,** `/usage` and `/session` only. Cost: the per-turn figures, which the agent already sends,
    stay unshown, and the owner's "nearly invisible" stands for the line.

  *Ruling, 2026-11-14: "--stats, plus /usage and /session split". **Decision: (a)** (D5).*
* **Q2. How the figures behave per view mode (0056's `terse`, `default`, `verbose`).**
  * **(a) Recommended. The same in every mode,** as 0056-MADR D2 has the status line and `--stats`. Cost: none to 0056's
    contract. `terse` shows them when Q1's condition holds.
  * **(b) `verbose` always, `default` per Q1, `terse` never.** Cost: amends D2's sentence and D14's `TurnEnd` row; the
    curation layer gains a fold on `TurnEnd`.
  * **(c) `verbose` only,** with `raw` showing the response's JSON anyway. Cost: the same amendment, and a `--stats`
    user in `default` sees none.

  *Ruling, 2026-11-14: "Same in every mode". **Decision: (a)** (D5). 0056 is not changed.*
* **Q3. The one-hour rate (D4).**
  * **(a) Recommended. Two times the input rate on Anthropic's routes, until the catalog carries a one-hour rate; then
    the catalog's.** It applies per tier, at two times that tier's input. It is the rate 0021-MADR F13 records. The
    plan's P0 checks it against Anthropic's published price page before any code. Cost: a provider-keyed multiplier
    in `llm/catalog`, removed when the snapshot gains the rate. A test fails when a snapshot that has the rate is
    still priced by the multiplier.
  * **(b) The catalog's rate only.** Until the snapshot has one, a one-hour write is priced at `CacheWrite`, and
    `/usage` marks the cost "approximate" when the session holds one-hour writes. Cost: the under-price stays, flagged.
  * **(c) A `models.json` override, `cost.cacheWrite1h`,** else `CacheWrite`. Cost: a new key in
    [docs/guides/models-json.md](../guides/models-json.md), set by hand per model, and wrong until it is set.

  *Ruling, 2026-11-14: "Catalog rate, flagged approximate". **Decision: (b)**, not the recommended option. D4 is
  reworked to it: the marker's words and where it shows are D4's.*

## More Information

* **Evidence.** Every file and line above was read at `512db99`. The SDK's and acp-go-sdk2's lines are from the module
  cache at the pinned versions. The snapshot's cost keys were counted with a stdlib Python script over
  `llm/catalog/data/models.json`. Nothing was measured against a live service.
* **Related records:**
  * [0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md): D3 the retention key, D5 the
    response's counts, D7 the deferral, F13 the one-hour rate;
  * [0056-MADR-curate-the-terminal-session-view-with-view-modes.md](0056-MADR-curate-the-terminal-session-view-with-view-modes.md):
    D1, D2, D7, D13 and D14;
  * [0037-MADR-hardening-integrity-and-performance.md](0037-MADR-hardening-integrity-and-performance.md): its
    2026-11-14 amendment, the same analysis's first record;
  * [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md):
    `v0.7.0`.
* **The plan:** [0057-PLAN-cache-visibility-and-one-hour-pricing.md](0057-PLAN-cache-visibility-and-one-hour-pricing.md).
