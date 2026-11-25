---
status: accepted
date: 2026-11-22
decision-makers: repository owner
consulted: "the owner's request of 2026-11-22, relayed by the coordinator; go-llmprovider-sdk's 0088-MADR-default-output-limit-from-model-facts.md (accepted 2026-11-21 with the owner's eight answers, reopened `proposed` 2026-11-22 by its research amendment, uncommitted); go-llmprovider-sdk's docs/reports/0088-REPORT-output-limits-across-agents-and-vendors.md (2026-11-22); independent record review 1 of 2026-11-22, each finding re-verified against the code before it was written in; this repository at 0589454, read only; go-llmprovider-sdk at 18e8e72, read only; the read-only checkouts kilocode at 58072bd6, opencode at 4ce137b3, codex at 57e64682 and grok-build at 2bdd1d6a; Anthropic's, OpenAI's and xAI's pages named in the report, read 2026-11-22"
informed: go-llmprovider-sdk (its 0088 amendment references this record's ownership table); magic-cli-remote, which shows gobble's stop reasons over ACP
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Let the SDK Choose Each Agent Turn's Output Limit from gobble's Own Catalogue, and Keep the Call Kinds, the Cut Answer and the Notices in gobble

Associated PLAN: [0065-PLAN-output-limit-ownership-between-gobble-and-the-sdk.md](0065-PLAN-output-limit-ownership-between-gobble-and-the-sdk.md)

*(2026-11-22, second draft after record review 1: the request's layers state only the output limit and the window (D2, R41); whether the snapshot outranks a live listing is a question (Q4, R42); compaction gives a reasoning summary the turn's limit (D4); the cut-answer mappings are keyed on the error's kind, its reason and what was shown, Responses included (D5 to D7); the ChatGPT routes (R43); R35 is open under Q1.)*

*(2026-11-22, round 2, after the owner's direction on go-llmprovider-sdk's OQ9, "i do not want to have to limit anything by hardcoding heuristics into my gobble app, or the sdk, if at all possible": every number gobble used for an output limit was traced (section "Every number gobble keeps, with its provenance"); the title and the compaction summary now send no limit at all (D3, D4, from opencode's and pi's evidence); a cut answer's complete tool calls run once the SDK tag carrying 0088 lands, since the owner put the SDK's cut-answer change inside 0088 (its OQ11, D7); Q1 is reworked so that gobble adds no arithmetic of its own unless the owner chooses it; Q3's costs follow the SDK's OQ9.)*

*(2026-11-22, third draft, after record review 2: the title stream is cancelled once its line is read (D3); the summary follows Q1's rule as a turn does, and Phases 1 and 2 ship together (D4, D12); the cut-answer mapping covers Gemini's Interactions (D5); the snapshot's rank against a live listing is decided by go-llmprovider-sdk's 0041-MADR D1 rather than asked (D14, R42); whether the snapshot counts as gobble's own statement is go-llmprovider-sdk's OQ17 (R45); Q1 gains pi's per-request fit; three questions remain.)*

*(2026-11-22, fourth note: the owner answered go-llmprovider-sdk's 0088 OQ14 (3), which decides Q1 here; the owner asked for model output-limit discovery in the SDK, its 0089-MADR, and gobble's side of it is [0066-MADR-set-each-session-model-up-from-discovered-limits.md](0066-MADR-set-each-session-model-up-from-discovered-limits.md): D10, D11's reported value and R31's move there; D14, D15, R42, R44 and R45 lapse, since gobble hands the SDK no snapshot; Q3 is asked with the SDK's ceiling as go-llmprovider-sdk 0089-MADR OQ4. See "Note (2026-11-22, fourth)" at the end, revised after record review 3.)*

*(2026-11-24, note from [0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md) D12: on go-llmprovider-sdk `v1.32.0`, gobble's turns and compaction summaries now name an output limit, the catalog's cap under a 64,000 ceiling and fitted to the window (0071 D1, D2, D6). So this record's D1 "Agent turns keep `MaxTokens: 0`", R1, R2, R3, R14 and R17 are superseded for turns by 0071 D1 and D2; D4 and R7 by 0071 D6; R34 by 0071 D8 (d), whose line rides `acpext.ErrorData.outputLimitNote` (0071's amendment of 2026-11-24, the owner's "Move the line into a new field"); D3, R9 and R10 are unchanged until 0071 V8. This record's adoption amendment after `v1.33.0` decides 0071 V8 and V9.)*

## Context and Problem Statement

A large multi-file turn on Kilo's `kilo-auto/balanced` with thinking `high`
failed with an empty answer cut at the output limit: the model spent all
8192 output tokens reasoning. go-llmprovider-sdk's 0088-MADR records the
case and decides the SDK's side: when a request names no limit, the SDK will
send the model's own stated limit, capped at 32,000 on a stream and 16,384 on
a plain request, held to half the window, never below 8192, with one resend
at 8192 when a raised request is refused for its size (its OQ9 may change the
streamed ceiling and Grok's). The owner now asks which part of choosing the
output limit belongs to the SDK and which to gobble, the agent harness, and
for the most idiomatic, flexible, extensible and dynamic way to set it for any
model.

### How gobble sets the limit today (read at 0589454)

* **Agent turns name no limit.** `agent/agent.go:685` builds the request
  with no `MaxTokens`; `llm/provider/provider.go:597-599` passes it on as the
  SDK's `MaxOutputTokens: 0`, so the SDK's default applies: 8192 today.
* **gobble's catalogue knows the limit; the provider does not.** The model
  lookup adds the whole embedded models.dev snapshot as a layer, then
  models.json, over the provider's live listing (`llm/provider/model.go:108-115`,
  `:158-171`); every caller layer ranks above the listing and models.dev
  (go-llmprovider-sdk `catalog/lookup.go:28-30`). `gobble models` shows the
  result as `max-out`. The provider that sends the request is built with
  `llmprovider.WithoutModelMetadata()` (`provider.go:215`) and no layers, so
  it reads only a listing the run's cache holds (`catalog/lookup.go:167-187`).
  Only Claude's, Gemini's and Kilo's listings state an output limit (0088
  C104 to C106).
* **The snapshot.** `llm/catalog/data/models.json` (dated 2026-10-14) holds
  sections for `anthropic`, `google`, `huggingface`, `kilo`, `openai`,
  `opencode`, `opencode-go`, `togetherai` and `xai`; `kilo-auto/balanced` has
  `limit: {context: 1000000, output: 65536}`. As a layer it states nothing for
  `chatcompat` or `ollama`, which have no section key
  (go-llmprovider-sdk `catalog/model_metadata.go:368-389`).
* **The SDK reads more than the limit from facts.** Providers read the input
  modalities (10 sites; Kilo refuses an image before the network when the
  facts say the model takes none, `kilo.go:340-341`), the reasoning efforts
  (12) and the reasoning budget (3), as well as the output limit (11).
* **models.json invents two facts.** A model models.json defines without
  `maxTokens` or `contextWindow` gets Pi's defaults, 16,384 and 128,000
  (`llm/catalog/modelsjson.go:115-118`, `:459-464`), each marked known
  (`model.go:201-202`).
* **Other call kinds.** A compaction summary is sent at the session's
  thinking level (`compaction.go:511`, from `autocompact.go:176` and
  `compact.go:70`) with a limit of 80 % of `compaction.reserveTokens`
  (13,107 by default), or 50 % (8,192) for a turn prefix, held to the model's
  limit, or none when the limit is unknown (`compaction/compaction.go:309-323`,
  `:473-477`). The title asks 128 (`acpserver/title.go:32`, `:187-191`) at the
  title model's lowest level, which is not `off` for a model that cannot turn
  reasoning off (`llm/catalog/catalog.go:57-63`). The four `llm.Request`
  constructions are the turn, compaction, the title and the adapter. gobble has
  no subagents and no tool that calls a model.
* **Thinking is an effort only** (`provider.go:726-735`); 0005-MADR's
  budgets (`:108`) are not sent.
* **Room in the window.** Automatic compaction runs when the context is over
  the window less `compaction.reserveTokens`, 16,384 by default
  (`compaction.go:36`, `:47`). An overflow refusal compacts once and resends
  (`agent/agent.go:518-531`).
* **A cut answer.** Chat Completions, Messages and generateContent return
  cut text finishing `length`: gobble ends the turn `max_tokens`
  (`agent.go:366-367`; `provider.go:500-508`) with `turn.stop.max_tokens`
  (`personality/keys.go:19`; `internal/cli/view.go:500`), and ACP gets the stop
  reason `max_tokens` (`acpserver/agent.go:857`). On those wires any tool call
  in a cut answer makes it `ErrIncomplete`, reason `length`, dropping every
  call; a cut answer with only reasoning is `ErrEmptyAnswer`, reason `length`.
  On Responses every cut answer is `ErrIncomplete`, reason
  `max_output_tokens`, text, reasoning or tool call alike, and the stream
  yields each completed item first (go-llmprovider-sdk
  `responses.go:700-709`), so the adapter has already passed on any complete
  tool call (`provider.go:430-433`). gobble maps both kinds to
  `llm.ErrIncomplete` (`provider.go:745`); with text or thinking shown the
  turn ends with the raw error (`agent.go:514-517`). That is the reported
  failure's text. A compaction summary cut while reasoning fails as
  "Summarization failed: …" (`compaction.go:525-527`), which pauses the turn,
  not the session (`acpserver/autocompact.go:61-64`).
* **0004-MADR named the override and it was never built.** Its 2026-10-01
  facts list "`MaxTokens` is 8192" among the defaults gobble must override
  (`docs/decisions/0004-MADR-go-module-architecture.md:743-745`), and its
  decision 6 says "`MaxOutputTokens` per model from `llm/catalog`" (`:798`);
  no request carries it.

### What the research found (go-llmprovider-sdk's 0088-REPORT)

* The reference agents compute the limit in the harness: opencode sends the
  catalogue limit, fitted to the room the prompt leaves, per call kind, gives a
  compaction summary the model's limit with a 32,000 floor
  (`opencode:packages/core/src/session/model-request.ts:294@4ce137b3`,
  `compaction.ts:139`), and continues after a cut answer; kilocode caps the
  catalogue limit at 32,000, fits it to the window, gives Claude its whole
  limit, sends its compaction through the normal request path
  (`kilocode:packages/opencode/src/kilocode/session/compaction-chunks.ts:299-300@58072bd6`),
  and only warns on a cut answer, with a separate sentence for one cut while
  reasoning; grok-build sends the server catalogue's limit and runs the
  complete tool calls of a cut answer; codex sends none. Their provider
  libraries have no facts-based default; this SDK, after 0088, does.
* Anthropic does not count `max_tokens` against its output-per-minute limit;
  Claude 4.5 and later stop at the window, earlier models refuse; thinking is
  inside `max_tokens`. OpenAI recommends 25,000 tokens of room for reasoning
  and output. xAI's limit counts visible output only and defaults to 128,000.
  (Report WS1 to WS4, verified on 2026-11-22.)

## Decision Drivers

* One set of model facts drives what gobble shows and what it sends, and the
  freshest source wins where sources disagree.
* One owner per responsibility: no limit computed twice by rules that drift.
* The SDK's protections for a limit it chose (the half-window rule, the
  resend after a refusal of its size, 0088 A2, A4, A5) reach gobble's calls.
* A change to the output limit changes nothing else a request carries.
* A cut answer is never shown as a raw error; the user is told what happened
  and what can help.
* No regression: a turn that succeeds at 8192 today still succeeds.
* Each step is buildable before the SDK changes it depends on, and its effect
  before them is stated.

## Considered Options

* A. Delegate the limit to the SDK, and hand the provider gobble's catalogue's
  output limit and window as layers; gobble owns the call kinds, the room, the
  cut answer, the settings and the notices
* B. gobble computes every request's limit itself, as opencode and kilocode
  do, and sends it
* C. Delegate to the SDK with no layers: the SDK's listings only
* D. Send no limit on any call, as codex does

## Decision Outcome

Chosen option: "A. Delegate the limit to the SDK, and hand the provider
gobble's catalogue's output limit and window as layers", because the SDK's
0088 already implements the catalogue-limit default the research ranks
best-evidenced, with the window rule and the refusal resend a harness would
otherwise rebuild; the layers make the SDK's facts gobble's own; and
everything that needs the conversation, the session or the user stays where
that knowledge is. Option B was rejected because a value gobble computes is
"given" to the SDK, which then neither raises, fits nor resends it (0088 ST15
to ST18, ST61), so gobble would own 0088's rules twice. The decision is
`proposed`: three questions are open (Owner Questions), and rows that depend
on go-llmprovider-sdk's OQ9 to OQ12 say what holds until they are answered.

### The ownership split

| Responsibility | Owner | What it owns | Why |
| :--- | :--- | :--- | :--- |
| Model facts (output limit, window) | both, by contract | the SDK: the sources (listing, models.dev document, layers), their merge and `SourceOf`, the listing cache; gobble: its catalogue (snapshot, models.json) and which of its facts it hands the provider: the output limit and the window, stated fields only (D1, D2), and the snapshot's rank against a live listing (D14: below it, as go-llmprovider-sdk's 0041-MADR D1 ranks models.dev) | the SDK already reads layers per request; gobble owns the catalogue the user edits and sees |
| The default when a request names none | SDK | R: the stated limit, capped, held to half the window, at least 8192; 8192 when unknown (0088 A1 to A3; its OQ9) | 0088 decided it for every caller |
| Raising above the SDK's ceiling | SDK | whether a stream takes the model's own limit (0088 OQ9, ST74); gobble raises nothing itself, since a value it sets switches off the window rule and the resend | one owner for the number the wire carries |
| Wire mapping | SDK | the field on each wire, required on Messages, none on the ChatGPT routes, xAI's visible-only meaning (0088 A6, ST4, ST63) | the wires are the SDK's |
| Reasoning accounting | SDK | thinking inside the limit, the budget rule, effort to wire (0088 ST2, ST30, ST68) | gobble sends an effort only |
| Room to reason, per call kind | gobble | which calls name a limit at all: none of gobble's do (turn, summary, title), so each takes the SDK's evidence-based value (D3, D4) | the call kinds are gobble's; the number is the SDK's |
| Window fit | both | the SDK: the half-window rule; gobble: the room its conversation leaves, through compaction now and a per-request ceiling if the SDK offers one (Q1, R35) | only gobble knows the conversation's size |
| Refusal recovery | both | the SDK: one resend of a limit it raised (0088 A4, A5), and a refused cap reported if its OQ12 is answered (1); gobble: overflow, compact once and resend (as today), and keeping learned caps across runs (D10) | the SDK sees the refusal's form; gobble owns compaction and the store |
| A `length` finish | both | the SDK: a faithful report (its OQ11); gobble: what the turn does next and what the user is told (D5 to D8, Q2) | continuing, stopping and notices are session behaviour |
| User settings | gobble | models.json `maxTokens` as a fact; any limit setting (Q3) | settings are gobble's surface |
| UI and notices | gobble | personality lines, the ACP stop reason, headless output (D8) | the SDK has no user |
| Telemetry | both | the SDK: usage with reasoning tokens, the finish reason, and the limit sent if its OQ12 adds it; gobble: logs and `/context` (D11) | each reports what it alone knows |

### The decisions

* **D1. The provider carries gobble's catalogue's limit and window.**
  `llm/provider.New` takes the layers the lookup builds, through an `Options`
  field (`model.go:108-115` builds them), and adds each as
  `catalog.WithModelLayer`, wrapped by D2. Agent turns keep `MaxTokens: 0`.
* **D2. Only the output limit and the window, and only when stated.** The
  wrapper passes on `FieldMaxOutputTokens` and `FieldContextWindow` and
  nothing else, so the input modalities, efforts and budgets a request reads
  stay what they are today. The models.json layer marks a field known only
  when models.json states it; Pi's defaults (16,384 and 128,000) stay in the
  catalogue gobble shows, not in the request's facts. `chatcompat` and
  `ollama` get facts only from models.json, since the snapshot has no section
  for them, so the SDK raises there only from a caller's statement (0088 OQ7).
* **D3. Title.** No limit (0), at every thinking level, so the SDK's
  default applies; the title is cut to its length by gobble after the reply,
  as today (`acpserver/title.go`'s `maxTitle` cut). opencode sends titles no
  limit "because their reasoning is hard to budget"; grok-build's 100 and
  gobble's 128 carry no stated reason. *(Round 2: replaces the second draft's
  128 and 128 plus 4096.)* *(Third draft: gobble cancels the title's stream
  as soon as the first non-empty line has ended or has passed `maxTitle`
  runes, the length it keeps today, so a title is billed for about the line it
  uses, not for the model's whole limit; reasoning before the line is still
  billed. No new number.)*
* **D4. Compaction summary.** No limit (0), at every thinking level, so the
  SDK's default applies, with its resend; `compaction.reserveTokens` only
  decides when compaction starts. pi made this change in commit 6fcb48670,
  "remove fixed output limits from summary requests", after its summaries,
  capped at 80 % and 50 % of the reserve as gobble's are, "failed with
  'generation hit the token cap' when reasoning used most of the fixed summary
  output limit" (go-llmprovider-sdk's 0088-REPORT, round 2, E34); opencode's
  compaction also takes the model's limit. *(Round 2: replaces the second
  draft's share of the reserve with thinking `off`.)* *(Third draft: pi still
  fits each request, the summary's included, to the window (its
  `clampMaxTokensToContext`), and compaction runs when the window is nearly
  full, so a summary on a strict route would be refused first; the summary
  therefore follows Q1's answer exactly as a turn does.)*
