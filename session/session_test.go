package session

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/jsonx"
)

// Lines in Pi's shapes (session-format.md, message-types.md at 312184edb,
// with valid JSON) decode, and an unchanged re-encode is the same JSON.
func TestPiShapesRoundTrip(t *testing.T) {
	lines := []string{
		`{"type":"session","version":3,"id":"019a1b2c-0000-7000-8000-000000000001","timestamp":"2024-12-03T14:00:00.000Z","cwd":"/path/to/project","parentSession":"/p/s.jsonl"}`,
		`{"type":"message","id":"a1b2c3d4","parentId":null,"timestamp":"2024-12-03T14:00:01.000Z","message":{"role":"user","content":"Hello","timestamp":1733234401000}}`,
		`{"type":"message","id":"b2c3d4e5","parentId":"a1b2c3d4","timestamp":"2024-12-03T14:00:02.000Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hm","thinkingSignature":"sig"},{"type":"text","text":"Hi!"},{"type":"toolCall","id":"call_123","name":"bash","arguments":{"command":"ls"}}],"api":"anthropic-messages","provider":"anthropic","model":"claude-sonnet-4-5","usage":{"input":10,"output":5,"cacheRead":0,"cacheWrite":0,"totalTokens":15,"cost":{"input":0,"output":0,"cacheRead":0,"cacheWrite":0,"total":0}},"stopReason":"toolUse","timestamp":1733234402000}}`,
		`{"type":"message","id":"c3d4e5f6","parentId":"b2c3d4e5","timestamp":"2024-12-03T14:00:03.000Z","message":{"role":"toolResult","toolCallId":"call_123","toolName":"bash","content":[{"type":"text","text":"output"}],"isError":false,"timestamp":1733234403000}}`,
		`{"type":"model_change","id":"d4e5f6a7","parentId":"c3d4e5f6","timestamp":"2024-12-03T14:05:00.000Z","provider":"openai","modelId":"gpt-4o"}`,
		`{"type":"thinking_level_change","id":"e5f6a7b8","parentId":"d4e5f6a7","timestamp":"2024-12-03T14:06:00.000Z","thinkingLevel":"high"}`,
		`{"type":"compaction","id":"f6a7b8c9","parentId":"e5f6a7b8","timestamp":"2024-12-03T14:10:00.000Z","summary":"User discussed X","firstKeptEntryId":"c3d4e5f6","tokensBefore":50000}`,
		`{"type":"compaction","id":"f6a7b8ca","parentId":"f6a7b8c9","timestamp":"2024-12-03T14:11:00.000Z","summary":"## Goal\nX","firstKeptEntryId":"f6a7b8ca","tokensBefore":0,"details":{"readFiles":["a.go"],"modifiedFiles":[]},"usage":{"input":10,"output":5,"cacheRead":0,"cacheWrite":0,"totalTokens":15,"cost":{"input":0,"output":0,"cacheRead":0,"cacheWrite":0,"total":0}},"fromHook":false}`,
		`{"type":"label","id":"a7b8c9d0","parentId":"f6a7b8c9","timestamp":"2024-12-03T14:30:00.000Z","targetId":"a1b2c3d4","label":"checkpoint-1"}`,
		`{"type":"session_info","id":"b8c9d0e1","parentId":"a7b8c9d0","timestamp":"2024-12-03T14:35:00.000Z","name":"Refactor auth module"}`,
		`{"type":"custom","id":"c9d0e1f2","parentId":"b8c9d0e1","timestamp":"2024-12-03T14:20:00.000Z","customType":"my-extension","data":{"count":42}}`,
		// 0005-PLAN F2: null, false and "" must survive as written.
		`{"type":"context_edit","id":"d0e1f2a3","parentId":"c9d0e1f2","timestamp":"2024-12-03T14:21:00.000Z","targetId":"a1b2c3d4","replacement":null}`,
		`{"type":"context_edit","id":"d0e1f2a4","parentId":"d0e1f2a3","timestamp":"2024-12-03T14:21:01.000Z","targetId":"b2c3d4e5","replacement":{"content":[{"type":"text","text":"x"}]}}`,
		`{"type":"custom_message","customType":"ext","content":[{"type":"text","text":"hidden"}],"display":false,"details":{"k":1},"id":"e1f2a3b4","parentId":"d0e1f2a4","timestamp":"2024-12-03T14:22:00.000Z"}`,
		`{"type":"usage","id":"f2a3b4c5","parentId":"e1f2a3b4","timestamp":"2024-12-03T14:23:00.000Z","kind":"cache_warm","provider":"anthropic","model":"m","usage":{"input":1,"output":0,"cacheRead":0,"cacheWrite":0,"totalTokens":1,"cost":{"input":0,"output":0,"cacheRead":0,"cacheWrite":0,"total":0}},"note":"n"}`,
		`{"type":"label","id":"a3b4c5d6","parentId":"f2a3b4c5","timestamp":"2024-12-03T14:24:00.000Z","targetId":"a1b2c3d4"}`,
		`{"type":"label","id":"a3b4c5d7","parentId":"a3b4c5d6","timestamp":"2024-12-03T14:24:01.000Z","targetId":"a1b2c3d4","label":""}`,
		`{"type":"branch_summary","id":"b4c5d6e7","parentId":null,"timestamp":"2024-12-03T14:25:00.000Z","fromId":"root","summary":"s"}`,
		`{"type":"session_info","id":"c5d6e7f8","parentId":"b4c5d6e7","timestamp":"2024-12-03T14:26:00.000Z","name":""}`,
		`{"type":"message","id":"d6e7f8a9","parentId":"c5d6e7f8","timestamp":"2024-12-03T14:27:00.000Z","message":{"role":"bashExecution","command":"sleep 9","output":"","exitCode":null,"cancelled":true,"truncated":false,"timestamp":1733235220000,"excludeFromContext":false}}`,
		`{"type":"message","id":"e7f8a9b0","parentId":"d6e7f8a9","timestamp":"2024-12-03T14:28:00.000Z","message":{"role":"custom","customType":"ext","content":"c","display":false,"timestamp":1733235280000}}`,
	}
	for i, line := range lines {
		var v any = &Entry{}
		if i == 0 {
			v = &Header{}
		}
		if err := json.Unmarshal([]byte(line), v); err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
		out, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
		want, got := jsontext.Value(line), jsontext.Value(out)
		if err := want.Canonicalize(); err != nil {
			t.Fatal(err)
		}
		if err := got.Canonicalize(); err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("line %d changed:\n got %s\nwant %s", i, got, want)
		}
	}
}

