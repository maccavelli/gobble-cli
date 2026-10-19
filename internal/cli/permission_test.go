package cli

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

func TestPermissionLine(t *testing.T) {
	line, keys, _ := permissionLine(acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "write \x1b[31ma.txt"},
		Options: []acpclient.PermissionOption{{ID: "ok", Kind: "allow_once"}, {ID: "no", Kind: "reject_once"}}})
	want := strings.Join(strings.Fields(term.Sanitize("write \x1b[31ma.txt")), " ") + "?  [y] allow once  [n] deny  [c] cancel "
	if line != want || strings.Contains(line, "\x1b") {
		t.Fatalf("line %q, want %q with no escape", line, want)
	}
	if len(keys) != 2 || keys['y'] != "ok" || keys['n'] != "no" {
		t.Fatalf("keys %v", keys)
	}
	// A tool call's question, which offers reject_once, ends with [f]
	// (0005-PLAN F6-2b decision F).
	line, keys, _ = permissionLine(acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "write a.txt"},
		Options: []acpclient.PermissionOption{{ID: "allow_once", Kind: "allow_once"}, {ID: "reject_once", Kind: "reject_once"}}})
	if !strings.HasSuffix(line, "[n] deny  [f] deny and say why  [c] cancel ") || keys['f'] != chooseWhy {
		t.Fatalf("line %q, keys %v", line, keys)
	}
}

// The reason is read up to Enter, echoed, with backspace erasing and a rune
// split across reads kept whole; Ctrl+C and the end of input cancel; it is
// trimmed and cut to maxWhy (decision F).
func TestReadWhy(t *testing.T) {
	feed := func(chunks ...string) func([]byte) (int, error) {
		return func(b []byte) (int, error) {
			if len(chunks) == 0 {
				return 0, os.ErrClosed
			}
			n := copy(b, chunks[0])
			chunks[0] = chunks[0][n:]
			if chunks[0] == "" {
				chunks = chunks[1:]
			}
			return n, nil
		}
	}
	var echo strings.Builder
	got, ok := readWhy(feed(" use gitt", "\x7f mv caf\xc3", "\xa9 \r ignored"), func(s string) { echo.WriteString(s) })
	if !ok || got != "use git mv café" {
		t.Fatalf("%q, %v", got, ok)
	}
	if e := echo.String(); !strings.Contains(e, "gitt\b \b mv café") {
		t.Errorf("echo %q", e)
	}
	for _, in := range []string{"no\x03more\r", "no end"} {
		if got, ok := readWhy(feed(in), func(string) {}); ok {
			t.Errorf("%q: %q, want a cancel", in, got)
		}
	}
	if got, _ := readWhy(feed(strings.Repeat("é", maxWhy)+"\n"), func(string) {}); len(got) != maxWhy || !utf8.ValidString(got) {
		t.Errorf("cut to %d bytes, valid %v", len(got), utf8.ValidString(got))
	}
}

// D16's prompt, numbered for a tool call's eight options: 1 lets a write
// run once, 5 refuses it, Ctrl+C cancels it.
func TestPermissionPrompt(t *testing.T) {
	for _, tc := range []struct {
		name, key, label string
		runs             bool
	}{
		{"allow", "1", "Allow once", true},
		{"deny", "5", "Reject", false},
		{"cancel", "\x03", "cancel", false},
		// [f] reads the reason, and rejects once with it (decision F).
		{"why", "fuse git\x7f\x7f\x7fgit mv\r", "", false},
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
			// A tool call's eight options share kinds, so each is numbered
			// (0005-PLAN F6-2b decision D).
			waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "[1] Allow once") })
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
			if tc.name == "why" {
				if !strings.Contains(h.stderr.String(), "[c] cancel deny and say why: use git") {
					t.Fatalf("stderr %q lacks the reason's prompt", h.stderr.String())
				}
				if got := lastToolResult(t, model); !strings.Contains(got, "The user said: use git mv") {
					t.Fatalf("the model read %q", got)
				}
				return
			}
			if !strings.Contains(h.stderr.String(), "[c] cancel "+tc.label+"\n") {
				t.Fatalf("stderr %q lacks the answer %q", h.stderr.String(), tc.label)
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
// (decision A).
func TestPermissionPromptOnceOnly(t *testing.T) {
	dir := t.TempDir()
	model := llmtest.NewScript(llmtest.ToolCall("b1", "bash", `{"command":"./run-*.sh"}`), llmtest.Text("done"))
	h := newModelHarness(t, model, dir)
	h.start(Prompt{})
	h.press("list them\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "[c] cancel ") })
	h.press("n")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "done") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if e := h.stderr.String(); !strings.Contains(e, "(no rule can match this call alone: its command or a path holds * ? [ or {)\n") ||
		!strings.Contains(e, "[y] allow once  [n] deny  [f] deny and say why  [c] cancel ") {
		t.Fatalf("stderr %q", e)
	}
}
