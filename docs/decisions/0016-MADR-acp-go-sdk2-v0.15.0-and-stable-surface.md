---
status: proposed
date: 2026-10-15
decision-makers: repository owner
consulted: none
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Take acp-go-sdk2 v0.15.0, close the gap it opens, and adopt the stable ACP surface gobble can honour

*(2026-10-13, [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md) D10: this record is re-based, not duplicated. 0019-PLAN P4 (`17cd497`) pins acp-go-sdk2 `v0.17.1`. D1 is superseded by 0019-MADR D3; D2 by the SDK's own `cwd` check, from `v0.16.0` (0019-MADR F5), and 0019-MADR D7 removed D2's empty-`cwd` fallback; D3 is superseded by 0019-MADR D8; D7's accessors are written over the SDK's `Negotiated()`. D4 to D6 and D8 to D13 stay this record's, `proposed`, and are re-checked against `v0.17.1` before its PLAN is written, since the SDK's gates make an ungated update fail a turn (0019-MADR F6).)*

*(2026-10-15: amended; see "Amendment (2026-10-15)". acp-go-sdk2 `v1.14.0` is pinned and partly adopted (A1), and D11's auth bullets are replaced by A2, decided by the owner. The rest of this record stays `proposed`.)*

## Context and Problem Statement

gobble already speaks ACP through acp-go-sdk2: [0014-MADR-move-to-acp-go-sdk2.md](0014-MADR-move-to-acp-go-sdk2.md) moved it off the fork to `v0.14.0` on 2026-10-10, and only `acpserver` and `acpclient/...` import it (`internal/archtest/check.go:186`, `:208-215`). The owner has asked for the SDK to be integrated fully: its API surfaces researched, and the most idiomatic, Go 1.27.2-conformant path chosen.

Since 0014, the SDK has tagged `v0.15.0` (2026-10-11), with two breaking type changes and a looser `Validate`. More of ACP is now in the stable schema than gobble's records say, and gobble uses only part of it. This record decides three things:

- whether and how gobble takes `v0.15.0`;
- which stable parts of the SDK gobble adopts now, and how, without advertising anything it cannot honour;
- which parts stay with the records that own them, or wait.

It decides nothing about the SDK itself, which is the owner's separate repository.

### What was measured, not assumed

Measured read-only on 2026-10-12. The source was the owner's local clone of acp-go-sdk2, at `bc0cb78`, one commit past `v0.15.0` (`d5a336c`). That commit changes only `docs/` (`git diff --stat v0.15.0 HEAD`: two files, both under `docs/`). Every SDK fact below was read with `git show v0.15.0:<file>`. gobble was built and tested against `v0.15.0` in scratch clones of its HEAD (`6a8b9d8`), which the repository never saw. Scripts and logs are in the session scratchpad, named in the evidence index.

**The release.**

- `v0.15.0` is published: `proxy.golang.org` serves it at `d5a336cbd9e06df2b5450377bec314ab435f32be`, `refs/tags/v0.15.0`. `sum.golang.org` records `h1:O6lhX2ac2QxVC+8hencA5nXueH0foxTW4IN8mJ+os38=`, and its `go.mod` sum is `v0.14.0`'s, `h1:KqDRvShTbZEOpKOQOyRUnbeFqlyH20LXi7k3UOsgv0I=`. The SDK still has no requirements.
- Its schema is still ACP `1.25.0` (`schema/version`), with protocol version 1 (`constants_gen.go:6`).
- The release notes (SDK `docs/reports/0013-REPORT-release-notes-v0-15-0.md`) list two breaking changes:
  - `ErrorCode` is an `int32` with eight constants (`types_gen.go:5189-5199`);
  - `ExtRequest`, `ExtResponse` and `ExtNotification` are `json.RawMessage`.
- They also list changes in what the SDK accepts:
  - elicitations carry a scope;
  - keys are matched case-insensitively on decode;
  - an optional raw `null` counts as absent;
  - lenient properties take their defaults;
  - `Validate` checks at every depth and "no longer rejects a present empty string; `v0.14.0` did, in 28 places".
- Of the SDK's hand-written files, only `helpers.go` changed (`git diff --stat v0.14.0 v0.15.0`): a null raw input is now treated as absent in `withRawPath`. `errors.go` is unchanged, and `RequestError.Code` is still an `int`.

**gobble on `v0.15.0`, unchanged** (`q398_experiments.py acp-0.15.0`):

- `go get` and `go mod tidy` change one `go.mod` line and two `go.sum` lines.
- `go build ./...`, `go vet ./...` for linux, darwin and windows (`CGO_ENABLED=0`), and `govulncheck ./...` ("No vulnerabilities found") all exit 0.
- `go test -count=1 ./...`: 28 packages ok, one failing:

  ```text
  --- FAIL: TestElicitationNotFound (0.01s)
      elicitation_test.go:75: elicitation/create: {"code":-32602,"message":"Invalid params","data":{"error":"json: error calling MarshalJSON for type *acp.CreateElicitationRequest: json: error calling MarshalJSON for type *acp.CreateElicitationUrl: CreateElicitationUrl: no member set"}}, want method not found
  ```

  The test's request (`acpclient/elicitation_test.go:71-72`) has no scope, and `v0.15.0` refuses to encode it, on the sending side, before gobble's handler sees anything. With `Session: &acp.ElicitationSessionScope{SessionId: "s0"}` added on the scratch clone, and the -32601 assertion unchanged, the test passes (`q415_elicitation_fixture.py`).
- gobble names neither `ErrorCode` nor any `Ext*` type (a search of every `.go` file).

**The looser `Validate` reaches gobble.** `v0.14.0` had 28 empty-string checks in `Validate` methods, and `v0.15.0` keeps none of them (`q414_empty_string_checks.py`).

- The SDK runs `Validate` on inbound requests only (`agent_gen.go:17` and on). Responses are not validated (`agent_gen.go:493-496`), and gobble's client answers every file and terminal request with -32601. So the three `Cwd` checks are the only ones that matter to gobble: `NewSessionRequest`, `LoadSessionRequest` and `ResumeSessionRequest`.
- gobble checks additional directories for absoluteness (`acpserver/agent.go:830-839`), but never checks `cwd`.
- A probe test, run on both pins (`q413_probe_cwd.py`), sent `session/new` with three values of `cwd`:

  | `cwd` | `v0.14.0` | `v0.15.0` |
  | :--- | :--- | :--- |
  | `""` | refused: `-32602 … "cwd is required"` | **accepted; roots `[]`; `Outside("C:\WINDOWS")` is false** |
  | `relative/dir` | accepted; roots `["relative/dir"]` | the same |
  | a temporary absolute directory | accepted; roots `[dir]`; `Outside` true | the same |

- `fsx.Workspace` documents "A Workspace with no roots confines nothing" (`internal/fsx/workspace.go:22`, `:32-34`, `:87-89`). On `v0.15.0`, then, a client's `session/new` with an empty `cwd` opens a session whose file tools have no workspace boundary.
- `open`'s fallback for an empty `cwd` on load and resume, the stored header's `cwd` (`acpserver/agent.go:728-730`), could not be reached on `v0.14.0`, which refused the request first. On `v0.15.0` it can.
- gobble's own CLI always sends an absolute `cwd` (`internal/cli/chat.go:76-82`, `os.Getwd` and `filepath.Abs`).

**The stable schema is wider than gobble's records say** (`q412_schema_stability.py`, splitting `schema.json` and `meta.json` from their `unstable` siblings):

- 25 methods are stable, including:
  - `session/delete`, `logout`, `elicitation/create` and `elicitation/complete`;
  - every `terminal/*` and `fs/*` method;
  - `$/cancel_request`.

  16 are unstable-only: `session/fork`, the five `document/*`, the five `nes/*`, the three `providers/*`, and `mcp/message` on both sides.
- 14 `SessionUpdate` variants are stable, among them `usage_update`, `notice`, `compaction_update` and `compaction_summary_chunk`. Five are unstable: `plan_update`, `plan_removed`, `subagent_update`, `session_message` and `session_message_chunk`.
- The stable capability properties include:
  - `ClientCapabilities.session`: `compaction`, `configOptions` and `notices`;
  - `ClientCapabilities.auth.terminal` and `ClientCapabilities.elicitation`: `form` and `url`;
  - `AgentCapabilities.auth.logout` and `SessionCapabilities.delete`.

  Unstable-only are `nes`, `positionEncoding(s)`, `providers`, `plan`, `subagents`, `SessionCapabilities.fork` and `McpCapabilities.acp`.
- The schema gates each of these on a capability:
  - `Notice`: "Agents MUST only send notices when the Client advertised `ClientSessionCapabilities::notices`. Otherwise, Agents may use an agent message";
  - `CompactionUpdate` and `CompactionSummaryChunk`: "Agents MUST only send this update when the Client advertised `ClientSessionCapabilities::compaction`";
  - `AuthMethodTerminal`: "Agents MUST advertise this method only when the client enabled its terminal authentication capability … The client MUST NOT pass this method to `authenticate`".
- The records still say otherwise:
  - 0004-MADR lists "fork, delete, NES, documents, providers, elicitation, `mcp/*`" as unstable (`0004-MADR-go-module-architecture.md:45-47`);
  - 0005-MADR calls `session/delete` unstable (`:148`), and ACP elicitation unstable twice (`:192`, `:614`);
  - 0002-MADR calls `session/delete` unstable (`:1030`) and `usage_update` non-standard (`:977`).

  Each was true of schema 0.13.5, which they were written against, and none is true of 1.25.0.

**What gobble uses, and what it does not** (`q411_acp_surface_map.py`; each row checked by reading the file it names):

- **Agent methods:**
  - real: `initialize`, `session/new`, `load`, `resume`, `list`, `prompt`, `cancel`, `close`, `set_mode` and `set_config_option`;
  - answering -32601, unadvertised: `session/delete`, `authenticate` and `logout` (`acpserver/agent.go:887-891`, `:933-941`).
- **Unstable methods:** none implemented. For a method whose interface the agent lacks, the SDK answers -32601 itself, by an interface assertion per method (`agent_gen.go:222-226`, `:361-365`; `client_gen.go:71-73`). gobble needs no stubs for them.
- **Client methods:** `session/update` and `session/request_permission` are real. The `fs/*`, `terminal/*` and `elicitation/create` methods answer -32601, and `elicitation/complete` is ignored (`acpclient/conn.go:414-462`).
- **Client capabilities:** gobble's client advertises none (`acpclient/conn.go:121-124`). The golden's `initialize` carries only the SDK's defaults, `{"auth":{},"fs":{}}` (`acpserver/testdata/session.golden:1`).
- **The agent reads client capabilities** only for `fs` and `terminal` (`acpserver/agent.go:226`, `:467-472`). No agent code reads `clientCapabilities.session`, `auth` or `elicitation`.
- **Updates sent:**
  - eleven of the fourteen stable variants, `user_message_chunk` among them, on replay and for a queued message (`acpserver/history.go:309`, `:323`; `acpserver/prompt.go:118`);
  - never `notice`, `compaction_update` or `compaction_summary_chunk`.

  Compaction reaches the client as agent text: "Compacting the conversation…" and "Compacted from N tokens." (`acpserver/compact.go:24`, `:37`), and its replay as agent text too (`acpserver/history.go:301-302`).
- **Updates the client translates** (`acpclient/update.go:125-186`):
  - nine kinds;
  - `user_message_chunk`, `config_option_update` and the three above become `KindOther`. Their params are kept (`acpclient/update.go:13-16`, `:74-76`), and the print sink streams them (`internal/cli/print.go:85`).
- **Diagnostics a user should see reach only the log:**
  - four MCP warnings (`acpserver/mcp.go:151`, `:153`, `:164`, `:168`);
  - a system-prompt file not being used (`acpserver/sysprompt.go:218`).

  Under `gobble acp` the logger has no stderr (`internal/cli/main.go:169`), so an editor's user never sees them.
- **Helpers:**
  - 21 of the SDK's 44 are used in non-test code;
  - the 23 unused are:
    - `AudioBlock`, `ResourceLinkBlock` and `ResourceBlock` (the last two in tests only);
    - `UpdateUserMessage`, `UpdateAgentMessage` and `UpdateAgentThought` (gobble uses their `…Text` forms);
    - `WithStartRawOutput`, `WithUpdateTitle`, `WithUpdateKind`, `WithUpdateRawInput` and `WithUpdateRawOutput`;
    - `StartReadToolCall` and `StartEditToolCall`;
    - the three `NewPlanUpdateContent…`, the two `NewSessionConfigOption…` and the four `NewUnstableNesSuggestion…` constructors;
    - `Ptr`. gobble uses Go 1.26's `new(expr)` for `Ptr`'s job, for example `new(oneLine(meta.Name))` at `acpserver/agent.go:297`.
  - Five updates have no SDK helper, and gobble builds them by hand (`acpserver/commands.go:330`, `:387`, `:516`; `acpserver/modes.go:50`; `acpserver/prompt.go:304`).
- **Hand-written discriminators.** Each of those five literals sets the variant's `SessionUpdate` field to its discriminator string. The SDK's encoder overwrites that key from the variant (`types_gen.go:12901-12923` and on: `m["sessionUpdate"] = …`), and its own helpers leave the field empty (`helpers.go:128-131`). A probe encoded a `usage_update` with the field empty and with the field `"wrong"`, and both gave `{"sessionUpdate":"usage_update","size":0,"used":5}` (`probe-acp015`).
- **Magic error numbers.** gobble's non-test code compares JSON-RPC codes by literal in three places: `authRequiredCode = -32000` (`acpclient/conn.go:31`), and `-32600` and `-32602` (`acpclient/sessions.go:47`, `:49`). `v0.15.0` names each: `ErrorCodeAuthenticationRequired`, `ErrorCodeInvalidRequest` and `ErrorCodeInvalidParams`. The probe printed `-32000 == int(ErrorCodeAuthenticationRequired)` true, and `-32601 == int(ErrorCodeMethodNotFound)` true.
- **Tests name codes by number:** 53 lines in 17 test files under `acpserver`, `acpclient` and `internal/cli`. These are assertions such as `acpserver/agent_test.go:162`, and `acptest` turns that set `ErrCode`, such as `acpclient/conn_test.go:328`.
- **The goldens conform to `v0.15.0`.** A probe decoded every frame of `acpserver/testdata/session.golden` (13 frames) and `acpclient/acptest/testdata/initialize.golden` (2 frames) into its SDK type and ran `Validate`. 15 of 15 passed (`q416_golden_conformance.py`). Two broken copies each failed, exit 1:
  - one without a `session/update`'s `sessionId`: `SessionNotification: missing required property "sessionId"`;
  - one with a string `protocolVersion`.

  No gobble test runs `Validate` today (a search of every `.go` file).
- **The connection wiring is the SDK's documented shape.** `NewAgentSideConnection(agent, out, in, acp.WithLogger(…))`, then `agent.SetConnection(conn)` (`acpserver/serve.go:32-33`), is the pattern of the SDK's `example/agent/main.go:416-418`. That example's comment names an `acp.AgentConnAware` interface (`:150`) which no SDK file declares. This is an SDK documentation defect, not gobble's.
- **F4-3's planned ACP auth breaks the schema's rule.** 0005-PLAN's F4-3 has `initialize` list "a terminal auth method for each login", and `authenticate` "answers once that provider has a credential" (`0005-PLAN-v1-feature-scope.md:3306`). It gates nothing on the client's `auth.terminal`. Its file list changes `session.golden` "(the auth methods)" (`:3314`), whose client advertises `"auth":{}`, so terminal auth is off.

### Findings

- **F1.** `v0.15.0` is published, checksummed and dependency-free, and gobble builds, vets and passes govulncheck on it unchanged. The one failing test is a test fixture: it fails in the SDK's encoder, and a session scope fixes it.
- **F2.** Neither breaking change reaches gobble's code.
- **F3.** On `v0.15.0`, `session/new` with an empty `cwd` opens a session with **no file confinement**. It is refused on `v0.14.0`. A relative `cwd`, accepted on both, confines to a root relative to the agent process's directory, which the schema forbids: `cwd` "must be an absolute path". The bump must not land without gobble checking `cwd` itself.
- **F4.** Four records call stable ACP surfaces unstable or non-standard: 0004-MADR, 0005-MADR (three places) and 0002-MADR (two places). Plans built on them would wrongly route stable features to `exp/` or `acp.unstable`.
- **F5.** F4-3's ACP auth bullet contradicts the schema twice:
  - it lists terminal methods without checking the client's `auth.terminal`;
  - it has `authenticate` answer for a terminal method, which a client must not send.

  It also never advertises `agentCapabilities.auth.logout`, which a client reads to know `logout` exists.
- **F6.** gobble's agent never reads `clientCapabilities.session`, so it cannot use the stable `notice` and `compaction_*` updates. As a result:
  - five user-facing diagnostics reach only a log, and under `gobble acp` no user sees it;
  - compaction reaches the client as prose.
- **F7.** gobble's client advertises nothing, and folds every update it does not render into `KindOther`. It cannot benefit from F6 until it advertises the capabilities and translates the updates.
- **F8.** Five hand-written discriminator strings are dead data. The SDK writes the key, so a typo in one would not reach the wire. That is exactly why they mislead a reader into thinking they matter.
- **F9.** Three error codes in non-test code are magic numbers that `v0.15.0` names. The tests' literals are different: they pin the wire value independently of the SDK, which is what a protocol test should do.
- **F10.** gobble's goldens conform to the schema as `v0.15.0` validates it, but no test checks that, so a later change could send a frame the schema forbids, and only a peer would notice.
- **F11.** For the unstable methods, the idiomatic path is to implement nothing. The SDK answers -32601 per method when the agent lacks it, and unstable variants and capabilities are simply never sent or advertised.
- **F12.** Several stable surfaces are product features with owners already: `session/delete` (0005-MADR "Archive and delete", 1.x), elicitation (0005-MADR "Sampling and elicitation" and "Ask the user", 1.x), ACP auth (0005-PLAN F4-3), and images and audio (0005-MADR, amendment of 2026-10-10, item 2). Each needs a product decision this record cannot make, such as a permission request before delete, or a CLI form renderer.

## Decision Drivers

- **No regression in confinement.** A library bump must not loosen what a session's tools may touch (F3).
- **Honest advertisement** (0002-MADR): nothing is advertised, and no capability-gated update is sent, unless both sides can honour it (the schema's MUST rules, F5, F6).
- **The wire is the contract.** Every exchange gobble's goldens record stays byte-identical unless a decision here names the change, as [0014-MADR-move-to-acp-go-sdk2.md](0014-MADR-move-to-acp-go-sdk2.md) D5 held.
- **0004-MADR's boundaries:** only `acpserver` and `acpclient/...` import the SDK, and `internal/cli` sees only `acpclient`'s translated types (rule 2).
- **Use the SDK's own vocabulary** (its constants, constructors and dispatch) rather than parallel copies. Where the SDK lacks something, gobble waits or asks for it upstream, rather than copying it, as AGENTS.md has it for go-tui-lib and go-selfupdate-lib.
- **Go 1.27.2 idioms gobble already uses:** `errors.AsType`, `new(expr)`, `encoding/json/v2` in gobble's own code and `encoding/json` at the SDK seam, and range-over-func iterators. The integration adds no second style.
- **Records match the schema** gobble actually speaks (F4, F5).

## Considered Options

- **A — Take `v0.15.0` with the `cwd` check in the same change, then adopt the stable, capability-gated surfaces that need no new product decision: notices, compaction updates, and the client side of both. Correct the records, and leave product features with their owners.**
- **B — Take `v0.15.0` with the `cwd` check and the test fixture only,** and defer everything else.
- **C — Take `v0.15.0` and adopt every stable surface now,** including `session/delete`, elicitation on both sides, and ACP auth, pulling 0005's 1.x items and F4-3 forward.
- **D — Stay on `v0.14.0`** until a later minor.

## Decision Outcome

Chosen: **A**, because it is the largest change that stays honest and decided.

- It closes the confinement gap the bump would open (F3).
- It adopts every stable surface whose behaviour the schema already fixes (F6, F7).
- It removes the dead and magic values (F8, F9), and puts a conformance check under the wire (F10).
- It leaves to their owners the features that need a product choice (F12).

### The decisions

- **D1. Require `github.com/maccavelli/acp-go-sdk2 v0.15.0`, exactly,** with the sums F1 records. [0014-MADR-move-to-acp-go-sdk2.md](0014-MADR-move-to-acp-go-sdk2.md) gets a dated note that its D1 pin is superseded here. Its PLAN's Deferred item "each move is an amendment of 0014-MADR" is annotated: this move, which changes behaviour beyond the pin, is decided here instead. A later tag is an amendment of this record. *(2026-10-13: superseded by 0019-MADR D3, which pins `v0.17.1` (0019-PLAN P4, `17cd497`).)*
- **D2. gobble checks `cwd` itself, in the same commit as D1.** A `session/new`, `session/load` or `session/resume` whose `cwd` is not absolute is refused with -32602, through the same `invalidParams` as `additionalDirs` (`acpserver/agent.go:237-239`, `:830-839`), with the reason "cwd %q is not an absolute path". On load and resume only, an empty `cwd` keeps the designed fallback, the stored header's `cwd` (`:728-730`), which must itself be absolute or the request is refused the same way. Every accepted `cwd` is kept `filepath.Clean`ed, as the additional directories are. D1 never lands without D2. *(2026-10-13: superseded. The SDK checks `cwd` itself from `v0.16.0` (0019-MADR F5), and 0019-MADR D7 removed the empty-`cwd` fallback (0019-PLAN P4, `17cd497`).)*
- **D3. `TestElicitationNotFound`'s request carries `Session: &acp.ElicitationSessionScope{SessionId: "s0"}`.** Its -32601 and no-advertisement assertions stay unchanged. The SDK made the same fix to its own tests in `0e32255` (SDK `0011-PLAN-engine-tests.md`). *(2026-10-13: superseded by 0019-MADR D8. The SDK refuses first from `v0.17.0`: the test asserts its `CapabilityError` for `elicitation.url`, and -32602 on a raw send (0019-PLAN P4, `17cd497`).)*
- **D4. Non-test code names JSON-RPC codes by the SDK's constants:**
  - `acpclient/conn.go`'s `authRequiredCode` goes, and the comparison is `re.Code == int(acp.ErrorCodeAuthenticationRequired)`;
  - `acpclient/sessions.go:47` and `:49` compare against `int(acp.ErrorCodeInvalidRequest)` and `int(acp.ErrorCodeInvalidParams)`.

  Tests keep their numeric literals, deliberately: they assert the wire value, and must fail if a constant ever changed (F9).
  *(2026-10-22: carried out by [0031-PLAN-sdk-current-releases-and-adoption.md](0031-PLAN-sdk-current-releases-and-adoption.md) P3, on acp-go-sdk2 `v1.20.0`.)*
- **D5. The five hand-written discriminator strings are removed** *(2026-10-22: carried out by [0031-PLAN-sdk-current-releases-and-adoption.md](0031-PLAN-sdk-current-releases-and-adoption.md) P3; the goldens unchanged)* (`acpserver/commands.go:330`, `:387`, `:516`; `acpserver/modes.go:50`; `acpserver/prompt.go:304`). The variant's `SessionUpdate` field is left empty, as the SDK's helpers leave it. `session.golden` staying byte-identical is the proof (F8). Helpers for these five variants, and for `notice` and the compaction updates, are a request to acp-go-sdk2, not local wrappers (D13).
- **D6. A conformance test runs under every golden.** `acptest` gains `ValidateGolden(t, path)`. It decodes each recorded frame into the SDK type of its method (a response by its request's method), and fails on a decode error, a `Validate` error, or a method it has no type for. Every test that compares a transcript with a golden calls it. The type table lives in `acptest`, inside rule 2. The probe of F10 is its prototype: 15 of 15 frames pass today, and both broken copies fail.
- **D7. The agent keeps the client's capabilities as a typed view.** `a.client` already holds them (`acpserver/agent.go:100`, `:226`). It gains small accessors, `offersNotices()` and `offersCompaction()`, each reading `Session != nil && Session.Notices != nil` (or `Compaction`). Every capability-gated update is sent only behind them. *(2026-10-13, 0019-MADR D10: the accessors are written over the SDK's `Negotiated()`.)*
- **D8. Notices.** When the client offers notices, each of the five diagnostics of F6 is also sent to its session as a `notice` update, with these severities:

  | Diagnostic | Severity |
  | :--- | :--- |
  | MCP server failed or needs auth | `warning` |
  | MCP config error | `warning` |
  | MCP exposure ignored | `warning` |
  | MCP server still connecting | `info` |
  | prompt file not used | `warning` |

  The title is the log message, and the description its detail: the server, its state and its first error line, or the file and the reason. Each diagnostic is logged as today, whatever the client offers, and keeps its existing once-per-session rule (`reported`, `noted`). Without the capability nothing new is sent, as the schema allows ("Agents may use an agent message"): gobble keeps its turns' text unchanged rather than adding messages.
- **D9. Compaction updates.** When the client offers compaction, `compactSession`, the one compaction of `/compact` and `_gobble/compact`, reports its run through the stable updates:
  - `compaction_update` `in_progress` first, once `compaction.Prepare` has succeeded, so a refusal ("Nothing to compact", "Already compacted", or a session gobble does not rewrite) sends no compaction update and keeps its error answer;
  - one `compaction_summary_chunk` with the summary as a text block;
  - then `compaction_update` `completed`, its `_meta.gobble.tokensBefore` the count today's text reports;
  - or `failed`, with `error` the failure's first line, or `cancelled`.

  The `compactionId` is a fresh id from `Options.NewID` per run. `/compact` then sends neither of its two agent-text lines, since the client renders the run. `_gobble/compact`'s response is unchanged. On `session/load`, a compaction entry replays as one `compaction_update` `completed` whose summary is the entry's, and whose id is the entry's id. Without the capability, the text and the replay are exactly today's, and `session.golden` stays byte-identical.
- **D10. gobble's client advertises and translates both:**
  - `acpclient.Options` gains `Notices` and `Compaction` (`bool`), which set `ClientCapabilities.Session` in `initialize`;
  - `translate` gains `KindNotice` (an `Update.Notice{Severity, Title, Description}`) and `KindCompaction` (an `Update.Compaction{ID, Status, Summary, Error, TokensBefore}`, with summary chunks appended in order).

  `internal/cli`'s line mode sets both. It prints a notice as a warning line, as it prints dropped notifications, and prints the compaction's progress and its summary in the words `/compact` uses today. Print mode sets neither, because its result is the turn's agent text (`internal/cli/print.go:70-72`): `gobble -p "/compact"` and its JSON stream stay as they are.
- **D11. The records are corrected by dated notes, never rewritten:**
  - 0004-MADR (`:45-47`), 0005-MADR (`:148`, `:192`, `:614`) and 0002-MADR (`:977`, `:1030`) each get a note citing schema 1.25.0's stable lists (F4);
  - 0005-PLAN's F4-3 ACP bullet (`:3306`) and its files row (`:3314`) get a dated amendment, for the owner to approve with this record (F5):
    - terminal auth methods are listed only when the client sent `auth.terminal: true`;
    - a terminal method is never answered through `authenticate`;
    - ~~`logout` is advertised through `agentCapabilities.auth.logout`;~~
    - ~~`session.golden`'s client, which offers no terminal auth, sees no auth methods, so that golden does not change for F4-3.~~

    *(2026-10-15: the first two hold. The last two are replaced by A2 of the amendment of 2026-10-15: `agent` methods are always listed, so `session.golden` changes, and `logout` is not advertised.)*
- **D12. Unstable surfaces are neither implemented nor advertised.** gobble declares no `AgentExperimental` or `ClientExperimental` method (F11). Its `_gobble/fork` and `_gobble/clone` stay the fork path (0002-MADR), and `McpCapabilities.acp` stays refused (`acpserver/mcp.go:112-113`). A future unstable feature lives under `exp/`, as 0004-MADR requires.
- **D13. These stable surfaces stay with their owners, and this record does not build them** (F12):
  - **`session/delete`:** 0005-MADR "Archive and delete", now on a stable method. It needs a `session.Store.Delete`, which the store lacks today, and the permission request that row describes. Until then, the -32601 answer and the absent capability stay (0014-MADR D3).
  - **Elicitation:**
    - agent side, for `ask_user` free text and MCP elicitation;
    - client side, a CLI form and URL renderer.

    These are 0005-MADR's rows, now stable. Each needs its own design. Until then, gobble's client keeps answering -32601 and advertising no elicitation.
  - **ACP auth:** F4-3, amended by D11.
  - **Images and audio:** these wait for `llm` to carry them (0005-MADR, amendment of 2026-10-10, item 2).
  - **Boolean config options** (`session.configOptions.boolean`) and **`session/list` pagination:** gobble offers selects only, and lists everything at once, both of which the schema allows.
  - **SDK requests, to acp-go-sdk2, the owner's repository:** helpers for `available_commands_update`, `current_mode_update`, `config_option_update`, `session_info_update`, `usage_update`, `notice` and the compaction updates, and a fix to `example/agent/main.go:150`'s comment, which names an interface that does not exist.

### Consequences

- Good: gobble moves to the SDK's current tag with no confinement regression. It also gains a check that predates the bump: a relative `cwd` was never refused, and now is.
- Good: an editor's user sees MCP and prompt problems that today only a log file records, and sees compaction as compaction, on any client that advertises the capabilities. That includes gobble's own CLI (D10).
- Good: the wire gains a schema check (D6) that makes a later SDK bump, or a new update, prove its frames conform.
- Good: the records stop sending stable features to `exp/` (D11), and F4-3 is fixed before it is built rather than after a client refuses it.
- Bad: a client that sends a relative `cwd` is now refused. No known client does: gobble's sends an absolute one, and magic-cli-remote's `cwd` handling is **[unverified]**.
- Bad: `acpclient` grows two kinds and two option fields, and `internal/cli`'s line view two renderers. That is more surface in packages that change often.
- Bad: the compaction path has two shapes, text or structured, chosen per connection. D9's tests must cover both.
- Neutral: magic-cli-remote sees no change unless it starts advertising `session.notices` or `session.compaction`. What it advertises today is **[unverified]**.
- Neutral: 0015-REPORT ordered "acp-go-sdk2's next minor" eighth, "when tagged" (F6 of that report). The tag exists, so this record need not wait. It shares two files with F4-3, `acpserver/agent.go` and `internal/cli/agent.go`, where `acpclient.Start` is called (`:122`), and none with F4-1a or F4-2. Its order against them is the owner's call.

### Confirmation

```text
go list -m github.com/maccavelli/acp-go-sdk2                             -> github.com/maccavelli/acp-go-sdk2 v0.15.0
grep -c 'acp-go-sdk2 v0.15.0' go.sum                                     -> 2
go test ./acpserver -run '^TestSession$'                                 -> ok; git diff --exit-code acpserver/testdata/session.golden
go test ./acpserver -run TestCwdMustBeAbsolute                           -> ok: "" and "relative/dir" refused -32602 on new; "" on load/resume takes the stored cwd
go test ./acpclient -run TestElicitationNotFound                         -> ok: -32601, unchanged assertion
go test ./acpclient/acptest -run TestValidateGolden                      -> ok; a broken copy (no sessionId) fails
go test ./acpserver -run 'TestNotice|TestCompactionUpdates'              -> ok: sent only when offered; text path unchanged when not
go test ./acpclient -run 'TestTranslateNotice|TestTranslateCompaction'   -> ok
grep -rn 'SessionUpdate: *"' --include=*.go acpserver                    -> nothing
grep -nE '(=|==) *-32[0-9]{3}' acpclient/conn.go acpclient/sessions.go   -> nothing
make archtest                                                            -> ok
make preflight                                                           -> preflight passed (Windows and WSL)
```

Each new test is seen failing first, on a scratch copy:

- D2's test fails with the check removed: an empty `cwd` is accepted with roots `[]`, as F3's probe showed.
- D6's fails on a broken golden.
- D8's and D9's fail when the gate is inverted.
- D5's proof is the unchanged golden, together with a negative that sets a discriminator to `"wrong"` and still matches it.

## Pros and Cons of the Options

### A — v0.15.0 with the cwd check, plus the stable capability-gated surfaces (chosen)

- Good, because it closes F3 in the same change that would open it.
- Good, because every new behaviour is gated on a capability the schema defines, so no client that does not ask sees a change. The goldens prove that.
- Good, because it uses the SDK's vocabulary (its constants, variants and dispatch), and adds no copy of anything the SDK has.
- Good, because it fixes four records, and one plan before it is built.
- Bad, because it is a larger change than B, across `acpserver`, `acpclient` and `internal/cli`.
- Bad, because D9 gives compaction two output shapes.

### B — v0.15.0, the cwd check and the fixture only

- Good, because it is the smallest safe bump: `go.mod`, `go.sum`, D2's check and one test line. This is the strongest argument for it: the bump and the integration could be reviewed separately.
- Bad, because it leaves F4–F10 standing. The records keep routing stable features to `exp/`, and F4-3 is built against the schema.
- Bad, because the owner asked for the integration, not only the bump. B would need a second record for the same research.

### C — every stable surface now

- Good, because gobble would advertise `session/delete`, elicitation and auth at once, which is the full stable surface.
- Bad, because each needs a product decision this record cannot honestly make:
  - deletion behind a permission request, with a `Store.Delete` that `session.Store` lacks today;
  - a form renderer in the CLI;
  - an `ask_user` design;
  - F4-3's terms checks.
- Bad, because it pulls 0005's 1.x items and F4-3 forward, past the owner's sequencing.

### D — stay on v0.14.0

- Good, because nothing changes, and `v0.14.0`'s `Validate` keeps refusing an empty `cwd`.
- Bad, because it rests gobble's confinement on an SDK check that the SDK has since removed on purpose (SDK 0012-PLAN). The relative-`cwd` defect stays either way.
- Bad, because the SDK owner's fixes, case-folded keys and lenient properties among them, do not reach gobble, and the next bump only grows.

## More Information

### Evidence index

| Claim | Source |
| :--- | :--- |
| `v0.15.0` is `d5a336c`, published, with these sums; `go.mod` sum unchanged | `proxy.golang.org` `@v/v0.15.0.info`; `sum.golang.org/lookup/…@v0.15.0`; scratch clone `go.sum` diff |
| Clone HEAD `bc0cb78` differs from the tag in `docs/` only | `git diff --stat v0.15.0 HEAD` |
| Schema `1.25.0`, protocol 1 | SDK `schema/version`; `constants_gen.go:6` |
| Breaking changes and accept changes | SDK `docs/reports/0013-REPORT-release-notes-v0-15-0.md` |
| `ErrorCode` constants; `RequestError.Code` an `int`; `errors.go` unchanged | SDK `types_gen.go:5189-5199`; `errors.go`; `git show v0.14.0:errors.go \| diff` |
| Only `helpers.go` of the hand-written files changed | `git diff --stat v0.14.0 v0.15.0 -- helpers.go …` |
| Build, vet ×3, govulncheck ok; one test fails, text quoted | `q398_experiments.py acp-0.15.0`; `exp/acp-0.15.0-logs/test.log`, `vuln.log` |
| The fixture fix suffices | `q415_elicitation_fixture.py`; `elicitation-fixture.log` |
| 28 empty-string checks, none kept | `q414_empty_string_checks.py`; `empty-checks.txt` |
| Inbound requests validated, responses not | SDK `agent_gen.go:17`, `:493-496` |
| Empty `cwd` refused on `v0.14.0`, unconfined on `v0.15.0`; relative accepted on both | `q413_probe_cwd.py`; `probe-cwd-v0.14.0.log`, `probe-cwd-v0.15.0.log` |
| No roots confine nothing | `internal/fsx/workspace.go:22`, `:32-34`, `:87-89` |
| Additional dirs checked, `cwd` not; load/resume fallback | `acpserver/agent.go:244-313`, `:706-731`, `:830-839` |
| gobble's CLI sends an absolute `cwd` | `internal/cli/chat.go:76-82` |
| Stable and unstable methods, variants, capability properties; the MUST rules | `q412_schema_stability.py`; `schema-stability.txt`; SDK `schema/schema.json`, `meta.json`, and their `unstable` files |
| Records calling stable surfaces unstable | `0004-MADR-go-module-architecture.md:45-47`; `0005-MADR-v1-feature-scope.md:148`, `:192`, `:614`; `0002-MADR-cli-acp-headless-mcp-v1.md:977`, `:1030` |
| F4-3's ACP auth bullet and files row | `0005-PLAN-v1-feature-scope.md:3306`, `:3314` |
| 53 test lines in 17 files name codes by number | a search of `*_test.go` for `-32[0-9]{3}` |
| gobble's agent and client method use; helpers 21 of 44 | `q411_acp_surface_map.py`; `acp-surface-map.txt`, each row read in its file |
| Stubs answering -32601 | `acpserver/agent.go:887-891`, `:933-941`; `acpclient/conn.go:414-462` |
| The SDK answers -32601 for an unimplemented unstable method | SDK `agent_gen.go:222-226`, `:361-365`; `client_gen.go:71-73` |
| Client advertises nothing; the golden's `initialize` | `acpclient/conn.go:121-124`; `acpserver/testdata/session.golden:1` |
| Agent reads only `fs` and `terminal` | `acpserver/agent.go:226`, `:467-472` |
| Compaction as text, live and replayed | `acpserver/compact.go:24`, `:37`; `acpserver/history.go:301-302`; `acpserver/extensions.go:217-247` |
| `KindOther`; params kept and streamed | `acpclient/update.go:13-16`, `:74-76`, `:125-186`; `internal/cli/print.go:70-85` |
| Diagnostics only logged; no stderr under `gobble acp` | `acpserver/mcp.go:151-168`; `acpserver/sysprompt.go:203-219`; `internal/cli/main.go:169` |
| Hand-built updates; the SDK writes the discriminator; helpers leave it empty | `acpserver/commands.go:330`, `:387`, `:516`; `acpserver/modes.go:50`; `acpserver/prompt.go:304`; SDK `types_gen.go:12901-12923`; `helpers.go:128-131`; `probe-acp015.txt` |
| Magic codes in non-test code; constants equal them | `acpclient/conn.go:31`, `:281`; `acpclient/sessions.go:47`, `:49`; `probe-acp015.txt` |
| Goldens conform; negatives fail | `q416_golden_conformance.py`; `golden-conformance.txt` |
| No `Validate` in gobble; no `ErrorCode` or `Ext*` | search of every `.go` file |
| Connection wiring; the example's undeclared interface | `acpserver/serve.go:32-33`; SDK `example/agent/main.go:150`, `:416-418`; no `ConnAware` outside `example/` |
| Who may import the SDK | `internal/archtest/check.go:186`, `:208-215` |

### Related records

- [0014-MADR-move-to-acp-go-sdk2.md](0014-MADR-move-to-acp-go-sdk2.md) and [0014-PLAN-move-to-acp-go-sdk2.md](0014-PLAN-move-to-acp-go-sdk2.md): the `v0.14.0` move, its wire rule (D5), and its deferrals of delete and elicitation.
- [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md): honest advertisement, and the fork path.
- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): rule 2, and `exp/` for unstable work.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) and [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): the owners of delete, elicitation, `ask_user`, images, and F4-3's auth.
- [0015-REPORT-sibling-library-pins-and-order.md](../reports/0015-REPORT-sibling-library-pins-and-order.md): the bump's test failure (its F3), and the order (its F6).
- acp-go-sdk2 `docs/reports/0013-REPORT-release-notes-v0-15-0.md`, `docs/decisions/0011-MADR-harden-code-generator-engine.md` and `0012-MADR-schema-fidelity-framework.md`: what `v0.15.0` changed, and why.