* **D5. A cut answer with text is a `max_tokens` stop on every wire.** The
  adapter turns a cut, the SDK's `ErrIncomplete` or `ErrEmptyAnswer` whose
  reason is `length` or `max_output_tokens`, into `Done{StopMaxTokens}` with a
  cause, read from what it passed on before the error: text shown, the cause
  is "text"; this makes a Responses text cut behave as a Chat Completions one.
  *(Third draft: on `v1.31.0` Gemini's Interactions API reports any cut as
  `ErrIncomplete` with the reason `incomplete`, "measured at
  max_output_tokens" (go-llmprovider-sdk `providers/gemini/interactions.go:167`);
  the mapping includes that reason for the Gemini provider; after the bump,
  the SDK returns it as a cut answer (its ST93).)*
* **D6. A cut answer with only reasoning.** Thinking shown and no text and no
  tool call: the cause is "thinking"; the turn ends, nothing is resent
  (0031-MADR D3), and the user is told the model ran out of room while
  thinking and can lower the thinking level, or compact (`/compact`) when the
  conversation is near the model's window, since the SDK reports a stop at
  the window as the same cut (0088 C29). It does not advise a larger limit,
  which cannot take a turn past the SDK's ceiling (personality key
  `turn.stop.max_tokens_thinking`, after kilocode's warning).
