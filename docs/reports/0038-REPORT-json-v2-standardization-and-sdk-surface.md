---
status: observation
date: 2026-10-31
subject: whether gobble-cli can standardize on encoding/json/v2 with no v1 import, what acp-go-sdk2 already provides in place of gobble's own JSON calls, and an evaluation of gobble-cli#7
examines: "gobble-cli 31e3c1c (with the owner's uncommitted edits to docs/README.md and 0035-MADR); acp-go-sdk2 v1.29.0; magic-cli-remote 7998b02f; the Go 1.27.3 and 1.26.10 toolchains' encoding/json, encoding/json/v2 and encoding/json/jsontext; the Go 1.25, 1.26 and 1.27 release notes; the encoding/json/v2 migration guide; ACP's extensibility page"
---

# REPORT 0038 — Standardizing on encoding/json/v2, and what acp-go-sdk2 already provides

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

This report records findings and recommendations. It decides nothing. The owner asked on 2026-10-31 to evaluate moving gobble-cli
off a dual JSON stack and onto `encoding/json/v2`, noting that "v1 can't be removed entirely" because acp-go-sdk2's API uses
v1's `json.RawMessage`. The owner asked for official documentation, codebase facts, implementation optimizations and standards
adherence, and for as much as possible to come from the SDK rather than gobble's own `Marshal` and `Unmarshal` calls.
[gobble-cli#7](https://github.com/maccavelli/gobble-cli/issues/7) is the plan under evaluation.

Every claim below names its source:

- a file and line at gobble-cli `31e3c1c`;
- an official document, quoted;
- or a probe. Each probe was a scratch Go module at Go 1.27.3 against acp-go-sdk2 `v1.29.0`, or a scratch copy of this
  repository. No file of this repository was changed by them.

## 1. Answer

**gobble-cli can drop every `encoding/json` (v1) import, and acp-go-sdk2 does not need to change for it.**

- **The premise does not hold at Go 1.27.** At Go 1.27.3, `encoding/json` declares `type RawMessage = jsontext.Value`, an alias.
  gobble's `go.mod` says `go 1.27.3`.
  - So wherever the SDK's API says `json.RawMessage`, gobble can write `jsontext.Value` and the types are identical.
  - A probe consumer that never imports `encoding/json` builds against `v1.29.0` and runs. It implements
    `acp.ExtensionMethodHandler` with `params jsontext.Value`, assigns a `func(..., jsontext.Value) ...` to `acp.MethodHandler`,
    and takes `CallExtension`'s result as a `jsontext.Value`.
- **The answer to #7's open question** is that U5's allowlist can start empty.
- **The SDK should not move its API to `jsontext.Value`.** It would gain gobble nothing, it would break the SDK's floor (Go
  1.26.10, where `jsontext` does not exist by default), and it would break the SDK's `GOEXPERIMENT=nojsonv2` gate
  (acp-go-sdk2 `0007-MADR-self-contained-acp-sdk.md`).
- **About half of gobble's v1 calls can go to the SDK rather than to v2.** The SDK's `ExtensionMux`, with `acp.Notify` and
  `acp.Handle`, decodes extension params itself. That removes `retryOf`, `compactionOf` and `personalityOf` on the client, and
  the hand-written dispatch table on the agent. The rest move to v2 with stated options.
- **Five behaviour differences matter**, each measured in section 5. One test would turn flaky, and a byte-level golden would
  change unless two escaping options are set.
- **#7 needs corrections before it becomes a PLAN.**
  - U4 is already done.
  - U2 as written cannot work through the SDK's types, and it conflicts with ACP's extensibility rules.
  - U3's premise, a 1 MiB read cap in magic-cli-remote, is wrong (10 MiB), and most of U3 already exists in the store.

## 2. What the official sources say

