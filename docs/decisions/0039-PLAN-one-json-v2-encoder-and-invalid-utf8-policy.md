---
status: in-progress
date: 2026-10-31
associated-madr: "0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement encoding and decoding gobble's JSON through one v2 package, under an invalid-UTF-8 policy that cannot crash or lose data

Associated MADR: [0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md)

## Goal

Carry out the MADR's D1 to D25 in its order (D25):

* Every JSON value gobble-cli defines is encoded and decoded by gobble's code through one package, `jsonx`, on
  `encoding/json/v2`. No file in the module imports `encoding/json` (v1).
* Data, meaning protocol, session, tool and provider data and CLI output, is repaired as v1 repaired it: U+FFFD for
  each invalid byte. Configuration and state files are strict: refused with an exact error, never repaired, and never
  holding a duplicate name.
* The three defects the MADR measured are gone:
  * non-UTF-8 text panics the agent;
  * encode errors are discarded and a saved message is emptied;
  * a cut-off tool call fails the prompt.
* `depguard`, `forbidigo` and two `internal/archtest` rules keep it so.

The plan is complete when every item of the MADR's Confirmation (1 to 22) is met, as recorded in the execution record,
and P10's close-out is done.

## Scope

**Base.** `main` at `31e3c1c`, Go 1.27.3, acp-go-sdk2 `v1.28.0`, go-tui-lib/v2 `v2.7.0`, go-llmprovider-sdk `v1.24.1`,
go-selfupdate-lib `v1.23.1`. If P0 lands on a later commit, that commit is the base, and each phase's line numbers are
re-read at its start (C2).

**In scope:** the MADR's D4 table, plus the files its D11 to D15 and D17 to D24 name. Each phase lists its own files,
and a file outside a phase's list is a deviation (C8).

**Out of scope,** by the MADR's D19 and D22:
* `RejectUnknownMembers` on ACP payloads;
* the agent's twenty `_gobble/*` methods through `acp.Handle`;
* `fsync` and entry size caps for the session store;
* the wire's 10 MiB bound for diffs and `rawInput`;
* acp-go-sdk2's `acptest`, and acp-go-sdk2 `v1.29.0` with `OverflowBlock` (gobble-cli#8);
* `updates.send`'s first-failure rule;
* `bytes.Lines` in the store;
* everything inside the SDKs, which stay v1.

