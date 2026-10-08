---
status: proposed
date: 2026-10-08
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# PLAN 0012 — Tool schemas and descriptions: the contract, the five built-in tools, and the strict probe

Implements [0012-MADR-tool-schemas-descriptions-and-loading.md](0012-MADR-tool-schemas-descriptions-and-loading.md) decisions D1, D2, D3's probe and D5, closing findings F2, F3, F5 and F6. F4 and F17 were measured by the MADR's Probe D before this plan's approval. It places D4 and D6–D14 in the 0005-PLAN sub-phases that build their tools, so findings F7–F16 close there, by the notes P1 writes.

*(2026-10-08, before approval: D3 was amended by the owner's choice. Its opt-out, `strict: false` for tools with optional arguments, waits on go-llmprovider-sdk's per-tool strict flag. It is named under Deferred, with the phase that lands it.)*

## Goal

The plan is done when each of these holds:

1. **The records.**
   - 0012-MADR's status is `accepted`.
   - 0005-MADR has an amendment of 2026-10-08 pointing to it. Its native-tools Exposure paragraph and its direct-list consequence are struck through, with 0012-MADR D7's text written after each.
   - 0005-PLAN has a 0012 note at F1 steps 5 and 8, F3, F6, F8 and X4, an amended placement table, and an Amendments entry.
   - `make check-records` prints no error.
2. **The tool contract.**
   - `tool.New` publishes a schema with no `null` type and with `tool.WithSchema`'s additions.
   - Its `Run` validates against the schema derived from the Go type alone, exactly as today.
   - `go test ./tool -run 'TestPublishedSchema|TestNewDerivesSchema|TestRun' -count=1` passes, and was seen failing on a scratch copy without the change.
3. **The five built-in tools' schemas.** `go test ./tool/builtin -run 'TestSchemaConvention|TestBoundsHold' -count=1` passes, and was seen failing on a scratch copy. Probe A, re-run, shows:
   - `edits` as `"type":"array"` with `"minItems":1`;
   - read's `offset` with minimum 1 and default 1, and `limit` 1–2000 with default 2000;
   - bash's `command` with `"minLength":1`, and `timeout` 1–600 with default 120.
4. **The descriptions.**
   - The five descriptions are rendered from `tool/builtin/describe/*.md`, and the five Go description strings are gone.
   - `go test ./tool/builtin -run TestDescriptionShape -count=1` passes, and was seen failing on a scratch copy.
5. **The strict probe.**
   - `go vet -tags live_openai ./llm/provider` exits 0.
   - Without `OPENAI_API_KEY`, `go test -tags live_openai -run TestStrictProbe -v ./llm/provider` reports the test as skipped.
   - `go test ./llm/provider -run TestStrictOf -count=1` passes, and was seen failing on a scratch copy.
6. **Every gate.**
   - `make preflight` prints `preflight passed` on Windows and in WSL.
   - WSL `go test -race ./...` exits 0.
   - `golangci-lint run ./...` prints `0 issues.` for GOOS linux, darwin and windows.

## Scope

### In scope (the only files any phase may touch)

| Phase | Files |
| :--- | :--- |
| P1 | `docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md`, `docs/decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md`, `docs/decisions/0005-MADR-v1-feature-scope.md`, `docs/decisions/0005-PLAN-v1-feature-scope.md`, `docs/README.md` |
| P2 | `tool/tool.go`, `tool/tool_test.go`, `tool/doc.go`, `docs/architecture.md` |
| P3 | `tool/builtin/schema.go` (new), `tool/builtin/schema_test.go` (new), `tool/builtin/read.go`, `tool/builtin/edit.go`, `tool/builtin/bash.go`, `tool/builtin/powershell.go`, `tool/builtin/shell.go` |
| P4 | `tool/builtin/describe.go` (new), `tool/builtin/describe_test.go` (new), `tool/builtin/describe/read.md`, `write.md`, `edit.md`, `bash.md`, `powershell.md` (new), `tool/builtin/read.go`, `tool/builtin/write.go`, `tool/builtin/edit.go`, `tool/builtin/bash.go`, `tool/builtin/powershell.go`, `docs/architecture.md` |
| P5 | `llm/provider/strict_test.go` (new), `llm/provider/strict_live_test.go` (new) |
| P6 | this PLAN (execution record), `docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md` (Observed section), `docs/README.md` |

### Out of scope

- **D4 and D14's MCP serving** belong to 0005-PLAN X4. They need `gobble mcp serve`, which is 1.x.
- **D6's system-prompt section** belongs to F3, which builds the sectioned prompt. Until it exists, the cross-tool guidance stays in bash's description (P4).
- **D7–D9's tiers, families and loading, D11's BM25, and the three MCP resource tools** belong to F1d, which builds `todo`, `tool_search` and those tools. D7's MCP default belongs to F8.
- **D10's shell interception** belongs to F6, which brings `mvdan.cc/sh`.
- **D12** needs no work, and **D13** is a request to another repository, not made here.
- **The new tools' schemas** are built with the tools: move, delete, copy, mkdir and `tree` in F1c; `request_permissions` in F6; the rest in X2, X7 and X8. Each applies D1, D2 and D5 through the pieces P2–P4 build.

## Stability rule

Each phase ends with these, run from the repository root, with output redirected to a log in the session scratchpad and the exit status captured before anything reads the log:

```text
make pre-add-check FILES="<the phase's Go files>" > "$LOG" 2>&1; echo $?     -> 0
go test -count=1 ./tool/... ./llm/... > "$LOG" 2>&1; echo $?                -> 0
make preflight > "$LOG" 2>&1; echo $?                                       -> 0, last line "preflight passed"
```

