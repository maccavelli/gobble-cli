---
status: accepted
date: 2026-11-24
decision-makers: repository owner
consulted: "the owner's rulings of 2026-11-24, \"5 now, converge to 1 (Recommended)\" and \"Yes, re-scope v1.33.0 (Recommended)\", and the owner's answers to this record's Q1 to Q4, relayed by the coordinator; an independent review of the first draft (eight findings, each re-read in the code before it was written in); this repository at 97fa1f6, read only, with one offline probe test run on a scratch clone; go-llmprovider-sdk at its tag v1.32.0 (gobble's pin, go.mod), read only, and at 33e6f09 (its main, carrying 0088 and 0089, unreleased), read only; the read-only checkouts pi at 1402973a9, opencode at 4ce137b39e, kilocode at 58072bd696 and qwen-code at 36fc21f5d8"
informed: go-llmprovider-sdk (V4 names a gap in its 0089 learning for a caller that names its limit, and the path its 0089 amendment adds); magic-cli-remote, which shows gobble's stop reasons and the prompt response's _meta
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Send Each Turn the Model's Known Output Cap, Under a Ceiling and Fitted to the Window, Now; Own the Per-Turn Limit on the SDK's Facts after v1.33.0

Associated PLAN: [0071-PLAN-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-PLAN-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md)

*(2026-11-24, first draft. Record 0070, `0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md`, accepted and not yet in the tree, holds the number before this one: `scripts/check_records.py --next` printed `0070` at 97fa1f6, and this record takes `0071`.)*

*(2026-11-24, second draft, after the owner's answers and an independent review. Owner's answers, relayed by the
coordinator: Q1 "Yes, short live calls (Recommended)"; Q2 "16,384, fitted, shown 'assumed' (Recommended)"; Q3, a new
question on a ceiling, "64,000 unless you set maxTokens (Recommended)". Review findings taken in: a cap can cause a credit or
rate refusal (F14, L15 now decided); listed caps can be wrong (F15, L36); the trigger and the fit now share one estimate,
anchored as pi's, with tools counted (F16, D3, L35, L37, L38); a floor of 1,024 and "send the cap when the window is full"
(kilocode's rule, D2); a refusal caused by the cap and not the input is resent once at the room or cap it states, before any
compaction (F17, D8, L39); the summary ladder recomputes its cap and does the same (F18, D6, L40); "always explicit" now
asked against the owner's SDK ruling (Q4, V2); the runaway bound stated; tests and plants corrected in the PLAN. The
first draft's L5, L6, L18, L33 and L34 are decided by the answers. Q4 is open, so the record stays `proposed`.)*

*(2026-11-24, amendment: the owner answered Q4, relayed by the coordinator, "Always, gobble's policy (Recommended)". V2 is
decided: every turn and summary names its limit on every wire, as gobble's own policy above the SDK's default; the SDK's
ruling "Only where required (Recommended)" (its 0089 OQ1) still governs the SDK's own choice for other callers. No row is
open; by the records rule this record is `accepted`. Its PLAN stays `proposed` until the owner approves it.)*

*(2026-11-24, amendment after Phase 1's first pass stopped: the owner's "Move the line into a new field", relayed by the
coordinator. D8 (d)'s line rides a new `acpext.ErrorData` member and not the error text; D13 gains it; L15 is amended and
L42 and L43 added. See "Amendment of 2026-11-24: D8 (d)'s line in an acpext member" under More Information.)*

*(2026-11-25, amendment: the owner's "gobble: v1.33.0 pin first, then 0072 (Recommended)". gobble pins go-llmprovider-sdk
`v1.33.0` first, with this record's Phase 2 and the adoption amendments of 0065 and 0066, then builds 0072 on it. Re-grounded
at 073fd1f and at the SDK's tag; rows P1 to P23 added; Q5 and Q6 asked. See "Amendment of 2026-11-25: pin v1.33.0 first, and
adopt it" at the end. Rows P6 and P21 are open, so that amendment is `proposed`; every row above it stands `accepted`.)*

*(2026-11-25, second amendment: the owner answered Q5, Q6 and cases 4, 5, 7, 9 and 10; P6 and P21 decided, P24 to P35 added,
stale claims corrected. No row is open. See "Amendment of 2026-11-25, second" at the end.)*

## Context and Problem Statement

The owner drives GLM-5.3 and GLM-5.3-flash at thinking `high` through gobble, mainly on Kilo and OpenCode Go, and on every
provider for testing, and keeps hitting the output limit ("incomplete response: length"), then compacting again and again.
The cause, verified by the coordinator and re-read here: gobble's main turn names no output limit, so go-llmprovider-sdk
`v1.32.0` fills its blanket default of 8,192, and the model's reasoning spends it.

The SDK's fix (its 0088 and 0089) is on the SDK's main and unreleased. The owner ruled on 2026-11-24, "5 now, converge to 1
(Recommended)": today a gobble patch sends the model's known output cap from gobble's catalog on each turn, as gobble already
does for titles and compaction; in parallel the SDK's `v1.33.0` is fast-tracked; then gobble owns the per-turn limit policy on
the SDK's facts (option 1), with probing (option 3) a later gobble feature. One new gobble record, with dated amendments to
0065 and 0066.

This record decides every case of the interim on `v1.32.0` (the L rows) and what is decided now of the convergence on
`v1.33.0` (the V rows), leaving the rest, named, to the adoption amendments of
`0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` and
`0066-MADR-set-each-session-model-up-from-discovered-limits.md`.

### Owner rulings

| Ruling, as given | Label | Rows |
| :--- | :--- | :--- |
| Send the model's known cap now; gobble owns the per-turn limit on the SDK's facts after `v1.33.0`; probing later | "5 now, converge to 1 (Recommended)" | D1, V1 to V10 |
| Cut `v1.33.0` from the SDK's main; move 0092, `WithResponses()` and 0088's R1 to R5 to `v1.34.0` | "Yes, re-scope v1.33.0 (Recommended)" | V1 |
| Q1: check each GLM cap with one short live call | "Yes, short live calls (Recommended)" | L34 |
| Q2: no source states the cap: Pi's 16,384, fitted, shown as assumed | "16,384, fitted, shown 'assumed' (Recommended)" | L5, L6, L18 |
| Q3: a ceiling of 64,000 on any cap the user did not set; an answer over 64,000 is cut until the user sets `maxTokens` for that model in models.json | "64,000 unless you set maxTokens (Recommended)" | L33, L36 |
| Q4: after `v1.33.0`, name a limit on every call, optional wires included, as gobble's own policy | "Always, gobble's policy (Recommended)" | V2 |
| *(2026-11-24)* Phase 1's stop: D8 (d)'s line goes in a new `acpext.ErrorData` member, not the error text, which stays byte-identical, so acpext's example frames stay as they are | "Move the line into a new field" | L15, L42, L43 |
| *(2026-11-24)* Phase 1's review: after an L39 resend at a stated value M, a later call of the same turn asked the full cap again, and a second refusal ended a turn that would have succeeded; every later call of the turn keeps the stated value | "Keep the stated value (Recommended)" | L44 |
| *(2026-11-24)* Phase 1b's stop: `TestPromptQuotaData` runs capped calls, so its data carries the note and cannot equal the acpext frames, which have none; its expected data is each frame with the expected note set | "Expect frame plus note (Recommended)" | L15 |
| *(2026-11-24)* Phase 1b: print mode's JSON `errorData` carries the note on a capped call's credit or rate refusal; print mode's text output is unchanged | "Test the JSON note, text as is (Recommended)" | L45 |
| *(2026-11-25)* Pin go-llmprovider-sdk `v1.33.0` first, with this record's Phase 2 and the 0065 and 0066 adoption amendments; then build 0072 on `v1.33.0`, dropping its `v1.32.0`-only steps | "gobble: v1.33.0 pin first, then 0072 (Recommended)" | P1 to P23 |

### Facts read at 97fa1f6

* **F1. The turn names no limit.** `turn.call` builds `llm.Request{Model, Messages, Tools, Thinking, Cache, Deferred}` and
  no `MaxTokens` (`agent/agent.go:685-686`); `llm.Request.MaxTokens` is "zero is the provider's default" (`llm/llm.go:82-84`);
  the adapter passes it as the SDK's `MaxOutputTokens` (`llm/provider/provider.go:598-599`).
* **F2. What 0 becomes on `v1.32.0`.** The SDK's settings default `MaxTokens` to 8,192 (go-llmprovider-sdk
  `llmprovider/settings.go:87-89` at `v1.32.0`). Kilo, OpenCode and Together send that default when the request names none,
  cut to the model's stated limit only when that is smaller (`providers/kilo/kilo.go:373-379`,
  `providers/opencode/opencode.go:446-459`, `providers/together/together.go:298-300`); a request's own value is sent as
  given. So every GLM-5.3 turn today asks for 8,192 output tokens, reasoning included.
* **F3. gobble already sends a catalog cap on two call kinds.** The title asks `min(128, info.MaxTokens)`
  (`acpserver/title.go:186-191`). Compaction builds `compaction.Model{…, MaxTokens: a.modelOf(provider, model).MaxTokens}`
  (`acpserver/compact.go:69-70`; `acpserver/autocompact.go:174-181`) and caps the summary at 80 % of
  `compaction.reserveTokens`, the turn prefix at 50 %, each held to that cap, and sends 0 when the cap is unknown
  (`compaction/compaction.go:309-323`, `:470-478`), which `v1.32.0` turns into 8,192.
* **F4. Where the cap comes from.** `a.modelOf(provider, model)` is `a.info` (`acpserver/config.go:47-48`): the embedded
  models.dev snapshot (`llm/catalog/data/models.json`, fetched 2026-10-14) and models.json as caller layers over the
  provider's cached listing (`llm/provider/model.go:107-115`, `:158-171`; go-llmprovider-sdk `catalog/lookup.go:28-30` at
  `v1.32.0`). `llm.Model.MaxTokens` and `ContextWindow` are "0 is unknown" (`llm/llm.go:159-162`); `llm.Model` does not say
  which layer gave them.
* **F5. models.json's invented cap.** A model models.json defines without `maxTokens` gets Pi's 16,384, and without
  `contextWindow` Pi's 128,000 (`llm/catalog/modelsjson.go:114-118`, `:459-463`); the layer marks both known
  (`llm/provider/model.go:198-203`). `modelOverrides` sets only what it gives (`modelsjson.go:450-453`, `:469-485`), so a
  `maxTokens` there is the override for a model gobble already knows (`docs/guides/models-json.md:70-75`).
* **F6. gobble's context estimate.** Before each call the automatic trigger estimates
  `compaction.EstimateText(a.systemText(s)) + compaction.EstimateEntries(s.tree.Path())` (`acpserver/autocompact.go:103-107`):
  chars/4 (`compaction/estimate.go:31-35`), anchored on the last valid assistant usage plus estimates of what follows it
  (`:288-333`), images at 4,800 characters (`:13-15`). It counts no tool specs, and it adds the system text even when the
  last usage already measured it. The trigger runs before each call (`agent/agent.go:324-327`) and compacts past `window −
  compaction.reserveTokens`, 16,384 by default and at least 1 (`compaction/compaction.go:34-47`;
  `internal/config/keys.go:107`). At the threshold a session `Prepare` refuses is left as it is and the call goes ahead
  (`acpserver/autocompact.go:129-135`).
