package cli

import (
	"context"
	"encoding/json/v2"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// toolOptions are acpserver's eight options for a tool call that offers
// "always" (acpserver/question.go), with its names.
func toolOptions() []acpclient.PermissionOption {
	var out []acpclient.PermissionOption
	for i, ids := range [][]string{
		{"allow_once", "allow_session", "allow_project", "allow_always"},
		{"reject_once", "reject_session", "reject_project", "reject_always"},
	} {
		for j, id := range ids {
			kind := [][]string{{kindAllowOnce, kindAllowAlways}, {kindRejectOnce, kindRejectAlways}}[i][min(j, 1)]
			out = append(out, acpclient.PermissionOption{ID: id, Name: "name of " + id, Kind: kind})
		}
	}
	return out
}

// The question built for each kind of request (0029-MADR D1 and D2, as
// amended by 0035-PLAN PQ4), every word boring's (deviation 4).
func TestPermissionQuestion(t *testing.T) {
	cancel := ask.Answer{Key: "c", Label: "cancel"}
	why := ask.Answer{ID: "reject_once", Key: "f", Label: "reject and say why", Echo: "reject", Reason: true}
	rows := func(hides string) []ask.Row {
		return []ask.Row{
			{Label: "Allow", Answers: []ask.Answer{
				{ID: "allow_once", Key: "1", Label: "once", Echo: "allow once"},
				{ID: "allow_session", Key: "2", Label: "session", Echo: "allow for this session"},
				{ID: "allow_project", Key: "3", Label: "project", Echo: "allow for this project"},
				{ID: "allow_always", Key: "4", Label: "everywhere", Echo: "allow everywhere"},
			}},
			{Label: "Reject", Answers: []ask.Answer{
				{ID: "reject_once", Key: "5", Label: "once", Echo: "reject once"},
				{ID: "reject_session", Key: "6", Label: "session", Echo: "reject for this session"},
				{ID: "reject_project", Key: "7", Label: "project", Echo: "reject for this project"},
				{ID: "reject_always", Key: "8", Label: "everywhere" + hides, Echo: "reject everywhere"},
			}},
			{Answers: []ask.Answer{why}},
		}
	}
	for _, tc := range []struct {
		name string
		r    acpclient.PermissionRequest
		want ask.Question
	}{
		{"tool call", acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "cut"},
			FullTitle: "run  git status;\n git log", Saves: []string{"git status", "git log *"}, Options: toolOptions()},
			ask.Question{Title: "Run git status; git log?", After: []string{"Patterns: git status, git log *"}, Rows: rows(""), Cancel: cancel}},
		{"tool call hiding the tool", acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "read a.txt"}, Hides: true, Options: toolOptions()},
			ask.Question{Title: "Read a.txt?", Rows: rows(" (hides the tool)"), Cancel: cancel}},
		{"once only", acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "write a.txt"}, OnceOnly: "no rule\ncan match",
			Options: []acpclient.PermissionOption{{ID: "allow_once", Kind: kindAllowOnce}, {ID: "reject_once", Kind: kindRejectOnce}}},
			ask.Question{Notes: []string{"(no rule can match)"}, Title: "write a.txt?", Rows: []ask.Row{{Answers: []ask.Answer{
				{ID: "allow_once", Key: "y", Label: "allow once"}, {ID: "reject_once", Key: "n", Label: "deny"},
				{ID: "reject_once", Key: "f", Label: "deny and say why", Reason: true}}}}, Cancel: cancel}},
		{"every kind once", acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "go on"}, Options: []acpclient.PermissionOption{
			{ID: "ok", Kind: kindAllowOnce}, {ID: "ok2", Kind: kindAllowAlways}, {ID: "no", Kind: kindRejectOnce}, {ID: "no2", Kind: kindRejectAlways}}},
			ask.Question{Title: "go on?", Rows: []ask.Row{{Answers: []ask.Answer{
				{ID: "ok", Key: "y", Label: "allow once"}, {ID: "ok2", Key: "a", Label: "always"},
				{ID: "no", Key: "n", Label: "deny"}, {ID: "no2", Key: "N", Label: "never"}}}}, Cancel: cancel}},
		// Where two options share a kind, each is numbered by its name
		// (0005-PLAN F6-1 decision 6).
		{"numbered", acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "Trust /p"}, Options: []acpclient.PermissionOption{
			{ID: "a1", Name: "Trust", Kind: kindAllowAlways}, {ID: "a2", Name: "Trust the  parent", Kind: kindAllowAlways},
			{ID: "n", Name: "Do not trust", Kind: kindRejectAlways}}},
			ask.Question{Title: "Trust /p?", Rows: []ask.Row{{Answers: []ask.Answer{
				{ID: "a1", Key: "1", Label: "Trust"}, {ID: "a2", Key: "2", Label: "Trust the parent"},
				{ID: "n", Key: "3", Label: "Do not trust"}}}}, Cancel: cancel}},
		{"untitled", acpclient.PermissionRequest{Options: []acpclient.PermissionOption{{ID: "ok", Kind: kindAllowOnce}}},
			ask.Question{Title: "Run the tool call?", Rows: []ask.Row{{Answers: []ask.Answer{{ID: "ok", Key: "y", Label: "allow once"}}}}, Cancel: cancel}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := permissionQuestion(nil, tc.r); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("question\n%#v\nwant\n%#v", got, tc.want)
			}
		})
	}
}

