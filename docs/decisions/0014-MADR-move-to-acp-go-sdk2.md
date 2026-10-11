---
status: accepted
date: 2026-10-10
decision-makers: repository owner
consulted: none
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Move the ACP SDK from the fork to acp-go-sdk2 v0.14.0

## Context and Problem Statement

gobble speaks ACP through `github.com/coder/acp-go-sdk v0.13.5`, replaced by the fork `github.com/maccavelli/acp-go-sdk v0.13.6-mcr.2` (`go.mod:14`, `:43`). The pin and the fork came from [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md) (its "SDK overflow" section) and [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md).

0005-PLAN F3-4 needs a prompt's text blocks to carry their own `_meta`: the CLI marks the stdin and the files it attached so that the agent does not scan them for `@path` (0005-PLAN, F3-4 deviation 1). The fork cannot send that mark, because its content-block marshaller drops every block's `_meta`. The owner directed, on 2026-10-10, that gobble move from the fork to the rewrite they maintain, `github.com/maccavelli/acp-go-sdk2`, at its current tag `v0.14.0`, rather than patching the fork or carrying the marks elsewhere.

This record decides that move, and what it changes in gobble.

### What was measured, not assumed

Read-only, on 2026-10-10. The SDK was read from a shallow clone of the tag in a scratch directory and from the module cache; gobble was compiled against it in a scratch copy of the tree, which the repository never saw.

- **The fork's defect.** `ContentBlock.MarshalJSON` in `acp-go-sdk@v0.13.6-mcr.2` (`types_gen.go:1473-1600`) rebuilds each variant from a fixed key list: a text block from `type` and `text` only, an image from `type`, `data`, `mimeType` and `uri`, and so on. `_meta` and `annotations` are never written, for any of the five variants. A test sending a text block with `_meta.gobble.mentions: false` through `acpclient` showed the agent receiving `{"text":"piped","type":"text"}`.
- **sdk2 keeps them.** `ContentBlock.MarshalJSON` in sdk2 (`types_gen.go:1926-1938`) marshals the whole variant struct and adds `type`; `union_wire_test.go:352-370` asserts `_meta` and `annotations` survive both a Go-built and a decoded block.
- **The module.** `module github.com/maccavelli/acp-go-sdk2`, `go 1.27.2`, no requirements (`go.mod`); package name `acp` (`doc.go:5`). ACP schema `1.25.0` (`version`), protocol version `1` (`constants_gen.go:6`).
- **The tag resolves publicly.** `proxy.golang.org` serves `v0.14.0` at commit `3588f8a3ca1ec0ee72be194eaa06ca32b6558f94`, `refs/tags/v0.14.0`. `sum.golang.org` records `h1:Db0ojAcFSQH5KeEdfxHwJKxJbis+xLgtHMZaqz655Eg=` and `go.mod h1:KqDRvShTbZEOpKOQOyRUnbeFqlyH20LXi7k3UOsgv0I=`.
- **The interfaces.** Read from both sources' interface declarations:
  - `Agent` gains `DeleteSession`; `AgentExperimental` gains `UnstableMessageMcp` and loses `UnstableDeleteSession`;
  - `Client` gains `CreateElicitation` and `CompleteElicitation`; `ClientExperimental` gains `UnstableMessageMcp` and loses the four unstable elicitation and MCP-connection methods.
- **The compile delta.** With every `github.com/coder/acp-go-sdk` import rewritten (42 files) and `go.mod` pointed at the clone, `go build -gcflags=-e ./...` gave 36 error lines and `go test -run '^$' -gcflags=-e ./...` 47, in eleven files:
  - `acpclient/conn.go`, `acpclient/update.go`;
  - `acpclient/acptest/client.go`, `script.go`, `pair.go`, `pair_test.go`;
  - `acpserver/agent.go`, `serve.go`, `clientterm.go`, `prompt.go`, `history.go`.
  The kinds: the four new interface methods; token counts `int` to `uint64`; line numbers and exit codes `int` to `uint32`; `TerminalId` a named string type; raw tool input `encoding/json` v1 `RawMessage` where gobble passes `jsontext.Value`.