P2–P5 also end with:

```text
GOOS=linux   CGO_ENABLED=0 golangci-lint run --uniq-by-line=false ./...      -> "0 issues."
GOOS=darwin  CGO_ENABLED=0 golangci-lint run --uniq-by-line=false ./...      -> "0 issues."
GOOS=windows CGO_ENABLED=0 golangci-lint run --uniq-by-line=false ./...      -> "0 issues."
go test -count=1 ./...                      (Windows)                        -> no "--- FAIL" or "panic:"
WSL, on a copy of the tree: make preflight; CGO_ENABLED=1 go test -race -count=1 ./...  -> "preflight passed", exit 0
```

- **The scans.** The identifier scan over the added lines, and the hidden-character scan over the staged files, report 0.
- **Commits.**
  - The agent stages each phase's files with `git add` on the current branch. The owner commits with `git commit --no-edit`, one commit per phase.
  - `git push` and tags are not permitted unless the owner asks in the same turn.
- **The records commit.** The 0012 MADR, this PLAN and their `docs/README.md` rows are committed alone, before P1, under the bootstrap exception.

## Cross-cutting contracts

- **C1. Validation behaves exactly as today.** `tool.New`'s `Run` validates against the schema `jsonschema.For[In]` derives, unchanged (0012-MADR D2 rule 8). The existing tests of `tool` and `tool/builtin` pass with no edit to their assertions.
  - **The exception:** P3's one new refusal, bash's and powershell's empty command, comes with a new test of its own.
  - **This is the contract most at risk.** A test that pins old behaviour is the quickest thing to edit when a phase goes red. It is never edited to pass. A failure is a deviation, and is stopped and put to the owner.
- **C2. No module changes.** `go.mod` and `go.sum` are unchanged by every phase. P4 uses `embed` and `text/template`, P5 uses `net/http`; all are standard library.
- **C3. Every new check is seen to fail first.**
  - It is run against a scratch copy of the tree with the change undone or broken.
  - The mutation is asserted to have landed (the anchor occurs exactly once before it is replaced).
  - The failure is read whole from its log, and quoted in the execution record.
  - The tree itself is never dirtied to make a check fail.
- **C4. No hand-written schema JSON.** Every published constraint is set in Go, on the derived `*jsonschema.Schema`, through `tool.WithSchema` (0012-MADR D1).
- **C5. The descriptions say what they say today.** P4 reshapes the existing text into D5's sections. It adds no claim the code does not make good, and drops none, except that numbers which would go stale are filled from the options. Each moved sentence is listed in P4.
- **C6. No identifiers.** Nothing committed carries a host name, account name or real-machine path (AGENTS.md, Identifiers).

## Dependency and delivery order

```text
F1b commit (the owner; staged 2026-10-08)
  └─ records commit: 0012-MADR, this PLAN, their README rows (the owner)
       └─ P1 records ─ P2 tool contract ─ P3 the five schemas ─ P4 descriptions ─ P5 strict probe ─ P6 record
```

- **P3 needs P2's `tool.WithSchema`.**
- **P4 touches the same files as P3,** so it follows it.
- **P5 needs P3's schemas,** because it probes what is published.
- **P6 records the phases.** It also records the probe's result when the owner has run P5's live test with a key in the environment. This plan cannot supply the key: no secret is passed on a command line or written to a file in the tree.

## Implementation Steps

Go code in this plan is indented with spaces, for the Markdown linter. `gofmt`, run by `make pre-add-check`, writes it with tabs.

### P1 — Records (accepts D1–D14; places D4, D6–D11 and D14; closes nothing by code)

**Step 1. `docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md`.**

- Change frontmatter `status: proposed` to `status: accepted`.
- Set `date:` to the date the owner approves this PLAN.

**Step 2. `docs/decisions/0005-MADR-v1-feature-scope.md`.**

- **(a) The Exposure paragraph.** In the native-tools amendment, wrap the paragraph that begins `**Exposure.** These tools are direct:` in `~~…~~`. Directly after it, add this paragraph:

  ```markdown
  *(2026-10-08, superseded by [0012-MADR](0012-MADR-tool-schemas-descriptions-and-loading.md) D7.)* **Exposure.** Direct, on every request: read, write, edit (or `apply_patch` in place of write and edit for GPT-family models), grep, find, `tree`, move, delete, copy, mkdir, bash, powershell on Windows, `todo`, and `tool_search` whenever anything is deferred. The job tools join them when X2 builds them. Deferred, found through `tool_search`: `request_permissions` (also loaded when a call is refused for a permission), `lsp`, `monitor`, `notebook`, the three MCP resource tools, and every MCP server's tools unless `mcp.json` sets `exposure` to `direct`.
  ```

- **(b) The direct-list consequence.** Wrap the bullet that begins `* The direct list is the 17 names under Exposure.` in `~~…~~`. After it, add this bullet:

  ```markdown
  * *(2026-10-08, 0012-MADR D7.)* The direct list is 14 names on Windows for most models and 13 for GPT-family models; elsewhere, without powershell, 13 and 12. The job tools add three when X2 builds them.
  ```

- **(c) Notes in three table rows.** Append each note inside the row's last cell, just before the cell's closing pipe:
  - the `tool_search` row: *(2026-10-08, 0012-MADR D9, D11: ranked by an in-house BM25 in `tool/toolsearch`; found tools load with their families, and loads are append-only.)*;
  - the `mcp.json` row: *(2026-10-08, 0012-MADR D7: a server with no `exposure` key is deferred.)*;
  - the `gobble mcp serve` row: *(2026-10-08, 0012-MADR D4: it also serves the native file tools, with the same schemas and rules.)*.
