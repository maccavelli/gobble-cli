---
status: proposed
date: 2026-11-22
associated-madr: "0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement Letting the SDK Choose Each Agent Turn's Output Limit from gobble's Own Catalogue

Associated MADR: [0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md](0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md)

*(2026-11-22, second draft after record review 1: a test hook for the layers, so no test depends on the embedded snapshot; Q4's tests in Phase 1; the summary's limit by thinking level; the cut-answer tests on Responses and the ChatGPT plan; four requests for Kilo's window form; plants reworked for review findings T2 and T3; R38's check.)*

*(2026-11-22, round 2: the title and the summary name no limit (MADR D3, D4), so their tests assert 0; the salvage of a cut answer's complete calls moves from Phase 4 to Phase 2, since the owner put the SDK's cut-answer change inside 0088 (its OQ11); Q1's tests follow its reworked options; Phase 4 holds only what the new SDK record of OQ12 allows.)*

*(2026-11-22, third draft after record review 2: the snapshot is not handed for Claude, Gemini and Kilo (MADR D14), its name is kept (D15), and the Q4 tests become R42's and R44's; the title's stream is cancelled once its line is read; Gemini's Interactions cut is mapped; the Kilo window-form test uses Kilo's real message; Phase 2's values follow the SDK's recommended answers; Q1's tests follow its third-draft options; Phases 1 and 2 ship together.)*

*(2026-11-22, fourth note: Q1 is decided (its option (2): no ceiling from gobble), so Phase 3's Q1 rows are `TestCompaction_NoCeiling_0065` only; Q3's rows and Phase 4's `TestProvider_LogsSentLimit_0065` and `TestAgent_KeepsLearnedCaps_0065` move to [0066-PLAN-set-each-session-model-up-from-discovered-limits.md](0066-PLAN-set-each-session-model-up-from-discovered-limits.md); R42's, R44's and R45's tests are replaced by that PLAN's `TestProvider_SnapshotNotHanded_0066`; Phase 1 hands the provider models.json only, through the wrapper, and D14's step is not built (revised after record review 3). Phase 2's expected values follow go-llmprovider-sdk 0089-MADR's OQ1. See the MADR's note of that date.)*

*(2026-11-24, note from [0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md) D12: Phase 1's steps that set the turn and the summary to 0 are not built while 0071 holds; 0071-PLAN builds the interim on `v1.32.0`, and its Phase 2 follows this PLAN's Phase 2 bump.)*

## Goal

After this PLAN, no call gobble makes names an output limit of its own: the
agent turn, the summary and the title each take the SDK's choice, made from
the output limit and window gobble shows (MADR D1 to D4, round 2); nothing
else a request carries changes (R41); every cut answer ends the turn with a
`max_tokens` stop and a cause's sentence, never a raw error, on every wire
(D5 to D8), and its complete tool calls run once the SDK tag carrying 0088
lands (D7); the owner's answers to Q1 to Q3 are built; and, when
go-llmprovider-sdk ships the record of its OQ12, the limit sent is shown and
learned caps are kept (R31, D10), with a ceiling only if Q1 (2) is chosen.

The MADR is `proposed`, with eight open rows. Nothing below Phase 0 starts
before the owner accepts the MADR, answers Q1 to Q3, approves this PLAN and
says `proceed`, `execute the plan` or `do phase N`.

## Prerequisites

* The owner's answers to the MADR's Q1 to Q3, recorded in the MADR as rows; go-llmprovider-sdk's answers to its OQ9, OQ14, OQ15 and OQ17, which set the SDK's values the tests below expect.
* go-llmprovider-sdk's 0088: Phase 2 needs the tag that carries it (the tag
  of six records, its OQ8), and its OQ9 answer sets Phase 2's streamed
  values; Phase 4 needs the SDK records its OQ11 and OQ12 call for, if
  answered so. Phases 1 and 3 need neither.

## Scope

### In scope

* `llm/provider`: an `Options` field carrying the catalogue layers, built by
  the caller from the lookup's layers (`model.go:108-115`), and the test hook
  it gives; the wrapper that passes on only the output limit and the window
  (D2); the models.json layer's stated-only form for requests; no snapshot facts
  for Claude, Gemini and Kilo (MADR D14); the adapter's mapping of every cut answer to a
  `max_tokens` stop with a cause (D5 to D7); the debug log (D11).
* `compaction`: the summary names no limit (D4; `replyCap` and the shares go).
* `acpserver/title.go`: the title names no limit; its reply is cut as today (D3).
* `agent`: the `End` event's cause of a `max_tokens` stop.
* `acpserver`, `internal/cli` (the view, headless print and JSON): the notices.
* `personality`: `turn.stop.max_tokens_thinking` and
  `turn.stop.max_tokens_tool`, with lines in `boring`, `dry`, `snarky` and
  `bombastic`.
* `go.mod` and `go.sum`: the bump to the go-llmprovider-sdk tag carrying
  0088, in Phase 2 (the module is already required).
* What Q1 to Q3 add, and what the SDK's OQ12 record allows.
* Docs: `docs/architecture.md`, `README.md`'s models.json and settings
  sections, the personality docs.
* Records: the dated annotation in
  `docs/decisions/0004-MADR-go-module-architecture.md` at `:745` (written with
  this pair, 2026-11-22), and the two index rows in `docs/README.md`.

### Out of scope

* Subagents, tool side-calls and probes (MADR R11 to R13); Anthropic's
  output rate (R33); credit reservation (R34); token-counting endpoints (R36).
* Any change in go-llmprovider-sdk: its records are its own.

## Rules for Every Phase

* **Re-ground first.** Each phase starts by re-reading every `path:line` the
  MADR cites, at the phase's start commit, and recording each moved line, with
  its new number, in the execution record. A citation whose text is gone is a
  stop.
* **Red, then green.** A test marked red runs at the phase's base and must
  fail on its assertion; one marked green must pass there. A test that does
  not compile at the base (it uses the new `Options` field) is red by its
  plant. After the change, every test passes.
* **Plants.** Each plant is the one production change named, made on a
  scratch copy of the tree, never in the tree, and each must turn its test
  red. The execution record quotes each failing line.
* **No test reads the embedded snapshot for a value it asserts.** Each
  provider-level test hands its own layers through the `Options` field, a
  models.dev-shaped test document standing in for the snapshot, and serves its
  own replies and listings with `httptest` (0046-MADR's hermetic rule). On
  `v1.31.0` a test shows a layer reaching the request by a cut (a stated limit
  below 8192), because that SDK never raises.
* **Heavy runs.** `make preflight`, `make race` and `go test ./...` pick a
  host first (`~/.agents/skills/available-dev-hosts/scripts/pick-host`) and
  run there through `scripts/run-on-host` (with `--race` for `make race` and
  `make preflight`), or on the session host through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`, one
  at a time per host. A single package's `go test` without `-race` may stay
  on the session host.
* **No live run.** Every test serves its own replies; none reaches a service.
* **Stop conditions,** each a stop and a prompt to the owner with evidence and
  resolutions: an existing test or golden needs a change; R6's test is red at
  Phase 1's base (the lookup and the provider do not share the listing key,
  which needs a decision this PLAN does not hold); a row cannot be met as
  written; a third build pass of one
  phase.
* **Exit criterion of every phase:** no blocker, every row tested, plants
  caught.

## Implementation Steps

### Phase 0: records

1. **Re-ground** the MADR's citations at HEAD, and go-llmprovider-sdk's 0088
   state (its OQ9 to OQ12 answered or not; its tag cut or not).
2. Record the owner's answers to Q1 to Q3 as rows of the MADR, each open row
   rewritten `decided` with its test and plant from the tables below; record
   the SDK's OQ9, OQ11 and OQ12 answers where they change a step here.
3. Add the two index rows to `docs/README.md`.
4. The MADR goes to `accepted` on the owner's word; this PLAN to
   `in-progress` when approved.

Exit criterion: no blocker, every row tested, plants caught (here: every
decided row names its test and plant, and none is open).

### Phase 1: the layers, the summary, the title, the cut answer and the notices, on `v1.31.0`

1. **Re-ground** (the rule above). Read at the base: `llm/provider.New` and
   `Options`, `modelInfoFrom`, `lookupModelWith`, `modelsJSONLayer`, `stopOf`,
   `mapErrorAt`, the stream loop at `provider.go:393`; `compaction.summariseOnce`
   and `replyCap`; `acpserver/title.go`; `autocompact.pauseOf`; `agent.End`;
   the view at `internal/cli/view.go:500`; the headless output; the
   personality keys. Name the existing test of `usageOf`.
2. **Tests first.**

   | Test | Rows | What it does | At the base | Plant, and why it turns red |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestProvider_CatalogLayers_0065` | R1, R4 | OpenAI with a key; a test document states 4096 for model X: the body's `max_output_tokens` is 4096; models.json states 2048 for X over the document: 2048 | does not compile (the `Options` field) | the layers left out of `New`: 8192 in both cases |
   | `TestProvider_UnknownLimit_0065` | R2 | a model no layer and no listing states: 8192 | green | the turn's `MaxTokens` set to 16,384 when the limit is unknown: 16,384 sent |
   | `TestProvider_LayersStateLimitAndWindowOnly_0065` | R41 | Kilo; the test document states, for the model, a limit of 4096, input `text` only and efforts `low` only: a request carrying an image is sent, not refused before the network; effort `high` is sent as today; `max_tokens` is 4096 | does not compile | the wrapper passes every field: the image request is refused before the network (`ErrUnsupported`, no request served) |
   | `TestProvider_LocalServerLayers_0065` | R5 | `chatcompat`, with the test document handed in: models.json silent: 8192; models.json states 4096: 4096 | does not compile | the models.json layer withheld from `chatcompat`: the second case sends 8192 |
   | `TestProvider_KiloListingShared_0065` | R6 | Kilo, a model the test document does not state; gobble's lookup reads a fixture listing whose entry states `max_completion_tokens` 4096; the next turn's `max_tokens` is 4096 | green expected; red is a stop (Rules) | the provider built without the run's cache: 8192 |
   | `TestCompaction_SummaryLimitByThinking_0065` | R7 | reserve 16,384; a summary and a turn-prefix summary at thinking `off` and at `high`, with L 100,000 and with L unknown: every request asks 0 | red: 13,107 and 8,192 at `off` | (a) the reserve share kept at `off`: 13,107; (b) the share kept for a turn prefix: 8,192 |
   | `TestCompaction_CutSummaryRefused_0065` | R8 | through the adapter and a test server: a Chat Completions summary cut with text: `ErrSummaryRefused`; a Responses summary with text, then `response.incomplete`: `ErrSummaryRefused`; a Chat Completions summary with only reasoning, cut: `ErrSummaryRefused`, and automatic compaction pauses the session | red on the second and third cases ("Summarization failed: …"; the turn paused) | the adapter's cut mapping removed: the third case returns "Summarization failed" and pauses the turn |
   | `TestTitle_LimitByThinking_0065` | R9, R10 | a title model that can turn reasoning off, and one whose lowest level is not `off`: both ask 0; a reply whose first line is "Fix the parser\n" followed by 2,000 more characters: the stream is cancelled after the newline, and the title is "Fix the parser"; a first line longer than `maxTitle` runes: cancelled once it passes them, the title cut to them | red: 128, and no cancel | (a) 128 kept for the model that can turn reasoning off: that case asks 128; (b) the cancel removed: the server sees the stream read to its end; (c) the cut removed: a long title |
   | `TestAgentTurn_NoLimitAtAnyLevel_0065` | R14 | at `off` and each of the six levels, the agent's request has `MaxTokens` 0 and the effort `reasoningFor` gives | green | `MaxTokens` set by level (16,384 at `high`): the `high` case fails |
   | `TestAgentTurn_ModelSwitchFacts_0065` | R17 | one provider; models.json states 4096 for model A and 2,048 for model B; a turn on A, a switch, a turn on B: 4096 then 2,048 | does not compile | `New` given a layer that answers only for `o.Model`: B's turn sends 8192 |
   | `TestProvider_SwitchBuildsLayers_0065` | R18 | a session switched from Kilo to `chatcompat`, models.json stating 4096 for the `chatcompat` model: the new provider's body states 4096 | does not compile | the switch path built without layers: 8192 |
   | `TestAdapter_CutText_0065` | R22 | OpenAI Responses stream: text deltas, then `response.incomplete` with `max_output_tokens`: the events are the text and `Done{StopMaxTokens}`, cause "text"; the turn ends `max_tokens` with the text kept | red: the turn fails with `llm.ErrIncomplete` | the mapping removed: the error |
   | `TestAdapter_CutWhileThinking_0065` | R23 | Chat Completions: reasoning deltas, then `length` and no text (the SDK's empty answer); Responses: reasoning deltas, then `response.incomplete`; Gemini's Interactions: reasoning, then status `incomplete` (the SDK's `ErrIncomplete`, reason `incomplete`): each ends `max_tokens`, cause "thinking"; one send each | red: the raw error | (a) the mapping matched only to `ErrEmptyAnswer`: the Responses case is the raw error; (b) the reason `incomplete` not mapped: the Interactions case is the raw error; (c) a resend after it: two sends |
   | `TestAdapter_CutToolCall_0065` | R24, R25 | Responses: a completed `function_call` item, then `response.incomplete`: the call runs and the next model call is made; Chat Completions: a tool call, then `length`: `max_tokens`, cause "tool call", no tool runs; Chat Completions with nothing but a cut call: cause "tool call" | red: each the raw error | (a) the Responses call discarded on the cut: no tool runs; (b) the mapping removed: the raw error |
   | `TestAdapter_ChatGPTPlanCut_0065` | R43 | the ChatGPT plan route, models.json stating 4096: the body carries no `max_output_tokens`; a cut with text ends `max_tokens`, cause "text" | red on the cut | the mapping removed: the raw error |
   | `TestACP_MaxTokensNotice_0065` | R28 | over ACP: stop reason `max_tokens`, and for R23's case an agent message with `turn.stop.max_tokens_thinking`'s line | red on the notice | the notice not sent: no message |
   | `TestCLI_MaxTokensLines_0065` | R29 | the view prints the line of `turn.stop.max_tokens`, `turn.stop.max_tokens_thinking` and `turn.stop.max_tokens_tool` for the three causes | red on the two new keys | every cause printed with the old key: the thinking case's line differs |
   | `TestHeadless_MaxTokensStop_0065` | R30 | print and JSON modes carry `max_tokens` and the cause's line | red on the cause | the cause dropped from the JSON: the field is absent |
   | `TestProvider_LogsRequestLimit_0065` | R31 | a turn's debug log carries the request's limit, 0 | red | the attribute removed: absent |
   | the existing test of `usageOf` named at step 1, or `TestUsageOf_ReasoningTokens_0065` if none | R32 | a reply's reasoning tokens reach `llm.Usage.ReasoningTokens` | green | reasoning tokens dropped: 0 |
   | `TestProvider_ConcurrentSessionsLayers_0065` | R39 | under `-race`, 16 sessions in one process, each with its own models.json, half stating 4096 and half 2,048 for the same model id: each body states its own | does not compile | the layers kept in a package variable each `New` overwrites: the earlier sessions send the last session's value, or the race detector reports the write |
   | `make check-records` | R38 | the 0004-MADR annotation's link resolves | green | on a scratch copy, the link pointed at a missing file: the check fails (seen 2026-11-22, see the execution record) |
   | `TestProvider_ListedProvidersNoSnapshot_0065` | R42, R17 | Kilo; the test document states 4096 for a model, handed in as the snapshot is: a cold cache: the turn carries the SDK's value for an unknown limit, not 4096; after a fixture listing stating 2,048 is cached: 2,048. OpenAI with a key, the same document stating 4096: 4096 (the snapshot is handed there) | does not compile | the snapshot handed for Kilo too: the cold turn sends 4096 |
   | `TestProvider_SnapshotKeepsItsName_0065` | R44 | the wrapped snapshot's `SourceOf` for the output limit reads "models.dev" | does not compile | the wrapper renames its layer: "gobble" |
   | (R45 follows go-llmprovider-sdk's OQ17: under its (3) a test here, `TestProvider_SilentWiresNoSnapshot_0065`, asserts that OpenAI with a key, Grok and Ollama get no snapshot limit; under (1) and (2) the SDK's own test pins it) | R45 | | | |

3. **Red run at the base**; quote the failing lines.
4. **The code.** The `Options` field and its caller in `internal/cli` and
   `acpserver`, built from the lookup's layers; the wrapper; the stated-only
   models.json layer; D14's rule; the summary's and the title's zero limits and
   the title's cancel; the
   adapter's cut mapping, keyed on the error's kind (`ErrIncomplete`,
   `ErrEmptyAnswer`), its reason (`length`, `max_output_tokens`) and what it
   passed on (text, thinking, tool calls); the `End` cause; the two keys with a
   line in each personality; the views; the debug log.
5. **Green and plants**: every test passes; each plant red on a scratch copy.
6. **Checks**: `make pre-add-check FILES="..."`; `make check-records`;
   `make preflight` (Rules).
7. **Execution record**: the pass number, red and green output, each plant's
   failing line, each check's exit status taken before any filter, and what
   was not done.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 2: the SDK tag carrying 0088

1. **Re-ground** at Phase 1's commit, and read the tag's 0088 code (the
   default, `Settings.MaxTokensGiven`, the resend, Kilo's window form) and its
   answers to OQ9, OQ14 and OQ15: the values below are those of its
   recommended answers (the stated limit, never above the window, no fixed
   ceiling); under its OQ9 (3) a stream is held to 32,000, under its OQ14 (2)
   to half the window. Phase 1 and Phase 2 ship in one release (MADR D12).
2. **Tests first**, red at Phase 1's commit (the old SDK):

   | Test | Rows | What it does | At the base | Plant |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestProvider_CatalogLayers_0065`, raise cases | R1, R37 | the test document states 65,536 for a Together model and a window of 131,072: a turn streams `max_tokens` 65,536; for an OpenAI model, 128,000: no `max_output_tokens` under the SDK's OQ9 (1) and OQ17 (1), 128,000 under its OQ17 (2) | red: 8192 | the layers left out: 8192 on Together |
   | `TestModelsJSONLayer_StatedOnly_0065` | R3 | a models.json model with no `maxTokens`: the turn sends 8192; `gobble models info` shows 16,384 | green | Pi's defaults handed to the provider: 16,384 sent |
   | `TestCompaction_SummaryLimitByThinking_0065`, raise case | R7 | thinking `high`, models.json stating 65,536 for a Kilo model: the summary's request carries the SDK's value for a call that names none (65,536 under its OQ9 (1); 32,000 under (2)) | red: 8192 | the summary given the reserve share: 13,107 |
   | `TestAgentTurn_SalvageCompleteCalls_0065` | R24 | Chat Completions: a cut answer with a complete call and a cut one: the complete call runs, the next model call is made, and the SDK's `WarningToolCallCut` is logged; no result is written for the cut call | red: the turn ends `max_tokens`, cause "tool call" | (a) the complete call dropped: no tool runs; (b) a result written for the cut call: the next request carries a result with no matching call |
   | `TestAgentTurn_KiloWindowFormCompacts_0065` | R20 | models.json states 65,536 and a window of 200,000 for a Kilo model; the server refuses a turn with HTTP 400 and Kilo's documented message, "This request exceeds the model's context window of 200000 tokens. Your request contains approximately 300000 tokens.", and answers a summary: the input alone (300,000 less 65,536) is over the window, so the SDK does not resend; three requests: the turn refused, the summary answered, the turn resent and answered | red: the turn fails `ErrInvalidRequest` after one request | (a) the overflow branch removed: the turn fails after one request; (b) the turn resent before compacting: the second request is a turn |
   | `TestAgentTurn_RaisedLimitRefusedResent_0065` | R26 | Claude, models.json stating 128,000; the server refuses any `max_tokens` above 8192 with `max_tokens: 32000 > 8192`: two sends, the second at 8192, the turn answered | red: one send, at 8192, accepted | the turn's `MaxTokens` set to the stated limit: one refused send |

3. **The change**: `go get` the tag; `go mod tidy`; nothing else in the code.
4. Green, plants, checks and the execution record as Phase 1's steps 5 to 7.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 3: the owner's answers to Q1 to Q3

1. **Re-ground** at Phase 2's commit.
2. **Tests first**, by the answer the owner gave (only the chosen option's
   tests are built):

   | Answer | Test | Rows | What it does | Plant |
   | :--- | :--- | :--- | :--- | :--- |
   | Q1 (1) | `TestAgentTurn_CeilingFromMeasured_0065` (Phase 4, once the SDK accepts a ceiling) | R19, R35, R7 | a 200,000-window model; the last call's reported input 150,000 and output 20,000; 2,000 bytes added since (500 tokens at 4 bytes): the turn's ceiling 200,000 − 170,000 − 500 = 29,500; a summary at the same point gets the same ceiling (red at its base) | (a) the output not counted: 49,500; (b) the estimate applied to the whole conversation instead of the added part: a ceiling far below 29,500 |
   | Q1 (2) | `TestCompaction_NoCeiling_0065` | R19, R35 | a 200,000-window model whose stated limit is 64,000, at 183,000 tokens: no compaction, and no request names a limit or a ceiling | a ceiling sent: a request names one |
   | Q1 (3) | `TestCompaction_ReservesReportedLimit_0065` (Phase 4, once the SDK reports the limit sent) | R19 | a 200,000-window model whose last call the SDK reports sent 64,000: compaction runs at 136,001 tokens, not at 136,000 | the reported limit ignored: no compaction until 183,617 |
   | Q2 (1) | `TestAgentTurn_CutTextStops_0065` | R21, R22, R40 | a cut text answer on Chat Completions and on Responses ends the turn `max_tokens`; one model call | a second call made: two |
   | Q2 (2) | `TestAgentTurn_CutTextContinues_0065` | R21, R22, R40 | a cut text answer: the hidden message is sent and the turn goes on; three cut answers in a row: two continuations, then `max_tokens`; the setting off: one call | (a) no continuation: one call; (b) no streak limit: a fourth call |
   | Q2 (3) | `TestCLI_CutTextOffersContinue_0065` and `TestACP_CutTextOffersContinue_0065` | R21, R22, R40 | the CLI and ACP offer "Continue"; headless stops | the offer not made |
   | Q3 (1) | `TestProvider_CatalogLayers_0065` (R4's case) and `TestConfig_NoOutputLimitKey_0065` | R15, R16, R27 | models.json `maxTokens` moves the request's limit; the settings schema has no output-limit key; ACP offers no such option | a key added to `internal/config/keys.go`: the schema test lists it |
   | Q3 (2) | `TestConfig_OutputLimit_0065` | R15, R16, R27 | `outputLimit` 20,000 with a stated limit of 16,000: 16,000 sent; refused for its size: the turn ends with the error naming the setting; no config option | the hold to the stated limit removed: 20,000 sent |
   | Q3 (3) | Q3 (2)'s test and `TestACP_OutputLimitOption_0065` | R15, R16, R27 | as (2), and the ACP select option sets the session's limit | the option not applied: R sent |

3. The code for the answers; green, plants, checks, execution record.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 4: what the SDK's OQ12 record allows (round 2: OQ11 is inside 0088, so its part is in Phase 2)

1. **Re-ground** at Phase 3's commit, and read the SDK tag's new API. A dated
   deviation here lists each exported name gobble calls, before code.
2. **Tests first**:

   | Test | Rows | What it does | Plant |
   | :--- | :--- | :--- | :--- |
   | `TestAgentTurn_SalvageCompleteCalls_0065` | R24 | moved to Phase 2 (round 2) | as in Phase 2 |
   | `TestAgentTurn_CeilingFromMeasured_0065` (third draft: only with Q1 (1), once the SDK accepts the ceiling) | R35 | as Phase 3's Q1 (1) row | as there |
   | `TestProvider_LogsSentLimit_0065` (OQ12, any option) | R31 | the SDK's reported limit is logged and `/context` shows it | the report ignored: 0 logged |
   | `TestAgent_KeepsLearnedCaps_0065` (OQ12 (1)) | D10, R26 | a reported refused cap is saved and handed back at the next provider's construction | the save skipped: the next provider gets none |

3. The code; green, plants, checks, execution record.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 5: planted breaks and the gates

1. **Re-ground** at the last build commit.
2. Every plant of Phases 1 to 4 again, on one scratch copy, each caught,
   each failing line quoted.
3. `make race` and `make preflight` (Rules: a picked host, `--race`), and
   `go test ./...` on the host `pick-host` prints.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 6: docs

1. `docs/architecture.md` (the provider's layers, the cut-answer stops),
   `README.md` (models.json `maxTokens` moves the request's limit within the
   SDK's ceiling; Q3's surface), the personality docs listing keys, and the
   records' index rows' status.
2. `make check-records` and `markdownlint` on the changed files.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 7: an independent review

1. A reviewer who did not build the phases reads the MADR, this PLAN and the
   diff, not the builder's summary, and reports by row: a row whose test does
   not pin it, a case the table lacks.
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

* Rollout: Phases 1 and 2 ship in one release (MADR D12, third draft): the
  provider carries gobble's limit and window, no call names a limit of its
  own, cut answers end with a stop and a sentence, and the SDK's 0088 rule
  applies.
* Rollback: revert the phase's commit. Phase 2's revert returns the SDK pin
  and the 8192 default; Phase 1's returns the raw errors of cut answers and
  the summary's reserve share at every level.

## Execution Record

No phase has run.

### Drafting (2026-11-22)

* First draft: the pair, from go-llmprovider-sdk's 0088 pair and amendment,
  its 0088-REPORT, this repository at 0589454 and the upstream checkouts, all
  read only; one dated annotation added to
  `docs/decisions/0004-MADR-go-module-architecture.md` at `:745`.
* Second draft, after record review 1: every finding re-verified before it
  was written in; two did not hold as stated (the MADR's "Review findings not
  taken as stated"). R38's check was seen to fail on a planted break: on a
  scratch copy of `docs/` and `scripts/`, with the annotation's link pointed
  at a missing file, `python3 -B scripts/check_records.py --check` exited 1
  with "broken relative link:
  docs/decisions/0004-MADR-go-module-architecture.md:746:
  0065-MADR-output-limit-ownership-missing.md"; on the tree it exits 0. The
  tree itself was not touched.
* No code, test, index row, staging or commit.
* Round 2 (2026-11-22), after the owner's direction on go-llmprovider-sdk's
  OQ9 and the owner's answers to its OQ10 to OQ12: the MADR's D3 and D4 send
  no limit, its provenance table and reworked Q1 and Q3 added; here, the
  summary and title tests assert 0, the salvage test moved to Phase 2, Q1's
  tests replaced, Phase 4 narrowed to the OQ12 record. Records only.
* Third draft (2026-11-22), after record review 2: each finding checked
  against the code before it was written in; the MADR's D3, D4, D5, D12,
  D14, D15, R2, R42, R44, R45 and its questions (Q4 decided, three left)
  changed; here, the rows named in the note at the top. Records only.

*(2026-11-22, fifth note: Q2 answered "Stop with a notice (Recommended)": Phase 3 builds `TestAgentTurn_CutTextStops_0065` for R21, R22 and R40, and not Q2's other tests; Q1's is `TestCompaction_NoCeiling_0065`; Q3's moved to 0066. Phase 2's bump is to the go-llmprovider-sdk release after the next tag, which carries 0088 and 0089 together (its OQ18 "Five now; 0088+0089 next (Recommended)"); Phases 1 and 2 still ship together, with 0066's phases. The MADR is `accepted`; this PLAN stays `proposed` until the owner approves it.)*

*(2026-11-24, note from `0070-PLAN-keep-a-failed-partial-reply-and-log-stream-health-per-call.md` Phase 2: D11's per-call request limit and R31's logged value are built there, as the `limit` field of the `model call` record in `llm/provider/record.go`; no step here builds that line again.)*
