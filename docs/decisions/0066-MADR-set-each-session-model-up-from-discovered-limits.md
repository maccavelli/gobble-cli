---
status: accepted
date: 2026-11-22
decision-makers: repository owner
consulted: "the owner's reply of 2026-11-22 on go-llmprovider-sdk's 0088-MADR OQ9, the owner's question on omitting the limit, the owner's design note, and the answers to its OQ14 to OQ16, relayed by the coordinator; go-llmprovider-sdk's 0089-MADR-model-output-limit-discovery.md (proposed, second draft, the SDK side of this work) and 0088-MADR's fourth amendment; this repository's 0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md (proposed, uncommitted); record review 3 of 2026-11-22, each finding re-verified before it was written in; this repository at dfaa1ac and go-llmprovider-sdk at 751e970, read only"
informed: go-llmprovider-sdk, whose 0089 exports name their call sites here (its R55); magic-cli-remote, which shows gobble's ACP config options and notices
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Set Each Session's Model Up from the SDK's Discovered Limits, Show Their Sources, and Keep the Caps a Service States

Associated PLAN: [0066-PLAN-set-each-session-model-up-from-discovered-limits.md](0066-PLAN-set-each-session-model-up-from-discovered-limits.md)

*(2026-11-22, second draft, after record review 3: discovery receives models.json through 0065's stated-only wrapper, so Pi's invented defaults never become evidence (M3); the snapshot is no longer handed to the SDK, which embeds its own registry extract; learned caps are kept with their base URL, scope and time, and expire; the session's wait ends with the session; a thinking-only change does not rediscover; the override question is asked with the SDK's ceiling, as go-llmprovider-sdk 0089-MADR OQ4.)*