- **(d) The amendment.** At the end of the file, add:

  ```markdown
  ### Amendment (2026-10-08): tool schemas, descriptions and loading

  **Why.** The owner asked on 2026-10-08 for the native tools' schemas, proposed JSON-RPC 2.0, Markdown descriptions, deferring every tool but `tool_search`, and bleve for BM25. [0011-REPORT](../reports/0011-REPORT-native-tools-survey.md) evaluated the proposal, and [0012-MADR](0012-MADR-tool-schemas-descriptions-and-loading.md) decides it.

  **What changes here.**
  * The Exposure paragraph of the native-tools amendment, and its direct-list consequence, are replaced by 0012-MADR D7. `request_permissions` is deferred; `todo` and `tool_search` are named direct.
  * An MCP server with no `exposure` key is deferred (0012-MADR D7), extending the bridged-Pi rule of this record's Exposure modes notes.
  * `gobble mcp serve` also serves the native file tools (0012-MADR D4).
  * `tool_search` ranks with an in-house BM25 in `tool/toolsearch`, and bleve is not adopted for it (0012-MADR D11, D12).
  * Tool schemas and descriptions follow 0012-MADR D1–D3 and D5.
  ```

**Step 3. `docs/decisions/0005-PLAN-v1-feature-scope.md`.**

- **(a) Six notes,** each added on a line of its own directly after the anchor named:
  - **F1 step 5,** after the line `*(2026-10-08, 0005-MADR amendment "native tools": F1c also builds …)*`:

    ```markdown
       *(2026-10-08, 0012-MADR: F1c's tools follow D1, D2 and D5, through `tool.WithSchema` and `tool/builtin/describe/`, with the schemas 0012-MADR's table gives them.)*
    ```

  - **F1 step 8,** after the step's last line, which ends `and the MCP resource tools.`:

    ```markdown
       *(2026-10-08, 0012-MADR: F1d builds D7's tiers, D8's families and D9's loading, D11's BM25 in `tool/toolsearch`, and D14's client data; the three resource tools take 0012-MADR's schemas.)*
    ```

  - **F3,** after its heading's existing note that begins `*(2026-10-07, 0005-MADR amendment "the best of several harnesses": choice 8`:

    ```markdown
    *(2026-10-08, 0012-MADR D6 and D9: the `tools` section carries the cross-tool guidance, moved out of bash's description, and the categories of deferred tools; the tools list is part of the frozen baseline and grows only by appending.)*
    ```

  - **F6,** after its heading's note of 2026-10-08, the one saying `request_permissions` lands there:

    ```markdown
    *(2026-10-08, 0012-MADR D10: a bash command that is one plain `rm`, `rmdir`, `mv`, `cp` or `mkdir` with flags that map exactly runs as the native tool; anything else is permission-checked as a shell command. D7: `request_permissions` is deferred, and loads when a call is refused for a permission.)*
    ```

  - **F8,** directly after the heading `### F8 — MCP completion (1.0; extends 0002-PLAN Phase 5)` and a blank line:

    ```markdown
    *(2026-10-08, 0012-MADR D7: an MCP server with no `exposure` key is deferred.)*
    ```

  - **X4,** directly after the heading `### X4 — clients and servers 1.x` and a blank line:

    ```markdown
    *(2026-10-08, 0012-MADR D4 and D14: `gobble mcp serve` also serves the native file tools with their schemas, and each gains an `outputSchema`.)*
    ```

- **(b) The placement table.** In the "native tools placed" table, change the `request_permissions` row's Exposure cell from `direct` to `~~direct~~ deferred (0012-MADR D7)`.
- **(c) The Amendments entry.** At the end of `## Amendments`, add:

  ```markdown
  **2026-10-08 — 0012-MADR: tool schemas, descriptions and loading.** [0012-PLAN](0012-PLAN-tool-schemas-descriptions-and-loading.md) builds D1, D2, D3 and D5 for the five built-in tools. The rest land where their tools are built, by the notes of this date at F1 steps 5 and 8, F3, F6, F8 and X4. `request_permissions` is deferred.
  ```

**Step 4. `docs/README.md`.**

- 0012 MADR row status: `accepted (2026-10-08; 38 verified claims)`.
- 0012 PLAN row status: `in progress (approved <date>)`.
- 0005-MADR row status: append `; 0012 amendment 2026-10-08` before the closing `)`.
- 0005-PLAN row status: append `; 0012 notes 2026-10-08` before the closing `)`.

**Verification.**

```text
make check-records > "$LOG" 2>&1; echo $?                                    -> 0
npx --no-install markdownlint-cli2 docs/README.md                            -> "0 issues"
grep -c "0012-MADR" docs/decisions/0005-PLAN-v1-feature-scope.md             -> 8 (six notes, the table row, the entry)
grep -c "0012-MADR" docs/decisions/0005-MADR-v1-feature-scope.md             -> at least 7 (two replacements, three row notes, the amendment's links and bullets)
grep -n "^status: accepted" docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md -> one line
```

A Python script in the scratchpad makes each edit. It asserts that each anchor occurs exactly once before it changes it, so a missed anchor fails loudly (C3).

### P2 — The tool contract publishes one schema and validates another (D1; D2 rules 5 and 8; closes F2)

**Step 1. `tool/tool.go`.**