func TestBlocksAndText(t *testing.T) {
	m := Message{Role: RoleUser, Content: jsontext.Value(`"hello there"`)}
	if bs, err := m.Blocks(); err != nil || len(bs) != 1 || bs[0].Text != "hello there" || m.Text() != "hello there" {
		t.Fatalf("string content: %+v, %v", bs, err)
	}
	if err := m.SetBlocks([]Block{{Type: BlockText, Text: "a"}, {Type: BlockImage, Data: "AA==", MimeType: "image/png"}, {Type: BlockText, Text: "b"}}); err != nil {
		t.Fatal(err)
	}
	if m.Text() != "a b" {
		t.Fatalf("text %q", m.Text())
	}
}

func TestPath(t *testing.T) {
	p := func(s string) *string { return &s }
	entries := []Entry{
		{Type: TypeMessage, ID: "a"},
		{Type: TypeMessage, ID: "b", ParentID: p("a")},
		{Type: TypeMessage, ID: "c", ParentID: p("b")},
		{Type: TypeMessage, ID: "d", ParentID: p("a")}, // a branch from a: the leaf
	}
	var ids []string
	for _, e := range Path(entries) {
		ids = append(ids, e.ID)
	}
	if strings.Join(ids, ",") != "a,d" {
		t.Fatalf("path %v, want a,d", ids)
	}
	if Path(nil) != nil {
		t.Fatal("empty path")
	}
	// A missing parent ends the path; a cycle does not loop.
	cyc := []Entry{{Type: TypeMessage, ID: "x", ParentID: p("y")}, {Type: TypeMessage, ID: "y", ParentID: p("x")}}
	if got := Path(cyc); len(got) != 2 {
		t.Fatalf("cycle path %+v", got)
	}
}