*(2026-11-22, the owner's answers, relayed by the coordinator: go-llmprovider-sdk 0089-MADR OQ1 "Only where required (Recommended)", OQ2 "Never wait; route's smallest (Recommended)", OQ3 "By default, can turn off (Recommended)", and OQ4, after the owner asked "how about option 1 and option 3? pros vs cons. explain." and the coordinator explained, "models.json only, for now (Recommended)"; and its 0088 OQ18 "Five now; 0088+0089 next (Recommended)". S32 to S34 are decided; D7's learning is the SDK's default, so gobble needs no opt-in; D12's release is the go-llmprovider-sdk release after the next tag, carrying 0088 and 0089. No row is open; by the records rule this record is `accepted`. Its PLAN stays `proposed` until the owner approves it.)*

*(2026-11-24, note from [0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md) D12: D11 and S29 ("gobble names no limit and no ceiling of its own") are superseded by 0071 V3 for every call where gobble names a limit; S39 by 0071's Q2 answer, "16,384, fitted, shown 'assumed' (Recommended)"; D8's session-report line is merged with 0071 D10's by 0071 V6. This record's adoption amendment decides 0071 V7.)*

## Context and Problem Statement

On 2026-11-22 the owner asked for a model output-limit discovery feature in
go-llmprovider-sdk "that provides an API surface that clients (gobble) can call
to determine the model output token values acccurately to set them correctly in
sessions", with "a pair in the gobble-cli repo for the adjoining work there"
(quoted in full in go-llmprovider-sdk's
`docs/decisions/0089-MADR-model-output-limit-discovery.md`). That SDK record
adds `catalog.LookupLimits` and `catalog.CachedLimits`, which return each limit
(output, window, input) with every source's value and time, "unknown" as a
state, and whether the wire requires a limit; a validated refusal adds a cap,
which a `WarningLimitLearned` reports for a caller to keep and pass back; and
every response says what output limit it carried and why
(`Response.OutputLimit`). Its OQ1 asks whether the SDK sends a limit only where
the wire requires one, as the owner asked. This record decides how gobble uses
it.

### How gobble sets a model up today (read at dfaa1ac)

* **A session's facts come from a cached lookup that never waits.** The
  agent's `ModelInfo` hook (`internal/cli/agent.go:117`) calls
  `provider.ModelInfoWith`, which calls the SDK's `catalog.CachedModel` with
  the embedded snapshot and models.json as layers
  (`llm/provider/model.go:107-120`, `:169`).
* **models.json invents two facts.** A model models.json defines without
  `maxTokens` or `contextWindow` gets Pi's defaults, 16,384 and 128,000
  (`llm/catalog/modelsjson.go:115-118`, `:458-462`), each marked known by the
  layer (`llm/provider/model.go:201-202`); 0065-MADR D2 keeps them from the
  provider's request by a stated-only wrapper.
* **The provider has no facts of gobble's.** `llm/provider.New` builds the
  SDK provider with `WithoutModelMetadata()` and no layers
  (`llm/provider/provider.go:212-216`).
* **`llm.Model` keeps one number per limit and no source**: "0 is unknown"
  (`llm/llm.go:160`); `modelOf` copies the merged value (`model.go:237`).
* **Where the facts are shown and used**: `gobble models` and `gobble models
  info` (`internal/cli/modelscmd.go:157`, `:292`); the ACP model option, which
  carries a name and a value (`acpserver/config.go:216-235`), although ACP's
  select option has an optional `description`; the session report's context
  line (`acpserver/commands.go:331`); automatic compaction's trigger
  (`acpserver/autocompact.go:105`); a model or thinking change
  (`acpserver/config.go:243`, the one path of `set_config_option`, `/model` and
  `/thinking`).
* **Learned refusals are kept across runs**: reasoning refusals are saved to
  `reasoning-refusals.json`, ten per provider, and handed back at provider
  construction (`internal/cli/refusal.go:17`, `:74-98`; `agent/warning.go:41`;
  0036-MADR D4).

## Decision Drivers

* The owner's words: accurate limits from discovered evidence, set correctly
  in sessions; "nothing means nothing".
* A session's first call already carries what discovery found.
* What gobble shows is what the SDK found and chose, with its source;
  "unknown" is shown as unknown, and no invented number becomes evidence.
* A kept cap cannot outlive its evidence: it carries its scope and time and
  expires.
* One owner per rule: the SDK discovers and chooses; gobble sets sessions up,
  shows, persists and tells the user (0065's ownership split).

## Considered Options

* A. Discover at session start and model switch, wait for it before the
  session's first call, and show and keep the SDK's evidence
* B. Discover lazily: rely on the SDK's request to discover on first use (its
  OQ2 (2)) and show what is cached
* C. Discover at startup for every model gobble lists
* D. Keep today's cached lookup, and add sources to what it shows

## Decision Outcome

Chosen option: "A. Discover at session start and model switch", because it
makes a session's first call accurate whatever the SDK answers to its OQ2 (B
relies on its (2)), reads one model per session rather than every listed model
(C), and carries the SDK's evidence to the user (D does not discover).

### The decisions

* **D1. Session start.** When a session's model is set (an ACP session
  created or loaded, the CLI's interactive start, a headless run), gobble
  starts `catalog.LookupLimits` for that provider and model in the background,
  with the session's context, the provider's own credential source, the run's
  cache and every SDK option `provider.New` uses (base URL, header transport,
  Claude workspace, Kilo organization), and the layers of D3. The session's
  first model call waits for that discovery to end; the SDK bounds it (10 s); a
  failure or a timeout lets the call go on with what is cached. A cancel ends
  the wait with the turn, as today; closing the session cancels the discovery.
  Later calls do not wait. The session's title model is discovered at the same
  time.
* **D2. Model switch.** A model change through `setConfig` (`/model` and
  `set_config_option`), to the same or another provider, starts discovery for
  the new model as D1, and the next call waits for it. A thinking-only change
  does not rediscover.
* **D3. What gobble hands the SDK.** models.json only, as today's layer named
  "models.json", which the SDK counts as the caller's statement, through 0065
  D2's stated-only wrapper (the output limit and the window, and only what
  models.json states), to discovery as to the provider; Pi's defaults never
  reach either. The embedded snapshot is not handed to the SDK for limits:
  the SDK reads models.dev itself and embeds its own limits-only extract below
  the live listings (go-llmprovider-sdk 0089 D3). The snapshot stays gobble's
  catalogue for its other facts.