// keysOf reads in, then reports the end of input.
func keysOf(in string) func(context.Context, []byte) (int, error) {
	return func(_ context.Context, b []byte) (int, error) {
		if in == "" {
			return 0, io.EOF
		}
		n := copy(b, in)
		in = in[n:]
		return n, nil
	}
}

// shownOn is s as a terminal width columns wide shows it, the scrollback
// included.
func shownOn(t *testing.T, s string, width int) string {
	t.Helper()
	scr := inlinetest.NewScreen(width, 40, ansi.WcWidth)
	if _, err := scr.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
	return scr.Text()
}

// askOn draws the question, ignores a key no answer shows, and gives the
// answer and its reason; c, Ctrl+C and the end of input cancel. The
// reason is cut at ask's 2048 bytes, as maxWhy cut it (decision F).
func TestAskOn(t *testing.T) {
	q := permissionQuestion(nil, acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "write a.txt"}, Options: toolOptions()})
	for _, tc := range []struct {
		name, keys, id, reason string
		canceled               bool
	}{
		{"once", "z1", "allow_once", "", false},
		{"never", "8", "reject_always", "", false},
		{"why", "fuse gitt\x7f mv\r", "reject_once", "use git mv", false},
		{"cancel", "c", "", "", true},
		{"ctrl+c", "\x03", "", "", true},
		{"end of input", "", "", "", true},
		{"ctrl+c in a reason", "fno\x03", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var w strings.Builder
			resp, rows, err := askOn(t.Context(), q, 80, false, &w, keysOf(tc.keys))
			if err != nil {
				t.Fatal(err)
			}
			if resp.Canceled != tc.canceled || !tc.canceled && (resp.Answer.ID != tc.id || resp.Reason != tc.reason) {
				t.Fatalf("%+v, want %q %q canceled %v", resp, tc.id, tc.reason, tc.canceled)
			}
			if len(rows) == 0 || !strings.HasPrefix(rows[0], "Write a.txt?") {
				t.Fatalf("rows %q", rows)
			}
		})
	}
	long := strings.Repeat("é", 2048)
	resp, _, err := askOn(t.Context(), q, 80, false, io.Discard, keysOf("f"+long+"\r"))
	if err != nil || len(resp.Reason) != 2048 {
		t.Fatalf("reason of %d bytes, %v; want 2048", len(resp.Reason), err)
	}
}