### Open questions for the plan

None that is a decision. The plan settles, by reading the code:

- where `mcpTools` gets the session id D8's notices need (today it takes only `ctx` and the session's `liveMCP`);
- the exact words of D10's line-view rendering, which must match `/compact`'s current text.

## Amendment (2026-10-15): acp-go-sdk2 `v1.14.0`, and ACP authentication that magic-cli-remote can use

**Why.** 0005-PLAN F4-3c builds ACP authentication on D11 of this record. Grounding F4-3c on 2026-10-15 found two
facts D11 did not have:

- acp-go-sdk2 has tagged `v1.14.0`;
- magic-cli-remote, gobble's first ACP host, cannot use the terminal methods D11 allows, and needs another kind.

The owner decided each point on 2026-10-15. The facts were read at acp-go-sdk2 `v0.17.1` (`c77f25a`) and `v1.14.0`
(`e24efbb`), at magic-cli-remote `c1d29a9d`, at go-llmprovider-sdk `v1.13.0`, and at gobble-cli `main` after F4-3b.

### What was found

**acp-go-sdk2 `v1.14.0`.**

- **The release.**
  - Tagged 2026-10-14 (`e24efbb`), the first v1 release. Its module path is unchanged, with no `/v2`, `go 1.27.2`,
    and no requirements.
  - It retracts an accidental `v1.13.0` (`go.mod`: `retract v1.13.0 // Published accidentally from an unfinished
    commit.`). `v0.18.0` was drafted, never tagged, and renamed `v1.14.0` (SDK
    `docs/reports/0019-REPORT-release-notes-v0-18-0.md:3-8`).
  - The schema is still ACP 1.25.0, and `git diff v0.17.1 v1.14.0 -- schema/ version` is empty.