* **D4. `llm.Model` carries the evidence.** It gains `Limits`, with `Output`,
  `Window` and `Input`, each `{Tokens, Known, Conflicting, Source, Stale,
  Others}` from the SDK's `Fact` and `Evidence`, and `Required`; its
  `MaxTokens` and `ContextWindow` keep the chosen value, 0 for unknown, for
  every existing reader.
* **D5. `gobble models`.** The table reads cached limits and never waits; an
  unknown cell is "-". `gobble models info <model>` discovers (the SDK's
  bound), and prints each limit with its source, each other source on its own
  line, "(stale)" past its source's freshness, and whether the model's wire
  requires an output limit.
* **D6. ACP model option.** Each model option's `description` reads
  "context N · max-out M" from cached limits, "unknown" where unknown; nothing
  is fetched to build it.
* **D7. Caps a service states.** gobble opts into the SDK's learning where its
  OQ3 makes it optional *(2026-11-22: OQ3 made it the default; gobble leaves
  it on)*. A `WarningLimitLearned` is saved to
  `limit-observations.json` in the state directory, beside
  `reasoning-refusals.json`, with its model, base URL, scope, tokens and time,
  newest first, ten per provider, base URL and scope (the bound the refusal
  file keeps), owner-only. On load an observation older than 24 hours is
  dropped, as the SDK would ignore it; an unreadable file is one warning and
  none handed back. The rest are handed back with `WithLimitObservations` at
  provider construction and to discovery (0065 D10, moved here).
