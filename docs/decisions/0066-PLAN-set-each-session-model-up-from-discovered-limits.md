---
status: proposed
date: 2026-11-22
associated-madr: "0066-MADR-set-each-session-model-up-from-discovered-limits.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement Setting Each Session's Model Up from the SDK's Discovered Limits

Associated MADR: [0066-MADR-set-each-session-model-up-from-discovered-limits.md](0066-MADR-set-each-session-model-up-from-discovered-limits.md)

*(2026-11-22, second draft, after record review 3: discovery through the stated-only wrapper; no snapshot handed; scoped, expiring caps; the order of discovery and the turn asserted; Claude named as the provider where a sent limit is asserted; cancel, close and thinking-only rows.)*

*(2026-11-24, note from [0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md) D12 and V6: this PLAN's phases that build discovery, the observation store and the notices come before 0071-PLAN's Phase 2, which uses them; D8's line is 0071 D10's.)*

## Goal

After this PLAN, every session discovers its model's limits before its first
call, and again on a model switch; `llm.Model` carries each limit's source;
`gobble models`, `gobble models info`, the ACP model option and the session
report show them; caps a service states are kept for a day, in their scope;
each call's output limit is logged and shown; and the user is told, once,
when a limit is unknown, a fallback, or learned.

The MADR is `accepted` (2026-11-22): go-llmprovider-sdk 0089-MADR's OQ4 was
answered "models.json only, for now (Recommended)", deciding S32 to S34. This PLAN stays `proposed`: nothing
below Phase 0 starts before the owner approves it and says `proceed`,
`execute the plan` or `do phase N`.

## Prerequisites

* The go-llmprovider-sdk release carrying 0088 and 0089 (its 0088-MADR OQ18),
  with its 0089 OQ1 to OQ4 answered.
* 0065-PLAN's Phase 2 (the bump to that release); this PLAN builds on its
  commit and ships in one release with 0065's Phases 1 and 2.

## Scope

### In scope

* `llm`: `Model.Limits` (`Output`, `Window`, `Input`, each `{Tokens, Known,
  Conflicting, Source, Stale, Others}`, and `Required`); `llm.Response`'s
  output limit and `llm.WarningLimitLearned`.
* `llm/provider`: `Discover(ctx, o Options, id, model, mj)` over
  `catalog.LookupLimits`, with the SDK options and credential `New` uses and
  models.json through 0065's stated-only wrapper; `modelOf` from
  `catalog.Limits`; `Options.Observations` handed to `New` and `Discover` as
  `WithLimitObservations`, and OQ3's opt-in where it exists.
* `agent`: the first call's wait on the session's discovery;
  `Config.SaveObservation`.
* `acpserver`: discovery on create, load and model switch; the model option's
  description; the session report; the notices; compaction's window.
* `internal/cli`: the CLI's and headless discovery; `limit-observations.json`
  (`rememberObservation`, `providerObservations`, as `refusal.go`); `gobble
  models` and `gobble models info`.
* `personality`: three keys, a line in each of four personalities.
* Docs: `docs/architecture.md`, `README.md` (models.json `maxTokens`; the
  state file; `gobble models info`), the personality keys' docs.

### Out of scope

* The override surface until go-llmprovider-sdk's OQ4 is answered (Phase 3);
  go-llmprovider-sdk's own code.

## Rules for Every Phase

* **Re-ground first.** Each phase starts by re-reading every `path:line` the
  MADR cites at the phase's start commit, and recording each moved line in the
  execution record. A citation whose text is gone is a stop.
* **Red, then green.** A test marked red runs at the phase's base and must
  fail on its assertion; one that does not compile at the base is red by its
  plant. After the change, every test passes.
* **Plants.** Each plant is the one production change named, made on a
  scratch copy, never in the tree; each must turn its test red; the execution
  record quotes each failing line.
* **No test reads the embedded snapshot for a value it asserts** (0065-PLAN's
  rule): each test hands its own models.json and serves its own listings with
  `httptest`.
