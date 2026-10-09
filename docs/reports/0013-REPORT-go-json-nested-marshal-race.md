# REPORT 0013 — A data race in Go's encoding/json when one Marshal runs inside another

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

This report decides nothing. It records a defect outside gobble that gobble's race gate hits, and the evidence that it is outside gobble. It was found on 2026-10-08 at 0005-PLAN F1c-2's gates. The owner chose, that day, to record it, report it upstream, and keep the race gate as it is (0005-PLAN, F1c-2 deviation 3).

## What was seen

F1c-2's WSL gate, `CGO_ENABLED=1 go test -race ./...` on Go 1.27.2, failed in one test: `internal/cli`'s `TestSessionIDAndName`, with `race detected during execution of test`. Every other package passed.

- **Where the report points.** It names one goroutine for both accesses, in one call:
  - **the read:** `encoding/json/jsontext.getBufferedEncoder` (`jsontext/pools.go:46`, and `:53` in other runs), reached from `encoding/json.Marshal`, called by `jsonschema-go`'s `Schema.MarshalJSON` (`jsonschema/util.go:339`, `jsonschema/schema.go:296`);
  - **the previous write:** `jsontext.(*encoderState).WriteValue`, reached the same way through `orderedProperties.MarshalJSON`.
- **What sits around it.** Both are under the `encoding/json/v2.Marshal` that `tool.New` calls on a tool's published schema (`tool/tool.go`), here building delete, through `builtin.ToolsWith` and `internal/cli`'s `toolSet`.
- **The pattern:** a type's `MarshalJSON` (v1) calls `json.Marshal` again while an outer `Marshal` is still writing. jsonschema-go's `Schema.MarshalJSON` does exactly that for nested schemas.

## The evidence that it is outside gobble

Every run below was in WSL, `CGO_ENABLED=1`, with the race detector, on 2026-10-08.

| Tree | Go | Runs | Failed |
| :--- | :--- | ---: | ---: |
| gobble at `c4ca6f5`, a clean export: none of F1c | 1.27.2 | 30, 200, 400 | 0, 1, 0 |
| gobble, F1c-1 staged (the index) | 1.27.2 | 200 | 0 |
| gobble, the working tree: F1c-1 and F1c-2 | 1.27.2 | 30, 200 | 1, 1 |
| gobble at `c4ca6f5`, with `go 1.27.1` in its `go.mod` | 1.27.1 | 400 | 0 |

That is for `go test -race -count=N -run '^TestSessionIDAndName$' ./internal/cli`.

**The reproducers.** A scratch module with only `jsonschema-go` v0.4.3 and the standard library:

| Test | Go | Runs of 20,000 marshals | Races reported |
| :--- | :--- | ---: | ---: |
| `jsonschema.For[T]` then `json/v2` `Marshal`, as `tool.New` does | 1.27.2 | 20 | 38 |
| the same | 1.27.1 | 20 | 7 |
| the same, with `encoding/json` (v1) `Marshal` at the top | 1.27.2 | 20 | 22 |
| **the standard library alone:** a `MarshalJSON` that calls v1 `json.Marshal`, under a `json/v2` `Marshal` | 1.27.2 | 20 | 3 |
| the standard library alone, v1 `Marshal` at the top | 1.27.2 | 20 | 3 |

**What the evidence shows:**

- ~~**It is in the standard library.** It reproduces with no gobble code, and with no jsonschema-go: the standard library's own JSON packages race when one `Marshal` nests inside another through a `MarshalJSON` method.~~ *(Not established: see the second amendment of 2026-10-09.)*
- **The toolchain bump did not cause it.** It reproduces on Go 1.27.1 as well as 1.27.2, so the move to 1.27.2 (0004-MADR, amendment of 2026-10-08) is not the cause.
- **gobble's race gate has carried it all along.** `tool.New` has marshaled every tool's schema through jsonschema-go since commit `9826454`, which first derived schemas there (`git log -S"jsonschema.For[In]" -- tool/tool.go`). gobble's tests marshal few enough schemas that it shows about once in 200 to 600 runs. More tools mean more marshals, so F1c-1 and F1c-2 make it slightly more likely.
- ~~**It is the same in every case.** Every race reported is the same pair: a read in `getBufferedEncoder`, and a previous write in `WriteValue`, in one goroutine.~~ *(Amended 2026-10-09: wrong; the pair varies. See the amendment.)*

**Inferred, not measured.**