* **D8. The limit a call carried.** Each call's `Response.OutputLimit` is
  logged at debug (tokens, sent or none, basis, source), and the session
  report's context line (`/context`, `/session`) shows the last call's
  ("output limit: 128,000, from listing:claude"; "output limit: none sent, the
  service's default applies"). 0065 D11's reported value and R31 move here.
  *(2026-11-24, note from `0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md` D19: the per-call
  debug line is 0070's `model call` record; this record adds its `out_limit`, `out_limit_sent`, `out_limit_basis` and
  `out_limit_source` fields when it is built. The session-report line stays this record's.)*
* **D9. The window gobble compacts against.** Automatic compaction and the
  session report read `Limits.Window`, else `Limits.Input` where a listing
  states only that (Claude, Gemini), else unknown as today; each with its
  source.
* **D10. Notices.** Once per session and model: when discovery ends with the
  output limit unknown, a notice says gobble names none, and that the
  service's own default applies (or, where the wire requires one, what the SDK
  sent instead); when a response's `OutputLimit` basis is `fallback`, a notice
  names the model and says to state `maxTokens` in models.json; when a cap is
  learned, a notice says the service stated it and that gobble keeps it for a
  day. Personality keys `model.limit.unknown`, `model.limit.fallback` and
  `model.limit.learned`, a line in each of `boring`, `dry`, `snarky` and
  `bombastic`. ACP sends each as an agent message; the CLI prints it; headless
  JSON carries it.
* **D11. The room in the window.** The SDK holds the limit to the room its
  estimate leaves (0088 OQ14, answered (3)); gobble names no limit and no
  ceiling of its own on any call, so 0065's Q1 is decided by that answer
  (a dated note in 0065).
* **D12. Ordering.** Built on the go-llmprovider-sdk release that carries
  0088 and 0089 (its OQ18 decides which), after 0065's Phase 2 bump to it, and
  shipped in one gobble release with 0065's Phases 1 and 2.

### Consequences

* Good, because a session's first call carries what discovery found, from the
  service's listing where one states it, Claude aliases included.
* Good, because the user sees which source set each limit, an unknown one as
  unknown, and no invented default is passed off as a fact.
* Good, because gobble's snapshot no longer outranks a live listing: the SDK's
  own registry tiers sit below it.
* Good, because a kept cap expires in a day and applies only to its scope.
* Bad, because a session's first call waits for one listing read (up to the
  SDK's 10 s on a slow service).
* Bad, because `llm.Model` grows, and `gobble models info` waits where it did
  not.
* Bad, because three personality keys and a state file are added.

### Confirmation

* The PLAN's tests, one per decided row, red at the base where a row changes
  behaviour, each with a plant caught on a scratch copy.
* `make preflight` passes, through `run-on-host` or `run-heavy`.
* An independent review of the last build phase finds no blocker.

### State table

"Today" is dfaa1ac. "The SDK" is go-llmprovider-sdk with 0088 and 0089 built.
Each decided row's test and plant are in the PLAN.

| Row | Input or state | Status | Outcome | Decided by |
| :--- | :--- | :--- | :--- | :--- |
| S1 | An ACP session created | decided | discovery for its model and its title model; the first call waits for it | D1 |
| S2 | An ACP session loaded or resumed | decided | as S1, for the model the session records | D1 |
| S3 | The CLI's interactive start | decided | as S1 | D1 |
| S4 | A headless run | decided | as S1 | D1 |
| S5 | A model switch, same provider | decided | discovery for the new model; the next call waits | D2 |
| S6 | A model switch to another provider | decided | the provider built first (as today), then discovery | D2 |
| S7 | Discovery fails | decided | the call goes on with what is cached; the error at debug; no notice unless S10 | D1 |
| S8 | Discovery slower than the SDK's bound | decided | the SDK ends it; the call goes on | D1 |
| S9 | A later call | decided | no wait | D1 |
| S10 | The output limit unknown after discovery | decided | one `model.limit.unknown` notice per session and model | D10 |
| S11 | A response whose `OutputLimit` basis is `fallback` | decided | one `model.limit.fallback` notice per session and model | D10 |
| S12 | A `WarningLimitLearned` | decided | saved with its scope and time (ten per provider, base URL and scope), one `model.limit.learned` notice | D7, D10 |
| S13 | `limit-observations.json` unreadable | decided | one warning; none handed back | D7 |
| S14 | Saved observations at the next start | decided | those under a day old handed back to the provider and to discovery; older ones dropped | D7 |
| S15 | Two sources disagree | decided | `models info` shows the chosen value and each other source | D5 |
| S16 | A fact past its source's freshness | decided | `models info` marks it "(stale)" | D5 |
| S17 | `gobble models` | decided | cached limits only, no wait | D5 |
| S18 | `gobble models info` | decided | discovers, then prints D5's lines | D5 |
| S19 | The ACP model option | decided | its description from cached limits; nothing fetched | D6 |
| S20 | The session report | decided | the last call's output limit and source; the window and its source | D8, D9 |
| S21 | The debug log | decided | each call's `OutputLimit` *(2026-11-24: as the `out_limit*` fields of `0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md`'s `model call` record, which 0070 built first, D19; absent until this record is built, 0070 T14)* | D8 |
| S22 | The window compaction uses | decided | `Limits.Window`, else `Limits.Input`, else unknown | D9 |
| S23 | The listing discovery reads and the request reads | decided | one key: the provider's first request reads discovery's (go-llmprovider-sdk 0089 DL72; 0065 R6) | D1 |
| S24 | Concurrent sessions, each with its own models.json | decided | each discovery and provider has its session's layer; nothing shared | D1, D3 |
| S25 | The snapshot, for every provider | decided | not handed to the SDK for limits; the SDK's registry tiers rank below the listings (supersedes 0065 D14, R42) | D3 |
| S26 | The snapshot's name | decided | not applicable: not handed (0065 D15, R44 lapse) | D3 |
| S27 | The snapshot's limit on OpenAI with a key, Grok, Ollama and chatcompat | decided | not handed, so none from gobble; what the SDK sends there is its OQ1's (0065 R45 lapses) | D3 |
| S28 | models.json `maxTokens` for a model | decided | the caller's statement, through the stated-only wrapper; chosen above the listing wherever the SDK sends a fact (its OQ1) | D3 |
| S29 | The room in the window | decided | the SDK's estimate; gobble names no limit and no ceiling of its own | D11 |
| S30 | The ChatGPT routes | decided | no limit sent; the window from the backend's `context_window` for compaction | D9; go-llmprovider-sdk 0089 DL8 |
| S31 | An Ollama or chatcompat session | decided | discovery reads Ollama's served context; compaction uses it | D9; go-llmprovider-sdk 0089 DL12 |
| S32 | A user override of the output limit | decided *(2026-11-22: the owner's answer)* | models.json `maxTokens` only: the caller's statement, sent where the SDK sends a fact (Messages routes); no global setting | go-llmprovider-sdk 0089-MADR OQ4 "models.json only, for now (Recommended)" |
| S33 | An ACP option for the output limit | decided *(2026-11-22: the owner's answer)* | none | go-llmprovider-sdk 0089-MADR OQ4 "models.json only, for now (Recommended)" |
| S34 | An overriding limit refused for its size | decided *(2026-11-22: the owner's answer)* | no overriding setting exists; a models.json `maxTokens` refused on a Messages route is resent by the SDK at the stated cap and remembered for the session's process (its OQ3), and the user is told (D10) | go-llmprovider-sdk 0089-MADR OQ4 "models.json only, for now (Recommended)" and OQ3 |
| S35 | `llm.Model`'s existing readers | decided | `MaxTokens` and `ContextWindow` keep the chosen value; 0 for unknown | D4 |
| S36 | The SDK release | decided | after 0065's Phase 2 bump to the release carrying 0088 and 0089 | D12 |
| S37 | A long session past the listing's freshness | decided | no rediscovery mid-session; the SDK reads stored facts of any age, marked stale (its DL23, DL66); a model switch or a new session discovers again | D1, D2 |
| S38 | Discovering every listed model at startup | out of scope | option C, rejected | |
| S39 | models.json defines a model without `maxTokens` or `contextWindow` | decided | Pi's defaults reach neither discovery nor the provider; `models info` shows the limit as unknown, from no source | D3; 0065 R3 |
| S40 | An ACP cancel, or the session closed, while the first call waits | decided | a cancel ends the turn as today; closing cancels the discovery | D1 |
| S41 | A thinking-only `setConfig` | decided | no rediscovery | D2 |

Counts: 40 decided, 1 out of scope, 0 open, 41 rows. *(2026-11-22, after the
owner's answers: S32 to S34 decided. Second draft: 37 decided, 1 out of scope,
3 open. First draft: 33 decided, 1 out of scope, 4 open, 38 rows.)*

## Pros and Cons of the Options

### A. Discover at session start and model switch

* Good, because the first call is accurate whatever the SDK answers to its
  OQ2.
* Good, because one model per session is read.
* Bad, because the first call can wait.

### B. Discover lazily

* Good, because nothing in gobble waits.
* Bad, because it rests on the SDK's OQ2 (2); under its (1) the first call goes
  out on the cache and the SDK's extract.

### C. Discover every listed model at startup

* Good, because every cell of `gobble models` is fresh.
* Bad, because a listing read per provider and, for Claude aliases and Ollama,
  one read per model, on every start.

### D. Add sources to today's lookup

* Good, because nothing waits.
* Bad, because nothing is discovered.

## Owner Questions

None in this record. The override question, 0065's Q3 with its options
restated, depends on the SDK's ceiling and is asked with it as
go-llmprovider-sdk `docs/decisions/0089-MADR-model-output-limit-discovery.md`
OQ4 (rows S32 to S34). 0065's Q1 is decided (D11); its Q2 stays in 0065.

## More Information

### Relation to other records

* `docs/decisions/0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md`
  (proposed): keeps its ownership split, D1 (narrowed to models.json), D2 (the
  wrapper, now also for discovery), D3 to D9, D12, D13 and its Q2; D10, D11's
  reported value, R31's reported part and Q3 move here or to the SDK's OQ4;
  D14, D15, R42, R44 and R45 lapse (the snapshot is not handed); Q1 is decided
  by D11; each by a dated note there.
* go-llmprovider-sdk `docs/decisions/0089-MADR-model-output-limit-discovery.md`:
  the API this record calls; its R55 names these call sites.
* go-llmprovider-sdk `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md`,
  its fourth amendment: the release (OQ18) and the estimate (OQ14).
* `docs/decisions/0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md`
  D4: the pattern D7 follows.

### What could not be established

* How long a listing read takes on each service; the first call's wait is
  bounded by the SDK, not measured.
