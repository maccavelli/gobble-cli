---
status: accepted
date: 2026-11-17
decision-makers: repository owner
consulted: "the owner's report of 2026-11-16 and rulings of 2026-11-16 and 2026-11-17, relayed by the cross-repository coordinator; gobble-cli at 2004980; mcp-server-magictools at 368ac6a; the read-only checkouts of codex at 57e64682f8, opencode at 4ce137b39e, kilocode at 58072bd696 and grok-build at 2bdd1d6a63; go-llmprovider-sdk v1.31.0 in the module cache, and its draft record on a typed tool-schema refusal (proposed, not placed); xAI's function-calling guide and Claude Code's public changelog, fetched 2026-11-17 (documentation); the drafting experiments under More Information, run outside the working tree"
informed: "go-llmprovider-sdk, whose draft record on a typed refusal names gobble as its first caller; mcp-server-magictools, the owner's server, which this record does not change; the cross-repository coordinator"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Give each branch of an MCP tool's root-level union the root's object schema at load, only where that keeps the inputs the schema accepts

## Context and Problem Statement

On 2026-11-16 the owner ran gobble in a terminal with the owner's `magictools` MCP server added. Under the model
`grok-4.6` every prompt failed (the owner's paste, P1):

```text
llm: api status 400: … grok HTTP 400 invalid-argument: Failed to start sampling: [invalid_client_tool_schema] mcp__magictools__execute_pipeline: tool parameter root must be an object type (root schema is an anyOf/oneOf union with a non-object branch)
```

Other models worked. This record decides what gobble does with an MCP tool whose input schema a provider refuses,
starting with root-level unions.

### The schema, as the server emits it

`execute_pipeline` (M1) is declared in `mcp-server-magictools`' static inventory, `internal/handler/inventory.go:286-336`
at `368ac6a`. With its seven property descriptions shortened to `…`:

```json
{
  "type": "object",
  "properties": {
    "session_id": {"type": "string", "description": "…"},
    "target": {"type": "string", "description": "…"},
    "intent": {"type": "string", "description": "…"},
    "plan_hash": {"type": "string", "description": "…"},
    "dry_run": {"type": "boolean", "description": "…"},
    "target_roles": {"type": "array", "items": {"type": "string"}, "description": "…"},
    "reject": {"type": "boolean", "description": "…"}
  },
  "additionalProperties": false,
  "anyOf": [
    {"required": ["target", "intent"]},
    {"required": ["session_id"]}
  ]
}
```

* **Shape class.** The root is itself an object schema, `"type": "object"` with `properties` and
  `additionalProperties: false` (M2, M3), and the `anyOf` sits beside them (M4). The root is not "nothing but the
  union". Each branch is a `required`-only fragment: no `type`, no `properties`. It is valid JSON Schema.
* **It reaches the wire as written.** The server registers the inventory's schema (M6) after a sanitiser that
  recurses into `properties` only, never into `anyOf` (M7), so the branches stay untyped. Of the inventory's 18 tools
  this is the only one with a root union (run R4).
* **The server validates nothing against it.** The go-sdk's low-level `Server.AddTool`, which the server calls, leaves
  validation to the caller (M9). The handler's own check refuses any call that lacks one of `target`, `intent` and
  `session_id` (M8), so it is stricter than the schema, and it refuses the "continuation mode" the `session_id`
  description advertises, `session_id` alone (M5). That is a defect of the server, independent of this record (Q5).

So "add `"type": "object"` to each branch" applies: the root is an object, and each branch is a fragment that only
narrows it.

### gobble today (2004980)

* **Discovery and load.** A server's tools are listed once, when it connects (G5), and each becomes a `tool.Tool` in
  `newTool` (G1), named `mcp__<server>__<tool>`, cut and hashed past 64 characters (G17), and declared direct (G4).
* **The one schema transform.** `inputSchema` adds `"type": "object"` and `"properties": {}` when either is missing,
  and makes a schema that is not a JSON object an empty object (G2, G3). It writes those keys into the go-sdk's own map
  (G2), and touches nothing else: no union handling, no size limit, no deferral. The result is `Spec.InputSchema` for
  every provider (G10); `tool_search` indexes its root `properties` (G16). gobble does not validate the model's
  arguments against an MCP schema; it only refuses arguments that are not a JSON object (G6b).
* **Every prompt sends every MCP tool.** The MCP tools are gathered for each prompt (G6), all direct, so each model
  call sends them (G7). A refused request is a stream error that ends the call (G8) and fails the prompt (G9). The next
  prompt gathers the same tools. One refused schema therefore disables the session on that model, for every prompt,
  until the user acts: the owner's report.
* **What the user can do today.** Switch to another model (`/model`); start with `--exclude-tools
  mcp__magictools__execute_pipeline` (G13); add a permission rule that denies the tool whole, which leaves it out of
  the request (G14); or remove the server (`gobble mcp remove`). Nothing tells the user which of these applies: the
  error is the provider's text.
* **The "signature" text match misfires, measured.** gobble resends a 400 once, with thinking stripped, when the
  error's text holds `signature` (G11, G12). A fake Grok endpoint that refuses a tool named
  `mcp__vault__verify_signature` the way P1 refuses `execute_pipeline` received two requests; for
  `mcp__magictools__execute_pipeline` it received one (run R3). The harm is one wasted request; the turn fails either
  way. The SDK draft inferred this; R3 measures it.
* **Docs.** `docs/guides/` has no MCP guide (four guides: developing, models-json, choosing-a-model, credentials). The
  `mcpclient` row of `docs/architecture.md` (G20) and `mcpclient/doc.go` say nothing about schemas. gobble keeps no
  changelog file (`git ls-files` at `2004980`).
* **Tests.** `acpserver/mcp_test.go:143` runs a session's prompt against stdio and HTTP MCP fixtures (G21); the fixture
  server registers its tools through the go-sdk's typed `mcp.AddTool`, so none has a union. No test covers
  `inputSchema`.

### Is the rewrite input-preserving? The argument

Write the root as `R ∧ U`, where `R` is the root without its union and `U` is `anyOf [B1 … Bn]` (or `oneOf`). An
instance is valid against the root exactly when it is valid against `R` and against `U`.

1. If every instance valid against `R` is valid against `C`, then for those instances `Bi` and `Bi ∧ C` give the same
   verdict. Replacing each `Bi` by `Bi ∧ C` leaves `R ∧ U` unchanged. For `oneOf`, the count of matching branches is
   unchanged instance by instance, so "exactly one" is too.
2. With a root `"type": "object"`, `C = {"type": "object"}` qualifies. So does `C =` the root's `properties`,
   `patternProperties` and `additionalProperties` taken together, since they act only on each other within one schema
   object, and the root already imposes them.
3. Adding `C`'s keywords to a branch equals `Bi ∧ C` only when the branch has none of them. A branch with its own
   `properties` keeps them; overwriting them would change the branch.
4. Under an object-only root, a branch whose type excludes object, or `false`, matches no accepted instance. Removing it
   changes neither `anyOf` nor `oneOf`. A branch `true` is `{"type": "object"}` there, and a branch type list that
   includes `object` narrows to `object`.
5. The argument needs the root's type to be exactly `object`. With no root type, or `["object", "null"]`, the root
   accepts non-objects and typing a branch rejects them. gobble already adds a missing root type (G2), and MCP
   arguments are always an object (G6b), so a missing root type is not a loss in practice; a type list is left alone.
6. Keywords that collect annotations across branches (`unevaluatedProperties`, `unevaluatedItems`) and dynamic
   references are outside the argument; the rule does not rewrite a root that carries them.
   *(Amended 2026-11-17, see "Amendment (2026-11-17)" below: this spoke of the root only; the review found these
   keywords, and ids, anchors and `$ref` at any depth, inside the union or the copied properties, change verdicts.)*

### Is the rewrite input-preserving? Measured

A stdlib-only validator of this record's own (`preserve.py`), cross-checked against `github.com/google/jsonschema-go`
`v0.4.3`, the validator already in gobble's module graph (G19), in a `git archive 2004980` copy under `<scratch>`. Each
shape had one object for every subset of its property names plus one unknown name, each with a value of its declared
type (4 to 256 objects), 20,000 seeded random objects over the same names and a 15-value pool, and 9 non-objects (run
R1). Twelve shapes, S1 to S12: 240,484 instances, 2,084,284 verdicts; the real schema verbatim (descriptions
included): 20,265 instances, 182,385 verdicts. The two validators disagreed on 0 verdicts (run R2). Draft-07 `$ref`
semantics gave the same table.

| Shape | Instances the original accepts | Type each branch | Copy root's object keywords into keyword-free branches | The rule (D2) | Drop the union | kilocode's flatten |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| S0, the real schema, verbatim | 218 of 20,256 | 0 | 0 | 0 | +2,637 | +2,262 / −218 |
| S2, S1 with `oneOf` | 279 | 0 | 0 | 0 | +3,808 | not applied (`anyOf` only) |
| S3, no root type | 963 | 0 objects, 9 non-objects | 0 objects, 9 non-objects | refused | +5,868 | +3,913 / −954, 9 non-objects |
| S4, a `{"type":"string"}` branch | 1,211 | 0 | 0 | 0 (branch removed) | +7,261 | +4,996 / −1,211 |
| S5, a nested `anyOf` in a branch | 952 | 0 | 0 | 0 | +4,030 | +4,030 / −952 |
| S6, `$ref` branches | 2,049 | 0 | 0 | refused | +6,914 | +5,034 / −2,049 |
| S7, branches with their own `properties` | 425 | 0 | 0 (overwriting: +945) | 0 | +8,464 | +5,382 / −408 |
| S8, root `["object","null"]` | 2,262 | 0 objects, 1 non-object | 0 objects, 1 non-object | refused | +6,737 | +5,030 / −2,261, 1 non-object |
| S9, `true` and `false` branches | 12,662 | 0 | 0 | 0 | 0 | −6,055 |
| S10, every branch a non-object type | 0 | 0 | 0 | refused (unsatisfiable) | +12,680 | +6,591 |
| S11, `patternProperties`, `oneOf` | 790 | 0 | 0 | 0 | +5,051 | not applied |
| S12, a branch typed `["object","null"]` | 969 | 0 | 0 | 0 | +5,118 | +5,118 / −969 |

`+n` is objects newly accepted, `−n` objects newly refused. S1 (the real schema without descriptions) gives S0's row.

What it establishes:

* **Input-preserving** on an object-only root: typing untyped branches, typing `true`, removing `false` and
  non-object branches, narrowing a branch type list to `object`, and copying the root's object keywords into branches
  that have none of them. Nested unions inside a branch and `oneOf` keep the result. `$ref` branches measured
  preserving in both drafts' semantics, but the rule leaves them alone (D2).
* **Not input-preserving:** dropping the union (on the real schema, 2,637 more objects accepted, `{}` among them);
  kilocode's flatten (it refuses all 218 valid calls, since the branches carry no `properties`); overwriting a branch's
  own `properties` (S7, +945); typing branches under a root that is not object-only (S3, S8).
* **Seen to fail.** The harness, given a planted rule that drops the union, reported differences on 9 of the 10 shapes
  it applied to (S9's `true` branch accepts every object, so dropping its union changes nothing) and exited 1; the cross-check, with one Python verdict per schema flipped, reported 104 disagreements and exited 1.

This is input preservation under JSON Schema. What a provider's grammar compiler does with the result is not covered
by it (next section).

### What each provider and route refuses

Only Grok's answer to the owner's shape is known: P1 (`grok-4.6`), and xAI's guide (X1): "The root of a `parameters`
schema must be an object", "A root `anyOf` or `oneOf` also works when every branch is itself an object", and a tool
whose root is "neither an object nor a union of objects" "is rejected with a `400` error that names the tool". The
guide says such a union "cannot be compiled into a tool-call grammar", and its `oneOf` example gives each branch its own
`properties` and `required`. Whether a branch with `"type": "object"` and no `properties` passes is not established.

The go-llmprovider-sdk draft's table records each route's answer to a root `{"type":"string"}`, not to this union:
refused by Grok and OpenAI Responses, OpenCode Zen's Responses and Google routes, and Anthropic Messages; accepted by
OpenCode Go's Messages route, Gemini Interactions and five Chat Completions servers. Refusal varies by route, not by
model family. For every route but Grok's, the answer to the owner's shape, and to the rewritten shape, is not
established.

### What the other agents do

| Agent | What it does with a root union | Source | Kind |
| :--- | :--- | :--- | :--- |
| Grok's own CLI (`grok-build`) | Never sends an MCP tool's schema as function parameters. MCP tools are reached through two meta-tools, `search_tool`, whose result carries each tool's `input_schema` as text (K2), and `use_tool`, whose `tool_input` is `{"type": "object", "additionalProperties": true}` (K1, K3). Its own `use_tool` schema, when file input is on, is a root `oneOf` of three branches, each `"type": "object"` with its own `properties` and `required`, under the comment "Root unions compile each branch without inheriting the root properties." (K4) | `grok-build` at `2bdd1d6a63` | source |
| Claude Code | 2.1.300: on a 400 that rejects an MCP tool's input schema, "Claude Code now sends that tool without parameters for the rest of the conversation" (CC1). 2.1.323: `/mcp` shows "how many of a server's tools were left out because the API would reject their input schema, and why" (CC2). 2.1.326: "Fixed MCP tools with a root `anyOf` or `oneOf` showing Claude only the first branch's form of an argument" (CC3), so it rewrites root unions in some form; the form is not documented | public changelog, fetched 2026-11-17; a read-only checkout at `7779afb12e` (changelog to 2.1.283) has none of these entries | documentation; the rewrite itself not established |
| codex | Lowers every MCP schema: an untyped schema with `required`, `properties` or `additionalProperties` becomes `"type": "object"` with `properties: {}`, branches included (C1). That is "type each branch" plus an empty `properties`, preserving by R1's `codex_like` column | `codex` at `57e64682f8` | source |
| opencode | Gives a root with no `type` an object type; a root that has one is returned unchanged (O1). The owner's schema would go out as written (inferred) | `opencode` at `4ce137b39e` | source |
| kilocode | In `packages/llm`, flattens a root `anyOf` into one object whose `properties` are the branches' merged, with `additionalProperties: false` (L1): on the owner's schema it accepts no valid call (R1). Which runtime path sends through it is not established | `kilocode` at `58072bd696` | source |

The nearest thing to a specification of what Grok accepts is K4: Grok's own client sends a root union only with each
branch a typed object carrying its own `properties`. Read with X1, that says xAI compiles each branch alone (inferred):
a branch typed `object` with no `properties` may pass the root check yet leave its arguments unconstrained in that
branch's grammar. That is why the rule copies the root's object keywords into each branch rather than only typing it.

### The owner's rulings so far

Relayed by the coordinator, quoted as labels:

* The owner asked whether normalising root-level unions at load "would … be best served by implementing it in the
  go-llmprovider-sdk, or in gobble-cli itself"; the coordinator assessed: in gobble, at load.
* Where the normalisation is recorded: "A gobble record, drafted after P6 commits (Recommended)".
* The typed error: "In the SDK, with its next record (Recommended)". On the SDK's draft (2026-11-17): "A typed error
  kind plus the tool's name (Recommended)"; "Stable, in `llmprovider` (Recommended)"; "Every route with a recorded
  refusal (Recommended)"; "The new kind, with the tool left empty (Recommended)"; "No; revisit after the live
  typed-branch probe (Recommended)"; "\"llmprovider: tool schema refused\", the tool name not printed (Recommended)";
  "All four in the next release with 0080, on my word (Recommended)". The SDK never rewrites a schema.

### Evidence index

Marks: **read** (a line at the commit named), **run** (an experiment for this record, outside the tree), **fetched**
(public documentation, 2026-11-17, nothing about this machine or its owner sent), **the owner's**.

| Id | Claim | Location | Repository at commit | Mark |
| :--- | :--- | :--- | :--- | :--- |
| P1 | The failure | the owner's paste, 2026-11-16 | — | the owner's |
| G1 | Each MCP tool's spec takes `inputSchema(t.InputSchema)` | `mcpclient/tool.go:41` | gobble-cli `2004980` | read |
| G2 | `type` and `properties` added only when missing, in the go-sdk's map | `mcpclient/tool.go:71`, `:74` | gobble-cli `2004980` | read |
| G3 | A schema that is not a JSON object becomes an empty one | `mcpclient/tool.go:67` | gobble-cli `2004980` | read |
| G4 | Every MCP tool is direct | `mcpclient/tool.go:44` | gobble-cli `2004980` | read |
| G5 | Tools listed once at connect; a spec built per listed tool | `mcpclient/conn.go:145`; `mcpclient/manager.go:80` | gobble-cli `2004980` | read |
| G6 | The MCP tools are gathered for every prompt | `acpserver/agent.go:730` | gobble-cli `2004980` | read |
| G6b | Arguments that are not a JSON object are refused; nothing else is checked | `mcpclient/tool.go:100` | gobble-cli `2004980` | read |
| G7 | Each model call sends the turn's tools | `agent/agent.go:671` | gobble-cli `2004980` | read |
| G8 | A stream error ends the call | `agent/agent.go:691` | gobble-cli `2004980` | read |
| G9 | and fails the prompt | `acpserver/agent.go:796` | gobble-cli `2004980` | read |
| G10 | The schema goes to the SDK as the spec holds it | `llm/provider/provider.go:616` | gobble-cli `2004980` | read |
| G11 | One resend with thinking stripped on a matched refusal | `llm/provider/provider.go:344` | gobble-cli `2004980` | read |
| G12 | matched by substrings of the error's text | `llm/provider/replay.go:123`, `:128` | gobble-cli `2004980` | read |
| G13 | `--exclude-tools` | `internal/cli/flags.go:51` | gobble-cli `2004980` | read |
| G14 | A deny rule for the whole tool hides it from the request | `permission/rules.go:500`; `acpserver/rules.go:205` | gobble-cli `2004980` | read |
| G15 | MCP diagnostics: a warning, and a notice for a client that offered notices | `acpserver/mcp.go:168`; `acpserver/notice.go:33` | gobble-cli `2004980` | read |
| G16 | `tool_search` indexes a schema's root properties | `tool/toolsearch/index.go:145` | gobble-cli `2004980` | read |
| G17 | Tool names and the 64-character limit | `mcpclient/name.go:18`, `:10` | gobble-cli `2004980` | read |
| G18 | The live-test pattern for Grok | `llm/provider/live_grok_test.go:8` | gobble-cli `2004980` | read |
| G19 | jsonschema-go, go-llmprovider-sdk and the go-sdk pins | `go.mod:15`, `:17`, `:22` | gobble-cli `2004980` | read |
| G20 | The `mcpclient` row; the package doc | `docs/architecture.md:131`; `mcpclient/doc.go:11` | gobble-cli `2004980` | read |
| G21 | The MCP prompt test | `acpserver/mcp_test.go:143`; fixture `mcpclient/mcptest/mcptest.go:64` | gobble-cli `2004980` | read |
| M1 | The tool | `internal/handler/inventory.go:286` | mcp-server-magictools `368ac6a` | read |
| M2 | Root `"type": "object"` | `internal/handler/inventory.go:290` | mcp-server-magictools `368ac6a` | read |
| M3 | `additionalProperties: false` | `internal/handler/inventory.go:322` | mcp-server-magictools `368ac6a` | read |
| M4 | The root `anyOf` | `internal/handler/inventory.go:323` | mcp-server-magictools `368ac6a` | read |
| M5 | `session_id` advertises a continuation mode without `target`/`intent` | `internal/handler/inventory.go:294` | mcp-server-magictools `368ac6a` | read |
| M6 | The inventory's schema is sanitised, then registered | `internal/handler/handlers.go:219`, `:225` | mcp-server-magictools `368ac6a` | read |
| M7 | The sanitiser recurses into `properties` only | `internal/util/schema_util.go:52` | mcp-server-magictools `368ac6a` | read |
| M8 | The handler requires all three of `target`, `intent`, `session_id` | `internal/handler/execute_pipeline.go:50` | mcp-server-magictools `368ac6a` | read |
| M9 | Low-level `AddTool` leaves validation to the caller | `mcp/server.go:228` | `github.com/modelcontextprotocol/go-sdk` `v1.6.1` (the server's pin), module source | read |
| K1 | `use_tool`'s `tool_input` is an open object | `crates/codegen/xai-grok-tools/src/implementations/use_tool/input.rs:193` | grok-build `2bdd1d6a63` | read |
| K2 | `search_tool` returns each tool's `input_schema` as result text | `crates/codegen/xai-grok-tools/src/implementations/search_tool/mod.rs:294` | grok-build `2bdd1d6a63` | read |
| K3 | MCP tools are called through `use_tool` | `crates/codegen/xai-grok-pager/docs/user-guide/07-mcp-servers.md:240` | grok-build `2bdd1d6a63` | read |
| K4 | A root `oneOf` of typed-object branches with their own properties, and the comment | `crates/codegen/xai-grok-tools/src/implementations/use_tool/input.rs:203`, `:211` | grok-build `2bdd1d6a63` | read |
| C1 | codex types a `required`-bearing schema as an object, recursively | `codex-rs/tools/src/json_schema.rs:143` (in `sanitize_json_schema`, `:91`); applied to MCP tools from `codex-rs/tools/src/mcp_tool.rs:60` | codex `57e64682f8` | read |
| O1 | opencode returns a typed root unchanged | `packages/ai/src/protocols/utils/tool-schema.ts:73` | opencode `4ce137b39e` | read |
| L1 | kilocode's flatten | `packages/llm/src/protocols/utils/tool-schema.ts:48`, `:60` | kilocode `58072bd696` | read |
| SDK1 | The SDK classifies the refusal; xAI's recorded body shape | `llmprovider/api_error.go:247`; `llmprovider/internal/wire/testdata/refusals-0062/responses/grok-string-schema-tool.json` | go-llmprovider-sdk `v1.31.0`, module source | read |
| X1 | xAI's root rules | <https://docs.x.ai/docs/guides/function-calling> | — | fetched |
| CC1 | Claude Code 2.1.300 | <https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md>, line 2027 of the file as fetched | — | fetched |
| CC2 | Claude Code 2.1.323 | the same file, line 704 | — | fetched |
| CC3 | Claude Code 2.1.326 | the same file, line 86 | — | fetched |
| R1 | Input preservation, 12 shapes and the real schema | `preserve.py`, under `<scratch>` | — | run |
| R2 | The jsonschema-go cross-check | `check_test.go` and `compare.py`, in a `git archive 2004980` copy | — | run |
| R3 | The `signature` misfire, and the tools body gobble sends | a scratch test in that copy's `llm/provider`, a fake Grok endpoint | — | run |
| R4 | The inventory's 18 tools; the rewrite's sizes | `realschema.py`, under `<scratch>` | — | run |

## Decision Drivers

* **One refused schema must not disable a session.** Today it fails every prompt on that model (G6 to G9).
* **The model keeps the server's contract.** A rewrite gobble cannot show to accept exactly the same arguments is not
  made silently; that rules out dropping or flattening the union (R1).
* **What is unmeasured is not assumed.** No route but Grok has answered the union shape; no route at all has answered
  the rewritten one. The decision is testable offline and confirmed live by the owner.
* **The owner's rulings:** the normalisation is gobble's and recorded here; the SDK types the refusal and never
  rewrites a schema.
* **Little to maintain.** A provider or route table in gobble is what the SDK draft rejected as unsupportable (its
  option C); a rule that holds for every provider needs none.
* **The user is told what gobble did not fix,** once, with the remedy, through the channel MCP diagnostics already use
  (G15).

## Considered Options

* **A. Normalise at load, every provider, input-preserving rewrites only.** In `mcpclient`, where the schema is
  loaded: the rule of D2 on an object-only root; anything outside it is sent as given, and a shape a known route
  refuses is flagged to the user.
* **B. The same rewrite, only for Grok, at request time.** In `llm/provider`, for provider `grok`.
* **C. A lossy flatten or drop of the union, with a notice.**
* **D. No normalisation: detect the refusal, drop the tool for the session, resend once, tell the user.** By text
  today; by the SDK's `ErrToolSchemaRefused` and `APIError.Tool` once released and pinned.
* **E. Validate at load, and refuse to register a tool a known route would refuse,** naming server and tool.
* **F. Nothing in gobble; ask the MCP server to change its schema.**
* **G. Degrade a refused schema to an open object, the real schema in the description** (Grok's own CLI for every MCP
  tool, K1 to K3; Claude Code after a refusal, CC1).

Combinations considered: A with D as the fallback once the SDK's error exists (the recommendation, D through Q4); A with
G for what A cannot rewrite (Q2 (b)); A with E (Q2 (c)).

## Decision Outcome

Chosen option: **"A", with what it cannot rewrite sent as given and flagged once, and D as the later fallback by a
record of its own once the SDK's typed error is released and pinned**, because A removes the observed failure before
any request on every provider, keeps every argument the server's schema accepts (0 differences in 2,266,669 verdicts,
R1 and R2), takes the form xAI documents and its own client sends (X1, K4), and needs no provider table.

Accepted 2026-11-17: the owner answered Q1 to Q7 with the recommended option each (listed under "Decided", below), and
approved the plan through its P3, the owner's live step, with the label "Approve through my live step too, then ask
about `main`", relayed by the coordinator. The decision sections below state what was decided.

**Where this differs from the coordinator's earlier assessment.** It agrees on gobble, at load. It differs in three
places:

1. **The rewrite is not only "type each branch".** That passes the root check, but xAI compiles each branch alone (K4,
   inferred), so a typed branch with no `properties` may leave the arguments unconstrained. D2 also copies the root's
   object keywords into each branch that has none, which R1 shows is still input-preserving.
2. **"As Claude Code does" is only partly established.** Claude Code's changelog shows it rewrites root unions in an
   undocumented form (CC3), leaves out at load the tools the API would reject (CC2), and after a refusal sends the tool
   without parameters (CC1). This record does not rest on any of them.
3. **Lossiness.** The coordinator first said the rewrite was lossy, then corrected that by inference. R1 and R2 now
   measure it: input-preserving under the conditions of D2, and not outside them (S3, S7 overwritten, S8).

### D1. Where: `mcpclient`, at load, for every provider (Q1, decided)

The rule applies to every provider, when the tool is loaded. Should the owner's live step (PLAN P3 step 6) show another
provider refusing the rewritten form, Q1 comes back to the owner. The rule runs in `inputSchema` (G1), on a deep copy, after gobble's existing type and properties additions (G2). Its
result is the tool's `Spec.InputSchema` for the session: what every provider is sent (G10) and what `tool_search`
indexes (G16), whose root `properties` the rule leaves alone. The go-sdk's own map is no longer written to (G2 writes
into it today; the copy ends that).

### D2. The rule

It applies only when all hold:

* the root's `type` is exactly `"object"`;
* the root has exactly one of `anyOf` and `oneOf`;
* the root's other keywords are among `type`, `properties`, `required`, `additionalProperties`, `patternProperties`,
  and the annotations `description`, `title`, `examples`, `default`, `deprecated`, `$comment`, `$schema`;
* every branch is `true`, `false`, or an object schema with no `$ref`.
* *(Added by the amendment of 2026-11-17 below.)* None of `$ref`, `$dynamicRef`, `$recursiveRef`, `$id`, `$anchor`,
  `$dynamicAnchor`, `$recursiveAnchor`, `unevaluatedProperties` and `unevaluatedItems` appears as a keyword anywhere in
  the union (any depth of any branch) or anywhere in the root's `properties`, `patternProperties` and
  `additionalProperties`. A property called one of them is a name, not the keyword.

Then, in order:

1. remove each branch that is `false`, or whose `type` excludes `object`;
2. if no branch is left, stop: the schema is unsatisfiable and is sent as given (D3);
3. a branch `true` becomes `{"type": "object"}`; a branch with no `type` gets `"type": "object"`; a branch type list
   that includes `object` becomes `"object"`;
4. a branch with none of `properties`, `patternProperties` and `additionalProperties` gets each of them that the root
   has, with every `description` inside the copies removed; a branch with any of them keeps its own;
5. nested unions inside a branch are not touched.

On the real schema this gives each of the two branches `"type": "object"`, the seven properties without descriptions,
`additionalProperties: false` and its own `required`: 1,156 bytes become 1,736 (3,324 if descriptions were kept; 1,188
for typing alone; R4).

### D3. What is not rewritten (Q2, decided: sent as given and flagged once)

A schema outside D2 is sent as gobble sends it today. When its root has a union with a branch that is not a typed
object, the shape P1 shows Grok refusing, the tool is flagged (D4). Nothing is dropped or degraded at load.

### D4. What the user is told (Q3, decided: rewrites in the debug log, one warning per session per flagged tool)

* A rewritten tool: nothing beyond a debug log line naming the server, the tool and the rule.
* A flagged tool: one warning per session per tool, through the MCP diagnostics' channel (G15): a log record, which the
  terminal client shows as a `Warning:` line (0024-MADR-cli-mode-polish.md, amendment of 2026-10-20), and a notice for
  an ACP client that offered notices. Title `mcp: tool schema sent as the server gave it`; text naming
  `<server>/<tool>`, the reason (for example "its root anyOf has a branch that is not an object, which some providers
  refuse"), and the remedy (`--exclude-tools mcp__<server>__<tool>`, or a permission rule that denies it).

### D5. Out of this record

* The SDK: no change; its draft's D5 stands (the SDK never rewrites a schema).
* The server (Q5, decided: both fixes, by that repository's own record): the findings M5 and M8 go to that record;
  nothing in `mcp-server-magictools` changes here (More Information, "Follow-ups").
* Exposure, naming and permission rules: unchanged. 0005-PLAN F8's deferred MCP tools, when built, send the same
  rewritten schema when loaded.
* The pins: no module is added or bumped. 0042-MADR-acpext-extension-contract-module.md stands: `acpext` is required by
  tag, with no `replace` and no committed `go.work`.

### D6. The typed-error follow-up (Q4 and Q7, decided)

A record of its own, drafted when the SDK's release is tagged, after go-llmprovider-sdk releases `ErrToolSchemaRefused` and `APIError.Tool` and gobble's pin
moves to that release. What it would decide, per the SDK draft's D9: on that kind with a tool named, before any event,
drop the tool for the session on that provider and model, resend once and tell the user; with no tool named, end the
turn naming the cause; test the kind before the `signature` match, which is where the misfire R3 measured is fixed
(Q7). What it must first settle: whether a history
holding calls to the dropped tool is accepted (the SDK draft's O5, not established), and how a drop interacts with
`tool_search` and loaded deferred tools.

### D7. Where it lands (Q6, decided as re-put)

Next, before 0052-PLAN P5, on `p3-acpext-v0.5.0`, with a fresh draft pull request to `main` so CI runs on every push.
On 2026-11-17 `main` was fast-forwarded to `2004980` and the branch's earlier pull request was merged, so Q6 was put
to the owner again; the "before the fast-forward" of the drafted option (b) is overtaken by events. `main` moves only on
the owner's word at a phase boundary. It ships in `v0.7.0`.

### Consequences

* Good, because the owner's failure is removed before any request, on every provider, with no table to maintain.
* Good, because the arguments the server's schema accepts are exactly the ones the rewritten schema accepts (R1, R2);
  the model is told the same contract, in the form xAI documents.
* Good, because the schema stays the same for a session's life, so the prompt cache's prefix is stable after the first
  request that carries it.
* Neutral, because the server validates nothing against its schema (M9), so what the server accepts is its handler's
  rule (M8), which neither the original nor the rewritten schema states.
* Bad, because every provider's request changes for a tool with a root union, and whether each route accepts the
  rewritten form is not established; a route that accepted the original and refuses the rewrite would fail every prompt
  as P1 does (the owner's live step covers Grok; the others are optional, PLAN P3).
* Bad, because the schema grows (1,156 to 1,736 bytes on the real tool): the model reads each property's type three
  times, its description once.
* Bad, because a tool outside D2 still fails every prompt on a refusing route until D's record lands; D4 tells the
  user the remedy.

### Confirmation

1. Unit tests over the real schema (copied as data, its source named) and the shapes S2 to S12: the rule's output, the
   cases it refuses, and input preservation checked with jsonschema-go over a generated corpus, each seen failing first.
2. An end-to-end test: an in-process MCP server offering the real schema, an in-process fake provider that refuses the
   un-normalised shape with xAI's recorded body, and a prompt that fails before the change and completes after it,
   with a call of the tool reaching the server.
3. A planted break per new assertion, on a scratch copy, each seen to fail.
4. An independent review of the rewritten schema, since it changes what the model is shown.
5. The owner's live step: `grok-4.6` with the magictools server, the prompt completes and the tool is called.

## Pros and Cons of the Options

### A. Normalise at load, every provider, input-preserving only

* Good, because measured input-preserving (R1, R2), and in the form X1 and K4 show.
* Good, because one place, one rule, no provider table; prevention, with no round trip.
* Neutral, because routes that accepted the original also receive the rewritten form; none has been asked (Q1).
* Bad, because it covers root unions only; every other refusal still needs D.

### B. The same rewrite, only for Grok, at request time

* Good, because no other route's request changes.
* Bad, because refusal is per route, not per provider (the SDK draft's table), so "Grok" is either too narrow (a Grok
  model behind another route) or a list to keep.
* Bad, because a tool's spec and what one provider is sent then differ within a session, so the wire tests must cover
  each provider the rewrite applies to.

### C. Lossy flatten or drop, with a notice

* Good, because simplest, and the request passes.
* Bad, because measured lossy: dropping accepts 2,637 more objects on the real schema; kilocode's flatten accepts no
  valid call (R1).
* Bad, because the model loses the rule "`target` and `intent`, or `session_id`".

### D. Detect, drop, resend, tell

* Good, because it recovers from refusals nobody has catalogued.
* Bad, because today it would parse provider text (W5 of the SDK draft: Anthropic advises against it), and `signature`
  already misfires (R3).
* Bad, because the first prompt of each session pays a failed request, and the tool is lost where a rewrite would have
  kept it.
* Neutral, because the SDK's typed error removes the parse; it is the right fallback behind A (D6).

### E. Validate at load and refuse to register

* Good, because the user learns at start, as Claude Code's `/mcp` shows (CC2).
* Bad, because which routes refuse what is not established, so the check either removes tools that work elsewhere or
  carries a table nobody measured.

### F. Nothing in gobble; fix the server

* Good, because the owner's server is the owner's (Q5).
* Bad, because gobble stays broken for every other server with the same shape, and the server's handler disagrees with
  its own schema anyway (M8).

### G. Degrade to an open object, the schema in the description

* Good, because the tool stays callable on every route, as in Grok's own CLI.
* Bad, because the model loses typed arguments and the provider's grammar no longer constrains them; arguments are
  checked only by the server's handler.
* Neutral, as Q2 (b) for what A cannot rewrite, on a known refusal.

## Open questions for the owner

### Decided (2026-11-17)

The owner's rulings, relayed by the coordinator; each label quoted as given, what was asked beside it. The owner was
asked this record's Q7 sixth, and Q6 last, re-put after `main` moved.

* **Q1, where the rewrite applies:** "Every provider, when the tool is loaded (Recommended)". Should the owner's live
  step show another provider refusing the rewritten form, the question comes back.
* **Q2, a schema the rule cannot rewrite without loss:** "Send it as given and flag it once (Recommended)".
* **Q3, what the user is told:** "Rewrites in the debug log; one warning per session for each flagged tool
  (Recommended)".
* **Q4, the typed-error follow-up:** "A record of its own, drafted when the SDK release is tagged (Recommended)".
* **Q5, the server-side fix in `mcp-server-magictools`:** "Yes, both, by that repository's own record (Recommended)":
  type the branches at the source and make the handler agree with the schema. That repository is outside this effort's
  six, and each commit there needs the owner's word.
* **Q7, the `signature` misfire:** "In the follow-up record, by checking the SDK's error kind first (Recommended)".
* **Q6, where and when it lands, re-put:** "Next, before 0052 P5, on the same branch with a fresh draft pull request
  (Recommended)". As put: work continues on `p3-acpext-v0.5.0` with a new draft pull request to `main`, CI on every
  push, `main` moved only on the owner's word at a phase boundary, shipping in `v0.7.0`.
* **How the pair proceeds:** "Approve through my live step too, then ask about `main`". As put: this pair accepted and
  committed with the answers; the rule, its tests, the warning and the docs built in one commit; an independent review;
  the owner's live commands; after they pass the plan closes, and the coordinator asks the owner only for the `main`
  fast-forward.

### The questions as drafted

In the order drafted: Q1 fixes the decision's reach, Q2 and Q3 its edges, Q4 and Q5 what follows, Q6 and Q7 timing and
an adjacent fix.

* **Q1 (D1). Every provider, or only where a refusal is known?**
  * **(a)** Every provider, at load. *Recommended:* the rewrite accepts exactly the same arguments, keeps one schema per
    tool for the session, and needs no route table. Cost: every route sees the rewritten form untested; the owner's
    optional PLAN P3 step 6 checks the providers the owner uses.
  * **(b)** Grok's provider only, at request time (option B). Cost: a list of routes to keep, and two forms of one
    tool's schema in one session.
  * **(c)** Every provider, but only "type each branch", without copying the root's keywords. Cost: per K4 (inferred),
    Grok may then compile those branches with unconstrained arguments.
* **Q2 (D3). A schema the rule cannot rewrite without loss?**
  * **(a)** Send it as given and flag it once (D4). *Recommended:* nothing that works on a route is taken away; D's
    record later recovers where a route refuses.
  * **(b)** Degrade it to an open object with the schema in its description (option G), for every provider. Cost: the
    model loses typed arguments everywhere, to avoid a refusal seen on one route.
  * **(c)** Leave the tool out at load and say so (option E). Cost: the tool is lost on routes that would accept it.
* **Q3 (D4). What the user is told, and where?**
  * **(a)** Nothing for a rewritten tool beyond a debug log line; one warning per session per flagged tool, as a log
    record and a notice. *Recommended:* a correct rewrite needs no attention, and an unfixed shape needs its remedy once.
  * **(b)** As (a), plus one info line per session per rewritten tool. Cost: noise on every start for a server like
    magictools.
  * **(c)** No warning; the debug log only. Cost: the user learns of a flagged tool from the provider's 400.
* **Q4 (D6). The typed-error follow-up: a phase here or a record of its own?**
  * **(a)** A record of its own, drafted when the SDK's release is tagged. *Recommended:* it is a different decision
    (changing what the model can do mid-session), it needs O5's answer first, and it waits on another repository's
    release with no date.
  * **(b)** A last phase of this plan, gated on the release and the pin bump. Cost: this plan stays `in-progress` until
    then, and the drop's own decision needs an amendment of this MADR before that phase runs.
  * **(c)** Not at all; A only. Cost: every other refused schema keeps failing every prompt.
* **Q5. The server-side fix in `mcp-server-magictools`?**
  * **(a)** Yes, by that repository's own record: type the branches (or rewrite them as D2 does), and make the handler
    agree with the schema (M8 refuses the `session_id`-only continuation M5 advertises). *Recommended:* the handler's
    disagreement is a defect whatever gobble does.
  * **(b)** Only the handler's disagreement; leave the schema to gobble's rule.
  * **(c)** Nothing there.
* **Q6 (D7). Where it lands.**
  * **(a)** On `p3-acpext-v0.5.0` at the first phase boundary after approval, shipping in `v0.7.0`. *Recommended:* the
    owner's daily use with Grok and magictools fails today. Cost: one more phase between the branch's phases, and a
    second `make preflight` queue slot.
  * **(b)** After 0052 P6 and 0051 P8 and P8a, before the fast-forward. Cost: the failure stays until the branch ends.
    (Overtaken by events: `main` was fast-forwarded to `2004980` on 2026-11-17; see "Decided".)
  * **(c)** On `main` after `v0.7.0`, in the next patch release. Cost: as (b), and one release later.
* **Q7. The `signature` misfire (R3)?**
  * **(a)** Fix it in D6's record, by testing the SDK's kind before the text match. *Recommended:* the typed kind
    replaces the text, and the misfire costs one wasted request on a turn that fails anyway.
  * **(b)** Fix it now in this plan, by excluding xAI's `[invalid_client_tool_schema]` tag from the match. Cost: one more
    provider text match to keep, which D6 would remove.
  * **(c)** Leave it.

## More Information

### The drafting experiments (2026-11-17, outside the working tree)

All in a `git archive 2004980` copy and a folder beside it, under `<scratch>`; scripts and result tables kept with the
session's records, not committed.

* **R1, `preserve.py`.** A stdlib JSON Schema validator for the keywords used (2020-12; draft-07 `$ref` on request),
  its own 16 self-checks, the rewrites (type each branch, copy the root's object keywords with and without overwriting,
  codex's lowering, remove non-object branches, drop the union, kilocode's flatten, and D2's rule), and the corpus
  described under Context. Run in 2020-12 mode (exit 0), draft-07 mode (exit 0, the same table), with the rule replaced
  by a planted one that drops the union (exit 1, differences on 9 shapes), and on the real schema verbatim (exit 0).
* **R2, the cross-check.** A Go test in the archive copy validates the same corpus with jsonschema-go `v0.4.3` and
  writes each schema's verdicts; `compare.py` compares them with R1's: 0 disagreements on both corpora (exit 0), and 104
  with one Python verdict per schema flipped (exit 1).
* **R3, the misfire.** A scratch test in the copy's `llm/provider` points gobble's `grok` provider at an `httptest`
  server that answers every request with status 400 and xAI's recorded body shape (SDK1), naming the request's tool. For
  `mcp__magictools__execute_pipeline` and `mcp__docs__fetch_api_key`: one request each; for
  `mcp__vault__verify_signature`: two. The error text equals P1's. The tools body gobble sent carried the union exactly
  as the server gave it, with `"strict": false`.
* **R4, `realschema.py`.** Parses the inventory: 18 tools, every root `type: object`, one root union. Sizes as in D2.

### Follow-ups (not part of this record)

* A record in `mcp-server-magictools` (Q5): type `execute_pipeline`'s branches at the source, and make the handler
  (M8) agree with the schema (M5). Outside this effort's six repositories; each commit there needs the owner's word.
* A gobble record for the typed-error fallback (Q4, D6), drafted when go-llmprovider-sdk tags the release carrying
  `ErrToolSchemaRefused` and `APIError.Tool`; it also fixes the `signature` misfire (Q7, R3).

### Not verified

* Whether Grok accepts D2's form (P3's live step), and whether a branch with `"type": "object"` alone passes.
* Every route but Grok's answer to the owner's shape and to D2's form.
* That xAI compiles each branch alone: inferred from K4's comment and X1's example.
* Claude Code's root-union rewrite: documentation names it (CC3), not its form.
* That the terminal client shows the warning as a `Warning:` line: read from 0024-MADR's amendment, not run; the PLAN's
  test checks it.
* Which runtime path of kilocode uses its flatten (L1).
* `unevaluatedProperties`, `$dynamicRef` and dependent keywords: argued outside the rule, not measured.

### Records this bears on

* `0012-MADR-tool-schemas-descriptions-and-loading.md`: tool schemas and loading; the MCP schema path is extended,
  nothing there superseded.
* `0005-PLAN-v1-feature-scope.md` F8: MCP exposure modes, which will send the same rewritten schema.
* `0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md` D8: MCP diagnostics as notices, the channel D4 uses.
* `0042-MADR-acpext-extension-contract-module.md`: the pin rule, unchanged.
* go-llmprovider-sdk, its draft record `NNNN-MADR-typed-error-for-a-refused-tool-schema.md` (proposed, not placed):
  the typed error D6 waits for.

### Related

* [0060-PLAN-normalise-root-level-unions-in-mcp-tool-schemas.md](0060-PLAN-normalise-root-level-unions-in-mcp-tool-schemas.md)
* [0012-MADR-tool-schemas-descriptions-and-loading.md](0012-MADR-tool-schemas-descriptions-and-loading.md)
* [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md)
* [0024-MADR-cli-mode-polish.md](0024-MADR-cli-mode-polish.md)
* [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md)
* [0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
* [0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)

## Amendment (2026-11-17): the rule refuses scope keywords anywhere in the union or the copied properties

*Accepted by the owner the same day, relayed by the coordinator, after the independent review (0060-PLAN P2) of
commit `cde93be`. Nothing above is rewritten; the lines this changes carry a pointer here.*

**What the review found.** `cde93be` implements D2 as written, and the notices, the remedies, the aliasing and the
call path work. D2's argument (Context, steps 1 to 6) holds for what it names, but its conditions looked only at the
root's own keywords and at a `$ref` at a branch's top level. Measured with two validators outside the tree
(`github.com/google/jsonschema-go` `v0.4.3` and `github.com/santhosh-tekuri/jsonschema` v6, which agreed on the
originals everywhere), the rule changed verdicts, or made the schema invalid, for three kinds of schema it rewrote:

1. **`unevaluatedProperties` in a union branch.** `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer"},"c":{"type":"boolean"}},"anyOf":[{"unevaluatedProperties":false,"required":["a"]},{"required":["b"]}]}`
   rejects `{"a":"x"}`; copying the root's properties into the branch makes `a` evaluated, and the rewrite accepts it
   (180 of that shape's objects newly accepted; 303 with the root's `additionalProperties: false`).
2. **A `$ref` anywhere but a branch's top level.** `{"type":"object","properties":{"a":{"$ref":"#/anyOf/0"}},"anyOf":[{"required":["a"]},{"required":["zz"]}]}`
   accepts `{"a":"x"}`; once the branch is typed `object` and given the properties, `#/anyOf/0` is that branch and `a`
   must be an object (up to 1,948 objects newly refused). A branch removed ahead of a `$ref` into the union shifts what
   it points at the same way (`{"type":"object","properties":{"a":{}},"anyOf":[{"type":"string"},{"properties":{"a":{"$ref":"#/anyOf/0"}},"required":["a"]}]}`,
   1,197 newly refused).
3. **`$id` or `$anchor` (and `$dynamicAnchor`) inside the properties the rule copies.** A copy repeats the name once
   per branch; the duplicates made the rewritten schema invalid for one of the two validators.

The real magictools schema has none of these and is preserved exactly (R1, R2, and this pass's corpus).

**The owner's ruling R-1:** "Refuse the rule whenever any of those keywords appears anywhere in the union or the copied
properties (Recommended)". As put: such a schema is sent as given and flagged, like every other shape the rule does not
handle; the cost is that it also declines a harmless self-reference (`$ref: "#"`), which is merely flagged.

**D2 as amended.** A fifth condition (added to the list above): none of `$ref`, `$dynamicRef`, `$recursiveRef`, `$id`,
`$anchor`, `$dynamicAnchor`, `$recursiveAnchor`, `unevaluatedProperties` and `unevaluatedItems` appears as a keyword
anywhere in the union (any depth of any branch) or anywhere in the root's `properties`, `patternProperties` and
`additionalProperties`. "Anywhere" means in a schema position of JSON Schema 2020-12 (and the draft names `definitions`,
`dependencies`, `additionalItems`): `properties`, `patternProperties`, `additionalProperties`, `items`, `prefixItems`,
`allOf`, `anyOf`, `oneOf`, `not`, `if`, `then`, `else`, `dependentSchemas`, `propertyNames`, `contains`, `contentSchema`,
`$defs`, looked into through lists and through the maps of names to schemas. The names in such a map are never keywords,
so a tool with an argument called `$ref` or `$id` is rewritten, not refused; and what a keyword holds as data (`enum`,
`const`, `default`, `examples`, `required`) is not looked into. A refused schema takes D3's path: sent as given, and
flagged when its root union has a branch that is not typed `object`, with the reason `the union or the keywords it would
copy hold "<keyword>", which the rule does not cover`. Context step 6 and "Not verified"'s last bullet (the keywords
"argued outside the rule, not measured") are answered by this: they are now measured, and refused.

**The owner's ruling R-2:** "Keep it, and add a test for hidden-then-shown (Recommended)". The behaviour built beyond
the plan in P1, that a tool the session's permission rules hide is neither sent nor warned of (D4's remedy "a permission
rule that denies it" has to end the warning), stays; a tool hidden and then shown within one session is warned of once,
at the first prompt that sends it. D4's notice names the remedy as a rule that denies the tool "with no pattern", which
is what hides it (`permission.Engine.Hidden`).

**Measured and held.** The preservation corpus of the plan, with jsonschema-go, over every rewritten case, now with the
review's counter-examples as cases the rule leaves as given; a matrix of the nine keywords in each of 38 positions of a
union or of the copied keywords, each refused and each position's control rewritten; and the property-named cases. One
validator is in the tree's tests: `santhosh-tekuri/jsonschema` is not in `go.mod`, and `AGENTS.md` requires a MADR that
names a module before it is required; this amendment does not add it, so the second validator's agreement is the
review's, outside the tree. jsonschema-go accepts the duplicate `$id` and `$anchor` copies, so the tree's test holds
those by the rule's own refusal (a search of every rewritten schema for the nine names), not by a verdict.

