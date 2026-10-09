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

- **It is in the standard library.** It reproduces with no gobble code, and with no jsonschema-go: the standard library's own JSON packages race when one `Marshal` nests inside another through a `MarshalJSON` method.
- **The toolchain bump did not cause it.** It reproduces on Go 1.27.1 as well as 1.27.2, so the move to 1.27.2 (0004-MADR, amendment of 2026-10-08) is not the cause.
- **gobble's race gate has carried it all along.** `tool.New` has marshaled every tool's schema through jsonschema-go since commit `9826454`, which first derived schemas there (`git log -S"jsonschema.For[In]" -- tool/tool.go`). gobble's tests marshal few enough schemas that it shows about once in 200 to 600 runs. More tools mean more marshals, so F1c-1 and F1c-2 make it slightly more likely.
- **It is the same in every case.** Every race reported is the same pair: a read in `getBufferedEncoder`, and a previous write in `WriteValue`, in one goroutine.

**Inferred, not measured.**

- **[unverified]** Why one goroutine is named for both accesses. The pattern fits an `Encoder` going back into `sync.Pool` while an outer `Marshal` still uses it, then being reset for the inner call (`pools.go:44-55`), but the source has not been traced to prove it.
- **[unverified]** Whether it can corrupt output outside the race detector. No gobble test has seen a wrong schema, and every schema test passes.

## What gobble does about it

By the owner's decision of 2026-10-08:

- **The race gate is unchanged.** It is not skipped, retried or narrowed, so it keeps reporting the defect whenever it shows.
- **A gate record that meets it says so.** It names this report, and that the same failure reproduces on a clean `HEAD`.
- **gobble keeps marshaling schemas through jsonschema-go.** Writing gobble's own schema encoder was considered and not chosen: it would not help other callers, such as the MCP SDK, and gobble would own a serializer that must track jsonschema-go's.
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

For `golang/go`, to be filed by the owner, or by the agent with the owner's approval:

> **encoding/json/jsontext: race detector reports a race in getBufferedEncoder when MarshalJSON calls json.Marshal**
>
> **What version of Go are you using?** go1.27.2 linux/amd64 (also go1.27.1).
>
> **What did you do?** Marshaled a value whose `MarshalJSON` method calls `encoding/json.Marshal`, from `encoding/json/v2.Marshal` or from `encoding/json.Marshal`, under `go test -race`. (The reproducer above.)
>
> **What did you see?** `WARNING: DATA RACE`: a read in `jsontext.getBufferedEncoder` (`pools.go:46` or `:53`) and a previous write in `jsontext.(*encoderState).WriteValue`, both reported in the same goroutine. About 3 races in 20 runs of 20,000 marshals. `github.com/google/jsonschema-go`'s `Schema.MarshalJSON`, which nests marshals the same way, makes it far more frequent.
>
> **What did you expect?** No race: there is one goroutine.

## Sources

- The race reports, from WSL `go test -race` logs of 2026-10-08, kept in the session scratchpad. Their file paths name a home directory, so the stacks above give only function names and source lines in the standard library and jsonschema-go.
- `encoding/json/jsontext/pools.go`, Go 1.27.2: `getBufferedEncoder` at lines 44-55, read on 2026-10-08.
- A search of `github.com`, `go.dev` and `groups.google.com` on 2026-10-08 found no existing report of this race.