- **(a) The hook.** Add a `schema func(*jsonschema.Schema)` field to `config`, and this option after `WithAnnotations`:

  ```go
  // WithSchema adds what struct tags cannot say to the schema a model sees:
  // bounds, defaults, enums and minItems (0012-MADR D1, D2). Run does not
  // validate against them: the tool's own code enforces what WithSchema
  // publishes, in its own words (0012-MADR D2 rule 8).
  func WithSchema(adjust func(s *jsonschema.Schema)) Option {
      return func(cfg *config) { cfg.schema = adjust }
  }
  ```

- **(b) Two schemas in `New`.** After the existing `resolved` is made, derive a second schema for publishing, and marshal that one instead of `schema`:

  ```go
      published, err := jsonschema.For[In](nil)
      if err != nil {
          panic(fmt.Sprintf("tool: %s: input schema: %v", name, err))
      }
      nonNull(published)
      if cfg.schema != nil {
          cfg.schema(published)
      }
      if _, err := published.Resolve(nil); err != nil {
          panic(fmt.Sprintf("tool: %s: resolve published schema: %v", name, err))
      }
      raw, err := json.Marshal(published)
  ```

- **(c) The non-null rewrite.** Add after `New`:

  ```go
  // nonNull drops the "null" that jsonschema-go adds to the type of a slice
  // or a pointer, here and in every nested property and item: a model is
  // never told an argument may be null (0012-MADR D2 rule 5).
  func nonNull(s *jsonschema.Schema) {
      if s == nil {
          return
      }
      if len(s.Types) == 2 && s.Types[0] == "null" {
          s.Type, s.Types = s.Types[1], nil
      }
      for _, p := range s.Properties {
          nonNull(p)
      }
      nonNull(s.Items)
  }
  ```

- **(d) `New`'s doc comment.** Append this sentence: `The schema a model sees (Spec.InputSchema) has no null types and carries WithSchema's additions; Run validates against the schema derived from In alone, so those additions are the tool's own code to enforce.`

**Step 2. `tool/tool_test.go`.** Add an import of `github.com/google/jsonschema-go/jsonschema`, and this test:

```go
type listIn struct {
    Items []string `json:"items" jsonschema:"the items"`
    Limit int      `json:"limit,omitzero" jsonschema:"how many"`
}

// The published schema has no null types and carries WithSchema's bounds,
// while Run validates against the Go type's schema alone, so the tool's
// code sees the arguments it sees today (0012-MADR D2 rules 5 and 8).
func TestPublishedSchema(t *testing.T) {
    var seen listIn
    tl := tool.New("list", "list", func(_ context.Context, in listIn, _ tool.Env) (string, error) {
        seen = in
        return "ok", nil
    }, tool.WithSchema(func(s *jsonschema.Schema) {
        s.Properties["items"].MinItems = new(1)
        s.Properties["limit"].Maximum = new(float64(10))
    }))
    s := string(tl.Spec().InputSchema)
    for _, want := range []string{`"items":{"type":"array"`, `"minItems":1`, `"maximum":10`} {
        if !strings.Contains(s, want) {
            t.Errorf("schema %s lacks %s", s, want)
        }
    }
    if strings.Contains(s, `"null"`) {
        t.Errorf("schema %s offers null", s)
    }
    r, err := tl.Run(t.Context(), tool.Call{Name: "list", Args: jsontext.Value(`{"items":[],"limit":50}`)}, tool.Env{})
    if err != nil || r.IsError || seen.Limit != 50 {
        t.Fatalf("run = %+v, %v, seen %+v; want the call to reach the tool, whose code enforces the published bounds", r, err, seen)
    }
}
```

**Step 3. `tool/doc.go`.** After the sentence ending `before calling it.`, add: `The schema it publishes may say more than the one it validates: WithSchema adds bounds, defaults and minItems for the model, and the tool's code enforces them.`

**Step 4. `docs/architecture.md`.** Change the `tool` row's role to: `The tool contract, with the workspace roots each call gets. A tool publishes a schema with its bounds and defaults, and validates the Go type's own.`

**Verification.**

```text
go test ./tool -count=1 -run 'TestPublishedSchema|TestNewDerivesSchema|TestRun' -v   -> ok, all three PASS
go test ./... -count=1  (Windows)                                                    -> no "--- FAIL"; C1 holds
```

**Seen failing (C3), each on a scratch copy of the tree:**

- **copy 1:** delete the line `nonNull(published)`. TestPublishedSchema fails with `offers null`.
- **copy 2:** apply `cfg.schema(schema)` before `schema.Resolve`, so the validator sees the bounds. TestPublishedSchema fails with `want the call to reach the tool`.

### P3 — The five built-in schemas follow the convention (D2; closes F3)

**Step 1. `tool/builtin/schema.go` (new).**

```go
package builtin

import (
    "strconv"

    "github.com/google/jsonschema-go/jsonschema"
)

// property is s's property name. A missing one is a programming error that
// the first test building the tool finds.
func property(s *jsonschema.Schema, name string) *jsonschema.Schema {
    p, ok := s.Properties[name]
    if !ok {
        panic("builtin: the schema has no property " + name)
    }
    return p
}

// bound publishes an integer property's minimum, its maximum when maximum
// is above 0, and its default (0012-MADR D2 rule 6). The tool's code
// enforces them (rule 8).
func bound(s *jsonschema.Schema, name string, minimum, maximum, def int) {
    p := property(s, name)
    p.Minimum = new(float64(minimum))
    if maximum > 0 {
        p.Maximum = new(float64(maximum))
    }
    p.Default = []byte(strconv.Itoa(def))
}

// nonEmpty publishes a string property's minLength of 1.
func nonEmpty(s *jsonschema.Schema, name string) { property(s, name).MinLength = new(1) }
```

`read.go`, `edit.go` and `shell.go` gain the import `github.com/google/jsonschema-go/jsonschema`. It is already a requirement of the module (0012-MADR V9), and no import-boundary rule names it (0004-MADR, rules 1–8).

**Step 2. read** (`tool/builtin/read.go`).

- **Its hook.** Add to `readWith`'s options:

  ```go
  tool.WithSchema(func(s *jsonschema.Schema) { bound(s, "offset", 1, 0, 1); bound(s, "limit", 1, maxLines, maxLines) }),
  ```

- **Its tag.** Change `Limit`'s tag from `how many lines (or entries) to show, at most 2000` to `how many lines (or entries) to show`; the schema now carries the bound.
- **The bounds hold** by today's code: `max(in.Offset, 1)`, and `min(in.Limit, maxLines)`, with 0 meaning `maxLines`.

**Step 3. edit** (`tool/builtin/edit.go`).

- **Its hook:**

  ```go
  tool.WithSchema(func(s *jsonschema.Schema) {
      edits := property(s, "edits")
      edits.MinItems = new(1)
      nonEmpty(edits.Items, "oldText")
  }),
  ```

- **The bounds hold** by today's refusals: `edit: edits needs at least one replacement` (0012-MADR V37), and `edit <path>: oldText is empty` (V38).

**Step 4. bash and powershell** (`tool/builtin/shell.go`, `bash.go`, `powershell.go`).

- **(a) The shared hook.** In `shell.go`:

  ```go
  // shellSchema publishes a non-empty command and the timeout's bounds and
  // default, the maximum from o (0012-MADR D2 rules 6 and 8).
  func shellSchema(o Options) tool.Option {
      return tool.WithSchema(func(s *jsonschema.Schema) {
          nonEmpty(s, "command")
          bound(s, "timeout", 1, int(o.MaxTimeout/time.Second), int(defaultTimeout/time.Second))
      })
  }
  ```

- **(b) Its use.** Add `shellSchema(o),` to the options of `bashWith` and `powershellWith`.
- **(c) The timeout's tag.** Change `Timeout`'s tag from `seconds before the command is stopped: 120 when not given, at most 600` to `seconds before the command is stopped`.
- **(d) The one new refusal.** At the top of `runShell`, before `sh.find()`:

  ```go
      if strings.TrimSpace(in.Command) == "" {
          return tool.Result{}, fmt.Errorf("%s: the command is empty; send the command to run", sh.name)
      }
  ```

  This is the one behaviour this plan adds (C1). Today an empty command runs the shell with nothing to do.

**Step 5. `tool/builtin/schema_test.go` (new).** It is package `builtin`, with two tests.

- **`TestSchemaConvention`** decodes `InputSchema` as `map[string]any` for each tool of `ToolsWith(Options{})` and of `ToolsWith(Options{MaxTimeout: 20 * time.Minute})`, and walks it. It fails, naming the tool and the JSON path, when any of these holds:
  1. the top-level `type` is not `"object"`;
  2. an object lacks `"additionalProperties": false`, or names in `required` a property it does not have;
  3. a property name does not match `^[a-z][a-zA-Z0-9]*$`, or has no non-empty `description`;
  4. any `type` is not a string: an array type is how `null` appears;
  5. an array has no `minItems`;
  6. an integer has no `minimum`, or is optional with no `default`;
  7. any key is `$ref`, `oneOf`, `anyOf`, `allOf`, `not`, `if`, `then`, `else`, `patternProperties` or `format`.

  It also asserts that bash's `timeout.maximum` is 1200 under the 20-minute options, and 600 under the defaults.
- **`TestBoundsHold`**, one subtest each:
  - read of a 3000-line file with `limit` 5000 shows `1-2000 of 3000` in its footer;
  - read with `limit` 0 shows the same;
  - read with `offset` 0 shows line 1 first, as `1:` and its text;
  - edit with `edits: []` is an error result whose text is `edit: edits needs at least one replacement`;
  - edit with `oldText: ""` is an error result containing `oldText is empty`;
  - bash with `command: "  "` is an error result `bash: the command is empty; send the command to run`;
  - `commandTimeout(100000, 10*time.Minute)` is `10*time.Minute`;
  - `commandTimeout(0, 10*time.Minute)` and `commandTimeout(-5, 10*time.Minute)` are `defaultTimeout`.

**Verification.**

```text
go test ./tool/builtin -count=1 -run 'TestSchemaConvention|TestBoundsHold' -v   -> ok
go run . (Probe A, scratch module specsize, with the schema printed)           -> edits "type":"array","minItems":1;
     read offset minimum 1 default 1, limit minimum 1 maximum 2000 default 2000;
     bash and powershell command "minLength":1, timeout minimum 1 maximum 600 default 120
```

**Seen failing (C3).**

- **Before step 2,** with only steps 1 and 5 in a scratch copy, TestSchemaConvention fails. It names: `edits` with no `minItems`; read's `offset` and `limit` with no `minimum` or `default`; bash's `timeout` with no `minimum` or `default`.
- **A scratch copy without step 4(d):** the empty-command subtest fails.

### P4 — Descriptions from Markdown templates (D5; closes F5, F6)

**Step 1. `tool/builtin/describe.go` (new).**

```go
package builtin

import (
    "embed"
    "fmt"
    "strings"
    "text/template"
    "time"
)

//go:embed describe/*.md
var describeFS embed.FS

// describeData is what a description template may name (0012-MADR D5).
type describeData struct {
    MaxLines, MaxKB, MaxLineChars int
    DefaultTimeout, MaxTimeout    int // seconds
}

// describeDataFor is o's limits for the templates.
func describeDataFor(o Options) describeData {
    return describeData{
        MaxLines: maxLines, MaxKB: maxBytes / 1024, MaxLineChars: maxLineChars,
        DefaultTimeout: int(defaultTimeout / time.Second), MaxTimeout: int(o.MaxTimeout / time.Second),
    }
}

// renderDescription is describe/<name>.md rendered with d, without its
// final newline.
func renderDescription(name string, d describeData) (string, error) {
    t, err := template.ParseFS(describeFS, "describe/"+name+".md")
    if err != nil {
        return "", err
    }
    var b strings.Builder
    if err := t.Execute(&b, d); err != nil {
        return "", err
    }
    return strings.TrimRight(b.String(), "\n"), nil
}

// description is a built-in tool's rendered description. A broken template
// is a programming error that the first test building the tool finds, as
// tool.New's schema panic is.
func description(name string, o Options) string {
    s, err := renderDescription(name, describeDataFor(o))
    if err != nil {
        panic(fmt.Sprintf("builtin: %s description: %v", name, err))
    }
    return s
}
```

**Step 2. The templates,** under `tool/builtin/describe/`, each ending in one newline, exactly:

- **`read.md`:**

  ```markdown
  Read a file, or list a directory's entries.

  ## Use when

  - Read a large file in a few big pieces rather than many small ones.

  ## Rules

  - Lines come numbered as `N: text`. The numbers are not part of the file, so leave them out of an edit's `oldText`.
  - At most {{.MaxLines}} lines or {{.MaxKB}} KB are shown, and a line longer than {{.MaxLineChars}} characters is cut. The last line of the output gives the `offset` to continue from.
  - Binary files are refused, and images are not sent to the model yet.
  ```

- **`write.md`:**

  ```markdown
  Write a whole file: create it, with any missing parent directories, or replace its content.

  ## Use when

  - To change part of an existing file, use edit instead.
  ```

- **`edit.md`:**

  ```markdown
  Edit one file by replacing text.

  ## Rules

  - Each `edits[].oldText` must occur once in the file as it was before this call, and no two may overlap: put changes to nearby lines into one edit.
  - Keep `oldText` short but unique, and copy it from the file without read's line numbers.
  - Small differences in whitespace, indentation, quotes and escaping are forgiven, but a forgiving match far larger than `oldText` is refused.
  ```

- **`bash.md`:**

  ```markdown
  Run a command with bash, in the working directory or in `workdir`.

  ## Use when

  - Use `workdir` instead of `cd`.
  - For files, prefer read, edit and write to `cat`, `sed` and `echo`.

  ## Rules

  - stdout and stderr come back together, as the last {{.MaxLines}} lines or {{.MaxKB}} KB, ending with the exit status. A longer output is saved whole to a file the output names, which read can open.
  - `timeout` is in seconds: {{.DefaultTimeout}} when not given, at most {{.MaxTimeout}}.
  ```

- **`powershell.md`:**

  ```markdown
  Run a command with PowerShell (pwsh when installed, else Windows PowerShell), in the working directory or in `workdir`.

  ## Use when

  - Use `workdir` instead of `Set-Location`.

  ## Rules

  - Output comes back as the last {{.MaxLines}} lines or {{.MaxKB}} KB, ending with the exit status. A longer output is saved whole to a file the output names, which read can open.
  - `timeout` is in seconds: {{.DefaultTimeout}} when not given, at most {{.MaxTimeout}}.
  - Windows PowerShell 5.1 has no `&&` or `||`: separate commands with `;` and check `$LASTEXITCODE`.
  ```

**C5's list of what moved.** Every sentence of today's five descriptions is in its template, reworded only to fit a heading or a list item. Two changes:

- bash's and powershell's `120` and `600`, and read's `2000`, `50` and `2000`, are now filled from the options, so a raised `MaxTimeout` is reported truly;
- read's "the last line of the output says the offset to continue from" now names `offset` as code.

**Step 3. The constructors.**

- `readWith` passes `description("read", o)` in place of `readDescription`, and the `readDescription` variable is deleted.
- `Write` passes `description("write", Options{}.withDefaults())` in place of its inline string.
- `Edit` passes `description("edit", Options{}.withDefaults())`, and `editDescription` is deleted.
- `bashWith` passes `description("bash", o)`, and `bashDescription` is deleted.
- `powershellWith` passes `description("powershell", o)`, and `powershellDescription` is deleted.

**Step 4. `tool/builtin/describe_test.go` (new).** It is package `builtin`.

- **`TestDescriptionShape`** renders every file `fs.Glob(describeFS, "describe/*.md")` returns, with `describeDataFor(Options{}.withDefaults())`. It asserts that the set of names is exactly `bash, edit, powershell, read, write`. For each, it fails when any of these holds:
  1. the first paragraph (up to the first blank line) has a newline, is over 160 bytes, starts with `#`, or does not end with `.`;
  2. a heading line is not exactly `## Use when`, `## Rules` or `## Example`, or one repeats, or they are out of that order;
  3. a line starts with `|`, or with an asterisk and a space, or the text contains `<` followed by a letter or `/`;
  4. the text is over 1,200 bytes.

  It also asserts:
  - for each tool of `ToolsWith(Options{})` that `Spec().Description` equals `description(name, Options{}.withDefaults())`;
  - and that the definition, `json.Marshal` of `{name, description, input_schema}`, is at most 2,000 bytes.
- **`TestDescriptionUsesOptions`** asserts that `description("bash", Options{MaxTimeout: 20 * time.Minute}.withDefaults())` contains `at most 1200`.

**Step 5. `docs/architecture.md`.** In the `tool/builtin` row, append this sentence to the role: `Descriptions are Markdown templates in describe/, filled from the options.`

**Verification.**

```text
go test ./tool/builtin -count=1 -run 'TestDescription' -v                    -> ok
grep -n "readDescription\|editDescription\|bashDescription\|powershellDescription" tool/builtin/*.go  -> no lines
npx --no-install markdownlint-cli2 "tool/builtin/describe/*.md"              -> "0 issues"
make preflight                                                              -> "preflight passed" (its markdownlint covers describe/)
```

**Seen failing (C3), on scratch copies:**

- **copy 1:** `read.md`'s first line is lengthened to 200 bytes. Rule 1 fails.
- **copy 2:** `edit.md` gains `## Notes`. Rule 2 fails.
- **copy 3:** `write.md` gains 1,300 bytes. Rule 4 fails.
- **copy 4:** `bash.md`'s `{{.MaxTimeout}}` is replaced with `600`. TestDescriptionUsesOptions fails.

### P5 — The strict probe (D3; makes Probe D repeatable in the tree)

*(2026-10-08, before approval: the MADR's Probe D has already measured, from a scratch module, what this phase's live test measures. P5 still builds the test, so that the measurement can be repeated from the tree. D3's opt-out then makes the test assert, under Deferred.)*

**Step 1. `llm/provider/strict_test.go` (new).**

- It holds the parser, package `provider_test`, with no build tag, so default CI runs its unit test:

  ```go
  // strictOf reads a Responses reply, an event stream or one response
  // object, and returns the strict value it reports for each tool
  // (0012-MADR D3).
  func strictOf(body []byte) (map[string]bool, error)
  ```

- **How it reads the reply.** *(Amended 2026-10-08: Probe D received one response object, not a stream.)*
  - **A stream:** it reads the lines that begin with `data:` and a space, decodes each as `{"type": string, "response": {"tools": [{"name": string, "strict": bool}]}}`, and returns the tools of the first event whose `type` is `response.created`.
  - **Otherwise,** it decodes the whole body as a response object, `{"tools": [{"name": string, "strict": bool}]}`, and returns its tools.
  - **Its error,** when neither form names any tools, is `no tools in the reply`.
- **`TestStrictOf`** has three cases:
  - a canned two-event stream: a `response.created` with `record` strict `true` and `other` strict `false`, then a `response.completed`. It asserts the map `{record: true, other: false}`;
  - a canned response object with the same two tools, asserting the same map;
  - a stream with only `response.completed`, asserting the error.

**Step 2. `llm/provider/strict_live_test.go` (new),** with build tag `//go:build live_openai`, package `provider_test`:

- **`TestStrictProbe`** builds the provider:
  - it gets the key with `llmtest.LiveCredential(t, "OPENAI_API_KEY")`, which skips without the variable;
  - it calls `provider.New("openai", provider.Options{APIKey: key, Model: "gpt-4.1-mini", HTTPClient: &http.Client{Transport: rec}})`. The model is named because the provider has none of its own: Probe D's first call without one got HTTP 404 `model_not_found` for an empty model name;
  - `rec` is a `RoundTripper` that records the request path and tees the response body into a buffer.
- **The request:**
  - `Tools` is `json.Marshal` of the `Spec()` of each tool of `builtin.Tools()`;
  - the one user message is `Reply with the word ok. Do not call a tool.`;
  - the stream is drained.
- **It fails when any of these holds:**
  - the stream returns an error;
  - the recorded path does not end in `/responses`;
  - `strictOf` returns an error.
- **Otherwise it logs** one line per tool, `strict=<value> tool=<name>`, and passes. It records; it does not judge.
- **Imports.** It imports `tool/builtin`. That is an `llm/provider → tool/builtin` edge in a test only, and no rule of 0004-MADR's import boundaries forbids it. `make archtest` confirms.

**Verification.**

```text
go test ./llm/provider -count=1 -run TestStrictOf -v                          -> ok
go vet -tags live_openai ./llm/provider                                       -> exit 0
go test -tags live_openai -count=1 -run TestStrictProbe -v ./llm/provider     -> "--- SKIP" "set OPENAI_API_KEY to run it" (no key)
make archtest                                                                 -> ok
```

**Seen failing (C3), on scratch copies:**

- **copy 1:** `strictOf` returns the tools of the last `data:` event instead of `response.created`'s. TestStrictOf's stream case fails.
- **copy 2:** the response-object branch is deleted. Its case fails.

**The live run.** `go test -tags live_openai -count=1 -run TestStrictProbe -v ./llm/provider` runs with `OPENAI_API_KEY` in the test process's environment, and nowhere else. Either:

- the owner runs it in a shell where the variable is already set; or
- with the owner's permission given in that turn, the agent runs it with the value loaded from the owner's key file into that one process, as Probe D was.

Either way, the key never appears on a command line, in a file in the tree, or in the transcript. Before anything reads the test's log, the log is checked for key-shaped text, and is read only if it has none. **The expected result** is Probe D's: `strict=true` for all five tools.

### P6 — Record (all)

**Step 1.** Append `## Execution record (<date>)` to this PLAN:

- which phases ran;
- each C3 failure, quoted;
- the gate results;
- what this plan predicted wrongly.

**Step 2.** Append `## Observed — execution results (<date>)` to 0012-MADR.

- ~~**When the live run has happened,** the section gives the probe's `strict=` lines per tool and what D3 concludes from them:~~
  - ~~all `true`: the convention holds;~~
  - ~~any `false`: a deviation, put to the owner, with a per-provider adapter as the candidate fix (D3).~~
- ~~**When it has not,** the section says so, and P6 is re-opened when it has.~~
- *(Amended 2026-10-08, after Probe D and the owner's D3 choice.)* The section gives the in-tree probe's `strict=` lines per tool, compared with Probe D's.
  - **When they agree** (all `true`), D3's opt-out is still needed, and still waits on the SDK.
  - **When they differ,** that is a deviation, put to the owner.
  - **When the live run has not happened,** the section says so, and P6 is re-opened when it has.

**Step 3.** `docs/README.md`: the 0012-PLAN row reads `completed (<date>)`, or `in progress (P1–P5 complete; the live probe pending)`.

**Verification.** `make check-records` exits 0, and `make preflight` prints `preflight passed`.

## Verification (whole plan)

```text
make preflight                       (Windows)                    -> "preflight passed"
WSL copy: make preflight; go test -race ./...                     -> "preflight passed"; exit 0
golangci-lint run ./...   GOOS=linux, darwin, windows             -> "0 issues." each
go test ./tool/... ./llm/... -count=1                             -> ok
git diff <the records commit>..HEAD -- go.mod go.sum              -> empty (C2)
```

### Acceptance criteria (mapped to MADR Confirmation)

| # | Criterion | MADR |
| :--- | :--- | :--- |
| A1 | 0012-MADR is accepted, and 0005-MADR and 0005-PLAN carry its amendment and notes; `make check-records` passes | Decision Outcome, "Changes to other records" |
| A2 | `TestPublishedSchema` passes and failed on both scratch copies; existing `tool` tests pass unedited | Confirmation, D2 rule 8 |
| A3 | `TestSchemaConvention` passes and failed before the hooks | Confirmation, D2 |
| A4 | `TestBoundsHold` passes; the empty-command subtest failed without the refusal | Confirmation, D2 rule 8 |
| A5 | Probe A shows the published bounds, `minItems` and no `null` | What was measured (Probe A), F3 |
| A6 | `TestDescriptionShape` and `TestDescriptionUsesOptions` pass and failed on four scratch copies; no description string remains in Go | Confirmation, D5 |
| A7 | `TestStrictOf` passes and failed on its scratch copy; the live test builds and skips without a key | Confirmation, D3 |
| A8 | The in-tree probe's `strict=` lines are recorded in 0012-MADR beside Probe D's, or the record says the run is pending | D3 (amended) |
| A9 | `go.mod` and `go.sum` unchanged | Decision Drivers 6; C2 |
| A10 | every gate in the Stability rule passes after each code phase | Confirmation, `make preflight` |

**A8 is the criterion most likely to be dropped.** Probe D has already answered the question, so repeating it from the tree looks optional. It is not: the in-tree test is what P7 will turn into an assertion. A "pending" in the record is honest; a silently missing section is not.

**C1 is the contract most likely to be broken.** When P2 or P3 turns a test red, editing its assertion is the fastest route back to green.

## Rollout and Rollback

- **Rollout.** One commit per phase, made by the owner, in order. Nothing changes at runtime until P3: a model then sees bounds, defaults and `minItems`, and bash refuses an empty command. P4 changes only the wording a model reads.
- **Rollback.**
  - **P2–P5** each revert as one commit. P3 and P4 depend on P2, so they revert first.
  - **P1** is records. A revert of it puts 0005's Exposure paragraph back in force, so it is reverted only with 0012-MADR set back to `proposed`.
  - Reverting is the owner's command, not the agent's.

## Deferred (named, so they are not mistaken for oversights)

- **D6, the system-prompt `tools` section:** F3, where the sectioned prompt is built. Until then, bash's "prefer read, edit and write" sentence stays in its description (P4), and F3 moves it.
- **D7–D9, D11 and D14's client data:** F1d. They need `tool_search`, `todo` and the resource tools, which F1d builds, so building the loader before them would be dead code.
- **D7's MCP default:** F8, which applies `exposure`.
- **D10, the interception:** F6, with `mvdan.cc/sh`, which AGENTS.md lets in only with the code that imports it.
- **D4 and D14's `outputSchema`:** X4, with `gobble mcp serve`.
- **D13:** a request to go-llmprovider-sdk. It is recorded in 0012-MADR, and not made from this repository.
- **D3's opt-out** *(added 2026-10-08)*. It waits on a go-llmprovider-sdk release whose `llmprovider.Tool` carries a per-tool strict flag; v1.2.1 has none (V25). When that release exists, this plan gains a phase P7, made executable as an amendment first, that:
  - requires that release, in the same commit as the code that uses the flag (AGENTS.md, Dependencies);
  - in `llm/provider`, where each `llmprovider.Tool` is built (`provider.go:234`), sets strict to false when the published schema has a property outside `required`, and true otherwise;
  - makes P5's live test assert `strict=false` for read, bash and powershell, and `strict=true` for write and edit;
  - adds a unit test, seen failing first, of the rule that sets the flag.
- ~~**The per-provider schema adapter:** built only if P5's probe, or a provider's error, shows a failure (D3).~~ *(2026-10-08: rejected for now by the owner's D3 choice; Probe D showed it only partly works.)*