- **One break reaches gobble.** `SessionSessionInfoUpdate.Title` changes from `*string` (`types_gen.go:12649` at
  `v0.17.1`) to `Patch[string]` (`:12600` at `v1.14.0`). gobble names it at four sites: `acpclient/update.go:181-182`,
  `acpserver/commands.go:420`, `acpserver/commands_test.go:301-302` and `acpserver/extensions_test.go:443`.
- **The rest compiles and passes.** Every other one of the 203 SDK identifiers gobble uses compiles unchanged at
  `v1.14.0`. With those four sites patched on a scratch copy, the tests of `acpserver`, `acpclient`, `internal/cli` and
  `cmd` pass against `v1.14.0`.
- **What `v1.14.0` adds that gobble does by hand** (release notes, lines 18-151):
  - **`AgentConnAware`.** The constructor calls `SetAgentConnection` on a handler that has it (`connection.go:575-585`).
    gobble's method is `SetConnection`, called after the constructor (`acpserver/serve.go:34-35`). The constructor
    starts reading at once, so a request can arrive before `a.conn` is set.
  - **`(*Connection).Close()`** (`connection.go:693`). `Serve` returns on `ctx.Done()` without closing the
    connection (`acpserver/serve.go:39-40`).
  - **`WithPromptOverlapPolicy`** (`connection.go:424`). The default, `PromptOverlapSupersede`, is unchanged from
    `v0.17.1`: the SDK cancels a running turn, then calls `Prompt`. gobble's `claimTurn` refuses a prompt while the
    turn's `cancel` is set (`acpserver/agent.go:552-567`), so a second prompt can cancel the first and then be refused
    -32600 before the first releases. `PromptOverlapConcurrent` passes every prompt to the agent
    (`connection.go:406-417`).
  - **`session/close`** now cancels a running turn before `CloseSession` runs. gobble's `CloseSession` closes the
    session's log without waiting for the turn to unwind (`acpserver/agent.go:964-977`).