* **A sent limit is asserted on a Messages route** (Claude, a test server),
  where every answer of the SDK's OQ1 sends one; an optional wire is asserted
  only where a row is about it.
* **Heavy runs.** `make preflight`, `make race` and `go test ./...` pick a
  host first (`~/.agents/skills/available-dev-hosts/scripts/pick-host`) and
  run there through `scripts/run-on-host` (with `--race` for `make race` and
  `make preflight`), or on the session host through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`, one
  at a time per host. A single package's `go test` without `-race` may stay
  on the session host.
* **No live run.** Every test serves its own replies.
* **Stop conditions**, each a stop and a prompt to the owner: an existing
  test or golden needs a change no row names; S23's test is red after the
  change; a row cannot be met as written; a third build pass of one phase.
* **Exit criterion of every phase:** no blocker, every row tested, plants
  caught.

## Implementation Steps

### Phase 0: records

1. **Re-ground** the MADR's citations at HEAD, and go-llmprovider-sdk's 0089
   and 0088 state.
2. Record go-llmprovider-sdk's OQ4 answer as rows S32 to S34, with their tests
   from Phase 3's table.
3. The two index rows in `docs/README.md`; the dated notes in 0065's pair are
   written (2026-11-22).
4. The MADR goes to `accepted` on the owner's word; this PLAN to
   `in-progress` when approved.

Exit criterion: no blocker, every row tested, plants caught (here: every
decided row names its test and plant, and none is open).

### Phase 1: discovery, the session's wait, and the facts' sources

1. **Re-ground**, and read at the base: `llm/provider/model.go`, `New` and
   `Options`, the adapter, `agent.Config` and the first call, `acpserver`'s
   session create, load and close paths, `setConfig`, `configOptions`,
   `cmdContext`, `autocompact`, the CLI wiring and headless runs,
   `internal/cli/refusal.go`, `modelscmd.go`, `llm/catalog/modelsjson.go`.
2. **Tests first**:

   | Test | Rows | What it does | At the base | Plant, and why it turns red |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestSession_DiscoversBeforeFirstCall_0066` | S1, S3, S4, S9 | ACP new session, CLI start and a headless run on a Claude test server whose listing answers after 200 ms; the server records the order of requests: gobble's listing request comes before the turn, and the turn carries the listing's 128,000; the second turn has no listing request before it | does not compile | (a) no wait: the turn reaches the server before the listing request completes; (b) a wait per call: a listing request before the second turn |
   | `TestSession_LoadDiscovers_0066` | S2 | a recorded session loaded: its model's listing request precedes its first turn | does not compile | discovery only on create |
   | `TestSession_TitleModelDiscovered_0066` | S1 | a title model on another provider: both discovered at start | does not compile | the title model left out |
   | `TestSetConfig_SwitchDiscovers_0066` | S5, S6, S41 | a switch to another Claude model, then to another provider's Claude-shaped Messages model: each next turn preceded by its listing request; a `/thinking` change: no listing request | does not compile | (a) no discovery on a switch; (b) discovery on a thinking change |
   | `TestSession_DiscoveryFails_0066` | S7, S8 | the listing answers 503; then stalls past the SDK's bound: the turn is sent with what is cached | does not compile | the failure ends the turn |
   | `TestSession_CancelAndCloseWhileWaiting_0066` | S40 | a cancel during the wait: the turn ends cancelled, as today; a close during the wait: the discovery's context is cancelled (the listing server sees the request dropped) | does not compile | (a) the wait outlives a cancel; (b) the discovery outlives the session |
   | `TestProvider_DiscoverySharesListingKey_0066` | S23 | Claude with a workspace: discovery, then a turn: one listing request in all, and the turn carries its value | does not compile | discovery built without the run's cache: two listing requests |
   | `TestProvider_ConcurrentSessionsDiscovery_0066` | S24 | under `-race`, 16 Claude sessions, each its own models.json stating 4096 or 2048 for one model id: each turn carries its own | does not compile | the layer held in a package variable |
   | `TestProvider_SnapshotNotHanded_0066` | S25, S26, S27 | discovery and the provider receive no layer named "models.dev" for any provider; `Limits.Output.Evidence` holds no gobble snapshot source | does not compile | the snapshot handed as a layer |
   | `TestProvider_ModelsJSONIsCaller_0066` | S28 | Claude, models.json `maxTokens` 20,000 over a listing of 128,000: 20,000, source "models.json" | does not compile | models.json handed under the name "models.dev": 128,000 |
   | `TestProvider_PiDefaultsNotEvidence_0066` | S39 | models.json defines a model with no `maxTokens` or `contextWindow`: discovery's `Limits.Output` unknown, no "models.json" evidence; `gobble models info` shows it unknown | does not compile | discovery given the unwrapped layer: 16,384 from "models.json" |
   | `TestModel_LimitsCarrySources_0066` | S35 | `modelOf` from a `catalog.Limits`: `Limits.Output` {128,000, known, listing:claude}; `MaxTokens` 128,000; unknown: 0, not known | does not compile | `MaxTokens` from a lower-rank source |
   | `TestCompaction_WindowFromLimits_0066` | S22, S30, S31 | Claude (input 1,000,000 only): the trigger at 1,000,000 less the reserve; the ChatGPT plan's 272,000; Ollama served at 8,192: 8,192; nothing known: no trigger | red on the ChatGPT and Ollama cases | (a) the input limit ignored; (b) the trained context used for Ollama |
   | `TestSession_NoRediscovery_0066` | S37 | a session older than the listing's freshness: no listing request before a turn; the turn still carries the stored value | does not compile | rediscovery per turn |
   | `TestSession_RoomFromSDK_0066` | S29 | every gobble call (turn, summary, title) carries `MaxTokens` 0 and no ceiling | green | a ceiling set from the window: a request names one |

