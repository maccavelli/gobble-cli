---
status: accepted
date: 2026-10-08
decision-makers: Project Owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Tool schemas, descriptions and loading: JSON Schema from Go types, sectioned Markdown descriptions, two exposure tiers, and an in-house BM25 `tool_search`

## Context and Problem Statement

[0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) decides which tools gobble has. Its amendment of 2026-10-08, "native tools", added move, delete, copy, mkdir, `tree`, `lsp`, the job tools, `monitor`, `notebook`, documents in read, and `request_permissions`. It does not decide four things every one of those tools needs:

1. how a tool's parameters are described to a model;
2. how its description is written;
3. whether the model sees it on every request, or only after a search;
4. how that search ranks tools.

The owner proposed, on 2026-10-08:

> For those native tools let's determine their schemas. I propose using json-rpc 2.0, and for tool descriptions use hybrid markdown. Lazy load all but tool_search and write a harness instruction to always consult tool_search before performing file or directory operations. Evaluate this, offer other ideas, recommendations, pros and cons of the approach. If we want or need bm25 search we can use bleve for that.

[0011-REPORT-native-tools-survey.md](../reports/0011-REPORT-native-tools-survey.md), section "Schemas, descriptions and loading (2026-10-08)", evaluated the proposal and decided nothing. The owner then asked for this record: "grounded in the report and research facts write the madr. ensure it is detailed, complete, and proves any assertions".

This record decides the four questions, gives every 1.0 tool its schema, and says what it changes in 0005-MADR.

### What was measured, not assumed

**Claim checks.** Every factual claim below carries an id, and the Evidence index gives its file and line.

- **Ids from 0011-REPORT** are written `0011 S5`, `0011 T6` and so on. They are that report's evidence table: 88 claims, re-run on 2026-10-08 for this record (`q178_claims.py` 43 claims, `q180_claims2.py` 62 including those 43, `q191_claims3.py` 26), all passing.
- **Ids V1–V30 are new to this record.** A script in the session scratchpad, `q200_claims_0012.py`, opens each cited file and checks the claim:
  - a positive claim passes when a pattern stating it matches a line of the file;
  - an absence claim passes when its pattern matches nothing.

  The run on 2026-10-08: 30 claims, 0 failed.
- **The check was seen to fail.** `q200_claims_0012.py --break` inverts every claim:
  - each positive claim gets a pattern that cannot match;
  - each absence claim (V2, V25) becomes a positive claim for a pattern that is in the file.

  That run exited 1. All 28 positive claims failed with "pattern not found". The two absence claims found their present pattern (`agent/agent.go:23`, `llmprovider/internal/wire/tools.go:9`), which shows those checks read the files they name.
- **A repository-wide search.** `grep -rn "ExposureDeferred\|ExposureDirect" --include=*.go .` found one use outside `tool/tool.go`: `mcpclient/tool.go:43`, which sets `ExposureDirect` (V4).

**Probe A: the tool definitions gobble sends today.** `specsize`, a scratch module, ran on 2026-10-08 on Windows, against this working tree with F1a staged and F1b unstaged:

- **What it does.** It replaces the gobble module with the local checkout, calls `builtin.Tools()`, and prints each tool in Anthropic's Messages shape: `{name, description, input_schema}`, encoded with `encoding/json/v2`.
- **What it does not do.** The bytes are that encoding, not a capture of a request, and they are bytes, not tokens.

| Tool | Description bytes | Schema bytes | Definition bytes |
| :--- | ---: | ---: | ---: |
| read | 428 | 387 | 865 |
| write | 137 | 275 | 461 |
| edit | 400 | 746 | 1,194 |
| bash | 389 | 402 | 839 |
| powershell | 461 | 402 | 917 |
| **5 tools** | 1,815 | 2,212 | **4,276** |

What the probe also shows:

- Every schema already has `"required"` for its non-optional fields and `"additionalProperties":false`.
- **edit's `edits` is typed `["null","array"]`, with no `minItems`,** although it is required and its description says "one or more replacements". So the schema lets a model send `null`, or an empty list.
- No integer has `minimum` or `maximum`. read's "at most 2000" and bash's "at most 600" exist only as prose in a property description.
- No tool sets an exposure (`exposure=""`).

**Probe C: the validator's messages** (added 2026-10-08, while writing the PLAN). `valmsg`, a scratch module on jsonschema-go v0.4.3:

- **The setup.** It derives a schema from an edit-like Go type, makes `edits` a non-null array with `minItems: 1`, gives `oldText` `minLength: 1`, and bounds `limit` to 1–2000.
- **The messages.** It validated five arguments, and `Resolved.Validate` printed:
  - for `edits: []`: `validating root: validating /properties/edits: minItems: array length 0 is less than 1`;
  - for `edits: null`: `… type: <invalid reflect.Value> has type "null", want "array"`;
  - for an empty `oldText`: `… minLength: "" contains 0 Unicode code points, fewer than 1`;
  - for `limit: 0`: `… minimum: 0/1 is less than 1.000000`;
  - for `limit: 5000`: `… maximum: 5000/1 is greater than 2000.000000`.

