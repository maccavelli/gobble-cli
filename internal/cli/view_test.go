package cli

import (
	"encoding/json/jsontext"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// stepClock starts at a fixed instant and advances 100 ms per reading.
func stepClock() func() time.Time {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	return func() time.Time {
		now = now.Add(100 * time.Millisecond)
		return now
	}
}

// viewHome is an absolute home on every OS: on Windows it gains the
// current volume, so ShortPath behaves the same everywhere.
func viewHome(t *testing.T) string {
	t.Helper()
	h, err := filepath.Abs("/home/<user>")
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func viewTurn(home string) []acpclient.Update {
	text := func(s string) acpclient.Update { return acpclient.Update{Kind: acpclient.KindAgentText, Text: s} }
	var long strings.Builder
	for i := 1; i <= 30; i++ {
		fmt.Fprintf(&long, "line %d\n", i)
	}
	old := "a\nb\n"
	return []acpclient.Update{
		{Kind: acpclient.KindThought, Text: "weighing the options"},
		text("# Plan\n\nSome **bold** te"),
		text("xt and `code`.\n\n| a | b |\n|---|--:|\n| 1 | 22 |\n\nnext"),
		{Kind: acpclient.KindToolCall, Tool: acpclient.ToolCall{ID: "t1", Title: "read notes.md", Status: "pending"}},
		{Kind: acpclient.KindToolCallUpdate, Tool: acpclient.ToolCall{ID: "t1", Status: "in_progress"}},
		{Kind: acpclient.KindToolCallUpdate, Tool: acpclient.ToolCall{ID: "t1", Status: "completed",
			Content: []acpclient.ToolContent{{Text: long.String()}}}},
		{Kind: acpclient.KindToolCall, Tool: acpclient.ToolCall{ID: "t2", Title: "edit a.go \x1b[2J", Status: "completed",
			Content: []acpclient.ToolContent{
				{Diff: &acpclient.Diff{Path: filepath.Join(home, "src", "a.go"), OldText: &old, NewText: "a\nB\nc\n"}},
				{Diff: &acpclient.Diff{Path: filepath.Join(home, "new.go"), NewText: "x\n"}},
				{Terminal: "term-1"},
			}}},
		{Kind: acpclient.KindPlan, Plan: []acpclient.PlanEntry{{Content: "read", Status: "completed"}, {Content: "fix", Status: "in_progress"}}},
		{Kind: acpclient.KindAgentText, Block: "image"},
		text("Done."),
		{Kind: acpclient.KindUsage, Context: acpclient.ContextUsage{Used: 150000, Size: 200000, Cost: &acpclient.Cost{Amount: 0.5, Currency: "USD"}}},
		{Kind: acpclient.KindOther},
	}
}

func renderTurn(t *testing.T, color bool, opts display, res acpclient.Result) (stdout, stderr string) {
	t.Helper()
	var out, errw strings.Builder
	s := term.Stream{TTY: true, Color: color, Width: 60}
	o := &Output{out: &out, err: &errw, caps: term.Caps{In: true, Out: s, Err: s, Unicode: true}}
	opts.home = viewHome(t)
	v := newView(o, opts, stepClock())
	for _, u := range viewTurn(opts.home) {
		v.on(u)
	}
	if err := v.finish(res); err != nil {
		t.Fatal(err)
	}
	vis := func(s string) string { return strings.ReplaceAll(s, "\x1b", "<ESC>") }
	return vis(out.String()), vis(errw.String())
}

func TestViewGoldens(t *testing.T) {
	res := acpclient.Result{StopReason: "end_turn", Usage: &acpclient.Usage{OutputTokens: 100}}
	for _, tc := range []struct {
		name  string
		color bool
		opts  display
	}{
		{"view-color", true, display{}},
		{"view-plain", false, display{}},
		{"view-thinking-verbose-stats", false, display{showThinking: true, verbose: true, stats: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr := renderTurn(t, tc.color, tc.opts, res)
			golden(t, tc.name, stdout+"--- stderr\n"+stderr)
		})
	}
}

// The view draws the real edit tool's change as a unified diff (0002-PLAN
// Phase 3 step 6).
func TestViewDrawsARealEdit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\ntwo\nthree\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := builtin.Edit().Run(t.Context(), tool.Call{Name: "edit", Args: jsontext.Value(`{"path":"a.txt","oldText":"two","newText":"2"}`)}, tool.Env{Cwd: dir})
	if err != nil || len(res.Diffs) != 1 {
		t.Fatalf("edit = %+v, %v", res, err)
	}
	d := res.Diffs[0]
	var out, errw strings.Builder
	s := term.Stream{Width: 80}
	v := newView(&Output{out: &out, err: &errw, caps: term.Caps{Out: s, Err: s}}, display{home: dir}, stepClock())
	v.on(acpclient.Update{Kind: acpclient.KindToolCall, Tool: acpclient.ToolCall{ID: "e1", Title: "edit a.txt", Status: "completed",
		Content: []acpclient.ToolContent{{Diff: &acpclient.Diff{Path: d.Path, OldText: d.OldText, NewText: d.NewText}}}}})
	if err := v.finish(acpclient.Result{StopReason: "end_turn"}); err != nil {
		t.Fatal(err)
	}
	want := "--- ~/a.txt\n+++ ~/a.txt\n@@ -1,3 +1,3 @@\n one\n-two\n+2\n three\n"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("view drew\n%s\nwant it to contain\n%s", out.String(), want)
	}
}

// A stop reason other than end_turn is named on the status line.
func TestViewStopReason(t *testing.T) {
	_, stderr := renderTurn(t, false, display{}, acpclient.Result{StopReason: "cancelled"})
	if !strings.HasSuffix(stderr, " • cancelled\n") {
		t.Fatalf("status line %q does not end with the stop reason", stderr)
	}
}

// The status line stays dim across the bar's own colour reset.
func TestDimmedReentersAfterReset(t *testing.T) {
	got := dimmed("1s • \x1b[32mbar\x1b[0m 5%", true)
	want := "\x1b[2m1s • \x1b[32mbar\x1b[0;2m 5%\x1b[0m"
	if got != want {
		t.Fatalf("dimmed = %q, want %q", got, want)
	}
}
