package cli

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

func TestPermissionLine(t *testing.T) {
	line, keys := permissionLine(acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "write \x1b[31ma.txt"},
		Options: []acpclient.PermissionOption{{ID: "ok", Kind: "allow_once"}, {ID: "no", Kind: "reject_once"}}})
	want := strings.Join(strings.Fields(term.Sanitize("write \x1b[31ma.txt")), " ") + "?  [y] allow once  [n] deny  [c] cancel "
	if line != want || strings.Contains(line, "\x1b") {
		t.Fatalf("line %q, want %q with no escape", line, want)
	}
	if len(keys) != 2 || keys['y'] != "ok" || keys['n'] != "no" {
		t.Fatalf("keys %v", keys)
	}
}

// D16's prompt: y lets a write run, n refuses it, Ctrl+C cancels it.
func TestPermissionPrompt(t *testing.T) {
	for _, tc := range []struct {
		name, key, label string
		runs             bool
	}{
		{"allow", "y", "allow once", true},
		{"deny", "n", "deny", false},
		{"cancel", "\x03", "cancel", false},
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
			waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "[y] allow once") })
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
			if !strings.Contains(h.stderr.String(), "[c] cancel "+tc.label+"\n") {
				t.Fatalf("stderr %q lacks the answer %q", h.stderr.String(), tc.label)
			}
		})
	}
}
