---
status: accepted
date: 2026-11-08
decision-makers: repository owner
consulted: "2026-11-08 refresh: gobble-cli at 4970cca (main), acp-go-sdk2 at tag v1.34.2, magic-cli-remote at c87d22e4 (master), each read from code"
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Take acp-go-sdk2 v0.15.0, close the gap it opens, and adopt the stable ACP surface gobble can honour

*(2026-10-13, [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md) D10: this record is re-based, not duplicated. 0019-PLAN P4 (`17cd497`) pins acp-go-sdk2 `v0.17.1`. D1 is superseded by 0019-MADR D3; D2 by the SDK's own `cwd` check, from `v0.16.0` (0019-MADR F5), and 0019-MADR D7 removed D2's empty-`cwd` fallback; D3 is superseded by 0019-MADR D8; D7's accessors are written over the SDK's `Negotiated()`. D4 to D6 and D8 to D13 stay this record's, `proposed`, and are re-checked against `v0.17.1` before its PLAN is written, since the SDK's gates make an ungated update fail a turn (0019-MADR F6).)*

*(2026-10-15: amended; see "Amendment (2026-10-15)". acp-go-sdk2 `v1.14.0` is pinned and partly adopted (A1), and D11's auth bullets are replaced by A2, decided by the owner. The rest of this record stays `proposed`.)*

*(2026-10-25: D4 and D5 were carried out by [0031-PLAN-sdk-current-releases-and-adoption.md](0031-PLAN-sdk-current-releases-and-adoption.md) P3 on acp-go-sdk2 `v1.20.0`. D13's request for `SessionUpdate` constructors is [acp-go-sdk2#11](https://github.com/maccavelli/acp-go-sdk2/issues/11), shipped in `v1.21.0`; gobble still pins `v1.20.0` until [gobble-cli#5](https://github.com/maccavelli/gobble-cli/issues/5). D8–D10 stay this record's, `proposed`. The `example/agent` comment D13 named is already fixed in the SDK.)*

*(2026-11-08: refreshed against gobble-cli `4970cca`, acp-go-sdk2 `v1.34.2` and magic-cli-remote `c87d22e4`; see
"Refresh (2026-11-08)" at the end. D4, D5, D12 and A1 to A3 are done; D1 to D3 stay superseded; D6 to D11 and D13 are
re-proposed there, each with three options and a recommendation, for the owner's answers. The text above is kept as
written on its dates. Answered the same day: D6 Option 2, the rest Option 1; the record is `accepted`.)*

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
- **D5. The five hand-written discriminator strings are removed** *(2026-10-22: carried out by [0031-PLAN-sdk-current-releases-and-adoption.md](0031-PLAN-sdk-current-releases-and-adoption.md) P3; the goldens unchanged)* (`acpserver/commands.go:330`, `:387`, `:516`; `acpserver/modes.go:50`; `acpserver/prompt.go:304`). The variant's `SessionUpdate` field is left empty, as the SDK's helpers leave it. `session.golden` staying byte-identical is the proof (F8). Helpers for these five variants, and for `notice` and the compaction updates, are a request to acp-go-sdk2, not local wrappers (D13). *(2026-10-25: those helpers ship in acp-go-sdk2 `v1.21.0` (issue 11). gobble does not call them until it pins that tag.)*
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
    *(2026-10-25: the helpers are [acp-go-sdk2#11](https://github.com/maccavelli/acp-go-sdk2/issues/11), in tag `v1.21.0`: `UpdateAvailableCommands`, `UpdateCurrentMode`, `UpdateConfigOptions`, `UpdateSessionTitle`, `UpdateUsage`, `UpdateNotice`, `UpdateCompaction`, `UpdateCompactionSummary` / `Text`, plus `SessionUpdate.Kind()` and `ClientRefusals`. The example comment is already fixed. gobble still pins `v1.20.0`; consuming them is [gobble-cli#5](https://github.com/maccavelli/gobble-cli/issues/5).)*
    *(2026-10-29: carried out by [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md) D7, at acp-go-sdk2 `v1.26.0`; 0034 is superseded.)*

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

## Refresh (2026-11-08): the record against today's three codebases

*Proposed for the owner's answers, from the owner's "refresh 0016 to ensure it is accurate and reflects the current
state of the codebases it is related to … provide 3 options for implementation of each item, include recommendations.
focus on being idiomatic, standards adherent and following best practices." Nothing above is rewritten; this section
supersedes the open decisions' wording where it differs, and its Q1 to Q7 are the owner's to answer.*

**Read at:** gobble-cli `4970cca` (`main`, equal to `origin/main`), acp-go-sdk2 tag `v1.34.2` (`4c94d2e`, read with
`git show v1.34.2:<path>`), and magic-cli-remote `c87d22e4` (`master`). Line numbers below are at those commits.

### Where each decision stands

| Item | Status at `4970cca` | Evidence |
| :--- | :--- | :--- |
| D1 pin `v0.15.0` | superseded (0019-MADR D3, then A1, then 0035) | `go.mod`: acp-go-sdk2 `v1.34.2` |
| D2 `cwd` check | superseded (the SDK checks `cwd`) | 0019-MADR F5, D7 |
| D3 elicitation fixture | superseded (0019-MADR D8) | `acpclient/elicitation_test.go` |
| D4 SDK error constants | **done** | `acpclient/conn.go:317`, `acpclient/sessions.go:55`, `:57`; no code literal in non-test code |
| D5 no hand-written discriminators | **done** (0031-PLAN P3; helpers by 0035-MADR D7) | no `SessionUpdate: "` literal; ten SDK helpers in use, e.g. `acpserver/prompt.go:304` `UpdateUsage` |
| D6 conformance under goldens | **not done** | no `ValidateGolden` call; `acpclient/acptest/recorder.go:140` compares bytes only |
| D7 capability view | **not done** | `acpserver/agent.go:198-201`, `:411` keep `a.client`; read only for `fs`, `terminal` (`:721-726`) and `auth.terminal` (`:418`); no `Negotiated()` call |
| D8 notices | **not done** | the five diagnostics are logged only (R3) |
| D9 compaction updates | **not done**; compaction goes out as `_gobble/compaction` and agent text (R4) | `acpserver/compact.go:22-46`, `autocompact.go:97-117` |
| D10 client side | **not done** | `acpclient/conn.go:40-57` (no option), `:145-148` (no capability); `acpclient/update.go` maps `notice` and both compaction variants to `KindOther` |
| D11 record notes | **not done** | R7 |
| D12 no unstable surface | **done** | no unstable method or capability; `exp/*/doc.go` stubs only; fork and clone are `_gobble/*` (`acpserver/extensions.go:39-40`) |
| D13 owners' surfaces | unchanged, facts moved (R8) | — |
| A1 `v1.14.0` adoption | **done**; its `Close()` bullet superseded by the SDK's `ServeAgent` | `acpserver/agent.go:372-374`; `acpserver/serve.go:45-48`; `CloseSession` waits up to `closeWait` (`agent.go:42`, `:1276`) |
| A2 ACP auth | **done** (0005-PLAN F4-3c) | `acpserver/auth.go:32-69`; `session.golden` lists `grok` and `keyed` |
| A3 `auth login --json` | **done** (0005-PLAN F4-3c) | `internal/cli/auth.go` |

### What changed since 2026-10-12 (grounded facts)

**gobble-cli.**

- **R1. Pins and boundaries.** acp-go-sdk2 `v1.34.2` and `github.com/maccavelli/gobble-cli/acpext v0.1.0`
  (`go.mod`). Only `acpserver/...` and `acpclient/...` import the SDK (archtest rule 2, `internal/archtest/check.go:264-274`).
  `acpext` imports the standard library alone (0042-MADR) and carries `_gobble/compaction`'s `CompactionNotice`
  (`acpext/types.go:84-91`: `sessionId`, `phase` `start`/`end`, `reason` `manual`/`threshold`/`overflow`,
  `tokensBefore`, `error`, `paused` `turn`/`session`). It has no notice type.
- **R2. What the goldens record.** `acpserver/testdata/session.golden:1`'s client sends `"clientCapabilities":{"auth":{},"fs":{}}`,
  so no golden exercises `session.notices` or `session.compaction`. The ACP goldens are that file
  (`TestSession`, `acpserver/agent_test.go:80`, `:127`) and `acpclient/acptest/testdata/initialize.golden`
  (`TestPairInitialize`, `acpclient/acptest/pair_test.go:93`, `:103`). No test runs the SDK's `Validate` on a recorded
  frame.
- **R3. The five diagnostics are logged, and reach no ACP client.**

  | Diagnostic | Site | Record | Once per |
  | :--- | :--- | :--- | :--- |
  | MCP server failed or needs auth | `acpserver/mcp.go:160-161` | Warn `"mcp: server needs attention"` (server, state, first error line) | server per session (`l.reported`, `:159`) |
  | MCP config error | `mcp.go:149` | Warn `"mcp: config error"` | first prompt (`first`, `:138-151`) |
  | MCP exposure ignored | `mcp.go:147` | Warn `"mcp: exposure and toolExposure apply from 0005-PLAN F8; …"` | first prompt |
  | MCP server still connecting | `mcp.go:164` | Warn `"mcp: server still connecting; its tools join a later prompt"` | first prompt |
  | Prompt file not used | `acpserver/sysprompt.go:223` | Warn `"prompt: a file was not used"` (file, reason) | file and message per session (`s.noted`) |

  `mcpTools(ctx, l *liveMCP)` (`mcp.go:133`) has no session id; its caller has the session (`acpserver/agent.go:675`).
  gobble's own CLI runs write Warn records to stderr (`internal/cli/main.go:82`, `:275`); only `gobble acp` turns that
  off (`:84`). So in gobble's CLI these five are already shown; under `gobble acp` no user sees them.
- **R4. Compaction today.** Four paths share `runCompaction` (`acpserver/compact.go:102`), which appends a
  `session.TypeCompaction` entry whose id the tree assigns on append (`compact.go:120`; `session/tree.go:219`):
  - `/compact` (`compact.go:22`): agent text "Compacting the conversation…", `_gobble/compaction` start (manual), the
    end notice (`manualEnd`, `:51`), then agent text "Compacted from N tokens." plus the summary, or "Compaction
    failed: …", then `usage_update`;
  - `_gobble/compact` (`acpserver/extensions.go:178`): the two notices and `usage_update`, no agent text, and its
    `CompactResult`;
  - automatic, at the threshold or on overflow (`acpserver/autocompact.go:62-117`): the two notices (reason
    `threshold` or `overflow`, `paused` on a failure), no agent text, then `usage_update`.
  - **Replay** on `session/load` is agent text, "Compacted from N tokens:" and the summary (`acpserver/history.go:359-360`,
    `:397-403`).
- **R5. gobble's client.** `acpclient.Options` has no capability field (`acpclient/conn.go:40-57`), and `initialize`
  sets no client capability (`:145-148`). `_gobble/compaction` becomes `KindCompaction` (`conn.go:451`,
  `acpclient/update.go:150`); the line view draws it on stderr (`internal/cli/view.go:182-189`, `:227`), skipping the
  manual reason, which `/compact`'s agent text covers; print mode does the same (`internal/cli/print.go:92-104`), never
  in stream-json.

**acp-go-sdk2 `v1.34.2`.**

- **R6. The stable surface, its gate, and the SDK's own helpers.**
  - Schema `1.25.0`, protocol 1 (`schema/version`, `constants_gen.go:6`). `notice`, `compaction_update` and
    `compaction_summary_chunk`, and `ClientSessionCapabilities.notices` and `.compaction`, are in the stable
    `schema/schema.json` (`:3827`, `:3843`, `:3859`; `:4777`, `:4801`).
  - **Types.** `SessionUpdateNotice` (`types_gen.go:15869`): `severity` (`info`, `warning`, `error`, open), a required
    non-empty `title`, `description`. `SessionCompactionUpdate` (`:15943`): `compactionId`, `status` (`in_progress`,
    `completed`, `failed`, `cancelled`, open), and `summary`, `error` and `_meta` with patch semantics.
    `SessionUpdateCompactionSummaryChunk` (`:16015`): `compactionId` and one content block. No tokens field.
  - **The gate is enforced on send.** `SessionUpdate.gates` (`types_gen.go:25695-25716`), called by
    `AgentSideConnection.SessionUpdate` (`agent_gen.go:1002`), returns `*CapabilityError` ("session/update: capability
    session.notices not advertised: variant notice") and writes nothing. A client that receives an ungated update drops
    and logs it (`connection.go:2426-2439`).
  - **Reading what was agreed.** `AgentSideConnection.Negotiated()` (`agent_gen.go:131`) returns a `Negotiation`;
    `Negotiation.AllowsValue(method, params)` (`helpers.go:1069`) runs the variant gates without sending.
  - **Helpers.** `UpdateNotice`, `UpdateCompaction`, `UpdateCompactionSummary`, `UpdateCompactionSummaryText`
    (`helpers.go:260-292`). `PresentNotice`, `PresentCompaction`, `PresentCompactionSummary` (`:998-1060`) send the
    update when the client advertised the capability, and otherwise an agent message ("Compaction <status>" for a
    compaction) or, for a summary chunk, nothing. Their doc says: "To send the notice or nothing, call SessionUpdate
    with UpdateNotice, which fails with a *CapabilityError without the capability."
  - **Client side.** `CompactionTimeline` (`compaction.go:25-118`) folds compaction updates, live or replayed, by id.
  - **Testing.** `acptest.ValidateGolden(tb, records)` (`acptest/golden.go:41`) checks JSON-RPC framing, decodes each
    frame's params and results into the method's type and runs `Validate`; it does not check capability gates, which
    the SDK enforces on send.
  - **Semantics** (SDK `docs/decisions/0039-MADR-five-adopter-additions-on-schema-1-25-0.md:46-64`): notices are live
    and "SHOULD NOT be included in session replay"; for compaction, "the v1 capability gate applies to replay as well
    as live delivery", the first update for an id fixes its timeline position, and a reported compaction must end in a
    terminal status.

**Records.**

- **R7. D11's notes were never written.** 0004-MADR `:45-47` still lists delete and elicitation as unstable; 0005-MADR
  `:148` (`session/delete`), `:192` and `:629` (elicitation; the claim 0016 cited at `:614` has moved); 0002-MADR
  `:977` (`usage_update` "not standard") and `:1030` (`session/delete`). None carries a note.
- **R8. D13's surfaces, today.** `session/delete` is stable and gated on `sessionCapabilities.delete`; gobble does not
  advertise it and `session.Store` has no `Delete` (`session/session.go:425-436`). Elicitation is stable; gobble's
  client refuses it through `acp.ClientRefusals` (`acpclient/conn.go:358-361`), and Magic advertises form elicitation
  (R9). Images: `llm` has `ContentImage`, unwired, and no audio (`llm/llm.go:121-149`). Config options are selects only
  (`acpserver/agent.go:1314`), while Magic advertises boolean options. `session/list` has no cursor
  (`acpserver/agent.go:1171-1198`).

**magic-cli-remote `c87d22e4`.**

- **R9. Magic advertises both capabilities to every ACP agent it hosts,** gobble included as planned.
  `defaultClientCapabilities` (`internal/provider/acpagent/acpagent.go:1356-1370`) sets `session.notices`,
  `session.compaction`, boolean config options and form elicitation; 0179 gives gobble "the default less
  `fs.readTextFile`, `fs.writeTextFile` and `terminal`" (0179-MADR `:377-378`).
- **R10. Magic already renders both updates, under accepted records of its own.**
  - `notice` → `event.TypeAgentNotice` (`internal/provider/acpagent/session.go:2145-2163`) → a notice row on the phone
    (`apps/mobile/lib/features/chat/chat_bubble.dart:615-650`), by magic-cli-remote
    `docs/decisions/0188-MADR-show-agent-notices-on-the-phone.md` (accepted).
  - `compaction_update` and `compaction_summary_chunk` → one fold per `compactionId`
    (`internal/provider/acpagent/compaction.go:28-146`) → `event.TypeCompaction` → one compaction row, by
    magic-cli-remote `docs/decisions/0190-MADR-show-agent-context-compaction-on-the-phone.md` (accepted). opencode
    sends these today (magic-cli-remote 0214-MADR).
  - `_meta` on these updates is declined silently: a Debug record, once per variant
    (`session.go:2197-2218`; `compaction.go:104`, `:127`).
- **R11. Magic's plan for gobble reads `_gobble/compaction`, into the same fold.** 0179-MADR D10 (`:515-516`) maps
  `_gobble/compaction` into `Host.UpdateCompaction` with "one compaction id per start", ids Magic makes up; the host
  path writes the same fold (`internal/provider/acpagent/host.go:255-268`). The fold reconciles by id string only, so a
  standard update and a `_gobble/compaction` notice for one compaction, under two ids, show as two rows. 0179 does not
  mention the standard updates or this record. `_gobble/retry` maps to Magic's own notice (0179-MADR `:508-514`).
- **R12. Magic's capture waits on a gobble release.** 0179-MADR D14 gate 3 (amended 2026-11-08) requires both
  repositories on one acp-go-sdk2 tag when fixtures are captured from an installed gobble release: Magic pins
  `v1.34.2`, every gobble release pins `v1.34.0`, and gobble's `main` pins `v1.34.2`. The first gobble release cut from
  `main` meets it. What gobble sends Magic about compaction should therefore be settled before that release.

### Decision drivers for the refresh

- **Standards first.** Where ACP's stable schema has a surface, gobble uses it rather than a `_gobble/*` equivalent;
  the extension stays for what the schema lacks, or for a client that does not offer the capability.
- **The SDK's vocabulary and gates.** Ask the SDK's gate (`Negotiation.AllowsValue`), use its constructors and its test
  helpers, and copy nothing it ships (AGENTS.md, 0035-MADR).
- **One channel per fact per connection.** A client never receives the same compaction twice through two channels.
- **The wire is the contract.** Without the capabilities, every golden stays byte-identical (0014-MADR D5).
- **Schema semantics.** Notices are live and not replayed; the compaction gate applies to replay; every reported
  compaction ends in a terminal status; ids are agent-owned and unique within a session.
- **Settle what Magic sees before the release Magic captures from** (R12).

### The open items, each with three options

#### D6. Conformance under the goldens

* **Option 1 — Call the SDK's `acptest.ValidateGolden` on gobble's ACP goldens.** A helper in `acpclient/acptest`
  reads a golden's `client→agent` / `agent→client` lines into the SDK's `acptest.Record`s; `TestSession` and
  `TestPairInitialize` call `ValidateGolden` after their byte comparison.
  * Good, because it is the SDK's own validator, used as shipped.
  * Good, because the goldens and gobble's recorder are unchanged.
  * Neutral, because it checks shape and types, not gates; the SDK already refuses an ungated send (R6).
  * Bad, because a small adapter maps gobble's direction labels to the SDK's.
* **Option 2 — Replace gobble's `acpclient/acptest` recorder and golden with the SDK's `acptest.Recorder` and
  `acptest.Golden`, and validate with `ValidateGolden`.**
  * Good, because it removes a gobble copy of what the SDK now ships (0035-MADR's direction).
  * Bad, because both golden files are rewritten into the SDK's format, a larger review than the check itself.
* **Option 3 — Write gobble's own decoder and validator** (D6 as written in 2026-10).
  * Bad, because it copies what the SDK ships.

*Recommended: Option 1.* Option 2 is a later clean-up once Option 1 shows the two formats agree.

#### D7. How the agent knows what the client offered

* **Option 1 — Ask the SDK's gate.** One helper, `offers(conn, sessionID, update) bool`, over
  `conn.Negotiated()` and `Negotiation.AllowsValue(acp.ClientMethodSessionUpdate, …)`.
  * Good, because the answer is by construction the gate the SDK applies on send, for every variant, including later
    ones.
  * Good, because gobble keeps no second copy of what was agreed.
  * Neutral, because `Negotiated()` returns a deep copy per call; it is called once per diagnostic or compaction.
* **Option 2 — Field accessors over `Negotiated()`,** `offersNotices()` and `offersCompaction()` reading
  `Request.ClientCapabilities.Session` (D7 as amended by 0019-MADR D10).
  * Good, because it is explicit and cheap to read.
  * Bad, because it restates the gate's rule in gobble, one accessor per capability.
* **Option 3 — Accessors over gobble's stored `a.client`** (D7 as first written; `acpserver/agent.go:198-201`).
  * Bad, because it keeps a second source of truth beside the SDK's negotiation record.

*Recommended: Option 1.*

#### D8. Notices for the five diagnostics

* **Option 1 — Notice or nothing.** When `offers` says yes, send `acp.UpdateNotice` with D8's severities (failed or
  needs auth, config error, exposure ignored, prompt file not used: `warning`; still connecting: `info`), the log
  message as title and its detail as description. Always log as today; keep each diagnostic's once-per-session rule;
  never replay or store a notice. `mcpTools` takes the session from its caller (`acpserver/agent.go:675`).
  * Good, because it follows the schema's MUST and the SDK's documented idiom for "the notice or nothing".
  * Good, because a client that does not offer notices sees no change, and every golden stays byte-identical.
  * Good, because Magic shows them through its accepted 0188 path, and an ACP editor through its own.
* **Option 2 — `acp.PresentNotice`:** a notice when offered, otherwise an agent message.
  * Good, because no client misses a diagnostic.
  * Bad, because it injects diagnostic text into the conversation of clients that did not ask, and into gobble's CLI,
    which already prints these on stderr (R3).
* **Option 3 — A `_gobble/notice` extension in `acpext`.**
  * Bad, because it duplicates a stable ACP surface and needs gobble-specific code in every client.

*Recommended: Option 1.* `_gobble/retry` is not a diagnostic and stays as it is.

#### D9. Compaction through the standard updates

* **Option 1 — One channel per connection: the standard updates for a client that offered `session.compaction`,
  `_gobble/compaction` and today's text for one that did not.**
  * Live: `compaction_update` `in_progress` once the run is prepared (a refusal sends none), then
    `compaction_summary_chunk` with the summary, then `completed`, or `failed` with the first line of the error, or
    `cancelled`. The id is fresh per run (`Options.NewID`). *(2026-11-08, "P3 re-assessment" below, P3-Q1 and P3-Q2:
    the summary travels in the `completed` update's `summary` field, not a chunk; the id is the compaction entry's,
    reserved before the run, and the same on replay; `Options.NewID` is not used.)*
  * gobble's own facts ride in `_meta.gobble` of the updates (`reason`, `tokensBefore`, `paused`), through the SDK's
    `MetaObject`; Magic ignores `_meta` silently (R10).
  * `/compact` sends neither of its two agent-text lines on such a connection; `_gobble/compact`'s result is
    unchanged.
  * Replay on `session/load`: a `compaction_update` `completed` with the entry's summary, its id the entry's id, for
    such a client; today's agent text otherwise.
  * Good, because any ACP client that offers the capability, Magic and editors alike, sees compaction as compaction,
    with its summary, through its standard path (Magic's accepted 0190).
  * Good, because no client receives one compaction twice.
  * Good, because `acpext` and the extension stay for clients without the capability, so nothing is withdrawn.
  * Bad, because the compaction path has two shapes, chosen per connection, and its tests cover both.
  * Bad, because Magic's 0179 D10 mapping of `_gobble/compaction` then receives nothing from gobble, and 0179 needs an
    amendment in that repository (R11).
* **Option 2 — Both channels always:** the standard updates when offered, and `_gobble/compaction` regardless.
  * Good, because 0179 works unchanged.
  * Bad, because Magic folds both into one fold by id string, and the two carry different ids, so each compaction
    shows twice (R11); every client that offers the capability receives every compaction twice.
* **Option 3 — Keep `_gobble/compaction` only** (withdraw D9).
  * Good, because nothing changes, in gobble or in 0179.
  * Bad, because every standard client sees gobble's compaction as prose or not at all, without the summary, and Magic
    needs gobble-specific code that invents ids.

*Recommended: Option 1*, decided before the gobble release Magic captures from (R12). `PresentCompaction` is not used:
its fallback, "Compaction <status>", would replace gobble's own text and the extension for clients without the
capability.

#### D10. gobble's own client

* **Option 1 — Advertise `session.compaction` only, in line mode, and render it with the SDK's `CompactionTimeline`.**
  `acpclient.Options` gains `Compaction bool`; `translate` folds `compaction_update` and `compaction_summary_chunk`
  into the existing `KindCompaction` view lines; print mode advertises nothing, so stream-json is unchanged.
  * Good, because gobble's own client exercises the standard path it serves, with the SDK's fold.
  * Good, because notices are not advertised: the CLI already prints those diagnostics on stderr (R3), and would
    otherwise show each twice.
  * Bad, because `acpclient` and the view gain a second compaction source, the standard one, beside the extension
    other clients' paths keep.
* **Option 2 — Advertise both, as D10 was written:** notices become warning lines too.
  * Bad, because each diagnostic then shows twice in gobble's CLI, unless the in-process agent's Warn records stop
    reaching stderr, which changes every non-ACP run.
* **Option 3 — No client change:** gobble's client advertises nothing and keeps reading `_gobble/compaction`, which
  D9 Option 1 still sends it.
  * Good, because it is the smallest change.
  * Bad, because gobble never runs the standard path it serves, and its tests cover it only from the agent side.

*Recommended: Option 1.*

#### D11. The records that call stable surfaces unstable

* **Option 1 — Dated notes at the current lines:** 0004-MADR `:45-47`; 0005-MADR `:148`, `:192`, `:629`; 0002-MADR
  `:977`, `:1030`, each citing schema 1.25.0's stable lists (R6).
  * Good, because the repository's rule is to annotate, never rewrite, and a reader meets the correction where the
    claim is.
* **Option 2 — One report of corrections,** with a pointer line in each record.
  * Bad, because the false sentence is still read first, and the reader must follow a link.
* **Option 3 — Leave them as history.**
  * Bad, because plans built on them route stable surfaces to `exp/`.

*Recommended: Option 1.*

#### D13. The stable surfaces that stay with their owners

* **Option 1 — Keep each with its owner, with R8's facts recorded here:** `session/delete` (0005-MADR "Archive and
  delete"), elicitation (0005-MADR "Sampling and elicitation", "Ask the user"), images and audio (0005-MADR amendment
  of 2026-10-10, item 2), boolean config options and `session/list` pagination.
  * Good, because each still needs a product decision (a `Store.Delete` and its permission; a form renderer; image
    input in `llm`), which a protocol record cannot make.
* **Option 2 — A MADR per surface now.**
  * Bad, because it opens five records without the product calls that would decide them.
* **Option 3 — Build them in this record's PLAN.**
  * Bad, because it is Option C of 2026-10, rejected for the same reason.

*Recommended: Option 1.* Magic's advertising form elicitation and boolean options (R9) is noted for those owners.

### Proposed outcome

The recommended set, `proposed` until the owner answers Q1 to Q7: D6 Option 1, D7 Option 1, D8 Option 1, D9 Option
1, D10 Option 1, D11 Option 1, D13 Option 1. Done items are not reopened. *(2026-11-08: answered; the decision is D6
Option 2, and Option 1 for the rest. See the answers below.)*

* **Q1 (D6)** Option 1, 2 or 3. **Q2 (D7)** Option 1, 2 or 3. **Q3 (D8)** Option 1, 2 or 3. **Q4 (D9)** Option 1, 2
  or 3. **Q5 (D10)** Option 1, 2 or 3. **Q6 (D11)** Option 1, 2 or 3. **Q7 (D13)** Option 1, 2 or 3.

*Answered 2026-11-08 by the owner:*

| Question | Answer | Decision |
| :--- | :--- | :--- |
| Q1 (D6) | "Adopt SDK recorder" | **Option 2**: gobble's `acpclient/acptest` recorder and golden give way to the SDK's `acptest.Recorder`, `acptest.Golden` and `acptest.ValidateGolden`; both ACP goldens are rewritten once into the SDK's record format |
| Q2 (D7) | "Ask the SDK gate" | Option 1 |
| Q3 (D8) | "Notice or nothing" | Option 1 |
| Q4 (D9) | "One channel/connection" | Option 1 |
| Q5 (D10) | "Compaction only" | Option 1 |
| Q6 (D11) | "Dated notes inline" | Option 1 |
| Q7 (D13) | "Keep with owners" | Option 1 |

*Found while writing the PLAN for Q1's answer, and kept by it: gobble's `Recorder.Golden` writes every JSON string
that is a path under `Abs("/")` in slash form before comparing, so a golden recorded on macOS matches on Windows
(`acpclient/acptest/recorder.go:165-189`, `hostIndependent`, 0019-MADR D4). The SDK's `acptest.Golden` canonicalizes
each frame's key order only (`acptest/golden.go:91-130`). That normalizer stays in gobble, applied to the records
before the SDK's `Golden`; everything else of the recorder and the golden comparison is the SDK's. The decision is
accepted: D6 Option 2 and D7 to D11 and D13 Option 1.*

*2026-11-08, 0016-PLAN deviation 1, the owner's "Replace, amend 0004": `acpclient/acptest` is a stable package
(0004-MADR's package map), so D6 Option 2 is a pre-v1.0.0 change to a stable API: its recorder names are removed, and
`Conn.Recorder` and `Pair` return the SDK's `*acptest.Recorder`. 0004-MADR carries the dated note.*

### Consequences of the recommended set

* Good, because MCP and prompt problems reach an ACP user, on Magic's phone and in editors, where today only a log has
  them.
* Good, because compaction reaches any standard client as compaction with its summary, through one channel.
* Good, because every golden whose client offers neither capability stays byte-identical, and the goldens gain the
  SDK's conformance check. *(2026-11-08, Q1 answered Option 2: the two ACP golden files are rewritten once, in the PLAN's
  P1, into the SDK's record format; the frames they hold are the same, which P1 checks by comparing the frames before
  and after the rewrite. From then on the rule holds for the new files.)*
* Neutral, because `acpext` is unchanged: `_gobble/compaction` remains for clients without the capability.
* Bad, because magic-cli-remote's 0179 D10 must change in that repository before Magic captures from the release that
  carries D9, or Magic shows no compaction from gobble through its `_gobble/compaction` mapping. Telling Magic is an
  outward action for the owner's word.

### Confirmation of the refresh

* Each item's status above cites the line that shows it; a reviewer can re-read each at `4970cca`, `v1.34.2` and
  `c87d22e4`.
* The PLAN, [0016-PLAN-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-PLAN-acp-go-sdk2-v0.15.0-and-stable-surface.md),
  carries the recommended set as phases, and is amended by any other answer before it runs.

## P3 re-assessment (2026-11-08): how D9 is built

*From the owner's "do better due-dilligence and research for P3. assess the magic-cli-remote codebase that relates to
it. evaluate this codebase as relates to it. re-assess and evaluate the most idiomatic, stable, compatible, flexible,
and extensible way to implement it." D9's choice, Option 1 (one channel per connection), stands. This section decides
how it is built: five questions, P3-Q1 to P3-Q5, each answered by the owner on 2026-11-08 with the recommended
option.* Read at gobble-cli `df5c21e`, acp-go-sdk2 `v1.34.2` and magic-cli-remote `c87d22e4`.

### What the deeper reading found

**magic-cli-remote.**

* **M1. A standard compaction renders with no Magic change.** Traced from code: the SDK delivers
  `compaction_update` → `internal/provider/acpagent/session.go:2164-2168` → the fold (`compaction.go:28-147`) →
  `event.TypeCompaction` → the manager's per-id dedupe (`internal/session/manager.go:1639-1645`) → the phone's
  `_upsertCompaction` (`apps/mobile/lib/data/chat/transcript_reducer.dart:864-901`) → one row
  (`apps/mobile/lib/features/chat/chat_bubble.dart:288-330`): "Conversation compacted", "Compaction failed",
  "Compaction cancelled", and "Compacting the conversation…" for `in_progress` **and for any unknown status**, with the
  error (when failed) and "What the agent kept:" and the summary.
* **M2. Chunks after the last update are never shown.** The fold emits on every update, but on a chunk only when the
  chunk creates the id (`compaction.go:125-134`). A concrete `summary` replaces the accumulated text (`:108-110`).
* **M3. Replay dedupe keys on the event, not the id.** On `session/load`, a replayed compaction event that matches one
  already in the history ring for that agent session is dropped (`manager.go:330-332`, `:431-464`; inference from
  code, no test found). A replayed id that differs from the live one can therefore show a second row, or none.
* **M4. `_meta` is declined silently** (a Debug record once per variant, `session.go:2197-2218`;
  `compaction.go:103-105`, `:126-128`), and processing continues. Nothing in Magic reads compaction facts beyond
  status, summary and error; reading `_meta.gobble` later needs a Magic change and an amendment of magic-cli-remote
  0214-MADR D14.
* **M5. Magic triggers gobble's compaction by passing `/compact` through as prompt text.** magic-cli-remote 0179-MADR
  D9 (`:490-502`) resolves gobble's commands `KindNative` and sets no `Ops.Compact`; no `_gobble/compact` call is
  planned. The daemon's own "Compacting the conversation…" line runs only for an `OpCompact` provider
  (`internal/session/commands.go:479-495`), so it never fires for gobble.
* **M6. What 0179 would amend.** With gobble sending standard updates to Magic, which offers `session.compaction`
  (0179-MADR `:377-378`), 0179's `_gobble/compaction` handler receives nothing: its D10 (`:515-516`), G3
  (`:297-298`), D13 (`:597-598`), D12's fixture and item 5 (`:558`, `:569`), and its PLAN's Goal (`:28-29`), P2 steps
  4, 9 and 10 (`:239`, `:273-275`, `:289`) and P3 (`:325`) would change; magic-cli-remote 0190-MADR `:84-86` and the
  comment at `internal/provider/acpagent/compaction_test.go:13-15` are out of date. That is magic-cli-remote's to
  amend.

**acp-go-sdk2 `v1.34.2`.**

* **S1. The update is an upsert with patch fields.** Schema `CompactionUpdate` (`schema/schema.json:4381`): "The first
  notification fixes the compaction's timeline position. Later updates with the same ID patch that entity in place …
  `summary`, `error`, and `_meta` have patch semantics: omission leaves the stored value unchanged, `null` clears it,
  and a concrete value replaces it." `summary` is "Complete replacement user-displayable summary content" (`:4401`).
  The SDK's `CompactionTimeline` replaces accumulated chunks with a concrete summary (`compaction.go:59-63`).
* **S2. `_meta` is replaced whole, not merged** (schema `:4415`, "Extensible metadata patch"; the timeline's
  `e.Meta = deepCopy(m)`, `compaction.go:70-74`). Each update that carries `_meta.gobble` must carry all of it.
* **S3. Vendor data belongs in a namespaced `_meta`,** never at a type's root: "Implementations MUST NOT add custom
  fields at the root of a specified type" (SDK 0039-MADR `:71-74`); "Implementations MUST NOT make assumptions about
  values at these keys" (schema `_meta`, e.g. `:287`). The SDK reads such an entry with `MetaObject` and
  `DecodeMeta[T]` (`helpers.go:661`, `:672`); gobble's key is already `gobble` (`acpext.MetaKey`).
* **S4. Status is an open set** (`:4351-4375`); `_` values are implementation-specific, and "The Agent MUST report a
  terminal status when a reported compaction ends" (SDK 0039-MADR `:54-65`).
* **S5. The gate applies to replay** (one send path, `agent_gen.go:1002-1007`), and the SDK refuses an ungated
  update with `*CapabilityError` before writing. `Patch` values are made with `acp.PatchValue` and `acp.PatchNull`
  (`types_gen.go:24934`, `:24942`).
* **S6. No guidance chooses a summary field over chunks,** nor fixes a replayed id; the schema asks only for an
  "Agent-owned ID … unique within the session" (`:4385`).

**gobble-cli.**

* **G1. A failed update stops the turn.** `updates.send` keeps the first error and drops every later frame
  (`acpserver/prompt.go:58-63`), and `Prompt` returns that error (`acpserver/agent.go:767-768`). An ungated standard
  update must therefore never be sent: the choice is made before sending.
* **G2. The summary is whole.** `compaction.Compact` collects the model's text and post-processes it
  (`compaction/compaction.go:297-330`, `:495`, `:521-534`); there is no stream to forward.
* **G3. Entry ids exist only after the write.** `Tree.Next` assigns a fresh id to every appended entry
  (`session/tree.go:71-79`), and `runCompaction` writes a baseline before the compaction entry
  (`acpserver/compact.go:125-129`). `Options.NewID` makes session ids only, and the tests' `counterIDs` would shift
  if compaction drew from it (`acpserver/agent_test.go:30-36`).
* **G4. Four paths, one runner, and today's outcome quirks.** `/compact` runs inside a prompt turn, `_gobble/compact`
  claims its own, and the automatic path runs before a model call inside a turn (threshold) or on a refused call
  (overflow); all reach `runCompaction` (`compact.go:102`), and `claimTurn` keeps them from overlapping in a session
  (`agent.go:785-802`). A cancel that lands after the entry is written is reported "cancelled" by `/compact` and
  `_gobble/compact` (`compact.go:32-33`, `:54-55`; `extensions.go:194-195`), though the compaction stands. On a
  refusal, `/compact` sends its start text and both notices before the refusal is known (pinned by
  `acpserver/autocompact_test.go:391-394`).
* **G5. Only the newest compaction replays, first** (`compaction/estimate.go:126-164`; `acpserver/history.go:359-360`),
  and the entry holds `TokensBefore` but not the reason.
* **G6. Tests that bound the change:** `TestAcpextNotificationsAreSent` (`internal/cli/acpext_test.go:41`) requires
  every `acpext.Notifications()` entry to be sent with a literal `notify(acpext.MethodX, …)`;
  `TestExtensionCompact` (`acpserver/extensions_test.go:370`) expects one `usage_update` after a success;
  `TestNotifyFramesUnchanged` (`acpserver/prompt_test.go:468`) pins the `_gobble/compaction` frames. Each uses a client
  that offers no `session.compaction`, so each stays as it is.

### The five questions

#### P3-Q1. The compaction id

* **Option 1 — Reserve the compaction entry's id before the run,** and use it live and on replay. `session.Tree`
  gains a reservation that `Next` honours; an unused reservation is released.
  * Good, because one id names the compaction live, on replay and in the session file, with no format change.
  * Good, because it avoids M3's replay mismatch.
  * Bad, because `session.Tree` gains a small API.
* **Option 2 — A fresh id live, the entry's id on replay** (D9 Option 1 as first written, without `Options.NewID`).
  * Bad, because live and replayed ids differ (M3).
* **Option 3 — Persist a separate compaction id in the entry.**
  * Bad, because it changes the session file for what the entry id already identifies.

*Answered: Option 1.*

#### P3-Q2. Where the summary travels

* **Option 1 — The `summary` field of the terminal `completed` update.**
  * Good, because the summary is whole (G2), the update is an upsert (S1), and one frame carries it, live and on replay.
  * Good, because Magic shows it when the update arrives (M2).
* **Option 2 — One `compaction_summary_chunk`, then `completed`.**
  * Bad, because it is two frames for one fact, and replay must mirror it.
* **Option 3 — Both.**
  * Bad, because the field replaces the chunk (S1), so the chunk is waste, and order-dependent clients could double it.

*Answered: Option 1.*

#### P3-Q3. The contract for gobble's own facts

* **Option 1 — A typed `acpext.CompactionMeta`** (`reason`, `tokensBefore`, `paused`), sent under `_meta.gobble`, added
  to acpext and tagged `acpext/v0.2.0` before the root requires it (0042-MADR: a change to acpext is tagged before the
  root uses it).
  * Good, because the shape is a published, versioned contract any client decodes with `DecodeMeta` (S3).
  * Good, because it is additive: acpext's `v0.1.0` types are unchanged, and `_gobble/compaction` stays.
  * Bad, because it needs an acpext tag, which the owner pushes, before P3's code.
* **Option 2 — An untyped `_meta.gobble` built in `acpserver`.**
  * Bad, because the shape becomes an undocumented contract.
* **Option 3 — No `_meta`.**
  * Bad, because standard clients lose the reason, the token count and the pause.

*Answered: Option 1.* Each update that carries it carries all of it (S2): `in_progress` and `cancelled` `{reason}`,
`completed` `{reason, tokensBefore}`, `failed` `{reason, paused}` when it paused. Replay carries `{tokensBefore}`, since
the entry keeps no reason (G5).

#### P3-Q4. `/compact`'s text, and the outcome of a late cancel

* **Option 1 — The row only.** On a connection that offered `session.compaction`, `/compact` sends no compaction agent
  text; a refusal ("Nothing to compact …", "Already compacted") keeps its one-line reply and sends no compaction update.
  A run whose entry was written reports `completed` even if a cancel came after; one that was not written reports
  `cancelled`.
  * Good, because one channel carries the compaction (D9), and the reported status is what the session holds.
* **Option 2 — A short "Compacted from N tokens." line beside the row.**
  * Bad, because the compaction is reported twice.
* **Option 3 — Today's text in full beside the row.**
  * Bad, because the summary appears twice.

*Answered: Option 1.* The extension path keeps G4's behaviour as today, byte for byte.

#### P3-Q5. How a client learns of a pause

* **Option 1 — `failed`, `_meta.gobble.paused`, and a notice.** The failed update carries the error and the pause, and,
  when the client offered `session.notices`, one `warning` notice says automatic compaction is paused and how it
  resumes (D8's path).
  * Good, because every part is standard, and Magic shows the notice today (its 0188).
* **Option 2 — `failed` and `_meta.gobble.paused` only.**
  * Bad, because no client reads that today (M4), so the pause is invisible.
* **Option 3 — A `_gobble_paused` status, terminal later.**
  * Bad, because Magic shows an unknown status as in progress (M1), and a terminal status is still owed (S4).

*Answered: Option 1.*

### Consequences

* Good, because any ACP client that offers `session.compaction` sees each gobble compaction once, with its summary and
  a stable id, through its standard path; Magic needs no code to render it (M1).
* Good, because gobble's own facts are a typed, versioned contract that a client can read later without guessing.
* Good, because the paths for a client without the capability, and their tests, are unchanged.
* Neutral, because P3 now needs `acpext/v0.2.0` tagged and a `session.Tree` reservation first.
* Bad, because magic-cli-remote's 0179 must be amended (M6) before Magic captures fixtures from a gobble release that
  carries P3; telling Magic is the owner's.

### Confirmation

The PLAN's P3 tests each answer above, each seen failing first; the frames a capable client receives pass
`ValidateGolden`; the extension path's frames stay byte-identical (`TestNotifyFramesUnchanged`).

## Completion (2026-11-08): 0016-PLAN carried out

[0016-PLAN-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-PLAN-acp-go-sdk2-v0.15.0-and-stable-surface.md) is
complete. This section supersedes the refresh's "Where each decision stands" for the items below; that table stays as
it was read at `4970cca`.

| Item | Status | Commit | Evidence |
| :--- | :--- | :--- | :--- |
| D6 conformance under goldens (Option 2) | **done** | `a233e3f` (P1) | `acpserver/agent_test.go` `TestSession` and `acpclient/acptest/pair_test.go` `TestPairInitialize` compare by the SDK's `acptest.Golden` and check by its `ValidateGolden`; both goldens byte-identical to `a233e3f` at P5 |
| D7 the SDK's gate (Option 1) | **done** | `feee808` (P2) | `acpserver/notice.go` `offers`, over `Negotiation.AllowsValue`; `offersCompaction` (`acpserver/compactreport.go`) asks it before any compaction frame |
| D8 notices (Option 1) | **done** | `feee808` (P2) | `acpserver/notices_test.go`: four MCP notices and one prompt-file notice when offered, none otherwise, none replayed |
| D9 compaction updates (Option 1; P3-Q1 to Q5) | **done** | `253e366` (P3a, `acpext/v0.2.0`), `f555311` (P3b), `3d6fb65` (P3c) | `acpserver/compactupdate_test.go` `TestCompactionUpdate*`; `acpext/roundtrip_test.go`; `session/tree_test.go` |
| D10 gobble's client (Option 1) | **done**, with 0016-PLAN deviation 2 | `315cbe4` (P4) | `acpclient/standardcompaction_test.go`, `internal/cli/compactionview_test.go` |
| D11 the dated notes (Option 1) | **done** | `5d482d9` (P0) | the notes in `0002-MADR-cli-acp-headless-mcp-v1.md`, `0004-MADR-go-module-architecture.md` and `0005-MADR-v1-feature-scope.md` |
| D13 the owners' surfaces (Option 1) | kept with their owners | — | no phase touched `session/delete`, elicitation, images and audio, boolean config options or `session/list` pagination |

**Acceptance** (0016-PLAN P5): a scratch run of the built `gobble acp`, driven over its stdio by the SDK's client offering
`session.notices` and `session.compaction`, with a failing MCP server and `/compact`, received one `warning` notice
("mcp: server needs attention"), one `compaction_update` `in_progress` and one `completed` with the summary and
`_meta.gobble` `{"reason":"manual","tokensBefore":160}`, and no `_gobble/compaction` and no `/compact` text; its 24
frames pass `ValidateGolden`. The run is in the PLAN's P5 record.

**Found at acceptance,** facts about the SDK that change no decision here:

* `ValidateGolden` reads `CompactionStatus` as an open set: a planted `completed` row with the status `"done"` passed
  it (acp-go-sdk2 `v1.34.2`, `types_gen.go`, `SessionCompactionUpdate.validate` checks only the summary's blocks). The
  four statuses gobble sends are held by its own tests (`TestCompactionUpdate*`), not by conformance.
* The SDK checks a closed value on send: a planted summary block whose `annotations.audience` was `"robot"` failed the
  `/compact` prompt with `… audience[0]: "robot" is not a Role`, so it never reached the wire. Through the SDK, such a
  frame cannot leave gobble; `ValidateGolden` still catches a frame the SDK did not write, which a corrupted record
  showed (`"jsonrpc" is not "2.0"`).

**Still the owner's:** magic-cli-remote's
`docs/decisions/0179-MADR-gobble-native-acp-provider.md` needs the amendment of R11 and M6 before Magic captures from a
gobble release that carries P3c. A note for it was drafted at P5 and handed to the owner, not posted.
