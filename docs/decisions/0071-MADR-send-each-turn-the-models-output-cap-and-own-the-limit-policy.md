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
  (b) **An input overflow** takes today's path: compact once and resend, the resend's cap computed again (D2 runs per call).
  (c) **Any other refusal of the cap.** An invalid request whose message names an output-limit field (`max_tokens`,
  `max_completion_tokens`, `max_output_tokens`, `maxOutputTokens`, without case), the SDK's "fills the context window"
  reason, or a cap-caused refusal still refused after (a)'s resend, fails the turn, and its error text gains: "gobble asked
  for at most N output tokens (B); to send less, set maxTokens for MODEL under modelOverrides in models.json."
  (d) **Credit and rate.** A 402 or quota failure, a 429 or rate-limit failure that outlasts the SDK's retries, and Kilo's
  503 for an upstream 402, on a call that carried gobble's cap, fail the turn as today, and the error text gains: "gobble
  asked for at most N output tokens (B); some services hold credit or rate against that number; set maxTokens for MODEL
  under modelOverrides in models.json to ask for less." Nothing is retried at a guessed smaller value.
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
  `acpclient.Result.OutputLimit` and its type.

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
| L15 | A 402, a 429 after the SDK's retries, or Kilo's 503 for a 402, on a capped call | decided | the turn fails; D8 (d)'s line | D8 (d) |
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
after Q4's answer: 45 decided, 6 out of scope, 0 open; 51 rows.)*

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

### What could not be established offline

* Whether each service accepts 64,000 for GLM-5.3, and whether a gateway holds credit or rate against it: Q1's live calls.
* How far chars/4 runs from GLM's tokenizer on code; the 4,096 margin is pi's, not a measurement.