* **F7. An overflow compacts once and resends** (`agent/agent.go:518-531`). On `v1.32.0` the SDK reads an overflow from the
  service's type or message (`llmprovider/context_overflow.go:19-58`), the OpenRouter form "maximum context length is N
  tokens" included (`:44`); when the message says the completion alone fills the window it is an invalid request with the
  reason "max_tokens alone fills the context window: lower it" (`:67-101`; `llmprovider/api_error.go:514`, `:651`).
* **F8. `v1.32.0` learns no output cap**: a refusal of the limit sent is an error, with no resend and no memo.
* **F9. Thinking is an effort only** (`llm/provider/provider.go:726-735`). On a Messages route the SDK raises `max_tokens` to
  the budget plus 4,096 when the limit sent is not above the budget (`internal/wire/messages/messages.go:625-655`; budgets
  4,096 and 1,024, `internal/wire/wire.go:46-53`). On Chat Completions, GLM's wire, reasoning counts inside `max_tokens`.
* **F10. The ChatGPT routes send no output limit** whatever the request asks (`providers/openai/openai.go:45-56`, `:490-513`).
* **F11. Where the limit would show.** `/context` and `/session` (`acpserver/commands.go:301-338`); the `--stats` suffix
  (`internal/cli/view.go:470-495`; `internal/cli/render/status.go:40-49`, pinned by `TestStats`,
  `internal/cli/render/plan_test.go:20-36`; headless, `internal/cli/print.go:232-237`). The prompt response already carries
  `_meta.gobble.*` members (`acpserver/commands.go:684-691`; `acpserver/agent.go:890-909`), which `acpclient` reads
  (`acpclient/conn.go:264-272`, `:340`).
* **F12. One agent per prompt, one `agent.Config`** (`acpserver/agent.go:783-803`); a model or provider switch takes effect at
  the next prompt. `Config.Compact` is `func(ctx, overflow bool)` (`agent/agent.go:64-72`), set by `acpserver` and by
  `agent/compact_test.go`'s fake.
* **F13. The SDK's `v1.33.0` rule for a caller's value** (go-llmprovider-sdk at 33e6f09): a `Request.MaxOutputTokens` is sent
  as given, basis `request`, never fitted (`internal/wire/limit_rule.go:58-61`); a refused value is not resent (`:214-216`);
  its stated cap is kept in the process memo (`:206-213`), but the `WarningLimitLearned` rides only a `Response`
  (`:239-253`). `catalog.CachedLimits` reads the memo as "observed" evidence (`catalog/limits.go:115-125`, `:193-213`);
  `Limits.StreamedOutput` is a streamed request's output (`:33-44`); `Response.OutputLimit` is a `SentLimit`
  (`llmprovider/sent_limit.go:5-47`); `WithLimitObservations` passes caps back (`llmprovider/limit_observation.go:30-49`).
  The SDK's 0089 amendment (its OQ5, with the owner) adds a way for a caller to get a refused cap.
* **F14. A cap can cause a credit or rate refusal.** OpenRouter counts `max_tokens` in its credit estimate and refuses a
  request over budget with a 402 `weight_exceeds_budget` asking to lower `max_tokens` (openrouter.ai/docs/api/reference/limits,
  as the review cites it); OpenAI counts `max_tokens` against tokens per minute, a 429. gobble maps a 402 to `llm.ErrQuota`
  and a 429 to `llm.ErrRateLimited` (`llm/kind.go:29-42`); through Kilo an upstream's 402 arrives as a 503 (0065 R34).
* **F15. Listed caps can be wrong.** In the snapshot, 89 of Kilo's 390 models list an output of exactly 90 % of the window
  (the review's count, re-counted here as `int(0.9 × context)`, `<scratch>/g0071/kilo_ninety.py`; GLM-5.3's 943,718 is one); models.dev lists `gpt-5-pro` at 272,000 where its real cap is 128,000, which
  pi corrects by hand (`<pi>/packages/ai/scripts/generate-models.ts:2881-2885`).
* **F16. Other harnesses' fit.** pi estimates the last usage plus what follows it, and nothing else again
  (`<pi>/packages/ai/src/utils/estimate.ts:97-112`), and fits with a 4,096 margin and a floor of 1
  (`<pi>/packages/ai/src/api/simple-options.ts:15-22`). kilocode subtracts 2,048, floors at 1,024, and, when the input alone
  fills the window, sends the configured value "so the provider returns a natural overflow error which triggers compaction"
  (`<kilocode>/packages/opencode/src/kilocode/session/llm.ts:9-10`, `:104-120`). qwen-code clips any cap the user did not set
  at 64,000 (`<qwen-code>/packages/core/src/core/tokenLimits.ts:25-35`).
* **F17. Refusals caused by the cap, not the input**, in the forms the SDK records: Anthropic's "input length and
  `max_tokens` exceed context limit: a + b > c" (go-llmprovider-sdk `llmprovider/window_0088_record_test.go:148-162` at
  33e6f09), which `v1.32.0` reads as a plain invalid request; OpenRouter's "maximum context length is W tokens. However, you
  requested about T tokens", which `v1.32.0` reads as an overflow (F7); Kilo's "exceeds the model's context window of W
  tokens. Your request contains approximately T tokens"; the split "maximum context length is W tokens … (I in the messages, C
  in the completion)"; and the cap forms "max_tokens: N > M, which is the maximum allowed number of output tokens",
  "supports at most M completion tokens, whereas you provided N" and "value of N but the supported range is from 1
  (inclusive) to M (exclusive)" (`internal/wire/limit_refusal.go:21-43`, `:102-115` at 33e6f09).
* **F18. The summary ladder reads any overflow as "input too large"**: `summariseFitting` drops more tool results on each
  `ErrContextOverflow` (`compaction/compaction.go:356-413`), even one the summary's cap caused.

### Measured: GLM-5.3's caps in gobble's catalog today

Read offline by a probe test on a scratch clone of 97fa1f6 (`provider.ModelInfo` with no key, the embedded snapshot alone;
`<scratch>/g0071/probe-caps.log`), checked against the snapshot's JSON and the SDK `v1.32.0` listing fixtures
(`llmprovider/catalog/testdata/ranking-2026-10-12/kilo.json`, `together.json`, `go.json`; `<scratch>/g0071/glm-caps.log`).

