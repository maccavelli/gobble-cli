package acpserver

import (
	"bytes"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/jsonx"
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

// session.BlockImage survives the history both ways, and a text-only user
// entry stays one quoted string (0036-MADR D8).
func TestHistoryImageRoundTrip(t *testing.T) {
	raw := []byte("PNG")
	b64 := base64.StdEncoding.EncodeToString(raw)
	msg := &session.Message{Role: session.RoleUser}
	if err := msg.SetBlocks([]session.Block{
		{Type: session.BlockText, Text: "look"},
		{Type: session.BlockImage, MimeType: "image/png", Data: b64},
	}); err != nil {
		t.Fatal(err)
	}
	st := &state{}
	st.addMessage(msg)
	if len(st.history) != 1 || len(st.history[0].Content) != 2 {
		t.Fatalf("history %+v, want the text and the image", st.history)
	}
	img := st.history[0].Content[1]
	if img.Type != llm.ContentImage || img.MediaType != "image/png" || !bytes.Equal(img.Data, raw) {
		t.Fatalf("content %+v, want the image's raw bytes", st.history[0].Content)
	}
	back := userEntryMessage(st.history[0], "2026-10-31T12:00:00.000Z")
	blocks, err := back.Blocks()
	if err != nil || len(blocks) != 2 || blocks[0].Type != session.BlockText || blocks[0].Text != "look" ||
		blocks[1].Type != session.BlockImage || blocks[1].MimeType != "image/png" || blocks[1].Data != b64 {
		t.Fatalf("blocks %+v, %v", blocks, err)
	}
	again := &state{}
	again.addMessage(back)
	if len(again.history) != 1 || len(again.history[0].Content) != 2 || !bytes.Equal(again.history[0].Content[1].Data, raw) {
		t.Fatalf("again %+v, want the same bytes", again.history)
	}
	only := userEntryMessage(llm.Message{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "look"}}}, "2026-10-31T12:00:00.000Z")
	if string(only.Content) != string(jsonx.Quote("look")) {
		t.Fatalf("text-only content %s, want one quoted string", only.Content)
	}
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

// A tool call's arguments, in jsonx.StoredArguments' form, are what the
// session file gives back: the same text from assistantEntryMessage, from a
// write and a read through the jsonl store, and from assistantContent
// (0037-MADR D42, D44). The cases are the ones a model's own spacing or
// escaping changes, and the ones the store keeps as raw text.
func TestToolArgsAsStored(t *testing.T) {
	cases := []struct{ name, args, want string }{
		{"compact", `{"path":"a.go"}`, `{"path":"a.go"}`},
		{"spaced", `{"path": "a.go"}`, `{"path":"a.go"}`},
		{"multi-line", "{\n  \"path\": \"a.go\",\n  \"limit\": 5\n}", `{"path":"a.go","limit":5}`},
		{"non-ASCII", `{"path":"café.go"}`, `{"path":"café.go"}`},
		{"escaped slash", `{"path":"a\/b.go"}`, `{"path":"a/b.go"}`},
		{"key order", `{"b":1,"a":2}`, `{"b":1,"a":2}`},
		{"number", `{"n":1.0}`, `{"n":1.0}`},
		{"empty", ``, `{}`},
		{"blank", ` `, ` `},
		{"array", `[1]`, `[1]`},
		{"spaced array", `[ 1 ]`, `[ 1 ]`},
		{"invalid", `{"path":`, `{"path":`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := jsonx.StoredArguments(tc.args)
			if got != tc.want {
				t.Fatalf("StoredArguments(%q) = %q, want %q", tc.args, got, tc.want)
			}
			ev := agent.Message{Message: llm.Message{Role: llm.RoleAssistant, Content: []llm.Content{
				{Type: llm.ContentToolCall, ToolCallID: "c1", ToolName: "read", ToolArgs: tc.args}}}}
			entry, err := assistantEntryMessage(ev, "claude", "m", llm.Model{}, "2026-11-14T12:00:00.000Z")
			if err != nil {
				t.Fatal(err)
			}
			blocks, err := entry.Blocks()
			if err != nil {
				t.Fatal(err)
			}
			if back := assistantContent(blocks); len(back) != 1 || back[0].ToolArgs != got {
				t.Fatalf("assistantContent(entry) = %+v, want arguments %q", back, got)
			}
			// And through a file: a write, and a read by a new store.
			dir := t.TempDir()
			h := session.Header{Type: session.TypeHeader, Version: session.Version, ID: "019a0000-0000-7000-8000-000000000001",
				Timestamp: "2026-11-14T12:00:00.000Z", Cwd: dir}
			log, err := jsonl.New(dir, jsonl.Options{}).Create(t.Context(), h)
			if err != nil {
				t.Fatal(err)
			}
			if err := log.Append(t.Context(), session.Entry{Type: session.TypeMessage, ID: "aaaa0001", Timestamp: h.Timestamp, Message: entry}); err != nil {
				t.Fatal(err)
			}
			if err := log.Close(); err != nil {
				t.Fatal(err)
			}
			_, entries, _, err := jsonl.New(dir, jsonl.Options{}).Read(t.Context(), h.ID)
			if err != nil || len(entries) != 1 {
				t.Fatalf("read back %d entries, %v", len(entries), err)
			}
			blocks, err = entries[0].Message.Blocks()
			if err != nil {
				t.Fatal(err)
			}
			if back := assistantContent(blocks); len(back) != 1 || back[0].ToolArgs != got {
				t.Fatalf("assistantContent(file) = %+v, want arguments %q", back, got)
			}
		})
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