**The order against other plans** (the MADR's Q1): P1 runs before
[0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md)'s
remaining phases and its deviation 13. P2 to P10 follow in order. 0035's phases, and
[0037-PLAN-permission-and-file-tool-hardening.md](0037-PLAN-permission-and-file-tool-hardening.md)'s P3 onward, may
interleave with P2 to P10 on the owner's word, each phase a commit of its own.

## Conventions every phase follows

* **C1. Start state.**
  * The tree is clean, apart from the records named in PQ1.
  * `HEAD` is the previous phase's commit.
  * `git rev-parse --path-format=absolute --git-path hooks` is the global hooks directory (checked once, at P0).
  * `go version` is `go1.27.3`.
* **C2. Line numbers.** Each phase re-reads the lines it cites at its start, since earlier phases move them. A cited line
  that no longer holds what the step says is a deviation only when the code differs in substance, not in place.
* **C3. Scratch work.**
  * Every probe and every deliberately broken copy lives in the session's scratch directory, never in the tree.
  * A broken copy is the tree's `git ls-files -co --exclude-standard` copied to `<scratch>/break-<phase>-<n>/`, the
    break applied there by an exact, asserted replacement (a Python script, old text present exactly once), and the
    named command run there.
  * The execution record keeps the command, its exit status and the failing lines.
* **C4. Escapes in test sources.** A JSON escape in a test input, such as a lone surrogate, is built in code: a
  backslash from `string(rune(92))`, or Go's own string escapes. A `\u` sequence typed into a source through some
  tools is decoded before it is written, which made one of the MADR's early probes wrong.
* **C5. Long output.** Every gate writes its whole output to `<scratch>/<phase>-<gate>.log`, and its exit status is taken
  before any filter.
* **C6. Local gates (G-local),** in this order, each passing:
  1. `make pre-add-check FILES="<every staged .go file>"`;
  2. `make preflight`;
  3. `make race`;
  4. for a phase that touches `internal/cli/...`, `make perf`.
* **C7. Host gates (G-hosts).** On the Linux machine and the Windows laptop, on a copy of the tree, never their own
  checkouts:
  * `go test -count=1 ./...`;
  * `CGO_ENABLED=1 go test -race -count=1 ./...`;
  * `make preflight`, up to `verify-build-metadata`, which needs a commit and is recorded as macOS-only;
  * `make check-records markdownlint`;
  * for a phase that touches `internal/cli/...`, `make perf` on both, and `make probe-conhost` on the Windows laptop.

  The copy recipe is 0031-PLAN's (its execution record). The hostnames are never written.
* **C8. Deviations.** Any failure the phase does not name as expected, a file outside its list, or an API not as written
  here stops the phase. The owner is given:
  * the evidence: file, line, exact failure, and whether it is pre-existing;
  * resolutions, recommended first, each with its cost;
  * what doing nothing means.

  No workaround is offered. The chosen resolution is recorded as a dated deviation here, and in the MADR when a decision
  or fact changes, before work continues.
* **C9. Seen failing.** Every new test, case, fuzz target and guard is first run against a broken copy (C3) and seen to
  fail as the phase says it will. The execution record quotes the failure.
  * For a defect the phase fixes, the broken copy is the phase's base with only the new test added. The failure is the
    defect itself, such as the panic.
  * For a guard, it is the finished tree with the forbidden thing planted.
* **C10. Commit.** At the end of each phase, after G-local and G-hosts pass:
  1. `git add` of exactly the phase's files, staged for the owner;
  2. the owner commits with `git commit --no-edit`;
  3. the phase's execution record and this plan's `docs/README.md` row are updated, in the same commit or the next, as
     the phase says.

  Before the owner pushes, the disclosure guard runs over the outgoing commits (`AGENTS.md`, "Identifiers").
* **C11. Identifiers.** No hostname, account name or absolute path of a real machine goes into a committed file, a log
  excerpt included. Paths are quoted relative to the repository.

## Order

P0, then P1, the crash and break fix, then P2 to P9 in number order, then P10. A phase starts only on the owner's word for
it, or for a named range ("do P1", "do P2–P4", "execute the plan").

## Questions this plan adds

* **PQ1. What P0 commits.** The tree holds three uncommitted records:
  * this pair;
  * the owner's [0038-REPORT-json-v2-standardization-and-sdk-surface.md](../reports/0038-REPORT-json-v2-standardization-and-sdk-surface.md),
    untracked;
  * 0035-MADR's D1 and D14 step 5 amendments of 2026-10-31, with their `docs/README.md` row. These are deviation 13's
    first half, whose PLAN entry is not written yet.

  `docs/README.md` holds rows for all three.
  * **(a) Recommended.** Two records commits. First 0035's amendments, with 0035-PLAN's deviation 13 entry written for
    them; then this pair with 0038. Each commit's `docs/README.md` rows match its records. The cost is writing deviation
    13's entry now.
  * **(b)** One records commit of all three. Faster, but one commit then carries two plans' records.
  * **(c)** This pair and 0038 only, with the 0035 rows held back from `docs/README.md`. That means editing
    `docs/README.md` back and forth, and the tree stays dirty until 0035's commit.
* **PQ2. How `translate` learns that a sink wants `Params`** (MADR D16).
  * **(a) Recommended.** `acpclient.Options` gains `Params bool`. `translate` encodes `Update.Params` only when it is
    set, and print mode sets it for `--output-format stream-json`. It is an added field, which `make apidiff` accepts.
  * **(b)** `Update.Params` becomes a method that encodes on demand. It is cleaner, but removing an exported field is an
    incompatible change that `apidiff` refuses, and it needs a record of its own.
  * **(c)** Keep encoding every update. No change, but every mode keeps paying for an output only `stream-json` reads.

**The owner's answers, 2026-10-31,** given with the plan's approval: "PQ1: 2 commits. PQ2: Recommended: an added
acpclient.Options.Params field, which apidiff accepts."

* PQ1 (a): two records commits, 0035's first, with its deviation 13 entry, and then this pair with 0038.
* PQ2 (a): `acpclient.Options.Params` (P5).

## Implementation Steps

### P0 — the records

Files: this plan; the MADR; `docs/README.md`; by PQ1, 0035's MADR and PLAN, and 0038.

1. C1's hooks check, once.
2. `python3 scripts/check_records.py` and `make check-records markdownlint`: exit 0.
3. Identifier scan over the staged records: none of the local account name, the hostname domain, the hosts' aliases, or
   a real-machine absolute path. Exit 1 from `grep -c`, meaning none, is the pass.
4. By PQ1 (a):
   * write 0035-PLAN's deviation 13 entry: the go-tui-lib `v2.7.1` pin as a new phase P23, and P18 held while the
     MADR's proposed D14 step 5 split into P18a and P18b waits for the owner's review, with
     [go-tui-lib#14](https://github.com/maccavelli/go-tui-lib/issues/14) filed. Stage 0035's records and their
     `docs/README.md` rows; the owner commits;
   * then stage this pair, 0038 and their rows; the owner commits.
5. These are records only: the bootstrap exception (`AGENTS.md`). No source, test, CI or configuration goes in them.

### P1 — the crash and break fix (MADR D1 to D3's core, D5 to D10, D20)

*Amended 2026-10-31, by deviation 1:*
* *`acpserver/personality.go` is added to the files;*
* *step 3 covers all eight callers of `acpserver`'s `mustJSON`;*
* *`jsonx.Fields` builds the four-string `reasoning` value, so `reasoningData` and `signatureData` keep their
  signatures and the stream loop does not change.*

*Amended 2026-10-31, by deviation 2:*
* *`compaction/project_test.go` is added to the files;*
* *`acpserver/convert_test.go`'s one `jsonRaw` call becomes `jsonx.Splice`.*

Files:
* new: `jsonx/doc.go`, `jsonx/jsonx.go`, `jsonx/jsonx_test.go`, `jsonx/fuzz_test.go`;
* `session/session.go`, `session/session_test.go`, `session/jsonl/store.go`, `session/jsonl/store_test.go`;
* `tool/tool.go`, `mcpclient/tool.go`;
* `prompt/sections.go`, `prompt/sections_test.go`;
* `acpserver/sysprompt.go`, `acpserver/history.go`, `acpserver/agent.go`, `acpserver/extensions.go`,
  `acpserver/prompt.go`, `acpserver/question.go`;
* new `acpserver/history_test.go`, and `acpserver/prompt_test.go`, `acpserver/sysprompt_test.go`,
  `acpserver/extensions_test.go`, `acpserver/convert_test.go` (unchanged; it must pass);
* `compaction/serialize.go`, `compaction/estimate.go`;
* `llm/provider/provider.go`;
* `agent/agent.go`;
* `tool/builtin/read_test.go`, `tool/builtin/shell_test.go`, `tool/builtin/search_test.go`;
* [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md), a dated amendment (MADR D23).

1. **`jsonx`** (MADR D1, D2, D3), at the module root. It imports only `bytes`, `encoding/json/v2` and
   `encoding/json/jsontext`. Every function has a doc comment.
   * `Options()` is `json.JoinOptions(jsontext.AllowInvalidUTF8(true), json.Deterministic(true))`.
     `StrictOptions()` is `json.Deterministic(true)` alone, which leaves v2's refusal of invalid UTF-8 and of duplicate
     names in force.
   * `Marshal(v, o...)` is `json.Marshal(v, Options(), o...)`, the caller's options last. `Unmarshal(b, v, o...)`,
     `MarshalStrict` and `UnmarshalStrict` are formed the same way.
   * `Quote(s string) jsontext.Value` returns `jsontext.AppendQuote(nil, s)`'s bytes. Its error means only that
     invalid UTF-8 was replaced (`go doc encoding/json/jsontext.AppendQuote`), which the data policy decides. The doc
     comment says so, and cites the MADR's measurement: 200,009 inputs, equal to `Marshal` byte for byte.
   * `From(b)` is nil for `len(b) == 0`, and `jsontext.Value(b)` otherwise. `Value(v)` is `From` of `Marshal(v)`, with
     `Marshal`'s error.
   * `Splice(v)` is nil when empty; `v` when `v.IsValid(jsontext.AllowInvalidUTF8(true))`; and `Quote(string(v))`
     otherwise.
   * `IsAbsent(v)` is `len(v) == 0`. `IsNull(v)` is `bytes.Equal(bytes.TrimSpace(v), []byte("null"))`.
   * `jsonx/doc.go` states the two policies, which data each covers (the MADR's D4 table), and that raw JSON is
     `jsontext.Value` while binary is `[]byte`.
   * `Strings`, `SetMember` and `Member` come in P4.
2. **`session`** (D7):
   * `(*Message).SetText(s string)`, with no error, sets `Content` to `[`, `{"type":"text","text":`, `jsonx.Quote(s)`,
     `}`, `]`, written with a `bytes.Buffer`. The encoder writes `Block`'s `type` first and `text` second; a test pins
     that `SetText(s)` equals `SetBlocks([]Block{{Type: BlockText, Text: s}})` byte for byte for valid `s`;
   * `SetBlocks` encodes through `jsonx.Marshal`, and `Blocks` decodes through `jsonx.Unmarshal`
     (`session/session.go:225-250`).
3. **No encode panics** (D6):
   * `tool/tool.go:244-249` `quote` and `mcpclient/tool.go:128-134` `quote` are deleted. Their callers call
     `jsonx.Quote`, at `tool/tool.go:228` and `:233`, and `mcpclient/tool.go:122`.
   * `prompt/sections.go:43-49` `mustJSON` is deleted. `BaselineData` and `PatchData` return
     `(jsontext.Value, error)` through `jsonx.Value`, and `ParseData` decodes through `jsonx.Unmarshal` (`:55`). Their
     tests in `prompt/sections_test.go` take the error.
   * `acpserver/sysprompt.go`:
     * `writePrompt` (`:256-268`) passes each error to the writer, through a new `(*writer).fail(err error)` that keeps
       the first error as `add` does;
     * `rebase` (`:285`) returns the error.
   * `acpserver/history.go:46-52` `mustJSON` is deleted.
     * `extra` (`:38-44`) returns `(jsontext.Value, error)`, through `jsonx.Value` of `gobbleMember`.
     * `userEntryMessage` and `assistantEntryMessage` return `(*session.Message, error)`.
     * `toolResultMessage` calls `SetText`, and has no error.
   * `acpserver/agent.go`'s writer calls (`:653`, `:884`, `:886`, `:889`, `:903`) pass each error to `w.fail`.
   * `llm/provider/provider.go:363-376`: `reasoningData` and `signatureData` return `(jsontext.Value, error)` through
     `jsonx.Value`. The stream loop (`:338`, `:340`) yields the error, as it yields a provider's at `:323`, and returns.
4. **No discarded `SetBlocks` error** (D7):
   * `compaction/serialize.go:65`, `compaction/estimate.go:222` and `:262` call `SetText`;
   * `acpserver/history.go:106`'s error is `assistantEntryMessage`'s return (step 3);
   * `compaction/serialize.go:72-75`, `Convert`'s custom case, keeps `Content: m.Content` and drops the `Blocks` and
     `SetBlocks` re-encode. Its three callers read the result through `Blocks` (MADR D7).

     *Amended 2026-10-31, by deviation 3:* the re-encode is Pi's `convertToLlm`, which makes a string content one text
     block. So a stored content that is a JSON string is written with `SetText`, and an array is kept as stored.
5. **Extension results and params** (D8, D10): `acpserver/extensions.go:73` encodes with `jsonx.Marshal`, and `:129`
   decodes with `jsonx.Unmarshal`.
6. **The splice** (D9): `acpserver/prompt.go`'s `rawInput` (`:199-204`) and `jsonRaw` (`:277-282`) are deleted. Their
   callers use `jsonx.Splice`: `acpserver/prompt.go:102`, `acpserver/question.go:151` and `acpserver/history.go:402`.
   `jsontext.Value` is the SDK's `json.RawMessage`, so the calls compile with no conversion. If that leaves
   `acpserver/prompt.go` with no v1 use, its `stdjson` import goes here, not in P3.
7. **The session store** (D2): `session/jsonl/store.go:201` and `:215` decode, and `:316` and `:342` encode, through
   `jsonx`. A line with a lone surrogate escape then loads, where today it is skipped.
8. **Empty raw values** (D20): `agent/agent.go:771` and `:951` build `Args: jsonx.From([]byte(c.Args))`.
9. **0004-MADR amendment** (D23): a dated note that `jsonx` is a leaf package at the module root, which contract
   packages may import and which imports no gobble package. Its `archtest` rule comes in P9.
10. **Tests** (Confirmation 1 to 7, 13, 15 and 16). Each is written first, and seen failing on a scratch copy of the
    base with only the test added (C9):

    | Test | File | Asserts | Seen failing at the base |
    | :--- | :--- | :--- | :--- |
    | `TestTurnReadsInvalidUTF8File` | `acpserver/prompt_test.go` | a scripted model's `read` of `caf\xe9 au lait` returns a result; the next request carries it; the turn ends `end_turn`; the saved tool result's text is `caf` U+FFFD ` au lait` | `panic: jsontext: invalid UTF-8` in `tool.quote` |
    | `TestReadInvalidUTF8`, `TestBashInvalidUTF8Output`, `TestGrepInvalidUTF8File` | `tool/builtin/read_test.go`, `shell_test.go`, `search_test.go` | each tool returns a result holding U+FFFD for each invalid byte (`printf 'a\377b\n'` for bash) | the same panic |
    | `TestInvalidUTF8ContextFile` | `acpserver/sysprompt_test.go` | an `AGENTS.md` of `R\xe9gle du projet.` gives a baseline entry with U+FFFD, and the first prompt completes | the panic in `prompt.mustJSON` |
    | `TestToolResultMessageInvalidUTF8`, `TestBadBlockReachesWriter` | `acpserver/history_test.go` | the content is not empty; a block holding `Arguments: jsontext.Value("{")` makes the prompt answer `appendFailed`'s error | content 0 bytes, no error |
    | `FuzzQuote` | `jsonx/fuzz_test.go` | `Quote(s)` equals `json.Marshal(s, jsontext.AllowInvalidUTF8(true))` and is valid JSON, for every input | with `Quote` as `json.Marshal(s)` and its error ignored: the outputs differ |
    | `TestCutOffToolCallArguments` | `acpserver/prompt_test.go` | arguments `{"path":"a.tx` give `end_turn`; the client sees the `tool_call` with `rawInput` the JSON string `{"path":"a.tx`, the result, and the reply | `-32602 Invalid params`, three updates |
    | `TestExtensionParamsLoneSurrogate` | `acpserver/extensions_test.go` | `_gobble/get_entries` whose `sessionId` ends in a lone surrogate escape is answered "unknown session" | `-32602`, `invalid surrogate pair` |
    | `TestLoneSurrogateLineLoads` | `session/jsonl/store_test.go` | the entry loads with U+FFFD, and is not counted as malformed | skipped, one malformed line |
    | `TestFrom`, `TestValue`, `TestSplice`, `TestIsNull` | `jsonx/jsonx_test.go` | #9's U1 table: nil and empty are absent; `null` is present and null; `{}` writes `{}`; `{` is stored and its parent's marshal fails; `Value` is never empty non-nil; `Splice` over the MADR's five rows | `TestSplice` with `Splice` the identity, on the cut-off row |
    | `TestSetTextInvalidUTF8`, `TestSetTextMatchesSetBlocks` | `session/session_test.go` | one text block with U+FFFD; equal to `SetBlocks` for valid text | with `SetText` delegating to `SetBlocks` and dropping its error: empty content |
    | `TestRawInputOnTheWire` | `acpserver/convert_test.go` | its cases and assertions unchanged; its one `jsonRaw` call is `jsonx.Splice` (deviation 2) | not new |

11. **Assert by search**, exit status taken before any filter:
    * no `func mustJSON` and no `func quote(` in the module;
    * no `_ = ` before `SetBlocks(`;
    * no "always encode" in a Go file but `internal/config/render.go:92`, which is YAML;
    * no `jsonRaw(` and no `rawInput(`.
12. **Goldens:** `session.golden`, `print-json.golden` and `print-stream-json.golden` are unchanged (`git diff --stat`
    shows none).
13. G-local, without `make perf`, since P1 touches no `internal/cli` file. G-hosts. **Commit:** "the crash and break
    fix".

### P2 — the configuration policy, and the duplicate-name gates (MADR D3, D4's configuration rows, Q6)

Files:
* `internal/auth/store.go`, `internal/auth/store_test.go`;
* `llm/catalog/catalog.go`, `llm/catalog/modelsjson.go`, `llm/catalog/modelsjson_test.go`;
* `mcpclient/file.go`, `mcpclient/config.go`, `mcpclient/config_test.go`, `mcpclient/file_test.go`;
* `internal/cli/mcp.go`, `internal/cli/mcp_test.go`;
* `permission/store.go`, `permission/trust.go`, and their tests;
* `internal/cli/config.go`, `internal/cli/config_test.go`;
* `jsonx/fuzz_test.go`.

1. **The strict sites** use `jsonx.MarshalStrict`, `jsonx.UnmarshalStrict`, or `jsonx.StrictOptions()` for a
   `jsontext` encoder or decoder:
   * `internal/auth/store.go:84`, `:111`, `:160`, `:243`, `:285`;
   * `llm/catalog/catalog.go:32` and `llm/catalog/modelsjson.go:141`;
   * `mcpclient/config.go:209`, `:226`, `:256`, `:273`, `:306`, `:355`, and `mcpclient/file.go:31` and `:80`;
   * `internal/cli/mcp.go:155` and `:181`;
   * `permission/store.go:72` and `:180`, and `permission/trust.go:150` and `:241`.

   Each keeps its existing error report. `choice.go` is P6's.
2. **`mcp.json`'s duplicates** (Q6): `mcpclient/file.go:31` drops `jsontext.AllowDuplicateNames(true)`, and `members`'
   doc comment drops "as JSON.parse does".
   * A duplicate fails `parseFile`, so `Load` returns one `Config.Errors` entry, `<path>: <message>`, naming the
     object's path and the name (`mcpclient/config.go:88-95`).
   * `Add` and `Remove` (`mcpclient/file.go:102-148`) fail on the same error before writing, so the file is not changed.
   * The agent's note (`acpserver/mcp.go:71-73`) and `gobble mcp list`'s `config error:` and exit 1
     (`internal/cli/mcp.go:250`, `:298`) need no change. A test pins each.
3. **`gobble config validate`** (`internal/cli/config.go:57-60`, `:315-341`) also checks the JSON configuration files.
   *Amended 2026-10-31, by deviation 5:* `models.json` is stripped first, as gobble reads it, through a new
   `catalog.StripModelsJSON`, and `ParseModelsJSON`'s own error is a finding too:
   * `mcpConfigPath()` (`internal/cli/agent.go:178-184`), `authPath()` (`internal/cli/credentials.go:20-25`),
     `models.json` (`internal/cli/modelsjson.go:40`), and `permission.PermissionsFile` and `permission.TrustFile` in
     the configuration directory;
   * a missing file gives no finding. Each present file is decoded with `jsonx.UnmarshalStrict` into `jsontext.Value`,
     which checks syntax, UTF-8 and duplicate names at every depth, and `mcp.json` is also loaded with `mcpclient.Load`,
     whose `Errors` become findings;
   * a `*jsontext.SyntacticError` becomes a `config.Diagnostic` (`internal/config/diag.go:17-23`): `File` the path,
     `Line` 1 plus the count of `\n` before `ByteOffset`, `Path` its `JSONPointer`, `Severity` error, and `Text` a
     message of gobble's own: "a name is repeated", "not valid UTF-8", or "not valid JSON". A value is never printed,
     since `auth.json` holds credentials;
   * `printDiagnostics` sorts them with the settings' findings and exits 1 on an error, as today (0018-MADR D7).
4. **Tests** (Confirmation 8 and 22), each seen failing by the break named:

   | Test | File | Asserts | Seen failing with |
   | :--- | :--- | :--- | :--- |
   | `TestAuthFileStrict`, `TestSetKeyInvalidUTF8` | `internal/auth/store_test.go` | an `auth.json` holding an invalid byte, or a lone surrogate escape, is refused; `SetKey` with a key holding `\xff` returns an error and leaves the file byte for byte | the site switched to `jsonx.Unmarshal` or `Marshal`, which repairs |
   | `TestModelsJSONStrict` | `llm/catalog/modelsjson_test.go` | an invalid byte is refused | the same switch |
   | `TestMCPConfigStrict` | `mcpclient/config_test.go` | an invalid byte is refused | the same switch |
   | `TestDuplicateServerRefused`, `TestDuplicateFieldRefused` | `mcpclient/config_test.go` | two `github` servers, or one server with `command` twice, give one error naming the path and the name, and no server from the file | `AllowDuplicateNames(true)` left in `members` |
   | `TestEditRefusesDuplicate` | `mcpclient/file_test.go` | `Add` and `Remove` return the error and leave the file byte for byte | the same; at the base, `Add` deleted the first `github` |
   | `TestMCPListDuplicate` | `internal/cli/mcp_test.go` | `config error:` printed, exit 1 | the same |
   | `TestConfigValidateJSONFiles` | `internal/cli/config_test.go` | a duplicate in each of the five files gives a finding with file, line and path, no value, and exit 1; clean files give none, and exit 0; a secret-shaped value in `auth.json` never appears in the output | validate without the JSON files: no finding |
   | `FuzzNoDuplicateWritten` | `jsonx/fuzz_test.go` | random `session.Entry` values, embedded unknown members included, encode through `jsonx.Marshal` either with an error or with output that `jsontext.Value.IsValid()` accepts | the encode done with `jsontext.AllowDuplicateNames(true)`: an output with a repeated name |

5. G-local with `make perf` (`internal/cli` touched). G-hosts with `make perf` and `make probe-conhost`. **Commit.**

### P3 — the renames (MADR D11's spellings)

Files: the 8 files that only spell `RawMessage`:
* `acpclient/acptest/client.go` (`:70`, `:77`), `acpclient/conn.go` (`:415`), `acpserver/extensions.go` (`:64`, `:77`),
  `acpserver/grants.go` (`:80`), and `acpserver/prompt.go` if P1 left it;
* `acpclient/compaction_test.go`, `acpclient/personality_test.go`, `acpclient/retry_test.go`;
* plus `acpclient/update.go:105` (`Params`).

1. Each `json.RawMessage` or `stdjson.RawMessage` becomes `jsontext.Value`. Each `encoding/json` import that is then
   unused goes.
2. Nothing changes at run time: the types are identical. `go build ./...`, `go vet ./...` and every test pass unchanged.
3. Assert: the 8 files import no `encoding/json`.
4. G-local, G-hosts. **Commit.**

### P4 — everything gobble defines, on v2 (MADR D12 to D15)

Files:
* `jsonx/jsonx.go`, `jsonx/jsonx_test.go`, `jsonx/fuzz_test.go`;
* `acpclient/conn.go`, `acpclient/update.go`, `acpclient/sessions.go`, and their tests (`acpclient/handler_test.go`,
  `retry_test.go`, `compaction_test.go`, `personality_test.go`, `sessions_test.go`);
* `acpserver/prompt.go`, `agent.go`, `extensions.go`, `commands.go`, `auth.go`, `compact.go`, `personality.go`,
  `question.go`, `grants.go`, `mentions.go`, `config.go`, and their tests;
* `acpclient/acptest/client.go`, `acpclient/acptest/script.go`.

1. **`jsonx` gains:**
   * `Strings(map[string]string) jsontext.Value`: an object, its keys sorted, each value through `Quote`, written with
     a `bytes.Buffer`. It cannot fail;
   * `SetMember(meta map[string]any, key string, v any) (map[string]any, error)`: a copy of `meta`, or a new map when
     `meta` is nil, with `key` set to `Value(v)`;
   * `Member[T any](meta map[string]any, key string) (T, bool)`: `meta[key]` re-encoded by `Marshal` and decoded into
     `T` by `Unmarshal`. It is false when the member is missing, does not encode, or does not decode.
2. **Receiving** (D10, D12): `acpclient`'s `handler` serves its extension methods through an `acp.ExtensionMux`.
   * Three `acp.Notify[jsontext.Value]` registrations, one each for `_gobble/retry`, `_gobble/compaction` and
     `_gobble/personality`, decode with `jsonx.Unmarshal` into today's types (`acpclient/update.go:125-195`).
   * They route to `h.sessions` and `h.pending` as `HandleExtensionMethod` does today (`acpclient/conn.go:415-448`),
     which goes.
   * `retryOf`, `compactionOf` and `personalityOf` keep their names and validation, and decode through `jsonx`.
3. **Sending** (D13):
   * `updates.notify` (`acpserver/prompt.go:68-72`) encodes `params` once with `jsonx.Value`. On error it keeps the
     error, as `send` keeps one, and sends nothing; otherwise it passes the `jsontext.Value` to `NotifyExtension`. Its
     eleven callers do not change;
   * `acpclient/sessions.go:107` passes `jsonx.Value(setNameParams{...})`, and returns its error.
4. **Error data** (D14): each of the 34 calls builds `jsonx.Strings(map[string]string{...})`:
   * `acpserver/agent.go` 12, `extensions.go` 6, `commands.go` 3, `prompt.go` 3, `auth.go` 2, `compact.go` 1,
     `personality.go` 1;
   * `acpclient/conn.go` 1;
   * `acpclient/acptest/client.go` 2 and `script.go` 3. `script.go:159`'s `"update": i` becomes `strconv.Itoa(i)`.

   The keys are the same constants (`keyReason`, `keyError`, `keySessionID`).
5. **`_meta.gobble`** (D15):
   * each of the eight writers defines a struct whose JSON names are today's map keys, and sets it with
     `jsonx.SetMember`;
   * each of the seven readers reads its members with `jsonx.Member`, one member at a time, keeping "a missing or
     mistyped one is none".
6. **Tests** (Confirmation 18 to 21):

   | Test | File | Asserts | Seen failing with |
   | :--- | :--- | :--- | :--- |
   | `TestNotificationDecodedWithV2` | `acpclient/retry_test.go` | a `_gobble/retry` whose `cause` ends in a lone surrogate escape reaches the callback with U+FFFD; one whose `attempt` is a string is logged and ignored | `jsontext.AllowInvalidUTF8(false)` planted in the handler's decode: refused, callback not called |
   | `TestNotifyFramesUnchanged` | `acpserver/prompt_test.go` | for each notification type, `notify`'s frame equals the frame the SDK writes from the Go value, for text with `<`, `&`, U+2028 and an invalid byte | `jsontext.AllowInvalidUTF8(false)` planted in `notify`'s encoding: the invalid-byte frame is not sent |
   | `FuzzStrings` | `jsonx/fuzz_test.go` | `Strings(m)` equals `json.Marshal(m, Options())`, and is valid JSON | `Strings` without the key sort |
   | `TestMetaMemberMistyped` | `acpclient/handler_test.go` | a permission request whose `_meta.gobble` has `repeated` as a string and `limit` as a string reads `limit` and no `repeated` | a whole-struct decode in place of `Member` |

   `session.golden`, the `errText` tests, and every existing `_meta.gobble` test pass unchanged.
7. G-local, G-hosts. **Commit.**

### P5 — `translate` (MADR D16; PQ2)

Files: `acpclient/conn.go`, `acpclient/update.go`, `internal/cli/agent.go`, `internal/cli/print.go`,
`internal/cli/print_test.go`, new `internal/cli/testdata/print-stream-json-escapes.golden`.

1. By PQ2 (a): `acpclient.Options` gains `Params bool`, documented as "encode `Update.Params`, for a client that passes
   the stream through". `Update.Params`' doc (`acpclient/update.go:103-104`) says it is set only then.
2. `translate` (`acpclient/update.go:244`) encodes only when `Params` is set, with
   `jsonx.Marshal(n, jsontext.EscapeForHTML(true), jsontext.EscapeForJS(true))`.
3. Print mode sets `Params` for `--output-format stream-json`. `startAgentWith` (`internal/cli/agent.go`) takes it from
   `printSink.stream` (`internal/cli/print.go:74`).
4. `writeJSON` (`internal/cli/print.go:145`) encodes with `jsonx.Marshal`, with no escaping options. A comment states the
   two styles, by MADR D16.
5. **Tests** (Confirmation 9):
   * `TestPrintStreamJSONEscapes`: a reply holding `<`, `&`, U+2028 and U+2029 is pinned by
     `print-stream-json-escapes.golden`, with the update lines escaped and the result line not. Seen failing with
     `EscapeForJS` dropped from `translate`.
   * `print-stream-json.golden` and `print-json.golden` are unchanged.
6. `make apidiff`, within preflight, reports only the added field.
7. G-local with `make perf`. G-hosts with `make perf` and `make probe-conhost`. **Commit.**

### P6 — the local files and the providers output (MADR D17)

Files: `internal/cli/choice.go`, `internal/cli/choice_test.go`, `internal/cli/providerscmd.go`,
`internal/cli/providerscmd_test.go`, new testdata files.

1. `choice.go:128` and `:211` decode with `jsonx.UnmarshalStrict`; `:152` and `:224` encode with `jsonx.MarshalStrict`,
   `:152` with `jsontext.WithIndent("  ")`. The v1 import goes.
2. `providerscmd.go:51` encodes with `jsonx.Marshal(rows, jsontext.WithIndent("  "))`. The v1 import goes.
3. **Tests** (Confirmation 10):
   * `TestRecentModelsV1File` and `TestKeptListingV1File`: fixtures exactly as v1 wrote them, `null` slices included,
     load as before;
   * `TestProvidersJSON`: pins `gobble providers --json` for a row whose reason holds `<` and `&`, unescaped, the change
     the MADR's Consequences name.

   Seen failing: the fixture tests, with the files' `null` read into a type that refuses it; `TestProvidersJSON`, with
   `jsontext.EscapeForHTML(true)` planted.
4. G-local with `make perf`. G-hosts with `make perf` and `make probe-conhost`. **Commit.**

### P7 — the remaining data sites, and the test files (MADR D4, D25 phase 7)

Files: every D4 row not done in P1 to P6, and the ten test files that import v1.

1. **The data sites,** each `json.X` call becoming `jsonx.X`, with any options it passes kept:
   * `acpserver/agent.go:834`, `commands.go:202`, `compact.go:115`, `grants.go:102`, `modes.go:237`,
     `personality.go:47`, `sysprompt.go:304`;
   * `agent/agent.go:216`;
   * `compaction/compaction.go:151`, `compaction/estimate.go:246` and `:261`, `compaction/prompts.go:248`, and
     `compaction/serialize.go:100`, `:228` and `:274`;
   * `internal/cli/auth.go:170` and `:409`, `config.go:196` and `:213`, `configpath.go:61`, `mcp.go:256`,
     `sessions.go:191`, `version.go:41`;
   * `internal/config/schema.go:34` and `:64`;
   * `llm/provider/provider.go:438`, `:445` and `:460`;
   * `mcpclient/tool.go:76` and `:143`;
   * `session/migrate.go:101` and `:103`, as `jsontext` decoder and encoder with `jsonx.Options()`;
   * `tool/builtin/edit.go` (ten calls, `:111` to `:163`), `grep.go:366`, `intercept.go:22` and `:57`, `paths.go:91`,
     `read.go:118`, `shell.go:316`, `:327`, `:338`, `:348` and `:360`;
   * `tool/tool.go:220`, `:478`, `:485`, `:507` and `:549`;
   * `tool/toolsearch/index.go:169`;
   * the test support: `acpclient/acptest/recorder.go:178`, `acpclient/acptest/script.go:158`, `:185` and `:203`, and
     `llm/llmtest/assert.go:42`.
2. **The ten test files that import v1:**
   * `acpclient/compaction_test.go`, `personality_test.go`, `retry_test.go`, `sessions_test.go`;
   * `acpserver/autocompact_test.go`, `convert_test.go`, `retry_test.go`;
   * `internal/cli/models_test.go`, `personalitycmd_test.go`, `providerscmd_test.go`.

   Those not already changed by P3 or P4 move to v2, as follows:
   * `acpserver/autocompact_test.go:198-201` compares with `reflect.DeepEqual`. Seen failing by changing one value in
     the expected map;
   * `acpserver/convert_test.go:17` calls the SDK value's own `MarshalJSON`;
   * the decodes into `map[string]any` use v2.
3. **Assert:** `grep -rlE '"encoding/json"$' --include=*.go .` returns nothing.
4. **Unchanged:** `session.golden`, `print-json.golden`, `print-stream-json.golden`, and every other golden; `make perf`
   (Confirmation 14).
5. G-local with `make perf`. G-hosts with `make perf` and `make probe-conhost`. **Commit.**

### P8 — `archtest`, the `inline` tags, the hatch's tests, and the docs (MADR D18, D11, D21, D23)

Files: `internal/archtest/check.go`, `acpserver/extensions.go`, `acpserver/personality.go`, `session/session_test.go`,
new `llm/llm_test.go`, `session/doc.go`, `docs/architecture.md`.

1. **`internal/archtest/check.go:68`** reads `go list -json`'s stream with `jsontext.NewDecoder(&stdout,
   jsonx.Options())` and `json.UnmarshalDecode` until `io.EOF`. `make archtest` passes, and its rules report what they
   reported before.
2. **The eight `json:",inline"` tags** (`acpserver/extensions.go:88`, `:93`, `:98`, `:103`, `:108`, `:113`, `:118`;
   `acpserver/personality.go:276`) are dropped. The `_gobble/*` and personality tests pass unchanged (Confirmation 17).
3. **The hatch's tests** (Confirmation 11):
   * `TestRawNullAndAbsent`, on `Entry.Replacement` and `Message.ExitCode`: absent is left out and read back absent;
     `null` is kept; an object round-trips. Seen failing with the field's tag changed to `omitempty`, which drops a
     `null`;
   * `TestEmptyRawValue`: an empty non-nil value planted on the field fails the marshal with the EOF error, and through
     `jsonx.From` is left out;
   * `TestContentDataBase64`, in `llm/llm_test.go`: `llm.Content.Data` encodes as base64. Seen failing with the field
     retyped `jsontext.Value`.
4. **The docs** (D23):
   * `session/doc.go` gains the hatch's table: raw JSON is `jsontext.Value`, binary is `[]byte`, empty is absent,
     `null` is kept, and tests compare `len` and `string`, never `String()`;
   * `docs/architecture.md`'s package notes gain `jsonx`, in present tense.
5. G-local, G-hosts. **Commit.**

### P9 — the guards (MADR D24)

Files: `.golangci.yml`, `internal/archtest/check.go`, `internal/archtest/check_test.go`.

1. **`depguard`**: a rule beside `logging` (`.golangci.yml:41-53`), for every file, tests included, denying
   `encoding/json$` with the message "use jsonx, on encoding/json/v2 (0039-MADR D24)".
2. **`forbidigo`**: a rule (`.golangci.yml:56-64`) forbidding `encoding/json/v2`'s `Marshal`, `MarshalWrite`,
   `MarshalEncode`, `Unmarshal`, `UnmarshalRead` and `UnmarshalDecode`, with `analyze-types: true`, its exact pattern
   settled when first seen failing. An exclusion keeps `jsonx/` and `_test.go` files out of it.
3. **`internal/archtest`**, two rules, each with a test case in `check_test.go`:
   * `jsonx` imports no package of this module;
   * a struct field of type `[]byte` with a `json` tag is allowed only where listed: today `llm/llm.go`
     `Content.Data`. The rule parses each package's Go files, as `go list` names them, with `go/parser`.
4. **Seen failing** (Confirmation 12), each on a scratch copy of the finished tree:
   * an `encoding/json` import planted in a non-test file fails `depguard`;
   * a `json.Marshal` call planted in `tool/` fails `forbidigo`;
   * a `jsonx` import of `session` fails `archtest`;
   * a `[]byte` field with a `json` tag planted in `session` fails `archtest`.
5. G-local, G-hosts. **Commit.**

### P10 — close-out

Files: this plan, the MADR if a fact changed, `docs/README.md`, `docs/architecture.md` if P8 left anything.

1. **The count, measured:** `go list` and a scan show no `encoding/json` import and no v2 call outside `jsonx` and tests.
   The execution record quotes them.
2. **Issue comments,** written into the execution record for the owner to post. Posting is outward-facing, so it is
   done only on the owner's word:
   * gobble-cli#7: U1 and U5 done, U2 declined, U3 elsewhere, U4 met, with the phases' commits;
   * gobble-cli#9: each of its units and acceptance items, with the phases' commits.
3. **0038-REPORT:** a dated note that its order was carried out by this plan.
4. **The disclosure guard** over the outgoing commits, its output recorded. Nothing is pushed.
5. **Status:** this plan's status becomes `complete` when every Confirmation item is met. The MADR's status stays
   `accepted`.

## Verification

Each Confirmation item of the MADR, with the phase and the test that meets it:

| Confirmation | Phase | Test or check |
| :--- | :--- | :--- |
| 1 a turn reading a Latin-1 file completes | P1 | `TestTurnReadsInvalidUTF8File` |
| 2 `bash` and `grep` on invalid UTF-8 | P1 | `TestReadInvalidUTF8`, `TestBashInvalidUTF8Output`, `TestGrepInvalidUTF8File` |
| 3 a Latin-1 `AGENTS.md` | P1 | `TestInvalidUTF8ContextFile` |
| 4 `toolResultMessage`, and an error reaching the writer | P1 | `TestToolResultMessageInvalidUTF8`, `TestBadBlockReachesWriter` |
| 5 `Quote` never fails | P1 | `FuzzQuote` |
| 6 a cut-off tool call | P1 | `TestCutOffToolCallArguments` |
| 7 a lone surrogate in `_gobble/*` params | P1 | `TestExtensionParamsLoneSurrogate` |
| 8 configuration is strict | P2 | `TestAuthFileStrict`, `TestSetKeyInvalidUTF8`, `TestModelsJSONStrict`, `TestMCPConfigStrict` |
| 9 `stream-json`'s bytes | P5 | `TestPrintStreamJSONEscapes`; the existing goldens |
| 10 v1-written local files load | P6 | `TestRecentModelsV1File`, `TestKeptListingV1File` |
| 11 null and absent; base64 | P8 | `TestRawNullAndAbsent`, `TestEmptyRawValue`, `TestContentDataBase64` |
| 12 the guards | P9 | the four planted breaks |
| 13 a lone-surrogate session line loads | P1 | `TestLoneSurrogateLineLoads` |
| 14 no change elsewhere | each phase, P7 in full | the goldens; `make perf` |
| 15 `jsonx`'s own tests | P1, P4 | `TestFrom`, `TestValue`, `TestSplice`, `TestIsNull`, `TestSetTextInvalidUTF8`, `TestSetTextMatchesSetBlocks` |
| 16 empty `rawInput` left out | P1 | `TestRawInputOnTheWire`, its cases and assertions unchanged |
| 17 `Convert` and the tags change nothing | P1, P8 | the existing replay, serialization, compaction, `_gobble/*` and personality tests |
| 18 notifications decoded with v2 | P4 | `TestNotificationDecodedWithV2` |
| 19 the wire unchanged | P4 | `TestNotifyFramesUnchanged`; `session.golden` |
| 20 `jsonx.Strings` | P4 | `FuzzStrings`; the `errText` tests |
| 21 `_meta.gobble` per member | P4 | `TestMetaMemberMistyped`; the existing readers' tests |
| 22 duplicates refused at every gate | P2 | `TestDuplicateServerRefused`, `TestDuplicateFieldRefused`, `TestEditRefusesDuplicate`, `TestMCPListDuplicate`, `TestConfigValidateJSONFiles`, `FuzzNoDuplicateWritten` |

Every phase also passes G-local and G-hosts (C6, C7), and keeps `make check-records` and markdownlint green.

## Rollout and Rollback

* **Each phase is one commit,** and reverting it restores the state before it. P1 stands alone. P2 to P9 each depend
  only on P1's `jsonx`, except:
  * P9's guards, which need P3 to P8 done;
  * P4's `Member` and `SetMember`, which P2 does not use.
* **No data migration.** Session files written under the data policy hold only valid UTF-8, which every earlier gobble
  reads. A session file that an earlier gobble could not write, its header holding a non-UTF-8 path, is now written,
  with U+FFFD in place of the path's invalid bytes (the MADR's Consequences).
* **User-visible changes,** for the release notes:
  * an `mcp.json` with a repeated name is refused, where it loaded before (P2);
  * `gobble config validate` checks the JSON configuration files (P2);
  * `gobble providers --json` no longer HTML-escapes (P6).
* **Rollback of P2 alone** restores `mcp.json`'s last-value rule, and so its silent deletion on edit. That is the
  measured defect, so a rollback is recorded as a deviation, never done quietly.
* **No configuration, CI workflow or release change,** apart from `.golangci.yml` in P9.

## Execution record

Each phase appends its record here: what it did, the gates' output, and each seen-failing run quoted, with a dated
entry for every deviation.

### P0 record (2026-10-31): the records

* `9b3bffe`: 0035's records, its deviation 13 and P23, by PQ1 (a). `check_records.py --check-all` and markdownlint passed
  on an export of exactly the staged tree, since `docs/README.md`'s 0038 and 0039 rows were held out of the commit.
* `153ef60`: this pair and 0038-REPORT. `make check-records markdownlint` passed with nothing unstaged.
* The hooks directory is the global one. The identifier scans of both staged diffs found none.

### Deviation 1 (2026-10-31): P1, the eight callers of `acpserver`'s `mustJSON`

* **Found,** at P1's start (`153ef60`), by listing the callers of every function P1 changes. Step 3 deletes
  `acpserver/history.go:46-52` `mustJSON`, but planned for two of its eight callers, `history.go:39` and `:66`. The other
  six are:
  * `acpserver/sysprompt.go:326` `contextEntry`, which encodes context files' paths, so a path that is not UTF-8
    panics today; it reaches the writer at `acpserver/agent.go:894`;
  * `acpserver/agent.go:822` `toolsEntry`, reaching the writer at `:672` and `:897`;
  * `acpserver/history.go:414` `todoEntry`, reaching the writer at `agent.go:891`;
  * `acpserver/personality.go:38` `personalityEntry`, a file outside P1's list, appended at `:176`, whose error is
    returned;
  * `acpserver/history.go:272` and `:287`, in `assistantContent`, which encode a `reasoning` value of four strings on
    replay. `replayPath` returns a `state`, with no error path (`acpserver/agent.go:505`, `acpserver/history.go:182`).

  `llm/provider/provider.go:363-376`'s `reasoningData` and `signatureData` encode the same four-string shape, and P1
  had them return errors into the stream loop.
* **The owner's answer:** "Extend P1, add jsonx.Fields (Recommended)". The options not taken were:
  * propagating errors everywhere, through `assistantContent`, `replayPath`'s `state` and its two callers, and the
    stream loop;
  * narrowing P1, with `mustJSON` kept on `jsonx.Marshal` until P7.
* **Decision.**
  * P1 covers all eight callers.
    * `contextEntry`, `toolsEntry`, `todoEntry` and `personalityEntry` return `(session.Entry, error)` through
      `jsonx.Value`.
    * Their callers pass the error to `w.fail`, or return it as `append`'s error is returned.
  * `jsonx` gains `type Field struct{ Name, Value string }` and `Fields(fs ...Field) jsontext.Value`. It writes an
    object of the fields whose `Value` is not empty, in the order given, each name and value through `Quote`, and `{}`
    when none is. That is what `json.Marshal` writes for a struct of `omitzero` string fields in that order, under the
    data policy. It cannot fail.
    * `FuzzFields` holds the equality against `jsonx.Marshal` of such a struct. It is seen failing with the fields
      sorted.
    * `reasoningData`, `signatureData` and `assistantContent`'s two values use it. They keep their signatures, and the
      stream loop and replay do not change.
  * MADR D1 and D6 have dated amendments.
* **Files added to P1:** `acpserver/personality.go`; `llm/provider/provider_test.go` no longer needs to change.

### Deviation 2 (2026-10-31): P1, two tests that call what P1 changes

* **Found** while P1 changed `prompt.BaselineData` and `PatchData` to return an error. Every test calling a function
  whose signature P1 changes, or which it deletes, was listed. There are two:
  * `compaction/project_test.go:188`, `:190` and `:191` call `BaselineData` and `PatchData`. The file is outside P1's
    list;
  * `acpserver/convert_test.go:17` calls `jsonRaw`, which P1 deletes. The PLAN's P1 table and verification row, and
    the MADR's Confirmation 16, said the test passes "unchanged", which cannot hold.
* **The owner's answer:** "Add the test file, change one call (Recommended)". The option not taken was to keep `jsonRaw`
  as an alias of `jsonx.Splice` so that the test stayed literally unchanged.
* **Decision.**
  * `compaction/project_test.go` joins P1. It takes the two functions' errors, with `t.Fatal`, and its assertions do
    not change.
  * `acpserver/convert_test.go:17`'s call becomes `jsonx.Splice`. Its cases and assertions do not change.
  * The MADR's Confirmation 16, P1's table row and the verification row say so.
* **Files added to P1:** `compaction/project_test.go`.

### Deviation 3 (2026-10-31): P1, `Convert`'s re-encode is Pi parity

* **Found** when `compaction`'s tests ran after step 4. `TestProjectMatchesPi` (`compaction/project_test.go:71-105`)
  failed on the v2, v2-branched, v3 and v3-branched fixtures. Measured on a scratch copy, a custom message's content
  came out as stored, `"A message from a hook."`, where Pi's `convertToLlm` writes `[{"type":"text","text":"A message
  from a hook."}]`.
  * The re-encode step 4 removed made a string content one text block.
  * The MADR's D7 said "no reader sees a change", which was wrong: the Pi fixtures compare the converted messages.
  * The fault was in the plan, not in the code.
* **The owner's answer:** "String to SetText, array kept (Recommended)". The options not taken were:
  * restoring the re-encode with a fallback to the stored content on error;
  * `Convert` returning an error through its three callers.
* **Decision.**
  * In `Convert`'s custom case, a stored content that `jsonx.Unmarshal` reads as a JSON string is written with
    `SetText`, which cannot fail and writes what `SetBlocks` writes for one text block. An array is kept as stored.
  * Every caller reads it through `Blocks`, and Pi's fixtures compare it decoded, so neither sees a difference from
    the re-encoded array.
  * Measured on a scratch copy before the change: `compaction`'s tests pass, `TestProjectMatchesPi` included.
  * The MADR's D7 bullet has a dated correction.
* **Files added:** none.

### Deviation 4 (2026-10-31): P1 step 11's asserts and three test sites

* **Found** when step 11's asserts ran. In non-test code they are clean:
  * no `func mustJSON` and no `func quote(` that panics;
  * no `_ = …SetBlocks(`;
  * no "always encode" but `internal/config/render.go:92`, which is YAML;
  * no `jsonRaw(` and no `rawInput(`.

  Step 11 said "in a Go file", and three test sites match:
  * `acpserver/tree_test.go:207-215` and `acpclient/conn_test.go:403-411`, two `jsonString` helpers that panic on an
    encode error of a test-chosen string ("a string always encodes");
  * `acpserver/autocompact_test.go:199`, which P7 changes to `reflect.DeepEqual`.

  `tool/builtin/subject_test.go:70`'s `quote(t, p)` matches by name only: it reports with `t.Fatal`.
* **The owner's answer:** "Fix the two helpers in P1 (Recommended)". The option not taken was to scope the assert to
  non-test code.
* **Decision.**
  * The two `jsonString` helpers return `string(jsonx.Quote(s))`, which cannot fail, so no test panics on an encode
    error.
  * The assert then holds for every Go file but `internal/config/render.go` and `acpserver/autocompact_test.go`,
    P7's site. `subject_test.go`'s `quote` stays.
* **Files added to P1:** `acpserver/tree_test.go`, `acpclient/conn_test.go`.

### P1 record (2026-10-31): the crash and break fix

The base is P0's `153ef60`. Deviations 1 to 4, above, changed the step's scope; this record states what was built.

1. **`jsonx`** (`jsonx/doc.go`, `jsonx/jsonx.go`): `Options`, `StrictOptions`, `Marshal`, `Unmarshal`, `MarshalStrict`,
   `UnmarshalStrict`, `Value`, `Quote`, `Field` and `Fields` (deviation 1), `From`, `Splice`, `IsAbsent`, `IsNull`. It
   imports `bytes`, `encoding/json/v2` and `encoding/json/jsontext` only. `Quote` is `jsontext.AppendQuote`, its error a
   replacement report, which the doc comment and a `nolint` comment state.
2. **`session`:** `SetText`, new and infallible; `SetBlocks` and `Blocks` on `jsonx`. `session/jsonl/store.go`'s two
   decodes and two encodes are on `jsonx` (step 7), and its v2 import goes.
3. **No encode panics:**
   * `tool/tool.go`'s and `mcpclient/tool.go`'s `quote` are gone; their three callers call `jsonx.Quote`.
   * `prompt/sections.go`'s `mustJSON` is gone. `BaselineData` and `PatchData` return `(jsontext.Value, error)`, and
     `ParseData` decodes through `jsonx`.
   * `acpserver/history.go`'s `mustJSON` is gone, and with it all eight of its uses (deviation 1):
     * `extra` returns its error;
     * `assistantEntryMessage` returns `(*session.Message, error)`;
     * `contextEntry`, `toolsEntry`, `todoEntry` and `personalityEntry` return `(session.Entry, error)`;
     * replay's two `reasoning` values use `reasoningValue`, on `jsonx.Fields`.
   * `userEntryMessage` writes its one string with `jsonx.Quote`, which cannot fail, so it keeps its signature. Step 3
     had it return an error; deviation 1's principle, an infallible helper where the value is strings only, applies.
   * `llm/provider`'s `reasoningData` is on `jsonx.Fields`. It and `signatureData` keep their signatures, and the
     stream loop does not change (deviation 1).
4. **No discarded `SetBlocks` error:**
   * `toolResultMessage`, `compaction/serialize.go`'s text and `compaction/estimate.go`'s two sites use `SetText`;
   * `Convert`'s custom case writes a string content with `SetText`, and keeps an array as stored (deviation 3).
5. **The writer:** `(*writer).fail` and `addEntry` (`acpserver/agent.go`). Each builder's error reaches `w.err`, which
   ends the prompt with `appendFailed`, as an append's does. `personalityEntry`'s error is returned with `append`'s.
6. **Extensions:** results by `jsonx.Marshal`, params by `jsonx.Unmarshal` (`acpserver/extensions.go`).
7. **The splice:** `rawInput` and `jsonRaw` are gone. `acpserver/prompt.go`, `question.go` and `history.go` call
   `jsonx.Splice`, and `prompt.go`'s `stdjson` and `jsontext` imports go with them.
8. **Empty raw values:** `agent/agent.go`'s two `tool.Call`s use `jsonx.From([]byte(c.Args))`.
9. **0004-MADR:** a dated amendment, "`jsonx`, gobble's one JSON package".
10. **Tests.** The PLAN's names, with two changes:
    * `TestBadBlockReachesWriter` became `TestWriterKeepsBuildError` (`acpserver/history_test.go`) and
      `TestSetBlocksBadJSON` (`session/session_test.go`). `assistantEntryMessage`'s error cannot be reached with a real
      model's input: its arguments are decoded before they are kept, and `extra`'s inputs are strings or decoded JSON.
      So each half of the path is pinned on its own;
    * `TestPolicies`, of the two policies, is added to `jsonx/jsonx_test.go`.

    **Seen failing at the base** (C9), each test alone in an export of `153ef60`:

    | Test | Exit | The failure |
    | :--- | :--- | :--- |
    | `TestToolResultMessageInvalidUTF8` | 1 | `saved "null": blocks [], <nil>`: content empty; `"null"` is #9's F3, `String()` of an absent value |
    | `TestTurnReadsInvalidUTF8File` | 1 | `panic: jsontext: invalid UTF-8 [recovered, repanicked]` |
    | `TestCutOffToolCallArguments` | 1 | `{"code":-32602,"message":"Invalid params","data":{"error":"json: error calling MarshalJSON for type *acp.SessionUpdate: …` |
    | `TestInvalidUTF8ContextFile` | 1 | `panic: jsontext: invalid UTF-8 within "/baseline/3/text" after offset 685 [recovered, repanicked]` |
    | `TestExtensionParamsLoneSurrogate` | 1 | `error -32602 "jsontext: invalid surrogate pair …"; want -32602, unknown session` |
    | `TestReadInvalidUTF8`, `TestBashInvalidUTF8Output`, `TestGrepInvalidUTF8File` | 1 each | `panic: jsontext: invalid UTF-8 [recovered, repanicked]` |
    | `TestLoneSurrogateLineLoads` | 1 | `read: 1 entries, 1 skipped, <nil>; want 2 and 0` |

    **Seen failing on a planted break** (C9), each on a scratch copy of the finished tree:

    | Test | Break | The failure |
    | :--- | :--- | :--- |
    | `TestSplice` | `Splice` the identity | `cut off: Splice("{\"path\":\"a.tx") = "{\"path\":\"a.tx", want …` |
    | `FuzzQuote` | `Quote` as `json.Marshal(s)`, its error ignored | `Quote("caf\xe9") = , Marshal = "caf…"`, on three seeds |
    | `FuzzFields` | the fields sorted by name | `Fields = {"format":…,"signature":…,"text":…}, Marshal = {"text":…,"signature":…,"format":…}` |
    | `TestFrom` | `From` without its empty case | `From([]byte{}) = "", want nil` |
    | `TestPolicies` | `Options` without `AllowInvalidUTF8` | `Marshal of invalid UTF-8: jsontext: invalid UTF-8 within "/s"` |
    | `TestIsNull` | no space trimmed | `IsNull(" null\n") = false, want true` |
    | `TestValue` | `Value` on `MarshalStrict` | `Value = "null", jsontext: invalid UTF-8 within "/s"` |
    | `TestSetTextInvalidUTF8` | `SetText` on a strict marshal, its error ignored | `session: message content: jsontext: invalid character ']' …` |
    | `TestSetTextMatchesSetBlocks` | `SetText`'s fields reversed | `SetText("plain") = [{"text":"plain","type":"text"}], SetBlocks = [{"type":"text","text":"plain"}]` |
    | `TestSetBlocksBadJSON` | `SetBlocks` ignoring its marshal error | `SetBlocks of bad arguments = "[{…\"arguments\"", want an error` |
    | `TestWriterKeepsBuildError` | `addEntry` adding despite the error | a nil `liveSession` dereference at `history_test.go:78` |

    The first planted break for `TestSetTextInvalidUTF8` did not compile, since it left `bytes` unused. It proves
    nothing, and was planted again as above.
    `FuzzQuote` and `FuzzFields` also ran 20 s each against the finished code: 1,124,870 and 143,780 executions, no
    failure.
11. **Assert by search** (step 11, with deviation 4):
    * no `func mustJSON`, and no `func quote(` that panics. `tool/builtin/subject_test.go:70`'s `quote(t, p)` reports with
      `t.Fatal`;
    * 0 `_ = …SetBlocks(`;
    * "always encode" only in `internal/config/render.go`, which is YAML, and `acpserver/autocompact_test.go`, P7's;
    * 0 `jsonRaw(` or `rawInput(`.
12. **Goldens:** none changed (`git status --short -- '*.golden'`: empty).
13. **G-local:**
    * `make pre-add-check` on the 34 changed and new Go files: clean;
    * `make preflight`: passed. Its first run stopped at golangci-lint's `unconvert`, at `jsonx/fuzz_test.go:55`, an
      unneeded `jsontext.Value(got)`. That was a fix to the step's own code, not a deviation;
    * `make race`: 37 ok, `jsonx` the new package;
    * no `make perf`, since P1 touches no `internal/cli` file.
14. **G-hosts,** on the Linux machine and the Windows laptop, with 813 files shipped to each:
    * `go test`: passed;
    * race: passed, 37 ok;
    * `make check-records markdownlint`: passed;
    * preflight stops at `verify-build-metadata`, as on every copy. A first host run, before step 13's lint fix,
      stopped at the same `unconvert` finding.

### Deviation 5 (2026-10-31): P2, `models.json` is JSONC

* **Found** while writing step 3. gobble reads `models.json` through `stripJSON`, which removes a BOM, `//` comments
  and trailing commas, "as Pi does" (`llm/catalog/modelsjson.go:117-145`). A plain strict decode, as step 3 had it,
  would refuse a valid commented file.
  * `stripJSON` keeps every newline: a comment is removed up to its `\n`, and a trailing comma's removal keeps the
    whitespace after it. So a line counted in the stripped bytes is the file's line.
  * `ParseModelsJSON` strips internally and does not expose the stripped bytes, so its syntax errors' offsets could
    not be mapped to lines.
* **The owner's answer:** "Strip as gobble reads it (Recommended)". The option not taken was to validate through
  `ParseModelsJSON` only, with no line or path.
* **Decision.**
  * `llm/catalog` exports `StripModelsJSON(data []byte) []byte`, which is `stripJSON`.
  * `gobble config validate` strips `models.json` with it, then decodes strictly. A duplicate, invalid UTF-8 or a
    syntax error gets its file, line and path.
  * When that passes, `ParseModelsJSON`'s own error, such as no `providers` object or a field of the wrong type, which
    makes gobble ignore the file whole, is a finding too.
* **Files added:** none; `llm/catalog/modelsjson.go` is already P2's.

### P2 record (2026-10-31): the configuration policy, and the duplicate-name gates

The base is P1's `6b5af2c`. Deviation 5 changed step 3.

1. **The strict sites** use `jsonx`'s strict forms:
   * `internal/auth/store.go`: its two decoders take `jsonx.StrictOptions()`, and its decode and two encodes use
     `UnmarshalStrict` and `MarshalStrict`;
   * `llm/catalog/catalog.go:32` and `llm/catalog/modelsjson.go:141` use `UnmarshalStrict`;
   * `mcpclient/config.go`'s six decodes use `UnmarshalStrict`, and `mcpclient/file.go`'s decoder and encoder take
     `StrictOptions()`;
   * `internal/cli/mcp.go:155`'s encoder takes `StrictOptions()`;
   * `permission/store.go` and `permission/trust.go` decode with `UnmarshalStrict`, and encode with `MarshalStrict` and
     `jsontext.WithIndent("  ")`.

   Every one of these already decoded with v2's defaults, so the only changes in behaviour are step 2's and step 3's.
   The existing tests of the five packages passed unchanged.
2. **`mcp.json`'s repeated names** (Q6): `members` (`mcpclient/file.go`) drops `AllowDuplicateNames(true)`, and its
   doc comment says a repeated name, at any depth, is an error. Its last-value bookkeeping went with it, since the
   decoder now refuses a repeat before it could run. `Load`, `Add`, `Remove`, the agent's note and `gobble mcp list`
   needed no further change.
3. **`gobble config validate`** (`internal/cli/config.go`), with deviation 5:
   * `jsonConfigFiles` names `mcp.json`, `auth.json`, `models.json`, `permissions.json` and `trust.json` in the
     configuration directory, through `systemDirs`, `mcpConfigPath` and `authPath`;
   * `jsonDiagnostics` reads each present file, strips `models.json` with the new `catalog.StripModelsJSON`, and
     decodes with `jsonx.UnmarshalStrict` into a `jsontext.Value`. Then it adds `ParseModelsJSON`'s error, or
     `mcpclient.Load`'s errors;
   * `jsonDiagnostic` takes the line and path from the `*jsontext.SyntacticError`. Its text is "a name is repeated"
     (`jsontext.ErrDuplicateName`), "not valid UTF-8" (the bytes valid under `AllowInvalidUTF8`), or "not valid JSON".
     No value is printed.

   Measured with the built command on a scratch home holding five bad files, `auth.json`'s two keys
   `sk-secret-1` and `sk-secret-2` among them:

   ```text
   <home>/auth.json:2: /openai: a name is repeated
   <home>/mcp.json:3: /mcpServers/a: a name is repeated
   <home>/models.json:3: /providers/x: a name is repeated
   <home>/permissions.json:2: /a: a name is repeated
   <home>/trust.json:2: not valid UTF-8
   ```

   The exit was 1, and no `sk-secret` was printed.
4. **Tests,** each passing, and each seen failing on its planted break (C9):

   | Test | Break | The failure |
   | :--- | :--- | :--- |
   | `TestAuthFileStrict` | the auth store's decoder and decode on the data policy | `invalid byte: Load accepted it`, `lone surrogate: Load accepted it` |
   | `TestSetKeyInvalidUTF8` | `SetKey`'s encode on the data policy | `SetKey accepted a key holding an invalid byte` |
   | `TestModelsJSONStrict` | `ParseModelsJSON`'s decode on the data policy | `ParseModelsJSON accepted an invalid byte` |
   | `TestMCPConfigStrict` | `members` and the six value decodes on the data policy | `Load` gave server `a` the command `x` with U+FFFD |
   | `TestDuplicateServerRefused`, `TestDuplicateFieldRefused`, `TestEditRefusesDuplicate` | `AllowDuplicateNames(true)` put back in `members` | `Load` gave the servers; `Remove edited a file holding a repeated name` |
   | `TestMCPListDuplicate` | the same | `mcp list` started both `a` servers, and printed no config error |
   | `TestConfigValidateJSONFiles` | `validate` without the JSON files | `exit 0` and no finding |
   | `FuzzNoDuplicateWritten` | `AllowDuplicateNames(true)` in `jsonx.Options()` | `encoded {"type":"message",…,"type":"other"}, which a strict reader refuses` |

   `FuzzNoDuplicateWritten` also ran 20 s against the finished code, with 2,142,401 executions and no failure. It
   lives in `jsonx/fuzz_test.go`, an external test package, which imports `session`.
5. **Notes for later phases:**
   * P9's `forbidigo` rule forbids v2's `MarshalEncode` and `UnmarshalDecode` outside `jsonx`.
     `internal/cli/mcp.go:181` calls `json.MarshalEncode`, and P8's step 1 calls `json.UnmarshalDecode`. `jsonx` will
     need thin wrappers for both before P9, or the rule exempts them. That is P8's or P9's to settle, as a deviation
     if it changes a step.
   * P9's `archtest` rule, that `jsonx` imports no package of this module, must count `jsonx`'s non-test imports only,
     since `jsonx/fuzz_test.go` imports `session`.
6. **G-local:**
   * `make pre-add-check` on the 16 changed Go files: clean; after the lint fixes below, on the 2 files they touched:
     clean;
   * `make preflight`: passed. Its first run stopped at golangci-lint's `modernize`, twice:
     `internal/cli/config.go:400`, an `errors.As` that `errors.AsType[*jsontext.SyntacticError]` replaces, and
     `internal/cli/config_test.go:396`, a range over `strings.Split` that `strings.SplitSeq` replaces. Both were fixes
     to the step's own code, not deviations;
   * `go test ./...`: 37 ok; `make race`: 37 ok;
   * `make perf`, since P2 touches `internal/cli`: passed, with ReplyRender 35.96 µs (baseline 30 µs), Reprint
     10.45 ms (baseline 9.4 ms), LineSessionStart 1.19 ms (baseline 830 µs, 5012 allocations) and KeyDraw 1.533 µs
     (baseline 1.5 µs).
   * LineSessionStart was above the 886 to 937 µs of 0035-PLAN's recent phases, so it was measured again against the tree before P1
     (`153ef60`) and P1 (`6b5af2c`), each from `git archive` into the scratchpad:
     * four interleaved rounds gave the base 945 to 1,036 µs, P1 1,026 to 1,125 µs and P2 1,068 to 1,141 µs, with
       the machine's load average between 24 and 29;
     * one more round each gave the base 1.232 ms, P1 982 µs and P2 1.072 ms;
     * the allocations per session start are 5,012, 5,011 and 5,011.

     The allocations are the same, the ranges overlap, and the slowest single run was the base's. They show no
     slowing by P2; the rise follows the machine's load. One `make perf` run under load failed every budget at about four times its baseline,
     KeyDraw among them, which P2 does not touch; it is not counted.
7. **G-hosts,** on the Linux machine and the Windows laptop, with 813 files shipped to each:
   * `go test`: passed;
   * race: passed, 37 ok;
   * `make check-records markdownlint`: passed;
   * preflight stops at `verify-build-metadata`, as on every copy. A first host run, before the lint fixes, stopped at
     the same two `modernize` findings;
   * `make perf`: passed on both. The Linux machine: ReplyRender 146.8 µs (baseline 110 µs), Reprint 34.95 ms
     (baseline 29 ms), LineSessionStart 4.00 ms (baseline 2.9 ms), KeyDraw 5.545 µs (baseline 5 µs). The Windows
     laptop: ReplyRender 28.91 µs (baseline 54 µs), Reprint 7.44 ms (baseline 12 ms), LineSessionStart 4.51 ms
     (baseline 6.9 ms), KeyDraw 1.215 µs (baseline 1.4 µs);
   * the conhost probe on the Windows laptop: passed.