| gobble provider | Model | Window | Listed cap | Today | Sent (Q3's ceiling) | Fitted below 64,000 once the estimate passes |
| :--- | :--- | ---: | ---: | ---: | ---: | ---: |
| `kilo` | `z-ai/glm-5.3` | 1,048,576 | 943,718 | 8,192 | 64,000 | 980,480 |
| `kilo` | `z-ai/glm-5.3-flash` | 1,048,576 | 943,718 | 8,192 | 64,000 | 980,480 |
| `opencode-go` | `glm-5.3` | 1,000,000 | 131,072 | 8,192 | 64,000 | 931,904 |
| `opencode-go` | `glm-5.3-flash` | 1,000,000 | 131,072 | 8,192 | 64,000 | 931,904 |
| `together` | `zai-org/GLM-5.3` | 1,048,576 | 262,144 | 8,192 | 64,000 | 980,480 |
| `together` | `zai-org/GLM-5.3-Flash` | 1,048,575 | 400,000 | 8,192 | 64,000 | 980,479 |

Kilo's figures are the snapshot's and its own listing's (`top_provider.max_completion_tokens` 943,718); OpenCode Go's and
Together's are the snapshot's (their listings state no output limit). Every one states reasoning with `low`, `high`, `max`.
A user who sets `maxTokens` for one of them in models.json gets that value, fitted, with no ceiling.

## Decision Drivers

* The owner's rulings and answers above.
* No 8,192 by accident on any call gobble makes.
* A cap never asks for more than the window leaves, or a strict route refuses the call and gobble compacts a conversation that
  did not need it; a refusal the cap caused is answered by a smaller cap, not a compaction.
* One estimate of the context, for the trigger and the fit alike.
* Every number traced to its source (see "Every number").
* What the user sees is what the call asked, and why.

## Considered Options

* A. Interim: gobble sends the catalog's cap under the owner's ceiling, fitted to the window, on every turn and summary;
  converge on gobble owning the limit with the SDK's facts
* B. Interim: the catalog's cap, unfitted
* C. Interim: the catalog's cap, fitted, with no ceiling
* D. No interim: wait for `v1.33.0` and let the SDK choose (0065's option A)

## Decision Outcome

Chosen option: "A", because it is the owner's rulings and answers, it fixes the reported failure on `v1.32.0` for every model,
and the fit keeps a cap from turning a long conversation into an overflow. B was rejected because a cap plus a long input
passes the window on a strict route. C was rejected by Q3's answer: listed caps can be wrong (F15) and a large cap can be
refused for credit or rate (F14). D was rejected by the ruling.

### The decisions: the interim, on `v1.32.0`

* **D1. Where a turn's cap comes from.** `a.modelOf(provider, model).MaxTokens` for the turn's provider and model (F3, F4),
  read for each call, with its source: `user` when models.json states `maxTokens` for the model (entry or override),
  `assumed` when nothing states it (Q2: 16,384, `llm.AssumedMaxTokens`, which `llm/catalog`'s `defaultMaxTokens` becomes, a
  models.json entry without `maxTokens` included), else `catalog` (the snapshot or the listing).
* **D2. The rule.** For a cap `C` from D1, a window `W` and the estimate `E` (D3): first the ceiling, `C' = min(C, 64,000)`
  unless the source is `user` (Q3; qwen-code's `OUTPUT_TOKEN_CEILING`, F16); then the fit, with the room `R = W − E −
  4,096` (pi's margin, F16):
  * `W` unknown: send `C'`;
  * `R ≥ C'`: send `C'`, basis `stated`, `ceiling` (when the ceiling cut it), `user` or `assumed`;
  * `1,024 ≤ R < C'`: send `R`, basis `window`;
  * `0 < R < 1,024`: send `min(C', 1,024)`, basis `window` (kilocode's floor, F16);
  * `R ≤ 0`: send `C'`, basis `full`: by gobble's estimate the window is full, and the service's refusal starts the
    overflow path (kilocode's rule, F16), which compacts, or, with compaction off or refused, ends the turn with the
    overflow's error.
* **D3. One estimate, pi's anchoring.** `compaction.EstimateContext(path) (tokens int, anchored bool)` splits
  `EstimateEntries`' two cases (`compaction/estimate.go:293-333`). Anchored, `E` is the last valid usage plus the estimates
  of what follows it, and nothing else (pi, F16): the system text and the tools are inside that usage. Not anchored (a
  session's first call, the call after a compaction), `E` is the system text, the request's tool specs and every message.
  The automatic trigger uses the same `E` (`acpserver/autocompact.go:104` changes): it no longer adds the system text to an
  anchored count, and it counts the tools when not anchored. So with the default reserve, a call the trigger lets through
  has `R ≥ 16,384 − 4,096 = 12,288`.
* **D4. The seam.** `agent.Config.Compact` gains the tool specs the next request will carry (`func(ctx, overflow bool,
  tools jsontext.Value)`), and `agent.Config.OutputLimit func(*llm.Request) llm.OutputLimit` is called by `turn.call` with
  the built request (`agent/agent.go:685-689`), its `Tokens` set as `req.MaxTokens`; nil leaves the request as today. The
  rule is `llm.FitOutput` in a new `llm/limit.go`. `acpserver` supplies both functions per prompt over one estimator, and
  records the last value on the live session under `a.mu`.
* **D5. Thinking.** The cap is the whole output, reasoning included; gobble still sends an effort only. A Messages route's
  budget raise (F9) is reachable only when D2 sends at most 4,096.
* **D6. The compaction summary takes the same rule, step by step.** `compaction.Model` gains `Window` and the cap's source;
  each summary and turn-prefix request, at every step of the ladder, is sent `llm.FitOutput` over that request's own
  estimate (its system text and prompt, not anchored). The 80 % and 50 % shares go (0065 D4's intent, pi's commit
  6fcb48670); `compaction.reserveTokens` only decides when compaction starts. A refusal of a summary request goes through
  D8 (a) before the ladder reads it as an overflow: a cap-caused refusal is resent once at the stated room or cap, and only
  an overflow that is not cap-caused drops tool results (F18).
* **D7. The title is unchanged**: `min(128, C)` with `C` raw (F3). 0065 D3 governs its later change.
* **D8. Refusals on `v1.32.0`.**
  (a) **Caused by the cap.** `llm.ReadLimitRefusal(err, sent)` reads the forms of F17 from the error's message: a window
  form gives the room `W − I` (I the input it states, or its total less `sent`), a cap form gives `M`. When the room or
  `M` is at least 1 and below `sent`, the call is resent once, before any compaction, at that value, basis `refusal`. A
  room at or below 0 is an input overflow, and takes (b).
  *(2026-11-24, amendment, the owner's "Keep the stated value (Recommended)": once a refusal has stated a room or cap M in a
  turn, every later call of that turn asks for at most M: its cap is computed again, as (b) and L11 say, then held to M,
  so a retry for another cause (an offline wait, an empty reply, an overflow's compaction, the next call after a tool
  round) does not ask again for what the service refused (L44).)*
  (b) **An input overflow** takes today's path: compact once and resend, the resend's cap computed again (D2 runs per call).
  (c) **Any other refusal of the cap.** An invalid request whose message names an output-limit field (`max_tokens`,
  `max_completion_tokens`, `max_output_tokens`, `maxOutputTokens`, without case), the SDK's "fills the context window"
  reason, or a cap-caused refusal still refused after (a)'s resend, fails the turn, and its error text gains: "gobble asked
  for at most N output tokens (B); to send less, set maxTokens for MODEL under modelOverrides in models.json."
  (d) **Credit and rate.** A 402 or quota failure, a 429 or rate-limit failure that outlasts the SDK's retries, and Kilo's
  503 for an upstream 402, on a call that carried gobble's cap, fail the turn as today, and the error text gains: "gobble
  asked for at most N output tokens (B); some services hold credit or rate against that number; set maxTokens for MODEL
  under modelOverrides in models.json to ask for less." Nothing is retried at a guessed smaller value.
  *(2026-11-24, amended by the owner's "Move the line into a new field": the line is not added to the error text, which
  stays byte-identical; it rides a new `acpext.ErrorData` member, `OutputLimitNote` (`outputLimitNote`). See "Amendment
  of 2026-11-24: D8 (d)'s line in an acpext member".)*
  Nothing is kept across calls (F8).
* **D9. The user's override** is models.json `maxTokens` (F5): source `user`, no ceiling, fitted. No setting and no ACP
  option (the owner's "models.json only, for now (Recommended)", 0066 S32, S33). `docs/guides/models-json.md` says so, and
  says that an answer over 64,000 is cut until `maxTokens` is set.
* **D10. Display.** `/context` and `/session` gain a line after the context line, "Output limit: N tokens on the last call",
  with the basis in words: "the model's stated limit", "gobble's 64,000 ceiling; set maxTokens in models.json to raise
  it", "your models.json maxTokens", "assumed: no source states this model's limit", "held to the room the window leaves",
  "the window is full by gobble's estimate", "the cap the service stated when it refused the first". Before the session's
  first call: "Output limit: none sent yet; the model's limit is N (source)" or "… is unknown". The prompt response carries
  the turn's last value as `_meta.gobble.outputLimit` `{tokens, basis}`; `acpclient.Result.OutputLimit` reads it; the
  `--stats` suffix, live and headless, adds "out-limit N" (`render.tokens`'s form, ASCII). A turn with no call carries none.
* **D11. 0070's `limit`** is the value sent. Whichever of 0070 and 0071 is built second updates T13's expected value to the
  request's `MaxTokens` (L29).
* **D12. Records.** Dated notes: 0065 (D1's "Agent turns keep `MaxTokens: 0`", R1, R2, R3, R14, R17 superseded for turns by
  D1 and D2; D4 and R7 by D6; R34 by D8 (d); D3, R9, R10 unchanged until V8); its PLAN (Phase 1's steps that set the turn and
  the summary to 0 are not built while this record holds); 0066 (D11 and S29 superseded by V3; S39 by Q2's answer; D8 merged
  with D10 by V6); 0070 (T13); 0004 (`docs/decisions/0004-MADR-go-module-architecture.md:745`).
* **D13. Exported names this adds** (gobble is below `v1.0.0`; `make apidiff` is a no-op): `llm.OutputLimit`,
  `llm.LimitBasis` and its values (`stated`, `ceiling`, `user`, `assumed`, `window`, `full`, `refusal`), `llm.CapSource`
  (`catalog`, `user`, `assumed`), `llm.FitOutput`, `llm.ContextSafetyTokens` (4,096), `llm.OutputCeiling` (64,000),
  `llm.MinOutputTokens` (1,024), `llm.AssumedMaxTokens` (16,384), `llm.ReadLimitRefusal` and `llm.LimitRefusal`;
  `llm.Model.MaxTokensFrom` (`json:"-"`); `catalog.Facts.MaxTokensStated`; `agent.Config.OutputLimit` and
  `agent.Config.Compact`'s new parameter; `compaction.Model.Window` and `MaxTokensFrom`; `compaction.EstimateContext`;
  `acpclient.Result.OutputLimit` and its type. *(2026-11-24, amendment: and, in the acpext module,
  `acpext.ErrorData.OutputLimitNote`, JSON `outputLimitNote`, a new optional member, which gobble uses from acpext's next
  minor tag; see the amendment below.)*

### The decisions: the convergence, on `v1.33.0`

Decided now:

* **V1. The facts.** After 0066 D1's `catalog.LookupLimits` at session start and model switch, each call reads
  `catalog.CachedLimits`: `StreamedOutput` is `C` (gobble always streams), its source `user` when the chosen evidence is the
  models.json layer, `assumed` when unknown; `Window`, else `Input` (0066 D9), is `W`. D2 and D3 then apply unchanged.
* **V2. Explicit limits, always.** ~~Open: Q4.~~ *(2026-11-24: decided by Q4, "Always, gobble's policy (Recommended)".)*
  Every turn and every summary names its limit on every wire, optional ones included (Together, OpenCode Go's chat route):
  `Request.MaxOutputTokens` is D2's value, so the SDK reports basis `request`. This is gobble's own policy above the SDK's
  default; nothing gobble sends is left to the SDK's default, fallback or a service's default.
* **V3. The window fit and the ceiling stay gobble's** wherever gobble names a limit, since the SDK fits only the values it
  chose (F13). This supersedes 0066 D11 and S29 ("gobble names no limit and no ceiling of its own") for those calls.
* **V4. A refused value.** The SDK does not resend a caller's value (F13). gobble gets the refused cap by the path the SDK's
  0089 amendment adds (its OQ5), and, until that path exists, by reading `CachedLimits` again for an `observed` fact below
  what was sent; then it resends once at it, saves the observation (model, base URL, scope, tokens, time) to 0066 D7's
  `limit-observations.json`, passes the file back with `WithLimitObservations`, and says so with 0066 D10's
  `model.limit.learned`. A window form is resent once at its room, not saved (D8 (a)). Otherwise D8 (c).
* **V5. What the user sees** is the SDK's report: `/context`, `/session` and `--stats` show the last call's
  `Response.OutputLimit` (the tokens on the wire after a thinking budget's raise, sent or not, basis, source); a ChatGPT
  route shows "none sent".
* **V6. One line, not two.** 0066 D8's session-report line and this record's D10 line become V5's line; 0066 D8's debug log
  is 0070's `limit` and `out_limit*` fields (0070 D19). V4 and V5 use 0066's discovery, store and notice, so Phase 2 follows
  the 0066-PLAN phases that build them.

Left to the adoption amendments of 0065 and 0066, by name: **V7** the unknown limit's notices under V2 (0066 D10's
`model.limit.unknown` and `fallback`); **V8** the title's limit (0065 D3); **V9** 0065 D1 and D2's layers to the provider.
Out of scope: **V10** probing (the ruling's option 3), a later record.

### Consequences

* Good, because every GLM-5.3 turn on Kilo, OpenCode Go and Together asks for 64,000 output tokens instead of 8,192, so
  reasoning at `high` has room, and a user can ask for the model's whole cap with `maxTokens`.
* Good, because the ceiling restores a bound: a runaway reply, reasoning that never ends included, stops at 64,000 tokens per
  call unless the user raised it, where an unfitted 943,718 would not stop for hours.
* Good, because the fit and the trigger read one estimate, so a call the trigger lets through has room to answer.
* Good, because a refusal the cap caused is answered once by the cap or room the service states, not by a compaction.
* Good, because no call gobble makes is left to the SDK's 8,192 by accident.
* Neutral, because the trigger's estimate changes: it compacts slightly later on an anchored count (no system text twice)
  and slightly sooner on an unanchored one (tools counted).
* Bad, because an answer longer than 64,000 tokens is cut until the user sets `maxTokens` (Q3's answer).
* Bad, because a wrong listed cap below 64,000 is still sent as listed (F15): too high, it is refused and D8 (a) resends at
  the cap the service states, when it states one; too low, the answer is cut until the user sets `maxTokens`.
* Bad, because a credit or rate refusal the cap causes fails the turn with a line, not a retry (D8 (d)).
* Bad, because a larger summary cap makes a reasoning summary longer and costlier than the 13,107 it was held to.
* Bad, because gobble reads refusal text in seven forms on `v1.32.0`, a copy of the SDK's readers until `v1.33.0`.

### Confirmation

* The PLAN's tests, one per decided row, red at the base where a row changes behaviour, each with a plant caught on a
  scratch copy.
* `make preflight` passes on this Mac, the Linux machine and the Windows laptop through `run-on-host`; `make race` on the
  Linux machine.
* An independent review of each build phase finds no blocker.

### State table

"Today" is 97fa1f6. Each decided row's test and plant are in the PLAN's tables, one to one. `C`, `C'`, `W`, `E`, `R` are D2's.

| Row | Input or state | Status | Outcome | Decided by |
| :--- | :--- | :--- | :--- | :--- |
| L1 | A turn, `C` from the catalog at most 64,000, `R ≥ C` | decided | `C`, basis `stated` (today 0, so 8,192) | D1, D2 |
| L2 | `1,024 ≤ R < C'` | decided | `R`, basis `window` | D2 |
| L3 | `0 < R < 1,024` | decided | `min(C', 1,024)`, basis `window` | D2 |
| L4 | `W` unknown | decided | `C'` | D2 |
| L5 | No source states `C` | decided | 16,384, fitted, basis `assumed` | Q2 |
| L6 | A model models.json defines without `maxTokens` | decided | as L5 | Q2 |
| L7 | models.json `maxTokens` stated | decided | that value, no ceiling, fitted, basis `user` | D1, D9; Q3 |
| L8 | Thinking `off` to `max` | decided | the same cap at every level; an effort only | D5 |
| L9 | The estimate | decided | D3's, one function | D3 |
| L10 | Several calls in one turn | decided | the cap computed for each call | D2, D4 |
| L11 | An input overflow (no cap-caused form, or a room at or below 0) | decided | compact once and resend, the cap computed again | D8 (b) |
| L12 | An invalid request naming an output-limit field, no form read | decided | the turn fails; D8 (c)'s line | D8 (c) |
| L13 | An invalid request naming no output-limit field | decided | as today, no line | D8 (c) |
| L14 | The SDK's "fills the context window" reason | decided | as L12 | D8 (c) |
| L15 | A 402, a 429 after the SDK's retries, or Kilo's 503 for a 402, on a capped call | decided | the turn fails; D8 (d)'s line *(2026-11-24: in `ErrorData.outputLimitNote`, the error text byte-identical; "Move the line into a new field")* *(2026-11-24, Phase 1b: the PLAN's "`TestPromptQuotaData` unchanged" is annotated, not erased. Its four prompts run capped calls, so a quota or rate failure's data carries the note, which the acpext frames have not. Ruled "Expect frame plus note (Recommended)": the test's expected data is each frame decoded, with the expected `OutputLimitNote` set, then re-encoded, so every other member is still compared byte for byte; the frames stay as they are)* | D8 (d) |
| L16 | A Messages route's budget raise | out of scope | the SDK's wire (F9); only when D2 sends at most 4,096 | |
| L17 | A summary and a turn prefix | decided | each request `llm.FitOutput` over its own estimate; the shares go | D6 |
| L18 | A summary, `C` unknown | decided | as L5 | Q2; D6 |
| L19 | A summary by the previous model | decided | that model's `C` and `W` | D6 |
| L20 | The title | decided | `min(128, C)`, `C` raw, unchanged | D7 |
| L21 | A model switch | decided | the next prompt's calls read the new model | D1; F12 |
| L22 | A provider switch | decided | as L21 | D1; F12 |
| L23 | Concurrent sessions | decided | each session its own value, under `a.mu` | D4 |
| L24 | A ChatGPT route | out of scope | the SDK sends no limit there (F10); V5 shows it | |
| L25 | `/context` and `/session` after a call | decided | D10's line, value and basis | D10 |
| L26 | `/context` before the first call | decided | "none sent yet", the limit and source, or "unknown" | D10 |
| L27 | `--stats`, live | decided | "out-limit N" from `_meta.gobble.outputLimit` | D10 |
| L28 | `--stats`, headless | decided | the same suffix on stderr | D10 |
| L29 | 0070's per-call `limit` | decided | the request's `MaxTokens` | D11 |
| L30 | A turn with no model call | decided | no `outputLimit`; `/context` keeps the last call's | D10 |
| L31 | The 0065, 0066, 0070 and 0004 records | decided | D12's dated notes | D12 |
| L32 | Where the override is documented | decided | `docs/guides/models-json.md` | D9 |
| L33 | `C` from the catalog above 64,000 | decided | 64,000, basis `ceiling` (all six GLM-5.3 models) | Q3; D2 |
| L34 | Whether the services accept the caps sent | decided | one short live call per GLM model at the cap sent; status, finish reason, token counts and any 402 or 429 recorded | Q1 |
| L35 | `R ≤ 0` | decided | `C'`, basis `full`; the refusal's overflow path compacts, or ends the turn | D2 |
| L36 | A wrong listed cap below 64,000 | decided | sent as listed; too high, refused and resent by L39 when stated; too low, cut until `maxTokens` | D2; F15 |
| L37 | The trigger at the threshold, `Prepare` refuses | decided | the call goes ahead (as today); D2 on its `R`, `full` when at or below 0 | D2, D3; F6 |
| L38 | A `compaction.reserveTokens` below 5,120 | decided | the trigger lets through `R` below 1,024: L3, or L35 | D2, D3 |
| L39 | A refusal in a form of F17, a room or cap below what was sent and at least 1 | decided | resent once at it, basis `refusal`, before any compaction; refused again, L12 | D8 (a) |
| L40 | A summary request refused in a form of F17 | decided | resent once at its stated room or cap before the ladder drops tool results | D6, D8 (a) |
| L41 | The trigger's estimate | decided | D3's: no system text twice on an anchored count; tools counted on an unanchored one | D3 |
| L42 *(2026-11-24)* | acpext's contract for D8 (d)'s line | decided | `ErrorData.OutputLimitNote`, `outputLimitNote`, a string, `omitzero`, in its sorted place; documented in `types.go` and `doc.go`; an example frame of its own; the existing frames unchanged; a client that does not know it ignores it | "Move the line into a new field"; the amendment below |
| L43 *(2026-11-24)* | gobble's use of the member, after acpext's next minor tag | decided | a capped call's credit or rate failure carries the line in `outputLimitNote`, the error text byte-identical; the terminal client shows it on the line after the error, as it shows the limit's line | "Move the line into a new field"; the amendment below |
| L44 *(2026-11-24)* | A later call of a turn in which a refusal stated M (L39): a retry after an offline wait, an empty reply or an overflow's compaction, or the next call after a tool round | decided | its cap computed again (L11), then held to at most M, basis `refusal` when held; a smaller stated M' lowers it again | "Keep the stated value (Recommended)"; D8 (a) as amended |
| L45 *(2026-11-24)* | Print mode (`-p`), a capped call's credit or rate refusal | decided | the JSON result's `errorData` carries `outputLimitNote` (L43's member, as the prompt error gave it); the text output is unchanged: no note line | "Test the JSON note, text as is (Recommended)"; the amendment below |
| V1 | `v1.33.0`: the facts a call reads | decided | `CachedLimits`' `StreamedOutput` and `Window` (else `Input`), with the source | V1 |
| V2 | `v1.33.0`: a limit on an optional wire (Together, OpenCode Go's chat route) | decided *(2026-11-24: Q4)* | every turn and summary names D2's value, basis `request` | Q4 "Always, gobble's policy (Recommended)"; V2 |
| V3 | `v1.33.0`: fit and ceiling where gobble names a limit | decided | D2, D3 with V1's facts | V3 |
| V4 | `v1.33.0`: a refused cap stated | decided | the SDK's caller path, else `CachedLimits`; one resend; saved; `model.limit.learned` | V4 |
| V5 | `v1.33.0`: a refusal with no cap stated | decided | D8 (a) for a window form, else D8 (c) | V4 |
| V6 | `v1.33.0`: display | decided | the SDK's `Response.OutputLimit` | V5, V6 |
| V7 | `v1.33.0`: the unknown limit's notices | out of scope | left by name to the adoption amendment of 0066 | V7 |
| V8 | `v1.33.0`: the title | out of scope | left by name to the adoption amendment of 0065 | V8 |
| V9 | `v1.33.0`: 0065 D1 and D2's layers | out of scope | left by name to the adoption amendment of 0065 | V9 |
| V10 | Probing | out of scope | a later record | V10 |

Counts: 44 decided, 6 out of scope (three left by name to the adoption amendments), 1 open; 51 rows. *(2026-11-24,
after Q4's answer: 45 decided, 6 out of scope, 0 open; 51 rows.)* *(2026-11-24, after "Move the line into a new field":
47 decided, 6 out of scope, 0 open; 53 rows.)* *(2026-11-24, after "Keep the stated value (Recommended)": 48 decided,
6 out of scope, 0 open; 54 rows.)* *(2026-11-24, after "Test the JSON note, text as is (Recommended)": 49 decided, 6 out of
scope, 0 open; 55 rows.)* *(2026-11-25, after "gobble: v1.33.0 pin first, then 0072 (Recommended)": rows P1 to P23 in the
amendment of that date; 69 decided, 7 out of scope, 2 open (P6 under Q5, P21 under Q6); 78 rows.)* *(2026-11-25, second:
the owner's answers and the review; 83 decided, 7 out of scope, 0 open; 90 rows.)*

## Every number this record uses

| Number | Where | Provenance |
| :--- | :--- | :--- |
| 64,000 | D2's ceiling, `llm.OutputCeiling` | Q3's answer; qwen-code's `OUTPUT_TOKEN_CEILING` (F16) |
| 4,096 | D2's margin | pi's `CONTEXT_SAFETY_TOKENS` (F16) |
| 1,024 | D2's floor | kilocode's `MIN_OUTPUT` (F16); opencode's `OUTPUT_TOKEN_MIN` |
| 16,384 | L5, L6, L18 | Q2's answer; Pi's default for a model models.json defines (`modelsjson.go:117`) |
| chars/4 | D3 | pi's, already gobble's (`compaction/estimate.go:31-35`) |
| 128 | the title | unchanged (`acpserver/title.go:32`) |
| 13,107 and 8,192 | the summary's shares | gone (D6) |

## Pros and Cons of the Options

### A. The catalog's cap under the ceiling, fitted

* Good, because it is the owner's rulings and answers, and the reported failure ends today.
* Good, because the ceiling bounds a wrong or runaway cap, and the user can lift it per model.
* Bad, because an answer over 64,000 is cut until the user sets `maxTokens`.

### B. Unfitted

* Good, because it is the smallest change.
* Bad, because a cap plus a long input passes the window on a strict route, and gobble compacts for nothing.

### C. Fitted, no ceiling

* Good, because no number of gobble's own bounds a model.
* Bad, because 89 Kilo models list 90 % of their window (F15), and a cap that large can be refused for credit or rate (F14).

### D. Wait for `v1.33.0`

* Good, because nothing is built twice.
* Bad, because the reported failure stays until the tag and gobble's bump.

## Owner Questions

*(2026-11-24: Q4 answered, "Always, gobble's policy (Recommended)", option (1). No question is open.)*

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| Q4. After the SDK update, should gobble name an output limit on every call, even where the service does not require one? On Together and OpenCode Go's chat route, the limit is optional; your SDK ruling "Only where required (Recommended)" (0089 OQ1) makes the SDK send nothing there, so the service's own default applies, and that default is not known for GLM-5.3. | (1) **Always, as gobble's own policy (Recommended)**: every turn and summary names the limit of D2 (64,000 under the ceiling, fitted), recorded as gobble's policy above the SDK's default; the SDK's ruling still governs other SDK users. (2) **Only where required or learned**: gobble names a limit on the Messages wire, and where a service stated a cap; elsewhere it sends none. (3) **Never on optional wires**: as the SDK does; the service's default applies. | (1) the same behaviour as the interim; gobble overrides the SDK's choice on those routes. (2) GLM-5.3 on Together and OpenCode Go runs on an unknown default, possibly as small as today's failure, until a refusal states a cap. (3) as (2), and a learned cap is not sent either. | V2 |

## More Information

### Relation to other records

* `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` (accepted, not built): option A, the SDK choosing the
  limit, is overtaken by the ruling's option 1 (D12, V9).
* `0066-MADR-set-each-session-model-up-from-discovered-limits.md` (accepted, not built): discovery, evidence, kept caps and
  notices stay; D11 and S29 are superseded (V3); D8 merges into V5.
* `0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md` (accepted, not in the tree): its `limit` field
  (D11); 0065's R21, "Stop with a notice", still governs a cut answer.
* `0004-MADR-go-module-architecture.md` decision 6: carried out for turns by D1.
* go-llmprovider-sdk `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md` and
  `docs/decisions/0089-MADR-model-output-limit-discovery.md`, with 0089's amendment for a caller's refused cap (its OQ5).

### A gap in the SDK's learning for a caller that names its limit

At go-llmprovider-sdk 33e6f09, a refused `Request.MaxOutputTokens` keeps the stated cap in the process memo but reports no
`WarningLimitLearned`, which rides only a `Response` (`internal/wire/limit_rule.go:198-253`). The SDK's 0089 amendment (OQ5,
with the owner) adds a caller's path; V4 uses it, with `CachedLimits` as the fallback.

### Amendment of 2026-11-24: D8 (d)'s line in an acpext member

Phase 1's first pass stopped (the PLAN's execution record): D8 (d)'s line, added to the error text, turned three of
`acpserver`'s `TestPromptQuotaData` frames red, because that test compares the prompt error's data byte for byte with
`acpext/testdata/prompt-quota.error.json`, `prompt-rate-over-cap.error.json` and `prompt-weekly.error.json`, the acpext
contract's example frames. The owner's ruling, relayed by the coordinator: **"Move the line into a new field".**

* **The member.** `acpext.ErrorData` gains `OutputLimitNote string` with the tag `json:"outputLimitNote,omitzero"`, between
  `limit` and `provider` in its JSON names' sorted order. It holds D8 (d)'s line as written there, from "gobble asked for
  at most N output tokens (B)" to "to ask for less", and is absent on every other failure. Its doc, in `types.go` and in
  `doc.go`'s failed-turn paragraph: "OutputLimitNote is set when a quota, rate or credit failure came from a call that asked
  for an output limit, which some services hold credit or rate against: the limit asked for, why, and how to ask for less
  (gobble-cli 0071-MADR D8 (d))."
* **Compatibility.** The member is optional and new: acpext's rule is that a client takes a member it does not know
  (`acpext/doc.go`, "Versions"), and `omitzero` leaves it out of every frame that has none, so every existing frame is
  unchanged byte for byte. A client built on `v0.5.0` decodes a failure that carries it with the member ignored, unless
  it decodes rejecting unknown members; magic-cli-remote's fixture test does so against acpext's frames, but of the same
  acpext version as its types, so the new frame arrives with the type that names it.
* **The release.** Each change to acpext's contract is a new minor tag (`acpext/doc.go`, "Versions";
  `0042-MADR-acpext-extension-contract-module.md`), and gobble pins `acpext v0.5.0` (`go.mod:20`) with no `replace`. So
  gobble sets the member only after the owner pushes `acpext/v0.6.0` and gobble's pin moves to it, as 0049-PLAN's P5a and
  P5b did for `windowMinutes`. Until then gobble adds no line (D8 (d)), and the error text is byte-identical.
* **D8 (c) is unchanged:** a refusal of the limit still gains its line in the error text; no frame of acpext is such a
  refusal.
* **The terminal client** shows the member on the line after the error, as it shows the limit's line (`internal/cli/limit.go`),
  once gobble's pin carries it (L43).
* **D13** gains `acpext.ErrorData.OutputLimitNote`; L15 is amended, and L42 and L43 added.
* **Print mode (2026-11-24, Phase 1b; L45).** Print mode's JSON result already writes the failed turn's data as
  `errorData` (`internal/cli/print.go`), so the member reaches it with no new code; its text output prints no limit line
  today and prints no note. Ruled "Test the JSON note, text as is (Recommended)": L45 pins the JSON and the unchanged text.
* **`TestPromptQuotaData` (2026-11-24, Phase 1b).** The test's four prompts run on a model the catalog does not know, so each
  call is capped (16,384, assumed) and a quota or rate frame's data carries the note. Ruled "Expect frame plus note
  (Recommended)": see L15.

### What could not be established offline

* Whether each service accepts 64,000 for GLM-5.3, and whether a gateway holds credit or rate against it: Q1's live calls.
* How far chars/4 runs from GLM's tokenizer on code; the 4,096 margin is pi's, not a measurement.

## Amendment of 2026-11-25: pin v1.33.0 first, and adopt it

*(2026-11-25. Additions only; nothing above is rewritten. Rows P6 and P21 wait on Q5 and Q6 below, so this amendment is
`proposed` until the owner answers them; every row above it stands `accepted`.)*

### The ruling

The owner's ruling of 2026-11-25, relayed by the coordinator, quoted exactly: **"gobble: v1.33.0 pin first, then 0072
(Recommended)"**. gobble pins go-llmprovider-sdk `v1.33.0` (tagged at `138d52f`) first, together with this record's Phase 2
and the adoption amendments of `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` and
`0066-MADR-set-each-session-model-up-from-discovered-limits.md`, and then builds all of
`0072-MADR-keep-a-refreshed-models-dev-copy-on-disk.md` on `v1.33.0`. 0072's steps that exist only for `v1.32.0` are dropped,
and its Phase 4 hand-in becomes one of its main phases (0072's amendment of this date).

### Why an amendment and not a new record

The pin takes no new architectural choice. Who owns the limit is V1 to V9; discovery, the kept caps and the notices are 0066's;
the hand-in is 0072 D17's. The ruling changes the order, and with it what gobble hands the SDK before 0072's copy exists; and
`v1.33.0`'s API is now measured where V4 and V5 could only name it. Each of those is a case of a decision already taken, so each
is a state row here, and this PLAN's Phase 2 maps them one to one. A new record would split one convergence across five numbers.

### Re-grounded at 073fd1f and at go-llmprovider-sdk v1.33.0

gobble at `073fd1f` (tag `v0.9.0`), tree clean; the SDK at `v1.33.0`, `138d52f`, its checkout's HEAD, tree clean; both read
only. The SDK's paths below are under its `llmprovider/` directory.

* **F13, moved to the tag.** A caller's `Request.MaxOutputTokens` is sent as given, basis `request`, never held
  (`internal/wire/limit_rule.go:54-61`); refused, it is not resent (`:226-227`); a validated cap form is kept in the process
  memo (`:215-225`) and set on the error that ends the call as `APIError.Observed` (`:251-265`, `:281-290`;
  `api_error.go:189-199`). That field is the caller's path of the SDK's 0089 amendment that V4 waited for. `catalog.CachedLimits`
  is `catalog/limits.go:202-216`; `Limits.StreamedOutput` `:37-44`; an "observed" cap of this process bounds whatever is
  chosen (`:450-475`); `Response.OutputLimit` is `item.go:147`, a `SentLimit` (`sent_limit.go:9-22`); `WithLimitObservations`
  is `limit_observation.go:30-50`.
* **F19. The eight removed names** (the SDK's CHANGELOG `[v1.33.0]`, "Removed"; its 0093-MADR D10) and gobble's uses, by
  `git grep` over `*.go`: `llmprovider.WithoutModelMetadata` at `llm/provider/provider.go:215` (`New`), `llm/provider/model.go:176`
  (the lookups), `llm/provider/signin.go:344` (the sign-in wizard's listings) and `llm/provider/image_test.go:209` (a test);
  `llmprovider.WithModelMetadataURL` at `llm/provider/list.go:130` and `:160`, behind gobble's own `Options.ModelMetadataURL`
  (`provider.go:53-55`), which eight test sites set to serve a document by URL: `llm/provider/list_test.go:103`, `:155`,
  `:204`, `:227`, `:277`, `sdkcache_test.go:45`, `chatserver_test.go:122`, `internal/cli/modelscmd_test.go:193`. gobble calls
  none of the other six.
* **F20. What `v1.32.0`'s production listing read.** No production `Options` sets `ModelMetadataURL`, so every listing but
  Kilo's and Ollama's fetched `https://models.opencode.ai/api.json` (0072 F3). At `v1.33.0` nothing is fetched: a listing ranks
  by the document its options hand in (`catalog.WithMetadata`, `catalog/model_metadata.go:242-260`), else by the curated order
  and the name rules (0093-MADR D3, MH11). The pin changes production's ranking unless gobble hands a document (Q5).
* **F21. Every model call gobble makes names a limit.** One stream call site (`llm/provider/provider.go:411`) and three
  `llm.Request`s: the turn, whose `MaxTokens` is `agent.Config.OutputLimit`'s (`agent/agent.go:917-933`), set on gobble's one
  `agent.Config` (`acpserver/agent.go:792-811`, `:810`); the summary, `llm.FitOutput`'s, at least 1
  (`compaction/compaction.go:412`, `:531`); the title, `min(128, C)` or 128 (`acpserver/title.go:187-191`). So `v1.33.0`'s
  refusal of a cold Messages-route request that names no limit (`internal/wire/limit_rule.go:78-81`; 0093 D8, MH18) and Kilo's
  cold "none" (MH51) never meet a production call. Tests do: `llm/provider/record_test.go:381`, `:484`, `replay_test.go:184`,
  `:206` and `workspace_test.go:18` build a Claude provider and send no `MaxTokens` (by `git grep`; Phase 2.0's red run lists
  every one).
* **F22. Cut answers at `v1.33.0`** (CHANGELOG, "Changed"): a cut with whole tool calls is a response finishing `length` with
  them, the cut call dropped and named in a `WarningToolCallCut`; a Responses or Interactions text cut is a `length` response; a
  cut with nothing usable is `ErrEmptyAnswer`, which matches `ErrIncomplete`, as at `v1.32.0`. Kilo's window form is
  `ErrContextOverflow` where it was `ErrInvalidRequest`. gobble's stream loop yields each whole call as `llm.ToolCallDone`
  (`llm/provider/provider.go:434-438`); `stopOf` returns `StopToolUse` when a call was yielded, else `StopMaxTokens` on
  `FinishLength` (`:549-559`); `agent.note` logs any warning kind at warn, once per kind and model (`agent/warning.go:40-56`).
* **F23. gobble's types carry none of `v1.33.0`'s reports.** `llm.Done` holds a stop reason only (`llm/event.go:137-140`);
  `llm.APIError` has no observed cap (`llm/errors.go:53-66`); `llm` imports no SDK (0072 F7). 0070's `model call` record writes
  `limit` (`llm/provider/record.go:128`) and no `out_limit*` field, which `TestModelCallRecordHoldsNoContent/no_out_limit`
  pins (`llm/provider/record_test.go:526`; 0070 T14).
* **F24. A remembered cap fills a gap; an observed one bounds.** `CachedLimits` reads a cap passed back with
  `WithLimitObservations` as the evidence "remembered", rank 1000, below every caller (10,000), service (5,000) and registry
  source (a "models.dev" layer 4,000, the document 3,000), so it is chosen only where nothing else states the output
  (`catalog/limits.go:105-117`, `:497-510`); a provider uses one only to cut a `WithMaxTokens` value
  (`limit_observation.go:30-47`), which gobble never passes. A cap the service stated in this process is "observed" and bounds
  whatever is chosen (`:450-475`). So a cap saved by one run does not lower the next run's first call wherever a listing, the
  document or models.json states the model's output (Q6).
* **F25. Scope.** gobble passes no Kilo organization (`git grep`: only a comment, `llm/provider/signinrun.go:233`), so a Kilo
  observation's scope is empty; a Claude key's workspace reaches lookups as `catalog.WithClaudeWorkspace`
  (`llm/provider/model.go:180-182`) and `New` as `claude.WithWorkspace` (`provider.go:241-243`).
* **F26. gobble lines moved since this record's base** that Phase 2 cites: F12's `acpserver/agent.go:783-803` is `:786-811`;
  D4's `agent/agent.go:685-689` is `:906-933`. Phase 2.0 re-reads every other citation and records each move.

### The decisions of the pin

* **P1. The pin.** `go.mod:16` names `v1.33.0`; `go.sum` follows; no `replace`; acpext's pin is unchanged.
* **P2. Each removed name's site, and what takes its place.**

  | Site at 073fd1f | At `v1.33.0` |
  | :--- | :--- |
  | `llm/provider/provider.go:215`, `New` | deleted. `New` hands no document under Q5 (1) and (2), as at `v1.32.0`, whose requests read none with the fetch off (0093 D3, MH4); under Q5 (3), `catalog.WithMetadata` with the embedded copy |
  | `llm/provider/model.go:176`, the lookups | deleted. The embedded copy stays a layer (`model.go:108-111`), which `CachedLimits` reads as registry evidence below the listing (`catalog/limits.go:340`), under Q5 (1) and (2); `catalog.WithMetadata` in its place under (3) |
  | `llm/provider/signin.go:344`, the wizard's listings | deleted; the wizard hands none, as with the fetch off (0093 MH22) |
  | `llm/provider/list.go:129-130`, `:159-160` | `catalog.WithMetadata` with `o.ModelData`'s document when it is set (P3), nothing otherwise |
  | `llm/provider/provider.go:53-55`, `Options.ModelMetadataURL` | removed: gobble's own field, for tests; P3 takes its place |
  | `llm/provider/image_test.go:209` | the option deleted, nothing else changed |

* **P3. A document seam, before 0072.** `provider.ModelData` (an immutable parsed document), `provider.ParseModelData(r
  io.Reader) (*ModelData, error)` (over `catalog.ParseMetadata`, so its 32 MiB bound and section checks hold) and
  `provider.Options.ModelData *ModelData`, nil handing none. They are three of 0072 D21's names, built here with this meaning;
  0072 adds its others. Before 0072 only `List` and its Chat Completions path read `Options.ModelData`.
* **P4. Tests that served a document by URL** (F19's sites but `list_test.go:155`, which P5 deletes) parse the same bytes with
  `ParseModelData` and set `Options.ModelData`; their expectations are unchanged.
* **P5. Tests of the fetch** go with it: `TestListFetchesMetadata` (`llm/provider/list_test.go:143-164`, which pinned
  0020-MADR D2's F6), and in `TestListFullCatalog` the two assertions that counted the document's request (`:117-122`: two
  requests, the document's with no credential), which become one request, the listing's, with the key; its model order is
  unchanged.
* **P6. Which document gobble hands where before 0072.** Open: Q5. Under its recommended option (1), production's listings
  (the `Options` of `internal/cli/models.go:323`) get the embedded 2026-10-14 copy, which `internal/cli` parses once with
  `ParseModelData` from `gcatalog.Snapshot()`; the lookups and the limits calls keep it as a layer; `New` and the wizard hand
  none. 0072's Phase 1 swaps the copy for its current document.
* **P7. Production never sends an unnamed limit** (F21): a guard that the turn, the summary and the title each carry a limit on a
  cold Claude wire, so none is refused before sending.
* **P8. Tests refused before sending.** A test F21 names, or that Phase 2.0's red run finds refused by `limit_rule.go:78-81`,
  gives its request `MaxTokens: llm.AssumedMaxTokens` (16,384), what production asks of a model no source states (L5), every
  assertion unchanged but one that names the limit sent, which then expects 16,384. Any other red is a stop.
* **P9. A cut answer with whole tool calls** (F22; 0065 R24 after the tag): the calls run, the turn goes on, the cut call is
  dropped with no result written, and `tool_call_cut` is logged at warn once per kind and model.
* **P10. A Responses or Interactions text cut** ends the turn `max_tokens` with its text kept (0065 R22); its cause line stays
  0065's Phase 1.
* **P11. A cut with only reasoning** is the raw `llm.ErrIncomplete`, as at `v1.32.0` (0065 R23, still 0065's Phase 1).
* **P12. Kilo's window form, now an overflow**: L39 resends once at a stated room of at least 1; a room at or below 0 compacts
  (L11); 0065 R20's outcome.
* **P13. The SDK's paths for a limit it chose** (its resend at a stated cap, its first-payload refusal resend, `Generate` over a
  stream, the route fallback): out of scope; gobble names every limit and streams.
* **P14. `Response.OutputLimit` reaches `llm`**: `llm.SentLimit {Tokens int; Sent bool; Basis, Source string}` and
  `llm.Done.OutputLimit` (`json:"outputLimit,omitzero"`), set from the done event's response.
* **P15. `APIError.Observed` reaches `llm`**: `llm.LimitObservation {Model, BaseURL, Scope string; Tokens int; Seen
  time.Time}` and `llm.APIError.Observed`, set in `mapErrorAt` (`provider.go:873`), through `WithStreamRetry`.
  `llm.WarningToolCallCut` and `llm.WarningLimitLearned` are the SDK's strings. A warning's `Observed` is not carried: the
  SDK delivers it only on a response after it resent a limit it chose (`limit_rule.go:215-227`, `:267-279`), which no gobble
  call gets.
* **P16. 0070's record** gains `out_limit`, `out_limit_sent`, `out_limit_basis` and `out_limit_source` from
  `Done.OutputLimit` on a call that completes, and none on one that fails before its done event (0066 D8 as 0070 D19 places
  them; 0070 T14 amended: its `no_out_limit` case becomes these).
* **P17. V1, measured.** Each call reads `catalog.CachedLimits` with the options the lookup uses (`llm/provider/model.go:175-186`:
  the transport and the run's cache, the workspace, the layers) and `WithLimitObservations` of the saved caps. `C` is
  `StreamedOutput.Tokens` when known (gobble streams). Its source is `user` when the chosen evidence is the "models.json" layer
  and models.json states `maxTokens` (`Facts.MaxTokensStated`); `assumed` when the "models.json" layer gives only Pi's default
  or nothing is known (16,384, L5, L6); else `catalog` (a listing, the copy, an observed or remembered cap). `W` is `Window`,
  else `Input`, else unknown. The facts ride 0066 D4's `llm.Model.Limits`, which gains `StreamedOutput`, filled by
  `modelInfoFrom`; `turnLimits.limit` and `compactionModel` (`acpserver/outputlimit.go:50-61`, `:29-33`) read it. D2 and D3
  are unchanged.
* **P18. V2, measured.** Already true at the tag: `toRequest` sets `MaxOutputTokens` (`llm/provider/provider.go:647-648`),
  which the SDK sends as given, basis `request` (`limit_rule.go:54-61`), on every wire but the ChatGPT routes (L24).
* **P19. V4, measured.** An error whose `llm.APIError.Observed.Tokens` is at least 1 and below the limit sent: the call is sent
  again once at it, basis `refusal`, before any compaction (D8 (a), L44's hold); the observation is saved through a new
  `agent.Config.SaveObservation` to 0066 D7's `limit-observations.json`; `model.limit.learned` is said once (0066 D10). The
  SDK's observed cap then bounds the process's later calls to that model (F24). The saved caps are passed to `New`, to
  discovery and to every `CachedLimits` with `WithLimitObservations`.
* **P20. A refusal with no `Observed`** (a window form, or a cap form the SDK did not validate): L39's reading of the text as
  built (`agent/agent.go:778-791`); resent once; nothing saved. Refused again: L12.
* **P21. What a saved cap does on the next run.** Open: Q6 (F24).
* **P22. V5 and V6, measured.** `/context` and `/session` show the last call's `Done.OutputLimit`: sent, "Output limit: N
  tokens on the last call (D10's basis words)", N the wire's value, with "; raised from M for the thinking budget" when the
  SDK raised gobble's M (F9); not sent (a ChatGPT route), "Output limit: none sent on the last call; the service's default
  applies". `_meta.gobble.outputLimit` and `--stats` carry the same N; not sent, tokens 0 and a new basis `none`
  (`llm.BasisNone`). Before the session's first call, D10's line as built.
* **P23. Exported names the pin and Phase 2 add** (gobble is below `v1.0.0`): P3's three; P14's and P15's; `llm.BasisNone`;
  0066 D4's `llm.Model.Limits` (as `llm.Limits`, with `StreamedOutput`, and `llm.LimitFact`); `agent.Config.SaveObservation`;
  `provider.Options.Observations`; `provider.Discover` (0066 D1); under Q6 (1), `llm.BasisRemembered`. Any other is a stop.

### State table: the pin

"The tag" is `v1.33.0`; "today" is 073fd1f. Each decided row's plant is in the PLAN, one to one.

| Row | Input or state | Status | Outcome | Test | Decided by |
| :--- | :--- | :--- | :--- | :--- | :--- |
| P1 | gobble's SDK requirement | decided | `v1.33.0`, no `replace` | the module builds at the tag with P3's and P14's names | the ruling; P1 |
| P2 | The six production and test sites of the removed names | decided | P2's table | `go vet ./...` on every module at the tag | P2; 0093 D10 |
| P3 | A document for a listing, before 0072 | decided | `Options.ModelData`, nil none; `ParseModelData` | `TestListHandsModelData_0071` | P3; 0072 D21 |
| P4 | The tests that served a document by URL | decided | the same bytes handed in; expectations unchanged | their own names | P4; 0093 D12 |
| P5 | The tests of the fetch | decided | `TestListFetchesMetadata` deleted; `TestListFullCatalog` counts one request | `TestListFullCatalog` | P5; 0093 MH30 |
| P6 | The document production hands each listing, lookup, limits call and `New` before 0072 | open | Q5 | Q5's option's test | Q5 |
| P7 | A turn, a summary or a title on a cold Claude wire | decided | each names a limit; none refused before sending | `TestEveryCallNamesLimit_0071` | P7; F21 |
| P8 | A test refused before sending at the tag | decided | `MaxTokens` 16,384; other assertions unchanged | each listed test, green | P8 |
| P9 | A cut answer with whole tool calls | decided | the calls run; the turn goes on; `tool_call_cut` logged once | `TestAgentTurn_SalvageCompleteCalls_0065` | P9; 0065 R24 |
| P10 | A Responses text cut | decided | `max_tokens`, its text kept | `TestResponsesCutTextStops_0071` | P10; 0065 R22 |
| P11 | A reasoning-only cut | decided | the raw `llm.ErrIncomplete`, as today | `TestReasoningOnlyCutUnchanged_0071` | P11 |
| P12 | Kilo's window form | decided | room at least 1: resent at it; else compacted and resent | `TestAgentTurn_KiloWindowFormCompacts_0065` | P12; L11, L39 |
| P13 | The SDK's paths for a limit it chose | out of scope | gobble names every limit and streams | | P13 |
| P14 | The limit a call carried | decided | `llm.Done.OutputLimit` from `Response.OutputLimit` | `TestDoneCarriesOutputLimit_0071` | P14 |
| P15 | A refused limit's stated cap | decided | `llm.APIError.Observed`; a window form none | `TestAPIErrorCarriesObserved_0071` | P15 |
| P16 | 0070's record | decided | the four `out_limit*` fields on a completed call; none on a failed one | `TestModelCallRecordOutLimit_0071` (T14's case replaced) | P16; 0066 D8 |
| P17 | `v1.33.0`: a call's `C`, its source, and `W` | decided | `StreamedOutput`; `user`, `assumed` or `catalog`; `Window`, else `Input` | `TestTurnCapFromLimits_0071` | P17; V1 |
| P18 | `v1.33.0`: the turn's and summary's limit on every wire | decided | basis `request`, D2's value | `TestExplicitLimit_0071` | P18; V2 |
| P19 | `v1.33.0`: a refusal with `Observed` below the limit sent | decided | one resend at it; saved; learned said once | `TestRefusedCapLearned_0071` | P19; V4 |
| P20 | `v1.33.0`: a refusal with no `Observed` | decided | L39 as built; nothing saved | `TestRefusalNoStatedCap_0071` | P20; V4 |
| P21 | A saved cap on the next run | open | Q6 | Q6's option's test | Q6 |
| P22 | `v1.33.0`: the line, `_meta` and `--stats` | decided | the wire's value; the raise named; ChatGPT "none sent" | `TestContextShowsSentLimit_0071` | P22; V5, V6 |
| P23 | Exported names | decided | P23's list; any other a stop | the PLAN's stop rule; `make apidiff`'s no-op | P23 |

Counts: 20 decided, 1 out of scope, 2 open; 23 rows.

### Owner questions (2026-11-25)

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| Q5. Until 0072 keeps its own copy, which model document should gobble hand the SDK? Today the model list is ranked by a live copy the SDK downloads from opencode.ai; from `v1.33.0` the SDK downloads nothing. | (1) **Hand the built-in copy to the model lists (Recommended)**: gobble's built-in 2026-10-14 models.dev copy ranks every model list; lookups and limits read it as they do today; a provider's requests read none, as today. (2) **Hand none**: the model lists rank by the SDK's curated order and name rules until 0072; lookups as today. (3) **Hand the built-in copy everywhere**, providers included, which 0072's end state does with its own copy. | (1) the lists rank by models.dev data six weeks old instead of a live copy, so a model released since 2026-10-14 ranks by name; 0072's Phase 1 swaps in its own copy, a few lines. (2) every list but Kilo's and Ollama's changes order now, and again when 0072 lands. (3) requests change now (OpenCode routes and the effort a model is offered come from the copy), and again when 0072 hands each session its own document; more tests to change in the pin. | P6 |
| Q6. When a service refuses gobble's output limit and states its cap, gobble saves the cap for a day. The SDK uses a saved cap only for a model nothing else describes, so gobble's next run asks the old, larger limit once, is refused once, and learns the cap again. Should gobble apply its saved caps itself? | (1) **Yes, gobble holds its limit to a saved cap (Recommended)**: a cap saved within the last 24 hours for that model, address and workspace bounds the limit gobble asks, shown as "a cap a service stated on an earlier run". (2) **No, as the SDK ranks it**: the saved file only fills in a model no source describes. (3) **Keep no file**: drop the day-long store; each run learns from its first refusal. | (1) one rule and one shown reason more; if the service raises its cap within the day, answers stay cut at the old cap until the entry expires. (2) one refused request per model per run where the stated cap is below what gobble asks; whether a refused request is billed is not known. (3) as (2), and no state file, no `model.limit.learned` on a later run; 0066 D7's store and V4's "saved" are struck. | P21; 0066 D7, S14 |

### Relation to the other records' amendments of 2026-11-25

* `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md`: V8 and V9 decided; its Phase 2 bump is this record's Phase 2a.
* `0066-MADR-set-each-session-model-up-from-discovered-limits.md`: V7 decided; D3 re-pointed to `catalog.WithMetadata`; its
  Phases 1 and 2 are built as this record's Phases 2b and 2c.
* `0072-MADR-keep-a-refreshed-models-dev-copy-on-disk.md`: M22 struck; M54, M55, M62 and its Phase 4 re-pointed; P3's names
  taken from its D21.

### What could not be verified here

* That `v1.33.0` is in the Go module proxy: not fetched, since this records-only step made no network request; Phase 2a's
  `go get` is the check.
* The full list of tests the tag turns red: no build was run (a build at the tag writes the module cache); Phase 2.0's red run
  on a scratch clone lists them, and P8 and the stop rule govern each.

## Amendment of 2026-11-25, second: the owner's answers and the review

*(2026-11-25. Additions only. The owner answered Q5, Q6 and the review's cases 4, 5, 7, 9 and 10, relayed by the
coordinator; the review's correctness gaps and stale claims are decided or corrected below. No row is open, so both of this
date's amendments are `accepted` with the rest of the record.)*

### Owner answers, quoted exactly

| Question or case | The owner's answer | Rows |
| :--- | :--- | :--- |
| Q5. The document before 0072 | "Built-in copy to listings only (Recommended)" | P6 |
| Q6. Saved caps on the next run | "Hand them in as a caller layer (Recommended)" | P21, P24 to P27 |
| Case 10. models.json `maxTokens` above a cap the service stated | "The service's cap wins, shown as such (Recommended)" | P28 |
| Case 4. Text, then a cut tool call, on Chat Completions or Messages | "End the turn, keep the text, notify (Recommended)" | P29 |
| Case 5. A title cut at its limit | "Discard a cut title (Recommended)" | P30 |
| Case 7. What the line shows | "gobble's basis, wire value (Recommended)" | P31 |
| Case 9. A failed call, and a loaded or forked session | "Show what was asked, marked (Recommended)" | P32 |

### Stale claims corrected

* **F21 and P8.** The red list is far longer than five tests: every test that streams through `wireStream`
  (`llm/provider/wire_test.go:131-156`) on a Claude or OpenCode messages-route case sends no limit, and is refused at the tag.
  `wireStream`'s callers are `TestWireDeferral`, `TestDeferralAfterCancelledTurn`, `TestWireDeferralGrokCall`
  (`deferral_test.go`), `TestWireLimits`, `TestWireLongWaitIsNotAWindow` (`limit_test.go`), `TestWireOffline`,
  `TestWireEmptyAnswer` (`offline_test.go`), `TestWireRefusal` (`refusal_test.go`), `TestWireReplay` (`replay_test.go`),
  `TestWireCases`, `TestWireRateLimitResets`, `TestWireOverflowSplit`, `TestWireToolResultError` and `TestWireStrict`
  (`wire_test.go`), by a scan of `llm/provider/*_test.go` (`<scratch>/g-pin133/wiretests.py`). P8 is re-pointed: the fix is
  one line in the helper, `wireStream` setting `req.MaxTokens = llm.AssumedMaxTokens` when the request names none (beside
  `req.Model = model`, `:137`), which covers every caller, `wireRequest()`'s (`:158-160`) and the others' alike; the direct
  `New` sites F21 names (`record_test.go:381`, `:484`, `replay_test.go:184`, `:206`, `workspace_test.go:18`) take the same
  value. No test or golden asserts a sent 8,192 (`git grep` over `llm/provider/*_test.go` and both `testdata` trees: none).
* **F24.** A remembered cap is sent on a route where the SDK chooses the limit (Claude, OpenCode's messages route, Kilo) when no
  other source states the model's output, and on an optional route it only cuts a given `WithMaxTokens`
  (`limit_observation.go:39-43`). gobble names every limit, so neither use reaches its calls; the conclusion stands, and Q6's
  answer replaces `WithLimitObservations` with a caller layer (P24).
* **P4.** `chatserver_test.go:122` sets `ModelMetadataURL` to a closed address (`http://127.0.0.1:1/v1/meta.json`) and serves
  no document; its fix is to delete the field, not to hand bytes in.
* **The DL39 premise.** The SDK's drop of a kept cap that a response's output exceeds (DL39) acts on its process memo only
  (`internal/limitmemo/limitmemo.go:134`; each provider's `observe`, such as `providers/kilo/kilo.go:531`), not on a caller's
  layer. A saved cap handed as a layer is therefore not dropped by the SDK; gobble drops it itself (P25), which carries out the
  owner's answer as intended.

### Decided

* **P6 (Q5).** Production's listings get the built-in 2026-10-14 copy as `Options.ModelData`; the lookups and limits calls keep
  it as today's layer; `New` and the wizard hand none.
* **P21 and P24 (Q6).** `provider.Options.Observations` (the saved caps of the provider being looked up or built) becomes an
  unexported `catalog.Layer` named "service-stated", stating only `FieldMaxOutputTokens`, for an entry whose model matches in
  lower case, whose base URL and scope are the options' and whose `Seen` is within 24 hours. It is handed with
  `catalog.WithModelLayer` before the models.json layer, so it ranks under it (`catalog/limits.go:336-345`), to every
  `modelInfoFrom` lookup, every `CachedLimits`, `provider.Discover`'s `LookupLimits` and every `New`. The SDK's ranking then
  applies it: a caller's layer outranks the listing and the document. `WithLimitObservations` is not passed.
* **P25.** When a call that completes reports output tokens above a saved cap of its provider, model, base URL and scope, the
  store drops that entry, as the SDK's DL39 drops its memo's.
* **P26.** Each saved cap carries its provider id; `internal/cli` hands each provider only its own.
* **P27.** On load, an entry with no provider, no model, `Tokens` below 1 or a zero `Seen` is dropped, with one warning per run,
  as S13 says of an unreadable file.
* **P28 (case 10).** When models.json's `maxTokens` is above a cap a service stated (this process's "observed" evidence, to
  which the SDK already holds the chosen value, `catalog/limits.go:450-475`; or a saved cap, which ranks below models.json),
  gobble sends the service's cap, basis `service` ("the cap the service stated"), and says once per session and model:
  "models.json asks for N output tokens for MODEL; the service stated M, which gobble sends." (personality key
  `model.limit.user_over`). For a saved cap this is the one rule gobble adds: the chosen models.json value is held to the
  "service-stated" evidence below it.
* **P29 (case 4).** A Chat Completions or Messages answer with text and then a cut call is, at the tag, a `length` response
  with the text and a `tool_call_cut` warning: the turn ends `max_tokens`, its text kept, the warning logged, and the user told
  with `turn.stop.max_tokens_tool` (0065 D7's key; 0065 R21 and R25 left its line unwritten, so it is decided here): "The reply
  hit the output limit while writing a tool call, so the call was dropped. Ask again, or set maxTokens for MODEL in
  models.json." The channel is 0065 D8's: an ACP agent message after the stop, the CLI's line, headless print and JSON. A reply
  that is only a cut call stays the raw `llm.ErrIncomplete` (`ErrEmptyAnswer`).
* **P30 (case 5).** A title whose call ends `max_tokens` is not used, as a summary's is not (`compaction/compaction.go:556-558`):
  its usage is written, the session takes no name from it, and the title's error is logged.
* **P31 (case 7).** The line shows the wire's tokens with gobble's own basis. An SDK basis other than `request` or `none` is
  logged at warn once per session and model and shown raw, as `sdk:<basis>`. "Refusal" has one meaning in gobble's words: its
  `refusal` basis, a limit held to a room or cap a service stated in the turn; the SDK's `refusal` can appear only raw, as
  `sdk:refusal`.
* **P32 (case 9).** A call that fails shows the value asked, marked: "Output limit: N tokens asked on the last call, which
  failed (basis words)"; `_meta.gobble.outputLimit` carries it with `asked: true`. A loaded or forked session shows D10's "none
  sent yet" line until its first call in this process.
* **P33 (review gap 1).** `CachedLimits` and `LookupLimits` get the session's credential (`llmprovider.WithTokenSource`, the
  stored login or the key, `llm/provider/model.go:82-90`) and `llmprovider.WithClientInfo`, so they key the cached listing as
  the lookup does (`catalog/limits.go:208-214`).
* **P34 (review gap 6).** Compaction's ladder (`compaction/compaction.go:415-422`) and the title read `llm.APIError.Observed`
  first, as the turn does (P19): one resend at it, the cap saved. D8 (c)'s check (`acpserver/outputlimit.go:123`) also reads a
  refused-again call by `Observed`, since a form `llm.ReadLimitRefusal` does not read, Kilo's range form
  ("限制数值范围[1,M]", the SDK's 0089 DL95), arrives only as `Observed`.
* **P35 (review gap 8).** A call with nothing sent carries `{tokens: 0, basis: "none"}` through `turnMeta`
  (`acpserver/agent.go:914`, which today keeps only `Tokens > 0`) and `acpclient` (`acpclient/conn.go:353`, likewise).
* **P19, re-pointed.** The saved caps reach later runs by P24's layer, not by `WithLimitObservations`.
* **P23, re-pointed.** Added: `llm.BasisService`, `personality.TurnStopMaxTokensTool`, the `model.limit.user_over` key,
  `acpclient.OutputLimit.Asked`. Dropped: `llm.BasisRemembered`.

### State table: the answers and the review

| Row | Input or state | Status | Outcome | Test | Decided by |
| :--- | :--- | :--- | :--- | :--- | :--- |
| P6 | *(decided 2026-11-25)* The document before 0072 | decided | the built-in copy to the listings only | `TestListingRanksByBuiltInCopy_0071` | Q5 |
| P21 | *(decided 2026-11-25)* A saved cap on the next run | decided | a caller layer below models.json; the SDK ranks it | `TestSavedCapLayer_0071` | Q6 |
| P24 | Where the saved caps' layer reaches | decided | every lookup, `CachedLimits`, `Discover` and `New` | `TestSavedCapLayer_0071` (`lookup`, `limits`, `discover`, `new`) | Q6; P24 |
| P25 | A completed call's output above a saved cap | decided | the entry dropped | `TestSavedCapDroppedWhenExceeded_0071` | Q6; P25 |
| P26 | A saved cap and another provider | decided | handed only to its own | `TestSavedCapPerProvider_0071` (an opencode-zen cap is not handed to claude) | P26 |
| P27 | A readable file with a bad entry | decided | the entry dropped; one warning | `TestAgent_UnreadableObservations_0066/bad_entry` | P27; 0066 S13 |
| P28 | models.json `maxTokens` above a service's cap | decided | the service's cap, basis `service`; one note | `TestTurnCapFromLimits_0071/user_over_observed`, `/user_over_saved` | case 10 |
| P29 | Text, then a cut call *(2026-11-25, third: its four lines decided, "Approve all four (Recommended)"; with messages queued, P36)* | decided | `max_tokens`, text kept, logged, `turn.stop.max_tokens_tool`; only a cut call: the raw error | `TestCutCallAfterText_0071` (`chat`, `messages`), `TestOnlyCutCall_0071` | case 4 |
| P30 | A title cut at its limit | decided | not used | `TestTitleCutDiscarded_0071` | case 5 |
| P31 | The line's tokens and basis | decided | wire tokens, gobble's basis; another SDK basis raw and logged | `TestContextShowsSentLimit_0071/sdk_basis_raw` | case 7 |
| P32 | A failed call; a loaded or forked session | decided | "asked", marked; D10's line until a call | `TestContextShowsSentLimit_0071/failed_asked`, `/loaded_session`, `/forked_session` | case 9 |
| P33 | The credential of a limits read | decided | the session's token source and client info | `TestTurnCapFromLimits_0071/listing_only` | P33 |
| P34 | Compaction, the title and the line on `Observed` | decided | one resend at it, saved; the line reads it | `TestSummaryCapRefusalResent_0071/observed`, `TestTitleObservedResent_0071`, `TestAPIErrorCarriesObserved_0071/kilo_range`, `TestLimitRefusalLine_0071/refused_again_observed` | P34 |
| P35 | A call with nothing sent, over ACP | decided | `{0, "none"}` in `_meta` and `acpclient` | `TestPromptResponseOutputLimit_0071/none` | P35 |

Counts after this amendment: 83 decided, 7 out of scope, 0 open; 90 rows (P6 and P21 decided; P24 to P35 added).

## Note of 2026-11-25: Phase 2.0's red list, measured

*(2026-11-25, a fact corrected by measurement; no decision changes. The PLAN's execution record holds the run.)* Phase 2.0's
run at `v1.33.0` (`go test -count=1 ./...` on a scratch clone of `2cbec91` with P2's deletions and P3's seam as stubs) found 24
tests refused before sending by `limit_rule.go:78-81`, every one on a Claude or OpenCode messages route: ten of the fourteen
`wireStream` callers "Stale claims corrected" names (`TestWireDeferralGrokCall`, `TestWireLongWaitIsNotAWindow`,
`TestWireOffline` and `TestWireOverflowSplit` were not red); the tests of the direct `New` sites `record_test.go:381`
(`TestModelCallSignatureResend`), `replay_test.go:184` and `:206` (`TestWireSignatureRetry`) and `workspace_test.go:18`
(`TestWireWorkspace`); and eleven tests the scan did not list: `TestWireStatedWindow` and `TestWireBudgetLimit`, through
`limitErr`, which calls `wireStream`, and nine that build their own request, `TestCacheWire` and `TestSessionIDPerProvider`
(each counts its requests: 0, want 1), `TestImageWires`, `TestTextOnlyModelImage`, `TestStreamIsLive`,
`TestFailureAfterFirstDeltaNotRetried`, `TestProviderRetryReported`, `TestWireBaseURL` and `TestWireRejectedKey`.
`record_test.go:484` (`TestModelCallCache`) is not refused: its request, `callReq()`, names 4,096. P8's
row decides every one ("a test that Phase 2.0's red run finds refused by `limit_rule.go:78-81`"); the other reds were P4's
(`TestListingSearchByFacts`, `TestModelsSearchListsFactsFirst`) and P5's (`TestListFullCatalog`, `TestListFetchesMetadata`).

## Amendment of 2026-11-25, third: Phase 2a's review

*(2026-11-25. Additions only. The independent review passed Phase 2a with notes; the owner answered two questions it raised,
relayed by the coordinator. No row is open.)*

### Owner answers, quoted exactly

| Question | The owner's answer | Rows |
| :--- | :--- | :--- |
| The P29 notice's lines in dry, snarky and bombastic, written in Phase 2a beside boring's: are they the decided text? | "Approve all four (Recommended)" | P29 |
| Text, then a cut call, while steering or follow-up messages are queued: the turn goes on with them (`agent/agent.go:386-403`), so it does not end `max_tokens`; is the notice shown? | "Show the notice anyway (Recommended)" | P36 |

### Decided

* **P29, its lines.** `turn.stop.max_tokens_tool` is, word for word: boring, "The reply hit the output limit while writing a
  tool call, so the call was dropped. Ask again, or set maxTokens for {model} in models.json."; dry, "The reply ran out of room
  in the middle of a tool call, so the call was dropped. Ask again, or set maxTokens for {model} in models.json."; snarky, "Ran
  out of tokens halfway through a tool call, so that call is gone. Ask again, or set maxTokens for {model} in models.json.";
  bombastic, "Even genius hits the output limit: it cut a tool call mid-stride, so the call was dropped. Ask again, or set
  maxTokens for {model} in models.json." Each built-in personality has that one variant.
* **P36.** When a call ends `max_tokens` with a tool call dropped (P29) and queued steering or follow-up messages carry the
  turn on, the notice is sent once, as P29's agent message, before the first queued message is shown; the turn then goes on
  as it does today. A later call's own cut is noticed again.

### Measured: a handed document and Kilo's and Ollama's listings

Phase 2a hands the built-in copy to every listing (P6), Kilo's and Ollama's included, which `v1.32.0` fetched no document
for (`docs/README.md:240`). Read at go-llmprovider-sdk `v1.33.0`:

* **Ollama:** its menu is built with no document: `listingResult.catalog` calls `ollamaMenu` with `factsOf(…, nil)`
  (`catalog/discovery.go:547`). A handed document changes nothing.
* **Kilo:** its band is `kiloCurate` over Kilo's own entries, no document (`discovery.go:723`; the comment above
  `buildListing`, `:701`). The document reaches only the facts `menuOf` reads (`discovery.go:564`): which models carry a
  caveat (`menu.go:56`: status, tools, output, retirement) and the newest-first order of the models after the band
  (`menu.go:78`). In those facts the listing's own fields replace the document's (`lookup.go:206`, `mergeLayers`: models.dev,
  then the listing), so the document counts only where a Kilo entry states nothing. On the SDK's own 391-model Kilo fixture
  (`catalog/testdata/ranking-2026-10-12/kilo.json`), every entry states its tools and output; only the five `kilo-auto`
  routers state no release date, and the copy gives them one, but they rank in the band, not after it. The copy states no
  retirement and no deprecated status for a Kilo model. A probe on a scratch copy listed that fixture through `provider.List`
  with the built-in copy and with none: Recommended (276 models) and the menu (285) were identical, and nine models
  carried caveats in each. So the built-in copy does not change Kilo's or Ollama's listing order. A later document could, for a Kilo
  model it marks deprecated or retiring: 0072's refreshed copy meets that, not this pin.

### State table: the review

| Row | Input or state | Status | Outcome | Test | Decided by |
| :--- | :--- | :--- | :--- | :--- | :--- |
| P29 | *(annotated 2026-11-25, third)* The notice's text in each built-in personality | decided | the four lines above, word for word | `TestBoringIsTodaysText`, `TestToolCutLines_0071` | "Approve all four (Recommended)" |
| P36 | Text, then a cut call, with steering or follow-up messages queued | decided | the notice once, before the first queued message; the turn goes on | `TestCutCallNoticeBeforeQueued_0071` | "Show the notice anyway (Recommended)" |

Counts after this amendment: 84 decided, 7 out of scope, 0 open; 91 rows (P36 added; P29 annotated).