// 0029-MADR D1's layout, on a terminal: the whole title wrapped, the
// patterns, the two rows and [f], then the answer echoed in words. A
// width above 120 draws at 120 (D5).
func TestAskOnDraws(t *testing.T) {
	q := permissionQuestion(nil, acpclient.PermissionRequest{
		FullTitle: "run echo '=== staged ==='; git diff --cached --stat; echo '=== log ==='; git log --oneline -5",
		Saves:     []string{"echo *", "git diff *", "git log *"}, Hides: true, Options: toolOptions()})
	for _, width := range []int{60, 100} {
		var w strings.Builder
		if _, _, err := askOn(t.Context(), q, width, false, &w, keysOf("2")); err != nil {
			t.Fatal(err)
		}
		got := shownOn(t, w.String(), width)
		for _, want := range []string{
			"Run echo '=== staged ==='; git diff --cached --stat;", "Patterns: echo *, git diff *, git log *",
			"Allow   [1] once  [2] session  [3] project  [4] everywhere", "[8] everywhere (hides the tool)",
			"[f] reject and say why", "[c] cancel", "> allow for this session",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("at %d columns, %q not shown in:\n%s", width, want, got)
			}
		}
		for line := range strings.SplitSeq(got, "\n") {
			if ansi.StringWidth(line) > width {
				t.Errorf("at %d columns, a row of %d: %q", width, ansi.StringWidth(line), line)
			}
		}
	}
	o := &Output{caps: term.Caps{Err: term.Stream{TTY: true, Width: 200}}}
	if got := promptWidth(o); got != 120 {
		t.Errorf("promptWidth at 200 columns: %d, want 120", got)
	}
}

// The prompt in the line session: 1 lets a write run once, 5 refuses it,
// Ctrl+C cancels it, and [f] rejects once with the reason, which the model
// reads.
func TestPermissionPrompt(t *testing.T) {
	for _, tc := range []struct {
		name, key, echo string
		runs            bool
	}{
		{"allow", "1", "> allow once", true},
		{"deny", "5", "> reject once", false},
		{"cancel", "\x03", "> cancel", false},
		// [f] reads the reason, and rejects once with it (decision F).
		{"why", "fuse git\x7f\x7f\x7fgit mv\r", "> reject: use git mv", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			args, err := json.Marshal(map[string]string{"path": "a.txt", "content": "x"})
			if err != nil {
				t.Fatal(err)
			}
			model := llmtest.NewScript(llmtest.ToolCall("w1", "write", string(args)), llmtest.Text("done"))
			h := newModelHarness(t, model, dir)
			h.start(Prompt{})
			h.press("write it\r")
			waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "[1] once") })
			h.press("z" + tc.key)
			waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "done") })
			h.press("\x04")
			if err := h.wait(); err != nil {
				t.Fatal(err)
			}
			_, statErr := os.Stat(filepath.Join(dir, "a.txt"))
			if tc.runs != (statErr == nil) {
				t.Fatalf("key %q: file written %v", tc.key, statErr == nil)
			}
			if got := shownOn(t, h.stderr.String(), 120); !strings.Contains(got, tc.echo) {
				t.Fatalf("stderr lacks %q:\n%s", tc.echo, got)
			}
			if tc.name == "why" {
				if got := lastToolResult(t, model); !strings.Contains(got, "The user said: use git mv") {
					t.Fatalf("the model read %q", got)
				}
			}
		})
	}
}

// lastToolResult is the text of the tool result the script was sent last.
func lastToolResult(t *testing.T, s *llmtest.Script) string {
	t.Helper()
	for _, req := range slices.Backward(s.Requests()) {
		m := req.Messages[len(req.Messages)-1]
		if len(m.Content) > 0 && m.Content[0].ToolCallID != "" {
			return m.Content[0].Text
		}
	}
	t.Fatal("no tool result was sent")
	return ""
}

// A question that offers "once" alone says why on a line before it
// (decision A), and its keys are D16's.
func TestPermissionPromptOnceOnly(t *testing.T) {
	dir := t.TempDir()
	model := llmtest.NewScript(llmtest.ToolCall("b1", "bash", `{"command":"./run-*.sh"}`), llmtest.Text("done"))
	h := newModelHarness(t, model, dir)
	h.start(Prompt{})
	h.press("list them\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "[c] cancel") })
	h.press("n")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "done") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	got := shownOn(t, h.stderr.String(), 120)
	for _, want := range []string{"(no rule can match this call alone: its command or a path holds * ? [ or {)",
		"[y] allow once  [n] deny  [f] deny and say why", "> deny"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stderr lacks %q:\n%s", want, got)
		}
	}
}