- **The auth surface is unchanged,** apart from line numbers:
  - `AuthMethod` has two variants, `Terminal *AuthMethodTerminalInline` and `Agent *AuthMethodAgent`, and "When no
    'type' is present, the method is treated as 'agent'" (`types_gen.go:623-631`).
  - `AuthMethodAgent` has `id`, `name`, `description` and `_meta` (`:750-763`).
  - `AuthMethodTerminalInline` adds `args`, "to append to the configured agent invocation for terminal auth", and
    `env` (`:577-596`).
  - Schema 1.25.0 has no env-var method.
  - The SDK refuses an `initialize` response that lists a terminal method unless the client sent `auth.terminal`
    (`:20012-20019`). It refuses `authenticate` with a terminal id (`ErrTerminalAuthMethod`, `:19969-19972`). It
    answers `logout` with -32601 unless the agent advertises `agentCapabilities.auth.logout` (`:19852-19858`).
  - `NewAuthRequired(data)` is code -32000 with free-form data (`errors.go:216`).

**magic-cli-remote cannot use terminal methods.**

- Its default client capabilities never set `auth` (`internal/provider/acpagent/acpagent.go:1092-1105`), so it sends
  `auth.terminal: false`.
- It keeps only `agent` methods' ids from `initialize` (`acpagent.go:1392-1398`), so a terminal method is dropped
  silently.