- **What gobble relies on from the fork is in sdk2:** the notification overflow policy and its options (`connection.go:144-159`, `:583`); a running prompt cancelled when a new `session/prompt` arrives for its session (`agent_gen.go:427-434`; the fork's `:410-416`); each request on its own goroutine (`connection.go:549`; the fork's `:543`); `ExtensionMethodHandler` (`extensions.go:19`); `SessionUsageUpdate` (`types_gen.go:9215`).
- **sdk2 checks required keys on decode.** Its newest commit generates `UnmarshalJSON` for structs with required fields, which refuses a missing key (`objectKeys`, `types_gen.go:14812-14823`; `NewSessionRequest`, `:6321-6333`). The fork has no such decoder for `NewSessionRequest`, though both have the same `Validate` (`mcpServers is required`; the fork's `types_gen.go:3255-3263`).
- **The gates that name the old path:** `internal/archtest/check.go:186`, `:208` (rules 1 and 2), and 0004-MADR's rule 2 (`0004-MADR-go-module-architecture.md:430`). `apidiff` is a no-op below v1.0.0 (`Makefile:2-3`, `:163`).

### Findings

- **F1.** The fork cannot carry a content block's `_meta` or `annotations` to the wire, so no per-block mark gobble writes reaches the other side, and neither does any annotation.
- **F2.** sdk2 v0.14.0 carries them, is published and checksummed, has no dependencies, and keeps the package name `acp`, so gobble's import alias and identifiers stay.
- **F3.** The move breaks the build in eleven files, all inside `acpserver` and `acpclient`, the two trees 0004-MADR allows to import the SDK. The fixes are mechanical: four interface methods and type conversions.
- **F4.** The compile delta of the test files is a floor, not a count: `acpserver`'s tests could not compile while `acptest` did not, so their own errors were not reached **[unverified]**.
- **F5.** Every fork behaviour gobble's records rely on is present in sdk2.
- **F6.** sdk2 refuses a request missing a required key, which the fork accepted. A client built on the fork emits every required key, since the fork's required fields have no `omitempty`, so gobble's own CLI and magic-cli-remote should be unaffected **[unverified for magic-cli-remote; gobble's own client is checked by the golden in Confirmation]**.
- **F7.** sdk2 is under active development by its owner; `v0.14.0` is its only tag. A later tag is a later decision.
- **F8.** Two gates and one rule of 0004-MADR name the old module path, so they move with it, amended together.
- **F9.** *(Amendment of 2026-10-10, 0014-PLAN P1 deviation 1.)* The schema moved JSON keys in five types gobble uses: `McpServerAcpInline`'s `id` is `serverId`; `ClientCapabilities` loses `planCapabilities` and gains `plan`, `session` and `subagents`; `PromptRequest.messageId` and `PromptResponse.userMessageId` are gone; `ToolCallUpdate` gains `name`, and `Usage` `_meta`. Neither gobble nor magic-cli-remote reads a removed key, and gobble refuses the ACP transport whatever its id. A fork client's old-shape ACP-transport entry now fails at sdk2's decode, still refused. The finding came from a key diff of every struct the two `types_gen.go` files share, run after the compile delta, which F4 had said was a floor.

## Decision Drivers

- F3-4's design, chosen by the owner, needs text blocks with their own `_meta` (F1).
- The owner is replacing the fork with sdk2 and directed the move.
- Wire compatibility with the clients gobble already serves, the CLI and magic-cli-remote, must hold.
- 0004-MADR's boundaries hold: only `acpserver` and `acpclient/...` import the SDK.
- Honest advertisement (0002-MADR): a method gobble must now implement is not advertised unless it works.

## Considered Options

- **A — Require `github.com/maccavelli/acp-go-sdk2 v0.14.0`, and drop the fork.**
- **B — Patch the fork** to marshal `_meta` and `annotations`, tag `v0.13.6-mcr.3`, and bump the replace.
- **C — Keep the fork, and carry the marks at the request level,** `_meta.gobble.unscanned` listing block indices.

## Decision Outcome

Chosen: **A**, as the owner directed. It fixes F1 at its source, on the SDK the owner maintains going forward, and its cost is the bounded, mechanical F3.

### The decisions

- **D1. Require `github.com/maccavelli/acp-go-sdk2 v0.14.0`,** exactly. Remove the `github.com/coder/acp-go-sdk` requirement and its `replace`. `go.sum` holds the sums F2 lists. A later tag is a later amendment of this record.
- **D2. Rewrite every import** of `github.com/coder/acp-go-sdk` to `github.com/maccavelli/acp-go-sdk2`, keeping the `acp` alias. No other package starts importing the SDK.
- **D3. Implement the new interface methods honestly:**
  - `acpserver.Agent.DeleteSession` answers JSON-RPC method-not-found, and `sessionCapabilities.delete` stays unadvertised;
  - `acpclient`'s handler and `acptest.Client` answer the `CreateElicitation` request with method-not-found, ignore the `CompleteElicitation` notification (a notification has no answer), and advertise no elicitation;
  - `acptest.ScriptAgent` and the test agents answer `DeleteSession` the same way.
- **D4. Convert at the boundary, and keep gobble's own types:**
  - `acpclient.Usage` keeps `int` fields;
  - token counts become `uint64`, and line numbers and exit codes `uint32`, as the SDK's fields require;
  - a conversion from a signed value checks its range, as `gosec` G115 asks; a value out of range is never wrapped;
  - `TerminalId` and raw tool input are converted with the SDK's own types.
- **D5. The wire does not change** for any exchange gobble's tests record. `acpserver/testdata/session.golden` stays byte-identical, and `acpclient`'s request still matches the SDK's own client byte for byte. A difference is a deviation for the owner, not a golden to regenerate.
- **D6. The gates move with the path.** `internal/archtest`'s rules 1 and 2 name `github.com/maccavelli/acp-go-sdk2`, and 0004-MADR's rule 2 is amended in the same change. [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md)'s SDK pin gets a dated note pointing here.
- **D7. The docs name the new SDK:** `docs/architecture.md`'s dependency row and `README.md`'s protocols line.
- **D8. 0005-PLAN F3-4 resumes after this record is executed,** with the per-block mark of its deviation 1. This record does not change F3-4's scope.

### Consequences

- Good: a block's `_meta` and `annotations` reach the wire, which F3-4 needs and any later use of annotations will.
- Good: gobble moves to the SDK its owner maintains, off a fork that is now a dead end.
- Good: required keys are checked on decode, so a malformed request is refused at the SDK, with an error naming the type and the key.
- Bad: eleven files change for an SDK move, and a test file or two more **[unverified, F4]**.
- Bad: sdk2 is young and changing; gobble pins a tag and moves deliberately.
- Neutral: magic-cli-remote stays on the fork. The wire between the two stays protocol version 1, and D5 holds gobble to it.

### Confirmation

```text
go list -m all | grep -c 'coder/acp-go-sdk\|maccavelli/acp-go-sdk '     -> 0
go list -m github.com/maccavelli/acp-go-sdk2                            -> github.com/maccavelli/acp-go-sdk2 v0.14.0
grep -rn 'github.com/coder/acp-go-sdk"' --include=*.go .                -> nothing
go test ./acpserver -run '^TestSession$'                                -> ok; session.golden unchanged (git diff --exit-code)
go test ./acpclient -run TestPromptParamsMatchSDKClient                  -> ok
go test ./acpserver -run TestDeleteSessionNotFound                       -> ok: -32601
go test ./acpclient -run TestElicitationNotFound                         -> ok: -32601
make archtest                                                            -> ok; a scratch copy importing sdk2 from internal/cli fails rule 2
make preflight                                                           -> preflight passed (Windows and WSL)
```

## Pros and Cons of the Options

### A — Require acp-go-sdk2 v0.14.0 (chosen)

- Good, because it fixes F1 at the SDK, for every block variant and for annotations as well as `_meta`.
- Good, because it is the SDK the owner is building, so a later fix lands where gobble already is.
- Good, because the fork behaviours gobble relies on are all present (F5).
- Bad, because it changes eleven files and adds four stub methods.
- Bad, because the SDK is moving fast, and its required-key check is a day old (F6).

### B — Patch the fork

- Good, because it is the smallest diff in gobble: a version bump, and no API churn. This is the strongest argument for it.
- Bad, because it extends a fork its owner is replacing, so the same move waits for later, with this patch to carry too.
- Bad, because it is work in another repository, and a release, for a generated file sdk2 has already fixed.

### C — Request-level marks on the fork

- Good, because it needs no dependency change, and works with the SDK as it is.
- Bad, because it moves a per-block fact into a side list keyed by position, which a client reordering blocks would break.
- Bad, because annotations and every other block `_meta` stay lost.
- Bad, because the owner chose against it.

## More Information

### Evidence index

| Claim | Source |
| :--- | :--- |
| gobble pins the fork | `go.mod:14`, `:43` |
| The fork drops block `_meta` and `annotations` | `acp-go-sdk@v0.13.6-mcr.2` `types_gen.go:1473-1600`; acpclient test of 2026-10-10 |
| sdk2 keeps them | sdk2 `types_gen.go:1926-1938`; `union_wire_test.go:352-370` |
| sdk2's module, package, schema and protocol | sdk2 `go.mod`; `doc.go:5`; `version`; `constants_gen.go:6` |
| `v0.14.0` is `3588f8a`, with its sums | `proxy.golang.org` `.info`; `sum.golang.org/lookup` |
| The interface changes | both sources' `type … interface` declarations, read by a script |
| The compile delta | `go build -gcflags=-e ./...` and `go test -run '^$' -gcflags=-e ./...` on a scratch copy |
| Overflow policy, prompt cancel, goroutine per request, extension handler, usage update | sdk2 `connection.go:144-159`, `:583`, `:549`; `agent_gen.go:427-434`; `extensions.go:19`; `types_gen.go:9215` |
| The fork's matching behaviour | the fork's `agent_gen.go:410-416`; `connection.go:543` |
| sdk2 refuses a missing required key; the fork does not | sdk2 `types_gen.go:14812-14823`, `:6321-6333`; the fork has no `NewSessionRequest.UnmarshalJSON`, and its `Validate` at `types_gen.go:3255-3263` |
| The gates naming the old path | `internal/archtest/check.go:186`, `:208`; 0004-MADR `:430` |
| `apidiff` below v1.0.0 | `Makefile:2-3`, `:163` |

### Related records

- [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md): the original pin, and the fork's overflow policy.
- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): rule 2, who imports the SDK.
- [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F3-4 and its deviations 1 and 3.
- magic-cli-remote `docs/decisions/0167-MADR-the-acp-sdk-is-dormant-and-its-bounded-queue-is-an-availability-defect.md` (the fork's queue policy) and `0179-MADR-pigo-native-acp-provider.md`.

### Open questions for the plan

None. The plan budgets for F4's test files, and treats any wire difference as a deviation (D5).

## Observed — execution results (2026-10-10)

0014-PLAN ran on 2026-10-10, approved by the owner, and every decision held as written:

- **D1, D2:** the module and 42 imports moved; `go list -m all` names sdk2 `v0.14.0` and neither old module.
- **D3:** `session/delete` and `elicitation/create` answer -32601, and neither is advertised; `elicitation/complete` is ignored.
- **D4:** every conversion clamps or leaves a value out. One, `ContextUsage`'s `int64` of a `uint64`, compiled and was found by `gosec`, not by the compile delta this record measured.
- **D5:** `session.golden` stayed byte-identical, and `acpclient`'s request matched sdk2's own client.
- **D6, D7:** the gates, the two notes and the docs name sdk2.

F4 was right to call the test-file count a floor. Two more test files changed, one of them for a kind of change this record had not measured, which became F9. 0014-PLAN's execution record has the detail.