3. **Red run at the base**; **the code**; **green and plants**; **checks**:
   `make pre-add-check FILES="..."`, `make check-records`, `make preflight`
   (Rules); the **execution record** (pass number, red and green output, each
   plant's failing line, each check's exit status taken before any filter,
   what was not done).

Exit criterion: no blocker, every row tested, plants caught.

### Phase 2: what is shown, kept and told

1. **Re-ground** at Phase 1's commit.
2. **Tests first**:

   | Test | Rows | What it does | At the base | Plant |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestModelsCmd_TableCachedOnly_0066` | S17 | `gobble models` with a stalled listing server: returns at once, "-" for unknown | green | discovery in the table: it waits |
   | `TestModelsCmd_InfoShowsEvidence_0066` | S15, S16, S18 | `gobble models info` on Claude, models.json 20,000 over a listing of 128,000: "max-out 20,000 (models.json)", a line "listing:claude: 128,000"; a stale fact: "(stale)"; the "requires an output limit" line | red | (a) the other sources dropped; (b) the stale mark dropped |
   | `TestACP_ModelOptionDescription_0066` | S19 | cached limits: "context 1,000,000 · max-out 128,000"; unknown: "max-out unknown"; no listing request while building | red | (a) no description; (b) a listing request while building |
   | `TestACP_ContextReportsOutputLimit_0066` | S20 | after a call with `OutputLimit` {128,000, fact, listing:claude}: "output limit: 128,000, from listing:claude"; after one with none sent: "output limit: none sent, the service's default applies" | red | the line omitted |
   | `TestProvider_LogsOutputLimit_0066` | S21 | a call's debug record carries tokens, basis and source | red | the attributes removed |
   | `TestAgent_SavesLearnedCap_0066` | S12, S14 | a `WarningLimitLearned` (Kilo, organization A, 64,000): saved with its scope and time; an eleventh for the same scope drops the oldest; one for organization B kept apart; at the next start, the provider for A gets A's and not B's; one saved 25 hours earlier is dropped | does not compile | (a) the save skipped; (b) the scope ignored; (c) no expiry on load |
   | `TestAgent_UnreadableObservations_0066` | S13 | a malformed file: one warning, the provider built, none handed back | does not compile | the error ends the build |
   | `TestNotice_LimitUnknown_0066` | S10 | discovery ends with the output unknown on Kilo: one `model.limit.unknown` line over ACP, in the CLI and in headless JSON; a second turn: none | does not compile | (a) no notice; (b) a notice per turn |
   | `TestNotice_LimitFallback_0066` | S11 | a response whose basis is fallback: one `model.limit.fallback` notice | does not compile | no notice |
   | `TestNotice_LimitLearned_0066` | S12 | a learned cap: one `model.limit.learned` notice | does not compile | no notice |
   | `TestPersonality_LimitKeys_0066` | S10 to S12 | each personality has a line for each new key | red | a key missing |
   | `make check-records` and the bump's `go.mod` line | S36 | the SDK requirement is the release carrying 0088 and 0089 | green after 0065's Phase 2 | the requirement set to `v1.31.0` on a scratch copy: the discovery tests do not compile |

3. The code; green, plants, checks, execution record, as Phase 1's step 3.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 3: the override (go-llmprovider-sdk 0089-MADR OQ4)

Only the chosen option's tests are built.

| Answer | Test | Rows | What it does | Plant |
| :--- | :--- | :--- | :--- | :--- |
| OQ4 (1) | `TestConfig_NoOutputLimitKey_0066` | S32 to S34 | no settings key; ACP offers no limit option; models.json moves the limit (S28's test) | a key added: the schema test lists it |
| OQ4 (2) | `TestConfig_OutputLimitCeiling_0066` | S32, S34 | `outputLimit` 20,000 over Claude's 128,000: the SDK's `OutputCeiling` 20,000, and its resend still runs on a refusal; 1,000: refused at load, below the SDK's 1,024 | (a) the setting sent as the call's own limit: no resend; (b) the floor not checked |
| OQ4 (3) | OQ4 (2)'s test and `TestACP_OutputLimitOption_0066` | S32 to S34 | as (2), and the ACP option sets the session's ceiling | the option not applied |

Then the code; green, plants, checks, execution record.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 4: planted breaks, the gates and the docs

1. **Re-ground** at the last build commit.
2. Every plant of Phases 1 to 3 again, on one scratch copy, each caught, each
   failing line quoted.
3. `make race` and `make preflight` on the picked host (Rules), and `go test
   ./...` there.
4. Docs; `make check-records` and `markdownlint` on the changed files.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 5: an independent review

1. A reviewer who did not build the phases reads the MADR, this PLAN and the
   diff, not the builder's summary, and reports by row.
2. A finding that changes or adds a row goes to the owner in one prompt,
   three options each, the recommended first; the ruling is a row in the MADR
   and a dated deviation here, and the phase it touches runs again as a new,
   numbered pass.

Exit criterion: no blocker, every row tested, plants caught.

## Verification

* Every decided row has its test green and its plant caught, quoted in the
  execution record; each test marked red was seen red at its base.
* `make preflight` passes at each phase's commit.

## Rollout and Rollback

* Rollout: in one gobble release with 0065's Phases 1 and 2 (MADR D12).
* Rollback: revert the phase's commit; `limit-observations.json` is left in
  the state directory and read by nothing; its entries expire in a day.

## Execution Record

No phase has run.

### Drafting (2026-11-22)

* First draft: from go-llmprovider-sdk's 0089 pair and 0088's fourth
  amendment, this repository at dfaa1ac and go-llmprovider-sdk at 751e970,
  read only; dated notes added to 0065's pair.
* Second draft, after record review 3: each finding re-verified (the MADR's
  note at its head). Records only: no code, test, index row, staging or
  commit.

*(2026-11-22, the owner's answers: Phase 3 builds OQ4 (1)'s `TestConfig_NoOutputLimitKey_0066` only. The SDK release this PLAN needs is the one after the next tag, which carries 0088 and 0089 together (go-llmprovider-sdk 0088 OQ18 "Five now; 0088+0089 next (Recommended)"); this PLAN follows 0065's Phase 2 bump to it and ships with 0065's Phases 1 and 2.)*