- It calls `authenticate` with a method id only (`acpagent.go:557-586`, `authselect.go:24-41`). It obtains credentials
  outside ACP, through its phone flow, which runs the agent CLI's own login (`codex login --with-api-key`, `codex
  login --device-auth`, `grok login --device-auth`) and parses the URL and code it prints
  (magic-cli-remote `docs/decisions/0107-MADR-grok-phone-api-key-and-device-code-auth.md`).
- Its accepted magic-cli-remote `docs/decisions/0186-MADR-integrate-current-library-releases-and-prepare-gobble.md`,
  Q4 (`:495-502`): "gobble advertises an ACP auth method and implements `authenticate` … on `go-llmprovider-sdk`'s
  `TokenStore` and logins … Magic adds no per-provider environment for gobble's keys … gobble is not registered until
  it advertises an auth method."

**Credentials are go-llmprovider-sdk's.**

- The SDK declares auth methods for `grok`, `kilo` and `openai` (`llmprovider/providers/*/descriptor.go`), each with
  `Interactive` and `HeadlessOK` flags. Every other provider has a key alone (`RequiresAPIKey`, `EnvVar`).
- gobble's logins and keys use them since 0005-PLAN F4-3a and F4-3b: `gobble auth login <provider> --api-key -`,
  `gobble auth login grok [--device]` and `gobble auth login kilo`.
- gobble's agent advertises `authMethods: []` today, and answers `authenticate` and `logout` with -32601
  (`acpserver/agent.go:263`, `:1003-1010`). `acpserver/testdata/session.golden` records `"authMethods":[]`.

### The decisions (the owner's, 2026-10-15)

- **A1. Require acp-go-sdk2 `v1.14.0`, and adopt what it now does for gobble.**
  - The four `Title` sites use `Patch[string]`.
  - The agent implements `SetAgentConnection`, and `Serve` drops its own `SetConnection` call.
  - `Serve` closes the connection with `Close()` when its context ends.
  - The agent's connection takes `WithPromptOverlapPolicy(PromptOverlapConcurrent)`. gobble's `claimTurn` stays the
    rule: a second prompt is refused -32600 "a prompt is already running", and the running turn is not cancelled. The
    wire stays as gobble's tests and `acpclient/sessions.go` pin it.
  - `CloseSession` waits for a cancelled turn to release before it closes the session's log.
  - This supersedes 0019-MADR D3's pin, as an amendment of this record (D1).
- **A2. gobble advertises ACP authentication, replacing D11's auth bullets.**
  - **An `agent` method for each provider that takes a credential,** always. Its id is gobble's provider id (`openai`,
    `grok`, `claude`, …), its name the SDK descriptor's label, and its description names the gobble command that
    stores one.
  - **`authenticate`** with such an id answers `{}` when that provider has a credential from any source in gobble's
    order (0005-PLAN F4-3a decision 5). Otherwise it is `auth_required`, its data naming the provider and the command.
    No secret crosses ACP.
  - **A `terminal` method for each such provider, only when the client sent `auth.terminal: true`.** Its id is
    `<provider>-login` and its `args` are `["--login", "<provider>"]`, which the client appends to its `gobble acp`
    invocation. `gobble acp --login <provider>` then runs the provider's login, or asks for its key, in that terminal,
    and exits.
  - **No `logout` over ACP.** `agentCapabilities.auth` stays without `logout`, and signing out stays `gobble auth logout
    <provider>`.
  - **`session.golden` changes,** since its `initialize` response now lists the `agent` methods. This is a wire change
    named here, as 0014-MADR D5 asks. Its client offers no `auth.terminal`, so no terminal method appears.
- **A3. `gobble auth login … --json`** prints the login's events as JSON lines, for magic-cli-remote's phone flow to
  read instead of prose (0005-PLAN F4-3c).

### Consequences

- Good: magic-cli-remote can register gobble. It advertises auth methods, `authenticate` answers, and the phone flow
  drives gobble's own login commands, as it drives Codex's and Grok's.
- Good: a client that can run a terminal, such as an editor, can offer a sign-in that needs no other tool.
- Good: gobble moves to the SDK's first v1 release, with a race at connection start closed and a turn no longer
  cancelled by a refused prompt.
- Bad: `session.golden` changes, and every client now sees one method per provider in `initialize`.
- Bad: `authenticate` only checks that a credential exists. A credential the provider would refuse still answers `{}`,
  and fails at the first prompt with `auth_required`, as today.
- Neutral: D4 to D10 and D12 to D13 stay `proposed`. F4-3c builds A1 to A3 only.