* **D7. A cut answer with tool calls.** Complete calls the adapter already
  passed on (a Responses stream, today) run, and the turn goes on, bounded by
  `MaxCalls` and the bad-arguments limit as every tool round is (grok-build's
  default). Where the SDK drops them (Chat Completions, Messages,
  generateContent), the cause is "tool call" and the turn ends with
  `turn.stop.max_tokens_tool`, until the SDK delivers them. *(Round 2: the
  owner put that change inside 0088, answering its OQ11 "Inside 0088, this
  tag", so it arrives with the tag gobble adopts in Phase 2: a cut answer is
  then a response finishing `length` with its complete calls, which run, and
  the cut call is named in the SDK's `WarningToolCallCut`, which gobble logs;
  the cut call is not in the reply, so no result is written for it.)* Nothing
  shown and no call passed on: the cause is "thinking" when reasoning was
  passed on, else "tool call".
* **D8. Modes.** ACP sends the stop reason `max_tokens` as today, with the
  cause's line as an agent message; the CLI prints the line; headless print
  and JSON carry the stop reason and the line as the CLI does.
* **D9. Room in the window.** Open: Q1.
* **D10. Learned caps.** If go-llmprovider-sdk's OQ12 is answered (1), gobble
  keeps a refused cap the SDK reports and hands the caps back at provider
  construction, as it keeps reasoning refusals (`agent/warning.go:40-43`,
  0036-MADR D4).