**Probe D: what OpenAI's Responses API does with gobble's schemas** (added 2026-10-08, before the PLAN's approval). `strictprobe`, a scratch module, sent `builtin.Tools()` on Windows through gobble's own `llm/provider` adapter.

- **The setup.**
  - Provider `openai`, which posts to `/v1/responses` (the SDK's `providers/openai/openai.go:264`).
  - Models `gpt-4.1-mini` (served as `gpt-4.1-mini-2025-04-14`) and `gpt-5` (`gpt-5-2025-08-07`).
  - The key came from the owner's key file and stayed in the probe's memory. Every printed line was redacted, and the log was checked for key-shaped text before it was read.
- **Three variants of the schemas:**
  - **today:** as gobble publishes them;
  - **planned:** reshaped as 0012-PLAN P2–P3 would publish them;
  - **nullable:** OpenAI's own strict form (Doc 5), with every optional property required and nullable.
- **One sample per model, variant and call.** The numbers below are observations, not rates.

| Observation | gpt-4.1-mini | gpt-5 |
| :--- | :--- | :--- |
| `strict` reported for each of the 5 tools, in the today and planned variants | `true` ×5, both | `true` ×5, both |
| read's parameters as OpenAI echoes them back (today) | `"required":["limit","offset","path"]`, with no `null` added | not printed |
| read's arguments: today, then planned | `{"limit":20,"offset":0,…}`, then `{"limit":20,"offset":1,…}` | `{"limit":2000,"offset":1,…}`, both |
| bash's arguments: today, then planned | `{"command":"echo hi","timeout":120,"workdir":"./"}`, then the same with `"."` | `"timeout":120,"workdir":"."`, both |
| gobble's validation of the arguments in those two variants | all accepted | all accepted |
| nullable variant: read's arguments | `{"limit":50,"offset":0,…}` | `{"limit":2000,"offset":1,…}` |
| nullable variant: bash's arguments | `"workdir":null` | `"workdir":null` |
| nullable variant: gobble's validation of bash | refused: `type: <invalid reflect.Value> has type "null", want "string"` | refused, the same |

**Probe B: bleve.** `bleveprobe`, measured on 2026-10-08 for 0011-REPORT §4. The numbers, and the flags used, are in that report's table and are not repeated here.

- It indexed six tool descriptions in memory with BM25, and searched them once.
- The binary was 15,223,808 bytes, against 1,650,688 for hello-world and 14,364,160 for gobble.
- 27 module requirements after `go mod tidy`, against gobble's 24.

**Provider documentation, fetched on 2026-10-08.** Quoted verbatim, with the URLs under Related records. A script cannot check a web page, so these quotes are the evidence.

- **Doc 1:** Anthropic, prompt caching.
- **Doc 2:** Anthropic, tool search tool.
- **Doc 3:** OpenAI, prompt caching.
- **Doc 4:** OpenAI, tool search.

0011-REPORT quotes all four. The fifth is new to this record:

- **Doc 5: OpenAI, function calling,** on `strict`:
  - "If you omit `strict`, the default depends on the API: Responses requests will attempt to normalize your schema into strict mode when possible, and will fall back to non-strict, best-effort function calling if the schema cannot be made compatible with strict mode. Chat Completions requests remain non-strict by default."
  - "When fallback happens, the response tool will show `strict: false`."
  - Strict mode needs "`additionalProperties` … set to `false` for each object" and "All fields in `properties` … marked as `required`". Optional fields are written "by adding `null` as a `type` option".

**Inferred, not measured.**

- ~~**[unverified]** Whether OpenAI Responses normalises gobble's schemas to strict mode or falls back. Each has optional properties left out of `required`. The SDK's recorded ChatGPT-backend reply reports `"strict":true` for a tool sent without `strict` (V26), but that tool's only property was required. D3's probe settles this.~~ *(Settled 2026-10-08 by Probe D: it normalises them, by making every property required; F17.)*
- **[unverified]** Whether a definition's bytes track its tokens closely enough to set the budget in bytes. D5 sets its budget in bytes because bytes can be tested offline, and treats the number as a ceiling, not a token count.

### Findings

**Schemas**

- **F1. JSON-RPC 2.0 cannot describe a tool's parameters, and a model never sees it.**
  - A model receives a tool as a name, a description and a JSON Schema, in its provider's own field: Anthropic's `input_schema` (0011 S5), OpenAI Responses' and Chat Completions' `parameters` (0011 S6, S7).
  - JSON-RPC 2.0 is the envelope between processes. ACP frames are JSON-RPC 2.0 (0011 S8). So is MCP, whose tools carry a JSON Schema `inputSchema` (0011 S9).
  - *Consequence:* the schema language is JSON Schema whatever the transport is, and JSON-RPC's place is the process boundary.
- **F2. gobble already has one source of truth for schemas.**
  - Each tool's JSON Schema is derived from its Go input type by `jsonschema.For[In]` (0011 S2), from jsonschema-go v0.4.3 (V9).
  - It travels as `Spec.InputSchema` (0011 S1), and goes to the SDK unchanged (0011 S3).
  - Probe A shows it is already close to strict shape: `required` lists and `additionalProperties:false`.
- **F3. The derived schemas have two defects.**
  - jsonschema-go infers every Go slice as `["null","array"]` (V29), so edit's required `edits` (V6) accepts `null`, and has no `minItems` (Probe A).
  - Numeric bounds are prose, not `minimum`/`maximum` (Probe A). The library's `Schema` can carry `minimum`, `maximum`, `minItems`, `default` and `enum` (V30); gobble sets none of them.
  - *Consequence:* the schema says less than the description, and a model that trusts the schema can send a call the tool refuses.
- **F4. Strict mode is already partly in play on OpenAI.**
  - The SDK sends no `strict` flag on any tool (V25).
  - OpenAI Responses tries to make an omitted-`strict` schema strict, and falls back when it cannot (Doc 5).
  - A recorded ChatGPT-backend reply reports `strict:true` for a tool the SDK sent without the flag (V26).
  - opencode rewrites schemas per model (0011 S10).
  - *Consequence:* how gobble writes schemas already decides whether OpenAI enforces them. Which way gobble's own schemas go is **[unverified]**.

- **F17. On OpenAI's Responses API, no tool argument is optional.** *(Added 2026-10-08, from Probe D.)*
  - **What OpenAI does.** It normalises every gobble tool to strict mode: `strict: true` for all five tools, on two models, in both the today and planned variants. It does so by making every property required, without making any nullable. read's echoed parameters require `limit` and `offset`.
  - **What the models then do.** They invent values for what gobble meant to leave out:
    - `gpt-4.1-mini` asked read for 20 lines (later 50) where omitting `limit` gives 2000, even with `default: 2000` published;
    - both models sent bash a `timeout` and a `workdir`.
  - **The nullable form does not fix it.** In OpenAI's own strict form, models send `null` for `workdir`, which gobble's validator refuses today. `gpt-4.1-mini` still invented a `limit` of 50.
  - *Consequence:* this happens today, before any 0012 change. It contradicts D3's premise that strict normalisation is harmless. Fixing it means either opting out of strict mode per tool, which needs the SDK's per-tool `strict` flag (D13), or an adapter in `llm/provider`. That choice is the owner's.

**Descriptions**

- **F5. Every surveyed harness that writes descriptions uses Markdown.**
  - opencode writes each description as a Markdown file with headings (0011 D3), and codex keeps Markdown templates (0011 D4).
  - Pi's `tool_search` description has a Markdown heading (0011 D2), and Pi puts usage guidelines in the system prompt, apart from the description (0011 D1).
  - gobble's descriptions are Go strings: constants (V7), or `fmt.Sprintf` with the tool's limits (V8).
- **F6. Every definition is sent on every request, and its schema weighs as much as its description.**
  - The provider adapter appends every tool in the request (V3).
  - Today's five tools are 4,276 bytes, 1,815 of description and 2,212 of schema (Probe A).
  - *Consequence:* a description budget alone does not bound a tool's cost. The schema's property descriptions count as much.

**Loading**

- **F7. Exposure is declared, but nothing acts on it.**
  - `tool.Exposure` has `deferred` (V1). The agent never reads a tool's exposure (V2); the only use outside `tool/tool.go` sets MCP tools to `direct` (V4).
  - mcp.json's `exposure` and `toolExposure` are accepted, with a warning that they apply from 0005-PLAN F8 (V5).
  - 0005-MADR plans `direct`, `deferred` and `hidden` (V13), and a bridged Pi server with no `exposure` key as `deferred` (V24).
- **F8. No surveyed harness defers its own core tools.**
  - codex defers only third-party tools (0011 T1), and gives deferral guidance only to models that support search (0011 T2).
  - Pi's `tool_search` is off until something is deferred (0011 T3), and it searches only deferred tools (0011 T4).
  - grok-build's search finds MCP tools (0011 T5).
- **F9. Both providers advise keeping the frequently used tools loaded, and say a change to the tools breaks the cache.**
  - **Keep frequent tools loaded.**
    - Anthropic: "Keep your 3–5 most frequently used tools non-deferred" (Doc 2).
    - OpenAI: keep loaded "a small set of functions, or functions needed for most tasks" (Doc 4).
  - **Deferral is for many tools.** Anthropic: standard tool calling "is a better fit when you have fewer than 10 tools"; selection "degrades once you exceed 30–50 available tools"; and "All tools cannot be deferred" (Doc 2).
  - **Tools lead the cache prefix.**
    - Anthropic: "Cache prefixes are created in the following order: `tools`, `system`, then `messages`", and "Modifying tool definitions … invalidates the entire cache" (Doc 1).
    - OpenAI counts changes to "tool names, descriptions, schemas, ordering" as prefix changes (Doc 3).
  - **Only provider-native deferral keeps the cache.**
    - Anthropic: "prompt caching is preserved" (Doc 2).
    - OpenAI: "designed to preserve the model's cache", but "changing the loaded tool set will break the model's cache from that point forward" (Doc 4).
- **F10. gobble cannot use provider-native deferral or cache control today.**
  - go-llmprovider-sdk v1.2.1 has no `defer_loading`, `tool_reference` or `cache_control` (0011 T6).
  - 0005-MADR already records that "The SDK sends no cache hints" (V20).
  - *Consequence:* a tool loaded mid-session can only be added to the `tools` list, which changes the prefix and breaks the cache from that request on.
- **F11. 0005-MADR already commits to a stable prefix and recorded loads.**
  - Choice 8: "F3 builds a frozen per-session context baseline" (V16).
  - Loaded tools "are recorded in the transcript so they survive resume and fork" (V12).
  - `tool_search` "is active when any tool is deferred" (V14).
  - *Consequence:* loading has to be append-only and recorded, or it contradicts those decisions.
- **F12. A deferred file tool loses to the shell, and harness enforcement is possible.**
  - With bash loaded and move deferred, `mv` needs no search. That defeats rules 2 and 3 of 0011-REPORT, "The rule": exact paths for permissions, and a structured result.
  - codex catches `apply_patch` run through its shell tool and runs the native tool instead (0011 S11).
  - 0005-MADR choice 5 parses bash with `mvdan.cc/sh` (V17), whose parser gives a simple command as a `CallExpr` (V27).
- **F13. The direct list 0005-MADR accepted on 2026-10-08 omits two 1.0 tools, and includes tools most tasks never call.**
  - The amendment makes the job tools and `request_permissions` direct (V15). Its count of 17 names leaves out `todo` and `tool_search`, which it says F1d leaves unchanged.
  - `todo` is 1.0 and publishes ACP `plan` updates (V21).
  - The MCP resource tools are 1.0: `list_mcp_resources`, `list_mcp_resource_templates`, `read_mcp_resource` (V23).
  - `apply_patch` replaces edit and write for GPT-family models (V22).

**Search**

- **F14. BM25 is the common ranking, and it is small.**
  - codex and grok-build rank with BM25 (0011 X12, K16).
  - Pi's `tool_search` is an in-house BM25 ranker (V28) of about 110 non-blank lines with its tokenizer and stop words (0011 M5).
  - bleve does BM25 only when a mapping asks (0011 M1); its default is TF-IDF (0011 M2).
  - bleve costs about 13.6 MB of binary over hello-world, and 27 requirements (Probe B).
  - *Consequence:* a catalog of tens to hundreds of short texts does not need an index engine.

**Results and transport**

- **F15. A tool result already separates what the model reads from what a client shows.**
  - `Result.Output` is the model's text (V11). `Title`, `Summary` and `Diffs` are for the client (V10).
  - MCP tools may also carry an `outputSchema` (0011-REPORT, open questions).
- **F16. gobble's JSON-RPC surface for tools is already planned.** `gobble mcp serve` is a 1.x row of 0005-MADR (V19). MCP is JSON-RPC 2.0 with JSON Schema `inputSchema` (0011 S9).

## Decision Drivers

1. **The model calls the right tool with valid arguments,** on the first try, on every provider gobble supports.
2. **The prompt cache survives a session** (0005-MADR choice 8). Today the SDK can neither defer tools natively nor mark cache points (F10).
3. **Safety is enforced in the harness, not requested in a prompt.** A prompt instruction is advice the model may skip. A native tool's confinement, permission rules and undo apply only if its calls reach the tool (F12).
4. **One source of truth.** A schema comes from the Go type the tool decodes, so the two cannot drift (F2).
5. **The cost of every request.** Every definition is sent every time, and schemas weigh as much as descriptions (F6).
6. **No dependency without need** (AGENTS.md, Dependencies). A module is added only for a feature that needs it (F14).
7. **Consistency with 0005-MADR** where it has decided, and an explicit amendment where this record differs (F11, F13).

## Considered Options

There are four questions, each with its own options. Within each, the letter `A` is the chosen one.

- **I. The schema language**
  - **I-A:** JSON Schema 2020-12 derived from Go input types, under one set of conventions. JSON-RPC 2.0 stays the transport.
  - **I-B:** JSON-RPC 2.0 method definitions as the tool schemas (the owner's proposal, read literally).
  - **I-C:** hand-written JSON Schema files per tool.
- **II. Descriptions**
  - **II-A:** sectioned Markdown: a standalone first sentence, then fixed optional sections, kept in embedded `.md` templates under a tested budget, with cross-tool guidance once in the system prompt (the owner's "hybrid markdown", given a shape).
  - **II-B:** plain prose in Go string constants, as now.
  - **II-C:** free-form Markdown with no shape and no budget.
- **III. Loading**
  - **III-A:** two tiers. The tools most tasks need are direct. The rest are deferred, and load by family, or on a named event, or at session start. Loads are append-only and recorded. The harness catches plain shell file commands.
  - **III-B:** defer everything except `tool_search`, with a system-prompt instruction to always consult `tool_search` before file or directory operations (the owner's proposal).
  - **III-C:** every tool direct, no deferral.
  - **III-D:** two tiers as III-A, without the shell interception.
- **IV. Search ranking**
  - **IV-A:** an in-house BM25 over the tool catalog, with no dependency.
  - **IV-B:** bleve with its BM25 scoring (the owner's suggestion).
  - **IV-C:** substring and name matching, no ranking.

## Decision Outcome

Chosen: **I-A, II-A, III-A and IV-A.**

- **I-A** keeps the schema in the one layer a model reads (F1), from the one source gobble already has (F2), and fixes the two defects that layer has now (F3).
- **II-A** follows every surveyed harness (F5), and turns "hybrid" into a shape a test can enforce, so the cost stays bounded (F6).
- **III-A** follows both providers' guidance and every surveyed harness (F8, F9). It keeps the cache stable as well as the SDK allows (F10, F11). It makes the safety of the native tools hold whichever spelling the model picks (F12).
- **IV-A** matches what the three harnesses with search do (F14), and adds no module.

The owner's proposals stand where the evidence supports them:

- JSON-RPC 2.0 is the transport for the native tools when they are served (D4).
- Descriptions are Markdown (D5).
- `tool_search` is the one discovery path for deferred tools (D9).
- bleve is the named candidate where a persistent index is needed (D12).

### The decisions

**Schemas**

- **D1. JSON Schema, from Go types.**
  - Every tool's parameters are JSON Schema 2020-12, derived from its Go input type by `jsonschema.For[In]`, as now (F2).
  - A schema is never hand-written JSON. Constraints the struct tags cannot express are set in Go, beside the input type, on the derived `*jsonschema.Schema`, through a schema hook passed to `tool.New`.
  - JSON-RPC 2.0 is not a schema layer (F1).
- **D2. One schema convention for every native tool.** A test in `tool/builtin` walks every tool `builtin.Tools()` and `ToolsWith` can return, and fails on any breach:
  1. Property names are `camelCase` ASCII, as `oldText` is now.
  2. Every property has a `description`, from its `jsonschema` tag.
  3. A property is in `required` exactly when the tool cannot run without it.
  4. Every object has `additionalProperties: false`. jsonschema-go already does this (Probe A); the test pins it.
  5. **No property is nullable.** A Go slice field is given `"type":"array"`, overriding the inferred `["null","array"]` (V29). A list that must not be empty has `minItems: 1`.
  6. Every integer the tool bounds has `minimum` and `maximum`. Every optional property with a default has `default`. Every fixed choice is an `enum` (V30).
  7. **The portable subset.** No `$ref`, `oneOf`, `anyOf`, `allOf`, `not`, `if`/`then`/`else`, `patternProperties` or `format`. A top-level `type` other than `object` is also a breach. These are the constructs whose support differs between providers. Keeping them out makes one schema serve every provider. [Which constructs each provider supports is **unverified** here. The subset is a deliberate restriction, not a measured requirement.]

  8. *(Added 2026-10-08, while writing the PLAN.)* **The schema a model sees is published. The structural schema is the one validated. The tool's code enforces the bounds.**
     - **The problem.** `tool.New` validates arguments against the same schema it publishes (V33). Publishing D2's constraints unchanged would therefore make the library's validator refuse calls that gobble words better, or serves, today. Probe C shows what the validator would say:
       - `type: <invalid reflect.Value> has type "null", want "array"`, where edit says `edit: edits needs at least one replacement` (V37);
       - `minimum: 0/1 is less than 1.000000`;
       - and a refusal of `limit: 5000`, where read now serves 2000 lines (V34). bash likewise cuts a larger `timeout` to its maximum (V35).
     - **The rule.** `tool.New` derives the schema twice from the Go type:
       - one is validated exactly as today: types, `required`, `additionalProperties`;
       - the other gets the non-null rewrite and the `WithSchema` constraints, and is published as `Spec.InputSchema`.
     - **Each published constraint is enforced by the tool's own code,** in gobble's wording, as it is today:
       - read and bash serve at most their maximum;
       - edit refuses an empty list or an empty `oldText` with its own messages (V37, V38);
       - an argument the published schema calls invalid but the tool can serve, such as `limit: 0` meaning the default, is served.
     - **A test holds the two together.** For every bound a tool publishes, a call just past it is either served within the bound or refused with the tool's own message. It is never passed through unchecked.
     - **A maximum that depends on options comes from those options.** bash's comes from `Options.MaxTimeout`, so the schema tells the truth when an operator raises it.

  F3's defects are fixed by this test: edit's `edits` gains `"type":"array"` and `minItems: 1`; read's `limit` and bash's `timeout` gain their bounds.
- ~~**D3. No strict flag in 1.0, and a probe that measures it.**~~ *(Deprecated 2026-10-08: Probe D measured it, and F17 contradicts the premise that normalisation is harmless. The owner chose the resolution on 2026-10-08. The new text follows this one.)*
  - ~~gobble does not ask any provider for strict mode in 1.0. The SDK has no way to (V25).~~
  - ~~D2's convention keeps every schema eligible for strict normalisation where a provider does that (Doc 5).~~
  - ~~**A live-tagged test** sends each 1.0 native tool to an OpenAI Responses model through gobble's provider adapter, and records the `strict` value the reply's tool reports. It records, and fails only on a transport error.~~
  - ~~The first run's results are written into this record as an amendment.~~
  - ~~A per-provider schema adapter, as opencode has (0011 S10), is built only if that probe, or a provider's error, shows a schema gobble writes failing on a provider gobble supports. It then gets its own decision here.~~
- **D3 (amended 2026-10-08, the owner's choice). gobble opts out of strict mode for tools that have optional arguments, through the SDK.**
  - **The fix.** A tool whose published schema has a property outside `required` is sent with `strict: false`, OpenAI's documented opt-out (Doc 5: "To opt out of strict mode in Responses and keep non-strict, best-effort function calling, explicitly set `strict: false`"). A model can then leave optional arguments out, as gobble intends.
  - **Tools with only required properties keep strict mode,** and with it OpenAI's guarantee that arguments match the schema. Today these are write and edit; read, bash and powershell are opted out.
  - **Rejected for now:** the adapter in `llm/provider`, which publishes OpenAI's nullable form and drops `null` arguments. Probe D showed it only partly works: `gpt-4.1-mini` still invented `limit: 50`.
  - **It needs the SDK.** v1.2.1 sends no `strict` key (V25), so gobble cannot opt out until go-llmprovider-sdk carries a per-tool strict flag. D13's item is therefore a blocker for this decision. Until a release has the flag, behaviour stays as Probe D measured it: calls work, and models fill in optional arguments.
  - **When the flag ships,** `llm/provider` sets it from the published schema, at the point where it builds each `llmprovider.Tool` (0011 S3). gobble's go-llmprovider-sdk requirement moves to the release with the flag, in the same commit.
  - **The live-tagged probe stays,** as 0012-PLAN P5 builds it. Once the opt-out is in, it asserts what it now records: `strict=false` for read, bash and powershell, and `strict=true` for write and edit.
  - D2's convention is unchanged. Strict-eligible schemas still matter for the tools that keep strict mode, and for any provider that applies its own.
- **D4. JSON-RPC 2.0 is the transport, and the native tools are served over it in 1.x.**
  - ACP and MCP stay gobble's JSON-RPC boundaries (F1, F16).
  - When 0005-PLAN X4 builds `gobble mcp serve` (V19), it also exposes the native file tools as MCP tools, with the same schemas as `inputSchema` (0011 S9) and the same confinement and permission rules. Any MCP host then gets gobble's file operations.
  - This adds to 0005-MADR's `gobble mcp serve` row. It is 1.x, not part of the 1.0 gate.

**Descriptions**

- **D5. One description shape, in embedded Markdown templates, under a budget.**
  1. Each native tool's description is a file, `tool/builtin/describe/<name>.md`, embedded with `go:embed`. It is rendered once at construction with `text/template`, from the tool's options: OS, shell, limits. This replaces the constants and the `fmt.Sprintf` (V7, V8).
  2. **The first paragraph is one sentence that stands alone,** at most 160 bytes. It says what the tool does, and is what `tool_search` shows (D9) and what a catalog shows.
  3. **The sections that may follow,** each optional, in this order and no others:
     - `## Use when`: when to pick this tool over another, or over the shell;
     - `## Rules`: what the tool refuses or requires, with the limits;
     - `## Example`: one call, only where the arguments are not obvious.

     Headings are level 2; the body is plain paragraphs and `-` lists. No tables, HTML, or emphasis inside headings.
  4. **The budget.** A rendered description is at most 1,200 bytes. A whole definition (description plus schema, Probe A's measure) is at most 2,000 bytes. The largest today are 461 and 1,194 (Probe A).
     - These numbers are ceilings chosen to leave room for the sections; they are not measured requirements.
     - A test renders every template, powershell's included, on every host, and fails over budget.
  5. A parameter's details go in its schema description, not the body (F6). The test fails when a property name appears as a heading.
- **D6. Guidance that spans tools is written once, in the system prompt's `tools` section.**
  - The section is F3's (V18). It holds:
    - prefer the native file tools to shell commands for moving, deleting, copying and creating directories;
    - read before edit;
    - a delete always asks;
    - **the categories of deferred tools `tool_search` can find,** as Anthropic advises (Doc 2), generated from the deferred families of the session (D8).
  - It does not tell the model to search before every file operation. The file tools are direct (D7), so there is nothing to search for.

**Loading**

- **D7. Two exposure tiers.** This amends 0005-MADR's native-tools Exposure paragraph (V15).
  - **Direct, on every request, in this fixed order:**
    1. read, write, edit, or `apply_patch` in place of write and edit for GPT-family models (V22);
    2. grep, find, `tree`;
    3. move, delete, copy, mkdir;
    4. bash, and powershell on Windows;
    5. `todo`;
    6. `tool_search`, whenever anything is deferred (V14).

    That is 14 names on Windows for most models, and 13 for GPT-family models. Elsewhere, without powershell, it is 13 and 12.
  - **When X2 builds them,** the job tools `job_output`, `job_input` and `job_kill` join the direct tier, as 0005-MADR has them. Any command that outlasts the yield window needs them at once, which is common in build and test loops. X2's own MADR may revise this with a measurement.
  - **Deferred, found by `tool_search` or loaded under D9:**
    - `request_permissions`, which is moved from 0005-MADR's direct list: most tasks never call it;
    - `lsp`, `monitor` and `notebook`, as 0005-MADR has them;
    - the three MCP resource tools (V23);
    - **every MCP server's tools, unless mcp.json sets `exposure` to `direct`.** A server with no `exposure` key is deferred, which extends 0005-MADR's rule for bridged Pi servers (V24) to every server. 0005-PLAN F8 applies it (V5); until then MCP tools stay direct (V4).
  - `hidden` keeps its 0005-MADR meaning (V13): never offered, never found.
- **D8. Deferred tools load as families.**
  - Each deferred native tool names a family:
    - `permissions`: `request_permissions`;
    - `code`: `lsp`;
    - `notebooks`: `notebook`;
    - `processes`: `monitor`;
    - `mcp-resources`: the three resource tools.
  - Loading any member loads its family.
  - An MCP server's tools are not a family: a server can have many tools, so they load one by one, as each is found.
- **D9. When a deferred tool loads, and what a load does.**
  1. **At session start, free.** Before the first request, gobble adds what the session's state already calls for:
     - the `mcp-resources` family, when a connected server offers resources;
     - every family a setting pins to load at start;
     - every tool the transcript records as loaded, on resume or fork (V12).

     These are in the first prefix, and cost no cache break.
  2. **By `tool_search`.** It ranks deferred tools with D11's BM25. It returns at most `limit` results, each with the tool's name and its description's first sentence, and loads them with their families. Its text says what was loaded and that the tools are callable from the next step.
  3. **On a named event.** A call refused because the session lacks a permission loads the `permissions` family, and the refusal says `request_permissions` is now available. No other event loads tools in 1.0. Each new event is added by amending this decision.
  4. **Loads are append-only.**
     - A loaded tool is appended after the tools already sent, in load order, and is never removed or reordered within the session.
     - The load is recorded in the transcript (V12). F3's frozen baseline treats the tools list as part of the baseline, which grows only by appending (V16).
     - Until the SDK carries native deferral (D13), each mid-session load breaks the prompt cache once, from that request on (F10). Never removing a tool means it never breaks it again for that tool.
- **D10. Plain shell file commands are caught in the harness.** This lands in F6, with its `mvdan.cc/sh` parsing (V17, V27).
  - **When it applies.** A bash command is intercepted when all of these hold:
    - it parses to exactly one simple command (`CallExpr`): no pipe, list, redirect, substitution, subshell, or glob or variable left to expand;
    - its program is `rm`, `rmdir`, `mv`, `cp` or `mkdir`;
    - each of its flags maps exactly onto the native tool:
      - `rm` and `rm -r`/`-R` → delete, with `recursive`;
      - `rmdir` → delete;
      - `mv` → move;
      - `mv -f` → move, with `overwrite`;
      - `cp`, and `cp -r`/`-R` → copy;
      - `cp -f` → copy, with `overwrite`;
      - `mkdir` and `mkdir -p` → mkdir.
  - **What happens.** gobble runs the native tool instead of the shell, as codex does for patches (0011 S11). The call gets the tool's confinement, permission rules, kind and journal entry, and its result begins with a line saying which native tool ran.
  - **What it does not catch.** Any other flag (for example `rm -f`, whose silence on a missing file `delete` does not reproduce), several operands for `rm` or `mkdir`, or a compound command. These are not intercepted. They go through F6's per-sub-command permission rules (0005-MADR choice 5), so they fail closed.
  - **PowerShell has no Go parser** (0005-MADR choice 5). Its file cmdlets are not intercepted, and stay one raw permission pattern.

**Search**

- **D11. `tool_search` ranks with an in-house BM25, with no dependency.**
  - ~~It is a new internal package, `internal/bm25`. Its tier and imports follow 0004-MADR's package map, which the PLAN amends with the package's row.~~ *(Corrected 2026-10-08, while writing the PLAN: 0004-MADR's package map already has this package, `tool/toolsearch/`, beta, "BM25 index over tool specs (deferred exposure)" (V31). The package exists as a placeholder with no code (V32).)* The index is `tool/toolsearch`, as 0004-MADR maps it. The package map is unchanged.
  - **The index.** It is built per session over the tools that session can find.
  - **What it indexes, with field weights:**
    - the name, split on `_`, weight 3;
    - the first sentence, weight 2;
    - the rest of the description, weight 1;
    - the schema's property names and descriptions, weight 1 (V12: "names, descriptions, and schema properties").
  - **Tokenizing:** lower-cased words on Unicode letter and digit boundaries, with an English stop-word list.
  - **Parameters:** k1 1.2 and b 0.75, BM25's common defaults, as constants with a test.
  - **Exact names first.** A query that names a tool exactly ranks that tool first.
- **D12. bleve is not adopted for `tool_search`.** It stays the named candidate for a feature that needs a persistent full-text index over a large corpus: X1's session search and project-memory recall, and X8's documents. That feature's MADR names it, as AGENTS.md requires, and weighs Probe B's cost (F14).
- **D13. A request to go-llmprovider-sdk, recorded here and not made in that repository.** gobble needs:
  - native deferral: Anthropic's `defer_loading` with tool references, and OpenAI's `tool_search` with `defer_loading`;
  - cache control: Anthropic `cache_control`;
  - a per-tool strict flag. *(2026-10-08: this item is now a blocker for D3's opt-out, by the owner's choice. The other two remain requests.)*

  When the SDK has native deferral, gobble's deferred tools use it, and D9's mid-session cache break goes away on those providers (Doc 2, Doc 4). Each adoption is an amendment here.

**Results**

- **D14. No `outputSchema` for native tools in 1.0.**
  - `Result.Output` stays the model's text, and `Title`, `Summary` and `Diffs` stay the client's (F15). F1d's ACP `locations` join them as client data.
  - When D4 serves the native tools over MCP, each gains an `outputSchema` derived from a Go output type, as D1 derives inputs.

**The schemas.** Each row is the contract D2's test checks for that tool.

- "req" marks a required property. Every other property is optional, with its `default` where it has one.
- Integers carry their bounds as `minimum`/`maximum`. Strings that must not be empty carry `minLength: 1`.
- Paths resolve against the working directory and are confined (0005-PLAN F1a).

| Tool | Tier (D7) | Parameters |
| :--- | :--- | :--- |
| `read` | direct | `path` req; `offset` integer, at least 1; `limit` integer 1–2000 |
| `write` | direct | `path` req; `content` req |
| `edit` | direct | `path` req; `edits` req: array, `minItems` 1, of `{oldText` req `minLength` 1, `newText` req`}` |
| `apply_patch` | direct, GPT-family | `patch` req, `minLength` 1: the envelope (0010-REPORT §1) |
| `grep` | direct | `pattern` req; `path`; `glob`; `ignoreCase` boolean, default false; `literal` boolean, default false; `context` integer 0–10, default 0; `limit` integer 1–1000, default 100 |
| `find` | direct | `pattern` req (a glob); `path`; `limit` integer 1–5000, default 1000 |
| `tree` | direct | `path`; `depth` integer 1–10, default 2; `limit` integer 1–2000 entries, default 500 |
| `move` | direct | `from` req; `to` req; `overwrite` boolean, default false |
| `delete` | direct | `path` req; `recursive` boolean, default false |
| `copy` | direct | `from` req; `to` req; `overwrite` boolean, default false |
| `mkdir` | direct | `path` req |
| `bash`, `powershell` | direct | `command` req, `minLength` 1; `timeout` integer 1–600 seconds, default 120; `workdir` |
| `todo` | direct | `todos` req: array of `{content` req `minLength` 1, `status` req enum `pending`, `in_progress`, `completed`; `priority` enum `high`, `medium`, `low`, default `medium``}` (V21). An empty array clears the list. |
| `tool_search` | direct when anything is deferred | `query` req, `minLength` 1; `limit` integer 1–20, default 5 |
| `request_permissions` | deferred, `permissions` | `permissions` req: `{read` array of paths, `write` array of paths, `network` boolean`}`, `minProperties` 1; `reason` req `minLength` 1; `scope` enum `turn`, `session`, default `turn` |
| `list_mcp_resources` | deferred, `mcp-resources` | `server`; `cursor` |
| `list_mcp_resource_templates` | deferred, `mcp-resources` | `server`; `cursor` |
| `read_mcp_resource` | deferred, `mcp-resources` | `server` req; `uri` req |

~~`minProperties` is outside V30's list, and its support in jsonschema-go v0.4.3 is **[unverified]**. The PLAN checks it, and if it is missing, `request_permissions` refuses an empty grant in its code, and the row loses `minProperties`.~~ *(Checked 2026-10-08, while writing the PLAN: jsonschema-go v0.4.3's `Schema` has `MinProperties` (V36), so the row stands. F6 shows by test that the validator refuses an empty grant.)*

The 1.x tools keep 0011-REPORT §5's drafts until their own MADRs settle them: the job tools, `monitor` (X2), `lsp` (X7) and `notebook` (X8). Each of those MADRs applies D1, D2 and D5, and states its tool's tier under D7.

### Consequences

**Good**

- Schemas say what the tools enforce, so a model that reads them sends fewer refused calls (F3).
- One convention, tested, keeps all tools eligible for strict normalisation without per-provider work, until a probe shows otherwise (D3).
- Descriptions get a predictable shape and a ceiling, and searches show a sentence written for the purpose (D5, D9).
- **The always-sent set stays small:** 14 on Windows, under Anthropic's "30–50" degradation range (Doc 2). Most file tasks take no search round trip.
- Moves, deletes and copies keep their confinement, permissions and undo whether the model calls the tool or types `rm`, `mv` or `cp` (D10).
- No new module for search (D11).
- A session that resumes rebuilds the same tools list in the same order (D9).

**Bad**

- **Until the SDK has a per-tool strict flag, OpenAI models keep filling in optional arguments** (F17, D3). Calls still work, but a small model may read 20 lines where gobble would show 2000, and needs more calls to finish.
- **Each mid-session load breaks the prompt cache once** until D13 is met (F10). Family loading, start-of-session loading and append-only lists limit how often. They do not remove it.
- **14 direct definitions are sent on every request.** At Probe A's mean of 855 bytes, that is about 12 KB, an estimate. The direct tier is larger than Anthropic's "fewer than 10" mark, which is that page's guidance for when to use tool search at all (Doc 2).
- **D10 catches only exact simple commands.** `rm -f x` and `rm -r a && make` reach the shell, under F6's permission rules rather than the native tool's.
- **Templated descriptions** add a file per tool and a rendering step. A template error is caught at construction and by the budget test, not at compile time.
- **The in-house BM25 is code gobble maintains,** where bleve would be maintained upstream.

**Changes to other records,** made when this record is accepted:

- **0005-MADR** gains an amendment pointing here. It covers:
  - the Exposure paragraph of 2026-10-08 (V15): `request_permissions` deferred; `todo` and `tool_search` named direct; the counts restated as D7's;
  - the mcp.json default exposure (D7);
  - the `gobble mcp serve` row (D4).
- **0005-PLAN** gains notes:
  - at F1c, D2 and D5 for the new tools;
  - at F1d, D7–D9, D11, and the three resource tools' schemas;
  - at F6, D10;
  - at F8, D7's MCP default;
  - at X4, D4 and D14.
- ~~**0004-MADR's package map** gains `internal/bm25` (D11).~~ *(Corrected 2026-10-08: the map already has `tool/toolsearch`, so 0004-MADR is unchanged.)*

### Confirmation

Each new test is seen to fail first, on a scratch copy, before it is trusted, as every new check in this repository is. The handoff records the failure.

```text
# D2: schema conventions. Fails today on edit.edits ["null","array"] with no minItems,
# and on read.limit and bash.timeout without maximum (Probe A).
go test ./tool/builtin -run TestSchemaConvention -count=1          -> ok

# D2 rule 8: a call just past each published bound is served within it, or
# refused in the tool's own words; the validated schema is today's
go test ./tool ./tool/builtin -run 'TestPublished|TestBoundsHold' -count=1   -> ok

# D5: description shape and budget; the test renders every template,
# powershell's included, so one host checks them all
go test ./tool/builtin -run TestDescriptionShape -count=1          -> ok

# D7–D9: tiers, families, start-of-session loads, append-only and recorded loads, resume
go test ./agent ./acpserver -run 'TestExposure|TestToolLoad' -count=1   -> ok

# D10: interception of exact simple commands, and pass-through of everything else
go test ./tool/builtin -run TestShellIntercept -count=1            -> ok (lands with F6)

# D11: BM25 ranking, exact-name first, weights, k1 and b (lands with F1d)
go test ./tool/toolsearch -count=1                                 -> ok

# D12: no bleve module
go list -m all > "$LOG"; grep -c blevesearch "$LOG"                -> 0

# D3: the strict probe (live; the key from the environment, never the tree)
go test -tags live_openai ./llm/provider -run TestStrictProbe -count=1 -v
    -> before the SDK's strict flag: one "strict=" line per tool (all true, as Probe D)
    -> after it: strict=false for read, bash, powershell; strict=true for write, edit

# every gate
make preflight                                                     -> exit 0, Windows and WSL
```

## Pros and Cons of the Options

### I-A — JSON Schema derived from Go types, one convention (chosen)

- Good: it is the only form every provider accepts (0011 S5–S7), and the one MCP uses (0011 S9).
- Good: it cannot drift from the decoder, because both come from one Go type (F2).
- Good: one convention, tested, keeps the tools portable across providers and eligible for OpenAI's strict normalisation (Doc 5).
- Bad: jsonschema-go's inference needs overrides for slices and bounds (V29), so each tool carries a small schema hook.
- Bad: the portable subset forbids constructs some tools could use, such as `oneOf` for an operation-dependent `lsp` call. Those tools validate in code instead.

### I-B — JSON-RPC 2.0 method definitions as the schemas

- Good: one wire format at every boundary, editor and MCP alike (the strongest argument for it).
- Good: a method per tool would make the native tools callable from outside gobble. D4 keeps that benefit through MCP.
- Bad: JSON-RPC 2.0 has no schema language. A method's `params` are whatever both sides agree, so it cannot describe a parameter to a model (F1).
- Bad: a model never receives JSON-RPC. Its request `id` and method have no place in a provider's tool field (0011 S5–S7).

### I-C — hand-written JSON Schema files per tool

- Good: every construct is available, and the schema reads exactly as sent.
- Bad: two sources of truth, the file and the Go type, which drift with the first renamed field.
- Bad: nothing checks the file against the decoder, except a test that would amount to I-A.

### II-A — sectioned Markdown in embedded templates, under a budget (chosen)

- Good: all surveyed harnesses that write descriptions use Markdown (F5).
- Good: a fixed shape means a model finds "when to use" and "rules" in the same place in every tool. The first sentence is written for search results (D9).
- Good: templating keeps OS and limit values true per build (V8), and the budget test bounds the per-request cost (F6).
- Bad: a file per tool, and a template step that can fail at construction.
- Bad: the budget numbers are chosen, not measured. A tool that needs more must argue for it in an amendment.

### II-B — plain prose in Go constants, as now

- Good: no files, no templates. The largest today is 461 bytes (Probe A).
- Bad: no structure for the model to rely on, and no first sentence written for search.
- Bad: `fmt.Sprintf` constants grow unreadable as limits multiply (V8).

### II-C — free-form Markdown, no shape or budget

- Good: each author writes what the tool needs.
- Bad: every byte is sent on every request (F6), and nothing stops growth.
- Bad: a search result has no reliable summary line to show.

### III-A — two tiers, family and event loads, append-only, shell interception (chosen)

- Good: it matches both providers' advice and every surveyed harness (F8, F9). The tools most tasks need are always there, and the long tail stays out of the prompt.
- Good: no search round trip before ordinary file work.
- Good: loads happen at start where possible, and by family otherwise. Lists are append-only, so the cache breaks as rarely as the SDK allows (F10, F11).
- Good: the native tools' safety holds against `rm`/`mv`/`cp`, in the harness, not in a prompt (F12).
- Bad: 14 definitions on every request (about 12 KB at Probe A's mean, an estimate).
- Bad: a mid-session load still breaks the cache until D13.
- Bad: interception adds a mapping table that must stay exact. A wrong mapping would change a command's meaning, so it is kept small and tested.

### III-B — defer everything except `tool_search`, with a "consult tool_search first" instruction (the owner's proposal)

- Good (the strongest argument): the smallest first prompt, one discovery path for every tool, and accurate selection however many MCP tools a session has. Anthropic documents selection degrading past 30–50 tools (Doc 2).
- Bad: **every task begins with a search,** one more model round trip and its tokens before any work.
- Bad: **each newly loaded set breaks the prompt cache** on every provider gobble uses, because the SDK cannot defer natively (F10). Most sessions would break it at least once, early.
- Bad: **an instruction is not enforcement.** A model can skip the search. With bash deferred too, it can do nothing until it searches; with bash direct, it routes around the deferred file tools through `rm` and `mv` (F12).
- Bad: Anthropic refuses a request whose tools are all deferred, and advises keeping 3–5 frequent tools loaded; OpenAI advises keeping "functions needed for most tasks" loaded (Doc 2, Doc 4). None of the six surveyed harnesses defers its core tools (F8).

### III-C — every tool direct

- Good: no search, no loads, no cache breaks from loading. The simplest.
- Bad: every MCP server's tools join every request. Sessions with several servers reach Anthropic's 30–50 range, where selection degrades (Doc 2).
- Bad: 0005-MADR's `deferred` exposure and `tool_search` (V12, V13) would have no purpose.

### III-D — two tiers, without shell interception

- Good: everything of III-A except the mapping table, which is the riskiest new code.
- Bad: a model that types `rm -r build` bypasses delete's root refusal, its always-ask rule and its journal entry. F6's permission rules still see the command, but X2's undo and ACP's `delete` kind do not (0011-REPORT, the rule, points 2 and 3).

### IV-A — in-house BM25 (chosen)

- Good: the corpus is tens to hundreds of short texts in memory. Pi does it in about 110 lines (0011 M5), and codex and grok-build use BM25 too (0011 X12, K16).
- Good: no module, no binary growth, nothing more for `govulncheck`.
- Good: the weights fit the job: name and first sentence above the body.
- Bad: gobble maintains it, its tokenizer included. No stemming, so "deleting" and "delete" match only through the name split and the description's wording.

### IV-B — bleve, BM25 scoring

- Good (the strongest argument): mature, pure Go without the `vectors` tag (0011 M3), BM25 built in, with stemming, analyzers and on-disk indexes.
- Bad: about 13.6 MB of binary over hello-world, nearly doubling gobble's 14.4 MB, and 27 requirements against gobble's 24 (Probe B).
- Bad: BM25 is opt-in, with TF-IDF the default (0011 M1, M2). A mapping mistake would silently score differently.
- Bad: a persistent index engine, for a corpus that is rebuilt per session in memory.

### IV-C — substring and name matching

- Good: trivial, and predictable for exact names.
- Bad: no ranking, so a query in the model's words ("rename a folder") finds nothing unless it shares a substring. 0005-MADR already chose BM25 (V12).

## More Information

### Evidence index

**This record's claims,** checked by `q200_claims_0012.py` on 2026-10-08, 38 of 38 passing. V31–V38 were added while writing the PLAN:

| Id | Claim | Source |
| :--- | :--- | :--- |
| V1 | `tool.Exposure` declares `deferred` | gobble-cli `tool/tool.go:59` |
| V2 | the agent never reads a tool's exposure | gobble-cli `agent/` (3 files, no match) |
| V3 | the provider adapter sends every tool in the request | gobble-cli `llm/provider/provider.go:234` |
| V4 | MCP tools are declared direct today | gobble-cli `mcpclient/tool.go:43` |
| V5 | mcp.json exposure settings wait for 0005-PLAN F8 | gobble-cli `acpserver/mcp.go:150` |
| V6 | edit's `edits` is a Go slice | gobble-cli `tool/builtin/edit.go:18` |
| V7 | descriptions are Go string constants | gobble-cli `tool/builtin/bash.go:14` |
| V8 | read's description is formatted with its limits | gobble-cli `tool/builtin/read.go:24` |
| V9 | schemas come from jsonschema-go v0.4.3 | gobble-cli `go.mod:15` |
| V10 | a result carries client data (`Diffs`) | gobble-cli `tool/tool.go:92` |
| V11 | a result's `Output` is what the model sees | gobble-cli `tool/tool.go:88` |
| V12 | `tool_search` is BM25 over names, descriptions and schema properties; loads are recorded in the transcript | gobble-cli `docs/decisions/0005-MADR-v1-feature-scope.md:126` |
| V13 | exposure modes `direct`, `deferred`, `hidden` | 0005-MADR `:132` |
| V14 | `tool_search` is active when any tool is deferred | 0005-MADR `:533` |
| V15 | the native-tools amendment makes the job tools and `request_permissions` direct | 0005-MADR `:879` |
| V16 | choice 8: a frozen per-session baseline | 0005-MADR `:804` |
| V17 | choice 5: bash parsed with `mvdan.cc/sh` | 0005-MADR `:786` |
| V18 | the sectioned system prompt has a `tools` section | 0005-MADR `:156` |
| V19 | `gobble mcp serve` is 1.x | 0005-MADR `:193` |
| V20 | "The SDK sends no cache hints" | 0005-MADR `:576` |
| V21 | `todo` is 1.0 and publishes ACP `plan` updates | 0005-MADR `:127` |
| V22 | `apply_patch` replaces edit and write for GPT-family models | 0005-MADR `:775` |
| V23 | the three MCP resource tools are 1.0 | 0005-MADR `:131` |
| V24 | a bridged Pi server with no `exposure` key is deferred | 0005-MADR `:531` |
| V25 | the SDK sends no `strict` flag on any tool | go-llmprovider-sdk v1.2.1 `llmprovider/internal/wire/tools.go` (no match) |
| V26 | a recorded ChatGPT-backend reply reports `strict:true` for a tool sent without it | go-llmprovider-sdk v1.2.1 `llmprovider/providers/openai/testdata/chatgpt-tool.sse:2` |
| V27 | `mvdan.cc/sh` gives a simple command as a `CallExpr` | mvdan.cc/sh v3.13.1 `syntax/nodes.go:348` |
| V28 | Pi's `tool_search` ranks with BM25 | pi `packages/coding-agent/src/extensions/tool-search/tool.ts:2` |
| V29 | jsonschema-go infers a Go slice as `["null","array"]` | jsonschema-go v0.4.3 `jsonschema/infer.go:221` |
| V30 | jsonschema-go's `Schema` carries `maximum`, with `minimum`, `minItems`, `default` and `enum` | jsonschema-go v0.4.3 `jsonschema/schema.go:79` (and `:64`, `:74`, `:78`, `:90`) |
| V31 | 0004-MADR maps `tool/toolsearch/`, beta, as the BM25 index over tool specs | 0004-MADR `:228` |
| V32 | `tool/toolsearch` exists, declared only, with no index | gobble-cli `tool/toolsearch/doc.go:4` |
| V33 | `tool.New`'s `Run` validates arguments against the schema it publishes | gobble-cli `tool/tool.go:245` |
| V34 | read cuts a larger `limit` to `maxLines` | gobble-cli `tool/builtin/read.go:118` |
| V35 | bash cuts a larger `timeout` to its maximum | gobble-cli `tool/builtin/shell.go:115` |
| V36 | jsonschema-go's `Schema` carries `minProperties` | jsonschema-go v0.4.3 `jsonschema/schema.go:100` |
| V37 | edit refuses an empty `edits` in its own words | gobble-cli `tool/builtin/edit.go:43` |
| V38 | edit refuses an empty `oldText` in its own words | gobble-cli `tool/builtin/editmatch.go:110` |

**0011-REPORT's claims cited here.** Their file, line and matched text are in that report's evidence table, re-run on 2026-10-08 and passing:

| Id | Claim |
| :--- | :--- |
| 0011 S1, S2 | gobble's schema is `inputSchema`, derived by `jsonschema.For[In]` |
| 0011 S3, S4 | gobble hands the SDK name, description and schema; that is all the SDK's `Tool` holds |
| 0011 S5–S7 | the SDK sends `input_schema` (Anthropic) and `parameters` (OpenAI Responses, Chat Completions) |
| 0011 S8 | ACP frames are JSON-RPC 2.0 |
| 0011 S9 | an MCP tool's schema is JSON Schema `inputSchema` |
| 0011 S10 | opencode rewrites schemas per model |
| 0011 S11 | codex intercepts `apply_patch` run through its shell |
| 0011 D1–D4 | Pi, opencode and codex write descriptions in Markdown; Pi puts guidelines in the system prompt |
| 0011 T1–T5 | codex, Pi and grok-build defer only third-party or extension tools |
| 0011 T6 | go-llmprovider-sdk v1.2.1 has no `defer_loading`, `tool_reference` or `cache_control` |
| 0011 M1–M5 | bleve's BM25 is opt-in with TF-IDF default; faiss needs `vectors`; cobra is CLI-only; Pi's BM25 is in-house |
| 0011 X12, K16 | codex and grok-build rank with BM25 |

**Measurements and documents:**

| Claim | Source |
| :--- | :--- |
| definition sizes; `edits` nullable with no `minItems`; no bounds; no exposure set | Probe A, `specsize`, 2026-10-08, this record's table |
| bleve's binary, requirement and package counts | Probe B, `bleveprobe`, 0011-REPORT §4 |
| jsonschema-go's messages for `minItems`, `null`, `minLength`, `minimum` and `maximum` | Probe C, `valmsg`, 2026-10-08, this record |
| OpenAI normalises every gobble tool to strict by making every property required; models then invent optional values; the nullable form gives `null`s that gobble refuses | Probe D, `strictprobe`, 2026-10-08, this record (one sample per model, variant and call) |
| the SDK's OpenAI provider posts to `/responses` | go-llmprovider-sdk v1.2.1 `llmprovider/providers/openai/openai.go:264` |
| tools lead the cache prefix; a tool change invalidates the cache | Doc 1 |
| keep 3–5 tools loaded; fewer than 10; degrades past 30–50; not all deferred; prompt section of categories; caching preserved | Doc 2 |
| tool changes are prefix changes | Doc 3 |
| `gpt-5.4` and later; cache preserved; changing the loaded set breaks it; keep "functions needed for most tasks" | Doc 4 |
| Responses normalises to strict when possible, else falls back and reports `strict: false`; strict's requirements | Doc 5 |

### Related records

- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md): the tool line this record describes. It is amended here once this record is accepted.
- [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F1c, F1d, F6, F8 and X4 carry these decisions.
- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): the package map, whose `tool/toolsearch` row D11 builds.
- [0011-REPORT-native-tools-survey.md](../reports/0011-REPORT-native-tools-survey.md): the evaluation this record decides from.
- [0010-REPORT-harness-design-survey.md](../reports/0010-REPORT-harness-design-survey.md): the harness survey, including `apply_patch`'s envelope.
- Doc 1: `https://platform.claude.com/docs/en/docs/build-with-claude/prompt-caching`
- Doc 2: `https://platform.claude.com/docs/en/agents-and-tools/tool-use/tool-search-tool`
- Doc 3: `https://developers.openai.com/api/docs/guides/prompt-caching`
- Doc 4: `https://developers.openai.com/api/docs/guides/tools-tool-search`
- Doc 5: `https://developers.openai.com/api/docs/guides/function-calling`

### Open questions for the plan

The PLAN resolves these. None leaves a decision above open. *(2026-10-08: [0012-PLAN](0012-PLAN-tool-schemas-descriptions-and-loading.md) resolves them all:*

- *1: one 0012-PLAN for D1–D3 and D5 over the five built-in tools, plus notes in 0005-PLAN for the rest;*
- *2: by V36;*
- *3: a `WithSchema(func(*jsonschema.Schema))` option, applied to a second, published schema (D2 rule 8);*
- *4: ~~the `openai` provider's default model~~ `gpt-4.1-mini`, since the provider has no default (Probe D), with the key read from `OPENAI_API_KEY` by `llmtest.LiveCredential`.)*

1. **The record shape.** The work is either one 0012-PLAN, with phases sequenced inside 0005-PLAN's F1c, F1d, F6 and F8, or steps added to those 0005-PLAN sub-phases with no 0012-PLAN.
   - The recommendation is a 0012-PLAN for the cross-cutting work: D1, D2 and D5 over the existing five tools, and D11's package.
   - Plus notes in 0005-PLAN for the rest, which lands where those sub-phases already build the tools.
2. ~~**jsonschema-go v0.4.3 and `minProperties`.** Whether the library carries it (the table's note). This is checked by reading `jsonschema/schema.go` before the `request_permissions` row is built.~~ *(Resolved 2026-10-08: it does, V36.)*
3. **The schema hook's shape on `tool.New`.** Either a `WithSchema(func(*jsonschema.Schema))` option, or `ForOptions.TypeSchemas` per input type. The PLAN picks one after reading `tool/tool.go`'s construction path, under D1's rule that constraints are set in Go.
4. **The probe's model and credentials.** D3's live test reads its key from the environment, outside the tree, and is skipped without one. The PLAN names the model, among those Doc 4 and Doc 5 apply to.
