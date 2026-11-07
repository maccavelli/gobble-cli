---
status: complete
date: 2026-11-01
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
6. `make apidiff`, within preflight, reports only the added field. *(Deviation 12: `make apidiff` is a no-op below
   v1.0.0. The evidence is a diff of `go doc -all ./acpclient` between the phase's base and its tree.)*
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
     the MADR's Consequences name. *(Deviation 13: no input reaches the reason. The row's credential holds them,
     from an `auth.json` entry of an unknown type.)*

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
3. **Assert:** `grep -rlE '"encoding/json"$' --include=*.go .` returns nothing. *(Deviation 15: at P7 it returns
   only `internal/archtest/check.go`, P8's; P8 asserts nothing.)*
4. **Unchanged:** `session.golden`, `print-json.golden`, `print-stream-json.golden`, and every other golden; `make perf`
   (Confirmation 14).
5. G-local with `make perf`. G-hosts with `make perf` and `make probe-conhost`. **Commit.**

### P8 — `archtest`, the `inline` tags, the hatch's tests, and the docs (MADR D18, D11, D21, D23)

Files: `internal/archtest/check.go`, `acpserver/extensions.go`, `acpserver/personality.go`, `session/session_test.go`,
new `llm/llm_test.go`, `session/doc.go`, `docs/architecture.md`.

1. **`internal/archtest/check.go:68`** reads `go list -json`'s stream with `jsontext.NewDecoder(&stdout,
   jsonx.Options())` and `json.UnmarshalDecode` until `io.EOF`. `make archtest` passes, and its rules report what they
   reported before. *(Deviation 15: then `grep -rlE '"encoding/json"$' --include=*.go .` returns nothing.)*
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

Files: `.golangci.yml`, `internal/archtest/check.go`, `internal/archtest/check_test.go`. *(Deviation 17 adds
`jsonx/jsonx.go`, `jsonx/jsonx_test.go`, `internal/cli/mcp.go` and `docs/architecture.md`; deviation 19 adds
`internal/archtest/doc.go`.)*

1. **`depguard`**: a rule beside `logging` (`.golangci.yml:41-53`), for every file, tests included, denying
   `encoding/json$` with the message "use jsonx, on encoding/json/v2 (0039-MADR D24)".
2. **`forbidigo`**: a rule (`.golangci.yml:56-64`) forbidding `encoding/json/v2`'s `Marshal`, `MarshalWrite`,
   `MarshalEncode`, `Unmarshal`, `UnmarshalRead` and `UnmarshalDecode`, with `analyze-types: true`, its exact pattern
   settled when first seen failing. An exclusion keeps `jsonx/` and `_test.go` files out of it. *(Deviation 17: the
   two stream calls outside `jsonx`, `internal/cli/mcp.go`'s `MarshalEncode` and `internal/archtest/check.go`'s
   `UnmarshalDecode`, move first to `jsonx`'s new stream functions.)*
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

### P3 record (2026-10-31): the renames

The base is P2's `f38c1c5`. No deviation.

1. **The spellings** (step 1): each `json.RawMessage` or `stdjson.RawMessage` is `jsontext.Value`:
   * `acpclient/acptest/client.go`, 2; `acpclient/conn.go`, 1; `acpserver/extensions.go`, 2; `acpserver/grants.go`, 1;
   * `acpclient/compaction_test.go`, 3; `acpclient/personality_test.go`, 4; `acpclient/retry_test.go`, 3;
   * `acpclient/update.go`, 4: `Params` at `:105`, and the parameters of `retryOf`, `compactionOf` and
     `personalityOf`, which step 1's "each" covers in a file the phase lists. `update.go` keeps `encoding/json`, for
     the decodes and the encode P4 moves to `jsonx`, and gains `encoding/json/jsontext`.

   `acpserver/prompt.go` had no spelling left after P1. `acpclient/sessions_test.go`'s one spelling is P7's, with that
   file's v1 decode.
2. **No change at run time** (step 2): the types are identical. `go build ./...` and `go vet` passed, with every test
   unchanged. No test was added, so C9 has nothing to plant.
3. **Assert** (step 3): `grep` for `"encoding/json"` and `stdjson` over the 8 files and `acpserver/prompt.go` finds
   nothing (exit 1). `acpclient/update.go` holds no `RawMessage`.
4. **G-local:**
   * `make pre-add-check` on the 8 changed Go files: clean;
   * `make preflight`: passed;
   * `make race`: 37 ok;
   * no `make perf`, since P3 touches no `internal/cli` file.
5. **G-hosts,** on the Linux machine and the Windows laptop, with 813 files shipped to each:
   * `go test`: passed;
   * race: passed, 37 ok;
   * `make check-records markdownlint`: passed;
   * preflight stops at `verify-build-metadata`, as on every copy.

### Deviation 6 (2026-10-31): P4, the extension tests and `ExtensionMux`

* **Found** reading step 2 against acp-go-sdk2 `v1.28.0`. `ExtensionMux.HandleExtensionMethod` takes a call for a
  notification only when `acp.IsNotification(ctx)` says so (`extensions.go:121-122`), which is true only for a context
  the connection marked. The SDK exports no way to mark one. `acpclient/retry_test.go:19-33`,
  `compaction_test.go:18-29` and `personality_test.go:19-38` call `h.HandleExtensionMethod(t.Context(), ...)` directly,
  so under step 2 each would be answered `-32601`. Two of their assertions are also D12's changes: an unregistered
  notification is ignored, no longer `-32601`, and bad params are logged with no callback, no longer `-32602`.
* **The owner's answer:** "Real connection pair (Recommended)". The options not taken were a per-method function per
  notification tested directly with one routing test, and holding P4 for an SDK issue.
* **Decision.** A test helper in `acpclient` wires an agent-side connection over `io.Pipe` to a client-side connection
  serving the `handler`, with a capturing `slog` handler, and sends real notifications. The three tests keep their
  assertions of the decoded update, its params, and nothing for another session, and assert D12's two changes as the
  new behaviour.
* **Files added:** none; the three test files are P4's.

### Deviation 7 (2026-10-31): P4, two integers in `acptest`'s error data

* **Found** counting step 4's 34 calls. D14 names one test-support integer, `"update": i`
  (`acpclient/acptest/script.go:159`). `acpclient/acptest/client.go:200`'s `"exitCode": n` is a second.
  `acpclient/acptest/script_test.go:56` asserts the first's bytes, `"update":0`, which `strconv.Itoa` makes
  `"update":"0"`. That file is not in P4's list.
* **The owner's answer:** "Itoa both, update test (Recommended)". The option not taken was a typed struct for the two
  sites, keeping the numbers.
* **Decision.** Both integers go through `strconv.Itoa`, and `script_test.go:56` asserts `"update":"0"`. The MADR's D14
  is amended. Also recorded: `acpclient/conn.go:435`, one of the 34, goes with `HandleExtensionMethod` in step 2, so
  33 calls are rebuilt with `jsonx.Strings`.
* **Files added:** `acpclient/acptest/script_test.go`.

### Deviation 8 (2026-10-31): P4, two tests read `_meta.gobble` as a Go map

* **Found** reading the `_meta.gobble` tests against step 5. The `acpserver` tests read what came over the wire, which
  the SDK decodes into `map[string]any`, so they pass unchanged. `acpclient/sessions_test.go:180`
  (`NewSessionOptions.meta()`) and `:285` (the permission answer's `_meta`) type-assert the writer's own Go value,
  which becomes a `jsontext.Value`. Step 6 says every existing `_meta.gobble` test passes unchanged.
* **The owner's answer:** "Read via jsonx.Member (Recommended)". The option not taken was a v1 round trip in the tests.
* **Decision.** The two tests read the members with `jsonx.Member`, with the same assertions. The MADR's Confirmation
  21 is amended. Also recorded: each `_meta.gobble` struct declares its fields in sorted JSON-name order, the order v1
  writes today's maps in, so the wire's bytes do not change.
* **Files added:** none; `acpclient/sessions_test.go` is P4's.

### Deviation 9 (2026-10-31): P4, `acpserver.Capabilities()` returns an error

* **Found** writing step 5. D15's writer at `acpserver/agent.go:385`, `_meta.gobble.extensions`, is inside
  `func Capabilities() acp.AgentCapabilities` (`:371`), which is exported and has no error result. `jsonx.SetMember`
  returns an error, and D7 discards none. Its one caller is `Initialize` (`:400`), which returns an error. The other
  seven writers are in functions that return an error already, or gain one with no exported change.
* **The owner's answer:** "Return an error (Recommended)". The option not taken was a `jsonx` helper writing a string
  array through `Quote`, which cannot fail, keeping the signature.
* **Decision.** `Capabilities()` returns `(acp.AgentCapabilities, error)`, and `Initialize` answers its error as an
  internal error whose data is built by `jsonx.Strings`. The MADR's D15 is amended. `apidiff` is a no-op below
  v1.0.0, so no gate reports the change; this entry and the amendment are its record.
* **Files added:** none; `acpserver/agent.go` is P4's.

### Deviation 10 (2026-10-31): P4, `errText` reads an in-process error's data

* **Found** running the tests after step 5. `errText` (`acpserver/extensions_test.go:121`) type-asserts `re.Data` as
  `map[string]any`. Sixteen of its seventeen callers read errors decoded from the wire, which are maps, and pass.
  `TestClaimTurnAdmitsOne` (`:628`) calls `a.claimTurn` in process, so its data is the `jsontext.Value` that
  `jsonx.Strings` built, and it failed: `refusal -32600 ""`. Step 6 and Confirmation 20 say the `errText` tests pass
  unchanged.
* **The owner's answer:** "errText decodes via jsonx (Recommended)". The option not taken was racing real prompts over
  the wire in that test.
* **Decision.** `errText` encodes `re.Data` with `jsonx.Marshal` and decodes it into `map[string]any` with
  `jsonx.Unmarshal`. No caller's assertion changes. The MADR's Confirmation 20 is amended.
* **Files added:** none; `acpserver/extensions_test.go` is a test of P4's `acpserver/extensions.go`.

### P4 record (2026-10-31): everything gobble defines, on v2

The base is P3's `4f76649`. Deviations 6 to 10 changed steps 2, 4, 5 and 6, and added
`acpclient/acptest/script_test.go`.

1. **`jsonx`** (step 1) gains three functions:
   * `Strings(map[string]string) jsontext.Value`, keys sorted with `slices.Sorted`, as v2 sorts a map of strings, and
     each key and value through `Quote`. It cannot fail. Where two keys holding invalid bytes are repaired to the same
     name, the later one is left out, because `Marshal` refuses that map for its repeated name and `Strings` writes no
     repeated name;
   * `SetMember(meta, key, v) (map[string]any, error)`, a copy of `meta` with `key` set to `Value(v)`;
   * `Member[T](meta, key) (T, bool)`, `meta[key]` re-encoded and decoded into `T`, false when missing, unencodable
     or mistyped.
2. **Receiving** (step 2, deviation 6): `acpclient`'s `handler` has an `acp.ExtensionMux`, registered on first use
   through a `sync.Once`, so the `&handler{}` literals in tests keep working.
   * `HandleExtensionMethod` hands every call to the mux.
   * The three `acp.Notify[jsontext.Value]` registrations call `retryOf`, `compactionOf` and `personalityOf`, which
     decode with `jsonx.Unmarshal`. A decode error is returned for the connection to log.
   * `deliver` routes to `h.sessions` and `h.pending` as before.
   * `acpclient/update.go` keeps `encoding/json` for `translate`'s encode, which is P5's.
3. **Sending** (step 3): `updates.notify` encodes `params` with `jsonx.Value`. On error it keeps
   `"<method>: <err>"` as `u.err` and sends nothing. Its eleven callers are unchanged. `Session.SetName` passes
   `jsonx.Value(setNameParams{...})` and returns its error.
4. **Error data** (step 4, deviation 7):
   * 33 calls build `jsonx.Strings(map[string]string{...})`. The 34th, `acpclient/conn.go:435`, went with
     `HandleExtensionMethod`'s switch;
   * `acp.SessionId` values are converted with `string(...)` at five sites: `agent.go` four, `extensions.go` one;
   * both integers go through `strconv.Itoa`: `acpclient/acptest/script.go:159`'s `update` and
     `acpclient/acptest/client.go:200`'s `exitCode`. `acpclient/acptest/script_test.go:56` asserts `"update":"0"`;
   * four new error sites from step 5 build their data the same way: `Initialize`, `withCleared`, `LoadSession` and
     `ResumeSession`. That is 37 calls in all.
5. **`_meta.gobble`** (step 5, deviations 8 and 9):
   * **Eight writers, each a struct:** `questionMeta` with `ruleMeta`, `capabilitiesMeta`, `clearedGobble` with
     `clearedQueues`, `resumedGobble`, `switchMeta`, `newSessionMeta` (client), `mentionsMeta` and `feedbackMeta`.
     * Each sets its value with `jsonx.SetMember`.
     * Each declares its fields in sorted JSON-name order, the order in which v1 wrote the map it replaces.
     * The writers that had no error now return one: `question.meta`, `Capabilities` (deviation 9), `clearedMeta`,
       `resumedMeta`, `NewSessionOptions.meta` and the client's `unscanned`. `copyTo` and `RequestPermission`
       already returned one.
     * A cancelled turn's meta is set by a new `withCleared`, which `cancelled` also uses. That keeps `Prompt`'s
       cognitive complexity within `gocognit`'s 50.
   * **Seven readers, member by member:** `jsonx.Member[map[string]any]` for `gobble`, then `jsonx.Member[T]` for
     each member, in `acpclient/conn.go` (`switchTo`, `gobbleMarks`), `acpclient/sessions.go` (`title`, `messages`),
     `acpserver/grants.go` and `question.go` (`feedback`), `acpserver/mentions.go` (`mentions`) and
     `acpserver/config.go` (`readNewSessionMeta`).
     * `gobbleMarks`'s hand-made `saves` loop and its `float64`/`int` switch became `Member[[]string]` and
       `Member[int]`.
   * **Deviation 8's two tests** in `acpclient/sessions_test.go` read through `jsonx.Member`, with the same
     assertions. **Deviation 10's** `errText` reads `re.Data` through `jsonx`.
6. **Tests,** each passing, and each seen failing on its planted break (C9), on a scratch copy:

   | Test | Break | The failure |
   | :--- | :--- | :--- |
   | `TestNotificationDecodedWithV2` | `retryOf`'s decode with `jsontext.AllowInvalidUTF8(false)` | `lone surrogate: updates [], want one with cause "offline�"` |
   | `TestNotifyFramesUnchanged` | `notify` encoding with `jsonx.MarshalStrict` | `_gobble/retry "x\xffy": notify wrote error: _gobble/retry: jsontext: invalid UTF-8 within "/error"`, and the same for compaction and personality |
   | `FuzzStrings` | `Strings` over `slices.Collect(maps.Keys(m))`, unsorted | `Strings(...) = {"reason":"boom","sessionId":"s1","error":"x"}, Marshal = {"error":"x","reason":"boom","sessionId":"s1"}` |
   | `TestMetaMemberMistyped` | `gobbleMarks` decoding the whole object into one struct | `asked {… Limit: Repeated:0 …}, want the limit and no repeated` |
   | `TestRetryNotification`, `TestCompactionNotification`, `TestPersonalityNotification` (deviation 6) | the three handlers returning `nil` on a decode error | each `bad params: … want it logged and not delivered`; `TestNotificationDecodedWithV2`'s `a string attempt` too |
   | `TestMetaWireBytes` (`acpserver`), added | `questionMeta`'s `Limit` declared before `Hides` | `question _meta {"gobble":{"limit":"L","hides":true,…}}, want {"gobble":{"hides":true,"limit":"L",…}}` |
   | `TestMetaWireBytes` (`acpclient`), added | `newSessionMeta`'s `SessionID` declared first | `session/new _meta {"gobble":{"sessionId":"i",…}}, want {"gobble":{"forkFrom":"f",…}}` |

   * **The two `TestMetaWireBytes` tests** were added to hold deviation 8's claim that the wire's bytes do not change.
     `session.golden` pins only the capabilities' `_meta`.
     * Their expected bytes were checked at the base. A probe on a scratch copy of `4f76649` encoded the old writers'
       maps with v1, and printed the same five strings for the agent's writers.
     * One case was first written wrong: a question with no marks expected `null`. v2 writes a nil map as `{}`, and
       the SDK leaves a nil `_meta` out. So the case now asserts that `meta()` returns nil.
   * **The notification tests** wait through `session/new`. `notifyPair` sends the notifications from the agent, then
     `session/new` from the client.
     * The SDK returns a response to its caller only after the notifications received before it are handled and
       logged.
     * A first version waited on a request from the agent. That raced, because the client handles requests
       concurrently with notifications, and the log assertion failed once.
   * **A bad notification over a real connection** is a mistyped member, such as `attempt` given as a string. The SDK
     refuses to send syntactically broken params.
   * **`TestNotifyFramesUnchanged` asserts** that the SDK's frame is a notification for the method and session, so two
     identical errors cannot pass it.
   * **`FuzzStrings` ran** 15 s while it was written, 249,212 executions, and 20 s on the finished code, 735,795
     executions, with no failure.
7. **A first try in step 5** tagged `clearedQueues`' slices `format:emitnull`, to keep a nil queue `null`. v2
   refuses that option on a `[]string` field, so `clearedMeta` failed, and `TestCancelClearsQueue` saw `{}`.
   `queue.clear` never returns nil, since an empty queue is `[]string{}`, which v1 wrote as `[]`. So the fields are
   plain `[]string`. That was a fix to the step's own code, not a deviation.
8. **Assert by search:**
   * no SDK error constructor in `acpserver` or `acpclient` takes a `map[string]any`;
   * no `map[string]any{gobble:` or `{metaGobble:` writer, and no `Meta[gobble].(` or `Meta[metaGobble].(` reader,
     outside tests;
   * `acpclient/conn.go` holds no `NewMethodNotFound` or `NewInvalidParams`;
   * 11 `notify` callers;
   * `session.golden` and every other golden are unchanged.
9. **G-local:**
   * `make pre-add-check` on the 27 changed Go files: clean; after the `gocognit` fix, on the 2 files it touched:
     clean;
   * `make preflight`: passed. Its first run stopped at golangci-lint's `gocognit`: `(*Agent).Prompt` at 53, over 50,
     from step 5's error branch. `withCleared` fixed it; that was a fix to the step's own code, not a deviation;
   * `go test ./...`: 37 ok; `make race`: 37 ok;
   * no `make perf`, since P4 touches no `internal/cli` file.
10. **G-hosts,** on the Linux machine and the Windows laptop, with 813 files shipped to each:
    * `go test`: passed;
    * race: passed, 37 ok;
    * `make check-records markdownlint`: passed;
    * preflight stops at `verify-build-metadata`, as on every copy.
11. **Notes for later phases:**
    * `acpclient/sessions.go:45` (`sessionError`) reads an agent error's data by `re.Data.(map[string]any)`. Over the
      wire that is the map the SDK decoded, so it works. P7 or P8 may want it read through `jsonx`, to match.
    * `acpserver/auth_test.go:87` reads error data the same way, from the wire.

### Deviation 11 (2026-10-31): P5, four test assertions read `Update.Params` unasked

* **Found** reading `Update.Params`' readers before step 2. The one production reader is `stream-json`
  (`internal/cli/print.go:113`), which step 3 has ask for it. Four test assertions, in two files outside P5's list,
  read it from a connection that does not:
  * `acpclient/conn_test.go:93-94` and `:127` (`TestTranslate`) and `:330` (`TestUpdatesBeforeNewSessionReturns`)
    start with `acpclient.Options{}`;
  * `internal/cli/models_test.go:310` (`replies`) decodes a `config_option_update`, which has no typed field, from the
    `Params` of a connection started by `e.startAgent` (`:231`).

  Under step 2 each would read nil `Params` and fail. That is D16 working, not a defect. The MADR's statement that
  `Params` is read in one place holds for production code, so it is not amended.
* **The owner's answer:** "Tests ask for Params (Recommended)". The option not taken was to encode `Params` always,
  amending D16.
* **Decision.**
  * The two `acpclient` tests start with `acpclient.Options{Params: true}`, and `TestTranslate` also asserts that
    `Options{}` leaves `Params` nil.
  * `models_test.go` starts through `startAgentWith(ctx, nil, true)`, step 3's new argument.
  * Every existing assertion is kept.
* **Files added:** `acpclient/conn_test.go`, `internal/cli/models_test.go`.

### Deviation 12 (2026-10-31): P5, `line.go`, and step 6's `apidiff`

* **Found** running P5's gates.
  * Step 3 gives `startAgentWith` a `params` argument. Its other caller, `internal/cli/line.go:126`, must pass
    `false`, and `line.go` is not in P5's list.
  * `make apidiff` prints `apidiff: no-op below v1.0.0` (the `Makefile`; also deviation 9), so step 6 cannot be
    shown by it.
  * Also found, and not a deviation: `startAgent`'s one caller was `internal/cli/models_test.go`, which deviation 11
    moved to `startAgentWith(ctx, nil, true)`. staticcheck then reported `startAgent` unused (U1000), so it is removed
    from `internal/cli/agent.go`, a file of P5's.
* **The owner's answers:** "Add line.go to P5 (Recommended)" and "go doc diff (Recommended)". The options not taken
  were a second start function for the stream, and holding P5 for an `apidiff` comparison.
* **Decision.**
  * `line.go` passes `false`: the line mode reads no `Params`.
  * Step 6's evidence is the diff of `go doc -all ./acpclient` between the base and the tree, and step 6 is annotated.
* **Files added:** `internal/cli/line.go`.

### P5 record (2026-10-31): `translate`

The base is P4's `5a20d06`. Deviations 11 and 12 added `acpclient/conn_test.go`, `internal/cli/models_test.go` and
`internal/cli/line.go`, and annotated step 6.

1. **`acpclient.Options.Params`** (step 1, PQ2 (a)): "encodes `Update.Params`, for a client that passes the stream
   through". The `handler` keeps it. `Update.Params`' doc says it is set only when `Options.Params` is, and that for
   `KindRetry`, `KindCompaction` and `KindPersonality` it is the notification's params, as received.
2. **`translate(n, params)`** (step 2) encodes only when asked, with
   `jsonx.Marshal(n, jsontext.EscapeForHTML(true), jsontext.EscapeForJS(true))`. `acpclient/update.go` imports no
   `encoding/json`.
3. **Print mode** (step 3, deviation 12):
   * `runPrint` builds its sink first, and calls `startAgentWith(ctx, e.printPermission, sink.stream)`;
   * `line.go` passes `false`;
   * `startAgent`, whose one caller deviation 11 moved, is removed, and its doc folded into `startAgentWith`'s.
4. **`writeJSON`** (step 4) encodes with `jsonx.Marshal`, with no escaping options. Its comment states the two styles,
   by D16. `internal/cli/print.go` imports no `encoding/json/v2`.
5. **Tests** (step 5, deviation 11):

   | Test | Break | The failure |
   | :--- | :--- | :--- |
   | `TestPrintStreamJSONEscapes`, with the new `print-stream-json-escapes.golden` | `translate` without `EscapeForJS` | the second update line `"text":"<U+2028>d<U+2029>"`, raw, where the golden has `"text":" d "` |
   | `TestParamsOnlyWhenAsked` (`acpclient/conn_test.go`), added by deviation 11 | `translate` encoding whatever `params` says | `updates = [{Kind:commands … Params:{…}}], want one KindCommands with no params` |

   * **The golden is not only the code's own output.** The same test was added to a scratch copy of the base,
     `5a20d06`, where `translate` encoded with v1's `json.Marshal`, and run there with `-update`. Its golden was
     identical, by `cmp`, to the one this phase commits.
     * Its update lines escape `<`, `&`, `>`, U+2028 and U+2029.
     * Its result line writes them raw.
   * `print-stream-json.golden` and `print-json.golden` are unchanged, and so is every other golden.
   * Deviation 11's three changed starts keep every assertion. `TestTranslate` and
     `TestUpdatesBeforeNewSessionReturns` ask for `Params: true`, and `models_test.go` starts through
     `startAgentWith(t.Context(), nil, true)`.
6. **The exported API** (step 6, deviation 12). `make apidiff` prints `apidiff: no-op below v1.0.0`. Instead,
   `go doc -all ./acpclient` was diffed between a scratch copy of `5a20d06` and the tree. The diff is the added
   `Options.Params` field with its comment, and `Update.Params`' longer doc comment, and nothing else.
7. **G-local:**
   * `make pre-add-check` on the 8 changed Go files: clean; after `startAgent`'s removal, on `internal/cli/agent.go`:
     clean;
   * `make preflight`: passed. Its first run stopped at staticcheck's U1000, `startAgent` unused, on linux, darwin and
     windows. That is deviation 12's third finding;
   * `go test ./...`: 37 ok; `make race`: 37 ok;
   * `make perf`: passed, with ReplyRender 33.54 µs (baseline 30 µs), Reprint 9.89 ms (baseline 9.4 ms),
     LineSessionStart 927.5 µs (baseline 830 µs), and KeyDraw 1.482 µs (baseline 1.5 µs). The machine's load average
     was about 23.
   * LineSessionStart made 5,009 allocations, against 5,011 at P2, the last phase that ran `make perf`. That is
     consistent with the line mode no longer encoding each update's params; it was not measured further.
8. **G-hosts,** on the Linux machine and the Windows laptop, with 814 files shipped to each:
   * `go test`: passed;
   * race: passed, 37 ok;
   * `make check-records markdownlint`: passed;
   * preflight stops at `verify-build-metadata`, as on every copy;
   * `make perf`: passed on both. The Linux machine: ReplyRender 111.8 µs (baseline 110 µs), Reprint 28.34 ms
     (baseline 29 ms), LineSessionStart 3.14 ms (baseline 2.9 ms), KeyDraw 4.473 µs (baseline 5 µs). The Windows
     laptop: ReplyRender 29.03 µs (baseline 54 µs), Reprint 7.26 ms (baseline 12 ms), LineSessionStart 4.57 ms
     (baseline 6.9 ms), KeyDraw 1.212 µs (baseline 1.4 µs);
   * the conhost probe on the Windows laptop: passed.

### Deviation 13 (2026-11-01): P6, `TestProvidersJSON` pins the credential, not the reason

* **Found** reading P6's files.
  * A row's reason is `acpserver.StateOf`'s (`acpserver/choice.go:71-95`): fixed words joined with the provider's id
    and an HTTP status. No input puts `<` or `&` in it, so step 3's test cannot be written as stated.
  * A row's credential can hold them. `credentialSource` (`internal/cli/auth.go:360`) quotes an `auth.json` entry's
    type when gobble does not use it. A scratch build of P5's tree, given an `auth.json` entry of type `a<b&c>`, printed
    that row's credential with `\u003c`, `\u0026` and `\u003e`. That is the behaviour before P6.
* **The owner's answer:** "Pin the credential (Recommended)". The option not taken was an encoding function split out
  of `providerscmd.go` for a unit test, a seam for the test alone that leaves the command unpinned.
* **Decision.** `TestProvidersJSON` writes an `auth.json` entry of type `a<b&c>` and pins that row's credential,
  unescaped. Step 3 is annotated. The MADR is unchanged: D17 and its Consequences speak of the strings, not the reason.
* **Files added:** none.

### Deviation 14 (2026-11-01): P6, a Windows teardown failure in `TestLoginAsksThroughQuestions`

* **Found** in P6's G-hosts. On the Windows laptop, `go test -count=1 ./...` failed once:
  `--- FAIL: TestLoginAsksThroughQuestions (0.21s)` with `testing.go:1619: TempDir RemoveAll cleanup: unlinkat
  <TMP>\TestLoginAsksThroughQuestions<n>\002: The directory is not empty.` The `002` directory is the test's
  `GOBBLE_HOME` (`newModelHarness`, `internal/cli/line_test.go:88`). Something wrote into it after the test ended. The
  race run on the same host, which runs the same test, passed.
* **Not reproduced,** on the Windows laptop: `-run '^TestLoginAsksThroughQuestions$' -count=30` gave 0 failures, and
  `go test -count=5 ./internal/cli/` gave 0, each on P6's copy and on P5's commit `cfa514f`.
* **Not P6's code.** The test lists offline, so `writeKept` is never reached. `readKept` finds no kept file and returns
  before it decodes. The late writer is not identified. One unverified candidate is a model listing the harness does not
  wait for, logging after `logs.close`.
* **The owner's answer:** "Record as pre-existing, rerun (Recommended)". The options not taken were diagnosing and fixing
  the harness inside 0039, and holding P6 for a record of its own.
* **Decision.** The failure is recorded as pre-existing and out of this plan's scope. Windows' `go test -count=1 ./...`
  was rerun on P6's copy: 37 ok. The flake waits for a record of its own; no assertion or test was changed.
* **Files added:** none.
* *Note (2026-11-07): the late writer is identified and removed by [0046-MADR-own-background-listings-and-keep-tests-hermetic.md](0046-MADR-own-background-listings-and-keep-tests-hermetic.md). A listing `settled` could not see wrote kept listings into the next test's home, or the user's cache, and the SDK's background model facts left an OAuth refresh lock. Its PLAN's P4 ran this test 60 times under `-race` on macOS and on the Windows laptop with no failure.*

### P6 record (2026-11-01): the local files and the providers output

The base is P5's `cfa514f`. Deviation 13 changed which field `TestProvidersJSON` pins; deviation 14 recorded a Windows
teardown failure outside P6. Neither added a file.

1. **`internal/cli/choice.go`** (step 1):
   * `loadRecent` and `readKept` decode with `jsonx.UnmarshalStrict`;
   * `rememberModel` encodes with `jsonx.MarshalStrict(…, jsontext.WithIndent("  "))`;
   * `writeKept` encodes with `jsonx.MarshalStrict`;
   * the v1 import is gone.

   A scratch program compared the writes, P5's v1 against P6's options, for the same values. `recent-models.json`,
   indented, was byte-identical, and so was a kept listing with usable ids. A kept listing whose usable ids are nil
   writes `"usable":[]` where v1 wrote `"usable":null`, as D17 says.
2. **`internal/cli/providerscmd.go`** (step 2) encodes with `jsonx.Marshal(rows, jsontext.WithIndent("  "))`. The v1
   import is gone.
3. **Tests** (step 3, Confirmation 10, deviation 13). The three fixtures, `internal/cli/testdata/recent-models-v1.json`,
   `recent-models-v1-null.json` and `catalog-v1.json`, were written by a scratch program with v1's `json.MarshalIndent`
   and `json.Marshal`, on `choice.go`'s types as P5 had them, and copied unchanged:
   * `recent-models-v1.json` holds two models, one with v1's `\u003c`, `\u003e` and `\u0026`;
   * `recent-models-v1-null.json` is `{"recent": null}`, as v1 wrote a nil slice;
   * `catalog-v1.json` is a kept listing with `"usable":null`, and a `fetchedAt` with nanoseconds and a −04:00 offset.

   | Test | Break | The failure |
   | :--- | :--- | :--- |
   | `TestRecentModelsV1File` | `Recent` retyped to a planted `refuseNull`, a `[]string` whose `UnmarshalJSONFrom` refuses `null` | `recent-models-v1-null.json`: `recent [], warnings "Warning: recent-models.json: json: cannot unmarshal into Go cli.refuseNull within \"/recent\": null refused; the recent models are not read\n"; want [] and none` |
   | `TestKeptListingV1File` | `Usable` retyped the same way | `kept [], usable [], at 0001-01-01 00:00:00 +0000 UTC; want [c1 c2], none, at 2026-10-31 09:30:00.123456789 -0400 -0400` |
   | `TestProvidersJSON` | `jsontext.EscapeForHTML(true)` planted in `providerscmd.go`'s encode | the row printed `"credential": "auth.json, a \"a\u003cb\u0026c\u003e\" entry gobble does not use"`, where the test wants `a<b&c>` |

   * The `refuseNull` break is needed because v2, like v1, decodes `null` into a slice, an array, a `time.Time` or a
     struct as its zero value. A scratch probe showed none of them refuses it.
   * `recent-models-v1.json`, which holds no `null`, passed under that break, as expected.
4. **G-local:**
   * `make pre-add-check` on the 4 changed Go files: clean;
   * `make preflight`: passed;
   * `go test ./...`: 37 ok; `make race`: 37 ok;
   * `make perf`: passed, with ReplyRender 34.04 µs (baseline 30 µs), Reprint 10.13 ms (baseline 9.4 ms),
     LineSessionStart 1.158 ms (baseline 830 µs), and KeyDraw 1.485 µs (baseline 1.5 µs). The machine's load average was
     about 25. LineSessionStart made 5,011 allocations, against 5,009 at P5 and 5,011 at P2.
5. **G-hosts,** on the Linux machine and the Windows laptop, with 817 files shipped to each:
   * `go test`: passed on the Linux machine. On the Windows laptop it failed once, in `TestLoginAsksThroughQuestions`'s
     teardown (deviation 14), and passed on its rerun, 37 ok;
   * race: passed on both, 37 ok;
   * `make check-records markdownlint`: passed on both;
   * preflight stops at `verify-build-metadata`, as on every copy;
   * `make perf`: passed on both. The Linux machine: ReplyRender 152.24 µs (baseline 110 µs), Reprint 30.38 ms
     (baseline 29 ms), LineSessionStart 3.39 ms (baseline 2.9 ms), KeyDraw 5.547 µs (baseline 5 µs). The Windows
     laptop: ReplyRender 27.75 µs (baseline 54 µs), Reprint 7.25 ms (baseline 12 ms), LineSessionStart 3.73 ms
     (baseline 6.9 ms), KeyDraw 1.199 µs (baseline 1.4 µs);
   * the conhost probe on the Windows laptop: passed.

### Deviation 15 (2026-11-01): P7, step 3's assertion and `internal/archtest/check.go`

* **Found** at P7's start. Step 3 asserts that no Go file imports `encoding/json`, but `internal/archtest/check.go:68`
  (`json.NewDecoder(&stdout)`) still does, and it is P8's file (P8 step 1). The plan's own ordering, not the code, is
  the cause.
* **The owner's answer:** "Assert at P7, check at P8 (Recommended)". The option not taken was moving P8 step 1 into P7.
* **Decision.** P7's assertion is that the grep returns only `internal/archtest/check.go`. P8 asserts that it returns
  nothing, once step 1 is done. P7 step 3 and P8 step 1 are annotated.
* **Files added:** none.

### Deviation 16 (2026-11-01): P7, the local `make perf` under another project's load

* **Found** in P7's G-local. `make perf` failed: `ReplyRender: 102.076µs per op, more than 3 times its baseline 30µs`
  (`internal/cli/perf_test.go:88`). The Mac's load average was 82, mostly another project's test processes. The
  allocations matched P6's: ReplyRender 257, Reprint 49,696.
* **Measured.** Three rounds alternating P7's tree and P6's commit `ec6298e` on the Mac, at load 34 to 35, all passed and
  overlapped: ReplyRender 38 to 47 µs against 40 to 43 µs, LineSessionStart 1.31 to 1.39 ms against 1.32 to 1.83 ms.
* **The owner's answer:** "run the perf on the linux host since the macos host is too busy".
* **Decision.** P7's perf evidence is the Linux machine's, two rounds alternating P7's copy and P6's commit, in the P7
  record. The budget and the test are unchanged.
* **Files added:** none.

### P7 record (2026-11-01): the remaining data sites, and the test files

The base is P6's `ec6298e`. Deviation 15 changed step 3's assertion; deviation 16 moved the perf evidence to the Linux
machine. Neither added a file.

1. **The data sites** (step 1). Each `json.Marshal` and `json.Unmarshal` became `jsonx`'s, with its options kept, 66
   calls in 33 files, counted by an asserted script:
   * the files and lines step 1 names, at their current lines (C2);
   * `acpserver/history.go`'s six `json.Unmarshal` calls (now `:79`, `:96`, `:107`, `:273`, `:293`, `:435`). Step 1 does
     not list them, but its Files line covers "every D4 row not done in P1 to P6", and `history.go`'s D4 row names
     `jsonx.Unmarshal`. P1 did the rest of that row;
   * the `jsontext` decoders at `compaction/serialize.go:231` and `session/migrate.go:103`, and the encoder at
     `session/migrate.go:105`, take `jsonx.Options()`. With `session/migrate.go`, step 1 changed 34 files.

   Kept as they were:
   * `json.Deterministic(true)` where a site passes it (`internal/config/schema.go:66`, `mcpclient/tool.go:77` and
     `:136`, `internal/cli/config.go:200`). It repeats `jsonx.Options()`, which the plan keeps as written;
   * `internal/cli/mcp.go:182`'s `json.MarshalEncode` onto P2's strict encoder, which no step names. `jsonx` has no
     wrapper for it, which P9's `forbidigo` will need (the P5 record's notes).
2. **The seven test files that still imported v1** (step 2):
   * `acpclient/sessions_test.go`: `nameAgent.HandleExtensionMethod` takes `jsontext.Value`, and decodes with v2;
   * `acpserver/autocompact_test.go`: `sameCompactionNotices` compares with `reflect.DeepEqual`, in place of two v1
     encodes. The notices' values are strings, and `tokensBefore` is removed before comparing when the want is -1;
   * `acpserver/convert_test.go`: `TestRawInputOnTheWire` calls the SDK value's own `MarshalJSON`;
   * `acpserver/retry_test.go`, `internal/cli/models_test.go`, `internal/cli/personalitycmd_test.go`: v2 decodes into
     the same values, all of whose names are tagged;
   * `internal/cli/providerscmd_test.go`: `TestProvidersCheckAndJSON` decodes into `[]providerRow`. Its untagged
     anonymous struct matched `id` and `signIn` only through v1's case-insensitive matching.

   | Break | The failure |
   | :--- | :--- |
   | `TestAutoCompactPauseTurn`'s expected `"paused": "turn"` changed to `"session"` | `autocompact_test.go:464: notices [… map[error:Summarization failed: llm: offline paused:turn phase:end reason:threshold] …], want [… map[error:Summarization failed: llm: offline paused:session phase:end reason:threshold] …]` |
3. **Assert** (step 3, deviation 15): `grep -rlE '"encoding/json"$' --include=*.go .` returns only
   `internal/archtest/check.go`.
4. **Unchanged** (step 4): `go test` changed no golden or `testdata` file; `git status` showed none.
5. **G-local:**
   * `make pre-add-check` on the 41 changed Go files: clean;
   * `make preflight`: passed;
   * `go test ./...`: 37 ok; `make race`: 37 ok;
   * `make perf`: deviation 16.
6. **G-hosts,** on the Linux machine and the Windows laptop, with 817 files shipped to each:
   * `go test`: passed on both; race: passed on both, 37 ok;
   * `make check-records markdownlint`: passed on both;
   * preflight stops at `verify-build-metadata`, as on every copy;
   * `make perf`: passed on both. The Linux machine: ReplyRender 179.80 µs (baseline 110 µs), Reprint 49.98 ms
     (baseline 29 ms), LineSessionStart 7.47 ms (baseline 2.9 ms), KeyDraw 6.333 µs (baseline 5 µs), its load not
     recorded. The Windows laptop: ReplyRender 30.94 µs (baseline 54 µs), Reprint 7.47 ms
     (baseline 12 ms), LineSessionStart 4.16 ms (baseline 6.9 ms), KeyDraw 1.307 µs (baseline 1.4 µs);
   * the conhost probe on the Windows laptop: passed.
7. **Perf on the Linux machine** (deviation 16), 4 CPUs, alternating P7's copy and P6's commit, every run passing:

   | Round | Tree | Load | ReplyRender | Reprint | LineSessionStart | KeyDraw |
   | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
   | 1 | P7 | 0.29 | 130.17 µs | 33.00 ms | 4.22 ms | 4.984 µs |
   | 1 | P6 | 0.78 | 100.62 µs | 27.85 ms | 3.19 ms | 3.831 µs |
   | 2 | P7 | 1.67 | 114.45 µs | 28.11 ms | 3.25 ms | 4.438 µs |
   | 2 | P6 | 1.61 | 103.83 µs | 27.90 ms | 3.63 ms | 3.887 µs |

   The baselines are ReplyRender 110 µs, Reprint 29 ms, LineSessionStart 2.9 ms and KeyDraw 5 µs. The allocations
   matched across the two trees: ReplyRender 257, Reprint 49,696, LineSessionStart 5,017 to 5,020.

### P8 record (2026-11-01): `archtest`, the `inline` tags, the hatch's tests, and the docs

The base is P7's `fe671d6`. No deviation; deviation 15's assertion is met here.

1. **`internal/archtest/check.go`** (step 1) reads `go list -json`'s stream with
   `jsontext.NewDecoder(&stdout, jsonx.Options())` and `json.UnmarshalDecode`, until `errors.Is(err, io.EOF)`.
   * `make archtest`: passed.
   * **Its rules report what they reported before.** A scratch test, added to a copy of P7's commit and to a copy of
     the tree, wrote `loadGraph`'s packages (import path, error, imports, test and external test imports, and the size
     of `Deps`) and each of rules 1 to 9's violations. The two dumps differ in one line, `internal/archtest`'s own
     imports, which now hold `encoding/json/jsontext`, `encoding/json/v2`, `errors` and `jsonx` in place of
     `encoding/json`. Both list 65 packages, and every rule reports `[]` in both.
   * **Deviation 15's assertion:** `grep -rlE '"encoding/json"$' --include=*.go .` returns nothing (exit 1).
2. **The eight `json:",inline"` tags** (step 2) are dropped: seven in `acpserver/extensions.go`, one in
   `acpserver/personality.go`, by an asserted script; `gofmt` realigned the fields. `go test ./acpserver/` passed
   unchanged, the `_gobble/*` and personality tests with it (Confirmation 17).
3. **The hatch's tests** (step 3, Confirmation 11):
   * `TestRawNullAndAbsent` (`session/session_test.go`) reads and writes seven lines through `jsonx`: a
     `context_edit` with `replacement` absent, `null` and an object, and a `bashExecution` message with `exitCode`
     absent, `null`, `7` and an object. Absent reads back with `len` 0; each line is written back byte for byte;
   * `TestEmptyRawValue` plants `jsontext.Value{}` on each field and wants `io.ErrUnexpectedEOF` from the marshal,
     then wants `jsonx.From([]byte{})` on it left out;
   * `TestContentDataBase64` (new `llm/llm_test.go`) writes `Content.Data` `[]byte{0, 1, 2, 0xff}` as
     `"data":"AAEC/w=="` and reads it back.

   | Test | Break | The failure |
   | :--- | :--- | :--- |
   | `TestRawNullAndAbsent` | `Replacement`'s and `ExitCode`'s tags `omitzero` changed to `omitempty` | `{"type":"context_edit","parentId":null,"targetId":"a","replacement":null}: written {"type":"context_edit","parentId":null,"targetId":"a"}, <nil>`, and the same for `"exitCode":null` |
   | `TestEmptyRawValue` | `jsonx.From` without its `len(b) == 0` check | `through jsonx.From: wrote {…,"replacement", json: cannot marshal from Go jsontext.Value: unexpected EOF within "/replacement" after offset 68`, and the same for `/message/exitCode` |
   | `TestContentDataBase64` | `Content.Data` retyped `jsontext.Value` | `wrote {"type":"image","mediaType":"image/png","data", json: cannot marshal from Go jsontext.Value: invalid character '\x00' at start of value within "/data" after offset 47` |
4. **The docs** (step 4):
   * `session/doc.go` gains "Raw JSON and binary": a table of absent, `null`, a JSON value and binary, with each one's
     Go value and how it is written; that an empty non-nil value fails the marshal and that `jsonx.From`, `Value` and
     `Quote` never give one; and that tests compare `len(v)` and `string(v)`, never `v.String()`. `go doc ./session`
     renders the table as a block;
   * `docs/architecture.md`'s package table gains a `jsonx` row, after `gobble`: the two policies, the raw-value
     builders, `Strings` and `Fields`, `Member` and `SetMember`, and that it imports no package of this module (which
     `go list` confirms: `bytes`, `encoding/json/jsontext`, `encoding/json/v2`, `maps`, `slices`).
5. **G-local:**
   * `make pre-add-check` on the 6 changed Go files: clean;
   * `make preflight`: passed;
   * `go test ./...`: 38 ok, `llm` being new to the count; `make race`: 38 ok.
6. **G-hosts,** on the Linux machine and the Windows laptop, with 818 files shipped to each:
   * `go test` and race: passed on both, 38 ok;
   * `make check-records markdownlint`: passed on both;
   * preflight stops at `verify-build-metadata`, as on every copy.

### Deviation 17 (2026-11-01): P9, the two stream calls outside `jsonx`, and `jsonx`'s stream functions

* **Found** at P9's start. Step 2's `forbidigo` rule forbids v2's `MarshalEncode` and `UnmarshalDecode` outside `jsonx/`
  in non-test code, and two sites call them:
  * `internal/cli/mcp.go:182`, `json.MarshalEncode(enc, m.value)`, onto P2's strict encoder. No step names it, and the
    file is not P9's;
  * `internal/archtest/check.go:85`, `json.UnmarshalDecode(dec, &raw)`, which P8 step 1 wrote as the plan said.

  The plan's own steps conflict; the code did not cause it.
* **Measured,** on a scratch program under Go 1.27.3. Options passed to `MarshalEncode` and `UnmarshalDecode` take
  precedence over the stream's own for that call, in both directions: a lenient call on a strict stream repairs invalid
  UTF-8 and lets a repeated name through, and a strict call on a lenient stream refuses both. A call passing only
  `json.Deterministic(true)`, which is all `StrictOptions()` set, overrides neither.
* **The owner's answers:** first "option 1 vs option 2, pros and cons. i am leaning toward option 2 because it seems
  more idiomatic, extensible, and api standards adherent. explain."; then, given the comparison, "option2 with 4
  function forms". The option not taken was rewriting the two sites with `MarshalStrict` then `WriteValue`, and
  `ReadValue` then `Unmarshal`.
* **Decision.**
  * `jsonx` gains four stream functions, each passing its policy's options ahead of the caller's:
    `MarshalEncode` and `UnmarshalDecode` under the data policy, `MarshalEncodeStrict` and `UnmarshalDecodeStrict`
    under the configuration policy.
  * `Options()` and `StrictOptions()` state their UTF-8 and duplicate-name rules explicitly, so a call overrides a
    stream built with the other policy. They are v2's defaults where they are not D2's, so every existing call's
    behaviour is unchanged.
  * `internal/cli/mcp.go` calls `jsonx.MarshalEncodeStrict`, and `internal/archtest/check.go` calls
    `jsonx.UnmarshalDecode`.
  * The MADR's D1, D18 and D24 are amended.
* **Files added:** `jsonx/jsonx.go`, `jsonx/jsonx_test.go`, `internal/cli/mcp.go`, and `docs/architecture.md` for
  `jsonx`'s row. `internal/cli` is touched, so G-local and G-hosts take `make perf` and `make probe-conhost` (C6, C7).

### Deviation 18 (2026-11-01): P9, `TestLoginAsksThroughQuestions`'s teardown failure, again

* **Found** in P9's G-local. `make race` failed once: `--- FAIL: TestLoginAsksThroughQuestions` with
  `testing.go:1619: TempDir RemoveAll cleanup: unlinkat <TMP>/TestLoginAsksThroughQuestions<n>/002: directory not
  empty`. It is deviation 14's failure, there on the Windows laptop, here on macOS.
* **Reproduced, and pre-existing,** on the Mac under `-race` with `-run '^TestLoginAsksThroughQuestions$'`: 1 failure
  in `-count=30` on P9's tree, and 1 in two runs of `-count=30` on P8's commit `b0c884f`, each with the same error. A
  rerun of `go test -race ./internal/cli/` passed.
* **The owner's answer:** "Record, rerun, keep it deferred (Recommended)". The option not taken was writing the flake's
  own record before P10.
* **Decision.** As deviation 14: recorded, out of this plan's scope, and left for a record of its own; no assertion or
  test changed. `make race` was rerun: 38 ok. The flake will recur in later runs until that record fixes it.
* **Files added:** none.
* *Note (2026-11-07): the late writer is identified and removed by [0046-MADR-own-background-listings-and-keep-tests-hermetic.md](0046-MADR-own-background-listings-and-keep-tests-hermetic.md). A listing `settled` could not see wrote kept listings into the next test's home, or the user's cache, and the SDK's background model facts left an OAuth refresh lock. Its PLAN's P4 ran this test 60 times under `-race` on macOS and on the Windows laptop with no failure.*

### Deviation 19 (2026-11-01): P9, `internal/archtest/doc.go`

* **Found** writing P9's record. `internal/archtest/doc.go` said the test "checks the eight rules in 0004-MADR", and
  that "Rules 4 and 8 name their edges even though those imports are not in the tree yet". There were nine rules
  before P9 and are eleven after it. `go list` shows both edges in the tree now: `llm/provider` imports
  go-llmprovider-sdk, and `internal/buildinfo` go-selfupdate-lib's `buildinfo`. The file is in no phase's list, so the
  first P9 record left it.
* **The owner's answer:** "update this: Not done: internal/archtest/doc.go still says it checks "the eight rules";
  there are now eleven. It was already out of date before P9 and isn't in any phase's file list, so I left it and noted
  it in the record."
* **Decision.** `doc.go` names the command as `loadGraph` runs it, `go list -e -deps -test -json ./...`, and the eleven
  rules: 1 to 9 from 0004-MADR as amended, 10 and 11 from 0039-MADR D24, with `docs/architecture.md` as their list. The
  stale sentence about rules 4 and 8 goes. The P9 record's "Not done" item is replaced.
* **Files added:** `internal/archtest/doc.go`.

### P9 record (2026-11-01): the guards

The base is P8's `b0c884f`. Deviation 17 added `jsonx`'s stream functions and four files; deviation 18 recorded the
login test's teardown failure, outside P9; deviation 19 added `internal/archtest/doc.go`.

1. **`jsonx`'s stream functions** (deviation 17):
   * `MarshalEncode`, `UnmarshalDecode`, `MarshalEncodeStrict` and `UnmarshalDecodeStrict`, each passing its policy's
     options ahead of the caller's to v2's own;
   * `Options()` is `AllowInvalidUTF8(true)`, `AllowDuplicateNames(false)`, `Deterministic(true)`, and
     `StrictOptions()` is `AllowInvalidUTF8(false)`, `AllowDuplicateNames(false)`, `Deterministic(true)`. The `false`
     rules are v2's defaults, so every existing call behaves as before; `go test ./...` passed unchanged;
   * `internal/cli/mcp.go:182` calls `jsonx.MarshalEncodeStrict`, and `internal/archtest/check.go` calls
     `jsonx.UnmarshalDecode`. Neither file imports `encoding/json/v2` now;
   * `TestStreamPolicies` (`jsonx/jsonx_test.go`) runs each function on a stream built with the other policy: the data
     functions repair invalid UTF-8 on a strict stream and refuse a repeated name on a lenient one; the strict functions
     refuse invalid UTF-8 and a repeated name on a lenient stream; both decoders read two values, then `io.EOF`.
2. **`depguard`** (step 1): a `json` rule beside `logging`, for `$all` files, denying `encoding/json$` with "use jsonx,
   on encoding/json/v2 (0039-MADR D24)".
3. **`forbidigo`** (step 2), the pattern as first seen failing:
   `^json\.(Marshal|MarshalWrite|MarshalEncode|Unmarshal|UnmarshalRead|UnmarshalDecode)$` with
   `pkg: ^encoding/json/v2$`, under the existing `analyze-types: true`. The existing `_test.go` exclusion covers tests,
   and a new one excludes `^jsonx/` for this rule's text, `0039-MADR D24`, only.
4. **`internal/archtest`** (step 3):
   * rule 10: `jsonx`'s `Imports` hold no package of this module. Its tests' imports are not counted, since
     `jsonx_test` imports `jsonx`;
   * rule 11: a struct field of type `[]byte` or `[]uint8` with a `json` tag, other than `json:"-"`, is one of
     `byteFieldsAllowed`, today `llm.Content.Data`. It parses each module package's `GoFiles`, which `loadGraph` now
     reads from `go list` with `Dir`. A struct inside a function is named by the function;
   * `Violation` gains `Field`, and reads "rule 11: <package> has <field>";
   * `check_test.go`: five table cases (rule 10 on the module, `jsonx` importing `session`, `jsonx`'s tests importing
     the module, `session` importing `jsonx`; rule 11 on the module), and `TestRule11ScratchFile`, over a scratch file
     with a tagged field, an untagged one, a `yaml` tag, two names on one field, `json:"-"`, a struct in a function, and
     `Content.Data` in `session` and in `llm`.
5. **Seen failing** (step 4, Confirmation 12), each on a scratch copy of the finished tree:

   | Break | Command | The failure |
   | :--- | :--- | :--- |
   | `stdjson "encoding/json"`, used, in `tool/tool.go` | `golangci-lint run ./tool/` | `tool/tool.go:5:2: import 'encoding/json' is not allowed from list 'json': use jsonx, on encoding/json/v2 (0039-MADR D24) (depguard)` |
   | the same in `tool/tool_test.go` | `golangci-lint run ./tool/` | `tool/tool_test.go:5:2: import 'encoding/json' is not allowed from list 'json': …` |
   | `json.Marshal(1)` in `tool/tool.go` | `golangci-lint run ./tool/` | `` tool/tool.go:221:41: use of `json.Marshal` forbidden because "encode and decode through jsonx, under its policies (0039-MADR D24)" (forbidigo) `` |
   | `v2.Unmarshal`, the package imported as `v2`, in `tool/tool.go` | `golangci-lint run ./tool/` | `` tool/tool.go:221:50: use of `v2.Unmarshal` forbidden because … `` |
   | the `^jsonx/` exclusion removed | `golangci-lint run ./jsonx/` | `` jsonx/jsonx.go:29:9: use of `json.Marshal` forbidden … ``, and the same for each of `jsonx`'s v2 calls |
   | `jsonx` imports `command` | `go test ./internal/archtest/` | `TestImportRules/rule_10_module: unexpected edge: rule 10: …/jsonx imports …/command` |
   | `jsonx` imports `session`, as step 4 names it | `go test ./internal/archtest/` | `import cycle not allowed`, `[setup failed]`: `session` imports `jsonx`, so this break fails the build before the rule runs. The `command` break above is the rule's own |
   | `` Raw []byte `json:"raw,omitzero"` `` on `session.Entry` | `go test ./internal/archtest/` | `TestImportRules/rule_11_module: unexpected edge: rule 11: …/session has a []byte field with a json tag, Entry.Raw` |
   | `Options()` without `AllowDuplicateNames(false)` | `go test -run TestStreamPolicies ./jsonx/` | `UnmarshalDecode on a lenient stream accepted a repeated name: map[a:2]` |
   | `StrictOptions()` without `AllowInvalidUTF8(false)` | the same | `MarshalEncodeStrict on a lenient stream wrote "\"a�b\"\n"`, and `UnmarshalDecodeStrict on a lenient stream read "a�b"` |
   | `MarshalEncode` passing only the caller's options | the same | `MarshalEncode on a strict stream wrote "{\"a\":\"x\"", jsontext: invalid UTF-8 within "/s" after offset 13` |
   | `UnmarshalDecodeStrict` under `Options()` | the same | `UnmarshalDecodeStrict on a lenient stream read "a�b"` |

   The first `depguard` break, a blank `_ "encoding/json"` placed out of order, failed only on `gofmt` and `revive`'s
   blank-import rule, with no `depguard` line; that `golangci-lint` keeps one issue per line by default is a likely cause,
   not checked. It was replaced by the used, sorted import above, which `depguard` alone reports.
6. **`docs/architecture.md`**: `jsonx`'s row names the stream functions, and the import rules gain 10 and 11 and a
   paragraph on the two lint rules.
7. **`internal/archtest/doc.go`** (deviation 19) names the eleven rules and the `go list` command `loadGraph` runs.
8. **G-local:**
   * `make pre-add-check` on the 5 changed Go files: clean; after deviation 19, on `internal/archtest/doc.go`: clean;
   * `make preflight`: passed, `golangci-lint` with the two new rules among it; after deviation 19: passed;
   * `go test ./...`: 38 ok; `make race`: deviation 18, then 38 ok;
   * `make perf`: passed at load 25, with ReplyRender 42.71 µs (baseline 30 µs), Reprint 12.53 ms (baseline 9.4
     ms), LineSessionStart 1.432 ms (baseline 830 µs), KeyDraw 1.671 µs (baseline 1.5 µs); LineSessionStart made
     5,011 allocations.
9. **G-hosts,** on the Linux machine and the Windows laptop, with 818 files shipped to each:
   * `go test` and race: passed on both, 38 ok;
   * `make check-records markdownlint`: passed on both;
   * preflight stops at `verify-build-metadata`, as on every copy;
   * `make perf`: passed on both. The Linux machine: ReplyRender 128.72 µs (baseline 110 µs), Reprint 31.18 ms
     (baseline 29 ms), LineSessionStart 4.52 ms (baseline 2.9 ms), KeyDraw 4.769 µs (baseline 5 µs). The Windows
     laptop: ReplyRender 29.07 µs (baseline 54 µs), Reprint 7.33 ms (baseline 12 ms), LineSessionStart 4.33 ms
     (baseline 6.9 ms), KeyDraw 1.208 µs (baseline 1.4 µs);
   * the conhost probe on the Windows laptop: passed.

### P10 record (2026-11-01): close-out

The base is P9's `b32942a`. No deviation.

1. **The count** (step 1), measured on `b32942a` by a scratch script over `go list -json ./...`, and by `grep`:
   * 50 packages in the module; 0 import `encoding/json`, counting `Imports`, `TestImports` and `XTestImports`.
     `grep -rlE '"encoding/json"$' --include=*.go .` exits 1;
   * outside `jsonx`, three non-test files import `encoding/json/v2`, and only for options, with no encode or decode
     call: `internal/cli/config.go` (`json.Options`, `json.Deterministic`), `internal/config/schema.go` and
     `mcpclient/tool.go` (`json.Deterministic`);
   * the only non-test calls to v2's `Marshal`, `MarshalWrite`, `MarshalEncode`, `Unmarshal`, `UnmarshalRead` and
     `UnmarshalDecode` are `jsonx/jsonx.go`'s eight, which P9's `forbidigo` rule allows.
2. **Issue comments** (step 2), for the owner to post. Posting is outward-facing, so it waits for the owner's word, and
   the commits they name resolve on GitHub only once pushed. Neither issue is closed here.

   gobble-cli#7:

   ```text
   gobble-cli#7 is carried out by 0039-MADR and its PLAN
   (docs/decisions/0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md, and the PLAN beside it).

   - U1, the single-marshaler sweep: done. No package of the module imports encoding/json,
     tests included (go list over its 50 packages). Commits: 4f76649 (RawMessage spelled
     jsontext.Value), 5a20d06 (everything gobble defines on v2: _gobble/* notifications,
     payloads, error data, _meta.gobble), cfa514f (session/update params, stream-json bytes
     unchanged), ec6298e (recent-models.json, the kept listing, providers --json), fe671d6 (every
     remaining site and test), b0c884f (archtest). The fixture acceptance: recent-models.json
     and a kept listing as v1 wrote them, null included, load (TestRecentModelsV1File,
     TestKeptListingV1File, ec6298e).
   - U2, strict decoding at the ACP boundary: declined (0039-MADR D19). The SDK decodes ACP's
     methods and its types keep unknown members; gobble's own _gobble/* params stay lenient, so
     a newer client's field does not make an older agent answer -32602.
   - U3, store hardening: moved to 0037 (0039-MADR D22).
   - U4, SDK lockstep: met at acp-go-sdk2 v1.28.0, as 0039-MADR records; v1.29.0 is #8's.
   - U5, the guard: done in b32942a. depguard denies encoding/json in every file, tests
     included; forbidigo keeps v2's Marshal/Unmarshal family inside jsonx in non-test code.
     Each was seen failing on a planted break. The open question: the allowlist is empty.

   Also from this work: non-UTF-8 text no longer panics the agent, encode errors are no
   longer discarded, and a cut-off tool call no longer fails the prompt (6b5af2c); the JSON
   configuration files are decoded strictly, a repeated name refused, and
   `gobble config validate` checks them (f38c1c5).
   ```

   gobble-cli#9:

   ```text
   gobble-cli#9 is carried out by 0039-MADR and its PLAN
   (docs/decisions/0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md, and the PLAN beside it).

   - U0, the records: 153ef60.
   - U1, jsonx: 6b5af2c, at the module root. From, Value (this issue's Marshal, returning
     jsontext.Value), Quote, Splice, IsAbsent, IsNull, and the two policies (Marshal and
     Unmarshal for data, MarshalStrict and UnmarshalStrict for configuration). 5a20d06 added
     Strings, Member and SetMember; b32942a the stream functions MarshalEncode, UnmarshalDecode
     and their Strict forms.
   - U2, ad hoc constructors replaced: 6b5af2c (prompt's mustJSON gone; the agent's tool calls
     through jsonx.From).
   - U3, absent against null: b0c884f (TestRawNullAndAbsent on Entry.Replacement and
     Message.ExitCode; TestEmptyRawValue).
   - U4, the ACP seam: 6b5af2c (jsonRaw and rawInput replaced by jsonx.Splice) and 4f76649
     (every RawMessage spelled jsontext.Value).
   - U5, the docs: b0c884f (session/doc.go's table; docs/architecture.md's jsonx row).
   - U6, the []byte guard: b32942a, internal/archtest rule 11; today only llm.Content.Data.
   - F1: From, Value and Splice never give an empty non-nil value; pinned by TestFrom and
     TestEmptyRawValue. F2: PreserveRawStrings is used nowhere; stream-json's update lines use
     EscapeForHTML and EscapeForJS, and their bytes are unchanged (cfa514f). F3: session/doc.go
     says to compare len and string(v), never String(). F4: 4f76649.
   - Acceptance 1: jsonx's tests; the empty-value EOF was seen failing first (P1's record).
     2: b0c884f. 3: every package's tests pass. 4: TestContentDataBase64 (b0c884f).
     5: depguard (b32942a). 6: archtest rule 10 (b32942a); a planted jsonx -> session import
     fails as an import cycle, since session imports jsonx, and a planted jsonx -> command
     import fails rule 10 itself. 7: b0c884f.
   - Open questions: From does not validate, and Splice does where an invalid value would break
     the encoder; PreserveRawStrings nowhere; the package is jsonx at the module root.
   ```
3. **0038-REPORT** (step 3) gains a dated note: its recommended order was carried out by this plan, with the phases'
   commits. The report's text above the note is unchanged.
4. **The disclosure guard** (step 4), run as `AGENTS.md` gives it over the outgoing commits, `origin/main` `31e3c1c` to
   `b32942a` (11 commits, `9b3bffe` to `b32942a`): exit 0, no output. This phase's own commit is outgoing too; the
   pre-push hook runs the same guard over it. Nothing was pushed.
5. **The Confirmation** (step 5). Each item, the phase that met it, and where its record is:

   | Item | Phase | Commit | Evidence |
   | :--- | :--- | :--- | :--- |
   | 1 to 7, 13, 16 | P1 | `6b5af2c` | the P1 record's tests, each seen failing at the base |
   | 8, 22 | P2 | `f38c1c5` | the P2 record |
   | 9 | P5 | `cfa514f` | `TestPrintStreamJSONEscapes`, its golden equal to the base's v1 output |
   | 10 | P6 | `ec6298e` | `TestRecentModelsV1File`, `TestKeptListingV1File` |
   | 11 | P8 | `b0c884f` | `TestRawNullAndAbsent`, `TestEmptyRawValue`, `TestContentDataBase64` |
   | 12 | P9 | `b32942a` | the twelve planted breaks |
   | 14 | each, P7 in full | `fe671d6` | no golden changed in any phase; `make perf` in each phase that touched `internal/cli` (P7's by deviation 16) |
   | 15 | P1, P4 | `6b5af2c`, `5a20d06` | `jsonx`'s tests, `FuzzQuote`, `FuzzStrings` |
   | 17 | P1, P8 | `6b5af2c`, `b0c884f` | the replay, serialization, compaction, `_gobble/*` and personality tests, unchanged |
   | 18 to 21 | P4 | `5a20d06` | `TestNotificationDecodedWithV2`, `TestNotifyFramesUnchanged`, `FuzzStrings`, `TestMetaMemberMistyped` |

   Confirmation 4's second test was split in P1 into `TestWriterKeepsBuildError` and `TestSetBlocksBadJSON`, as the
   P1 record says. Every test the Verification table names is in the tree.

   The records commits are P0's `9b3bffe` (0035's records, by PQ1 (a)) and `153ef60` (this pair and 0038).
6. **Left for other records,** named so they are not read as oversights:
   * `TestLoginAsksThroughQuestions`'s teardown failure (deviations 14 and 18), pre-existing, about 1 in 30 to 60 runs
     under `-race`, waits for a record of its own;
   * #7's U3, store hardening, and the wire's 10 MiB bound are 0037's (MADR D22);
   * acp-go-sdk2 `v1.29.0` and `OverflowBlock` are gobble-cli#8's;
   * the agent's twenty `_gobble/*` methods through `acp.Handle`, and `bytes.Lines` in the store (MADR D22).
7. **Status:** every Confirmation item is met, so this plan is `complete`. The MADR stays `accepted`; no fact it states
   changed in P10.