- **[unverified]** Why one goroutine is named for both accesses. The pattern fits an `Encoder` going back into `sync.Pool` while an outer `Marshal` still uses it, then being reset for the inner call (`pools.go:44-55`), but the source has not been traced to prove it.
- **[unverified]** Whether it can corrupt output outside the race detector. No gobble test has seen a wrong schema, and every schema test passes.

## What gobble does about it

By the owner's decision of 2026-10-08:

- **The race gate is unchanged.** It is not skipped, retried or narrowed, so it keeps reporting the defect whenever it shows.
- **A gate record that meets it says so.** It names this report, and that the same failure reproduces on a clean `HEAD`.
- ~~**gobble keeps marshaling schemas through jsonschema-go.** Writing gobble's own schema encoder was considered and not chosen: it would not help other callers, such as the MCP SDK, and gobble would own a serializer that must track jsonschema-go's.~~ *(Superseded 2026-10-09 by the owner's decision in the amendment below: gobble removes its own nested marshal.)*
- **The defect is reported upstream** with the standard-library-only reproducer below. When a Go release fixes it, the toolchain moves to that release with an amendment to 0004-MADR, and this report gains a dated note.

## The reproducer

The standard library alone, as one test file of a module with no requirements:

```go
package racerepro

import (
    jsonv1 "encoding/json"
    "encoding/json/v2"
    "testing"
)

// node marshals itself with encoding/json (v1), nesting a Marshal inside
// the json/v2 Marshal that called it.
type node struct {
    Name     string
    Children []*node
}

func (n *node) MarshalJSON() ([]byte, error) {
    kids := make([]any, len(n.Children))
    for i, c := range n.Children {
        kids[i] = c
    }
    return jsonv1.Marshal(map[string]any{"name": n.Name, "children": kids})
}

func TestNestedMarshal(t *testing.T) {
    tree := &node{Name: "root", Children: []*node{{Name: "a", Children: []*node{{Name: "a1"}}}, {Name: "b"}}}
    for range 20000 {
        if _, err := json.Marshal(tree); err != nil {
            t.Fatal(err)
        }
    }
}
```

`CGO_ENABLED=1 go test -race -count=20 .` reported 3 races on Go 1.27.2, linux/amd64.

## The upstream report, drafted

*(2026-10-09, second amendment: not to be filed. Its premise, a defect in `encoding/json`, is not established.)*

For `golang/go`, to be filed by the owner, or by the agent with the owner's approval:

> **encoding/json/jsontext: race detector reports a race in getBufferedEncoder when MarshalJSON calls json.Marshal**
>
> **What version of Go are you using?** go1.27.2 linux/amd64 (also go1.27.1).
>
> **What did you do?** Marshaled a value whose `MarshalJSON` method calls `encoding/json.Marshal`, from `encoding/json/v2.Marshal` or from `encoding/json.Marshal`, under `go test -race`. (The reproducer above.)
>
> **What did you see?** `WARNING: DATA RACE`: a read in `jsontext.getBufferedEncoder` (`pools.go:46` or `:53`) and a previous write in `jsontext.(*encoderState).WriteValue`, both reported in the same goroutine. About 3 races in 20 runs of 20,000 marshals. `github.com/google/jsonschema-go`'s `Schema.MarshalJSON`, which nests marshals the same way, makes it far more frequent.
>
> The reported pair varies between runs: reads in `runtime.slicecopy`, `jsonwire.ConsumeSimpleString`, `json/v2.getStrings` and `reflect.Value` methods are reported as well, always against a previous write in the same goroutine.
>
> **What did you expect?** No race: there is one goroutine.

## Amendment, 2026-10-09 — the signature varies, and gobble removes its own nested marshal

0005-PLAN F1d-1's WSL race gate met this defect again, on 2026-10-08. The owner decided on 2026-10-09 how gobble answers it (0005-PLAN, F1d-1 deviation 6).

### What was seen

- **The test:** `internal/cli`'s `TestSessionValues`, with `race detected during execution of test`. Every other package passed.
- **The stack:** `tool.New`'s `Marshal` of `tree`'s published schema, through `builtin.ToolsWith` and `toolSet`, as before. No F1d-1 code is in it.
- **The pair was a new one.** It was not the pair this report records:
  - **the read:** `runtime.slicecopy` in `bytes.growSlice`, from `bytes.(*Buffer).Write` in jsonschema-go's `orderedProperties.MarshalJSON` (`jsonschema/schema.go:345`);
  - **the previous write:** `bytes.(*Buffer).WriteByte`, in the same function (`schema.go:338`);
  - both in one goroutine, on a `bytes.Buffer` local to that call.

### The evidence of 2026-10-08 and 2026-10-09

Every run was in WSL, `CGO_ENABLED=1`, Go 1.27.2, with the race detector.

| What ran | Tree | Runs | Races |
| :--- | :--- | ---: | ---: |
| `internal/cli`, `TestSessionValues` and `TestSessionIDAndName` | a clone of `c4ca6f5` | 20 | 0 |
| the same | the F1d-1 working tree | 20 | 0 |
| the whole `internal/cli` package | a clone of `8d681d9`: F1c-1, F1c-2 and the F1d records, no F1d-1 | 15 | 0 |
| the same | the F1d-1 working tree | 15 | 0 |
| the reproducer: `jsonschema.For[T]` then `json/v2` `Marshal`, as `tool.New` does | none of gobble's code | 20, then 60 | 0, then 15 |
| the same, v1 `Marshal` at the top | none of gobble's code | 20, then 60 | 2, then 8 |
| the standard library alone, v1 `Marshal` at the top | none of gobble's code | 20, then 60 | 14, then 0 |
| the standard library alone, as in [The reproducer](#the-reproducer) | none of gobble's code | 60 | 0 |

**What the evidence shows:**

- **The reported pair varies.** Told apart by the first frame of the reported read, the reproducers gave eight kinds: this report's `getBufferedEncoder` pair; reads in `runtime.slicecopy` (from `bytes.Clone` in `json/v2.Marshal`), `jsonwire.ConsumeSimpleString`, `json/v2.getStrings`, `reflect.Value.MapRange`, `reflect.Value.IsNil` and `reflect.Value.lenNonSlice`; and jsonschema-go's `(*Schema).basicChecks`.
  - The jsonschema-go reproducer, with no gobble code, raced in the `json.Marshal` calls that `orderedProperties.MarshalJSON` makes (`schema.go:332` and `:341`): the gate's function, with the read in `bytes.Clone` rather than in the function's own buffer.
  - The gate's exact pair, on that local buffer at `schema.go:338` and `:345`, was not seen again.
- **The rate comes in bursts.** One test gave 0 races in 20 runs, then 15 in 60. Another gave 14 in 20, then 0 in 60. So a single count is a sample, not a rate. This report's own counts of 2026-10-08 are samples in the same way.
- **This gate's failure was not reproduced on gobble without F1d-1.** 15 runs of `internal/cli` without F1d-1, and 15 with it, raced 0 times each. At the rate of about once in a few hundred runs, that sample cannot tell the trees apart.
- ~~**The defect is the same.** The same mechanism, a `Marshal` nested inside another through `MarshalJSON`, is reported in one goroutine, in code that has no gobble in it.~~ *(Not established: see the second amendment of 2026-10-09.)*

### Corrections to this report

- *It is the same in every case* is wrong, and is struck through above.
- The upstream draft's *What did you see* gains a sentence saying the pair varies.
- [docs/README.md](../README.md) called this defect "reported upstream". It has not been filed. Filing still waits on the owner, or on the agent with the owner's approval.

### What gobble does about it, from 2026-10-09

The owner's decision of 2026-10-09 replaces the third bullet of *What gobble does about it*:

- **gobble removes its own nested marshal.** *(On hold from 2026-10-09, with 0012-PLAN P9: see the second amendment.)*
  - `tool.New` is the only place gobble's code marshals a jsonschema-go `Schema`. It is also the only nested marshal in gobble's code: gobble defines no `MarshalJSON` method, and MCP tool schemas arrive as maps.
  - `tool.New` will write the published schema itself, with no nested `Marshal`. The work is a new phase of 0012-PLAN, P9, under an amendment to [0012-MADR](../decisions/0012-MADR-tool-schemas-descriptions-and-loading.md). Both will be written and approved before any code.
  - What it leaves:
    - the standard library's defect, until Go fixes it;
    - and nested marshals in code gobble does not own, such as the MCP SDK's `mcp.AddTool`, which `mcpclient/mcptest`'s fixture server uses in tests.
- **The rest stands.** The race gate is unchanged, and a gate record that meets the defect names this report.
- **"Reproduces on a clean `HEAD`" is not always possible.** At this rate, a gate record names the runs it made on the clean tree and what they showed, rather than claiming a reproduction it does not have.

## Amendment, 2026-10-09 (second) — corrupted memory, not `encoding/json`

0012-PLAN P9's step 7, Probe F, tested the first amendment's account, and contradicted it. On 2026-10-09 the owner decided to pause P9 until the probe has run on another machine (0012-PLAN P9, deviation 2).

### What was seen

Probe F builds gobble's built-in tools 2,000 times a pass (`builtin.Tools()`). Every run was in WSL on this host, with `CGO_ENABLED=1` and `-timeout 0`, except the control, which ran under the default timeout.

| Tree | Go | Mode | Passes | Race reports | Crashes |
| :--- | :--- | :--- | ---: | ---: | :--- |
| the control: `327fc05`, with `tool.New`'s nested `Marshal` | 1.27.2 | `-race` | about 24, stopped at the default 10-minute timeout | 20 | none |
| the P9 tree: no nested `Marshal` in gobble's code | 1.27.2 | `-race` | 60 | 9 | 1: `interface conversion: interface {} is , not []*jsonschema.Schema` |
| the same, run again | 1.27.2 | `-race` | 60 | 2 | 1, the same panic |
| the same, `GOEXPERIMENT=nogreenteagc` | 1.27.2 | `-race` | 60 | 25 | none |
| the same | 1.27.2 | no race detector | 300 | — | 1: `MapIter.Key called on exhausted iterator` |
| the same, `GOEXPERIMENT=jsonv2` | 1.26.3 | `-race` | 120 | 9 | none |

### What it shows

- **No report is one a correct race detector can make.** Each pairs two accesses in one goroutine, or a goroutine's access with the main goroutine's, made before that goroutine was created. Several give an address that a package `init` had used for a different object, since freed.
- **The memory is really corrupted.** Three passes crashed on values no correct program can hold:
  - an interface whose type is empty, twice;
  - a map iterator exhausted at its first key, in a plain build with no race detector.
- **The nested `Marshal` is not the cause.** The P9 tree, with none in gobble's code, raced and crashed. The reports and crashes are in jsonschema-go's inference and `Resolve`, in `reflect`, `regexp` and `encoding/json`'s token reader: wherever the probe allocates.
- **It is not new in Go 1.27,** since Go 1.26.3 shows it. **It is not the Green Tea collector,** which was off in one run.
- **No code on these paths uses `unsafe` or cgo:** gobble's `tool` and `internal/fsx`, and jsonschema-go v0.4.3.
- **Two explanations remain:**
  - **this machine:** its memory, its processor or its WSL virtual machine corrupting a process's memory. The Windows side of this host has a recorded history of corrupted goroutine stacks, attributed to an endpoint-security product's hooks;
  - **a Go runtime defect,** present since 1.26 at least and seen nowhere else.
- **CI has not seen it.** GitHub's Ubuntu and macOS runners run `make race`. None of the last 20 CI runs failed on a race; the three that failed did so on `govulncheck` and on the SBOM step. About 40 race runs is too few to rule out a defect this rare.

### Corrections

- The first section's *It is in the standard library* is not established, and is struck through. The standard-library reproducer's reports are the same impossible pairs.
- The first amendment's *The defect is the same* is struck through, for the same reason.
- The first amendment's *gobble removes its own nested marshal* is on hold, with 0012-PLAN P9.
- The title names `encoding/json`. It is kept, since a record is not renamed, and this amendment says the attribution is not established.
- **The upstream draft is not to be filed.** Its premise is not established.

### The next test

Probe F's control, on another machine. On this host it reported 20 races in about 24 passes, so 24 passes elsewhere separate the two explanations:

```text
git clone <the gobble-cli remote> gobble && git -C gobble checkout f5b69b6
mkdir probef && cd probef
# probe_test.go is 0012-PLAN P9 step 7's test; go.mod: module probef, go 1.27.2, require github.com/maccavelli/gobble-cli v0.0.0
go mod edit -replace=github.com/maccavelli/gobble-cli=../gobble && go mod tidy
CGO_ENABLED=1 go test -race -timeout 0 -count=24 .
```

- **No report there:** this host is the cause. Its race gate cannot be trusted until the machine is fixed, and gobble's records stop attributing its failures to Go.
- **The same kinds of report there:** a Go runtime defect, to be reported upstream with Probe F as the reproducer.

## Sources

- The race reports, from WSL `go test -race` logs of 2026-10-08, kept in the session scratchpad. Their file paths name a home directory, so the stacks above give only function names and source lines in the standard library and jsonschema-go.
- `encoding/json/jsontext/pools.go`, Go 1.27.2: `getBufferedEncoder` at lines 44-55, read on 2026-10-08.
- A search of `github.com`, `go.dev` and `groups.google.com` on 2026-10-08 found no existing report of this race.
- The amendment's runs, from WSL `go test -race` logs of 2026-10-08 and 2026-10-09, kept in the session scratchpad. Their stacks are quoted by function name and source line only, for the same reason.
