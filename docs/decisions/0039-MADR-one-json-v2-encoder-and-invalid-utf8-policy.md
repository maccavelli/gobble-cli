---
status: accepted
date: 2026-10-31
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Encode and decode gobble's JSON through one v2 package, under an invalid-UTF-8 policy that cannot crash or lose data

## Context and Problem Statement

gobble is pure Go at Go 1.27.3 (`go.mod:3`). At that toolchain `encoding/json/v2` and `encoding/json/jsontext` are on by
default. v1 `encoding/json` "is implemented in terms of the v2 package with the appropriate set of options specified
that preserve the historical behavior of v1" (`go doc encoding/json`, "Migrating to v2"). gobble half-moved: 87 files
import v2 and 20 still import v1.

Two issues ask to finish the move, and this record takes in both:

* [gobble-cli#7](https://github.com/maccavelli/gobble-cli/issues/7) asks for one marshaler, `encoding/json/v2`, and
  one on-disk framing, JSONL. Its units are:
  * U1, a v1 sweep;
  * U2, strict decoding at the ACP boundary;
  * U3, store hardening;
  * U4, lockstep with magic-cli-remote's acp-go-sdk2 pin;
  * U5, a guard.
* [gobble-cli#9](https://github.com/maccavelli/gobble-cli/issues/9) asks what carries raw JSON once v2 is the only
  encoder. Its answers:
  * `jsontext.Value`, with constructors that close three footguns (F1 to F3);
  * a leaf helper package, `jsonx`;
  * the remaining v1 `RawMessage` spellings at the ACP seam (F4);
  * tests that pin absent against `null` (U3), docs (U5), and a guard on `[]byte` fields (U6).

[0038-REPORT-json-v2-standardization-and-sdk-surface.md](../reports/0038-REPORT-json-v2-standardization-and-sdk-surface.md)
evaluated #7 against acp-go-sdk2 `v1.29.0` and the official documentation, and recommended an order.

While the evidence for this record was gathered, three defects were found that neither issue nor the report names. They
decide this record's order. All three come from one fact: **v2 refuses to encode or decode what v1 accepted and
repaired, and gobble's own code was written as if it could not fail.** v1 replaced each invalid UTF-8 byte with U+FFFD,
decoded a lone surrogate escape as U+FFFD, and let its callers splice raw bytes without checking them.

**Defect 1: invalid UTF-8 in text gobble reads ends the program.**

* Every text tool result is built by `tool.TextResult`, `tool.ErrorResult` or `tool.BadArgumentsResult`. Each calls
  `quote` (`tool/tool.go:227-249`), which panics on an encoding error ("a string always encodes"). Fifty-seven calls in
  sixteen files build results so. Ten are in `tool/tool.go` itself, among them `ErrorResult(err.Error())` (`:493`), which
  carries a tool's own error text: an OS error naming a file whose name is not UTF-8 panics too.
* Measured on gobble's own code, each of these ended in `panic: jsontext: invalid UTF-8`:
  * `read` of a file holding `caf\xe9 au lait` (Latin-1);
  * `bash` running `printf 'a\377b\n'`;
  * `grep` over the Latin-1 file.
* Driven through the real agent, a scripted model's `read` of that file ended the test binary:

  ```text
  panic: jsontext: invalid UTF-8 [recovered, repanicked]
  tool.quote → tool.TextResult → builtin.readWith.func1 → tool.(*generic).Run → agent.(*turn).runTool
    → runTools → (*turn).run → (*Agent).Run → acpserver.(*Agent).Prompt
  ```

* A project's `AGENTS.md` does the same on the session's first prompt:
  * context files are read as raw bytes, a BOM trimmed (`prompt/context.go:49-54`; `SYSTEM.md` and
    `APPEND_SYSTEM.md` at `:171-176`; nested instructions at `:190-191`);
  * the baseline entry encodes them with `prompt.mustJSON` (`prompt/sections.go:43-49`), which panics ("plain strings
    always encode").

  Measured with an `AGENTS.md` of `R\xe9gle du projet.`:

  ```text
  panic: jsontext: invalid UTF-8 within "/baseline/3/text" after offset 685 [recovered, repanicked]
  prompt.mustJSON → prompt.BaselineData → acpserver.(*Agent).writePrompt → (*Agent).Prompt → acp-go-sdk2 handler
  ```

* **No `recover` exists in `agent`, `acpserver` or `tool`.** The CLI runs its agent in process through
  `agentproc.InProcess`, whose documentation says "A panic in Serve is not recovered: it ends the program". So gobble
  exits mid-turn. Under `gobble acp`, the agent process exits under the editor.

**Defect 2: text that escapes the panic is saved empty, or fails a whole request.**

* `toolResultMessage` and `assistantEntryMessage` (`acpserver/history.go:106`, `:120-124`), and four compaction sites
  (`compaction/serialize.go:65`, `:74`; `compaction/estimate.go:222`, `:262`), call
  `session.(*Message).SetBlocks` (`session/session.go:241-250`) and discard its error ("blocks of plain fields always
  encode"). Measured: `toolResultMessage` given `\xe9t\xe9` or `out: \xff\xfe` stores a message whose `content` is 0
  bytes, with no error and no log.
* `HandleExtensionMethod` encodes each `_gobble/*` result with v2, and answers an encoding error `-32603` for the whole
  request (`acpserver/extensions.go:72-75`).

**Defect 3: a raw JSON value that is not valid JSON breaks the turn for the client (#9's hatch, at the handler).**

* A model's tool-call arguments are spliced into ACP as `rawInput`, unchecked:
  * the live `tool_call` update (`acpserver/prompt.go:102`, through `rawInput` at `:199-204` and `jsonRaw` at `:277-282`);
  * the permission request (`acpserver/question.go:151`);
  * the replay on `session/load` (`acpserver/history.go:402`, whose arguments are checked already).
* A reply cut off mid-call gives arguments that are not JSON. acp-go-sdk2 encodes with v1, and v1 refuses to splice
  them. `updates.send` keeps the first send error and then sends nothing more ("once one fails, nothing more is sent",
  `acpserver/prompt.go:57-72`), and the prompt returns it (`acpserver/agent.go:749-750`).
* Measured through the real agent, with a scripted model whose `read` call's arguments were `{"path":"a.tx`, then a
  text reply:

  | Arguments | Prompt result | Updates the client saw | Model requests |
  | :--- | :--- | :--- | :--- |
  | `{"path":"a.txt"}` | `end_turn` | 8, among them the `tool_call`, two `tool_call_update`s and the reply text | 2 |
  | `{"path":"a.tx` | error `-32602 Invalid params`: `json: error calling MarshalJSON for type *acp.SessionUpdate: json: error calling MarshalJSON for type *jsontext.Value: unexpected end of JSON input` | 3: no `tool_call`, no result, no reply text | 2 |
  | empty | `end_turn` | 8, as for valid arguments | 2 |

  The agent ran on, and the model answered. The client saw neither, and its prompt failed, with a code that blames the
  client's own params.

**And one strictness gobble's handlers have that the SDK in front of them does not.** gobble decodes its own `_gobble/*`
request params from the raw bytes, with v2. A JavaScript client's `JSON.stringify` writes a lone UTF-16 surrogate as an
escape such as `\ud83d`. Measured: `_gobble/get_entries` with `{"sessionId":"nope"}` is answered "unknown session";
with `{"sessionId":"nope\ud83d"}` it is refused `-32602`, with the reason `jsontext: invalid surrogate pair`. Every
other ACP param goes through the SDK's v1 decoding, which reads that escape as U+FFFD.

The live model and the ACP wire are not where the damage starts:
* acp-go-sdk2 encodes and decodes with v1 only: 21 non-test files on `encoding/json`, none on v2, at `v1.28.0` and
  `v1.29.0`.
* go-llmprovider-sdk is mostly v1, 41 files against 2.
* So text that reaches either is repaired on the way, as v1 always did.

The damage starts in gobble's own v2 calls, which the half-finished migration put on the path of unchecked text and
unchecked raw JSON.

The question this record answers: **how gobble finishes the move to one v2 encoder, so that #7's and #9's goals are
met, and no input can crash gobble, empty a saved session, or break a turn through JSON encoding.**

### The facts, measured at `31e3c1c`

Every claim below was read from source at `31e3c1c`, quoted from official documentation, or measured by a probe. Each
probe ran in a scratch Go module or a scratch copy of this repository, never in the tree. "How it was read", at the end,
lists them.

**The toolchain.**

* `encoding/json` declares `type RawMessage = jsontext.Value`: an alias. Wherever acp-go-sdk2's API says
  `json.RawMessage`, gobble can write `jsontext.Value`, and the types are identical (0038, section 3, by probe against
  `v1.29.0`).
* The differences that matter here (`go doc encoding/json`):
  * v2 rejects invalid UTF-8 and duplicate object names;
  * it matches field names case-sensitively;
  * it does not escape `<`, `>` and `&`;
  * it marshals a nil slice as `[]` and a nil map as `{}`;
  * "a Go map is marshaled in a non-deterministic order".

  Each has an option that restores v1.

**gobble's JSON calls,** in non-test code, by a scan of each file's imports and calls:

| Package | Calls | Files |
| :--- | :--- | ---: |
| `encoding/json/v2` | `Marshal` 35, `MarshalEncode` 1, `Unmarshal` 60 | 43 |
| `encoding/json/jsontext` | `NewEncoder` 3, `NewDecoder` 5 | 5 |
| `encoding/json` (v1) | `Marshal` 3, `MarshalIndent` 2, `Unmarshal` 7, `NewDecoder` 1 | 5 calling, 10 importing |

Of the 20 v1 importers, 8 only spell `RawMessage`, 5 of them non-test, and 12 call v1's functions. #7's own
list is partly stale:
* `session/session.go` and `session/migrate.go` import only v2 now;
* `internal/cli/choice.go`, `internal/cli/providerscmd.go` and `internal/archtest/check.go` do import v1, and are not
  named.

**The encode sites that assume they cannot fail.** Each is a panic or a discarded error on an encoding error:

| Site | On failure | Input that can carry invalid UTF-8 |
| :--- | :--- | :--- |
| `tool/tool.go:244-249` `quote`, under `TextResult`, `ErrorResult`, `BadArgumentsResult` | panic | yes: file text, process output, file names, OS error text |
| `mcpclient/tool.go:128-134` `quote` | panic | MCP result text, decoded by the MCP go-sdk's own decoder (not measured) |
| `prompt/sections.go:43-49` `mustJSON`, under `BaselineData` and `PatchData` | panic | yes: context files, `SYSTEM.md`, templates |
| `acpserver/history.go:46-52` `mustJSON` | panic | user text, which arrives over ACP already repaired by the SDK's v1 decoding |
| `llm/provider/provider.go:363-368` `reasoningData` | panic | reasoning text, decoded by go-llmprovider-sdk's v1 decoding |
| `acpserver/history.go:106`, `:122`; `compaction/serialize.go:65`, `:74`; `compaction/estimate.go:222`, `:262` | `SetBlocks` error discarded | yes |
| `acpserver/extensions.go:72-75` | `-32603` for the whole request | yes |

`internal/config/render.go:92` also says "always encodes", but it is YAML, not JSON, and outside this record.

**The configuration files.** [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md) settled their
formats:
* "D3. YAML, read strictly", for `settings.yaml`. Its parser refuses a duplicate key and a second document
  (`internal/config/read.go:20-45`).
* D2: "`mcp.json` and `models.json` stay JSON, because other tools share their formats", and "`trust.json` and
  `auth.json` stay JSON, because gobble writes them and users do not edit them".

gobble also writes `permissions.json` and `recent-models.json`. Today each JSON file is decoded with v2's defaults,
which refuse invalid UTF-8 and duplicate names, except `recent-models.json` and the kept listing, which v1 reads.

**The repair, measured.**

* For `x\xff\xfe\xfdy`: v1 and v2 with `jsontext.AllowInvalidUTF8(true)` each write `x`, then three U+FFFD, then `y`:
  one for each invalid byte. `strings.ToValidUTF8(s, "\uFFFD")` writes one for the run. So only the encoder option
  gives the bytes the model and every ACP client already see through the v1 SDKs.
* Decoding, with the same option:
  * a raw invalid byte reads as U+FFFD;
  * a lone surrogate escape reads as U+FFFD.

  v2 refuses both without the option.
* A spliced `jsontext.Value` holding a raw invalid byte:
  * v2 refuses it;
  * with `AllowInvalidUTF8`, it is written with U+FFFD;
  * with `AllowInvalidUTF8` and `PreserveRawStrings`, **the invalid byte is written out unchanged**.

**Splicing raw values at the ACP seam, measured** with v1, as acp-go-sdk2 encodes. The type is the SDK's `json.RawMessage`
under `omitempty`:

| Raw value | `IsValid()` | `IsValid(AllowInvalidUTF8(true))` | v1's splice |
| :--- | :--- | :--- | :--- |
| `{"path":"a.txt"}` | true | true | spliced |
| `{"path":"a.tx` | false | false | **error**: `unexpected end of JSON input` |
| `{"path":"a\xffb"}` | false | true | spliced, the byte written as U+FFFD |
| `{"path":"a\ud83d"}` | false | true | spliced, the escape kept |
| empty non-nil | false | false | left out |

So `IsValid(jsontext.AllowInvalidUTF8(true))` is exactly the test of what the SDK can splice.

**Quoting a string without an error path, measured.** `jsontext.AppendQuote` "replaces invalid UTF-8 bytes with the
Unicode replacement character and an error is returned at the end indicating the presence of invalid UTF-8"
(`go doc encoding/json/jsontext.AppendQuote`). Over 200,009 inputs, nine fixed cases and 200,000 random byte strings up
to 24 bytes:
* its output equals `json.Marshal(s, jsontext.AllowInvalidUTF8(true))` byte for byte in every case, and is always valid
  JSON;
* its error is returned for each of the 182,061 inputs holding invalid UTF-8, and for none of the others.

So its error reports a repair, which the data policy makes, and never a failure.

**What #9's footguns are, measured.**

* **F1. Confirmed, and latent.**
  * An empty non-nil `jsontext.Value{}` on an `omitzero` field fails the whole marshal: on
    `session.Entry.Replacement`, `cannot marshal from Go jsontext.Value: unexpected EOF`.
  * `agent/agent.go:771` and `:951` build `tool.Call{Args: jsontext.Value(c.Args)}`, which is empty non-nil when
    `c.Args` is `""`.
  * But no path marshals it with v2 today:
    * `tool.Call` is never marshalled;
    * `permission.Rule.Input` reaches ACP through `jsonRaw`, which maps empty to nil;
    * `session.Block.Arguments` falls back to `"{}"` (`acpserver/history.go:89-96`);
    * no production code builds a `hook.Event`.
* **F2. Confirmed.** v2 reformats a spliced value. Measured on `{ "n": 1e2, "s": "a</b>\u0041" }`:
  * v2 wrote `{"n":1e2,"s":"a</b>A"}`;
  * v2 with `jsontext.PreserveRawStrings(true)` kept `\u0041`;
  * v1 wrote `"a\u003c/b\u003e\u0041"`.
* **F3. Confirmed in part.** `jsontext.Value(nil).String()` is `"null"`; an empty non-nil value's is `""`. So an absent
  value logs as `null`.
* **F4. Confirmed.** The five `stdjson.RawMessage` spellings at the ACP seam are `acpserver/prompt.go:199` and `:277`,
  `acpserver/extensions.go:64` and `:77`, and `acpserver/grants.go:80`.
* **Absent and `null` are already distinct.** Measured on `session.Entry`:
  * an absent `replacement` is left out;
  * a `null` one is kept as `null`;
  * `{"content":"x"}` round-trips.

  #9's U3 pins today's behaviour; it fixes nothing.
* **U6's premise holds.** `llm.Content.Data` (`llm/llm.go:126`) is the only `[]byte` field with a `json` tag in non-test
  code.

**`stream-json` already mixes encoders.** Measured with a reply of `if a < b && c > d`:
* the update line, re-encoded by `acpclient`'s `translate` with v1 (`acpclient/update.go:244`, written at
  `internal/cli/print.go:113`), holds `"if a \u003c b \u0026\u0026 c \u003e d"`;
* the result line, written by `writeJSON` with v2 (`internal/cli/print.go:145`), holds `"if a < b && c > d"`.

`internal/cli/testdata/print-stream-json.golden` holds none of `<`, `>`, `&`, U+2028 or U+2029, so it can see neither.
0038 measured that v2 with `jsontext.EscapeForHTML(true)` and `jsontext.EscapeForJS(true)` matched v1 on all 21 samples
of acp-go-sdk2's `session_update` goldens.

**Sorting map keys, measured** with `testing.Benchmark` at Go 1.27.3, darwin/arm64, one run each:

| Value | Without `Deterministic` | With it |
| :--- | :--- | :--- |
| a struct, no map | 341 ns, 2 allocations | 348 ns, 2 allocations |
| a struct with a 3-key map | 750 ns, 4 | 783 ns, 7 |
| a map of 8 keys | 1,645 ns, 20 | 1,879 ns, 28 |
| a map of 200 keys | 28.6 µs, 204 | 61.6 µs, 404 |

gobble's hot encodes, a session line and `translate`, are structs. The SDK's methods fix `translate`'s order.

**What 0038 corrected in #7,** re-checked here:

* U4 is met: gobble and magic-cli-remote both pin acp-go-sdk2 `v1.28.0` (magic-cli-remote's `go.mod:11`).
* U2 cannot work through the SDK:
  * its generated types decode themselves, so `json.RejectUnknownMembers(true)` does not reach inside them;
  * ACP's schemas carry `additionalProperties: true` about 125 times and `false` never (acp-go-sdk2's
    `0012-MADR-schema-fidelity-framework.md`);
  * ACP's extensibility page says "All possible names are reserved for future protocol versions".
* U3's premise is wrong. magic-cli-remote's ACP read cap is 10 MiB (`maxACPMessageSize = 10 << 20`,
  `internal/provider/acpagent/acpagent.go:1144`), the SDK's default, not 1 MiB. Most of U3 is in the store already.

**One error in 0038,** found here. Its section 5 row D1 shows the same unescaped string, `"<b>a & b</b>"`, for v1 and for
v2. v1 writes `"\u003cb\u003ea \u0026 b\u003c/b\u003e"`.

**A size limit #7 aimed at the wrong place.**
* `edit` and `write` put the whole old and new file in a diff (`tool/builtin/edit.go:73`, `tool/builtin/write.go:37`).
* `resultContent` sends it whole (`acpserver/prompt.go:215-219`), and `rawInput` sends a call's whole arguments (`:199`).
* Above 10 MiB a reader logs "incoming message exceeds the size limit; answered and skipped", and drops the update
  (acp-go-sdk2 `v1.29.0` `connection.go:1385-1389`).

That is a wire bound, for [0037-MADR-hardening-integrity-and-performance.md](0037-MADR-hardening-integrity-and-performance.md)'s
resource bounds, not a marshaler change.

## Decision Drivers

* **Nothing gobble reads may crash it, empty a saved session, or break a turn.** Text from files, processes, file names
  and other programs' JSON is outside gobble's control. So are a model's arguments. A non-UTF-8 byte or a cut-off call
  is ordinary: Latin-1 and Windows-1252 files, `git log` of old repositories, Linux file names, a reply that hit its
  token limit.
* **What is saved must match what the model saw.** The model sees tool output through go-llmprovider-sdk's v1 encoding,
  which replaces each invalid byte with U+FFFD. A resumed, forked or cloned session must hold the same text.
* **Configuration is strict** (the owner's answer to Q2). A file a user or gobble keeps as configuration or state is
  refused with a diagnostic when it is malformed, never silently repaired. A credential must never be altered.
* **One encoder, as Go recommends.** "All new usages of "json" in Go should use the v2 package" (`go doc encoding/json`).
  One package holding gobble's options means one place for a policy, where today 43 files each choose their own.
* **Everything gobble-cli defines is on v2** (the owner, 2026-10-31: "we need to move everything to jsonv2 for
  gobble-cli"). Every JSON value gobble defines is encoded and decoded by gobble's code, through `jsonx`:
  * its files;
  * its extension methods' params, results and notifications;
  * its `_meta.gobble` members;
  * its JSON-RPC error data;
  * its CLI output.

  Where acp-go-sdk2 routes a gobble payload (`ExtensionMux`), it hands gobble the raw bytes, and gobble decodes them.
  What stays outside gobble's code is ACP's own envelope and spec types, which acp-go-sdk2 encodes and decodes, and the
  provider and MCP protocols, which their SDKs encode and decode.
* **No byte a script reads changes,** unless decided and recorded: `stream-json` and `--output-format json`.
* **Each guard and test is seen failing first,** on a scratch copy (`AGENTS.md`, "Checks").
* **ACP's extensibility is kept.** Unknown members are tolerated and preserved, as the protocol says.

## Considered Options

* **A. Fix only the panics.** Recover in the agent, or check UTF-8 at the panicking sites, and leave the rest as it is.
* **B. Return gobble's encode sites to v1, or to v2 with `DefaultOptionsV1`.** v1 never fails on invalid UTF-8.
* **C. Repair text where it enters**, with `strings.ToValidUTF8` at each source: tool output, context files, templates.
* **D. One encoder package, `jsonx`, which every gobble encode and decode goes through.** It holds two policies: a
  lenient one for protocol, session and tool data, and a strict one for configuration. It validates raw JSON at the ACP
  seam. v1 goes, the discarded errors and panics are replaced, and guards keep it so.

## Decision Outcome

Chosen option: **"D. One encoder package"**, because:
* it closes all three defects at the encoder and the seam, for every site, the 57 result constructor calls and any site
  written later included;
* its data repair is v1's byte for byte, which is what the model and the ACP clients already see;
* its configuration stays strict, as the owner decided;
* it meets #7's and #9's goals in the same package.

The other options fall short:
* A treats one symptom at a time, and leaves the next unchecked site to crash.
* B keeps two behaviours where Go recommends one.
* C collapses runs of invalid bytes where the model's copy does not, and has to find every source.

Status: **accepted** 2026-10-31, by the owner's "accept the madr". The owner answered Q1 to Q6 the same day, and the
decisions below are written to those answers (More Information, "The owner's answers"). Its PLAN is
[0039-PLAN-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-PLAN-one-json-v2-encoder-and-invalid-utf8-policy.md).

### The package

* **D1. `jsonx`, a leaf package at the module root,** `github.com/maccavelli/gobble-cli/jsonx` (Q5). It imports only the
  standard library: `encoding/json/v2`, `encoding/json/jsontext` and `bytes`. Contract packages cannot import
  `internal/`, so it is not internal. They are `session`, `tool`, `hook`, `permission`, `llm`, `prompt`, `mcpclient` and
  `compaction`. Its surface, whose names may change in review but whose roles may not:

  ```go
  // The data policy (D2): protocol, session, tool and provider data, and CLI output.
  func Marshal(v any, o ...json.Options) ([]byte, error)
  func Unmarshal(b []byte, v any, o ...json.Options) error
  func Options() json.Options // for a jsontext.Encoder or Decoder, or a v2 call that needs one

  // The configuration policy (D3): configuration and state files.
  func MarshalStrict(v any, o ...json.Options) ([]byte, error)
  func UnmarshalStrict(b []byte, v any, o ...json.Options) error
  func StrictOptions() json.Options

  // The hatch (#9).
  func Value(v any) (jsontext.Value, error)  // Marshal's bytes as a hatch value; never empty non-nil
  func Quote(s string) jsontext.Value        // s as a JSON string; cannot fail under D2
  func From(b []byte) jsontext.Value         // empty is absent (nil); does not validate
  func Splice(v jsontext.Value) jsontext.Value
  func IsAbsent(v jsontext.Value) bool       // len(v) == 0
  func IsNull(v jsontext.Value) bool         // a JSON null, spaces trimmed; absent is not null
  ```

  `Splice(v)` decides what a raw value becomes where it is spliced into another encoder, the ACP seam (D9):
  * an empty value is absent;
  * a value `IsValid(jsontext.AllowInvalidUTF8(true))` accepts is itself;
  * anything else is `Quote(string(v))`, the text as a JSON string.

  *Amended 2026-10-31, by the PLAN's deviation 1: `jsonx` also has `type Field struct{ Name, Value string }` and
  `Fields(fs ...Field) jsontext.Value`. It writes an object of the fields whose `Value` is not empty, in the order given,
  through `Quote`. That is what `Marshal` writes for a struct of `omitzero` string fields, and a fuzz target holds the
  equality. It cannot fail. It serves the four-string `reasoning` value (D6).*

  `Quote(s)` is `jsontext.AppendQuote(nil, s)`. Its error means only that invalid UTF-8 was replaced, as measured
  above, which is D2's policy, so `Quote` returns the literal and no error. The fuzz target of Confirmation 5 holds the
  measurement: `Quote` equals `Marshal` under D2, byte for byte.

  `o` comes after the policy's options, so a caller adds options, such as `jsontext.WithIndent("  ")`. Undoing the
  policy is what the guard (D24) and the review are for.

### The two policies

* **D2. The data policy is `jsontext.AllowInvalidUTF8(true)` and `json.Deterministic(true)`.** It applies to protocol
  data, session files, tool arguments and results, provider data, and CLI output.
  * **Encoding:** an invalid byte in a string is written as U+FFFD, one for each byte, as v1, acp-go-sdk2 and
    go-llmprovider-sdk write it. A spliced value's invalid bytes are written so too.
  * **Decoding:** an invalid byte, and a lone surrogate escape, read as U+FFFD, as v1 read them. Duplicate names are
    still refused, names still match case-sensitively, and unknown members are still ignored, all as today.
    `RejectUnknownMembers` is not set (D19).
  * **Map keys are sorted** (Q3), as v1 sorted them. Each file, session line and output is then the same from run to
    run. The cost is in the measured table: none on a struct, 2.2× on a 200-key map. `Deterministic` affects only maps;
    a struct's order is its fields'.
  * **HTML escaping is off,** v2's default, except at `translate` (D16).
  * **`PreserveRawStrings` is used nowhere.** With `AllowInvalidUTF8` it writes a spliced value's invalid bytes
    unchanged (measured).
* **D3. The configuration policy is v2's defaults with `json.Deterministic(true)`** (Q2), for decoding and encoding:
  * a configuration or state file holding invalid UTF-8, a lone surrogate escape, or a duplicate name is refused with
    its error. That is today's behaviour, except for `recent-models.json` and the kept listing, which v1 reads
    leniently today (D17). The caller reports the file and the error, as each does now;
  * **no exception, `mcp.json` included** (Q6). A duplicate name is refused at three gates:
    * **gobble never writes one.** v2's encoder refuses to write a duplicate name, by every route. Measured: a struct
      whose embedded unknown members repeat a known name gives `duplicate object member name "type"`; an encoder
      written token by token gives `duplicate object member name "github"`; a raw value holding one is refused when
      spliced. With every write through `jsonx` (D4), no file gobble writes can hold one;
    * **gobble refuses to read one.** Today `mcp.json`'s loader and editor read each object with
      `AllowDuplicateNames(true)`: "A repeated name keeps its first place and its last value, as JSON.parse does"
      (`mcpclient/file.go:27-31`). That one function, `members`, reads the top level, the `mcpServers` object, each
      server's fields and their nested objects (`mcpclient/config.go:113`, `:127`, `:137`, `:219`, `:250`, `:341`).
      No record chose it. It loses data silently. Measured on an `mcp.json` whose `mcpServers` held `github` twice,
      `"command":"first"` then `"command":"second"`:
      * `Load` gave `github` the second, with no error;
      * `mcpclient.Add` of an unrelated server wrote the file back with only the second `github`, the first deleted.

      `members` loses `AllowDuplicateNames`. A duplicate fails `Load` as an unreadable file does, one error in
      `Config.Errors`, `<path>: <message>` (`mcpclient/config.go:66-71`, `:88-95`). The error names the object's path
      and the repeated name, and the agent shows it as a note (`acpserver/mcp.go:71-73`). `gobble mcp list` prints it
      as `config error:` and exits 1 (`internal/cli/mcp.go:250`, `:298`). `Add` and `Remove` refuse to edit the file and
      return the same error, so no edit deletes a value;
    * **`gobble config validate` checks the JSON configuration files before gobble runs.** Today it checks only the
      settings files (`internal/cli/config.go:57-60`, `:315-341`).
      * It also checks `mcp.json`, `models.json`, `auth.json`, `permissions.json` and `trust.json`, at the paths gobble
        reads them from, all in the configuration directory (`internal/cli/agent.go:178-184`,
        `internal/cli/credentials.go:20-25`, `internal/cli/modelsjson.go:40`, `permission/store.go:20`,
        `permission/trust.go:21`). gobble reads no project-level `mcp.json`. Each is decoded by `jsonx.UnmarshalStrict`,
        and `mcp.json`'s entries by `mcpclient.Load`.
      * Each finding is a `config.Diagnostic` (`internal/config/diag.go:17-23`). A decode error is a
        `*jsontext.SyntacticError`, whose `ByteOffset` gives the line, counted in the file's bytes, and whose
        `JSONPointer` gives the path.
      * A finding names the file, the line, the path and the problem, and never a value: `auth.json` holds
        credentials.
      * An error exits 1, as a settings error does today (0018-MADR D7). So a user, CI or a pre-commit hook can refuse
        a bad file before gobble reads it.

    The cost: a file that `JSON.parse`-based tools accept with a duplicate, keeping its last value, is refused by
    gobble until the duplicate is removed. Such a file is ambiguous: no reader can tell which value was meant;
    * invalid UTF-8 is refused there, as everywhere under this policy;
  * a value that cannot be encoded strictly is refused, never repaired. A credential holding an invalid byte is not
    written to `auth.json` altered; `SetKey` returns the error, as it does today (`internal/auth/store.go:160-165`);
  * unknown members are tolerated, as today: `mcp.json` and `models.json` are shared with other tools, whose members
    gobble keeps;
  * it covers `auth.json`, `models.json`, `mcp.json`, `permissions.json`, `trust.json`, `recent-models.json` and the
    kept model listing, and gobble's embedded catalog (`llm/catalog/catalog.go:32`).

  The owner's answer to Q2 keeps `settings.yaml` as gobble's configuration file, YAML and read strictly by
  `internal/config` (0018-MADR D3). This record changes no format. The JSON files stay JSON, by 0018-MADR D2.

### Every call site

* **D4. Each non-test file that calls JSON or spells v1's types** moves to `jsonx` under the policy its data has. That
  includes the test-support packages `acpclient/acptest` and `llm/llmtest`, which are not test files, so D24's guards
  cover them:

  | File | Today | Lines | Policy | Change |
  | :--- | :--- | :--- | :--- | :--- |
  | `acpclient/acptest/client.go` | v1 | `:70`, `:77` | data | `jsontext.Value` in the field and the signature (D11) |
  | `acpclient/acptest/recorder.go` | v2 | `:178` | data | `jsonx.Unmarshal` |
  | `acpclient/acptest/script.go` | v1 | `:158`, `:185`, `:203` | data | `jsonx.Unmarshal` and `Marshal`; the SDK types' own methods decide their form |
  | `acpclient/update.go` | v1 | `:135`, `:167`, `:191`, `:244` | data | the three decoders go (D12); `translate` per D16 |
  | `acpserver/agent.go` | v2 | `:834` | data | `jsonx.Unmarshal` |
  | `acpserver/commands.go` | v2 | `:202` | data | `jsonx.Unmarshal` |
  | `acpserver/compact.go` | v2 | `:115` | data | `jsonx.Marshal` |
  | `acpserver/extensions.go` | v1+v2 | `:73`, `:129` | data | results by `jsonx.Marshal`; params by `jsonx.Unmarshal` (D10); `jsontext.Value` in the signature (D11) |
  | `acpserver/grants.go` | v1+v2 | `:102` | data | `jsonx.Unmarshal`; `jsontext.Value` at `:80` (D11) |
  | `acpserver/history.go` | v2 | `:39`, `:47`, `:78`, `:91`, `:98`, `:262`, `:282`, `:423` | data | `mustJSON` goes (D6); `jsonx.Value` and `jsonx.Unmarshal`; errors to the writer (D7) |
  | `acpserver/modes.go` | v2 | `:237` | data | `jsonx.Unmarshal` |
  | `acpserver/personality.go` | v2 | `:47` | data | `jsonx.Unmarshal` |
  | `acpserver/prompt.go` | v1 | `:199`, `:277` | data | `jsonRaw` becomes `jsonx.Splice` (D9); `jsontext.Value` (D11) |
  | `acpserver/sysprompt.go` | v2 | `:304` | data | `jsonx.Unmarshal` |
  | `agent/agent.go` | v2 | `:216` | data | `jsonx.Marshal`; `:771`, `:951` through `jsonx.From` (D20) |
  | `compaction/compaction.go` | v2 | `:151` | data | `jsonx.Unmarshal` |
  | `compaction/estimate.go` | v2 | `:246`, `:261` | data | `jsonx.Unmarshal`; the two `SetBlocks` errors returned (D7) |
  | `compaction/prompts.go` | v2 | `:248` | data | `jsonx.Unmarshal` |
  | `compaction/serialize.go` | v2, jsontext | `:100`, `:228`, `:274` | data | `jsonx.Unmarshal`; the two `SetBlocks` errors returned (D7) |
  | `internal/archtest/check.go` | v1 | `:68` | data | a `jsontext.Decoder` with `jsonx.Options()` (D18) |
  | `internal/auth/store.go` | v2, jsontext | `:84`, `:111`, `:160`, `:243`, `:285` | configuration | `jsonx.MarshalStrict` and `UnmarshalStrict`; its decoders take `jsonx.StrictOptions()` |
  | `internal/cli/auth.go` | v2 | `:170`, `:409` | data (output) | `jsonx.Marshal` |
  | `internal/cli/choice.go` | v1 | `:128`, `:152`, `:211`, `:224` | configuration | `jsonx.MarshalStrict` and `UnmarshalStrict` (D17) |
  | `internal/cli/config.go` | v2 | `:196`, `:213` | data (output) | `jsonx.Marshal` with `WithIndent`; it sets `Deterministic` itself today |
  | `internal/cli/configpath.go` | v2 | `:61` | data (output) | `jsonx.Marshal` |
  | `internal/cli/mcp.go` | v2, jsontext | `:155`, `:181`, `:256` | configuration at `:155` and `:181`, which write `mcp.json`'s members; data (output) at `:256` | `StrictOptions()` and `jsonx.Marshal` |
  | `internal/cli/print.go` | v2 | `:145` | data (output) | `jsonx.Marshal`; bytes unchanged (D16) |
  | `internal/cli/providerscmd.go` | v1 | `:51` | data (output) | `jsonx.Marshal` with `WithIndent` (D17) |
  | `internal/cli/sessions.go` | v2 | `:191` | data | `jsonx.Unmarshal` |
  | `internal/cli/version.go` | v2 | `:41` | data (output) | `jsonx.Marshal` |
  | `llm/llmtest/assert.go` | v2 | `:42` | data | `jsonx.Unmarshal` |
  | `internal/config/schema.go` | v2 | `:34`, `:64` | data (output) | `jsonx.Marshal` |
  | `llm/catalog/catalog.go` | v2 | `:32` | configuration (embedded) | `jsonx.UnmarshalStrict` |
  | `llm/catalog/modelsjson.go` | v2 | `:141` | configuration | `jsonx.UnmarshalStrict` |
  | `llm/provider/provider.go` | v2 | `:364`, `:438`, `:445`, `:460` | data | `reasoningData` through `jsonx.Value`, its error returned (D6); `jsonx.Unmarshal` |
  | `mcpclient/config.go` | v2 | `:209`, `:226`, `:256`, `:273`, `:306`, `:355` | configuration | `jsonx.UnmarshalStrict` |
  | `mcpclient/file.go` | jsontext | `:31`, `:80` | configuration | the decoder and encoder take `jsonx.StrictOptions()` |
  | `mcpclient/tool.go` | v2 | `:76`, `:130`, `:143` | data | `quote` becomes `jsonx.Quote` (D6); `jsonx.Marshal` |
  | `permission/store.go` | v2 | `:72`, `:180` | configuration | `jsonx.MarshalStrict` and `UnmarshalStrict` |
  | `permission/trust.go` | v2 | `:150`, `:241` | configuration | `jsonx.MarshalStrict` and `UnmarshalStrict` |
  | `prompt/sections.go` | v2 | `:44`, `:55` | data | `mustJSON` goes; `BaselineData` and `PatchData` return `(jsontext.Value, error)` (D6) |
  | `session/jsonl/store.go` | v2 | `:201`, `:215`, `:316`, `:342` | data | `jsonx.Marshal` and `Unmarshal`; a line with a lone surrogate escape loads (D2) |
  | `session/migrate.go` | jsontext | `:101`, `:103` | data | the decoder and encoder take `jsonx.Options()` |
  | `session/session.go` | v2 | `:230`, `:234`, `:245` | data | `SetBlocks` by `jsonx.Marshal`; `Blocks` by `jsonx.Unmarshal` |
  | `tool/builtin/edit.go` | v2 | `:111` to `:163`, ten calls | data | `jsonx.Marshal` and `Unmarshal` |
  | `tool/builtin/grep.go` | v2 | `:366` | data | `jsonx.Unmarshal` |
  | `tool/builtin/intercept.go` | v2 | `:22`, `:57` | data | `jsonx.Unmarshal` and `Marshal` |
  | `tool/builtin/paths.go` | v2 | `:91` | data | `jsonx.Unmarshal` |
  | `tool/builtin/read.go` | v2 | `:118` | data | `jsonx.Unmarshal` |
  | `tool/builtin/shell.go` | v2 | `:316`, `:327`, `:338`, `:348`, `:360` | data | `jsonx.Unmarshal` |
  | `tool/tool.go` | v2 | `:220`, `:245`, `:478`, `:485`, `:507`, `:549` | data | `quote` becomes `jsonx.Quote` (D6); `jsonx.Marshal` and `Unmarshal` |
  | `tool/toolsearch/index.go` | v2 | `:169` | data | `jsonx.Unmarshal` |

  D13 to D15 also reach files with no JSON call today:
  * `acpclient/conn.go`: its `HandleExtensionMethod` switch (`:415-448`) gives way to D12's `ExtensionMux`, which
    removes its v1 spelling; plus its error data (D14) and its `_meta.gobble` (D15);
  * `acpclient/sessions.go`: D13's request, and D15's writer and reader;
  * `acpserver/question.go`, `acpserver/mentions.go`, `acpserver/config.go`: D15's readers and writer;
  * `acpserver/auth.go`, `acpserver/autocompact.go`: D14's error data and D13's notifications, through `notify`.

  A tool's arguments keep v2's refusal of duplicate names. A model that sends `{"path":"a","path":"b"}` gets a
  `BadArgumentsResult`, as today, and can correct its call.

### Defects 1 and 2: invalid UTF-8

* **D5. Text is not repaired at its source.** Tool output, file text and file names go to the model as they are today.
  The provider SDK repairs them on the way, and D2 repairs gobble's saved copy the same way, byte for byte. The terminal
  views clean what they show as they do today: `term.Sanitize` and `safetext` (0035-MADR D14).
* **D6. No encode panics.**
  * `tool.quote` and `mcpclient`'s `quote` become `jsonx.Quote`. A Go string always encodes under D2, so `Quote` has no
    error. A fuzz target holds that claim (Confirmation 5).
  * `prompt.mustJSON` and `acpserver`'s `mustJSON` go. `BaselineData`, `PatchData`, `userEntryMessage` and the
    `gobbleMember` encoder (`acpserver/history.go:39`) use `jsonx.Value`, and return its error.
  * `reasoningData` and `signatureData` (`llm/provider/provider.go:363-376`) return `(jsontext.Value, error)` through
    `jsonx.Value`. Their callers in the stream loop (`:338`, `:340`) yield that error, as the loop yields a provider's
    (`:323`).

    *Amended 2026-10-31, by the PLAN's deviation 1:* `acpserver`'s `mustJSON` has eight callers, not two. The four
    entry builders, `contextEntry`, `toolsEntry`, `todoEntry` and `personalityEntry`, return their error to the writer
    or to `append`. The `reasoning` value, four strings, is built by `jsonx.Fields`, which cannot fail, in
    `reasoningData`, `signatureData` and `acpserver`'s replay (`acpserver/history.go:272`, `:287`). So those keep their
    signatures, and neither the stream loop nor the replay gains an error path.
* **D7. No encode error is discarded.** Each of the six `_ = m.SetBlocks(...)` goes, in one of three ways:
  * **`session.(*Message).SetText(s string)`,** new, with no error: the content is one text block, built from
    `jsonx.Quote`, so it cannot fail. It replaces `SetBlocks` at the four sites that write one text block:
    `acpserver/history.go:122` (`toolResultMessage`), `compaction/serialize.go:65`, and `compaction/estimate.go:222` and
    `:262`. No signature changes.
  * **`SetBlocks` keeps its error,** and `assistantEntryMessage` (`acpserver/history.go:106`) returns it. Its blocks hold
    the model's tool-call arguments, a `jsontext.Value`, which can be bad JSON.
  * **`Convert`'s custom case stops re-encoding** (`compaction/serialize.go:72-75`). The converted message keeps the
    custom message's content as stored. `Convert`'s three callers each read its result through `Message.Blocks` at once:
    * `acpserver/history.go:218`, through `addMessage` at `:226-227`;
    * `compaction/serialize.go:161`, at `:165`;
    * `compaction/compaction.go:341`, at `:345`.

    `Blocks` reads a string or an array of blocks alike (`session/session.go:225-238`). So no reader sees a change, and
    `Convert` and its callers, `replayPath`, `serializeOmitting` and `turnRequest`, which return no error, keep their
    signatures.

    *Corrected 2026-10-31, by the PLAN's deviation 3:* a reader does see the change. `TestProjectMatchesPi` compares
    the converted messages with Pi's `convertToLlm`, which makes a string content one text block, and that is what the
    re-encode did. So a string content is written with `SetText`, and an array is kept as stored. Neither has an error
    path, and the Pi fixtures pass.

  Under D2, a string and a struct of strings always encode. What can still fail is a `jsontext.Value` holding bad JSON,
  which is a defect worth seeing. In `acpserver`, `assistantEntryMessage`, `userEntryMessage`, `BaselineData` and
  `PatchData` return such an error. The session writer keeps it as it keeps an append's error (`writer.add`,
  `acpserver/agent.go:870-874`), and the prompt answers with `appendFailed`'s error, as for a failed append
  (`acpserver/agent.go:644-645`, `:654-655`, `:673-674`, `:731-733`).
* **D8. Extension results** are encoded by `jsonx.Marshal`, so text in a result no longer fails the request `-32603`.
  An error that remains is a `jsontext.Value` holding bad JSON, answered as today.

### Defect 3 and the handlers

* **D9. Every raw value spliced into ACP goes through `jsonx.Splice`:**
  * `rawInput` and `jsonRaw` (`acpserver/prompt.go:199-204`, `:277-282`) become `jsonx.Splice`, which serves the live
    `tool_call`, the permission request (`acpserver/question.go:151`), and the replay (`acpserver/history.go:402`);
  * a valid value is sent as it is, and an empty one is left out, both as today;
  * arguments that are not JSON are sent as a JSON string of their text. The client sees what the model sent, and the
    update is encoded. The model gets the tool's own `BadArgumentsResult`, as today.

  The alternatives were to leave `rawInput` out, which hides what the model sent, or to refuse the call before its
  update, which changes the agent. Neither keeps the client's view and the agent's behaviour.
* **D10. Every `_gobble/*` payload gobble receives is decoded by gobble, with v2, under the data policy:**
  * the agent's request params (`acpserver/extensions.go:129`), which today refuse a lone surrogate escape
    (`-32602 invalid surrogate pair`, measured), and then read it as U+FFFD;
  * the client's three notifications, decoded with v1 today (`acpclient/update.go:135`, `:167`, `:191`), and with v2
    through D12.

  So no `_gobble/*` payload is decoded with v1 semantics, and none is refused where the SDK in front of it would read
  it.
* **D11. The `RawMessage` spellings become `jsontext.Value`** (#9's F4). They are the five seam sites above and the 8
  files that only spell the type, 0038's step 1. The types are identical, so nothing changes at run time.

  The eight `json:",inline"` tags go with them: `acpserver/extensions.go:88`, `:93`, `:98`, `:103`, `:108`, `:113` and
  `:118`, and `acpserver/personality.go:276`. Go 1.27 renamed that option `embed`. Each tags an embedded `sessionParams`,
  which "is implicitly JSON embedded unless an explicit JSON name is specified" (`go doc encoding/json/v2`). 0038
  measured that v2 accepts the stale option without error, and that the fields decode and encode the same, so the tag
  does nothing. `session/session.go:90` already writes `embed`.
* **D12. The client's three `_gobble/*` notifications are routed by acp-go-sdk2's `ExtensionMux`, and decoded by gobble
  with v2.** Each is registered with `acp.Notify[jsontext.Value]`.
  * `acp.Notify` decodes params with the SDK's `decodeParams`, which is v1's `json.Unmarshal`, `{}` when absent
    (acp-go-sdk2 `v1.28.0` `extensions.go:87-98`, `connection.go:110-115`). Into a `jsontext.Value`, v1's decode only
    copies the value. Measured: whitespace inside the value, a lone surrogate escape, a raw invalid byte and a
    duplicate name all reached the handler verbatim, only the surrounding spaces trimmed.
  * Each handler decodes the bytes with `jsonx.Unmarshal` into the types `retryOf`, `compactionOf` and `personalityOf`
    use today (`acpclient/update.go:125-195`). Those functions move to v2, and with them the v1 import, and the
    routing to `h.sessions` and `h.pending` stays as it is. The sender is gobble's own agent, which writes the members'
    exact names, so v2's case-sensitive match changes nothing.
  * An unregistered `_` notification is ignored, as ACP says one "SHOULD" be. Today it is answered `-32601`, which the
    SDK drops.
  * This is 0038's step 2, with one change: the report had the SDK decode into the types, which is v1. Here the SDK
    routes and gobble decodes. The agent's twenty methods stay on gobble's map (D22).
* **D13. Every `_gobble/*` payload gobble sends is encoded by gobble, with v2, and handed to the SDK as a
  `jsontext.Value`:**
  * the agent's notifications, all of which pass through `updates.notify` (`acpserver/prompt.go:68-72`): `_gobble/retry`
    (`:111`), `_gobble/compaction` (`acpserver/compact.go:26`, `:28`; `acpserver/autocompact.go:127`, `:131`, `:138`,
    `:146`; `acpserver/extensions.go:243`, `:245`) and `_gobble/personality` (`acpserver/personality.go:182`;
    `acpserver/commands.go:731`). `notify` encodes once, with `jsonx.Value`, and its error is the update error it already
    keeps;
  * the client's `_gobble/set_session_name` request (`acpclient/sessions.go:107`);
  * the agent's extension results, which `extensions.go:73` already encodes with v2 (D8).

  The SDK's v1 encoder then splices the value. Measured on a `compactionNotice`-shaped struct, plain, with `<`, `&` and
  U+2028, and with an invalid byte: the SDK's encoding of the Go value and its splice of gobble's v2 encoding were
  **identical, byte for byte**. The payload structs tag their optional members `omitzero` (`retryNotice`,
  `acpserver/prompt.go`; `compactionNotice`, `acpserver/autocompact.go`), which means the same in v1 and v2, so the
  members sent do not change.
* **D14. gobble's JSON-RPC error data is encoded by gobble.** 34 calls in 10 files build it as a `map[string]any` for
  the SDK's error constructors, whose `data` is `any`:
  * `acpserver/agent.go` (12), `acpserver/extensions.go` (6), `acpserver/commands.go` (3), `acpserver/prompt.go` (3),
    `acpserver/auth.go` (2), `acpserver/compact.go` (1), `acpserver/personality.go` (1), `acpclient/conn.go` (1);
  * `acpclient/acptest/client.go` (2) and `acpclient/acptest/script.go` (3).

  Every value is a string, except one test-support integer (`"update": i`, `acpclient/acptest/script.go:159`).
  `jsonx.Strings(map[string]string) jsontext.Value` writes such an object, its keys sorted and each value through
  `jsonx.Quote`, so it cannot fail. The integer is formatted with `strconv.Itoa`. A map of strings the SDK encodes is
  sorted by v1, and its strings re-escaped when spliced, so the wire does not change.
* **D15. `_meta.gobble` is typed, and encoded and decoded by gobble.** It is gobble's own schema in ACP's
  extensibility slot. ACP's `Meta` fields are the SDK's `map[string]any`, so gobble puts a `jsontext.Value` under its
  key, which the SDK's v1 encoder splices, and reads its key's value back through `jsonx`:
  * **eight writers,** each becoming a typed struct through `jsonx.Value`: `acpserver/question.go:124`,
    `acpserver/agent.go:385`, `:794` and `:1055`, `acpserver/commands.go:685`, `acpclient/sessions.go:79`, and
    `acpclient/conn.go:229` and `:477`;
  * **seven readers,** each decoding one member at a time, so that "a missing or mistyped one is none" holds as it does
    today (`acpclient/conn.go:490-491`): `acpclient/conn.go:310` and `:492-526` (`gobbleMarks`),
    `acpclient/sessions.go:149`, `acpserver/grants.go:138`, `acpserver/question.go:161`, `acpserver/mentions.go:120`,
    and `acpserver/config.go:404` (`readNewSessionMeta`);
  * the SDK has already decoded `_meta` with v1 into Go values, numbers as `float64`. A reader re-encodes its member
    with `jsonx.Marshal` and decodes it into its field with `jsonx.Unmarshal`. No member gobble defines holds a number
    past 2^53;
  * `jsonx` gains two generic helpers for this: `SetMember(meta map[string]any, key string, v any) (map[string]any,
    error)` and `Member[T any](meta map[string]any, key string) (T, bool)`. The second reports a missing or mistyped
    member as not present.

### The move off v1, and the bytes a script reads

* **D16. `translate` encodes only when a sink asks,** through `jsonx.Marshal` with `jsontext.EscapeForHTML(true)` and
  `jsontext.EscapeForJS(true)` (Q4).
  * `Update.Params` is read in one place, `stream-json` (`internal/cli/print.go:113`). Today every update of every mode
    is encoded for it. The line mode, the TUI and `task` stop paying for that.
  * The update lines' bytes do not change. `Update.Params` is documented as "the session/update params object as the
    SDK encodes it" (`acpclient/update.go:103-104`), and with the two options v2 matched v1 on all 21 samples (0038).
  * gobble's own `stream-json` lines, the header and the result, stay as `writeJSON` writes them today: v2, not
    HTML-escaped. One stream keeps two escaping styles, by this decision and documented in `print.go`, not by accident.
    No byte a script reads changes.
* **D17. The local files and the providers output:**
  * `internal/cli/choice.go`'s `recent-models.json` and kept listing move from v1 to the configuration policy, with
    `jsontext.WithIndent("  ")` where indented today. A file v1 wrote still loads: the fields are tagged, `time.Time`
    is RFC 3339 either way, `null` decodes to a nil slice, and v1 never wrote invalid UTF-8. A new file writes `[]`
    where v1 wrote `null`;
  * `internal/cli/providerscmd.go`'s `gobble providers --json` moves to `jsonx.Marshal` with `WithIndent("  ")`. It stops
    HTML-escaping `<`, `>` and `&` in its strings. `providerRow` holds only strings, and its rows slice is never nil.
* **D18. `internal/archtest/check.go` reads `go list -json` with a `jsontext.Decoder`** and `jsonx.Options()`, decoding
  each object until `io.EOF`. `go list` writes exact field names, which `modPkg`'s untagged fields match.

### #9's hatch

* **D19. No `RejectUnknownMembers` on ACP payloads** (#7's U2).
  * The SDK decodes ACP's own methods, its types keep unknown members in their `Extra` and `Unknown` fields, and ACP
    reserves every name.
  * gobble's own `_gobble/*` params stay lenient too. magic-cli-remote may read them, and a newer client's field must
    not make an older agent answer `-32602`.
* **D20. Raw JSON is `jsontext.Value`; binary is `[]byte`.** No wrapper type is introduced, as #9's D1 says.
  * Raw JSON enters a struct through `jsonx.From`, `jsonx.Value` or `jsonx.Quote`, so no field holds an empty non-nil
    value (F1). `agent/agent.go:771` and `:951` use `jsonx.From`. `tool/builtin/edit.go:149` keeps its own `IsValid`
    check, and `acpserver/history.go:92` its fallback. Literal values such as `mcpclient`'s `jsontext.Value("{}")` are
    never empty, and stay.
  * Tests and diagnostics compare `len(v)` and `string(v)`, never `v.String()` (F3). `session/doc.go` says so.
* **D21. `null` and absent stay distinct, pinned by tests** on `session.Entry.Replacement` and `Message.ExitCode` (#9's
  U3):
  * absent is left out, and read back absent;
  * `null` is kept, and read back `null`;
  * an object round-trips;
  * an empty non-nil value planted on the field fails the marshal, and through `jsonx.From` is left out.

  `llm.Content.Data` stays `[]byte`, base64, pinned by a test. No `json.WithMarshalers` override is set on `[]byte`,
  since it would recode that field (#9's D3).
* **D22. Not in this record:**
  * the agent's twenty `_gobble/*` methods through `acp.Handle`. It changes a decode failure's error data from `reason`
    to `error`, which is a wire change of its own;
  * #7's U3: an `fsync` policy and an entry size cap for the session store, both durability decisions;
  * the wire's 10 MiB frame bound for diffs and `rawInput`, for 0037's resource bounds;
  * acp-go-sdk2's own `acptest` (0035-MADR D18);
  * acp-go-sdk2 `v1.29.0` and `OverflowBlock`, which is [gobble-cli#8](https://github.com/maccavelli/gobble-cli/issues/8)'s.
    #7's U4, lockstep with magic-cli-remote, is met at `v1.28.0`;
  * a change to `updates.send`'s rule that one failed send ends the turn's updates. With D9, no known input makes a send
    fail but a closed connection, for which the rule is right;
  * the code acp-go-sdk2 answers for a local encode failure, `-32602 Invalid params`, which blames the client. That is
    the SDK's to change, and it is noted for the owner's acp-go-sdk2 research;
  * 0038's other requests of acp-go-sdk2, an accessor for a handler's raw params so that `stream-json` could pass the
    agent's own bytes through, and a README sentence on the `RawMessage` alias. Neither blocks gobble;
  * 0038's `bytes.Lines` for reading a session file (`session/jsonl/store.go:172-216`), a performance change, for 0037's
    performance category;
  * a second pass over a large extension result: v2 encodes it, and the SDK's v1 validates and compacts it again
    (0038, section 7, item 3). It is measured only if it shows in a profile.

### The documents and the guards

* **D23. The docs** (#9's U5):
  * `session/doc.go` gains the hatch's table: raw JSON is `jsontext.Value`, binary is `[]byte`, empty is absent, `null`
    is kept;
  * `jsonx`'s package documentation states the two policies and which data each covers;
  * `docs/architecture.md` gains `jsonx` in its package notes, in present tense;
  * [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md)'s package map gains `jsonx`, by a dated
    amendment made with this record's first phase.
* **D24. The guards,** each seen failing on a planted break first:
  * **`depguard` denies `encoding/json$`** in every file, tests included, beside the existing `logging` rule
    (`.golangci.yml:41-53`). The allowlist is empty: no call in gobble needs v1's exact semantics. 0038's scratch trial
    of this rule reported exactly the 20 importers;
  * **`forbidigo` forbids v2's `Marshal`, `MarshalWrite`, `MarshalEncode`, `Unmarshal`, `UnmarshalRead` and
    `UnmarshalDecode` outside `jsonx/`,** in non-test code. `.golangci.yml:56-64` already forbids calls, with
    `analyze-types: true`. The exact pattern is settled when it is first seen failing. Tests may call v2 directly, since
    some compare bytes as v2 writes them;
  * **`internal/archtest` gains two rules:** `jsonx` imports no gobble package; and a `[]byte` struct field with a
    `json` tag is allowed only where listed, which today is `llm.Content.Data` (#9's U6). The second reads struct tags
    with `go/parser` over the packages `go list` reports. Rule 1 today only stops contract packages importing outside
    SDKs (`internal/archtest/check.go:213-231`), so contract packages may import `jsonx` with no change.

### The order

* **D25.** Each phase is its own commit, with `make preflight`, `make race` and the host gates green, and each new test
  or guard seen failing first on a scratch copy. Phase 1 runs first, before 0035-PLAN's remaining phases (Q1).
  1. **The crash and break fix.** `jsonx` with D1 to D3, then D5 to D10 and D20. That covers the panics, the discarded
     errors with `SetText` and `Convert`'s change, the extension results, the splice and the handlers' params. Its tests
     are Confirmation 1 to 7, 13, 15 and 16. It can land alone.
  2. The configuration policy at each configuration site (D3, D4's configuration rows), with Confirmation 8, and
     the duplicate-name gates and `gobble config validate`'s JSON files (D3, Q6), with Confirmation 22.
  3. The renames (D11).
  4. Everything gobble defines, on v2: `ExtensionMux` with gobble's own decoding (D12), the payloads gobble sends
     (D13), its error data (D14) and `_meta.gobble` (D15), with Confirmation 18 to 21.
  5. `translate` (D16), with Confirmation 9.
  6. The local files and the providers output (D17), with Confirmation 10.
  7. The remaining data sites of D4, and the ten test files that import v1:
     * `acpclient/compaction_test.go`, `personality_test.go`, `retry_test.go` and `sessions_test.go`;
     * `acpserver/autocompact_test.go`, `convert_test.go` and `retry_test.go`;
     * `internal/cli/models_test.go`, `personalitycmd_test.go` and `providerscmd_test.go`.

     `acpserver/autocompact_test.go:198-201` compares two maps by their v1 bytes, and becomes `reflect.DeepEqual`.
     `acpserver/convert_test.go:17` encodes an SDK value with v1, to mean "as the SDK's encoding/json writes the update".
     It calls the value's own `MarshalJSON` instead, which is the SDK's encoding, with no v1 import: `*acp.SessionUpdate`
     has that method, as defect 3's error names it. The decodes into `map[string]any` move to v2.
  8. `archtest` (D18), the `inline` tags (D11), the hatch's tests (D21), and the docs (D23).
  9. The guards (D24), last, when nothing they forbid is left.

### Consequences

* Good, because a non-UTF-8 byte in a file, a process's output, a file name or a context file no longer ends gobble, and
  no longer empties a saved message.
* Good, because a cut-off tool call no longer fails the prompt or hides the rest of the turn from the client.
* Good, because a resumed, forked or cloned session holds the same text the model saw: U+FFFD for each invalid byte.
* Good, because configuration stays strict. A malformed `mcp.json` or `auth.json` is refused with its error, and a
  credential is never altered.
* Good, because one package decides gobble's JSON behaviour, and the guards keep it so.
* Good, because every JSON value gobble defines, its files, its `_gobble/*` payloads, its `_meta.gobble`, its error
  data and its output, is encoded and decoded with v2 by gobble's code.
* Neutral, because v1 remains in the process inside the SDKs: acp-go-sdk2 for ACP's envelope and spec types (21 files),
  go-llmprovider-sdk (41 files against 2 on v2), and the MCP go-sdk. Each is its owner's to move. acp-go-sdk2's is for
  the owner's research, and does not block this record.
* Good, because `stream-json`'s bytes, ACP's bytes and the `_gobble/*` wire do not change.
* Good, because the line mode, the TUI and `task` stop encoding every update for an output only `stream-json` reads.
* Good, because a session line or a `_gobble/*` request with a lone surrogate escape is read, where today it is skipped
  or refused.
* Neutral, because a file's invalid bytes still reach the model as the provider SDK writes them, U+FFFD. gobble does
  not guess an encoding such as Latin-1.
* Neutral, because maps are written sorted. Nothing reads their order, and each output is reproducible.
* Neutral, because a state file keyed by a path refuses a path that is not UTF-8, as it does today. `trust.json` is a
  `map[string]bool` (`permission/trust.go:141`), and `permissions.json` a `map[string][]StoredRule`
  (`permission/store.go:63`). Both already sort their keys and return an encode error (`permission/trust.go:241`,
  `permission/store.go:180`).
* Bad, because the data repair is lossy. A Linux working directory whose name is not UTF-8 is saved in a session header
  with U+FFFD, so `Store.List`'s comparison with the real directory (`session/jsonl/store.go:247`) never matches it, and
  that directory's sessions are not listed. Today such a session cannot be created at all, since its header fails to
  encode.
* Bad, because an `mcp.json` that `JSON.parse`-based tools accept with a duplicate name is refused by gobble until
  the duplicate is removed. The error names the file, the path and the name, and `gobble config validate` finds it
  first.
* Good, because no edit of `mcp.json` deletes a value any more, and `gobble config validate` gates every JSON
  configuration file as it gates `settings.yaml`.
* Bad, because `gobble providers --json` stops HTML-escaping `<`, `>` and `&`. JSON readers see the same values; a byte
  comparison does not.
* Bad, because `recent-models.json` and the kept listing write `[]` where v1 wrote `null`. gobble reads both.
* Bad, because a map with many keys costs up to twice as much to encode. gobble's hot paths hold none.
* Bad, because it is nine phases, each with three-host gates. Only the first is urgent.

### Confirmation

Each test below is seen failing first on a scratch copy, as stated, and the execution record quotes the failure.

1. **A turn reading a Latin-1 file completes.** Through the real agent (`promptAgent` and a scripted model), a `read` of
   `caf\xe9 au lait` returns a result, the scripted model's next request carries it, and the turn ends `end_turn`. The
   saved tool result holds `caf`, U+FFFD and ` au lait`. Seen failing at `31e3c1c`: the panic quoted in the context.
2. **The same for `bash`** running `printf 'a\377b\n'`, and for `grep` over the file. Seen failing: `panic: jsontext:
   invalid UTF-8`.
3. **A Latin-1 `AGENTS.md`** gives a baseline entry with U+FFFD and a first prompt that completes. Seen failing: the panic
   in `prompt.mustJSON`.
4. **`toolResultMessage`** of `\xe9t\xe9` stores a non-empty `content`; a `SetBlocks` given a `jsontext.Value` holding
   bad JSON returns an error that reaches the writer and the prompt's answer. Seen failing: content 0 bytes, no error.
5. **`jsonx.Quote` never fails,** by a fuzz target over arbitrary bytes. Seen failing with D2's `AllowInvalidUTF8`
   removed.
6. **A cut-off tool call:** a scripted model's `read` with arguments `{"path":"a.tx`, then a text reply. The prompt ends
   `end_turn`; the client sees the `tool_call` with `rawInput` the string `{"path":"a.tx`, the result, and the reply.
   Seen failing at `31e3c1c`: the `-32602` error and three updates, as in the context.
7. **A `_gobble/*` request whose params hold a lone surrogate escape** is answered as the same request without it is:
   "unknown session" for an unknown id. Seen failing: `-32602 invalid surrogate pair`.
8. **Configuration is strict:**
   * an `auth.json`, a `models.json` and an `mcp.json` holding an invalid byte or a lone surrogate escape are each
     refused with their error;
   * `SetKey` given a key holding an invalid byte returns an error and leaves `auth.json` unchanged.

   Each is seen failing by switching its site to the data policy on a scratch copy, which repairs instead. A duplicate
   name is refused under both policies, so a test pins it, and `mcp.json`'s editor keeps its JSON.parse rule. Both are
   seen failing with the opposite option planted.
9. **`stream-json`'s bytes are unchanged:** a new golden case whose reply holds `<`, `&`, U+2028 and U+2029 pins both
   styles, the update lines escaped and the result line not. Seen failing with `EscapeForJS` dropped from `translate`.
10. **`recent-models.json` and the kept listing, as v1 wrote them,** `null` included, load.
11. **D21's cases** and `llm.Content.Data`'s base64. The planted empty non-nil value is seen failing with the EOF error.
12. **The guards:**
    * a planted v1 import fails `depguard`;
    * a planted `json.Marshal` outside `jsonx` fails `forbidigo`;
    * a planted `jsonx` import of `session` fails `archtest`;
    * a planted `[]byte` field with a `json` tag fails `archtest`.
13. **A session line with a lone surrogate escape loads,** and is not counted as malformed. Seen failing: it is skipped.
14. **No change elsewhere:** `session.golden`, `print-json.golden` and `print-stream-json.golden` are unchanged, and
    0030-MADR D5's benchmarks hold (`make perf`).
15. **`jsonx`'s own tests,** #9's U1 table and the splice:
    * `From(nil)` and `From([]byte{})` are absent; `From("null")` is present and null; `From("{}")` writes `{}`;
      `From("{")` is stored, and a parent's marshal reports it;
    * `Value` of a struct is never empty non-nil;
    * `Splice` of each row of the splicing table: valid and invalid-byte and lone-surrogate values unchanged, a cut-off
      value as a JSON string, empty as absent. Seen failing with `Splice` the identity, on the cut-off row;
    * `SetText` of invalid UTF-8 writes one text block with U+FFFD. Seen failing with `SetBlocks` in its place.
16. **An empty `rawInput` is left out, and a valid one carried as it is,** through `Splice`. `TestRawInputOnTheWire`
    (`acpserver/convert_test.go:12-17`) pins both today: "A tool call with no input sends no rawInput key, and one with
    input carries it verbatim". Its cases and assertions are unchanged. Its one `jsonRaw` call becomes `Splice`, since
    `jsonRaw` goes (the PLAN's deviation 2).
17. **`Convert` and the `inline` tags change nothing a reader sees:** the replay, serialization and compaction tests,
    and the `_gobble/*` and personality tests, pass unchanged.
18. **The client decodes `_gobble/*` notifications with v2:** a `_gobble/retry` whose cause holds a lone surrogate
    escape reaches the session's callback with U+FFFD; one whose `attempt` is a string is logged and ignored, as a bad
    notification is today. Seen failing with `jsontext.AllowInvalidUTF8(false)` planted in the handler's decode on a
    scratch copy: the notification is refused, and the callback is not called.
19. **The wire does not change:** `session.golden`, the agent's recorded ACP session, is unchanged, and a test compares
    `updates.notify`'s frame for each notification type with the frame the SDK writes from the Go value, a text with
    `<`, `&`, U+2028 and an invalid byte among them. Seen failing with `jsontext.AllowInvalidUTF8(false)` planted in
    `notify`'s encoding on a scratch copy: the invalid byte's frame is not sent.
20. **`jsonx.Strings`** writes valid JSON, its keys sorted, equal to `json.Marshal` of the same `map[string]string`
    under D2, by a fuzz target. The `errText` tests in `acpserver` pass unchanged.
21. **`_meta.gobble`:** each reader's existing tests pass unchanged, a mistyped member included, and a test plants one
    mistyped member beside a good one and reads the good one. Seen failing with a whole-struct decode in place of
    `Member`.
22. **Duplicates are refused at every gate:**
    * an `mcp.json` with two servers named `github` fails `Load` with one error naming `mcpServers` and `github`, and
      loads no server from it; a server whose `command` is set twice fails the same way;
    * `gobble mcp list` on that file prints the `config error:` and exits 1;
    * `mcpclient.Add` and `Remove` on that file return the error and leave the file byte for byte as it was. Seen
      failing at `31e3c1c`: `Add` deleted the first `github`, as measured above;
    * `gobble config validate` reports a duplicate in each of `mcp.json`, `models.json`, `auth.json`,
      `permissions.json` and `trust.json` with its file, line and path, prints no value, and exits 1. A clean file
      gives no finding and exit 0;
    * gobble writes no duplicate: a fuzz target encodes random session entries, embedded unknown members included,
      through `jsonx`, and every output passes `jsontext.Value.IsValid()`, which refuses a duplicate.

    Seen failing with `AllowDuplicateNames(true)` left in `members`, on a scratch copy, for the first three.

## Pros and Cons of the Options

### A. Fix only the panics

* Good, because it is the smallest change: a `recover` in `runTool`, or a `utf8.ValidString` check in `quote`.
* Bad, because a `recover` turns a crash into a failed tool call over text the model could have read: the provider SDK
  would have repaired it.
* Bad, because the silent losses (`SetBlocks`, the compaction sites), the extension failures and defect 3 stay.
* Bad, because it leaves 43 files choosing their own options, so the next unchecked site fails the same way.
* Bad, because #7's and #9's goals stay open.

### B. Return gobble's encode sites to v1, or to v2 with `DefaultOptionsV1`

* Good, because v1's behaviour is known, and the SDKs already use it.
* Neutral, because at Go 1.27 v1 is v2 underneath, so speed is the same.
* Bad, because it brings v1's other legacy behaviour back on everything gobble writes: case-insensitive names, HTML
  escaping, and `omitempty` on bools.
* Bad, because configuration would be repaired, not refused: a credential with an invalid byte would be written altered.
* Bad, because `DefaultOptionsV1` lives in `encoding/json`, so the v1 import stays, against #7's goal and Go's
  recommendation.
* Bad, because defect 3 stays: v1 refuses to splice a raw value that is not JSON, as the SDK shows.

### C. Repair text where it enters

* Good, because the model would see a repaired copy that gobble chose, not the provider SDK's.
* Bad, because `strings.ToValidUTF8` replaces a run of invalid bytes with one U+FFFD, where the provider SDK, and so the
  model, has one for each byte. The saved copy would differ from the model's.
* Bad, because every source must be found: `read`, `grep`, `find`, `bash`, `powershell`, file names in errors, context
  files, templates, MCP results, and any added later. A missed one still crashes.
* Bad, because defect 3, a raw value that is not JSON, is not text, and stays.
* Neutral, because it could be added to D later, for a source that needs gobble's own repair.

### D. One encoder package, with two policies, a checked splice, the move off v1, and guards

* Good, because one place, `jsonx`, holds the repair, the strictness, the map order and the splice. The guards make it
  the only place.
* Good, because the data repair matches the SDKs' byte for byte, and configuration stays strict.
* Good, because it carries out #7 (U1, U5) and #9 (F1 to F4, U3, U5, U6), and keeps their non-goals.
* Neutral, because `jsonx` is a new package, and 0004's map changes.
* Bad, because it is a wide change: the guards touch every package's JSON calls. The order puts the urgent fix first,
  and the guards last.

## More Information

### The owner's answers, 2026-10-31

* **Acceptance:** "accept the madr. write the detailed, actionable, and comprehensive plan."

* **Q1. The order:** "run the crash fix first". Phase 1 (D25) runs before 0035-PLAN's remaining phases, as soon as this
  record and its PLAN are approved.
* **Q2. Configuration:** "config files should be strictly decoded, and they should be yaml". Asked which files that
  covers, the owner chose "settings.yaml only (Recommended)": `settings.yaml` is the configuration file, already YAML
  and read strictly; the JSON files stay JSON by 0018-MADR D2, decoded strictly (D3). The options not taken were moving
  gobble's own JSON files to YAML, and moving every configuration file to YAML. Each would amend 0018-MADR, in a record
  of its own.
* **Q3. Map order:** "Sort everywhere (Recommended)" (D2). The options not taken were sorting only for files and
  standard output, and never sorting.
* **Q4. Bytes a script reads:** "Keep today's bytes (Recommended)" (D16). The options not taken were escaping the whole
  of `stream-json` as the SDK does, and escaping none of it.
* **Q5. The package:** `jsonx`, at the module root (D1).
* **Q6. Duplicate names:** first answered "Accept, and warn (Recommended)" for `mcp.json`. The owner then asked
  why duplicates were not simply prevented by validations and gates. Measured: v2's encoder cannot write one, so
  gobble never produces one; only another writer can. The owner chose "Refuse, plus validate gate (Recommended)" (D3):
  no exception for `mcp.json`, a duplicate refused with an exact error, and `gobble config validate` extended to the JSON
  configuration files. The warnings design, `Config.Warnings` and a `"warnings"` member, was dropped.
* **The scope:** "we need to move everything to jsonv2 for gobble-cli". The second assessment had left the client's
  `_gobble/*` notifications to the SDK's v1 decoding, as 0038 recommended. The owner rejected that. D10 and D12 were
  rewritten, and D13 to D15 added, so that every JSON value gobble defines is encoded and decoded by gobble with v2.

### Relationship to the issues and records

* **gobble-cli#7:** U1 is D4 and D11 to D18; U5 is D24; U2 is declined (D19); U3 and U4 are D22's.
* **gobble-cli#9:**
  * its `jsonx` is D1, with `Quote`, `Value` and `Splice` added for the defects, and the two policies;
  * F1 is D20; F2 is answered by D2 (`PreserveRawStrings` nowhere); F3 is D20; F4 is D11;
  * U3 is D21; U5 is D23; U6 is D24's `archtest` rule;
  * its open questions are answered:
    * `From` does not validate (D1). `Splice`, at the one seam where an invalid value breaks an encoder, does;
    * `PreserveRawStrings` is used nowhere, since 0038 measured that the two escape options alone match the SDK;
    * `jsonx` at the root (Q5).
* [0038-REPORT-json-v2-standardization-and-sdk-surface.md](../reports/0038-REPORT-json-v2-standardization-and-sdk-surface.md):
  its order is D25's phases 3 to 9. This record adds phase 1, the two policies, the splice, and D16's decision on
  `writeJSON`.
* [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md): its D2 and D3 stand. D3 here applies its
  strictness to the JSON files 0018 kept.
* [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): amended for `jsonx` (D23, D24).
* [0037-MADR-hardening-integrity-and-performance.md](0037-MADR-hardening-integrity-and-performance.md): receives the wire
  frame bound and U3's durability items (D22).
* [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md):
  its remaining phases wait for this record's phase 1 (Q1).

### Every unit of work, traced

Each item the two issues, 0038 and this record's evidence name, with the decision that settles it, the phase of D25
that carries it out, and the Confirmation item that proves it. "Declined" and "elsewhere" are decisions too.

| Item | Decision | Phase | Proof |
| :--- | :--- | :--- | :--- |
| Defect 1: invalid UTF-8 panics a tool result, a context file, `reasoningData` | D2, D5, D6 | 1 | 1, 2, 3, 5 |
| Defect 2: `SetBlocks` errors discarded; extension results fail whole | D7, D8 | 1 | 4, 15, 17 |
| Defect 3: a cut-off tool call fails the prompt | D9 | 1 | 6, 15, 16 |
| The handlers' params stricter than the SDK | D10 | 1 | 7 |
| A session line with a lone surrogate escape is skipped | D2 | 1 | 13 |
| #7 U1: the single-marshaler sweep | D4, D11 to D18 | 3 to 7 | 9, 10, 14 |
| #7 U1's acceptance: old fixtures load | D17 | 6 | 10 |
| #7 U2: strict decoding at the ACP boundary | declined, D19 | none | none |
| #7 U3: store hardening | elsewhere, D22 (0037) | none | none |
| #7 U4: SDK lockstep | met at `v1.28.0`; `v1.29.0` is #8's (D22) | none | magic-cli-remote's `go.mod:11` |
| #7 U5: the guard | D24 | 9 | 12 |
| #7's open question: the allowlist | D24: empty | 9 | 12 |
| #9 U0: the records | this record and its PLAN | before 1 | `make check-records` |
| #9 U1: `jsonx` | D1 | 1 | 5, 15 |
| #9 U2: ad hoc constructors replaced | D6, D20 | 1 | 15 |
| #9 U3: absent against `null` pinned | D21 | 8 | 11 |
| #9 U4: the ACP seam | D9, D11 | 1, 3 | 6, 16 |
| #9 U5: the docs | D23 | 8 | review |
| #9 U6: the `[]byte` guard | D24 | 9 | 12 |
| #9 F1: an empty non-nil value | D20 | 1 | 11, 15 |
| #9 F2: a spliced value reformatted | D2: `PreserveRawStrings` nowhere | 1 | 14 |
| #9 F3: `String()` of an absent value | D20 | 8 | review of `session/doc.go` |
| #9 F4: the `RawMessage` spellings | D11 | 3 | 12 (`depguard`) |
| #9 acceptance 1: `jsonx`'s table, the EOF case seen failing | D1, D20 | 1 | 11, 15 |
| #9 acceptance 2: `Replacement` and `ExitCode` | D21 | 8 | 11 |
| #9 acceptance 3: the packages' tests pass | all | each | 14 |
| #9 acceptance 4: `llm.Content.Data` base64 | D21 | 8 | 11 |
| #9 acceptance 5: no new v1 import | D24 | 9 | 12 |
| #9 acceptance 6: `jsonx` a leaf, seen failing | D24 | 9 | 12 |
| #9 acceptance 7: the docs | D23 | 8 | review |
| #9 open question 1: validate in `From` | D1: no; `Splice` does at the seam | 1 | 15 |
| #9 open question 2: `PreserveRawStrings` | D2: nowhere | 1 | 9 |
| #9 open question 3: the package's name | D1, Q5 | 1 | review |
| 0038 step 1: the renames | D11 | 3 | 12 |
| 0038 step 2: `ExtensionMux` | D12 | 4 | 14 |
| 0038 step 3: `translate` | D16 | 5 | 9 |
| 0038 step 4: the local files and the providers output | D17 | 6 | 10 |
| 0038 step 5: the tests | D25 phase 7 | 7 | 14 |
| 0038 step 6: `archtest` | D18 | 8 | 14 |
| 0038 step 7: the guard | D24 | 9 | 12 |
| 0038 step 8: `acp.Handle`; the `inline` tags; `bytes.Lines` | D22; D11; D22 (0037) | none; 8; none | none; 17; none |
| 0038 section 10: the SDK's requests | D22 | none | none |
| Q1 to Q5, the owner's answers | D25, D3, D2, D16, D1 | as each | as each |
| The owner's directive: everything gobble-cli defines on v2 | D10, D12 to D15 | 1, 4 | 7, 18 to 21 |
| `_gobble/*` notifications the client receives | D12 | 4 | 18 |
| `_gobble/*` payloads gobble sends | D13 | 4 | 19 |
| JSON-RPC error data | D14 | 4 | 20 |
| `_meta.gobble` | D15 | 4 | 21 |
| Duplicate names: never written, refused on read, `gobble config validate` (Q6) | D3 | 2 | 22 |

"Review" marks a document change, which no test proves; the PLAN's execution record quotes it.

### How it was read

* **Source:** each file and line cited, at `31e3c1c`. A scratch script resolved every `path:line` citation in this
  record against the tree and printed the cited lines beside it; each was read against its claim.
* **Scans:** scratch scripts listed:
  * each non-test Go file's JSON imports and calls, by file and line (D4's table);
  * every encode site, and every comment saying an encode cannot fail;
  * every non-literal `jsontext.Value` conversion;
  * every `[]byte` field with a `json` tag;
  * every file name gobble reads or writes as JSON or YAML.
* **Probes, in scratch copies of the tree with one test file added:**
  * `read`, `bash` and `grep` on invalid UTF-8, through `tool/builtin`'s own test helpers;
  * a turn through `promptAgent` whose scripted model reads a Latin-1 file;
  * a first prompt with a Latin-1 `AGENTS.md`;
  * `toolResultMessage` with invalid UTF-8;
  * a turn whose scripted model's tool call has arguments valid, cut off, and empty, the client's updates counted;
  * `_gobble/get_entries` with and without a lone surrogate escape in its params;
  * `stream-json` with a reply holding `<`, `>` and `&`;
  * `translate`'s v1 and v2 encodings of a decoded `acp.SessionNotification`;
  * `session.Entry`'s `Replacement`: nil, empty non-nil, `null` and `{}`, written and read.
* **Probes, in scratch modules at Go 1.27.3:**
  * v1 and v2 on invalid UTF-8, lone surrogate escapes, duplicate names, case, HTML characters, nil slices and map order;
  * `PreserveRawStrings` with and without `AllowInvalidUTF8`;
  * `IsValid` and v1's splice of valid, cut-off, invalid-byte, lone-surrogate and empty raw values;
  * `testing.Benchmark` of `Deterministic` on structs and maps;
  * `jsontext.AppendQuote` against `Marshal` under `AllowInvalidUTF8`, over 200,009 inputs.

  In each probe, a backslash escape in a JSON input was built in code. A `\u` sequence typed into a probe's source
  can be decoded before it is written, which made one early probe of F2 report a wrong result.
* **External:**
  * magic-cli-remote's `go.mod` and `internal/provider/acpagent/acpagent.go`, read through the GitHub API;
  * acp-go-sdk2 `v1.28.0` and `v1.29.0`, and go-llmprovider-sdk `v1.24.1`, from the module cache;
  * `go doc` at Go 1.27.3.