func TestIDs(t *testing.T) {
	for id, ok := range map[ID]bool{"019a-ok": true, "a": true, "a.b_c-1": true, "": false, "-a": false, "a-": false, "a/b": false} {
		if ValidID(id) != ok {
			t.Errorf("ValidID(%q) = %v", id, !ok)
		}
	}
	taken := map[string]bool{}
	for range 50 {
		id := NewEntryID(func(s string) bool { return taken[s] })
		if len(id) != 8 || taken[id] {
			t.Fatalf("entry id %q", id)
		}
		taken[id] = true
	}
	if NewEntryID(func(string) bool { return true }) == "" {
		t.Fatal("no fallback id")
	}
	if got := Timestamp(time.Date(2024, 12, 3, 14, 0, 1, 5e6, time.UTC)); got != "2024-12-03T14:00:01.005Z" {
		t.Fatalf("timestamp %q", got)
	}
}

// A custom entry's members are typed: gobble's own reads back as written,
// and another extension's, with a member gobble does not know, round-trips.
func TestCustomEntry(t *testing.T) {
	todo := Entry{Type: TypeCustom, ID: "c1", CustomType: CustomTodo, Data: jsontext.Value(`{"todos":[{"content":"a","status":"pending"}]}`)}
	b, err := json.Marshal(todo)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"type":"custom","id":"c1","parentId":null,"customType":"gobble.todo","data":{"todos":[{"content":"a","status":"pending"}]}}`
	if string(b) != want {
		t.Fatalf("encoded\n got %s\nwant %s", b, want)
	}
	var back Entry
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Type != TypeCustom || back.CustomType != CustomTodo || string(back.Data) != string(todo.Data) || len(back.Unknown) != 0 {
		t.Fatalf("read back %+v", back)
	}
	const other = `{"type":"custom","id":"c2","parentId":"c1","customType":"my-extension","data":{"count":42},"extra":true}`
	var e Entry
	if err := json.Unmarshal([]byte(other), &e); err != nil {
		t.Fatal(err)
	}
	if e.CustomType != "my-extension" || string(e.Data) != `{"count":42}` {
		t.Fatalf("decoded %+v", e)
	}
	out, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != other {
		t.Fatalf("round trip\n got %s\nwant %s", out, other)
	}
}

func TestMemoryStore(t *testing.T) {
	s := NewMemoryStore()
	ctx := t.Context()
	l, err := s.Create(ctx, Header{Type: TypeHeader, Version: Version, ID: "s1", Timestamp: "2024-12-03T14:00:00.000Z", Cwd: "/w"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, Header{ID: "s1"}); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate create: %v", err)
	}
	if _, err := s.Open(ctx, "s1"); !errors.As(err, new(*ErrLocked)) {
		t.Fatalf("second writer: %v", err)
	}
	user := &Message{Role: RoleUser, Content: jsontext.Value(`"hi"`), Timestamp: 1733234401000}
	if err := l.Append(ctx, Entry{Type: TypeMessage, ID: "e1", Message: user}); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	var got []Summary
	for sum, err := range s.List(ctx, Filter{Cwd: "/w"}) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, sum)
	}
	if len(got) != 1 || got[0].First != "hi" || got[0].Messages != 1 {
		t.Fatalf("list %+v", got)
	}
	if _, err := s.Open(ctx, "s1"); err != nil {
		t.Fatalf("reopen after close: %v", err)
	}
	if _, _, _, err := s.Read(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("read unknown: %v", err)
	}
}

// SetText writes one text block, with U+FFFD for each invalid byte, and
// cannot fail (0039-MADR D7; Confirmation 15).
func TestSetTextInvalidUTF8(t *testing.T) {
	var m Message
	m.SetText("caf\xe9")
	bs, err := m.Blocks()
	if err != nil || len(bs) != 1 || bs[0].Type != BlockText || bs[0].Text != "caf\uFFFD" {
		t.Fatalf("content %q: blocks %+v, %v", m.Content, bs, err)
	}
}

// SetText writes what SetBlocks writes for one text block, byte for byte.
func TestSetTextMatchesSetBlocks(t *testing.T) {
	for _, s := range []string{"", "plain", "a<b&c>d", "tab\tnl\n", "caf\xe9", "\u2028"} {
		var a, b Message
		a.SetText(s)
		if err := b.SetBlocks([]Block{{Type: BlockText, Text: s}}); err != nil {
			t.Fatal(err)
		}
		if string(a.Content) != string(b.Content) {
			t.Errorf("SetText(%q) = %s, SetBlocks = %s", s, a.Content, b.Content)
		}
	}
}

// SetBlocks reports a block holding arguments that are not JSON, rather
// than writing an empty content (0039-MADR D7; Confirmation 4).
func TestSetBlocksBadJSON(t *testing.T) {
	var m Message
	err := m.SetBlocks([]Block{{Type: BlockToolCall, ID: "c1", Name: "read", Arguments: jsontext.Value("{")}})
	if err == nil {
		t.Fatalf("SetBlocks of bad arguments = %q, want an error", m.Content)
	}
}

// Entry.Replacement and Message.ExitCode keep null apart from absent:
// absent is left out and read back absent, null is kept, and a value
// round-trips (0039-MADR D21).
func TestRawNullAndAbsent(t *testing.T) {
	replacement := func(e Entry) jsontext.Value { return e.Replacement }
	exitCode := func(e Entry) jsontext.Value { return e.Message.ExitCode }
	for _, c := range []struct {
		line  string
		field func(Entry) jsontext.Value
		want  string // "" is absent
	}{
		{`{"type":"context_edit","parentId":null,"targetId":"a"}`, replacement, ""},
		{`{"type":"context_edit","parentId":null,"targetId":"a","replacement":null}`, replacement, "null"},
		{`{"type":"context_edit","parentId":null,"targetId":"a","replacement":{"content":[{"type":"text","text":"x"}]}}`, replacement, `{"content":[{"type":"text","text":"x"}]}`},
		{`{"type":"message","parentId":null,"message":{"role":"bashExecution","command":"ls"}}`, exitCode, ""},
		{`{"type":"message","parentId":null,"message":{"role":"bashExecution","command":"ls","exitCode":null}}`, exitCode, "null"},
		{`{"type":"message","parentId":null,"message":{"role":"bashExecution","command":"ls","exitCode":7}}`, exitCode, "7"},
		{`{"type":"message","parentId":null,"message":{"role":"bashExecution","command":"ls","exitCode":{"signal":"TERM"}}}`, exitCode, `{"signal":"TERM"}`},
	} {
		var e Entry
		if err := jsonx.Unmarshal([]byte(c.line), &e); err != nil {
			t.Fatalf("%s: %v", c.line, err)
		}
		if got := c.field(e); string(got) != c.want || (c.want == "" && len(got) != 0) {
			t.Errorf("%s: read %q, want %q", c.line, got, c.want)
		}
		b, err := jsonx.Marshal(e)
		if err != nil || string(b) != c.line {
			t.Errorf("%s: written %s, %v", c.line, b, err)
		}
	}
}

// An empty non-nil raw value is not JSON: planted on a field it fails the
// marshal, and through jsonx.From it is absent and left out (0039-MADR D20,
// D21).
func TestEmptyRawValue(t *testing.T) {
	for _, c := range []struct {
		entry func(jsontext.Value) Entry
		want  string
	}{
		{func(v jsontext.Value) Entry { return Entry{Type: "context_edit", TargetID: "a", Replacement: v} },
			`{"type":"context_edit","parentId":null,"targetId":"a"}`},
		{func(v jsontext.Value) Entry {
			return Entry{Type: "message", Message: &Message{Role: "bashExecution", Command: "ls", ExitCode: v}}
		}, `{"type":"message","parentId":null,"message":{"role":"bashExecution","command":"ls"}}`},
	} {
		if b, err := jsonx.Marshal(c.entry(jsontext.Value{})); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("an empty value planted: wrote %s, %v; want the EOF error", b, err)
		}
		if b, err := jsonx.Marshal(c.entry(jsonx.From([]byte{}))); err != nil || string(b) != c.want {
			t.Errorf("through jsonx.From: wrote %s, %v; want %s", b, err, c.want)
		}
	}
}