* **D11. Telemetry.** gobble logs each call's request limit at debug (0 is
  "the SDK's default"); when the SDK reports the limit sent, gobble logs that
  and `/context` shows it.
  *(2026-11-24, note from `0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md` D19: built by 0070
  as the `limit` field of its per-call `model call` record, at debug for a healthy call and at info for one that
  failed or was retried; the value is the request's `MaxTokens`, which 0071 now sets on every turn.)*
* **D12. Ordering against the SDK.** gobble adopts the go-llmprovider-sdk tag
  that carries 0088 (the tag of six records, 0088 OQ8). D1 to D8 are built
  before it, on `v1.31.0`: with D2's wrapper the only change to a request
  there is that a stated limit below 8192 now cuts it (0041-MADR D2's cut),
  and D3 to D8 change the title, the reasoning summary's limit (on `v1.31.0`
  the SDK's 8192 instead of 13,107, until the bump raises it) and the cut
  answer's handling. The raise lands with the bump. D7's salvage on the other
  wires, D10 and R35 wait for the SDK records of its OQ11 and OQ12. *(Round 2:
  D7's salvage lands with the bump, since OQ11 is inside 0088; D3's and D4's
  zero limits make the title and summary take the SDK's 8192 on `v1.31.0` and
  its OQ9 rule after the bump; D10, R31's reported value and any ceiling wait
  for the new SDK record of its OQ12.)* *(Third draft: Phases 1 and 2 ship in
  one release, so the interval in which the summary and the title get the
  SDK's old 8192 never reaches a user.)*
* **D13. 0004-MADR.** Decision 6's "`MaxOutputTokens` per model from
  `llm/catalog`" is carried out by D1 for agent turns, through the SDK's
  default, not by a value gobble computes; 0004-MADR carries a dated
  annotation at `:745`.
* **D14. The snapshot against a live listing.** *(Third draft: decided, no
  longer asked.)* go-llmprovider-sdk's 0041-MADR D1, an accepted record, ranks
  a first-party listing above models.dev; gobble's snapshot is a models.dev
  copy, and a caller's layer ranks above the listing, so handing it as a layer
  for Claude, Gemini and Kilo would invert that ranking. gobble hands no
  snapshot facts for those three providers; their listing decides, and before
  it is cached the SDK's value for an unknown limit applies. gobble asks the
  new SDK record of its OQ12 for a caller-supplied models.dev document, ranked
  where the SDK's own is, below the listing; once it exists, gobble hands the
  snapshot there for every provider and stops handing it as a layer.
* **D15. The snapshot's name.** D2's wrapper keeps the wrapped layer's name,
  so the snapshot's facts keep the source name "models.dev"; whether the SDK
  treats them as gobble's own statement on the wires that otherwise send
  nothing (OpenAI with a key, Ollama, Grok) is go-llmprovider-sdk's OQ17
  (R45).

### Consequences

* Good, because the reported failure cannot recur on any model gobble's
  catalogue states, on every provider but the ChatGPT routes, which send no
  limit at all (R43); a stream gets the SDK's default where it had 8192
  (under the SDK's OQ9 (1), the model's own stated limit; under (2), up to
  32,000).
* Good, because the number the user sees in `gobble models` is the number the
  request is based on, and a models.json `maxTokens` moves both, within any
  bound the SDK's OQ9 answer keeps.
* Good, because gobble names no output limit of its own on any call (D3, D4):
  no number in gobble decides how much a model may write.
* Good, because the SDK's half-window rule and resend protect the turn and a
  reasoning summary, which a gobble-computed limit would switch off.
* Good, because every cut answer ends with a stop and a sentence, never a raw
  error, and a model cut while thinking is told apart.
* Neutral, because Anthropic does not count the larger limit against its
  output rate (report WS1).
* Bad, because a model no layer and no cached listing states still sends
  the SDK's value for an unknown limit (0088 ST12), and so does a session's
  first turn on Claude, Gemini and Kilo before their listing is cached (D14).
* Bad, because Pi's invented 16,384 for a models.json model without
  `maxTokens` is shown but not sent (D2).
* Bad, because on `v1.31.0`, before the bump, the summary gets the SDK's 8192
  instead of 13,107 (D4, D12); Phase 1 and Phase 2 should ship together if
  that interval matters.
* Bad, because a reasoning-only or Responses text cut of a summary changes
  from pausing the turn to pausing the session (R8).
* Bad, because between the window less R and the window less the reserve, a
  raised turn can meet a refusal or a cut answer until Q1's answer, and the
  SDK's ceiling if it comes, land.
* Bad, because three decisions wait for SDK records not yet written (D7's
  salvage on three wires, D10, R35).

### Confirmation

* The PLAN's tests, one per decided row, red at the base where a row changes
  behaviour, each with a plant caught on a scratch copy.
* `make preflight` passes, run through `run-on-host` or `run-heavy`.
* An independent review of the PLAN's last build phase finds no blocker.

### State table

"Decided" names what decides the row. Each decided row's test and plant are
in the PLAN's table, one to one. "Today" means at 0589454. "R" is the SDK's
value for a request that names no limit (0088 A2, as its OQ9 answers).

| Row | Input or state | Status | Outcome | Decided by |
| :--- | :--- | :--- | :--- | :--- |
| R1 | Agent turn on a provider whose listing states no output limit (OpenAI with a key, Grok, Together, Hugging Face, OpenCode Zen and Go); no user setting | decided | `MaxTokens` 0; the provider carries the snapshot and models.json limit and window (R41); the SDK sends R from them | D1, D2; 0088 A2 |
| R2 | Agent turn, no layer and no cached listing states a limit | decided | `MaxTokens` 0; the SDK sends its value for an unknown limit (8192 today; under its recommended OQ9, nothing on a wire that does not require a value, 8192 on Messages) | 0088 A3, OQ4; its OQ9 |
| R3 | models.json defines a model without `maxTokens` or `contextWindow` | decided | the request's facts state neither; the SDK sends 8192; `gobble models` still shows Pi's 16,384 and 128,000 | D2 |
| R4 | models.json states `maxTokens` for a model | decided | a caller-layer fact above the snapshot and the listing; the SDK raises or cuts from it, never past its ceiling and half the window, and its resend still applies | D1; 0088 A2, A4 |
| R5 | `chatcompat` and `ollama` | decided | the snapshot states nothing for them; only models.json's stated limit and window reach the SDK | D2; `catalog/model_metadata.go:368-389`; 0088 OQ7 |
| R6 | Kilo, a model the snapshot does not state, after gobble's lookup has warmed the listing in the run's cache | decided | the provider's next turn reads the same listing (one cache key) and its stated limit applies | D1; 0088 ST58 (gobble's to show) |
| R7 | Compaction summary and turn-prefix summary, at any thinking level, the model's limit known or not | decided | 0: the SDK's default, with its resend; the reserve decides only when compaction starts *(round 2)* | D4; pi 6fcb48670 |
| R8 | Compaction summary cut: with text on Chat Completions, Messages or generateContent; with text on Responses; with only reasoning | decided | the first as today, `ErrSummaryRefused`; the second and third, an error today ("Summarization failed: …", the turn paused), become `ErrSummaryRefused` through D5 and D6, so automatic compaction pauses the session (`autocompact.go:61-64`) | D4, D5, D6 |
| R9 | Title, sent with thinking `off` | decided | 0: the SDK's default; the reply cut to the title's length by gobble *(round 2)* | D3 |
| R10 | Title, the title model cannot turn reasoning off | decided | 0, as R9 *(round 2)* | D3 |
| R11 | Subagents | out of scope | gobble has none at 0589454; a later subagent record starts from R1 | |
| R12 | Tool side-calls to a model | out of scope | no built-in tool calls a model at 0589454 | |
| R13 | Model probes | out of scope | off: `WithModelProbes(false)` (`provider.go:212-216`) | |
| R14 | Agent turn at thinking `off` and each of `minimal`, `low`, `medium`, `high`, `xhigh`, `max` | decided | `MaxTokens` 0 at every level; the SDK sends the same R at every level; gobble sends an effort only | 0088 ST28, ST29; `reasoningFor` |
| R15 | A user setting for the output limit | open *(2026-11-22, fourth: moved to 0066-MADR S32, under go-llmprovider-sdk 0089-MADR OQ4)* | Q3 | |
| R16 | An ACP config option for the output limit | open *(2026-11-22, fourth: moved to 0066-MADR S33, under go-llmprovider-sdk 0089-MADR OQ4)* | Q3 | |
| R17 | Model switch mid-session | decided | facts are read per request for the request's model; a model a layer states gets R from its first turn; one only a listing states sends 8192 until the listing is cached (Claude, Gemini and Kilo models, by D14); compaction uses the new model's window (`autocompact.go:176-179`) | D1; 0088 ST54 |
| R18 | Provider switch mid-session | decided | the new provider is built as R1's, with its own layers | D1, D2 |
| R19 | The context between the window less R and the window less the reserve | open *(2026-11-22, fourth: decided by the owner's answer to go-llmprovider-sdk 0088 OQ14, option (3); this record's Q1 option (2); see the note at the end)* | Q1 | |
| R20 | An overflow refusal | decided | as today: compact once and resend; after the SDK tag carrying 0088, Kilo's documented window form is an overflow (0088 ST51): a raised turn so refused is first resent by the SDK at 8192, then, refused again, compacted and resent by gobble | `agent.go:518-531`; 0088 ST51, A4 |
| R21 | A cut answer with text on Chat Completions, Messages or generateContent | open *(2026-11-22, fifth: decided by Q2 "Stop with a notice (Recommended)"; see the note at the end)* | the turn ends `max_tokens` with the cause's notice; gobble does not continue on its own | Q2 |
| R22 | A cut answer with text on Responses (OpenAI with a key, OpenCode's responses route, the ChatGPT routes) | decided | the turn ends `max_tokens`, cause "text", its text kept, as R21 today; what follows is Q2's | D5 |
| R23 | A cut answer with only reasoning, on any wire | decided | the turn ends `max_tokens`, cause "thinking", with `turn.stop.max_tokens_thinking`; nothing is resent | D6; 0031-MADR D3 |
| R24 | A cut answer with complete tool calls | decided | on a Responses stream the complete calls run and the turn goes on; on the other wires the turn ends `max_tokens`, cause "tool call", until the SDK tag carrying 0088, after which the complete calls run, the turn goes on, and the SDK's `WarningToolCallCut` is logged *(round 2)* | D7; go-llmprovider-sdk 0088 OQ11 "Inside 0088, this tag" |
| R25 | A cut answer with no complete call and nothing shown | decided | the turn ends `max_tokens`, cause "tool call" ("thinking" if reasoning was passed on) | D7 |
| R26 | A raised limit refused for its size (Anthropic's `max_tokens: N > M`) | decided | the SDK resends once at 8192; gobble sees the answer | 0088 A4, ST49 |
| R27 | A limit the user set, refused for its size | open *(2026-11-22, fourth: moved to 0066-MADR S34, under go-llmprovider-sdk 0089-MADR OQ4)* | Q3 | |
| R28 | ACP mode | decided | stop reason `max_tokens` as today; the cause's line as an agent message | D8 |
| R29 | CLI interactive mode | decided | the line of `turn.stop.max_tokens`, `turn.stop.max_tokens_thinking` or `turn.stop.max_tokens_tool` | D8 |
| R30 | Headless print and JSON | decided | the stop reason and the cause's line, as the CLI | D8 |
| R31 | The limit a call was sent | decided | *(2026-11-22, fourth: the reported value and the context line move to 0066-MADR D8, S20, S21; the request's 0 logged here stays)* logged at debug: the request's value (0, the SDK's default); the SDK's reported value and its source, and `/context`'s line, when the new SDK record of its OQ12 ("New record, all three") reports it *(2026-11-24: built by `0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md` D19 as its record's `limit`, the request's `MaxTokens`; tested by 0070's `TestModelCallRecord`, T13)* | D11 |
| R32 | Usage with reasoning tokens | decided | as today (`usageOf`) | `provider.go` `usageOf` |
| R33 | Anthropic's output-per-minute limit | out of scope | `max_tokens` is not counted (report WS1) | |
| R34 | A gateway that reserves credit from the limit | out of scope | through `chatcompat` only models.json can raise (R5), and a 402 is `llm.ErrQuota`; through Kilo an upstream's 402 arrives as 503 (0088 C96), `llm.ErrUnavailable`, as today | |
| R35 | A per-request ceiling from the conversation's size | open *(2026-11-22, fourth: decided as R19: no ceiling from gobble)* | Q1 (third draft: its option (1) sends one from measured counts once the SDK accepts it; (2) and (3) send none) | |
| R36 | A token-counting endpoint before each turn | out of scope | an extra request per turn; the measured usage plus an estimate serves (R35) | |
| R37 | The SDK version | decided | D1 to D8 build on `v1.31.0` with the effects D12 states; the raise lands with the bump to the tag carrying 0088 | D12 |
| R38 | 0004-MADR decision 6 and its "defaults gobble must override" | decided | carried out for agent turns by D1; a dated annotation at `:745`, whose link `make check-records` checks | D13 |
| R39 | Concurrent sessions in one process, each with its own models.json | decided | each provider is built per session with its own layers; nothing is shared between sessions' layers | D1; one provider per ACP session (0004-MADR decision 6); 0088 ST46 |
| R40 | A continuation streak after cut answers | open *(2026-11-22, fifth: decided by Q2 "Stop with a notice (Recommended)": no automatic continuation, so no streak)* | Q2 | |
| R41 | What the request's layers state | decided | the output limit and the window only; image refusal, efforts and budgets read what they read today | D2 |
| R42 | Claude, Gemini and Kilo, whose live listing states the limit and the window, and the snapshot states them too | decided | *(2026-11-22, fourth: superseded by 0066-MADR S25: gobble hands the SDK no snapshot for limits; the SDK's own registry tiers rank below the listing)* *(third draft)* no snapshot facts are handed for them; the listing decides; before it is cached, the SDK's value for an unknown limit; with the new SDK record's document slot, the snapshot is handed there, below the listing | D14; go-llmprovider-sdk 0041-MADR D1 |
| R44 | The snapshot's source name through D2's wrapper | decided | *(2026-11-22, fourth: lapses, 0066-MADR S26: no snapshot is handed)* "models.dev", unchanged | D15 |
| R45 | The snapshot's limit on OpenAI with a key and on Grok | open *(2026-11-22, fourth: lapses, 0066-MADR S27 decided: no snapshot is handed)* | go-llmprovider-sdk's OQ17: under its (1) the SDK sends nothing there whatever the snapshot says; under (2) it sends the snapshot's number (30,000 on `grok-4.3`; 272,000 on `gpt-5-pro`); under (3) gobble withholds the snapshot for those providers | go-llmprovider-sdk OQ17 |
| R43 | OpenAI's ChatGPT backend and ChatGPT plan | decided | no limit is sent, whatever the facts say (0088 ST4); their cut answers follow R22 to R25 | 0088 ST4; D5 to D7 |

Counts: 29 decided, 6 out of scope, 8 open (43 rows). *(Third draft: R42 decided, R44 added decided, R45 added open: 31 decided, 6 out of scope, 8 open, 45 rows.)*

## Pros and Cons of the Options

### A. Delegate to the SDK, with gobble's limit and window as layers

* Good, because one rule, the SDK's, sets the limit, with its window rule and
  resend; gobble keeps only what needs the session.
* Good, because the facts gobble shows are the facts the request uses, and
  nothing else a request reads changes (D2).
* Good, because a later SDK addition (the ceiling, learned caps, the report)
  reaches gobble without a second implementation.
* Neutral, because the dynamic window fit needs an SDK addition (its OQ12);
  until then compaction keeps the room (Q1).
* Bad, because gobble cannot see the number sent until the SDK reports it,
  and cannot raise past the SDK's ceiling (0088 OQ9 owns that).

### B. gobble computes every limit

* Good, because it is what opencode and kilocode do, and gobble knows the
  conversation's size for an exact fit.
* Bad, because the SDK treats a computed value as the caller's own: no raise,
  no half-window rule, no resend (0088 ST15 to ST18, ST61), so gobble must
  rebuild 0088's refusal forms and resend.
* Bad, because two rules would govern one request path, and drift.

### C. Delegate with no layers

* Good, because nothing changes in gobble once the SDK tag lands.
* Bad, because only Claude's, Gemini's and Kilo's listings state a limit:
  OpenAI, Grok, Together, Hugging Face, OpenCode and every Chat Completions
  server keep 8192, and the reported failure stays on them.

### D. Send no limit at all

* Good, because the service's own default applies, as in codex.
* Bad, because Messages requires a limit, so the SDK sends one anyway, and
  what Kilo and the Chat Completions hosts default to was not established.

## Every number gobble keeps, with its provenance

*(Round 2, 2026-11-22.)* The owner's direction is that no output limit come
from a picked number. Each number gobble used or proposed for an output
limit, and what becomes of it:

| Number | Where | Provenance | Now |
| :--- | :--- | :--- | :--- |
| 128 | the title's limit (`acpserver/title.go:32`) | none stated | gone: the title names no limit (D3) |
| 4,224 (128 + 4096) | the second draft's title limit | the SDK's default thinking budget, itself unmeasured | gone (D3) |
| 13,107 and 8,192 (80 % and 50 % of the reserve) | the summary's limit (`compaction.go:309-323`) | Pi's former caps; pi removed them for the failure they cause (0088-REPORT E34) | gone: the summary names no limit (D4) |
| 16,384 | `compaction.reserveTokens`' default (`internal/config/keys.go:107`) | Pi's default, no stated reason (0088-REPORT E35) | kept as a compaction setting only: it decides when compaction starts, not any output limit (pi's own ruling, E34); a user may change it |
| 20,000 | `compaction.keepRecentTokens`' default | Pi's default, no stated reason | kept; it sets what compaction keeps, not an output limit; out of this record's scope |
| 16,384 and 128,000 | Pi's defaults for a models.json model without `maxTokens` or `contextWindow` | Pi's, no stated reason | shown in `gobble models` only; never handed to the SDK as facts (D2) |
| 4 bytes a token | Q1 (1)'s estimate of the messages added since the last measured count (third draft) | grok-build's, codex's and opencode's practice, not measured; applied only to the added part | only if the owner picks Q1 (1) |
| 1.15, 1 %, 1,024 | the second draft's per-request ceiling | opencode's comments, unmeasured | gone (third draft) |
| 4096 | the SDK's default thinking budget | the SDK's, cited to its 0062-MADR | not gobble's; gobble sends an effort only |

## Owner Questions

Three questions, each with three options, the recommended one first, each
with its cost and the rows it decides. *(Third draft: Q1 is reworked with
pi's per-request fit; Q4 is decided by go-llmprovider-sdk's 0041-MADR D1 and
no longer asked (D14); Q2 stands; Q3's costs name the SDK's OQ17.)* "The
SDK's limit" is what go-llmprovider-sdk sends when a call names none: its
OQ9, OQ14 and OQ15, still open, decide it (under the recommended answers, the
model's own stated limit, never above its window; nothing on Grok, OpenAI with
a key, Ollama and chatcompat unless gobble states a limit; nothing where no
limit is stated on a wire that does not require one). gobble always streams.
"The reserve" is `compaction.reserveTokens`, 16,384 by default: gobble
compacts once the conversation is within that many tokens of the window.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| Q1. On routes that do not fit an answer to the window themselves (Kilo and OpenRouter, Together, Hugging Face, chatcompat, Claude before 4.5), a call whose input is larger than the window less the SDK's limit is refused, then sent again smaller. pi, the harness gobble ports, avoids that by fitting every request to the room left. Should gobble fit its calls, turns and summaries alike? | (1) **Yes, from measured counts**: once the SDK accepts a per-request ceiling (its new record), gobble sends the window less the provider's own count for the last call (its input and output, which the next call carries), less an estimate of only the messages added since, at 4 bytes a token, with no spare margin; the SDK's resend covers an estimate that runs low. (2) **No**: gobble sends nothing, and the SDK's answer to its OQ14 alone decides. (3) **Compact earlier**: the trigger keeps free the limit the SDK reports it sent on the last call. | (1) one estimate, 4 bytes a token (grok-build's, codex's and opencode's practice, not measured), applied only to what was added since the last count; needs the SDK's ceiling, and until it exists (2) holds. (2) on those routes, one refused and resent call each turn whenever the input is larger than the window less the limit; whether refused calls are billed is not known. (3) compacts much sooner when the limit is large (a 200,000-token Claude 4.5 model with a 64,000 limit at 136,000 instead of 183,616) and needs the SDK's report of the limit sent. | R19, R35 |
| Q2. When a turn's answer is cut at the output limit with text, does gobble continue on its own? | (1) **No: stop with the notice** (kilocode, codex, grok-build's default). (2) **Continue automatically**: a hidden user message, "Your last response hit the output token limit. Do not apologise, recap or repeat yourself. Break the remaining work into smaller pieces.", at most two in a row, then stop with the notice (opencode). (3) **Ask**: in the CLI and over ACP, offer "Continue" to the user; headless stops. | (1) none; the user sends "continue" by hand. (2) up to two more billed calls per cut answer; a message in the session the user did not type, written in Pi's format as a user message; a setting to turn it off. (3) a prompt in the CLI and an ACP extension or a plain message the client must show; headless differs from the other modes. | R21, R22, R40 |
| Q3. How does the user override the limit? | (1) **models.json only**: `maxTokens` for a model (built-in or defined) is the model's limit, which the SDK reads as gobble's own statement (R4); no other setting. (2) **models.json and a global setting `outputLimit`**, sent as the call's own limit, held to the model's stated limit. (3) **(2) plus an ACP select option per session** (auto, 16k, 32k, 64k, the model's limit). | (1) no new surface; a stated `maxTokens` is sent as stated, within the window, on every wire, OpenAI, Grok and Ollama included, under the SDK's recommended answers (held to 32,000 under its OQ9 (3), to half the window under its OQ14 (2)); it also changes what `gobble models` shows; whether the built-in snapshot counts the same way is the SDK's OQ17. (2) one settings key and its docs; the set value is the caller's own, so the SDK neither fits it nor resends it, and a refusal ends the turn naming the setting (R27); one number for every model is itself a picked number. (3) as (2), and a config option every ACP client shows, with a value that caps or raises every model in the session. | R15, R16, R27 |

## More Information

### Relation to other records

* go-llmprovider-sdk `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md`:
  the SDK's default this record delegates to; its amendment of 2026-11-22
  references this record's ownership table; its OQ9 (the streamed ceiling and
  Grok), OQ10 (Claude before 4.5), OQ11 (the cut answer) and OQ12 (the report,
  the ceiling, learned caps) bound D7, D10, R31, R35 and Q1's costs.
* go-llmprovider-sdk `docs/reports/0088-REPORT-output-limits-across-agents-and-vendors.md`:
  the evidence.
* `docs/decisions/0004-MADR-go-module-architecture.md`: decision 6 and the
  2026-10-01 defaults (D13).
* `docs/decisions/0005-MADR-v1-feature-scope.md`: the thinking budgets
  (`:108`), not sent; the catalogue (`:177`).
* `docs/decisions/0031-MADR-sdk-current-releases-and-adoption.md` D3: a
  call that showed text or thinking is not sent again (D6).
* `docs/decisions/0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md`:
  learned reasoning refusals kept across runs (D10's pattern).

### Review findings not taken as stated

* Record review 1 listed `FieldTools` and `FieldStatus` among the facts a
  request reads; at 18e8e72 they are read only by the SDK's internal catalogue
  tool, not on a request path. D2's wrapper excludes them anyway.
* The first draft's D2 said the snapshot handed to `chatcompat` and `ollama`
  would count as a caller's statement; it cannot, as the snapshot has no
  section for them (`catalog/model_metadata.go:368-389`). R5 says so.

### What could not be established

* Whether gobble's lookup and its provider compute the same Kilo listing
  key; the PLAN's R6 test shows it (0088 ST58).
* The text of Claude's refusal on models before 4.5 (the SDK's OQ10).
* Whether continuation (Q2 (2)) improves outcomes: no source measures it.
* What Kilo and the Chat Completions hosts default to when no limit is sent.

## Note (2026-11-22, fourth): what this record keeps and what moves to 0066-MADR

Additions only. go-llmprovider-sdk's 0089-MADR-model-output-limit-discovery.md
(proposed) adds the discovery the owner asked for: each limit with every
source's value, kind and time, "unknown" as a state, validated refusals
remembered and passed back, and each response's output limit and its source.
gobble's use of it is [0066-MADR-set-each-session-model-up-from-discovered-limits.md](0066-MADR-set-each-session-model-up-from-discovered-limits.md).

* **Kept here:** the ownership split; D1, narrowed to models.json, and D2 (the
  stated-only wrapper, the output limit and window only, which 0066 also puts
  before discovery); D3 and D4 (the title and the
  summary name no limit); D5 to D9 (the cut answer and its notices); D12 (the
  ordering, which 0066 follows); D13 (0004's annotation); Q2 (continuation),
  still to be asked; and every row not named below.
* **Decided:** Q1, by the owner's answer to go-llmprovider-sdk 0088 OQ14,
  "we will be going with option 3, but only after the llmprovider-sdk work
  related to model discovery and output limit work is complete.": the SDK
  estimates the room itself, so gobble sends no ceiling, Q1's option (2).
  R19 and R35 are decided so; 0066-MADR D11 records it.
* **Moved to 0066-MADR:** D10 (learned caps, its D7); D11's reported value and
  the context line (its D8; the request's own 0 is still logged here); R31's
  reported part (its S20, S21); R15, R16, R27 (its S32 to S34, under
  go-llmprovider-sdk 0089-MADR OQ4, which asks Q3 with the SDK's ceiling).
* **Lapsed:** D14, D15, R42, R44 and R45: gobble hands the SDK no snapshot for
  limits; the SDK reads models.dev and embeds its own extract below the
  listings (0066-MADR D3, S25 to S27). *(Revised after record review 3; the
  first version of this note moved them to 0066 with the snapshot still
  handed.)*
* **Questions left here:** Q2 only.

Counts after this note: 33 decided (R19 and R35 added), 6 out of scope, 2 open
(R21 and R40, both under Q2), 3 moved to 0066-MADR (R15, R16, R27), 1 lapsed
(R45); 45 rows.

## Note (2026-11-22, fifth): the owner's answers

The owner answered Q2, relayed by the coordinator: "Stop with a notice (Recommended)", option (1). R21
and R40 are decided: a cut answer with text ends the turn `max_tokens` with
its notice, and gobble never continues on its own, so there is no streak; the
test is the PLAN's `TestAgentTurn_CutTextStops_0065` (Q2 (1)). The owner
also answered go-llmprovider-sdk 0089-MADR's OQ4 (this record's Q3, asked
there) "models.json only, for now (Recommended)", deciding 0066-MADR's S32 to S34, and its 0088 OQ18 "Five now; 0088+0089 next (Recommended)", so
Phase 2's bump is to the go-llmprovider-sdk release after the next tag, which
carries 0088 and 0089 together. No row of this record is open; by the records
rule it is `accepted`. Its PLAN stays `proposed` until the owner approves it.

Counts after this note: 35 decided, 6 out of scope, 0 open, 3 moved to
0066-MADR (R15, R16, R27, now decided there), 1 lapsed (R45); 45 rows.