**Go 1.27 release notes** ([go.dev/doc/go1.27](https://go.dev/doc/go1.27), August 2026):

- "The `encoding/json/v2` package is a major revision of `encoding/json`." The `encoding/json/jsontext` package "provides
  lower-level syntactic processing of JSON."
- "The v2 package chooses stricter, more interoperable defaults than v1: it rejects invalid UTF-8 in JSON strings and rejects
  duplicate names within a JSON object."
- "The `encoding/json` package is now backed by the v2 implementation. Marshaling and unmarshaling behavior is preserved, but the
  exact text of error messages may differ. … The v1 API will continue to be supported and users are not required to migrate."
- "Marshal performance is broadly at parity with the previous implementation, while unmarshal performance is significantly
  faster."
- "Users who encounter compatibility problems with the new implementation may disable it by setting `GOEXPERIMENT=nojsonv2` at
  build time … This opt-out is expected to be removed in a future release."
- Removed during the experiment: "the `format` tag option …, the `unknown` tag option …, the `DiscardUnknownMembers` marshal
  option …. In addition, the `inline` tag option was renamed to `embed`".

**Go 1.25 release notes** ([go.dev/doc/go1.25](https://go.dev/doc/go1.25)) introduced both packages behind
`GOEXPERIMENT=jsonv2`. **Go 1.26's** ([go.dev/doc/go1.26](https://go.dev/doc/go1.26)) say nothing about them.

**`encoding/json`'s package documentation at Go 1.27.3**, "Migrating to v2" (`go doc encoding/json`):

- "All the behavior of the v1 package is implemented in terms of the v2 package with the appropriate set of options specified
  that preserve the historical behavior of v1."
- "All new usages of "json" in Go should use the v2 package, but the v1 package will forever remain supported."
- It lists 14 behaviour differences. The ones that matter here are listed below; each has an option that restores v1:
  - case-sensitive name matching;
  - `omitempty` redefined over JSON values, with "Existing usages of `omitempty` on a Go bool, number, pointer, or interface value
    should migrate to specifying `omitzero` instead";
  - a nil slice or map marshalled as `[]` or `{}`;
  - "In v1, a Go map is marshaled in a deterministic order. In contrast, in v2 a Go map is marshaled in a non-deterministic
    order";
  - no HTML escaping;
  - invalid UTF-8 and duplicate names rejected;
  - `time.Duration` with "no default representation".

**The migration guide** ([go.dev/doc/jsonv2-migration](https://go.dev/doc/jsonv2-migration)):

- Migration is optional; "migration to v2 must involve careful testing to ensure compatibility."
- It describes three approaches: all at once, option by option starting from `jsonv1.DefaultOptionsV1()`, and the `jsonsplit`
  wrapper for production servers.
- "the new struct tags from Go 1.27 are also supported by `encoding/json`". The probe in section 5 confirms that for `embed`.

**The dependency between the two:** `DefaultOptionsV1` and the other v1-compatibility options live in package `encoding/json`
(v1). Any call that needs exact v1 semantics through v2 therefore imports v1. An empty allowlist means that no call in gobble
needs exact v1 semantics. Section 4 shows that none does.

**ACP's extensibility page** ([agentclientprotocol.com/protocol/extensibility](https://agentclientprotocol.com/protocol/extensibility)):

- "Implementations can add custom behavior without breaking peers that don't know about it."
- "Implementations MUST NOT add any custom fields at the root of a type that's part of the specification."
- "All possible names are reserved for future protocol versions."
- Of custom notifications: "implementations **SHOULD** ignore unrecognized notifications."

## 3. The SDK boundary

**The alias, measured** with `go list` at each toolchain:

| Toolchain | `GOEXPERIMENT` | `encoding/json`'s file | `json.RawMessage` | `encoding/json/jsontext` |
| :--- | :--- | :--- | :--- | :--- |
| go1.27.3 | unset (the default) | `v2_stream.go` | `= jsontext.Value` | available |
| go1.27.3 | `nojsonv2` | `stream.go` | `[]byte`, its own type | not available |
| go1.26.10 | unset | `stream.go` | `[]byte`, its own type | not available |

At go1.27.3 the toolchain's `internal/buildcfg/exp.go` has `JSONv2: true`.

**What the SDK's exported API names in v1**, at `v1.29.0`, all `json.RawMessage`:

- `MethodHandler`;
- `ExtensionMethodHandler.HandleExtensionMethod`'s `params`;
- `CallExtension`'s result, on both sides;
- `WithStartRawInput` and `WithStartRawOutput`, `WithUpdateRawInput` and `WithUpdateRawOutput`, and `StartEditToolCall`'s
  content;
- the `RawInput` and `RawOutput` fields of `ToolCallTimeline`'s entries;
- `acptest.Recorder.Frames`;
- the generated `ExtRequest`, `ExtNotification` and `ExtResponse`, which are aliases of `json.RawMessage`;
- the generated `Extra map[string]json.RawMessage` fields.

At go1.27.3 every one of them is a `jsontext.Value`. No exported API of the SDK names any other v1 type: no `json.Decoder`,
`json.Number` or `json.Marshaler`.

**How the SDK encodes.** The SDK encodes every message with `encoding/json`, which at go1.27.3 is v2 underneath with v1's
options. Its generated types decode themselves: 317 of 322 have an `UnmarshalJSON` method, and 72 have a `MarshalJSON` method.
So the SDK's own decoding, the hot path for every `session/update`, already gets v2's faster unmarshal at go1.27.3, whatever
gobble's own code imports.

**What the SDK provides that gobble writes by hand:**

| SDK surface | Since | What it does | gobble's own code it can replace |
| :--- | :--- | :--- | :--- |
| `ExtensionMux` | `v1.23.0` (acp-go-sdk2 `0039-MADR-five-adopter-additions-on-schema-1-25-0.md`, item 4) | routes `_` methods by name; answers an unregistered request `-32601`; ignores an unregistered notification, as the specification says; tells requests from notifications | the `switch` in `acpclient/conn.go:415`; the `extensions` map and lookup in `acpserver/extensions.go:28-71` |
| `acp.Notify[P]` | the same | decodes a notification's params into `P`, `{}` when absent; logs a decode failure | `retryOf`, `compactionOf` and `personalityOf` (`acpclient/update.go:125-195`), each a v1 `json.Unmarshal` into a local struct |
| `acp.Handle[P, R]` | the same | decodes a request's params into `P`; answers a decode failure `-32602` with `{"error": …}` | `extSession`'s decode (`acpserver/extensions.go:122-127`) |
| `ExtensionMux.Methods()` | the same | the registered names, sorted | `extensionNames()` (`acpserver/extensions.go:57`) |
| the `acptest` package | `v1.23.0` to `v1.28.0` | `Connect`, `ConnectRecorded`, `Recorder`, `ScriptAgent`, `MemoryClient`, `ValidateGolden`, replay | gobble's own `acpclient/acptest`: 1,236 lines, 60 importing test files |

gobble's [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](../decisions/0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md)
listed `ExtensionMux` as "not a replacement for anything gobble has", because "gobble's `_gobble/*` handler is one switch". Its
D18 left `ExtensionMux` and the SDK's `acptest` for later records, because "Each gives gobble something new, rather than removing
a copy". The goal of a single marshaler changes that for `ExtensionMux`. With it, the three decoders and their v1 import go: the
SDK decodes, and gobble's code calls no JSON function there at all. For `acptest`, D18's reasoning still holds; see section 6.

**What the SDK does not provide, and gobble does not need.** There is no way to get a notification's raw inbound bytes:
`WithInboundRewrite` and `WithInboundValidation` (`connection.go:666`, `:699`) receive the decoded `params any`. `translate`
does not need them (section 7, item 2).

## 4. gobble-cli's JSON today

**Counts at `31e3c1c`**, by a scan of each tracked Go file's imports:

| | Files | Of them, non-test |
| :--- | ---: | ---: |
| import `encoding/json/v2` | 87 | 43 |
| import `encoding/json/jsontext` | 59 | — |
| import `encoding/json` (v1) | 20 | 10 |
| import both v1 and v2 | 2 | 2: `acpserver/extensions.go`, `acpserver/grants.go` |

**A trial of the guard.** A `depguard` rule, `- pkg: encoding/json$`, was added beside the existing `logging` rule in a scratch
copy of `.golangci.yml`, and `golangci-lint` 2.14.0 was run with only `depguard` enabled. It reported exactly these 20 files,
10 non-test and 10 test, and nothing in `encoding/json/v2` or `encoding/json/jsontext`.

**Every non-test v1 use**, with what replaces it:

| Site | v1 use | Replacement | Behaviour risk |
| :--- | :--- | :--- | :--- |
| `acpclient/conn.go:415` | `params json.RawMessage` in `HandleExtensionMethod` | `ExtensionMux` with `acp.Notify` for the three notifications (section 3), or `jsontext.Value` in the signature | none: the same type |
| `acpclient/update.go:105` | `Update.Params json.RawMessage` | `jsontext.Value` | none |
| `acpclient/update.go:125-195` | `retryOf`, `compactionOf`, `personalityOf`: `json.Unmarshal` of `_gobble/*` params | `acp.Notify[P]` handlers. The SDK decodes, as it decodes every other ACP payload gobble receives | the SDK's decoding is v1's: case-insensitive names, duplicates allowed. The params are gobble's own, sent in process |
| `acpclient/update.go:244` | `translate`: `json.Marshal(n)` of every `acp.SessionNotification` | v2 `json.Marshal(n, jsontext.EscapeForHTML(true), jsontext.EscapeForJS(true))`, and only when `Params` is read (section 7) | v2's default output differs from the SDK's (section 5, D1). With the two options it matched on every sample |
| `acpclient/acptest/client.go:70`, `:77` | `json.RawMessage` field and signature | `jsontext.Value` | none |
| `acpclient/acptest/script.go:158`, `:185` | `json.Unmarshal` of a script's update and usage into SDK types | v2 `json.Unmarshal`; the SDK types' own `UnmarshalJSON` decides | v2 rejects duplicate names and invalid UTF-8 before calling the method; test scripts have neither |
| `acpclient/acptest/script.go:203` | `json.Marshal(p)` of an `acp.PromptRequest` | v2 `json.Marshal` | none for `PromptRequest`: the probe found v1 and v2 equal |
| `acpserver/extensions.go:64`, `:77` | `stdjson.RawMessage` in the signature and the result | `jsontext.Value`, or `ExtensionMux` with `acp.Handle` (section 6) | none |
| `acpserver/grants.go:80` | `stdjson.RawMessage` for a schema default | `jsontext.Value` | none |
| `acpserver/prompt.go:199`, `:277` | `rawInput` and `jsonRaw` return `stdjson.RawMessage` for `WithStartRawInput` | return `jsontext.Value`; the SDK option takes it as is | none |
| `internal/archtest/check.go:68` | `json.NewDecoder` over `go list -json`'s stream | `jsontext.NewDecoder` and a `json.UnmarshalDecode` loop to `io.EOF` | `go list` writes exact field names; `modPkg`'s untagged fields match them case-sensitively |
| `internal/cli/choice.go:128`, `:152`, `:211`, `:224` | `recent-models.json` and the kept listing, written and read with v1 | v2, with `json.Deterministic(true)` and `jsontext.WithIndent("  ")` as `internal/cli/config.go:194` does | old files load: the fields are tagged, `time.Time` is RFC 3339 either way, `null` decodes to a nil slice. New files write `[]` for a nil slice where v1 wrote `null` |
| `internal/cli/providerscmd.go:51` | `json.MarshalIndent(rows, "", "  ")`, `gobble providers --json` | v2, with `json.Deterministic(true)` and `jsontext.WithIndent("  ")` | user-visible output: no HTML escaping. `providerRow` (`internal/cli/providerscmd.go`) has only string fields, and `rows` comes from `make`, so it is never nil |

**The test sites** are mostly `json.RawMessage` literals, which become `jsontext.Value`, and decodes into `map[string]any`. One
needs care. `acpserver/autocompact_test.go:198-201` compares two `map[string]any` by their v1-encoded bytes. That works only
because v1 sorts map keys. Under v2 it needs `json.Deterministic(true)`, or better, `reflect.DeepEqual`. Moved naively it would
pass or fail by chance. `acpserver/convert_test.go:17` checks an SDK value "on the wire" with v1. The value's bytes are the
same under v2 for its input, but the test means the SDK's encoding.

## 5. Behaviour differences, measured

Each was measured at go1.27.3 against `v1.29.0`.

| # | Difference | Measured | Where it reaches gobble |
| :--- | :--- | :--- | :--- |
| D1 | HTML escaping | v1 wrote `"<b>a & b</b>"`; v2 wrote `"<b>a & b</b>"`, for a text chunk and for a tool call's title | `translate`'s `Params`, that is, `stream-json` output, pinned by `internal/cli/testdata/print-stream-json.golden` (`internal/cli/print_test.go:118`) |
| D2 | `omitempty` on a `bool` | `acp.ClientCapabilities{}`: v1 `{"auth":{},"fs":{}}`; v2 `{"auth":{"terminal":false},"fs":{"readTextFile":false,"writeTextFile":false},"terminal":false}`. The SDK has 11 such fields, all on capability types, none with a `MarshalJSON` | any capability value gobble encodes itself; none was found |
| D3 | map order | v1 sorted; v2 random unless `json.Deterministic(true)` | `autocompact_test.go:198-201`; the kept listing and the providers output hold no map |
| D4 | nil slice | v1 `null`; v2 `[]` | `recent-models.json` and the kept listing |
| D5 | U+2028 and U+2029 | v1 escapes them; v2 with `EscapeForHTML` alone does not | `translate` |

**Escaping options against the SDK's goldens.** All 18 `session_update_*.json` goldens of acp-go-sdk2's `testdata/json_golden`,
plus three synthetic chunks, were decoded, wrapped in an `acp.SessionNotification`, and encoded with v1 and with v2:

| Encoder | Samples differing from v1 |
| :--- | ---: |
| v2, no options | 2 of 21 |
| v2 + `jsontext.EscapeForHTML(true)` | 1 of 21, the U+2028 chunk |
| v2 + `jsontext.EscapeForHTML(true)` + `jsontext.EscapeForJS(true)` | 0 of 21 |

The three synthetic chunks had `<b>a & b</b>`, U+2028 and U+2029, and plain text.

**Not a difference.**

- The `embed` fallback round-trips the same under v1 and v2 at go1.27.3. Members a struct does not know land in its
  `jsontext.Value` field tagged `json:",embed"`, and both write them back at the top level.
- `time.Duration` reaches no gobble JSON. `_gobble/retry` carries `delayMs` as an `int64` (`acpclient/update.go:132`).
- No gobble type uses the removed `unknown` or `format` tag options. `session/doc.go:8` records the move to `embed`.

**Stale, not broken.** `acpserver/extensions.go:88-118` (seven types) and `acpserver/personality.go:276` tag an embedded
`sessionParams` with `json:",inline"`. Go 1.27 renamed that option to `embed`. The probe found that v2 accepts the unknown
option without error, and that the fields decode and encode correctly anyway. A Go-embedded struct "is implicitly JSON embedded
unless an explicit JSON name is specified" (`go doc encoding/json/v2`). So the tag does nothing. Dropping it, or writing
`embed` as `session/session.go:90` does, keeps the code honest.

## 6. Adopting the SDK's surface

**The client's three notifications.** `acpclient/conn.go:415-450` is a three-way switch whose only work, besides routing to the
session's callback, is decoding.

- `ExtensionMux` with `acp.Notify` registers each one with a typed struct.
- The SDK decodes, and an unregistered notification is ignored, as ACP says one SHOULD be. Today it is answered
  `-32601`, which the SDK drops for a notification.
- `Update.Params` for these three kinds is never read: `print.go` returns before streaming retry, compaction and personality
  updates.
- The routing to `h.sessions` and `h.pending` stays in the handlers.
- `retryOf`, `compactionOf`, `personalityOf`, and `update.go`'s v1 import go.

**The agent's twenty methods.** `acpserver/extensions.go` is a map and a generic decoder, `extSession`. `acp.Handle[P, R]`
would replace the map, the lookup, the `-32601` and `extSession`'s decode, and `Methods()` would replace `extensionNames()`.
Three things to decide:

1. **Results.** Today each result is encoded with v2 and handed to the SDK as raw JSON (`extensions.go:72-77`). Returning a Go
   value instead would let the SDK encode it with v1 semantics. Nil slices would then go out as `null` where Pi's RPC shape and
   today's wire have `[]` (D4), and text would be HTML-escaped (D1). Keeping `R` as `jsontext.Value`, encoded with v2 as now,
   keeps the wire unchanged. The `embed` fallback is not a reason either way at go1.27.3 (section 5).
2. **Error data.** `acp.Handle` answers a decode failure `-32602` with `{"error": …}`. gobble answers `{"reason": …}`
   (`acpserver/agent.go:45`, `keyReason`). The "unknown session" answer stays gobble's. A client that reads `reason` on a
   decode failure would see a different key.
3. **Strictness.** The SDK's decode is lenient, as v1 is. These are gobble's own params. See U2 in section 8.

**The SDK's `acptest`** is not part of this change. gobble's own `acpclient/acptest` is already on v2: its transcripts are RFC
8785 canonical JSON through `jsontext.Value.Canonicalize`. Its only v1 uses are the three in `script.go` and the two in
`client.go`, all small. The SDK's `acptest` "is on encoding/json v1" (`go doc .../acptest`). Moving 60 test files to it
replaces a v2 harness with a v1 one, and rewrites gobble's goldens once, as acp-go-sdk2's
`0042-MADR-in-process-agent-runner.md` noted. That is 0035-MADR D18's "something new", for its own record.

## 7. Implementation optimizations

1. **`translate` encodes every notification eagerly** (`acpclient/update.go:244`), and its result, `Update.Params`, is read in
   one place: `stream-json` output (`internal/cli/print.go:113`).
   - Measured with `testing.Benchmark` on a text chunk, darwin/arm64: v1 1,198 ns, 1,024 B and 9 allocations per call; v2
     1,288 ns, 1,056 B and 10 allocations.
   - The line mode, the TUI and `task` pay this for nothing on every update.
   - Encoding only when the sink asks for it, a flag on `acpclient.Options` or a func field, removes it. v2 does not help: marshal
     is "broadly at parity", as the release notes say and the benchmark shows.
2. **Re-encoding is the contract, not a gap.**
   - `stream-json` promises "the session/update params object as the SDK encodes it" (`update.go:103-104`, 0008-MADR D9), not
     the bytes the agent sent. Re-encoding the decoded value is exactly that.
   - gobble's agent is gobble, in process, encoding with the same SDK. So the re-encoding gives the bytes the agent wrote, apart
     from members the SDK's types do not keep.
   - The cost is item 1's, and it is removed in the consumer by encoding only when `stream-json` asks. Nothing is needed from the
     SDK.
   - Should the agent's exact bytes ever be wanted, they are within gobble's reach. gobble supplies the connection's reader, and
     `agentproc.InProcess` has an `OutputTap`; the work would be matching each tapped frame to its handler call. An SDK accessor
     would instead keep every queued notification's bytes alive, likely through a request context, for one consumer's
     convenience.
3. **Extension results are processed twice.** v2 encodes a result, then the SDK's v1 marshal of the returned `RawMessage`
   validates, compacts and HTML-escapes it again. For `_gobble/get_messages` and `_gobble/get_entries`, which return whole
   sessions, that is a second linear pass over a large value. It is correct, and only worth measuring if those calls show up
   in a profile.
4. **Reading a session file.** `session/jsonl/store.go:172-216` reads the file whole and splits it with `bytes.Split`, which
   builds a `[][]byte` of every line first. `bytes.Lines` (Go 1.24) iterates without that slice and keeps the per-line
   skip-and-warn semantics. A streaming `jsontext.Decoder` would not: one malformed line ends the stream, and Pi's format needs
   malformed lines skipped.

## 8. gobble-cli#7, unit by unit

**Its premises, at `31e3c1c`:**

| #7 says | Measured |
| :--- | :--- |
| 87 files import v2 | 87, 43 non-test: correct |
| 20 files import v1, including `session/{session.go,migrate.go}` | 20, correct; but the `session` package imports only v2, and neither file was flagged by the `depguard` trial |
| "v1 defaults on HTML escaping / unknown-member tolerance / case folding differ from v2's" | escaping and case folding differ. Unknown members do not: v2 also ignores them "by default", and rejects them only with `RejectUnknownMembers` (`go doc encoding/json/v2`) |
| "`session/jsonl` written by one encoder, portions of `session/` read by another" | no longer true: the store and the `session` package are both v2 |
| requires `acp-go-sdk2 v1.26.0` (`go.mod:16`); magic-cli-remote on `v1.28.0` | gobble pinned `v1.28.0` in `6f7787f` (2026-10-31, after the issue was opened). `v1.29.0` is current; [gobble-cli#8](https://github.com/maccavelli/gobble-cli/issues/8) proposes it |
| "Hot-path wins on decode/allocations for delta streaming" | the decode hot path is the SDK's, and the SDK's `encoding/json` is already v2 underneath at go1.27.3. gobble's own calls are not on it, except `translate`'s encode, where v2 is at parity |
| "mcremote's consuming side caps engine frames at 1 MiB on read" | magic-cli-remote's ACP cap is `maxACPMessageSize = 10 << 20` (`internal/provider/acpagent/acpagent.go:1144`), the SDK's own default (`defaultMaxMessageSize`, 10 MiB). Its 1 MiB figures are its relay's and its WebSocket frames' |
| "verified compiling … with no `GOEXPERIMENT` in `go.mod`" | a `GOEXPERIMENT` is not a `go.mod` setting. At go1.27.3 v2 is on by default. gobble cannot build with `GOEXPERIMENT=nojsonv2` today, since 87 files import v2, and that opt-out "is expected to be removed in a future release" |

**U1, the single-marshaler sweep: sound, smaller than it reads.**

- Of the 10 non-test files, 6 only spell the SDK's type, a rename to `jsontext.Value`.
- Two move to the SDK's `ExtensionMux` (section 6). The other two, `choice.go` and `providerscmd.go`, move to v2 with
  `Deterministic` and `WithIndent`, plus `translate` with the two escaping options. `archtest` moves to `jsontext.Decoder`.
- U1's fixture test, "a fixture written by the old path still loads identically", applies to `recent-models.json` and the kept
  listing, the only files v1 writes. The session store is already v2.

**U2, strict decode at the ACP boundary: not as written.**

- gobble does not decode ACP's own methods; the SDK does. Every generated type decodes itself, so
  `json.RejectUnknownMembers(true)` does not reach inside it. The probe decoded a `session/update` carrying two unknown members
  into `acp.SessionNotification` with the option set, and got no error. A plain struct with the option did reject.
- Rejecting would also run against ACP's extensibility: "All possible names are reserved for future protocol versions". The SDK's
  generated types keep unknown members, in 7 `Extra` and 11 `Unknown` fields, so they survive a round trip.
- What remains is gobble's own `_gobble/*` params. Strictness there is gobble's choice. The cost is skew: a newer client
  sending a field an older agent does not know would be refused `-32602`. magic-cli-remote's non-test code sends no `_gobble/`
  method today. Leniency is the ACP-consistent default.

**U3, store hardening: mostly present.**

- `session/jsonl/store.go` already appends one line per `Write` with `O_APPEND` under a writer lock (`:303-325`).
- It detects a torn last line and puts the next entry on a line of its own (`:183`, `:320-322`).
- It skips malformed lines with one warning per file (`:180-181`, `:215-216`).
- "Flush-then-rename" does not fit an append log: renaming rewrites the file.
- What is missing: an fsync policy, since nothing is synced today, and a cap on an entry's size. Each is a durability decision
  of its own, outside a marshaler change.

**U4, SDK lockstep: done.** Both repositories pin `v1.28.0`. The next step, `v1.29.0`, is gobble-cli#8's.

**U5, the guard: yes, with an empty allowlist, through `depguard`.**

- `.golangci.yml` already bans packages this way: `log`, logrus, zap and zerolog, "use log/slog (0008-MADR D18)". A deny of
  `encoding/json$` beside them is the same idiom, reports in the editor and covers tests.
- The scratch trial reported exactly the 20 files.
- `internal/archtest` also bans an import, go-tui-lib's v1 path. But its rules are about module edges, and it parses `go list`,
  so it would itself have to move off v1 first.

## 9. Recommended order

Each step ends with `make preflight` green, and each new check seen to fail on a planted break first.

1. **The renames.** Replace `json.RawMessage` and `stdjson.RawMessage` with `jsontext.Value` in the 6 files and their tests.
   There is no behaviour change; the types are identical.
2. **The client's notifications through `ExtensionMux` and `acp.Notify`**, removing `retryOf`, `compactionOf` and
   `personalityOf`. The `retry`, `compaction` and `personality` tests keep their cases. The bad-params case becomes "logged and
   ignored", as for any notification.
3. **`translate`.** Encode only when `stream-json` asks for it. Use v2 with `EscapeForHTML(true)` and `EscapeForJS(true)`. The
   `print-stream-json` golden must not change; seen to fail, by dropping one option, on a chunk with `<` or U+2028.
4. **The local files and the CLI output.** `choice.go` and `providerscmd.go` on v2 with `Deterministic` and `WithIndent`.
   - A fixture of each file as v1 wrote it, nil slices as `null` included, must load.
   - `providers --json` gets a golden, or the end of its HTML escaping is accepted on purpose.
5. **The tests.** `autocompact_test.go`'s comparison by `reflect.DeepEqual`, seen to fail by changing one value. The decodes into
   `map[string]any` move to v2.
6. **`archtest`** on `jsontext.Decoder`.
7. **The guard.** `depguard`'s `encoding/json$` deny, seen to fail on a planted v1 import.
8. **Optional, each its own decision:**
   - the agent's twenty methods through `acp.Handle` with `R` as `jsontext.Value`, accepting `error` for `reason` in decode
     failures;
   - the `inline` tags dropped;
   - `bytes.Lines` in the store.

Not in this change: U2's strictness, U3's fsync and size cap, and the SDK's `acptest`.

## 10. What acp-go-sdk2 could add

None of this blocks gobble.

- **A sentence in its README:** at Go 1.27, with the default build, `json.RawMessage` is `jsontext.Value`, so a consumer on v2
  never needs to import `encoding/json` to use the SDK.
- **Its API stays on `json.RawMessage`** while its floor is below Go 1.27 and it keeps the `nojsonv2` gate. When the opt-out is
  removed and its floor reaches 1.27, the spelling stops mattering.

## How it was read

- **gobble-cli and magic-cli-remote:** scratch scripts listed each tracked Go file's JSON imports and every use of the v1 import
  with its function. The table rows were read from source.
- **The `depguard` trial:** ran in a scratch copy of gobble-cli, with one rule added to `.golangci.yml`.
- **The probes:** scratch modules at go1.27.3 requiring acp-go-sdk2 `v1.29.0`:
  - a consumer naming only `jsontext.Value`, built by default and with `GOEXPERIMENT=nojsonv2`, where it fails because
    `jsontext` does not exist;
  - the `inline`, `embed` and untagged embeddings, decoded and encoded;
  - v1 against v2 on SDK values;
  - `RejectUnknownMembers` into an SDK type and into a plain struct;
  - the `embed` fallback under v1 and v2;
  - the escaping options over the SDK's goldens;
  - `testing.Benchmark` of the two encoders.
- **The official texts:** the package documentation is from `go doc` at go1.27.3. The release notes, the migration guide and
  ACP's page were read at the URLs above on 2026-10-31.
