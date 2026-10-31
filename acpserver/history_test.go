package acpserver

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// replacement is U+FFFD, as the data policy writes an invalid byte.
var replacement = string(utf8.RuneError)

// savedEntries are session id's entries as the store wrote them under dir.
func savedEntries(t *testing.T, dir, id string) []session.Entry {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*", "*_"+id+".jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("session %s files %v, %v", id, files, err)
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var out []session.Entry
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n")[1:] {
		var e session.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("line %q: %v", line, err)
		}
		out = append(out, e)
	}
	return out
}

// savedToolResult is the text of the saved result of call id.
func savedToolResult(t *testing.T, entries []session.Entry, id string) string {
	t.Helper()
	for _, e := range entries {
		if e.Message != nil && e.Message.Role == session.RoleToolResult && e.Message.ToolCallID == id {
			blocks, err := e.Message.Blocks()
			if err != nil || len(blocks) != 1 {
				t.Fatalf("result %s blocks %+v, %v", id, blocks, err)
			}
			return blocks[0].Text
		}
	}
	t.Fatalf("no saved result for %s", id)
	return ""
}

// A tool result that is not valid UTF-8 is saved with U+FFFD for each
// invalid byte, not emptied (0039-MADR D7; Confirmation 4).
func TestToolResultMessageInvalidUTF8(t *testing.T) {
	m := toolResultMessage("c1", "read", "\xe9t\xe9", false, "2026-10-31T12:00:00.000Z")
	blocks, err := m.Blocks()
	if err != nil || len(blocks) != 1 || blocks[0].Text != replacement+"t"+replacement {
		t.Fatalf("saved %q: blocks %+v, %v", m.Content, blocks, err)
	}
}

// A builder's error is the writer's, and nothing after it is appended: the
// prompt then answers appendFailed's error (0039-MADR D7; Confirmation 4).
func TestWriterKeepsBuildError(t *testing.T) {
	boom := errors.New("boom")
	w := &writer{}
	w.addEntry(session.Entry{}, boom)
	if !errors.Is(w.err, boom) {
		t.Fatalf("w.err = %v, want the build error", w.err)
	}
	w.addEntry(session.Entry{Type: session.TypeCustom}, nil) // w.s is nil: an append would panic
	if !errors.Is(w.err, boom) {
		t.Fatalf("w.err = %v after a later entry, want the first error kept", w.err)
	}
}

// readAgent is an agent with the read tool, as promptAgent builds one, and
// the client it talks to, for a test that reads the client's updates.
func readAgent(t *testing.T, dir string, p llm.Provider) (*acptest.Conn, *acptest.Client) {
	t.Helper()
	var logs bytes.Buffer
	ag := New(Options{Version: "1.2.3", Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Models: func() ModelChoice { return testModels }, Store: jsonl.New(dir, jsonl.Options{}),
		Tools: []tool.Tool{builtin.Read()}, Prompt: PromptOptions{ConfigDir: t.TempDir()},
		Logger: slog.New(slog.NewTextHandler(&logs, nil))})
	client := &acptest.Client{Allow: true}
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c, client
}

// requestToolResults is the text of the tool results req carries.
func requestToolResults(req *llm.Request) string {
	var b strings.Builder
	for _, m := range req.Messages {
		for _, c := range m.Content {
			if c.Type == llm.ContentToolResult {
				b.WriteString(c.Text)
			}
		}
	}
	return b.String()
}
