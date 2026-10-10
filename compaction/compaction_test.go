package compaction

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
)

// path builds a linear path of entries, each the child of the one before.
type path struct {
	entries []session.Entry
}

func (p *path) add(e session.Entry) *path {
	e.ID = fmt.Sprintf("e%07d", len(p.entries))
	if n := len(p.entries); n > 0 {
		e.ParentID = new(p.entries[n-1].ID)
	}
	if e.Type == "" {
		e.Type = session.TypeMessage
	}
	p.entries = append(p.entries, e)
	return p
}

func text(role, s string) session.Entry {
	m := &session.Message{Role: role}
	if err := m.SetBlocks([]session.Block{{Type: session.BlockText, Text: s}}); err != nil {
		panic(err)
	}
	return session.Entry{Message: m}
}

func call(name, args string) session.Entry {
	m := &session.Message{Role: session.RoleAssistant}
	if err := m.SetBlocks([]session.Block{{Type: session.BlockToolCall, ID: "c-" + name, Name: name, Arguments: jsontext.Value(args)}}); err != nil {
		panic(err)
	}
	return session.Entry{Message: m}
}

// chars is n characters: n/4 tokens.
func chars(n int) string { return strings.Repeat("x", n) }

func TestCutAtATurn(t *testing.T) {
	p := (&path{}).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400))).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 150})
	if err != nil {
		t.Fatal(err)
	}
	if prep.FirstKeptEntryID != p.entries[2].ID || prep.SplitTurn || len(prep.toSummarize) != 2 || len(prep.turnPrefix) != 0 {
		t.Fatalf("prep %+v", prep)
	}
	if prep.TokensBefore != 400 {
		t.Fatalf("tokens before %d, want 400", prep.TokensBefore)
	}
}

// One turn larger than the budget is cut at an assistant message, and its
// start is summarised as a turn prefix.
func TestSplitTurn(t *testing.T) {
	p := (&path{}).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400))).
		add(text(session.RoleToolResult, chars(400))).
		add(text(session.RoleAssistant, chars(400)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 150})
	if err != nil {
		t.Fatal(err)
	}
	if prep.FirstKeptEntryID != p.entries[3].ID || !prep.SplitTurn || len(prep.toSummarize) != 0 || len(prep.turnPrefix) != 3 {
		t.Fatalf("prep %+v", prep)
	}
}

func TestNothingAndAlready(t *testing.T) {
	small := (&path{}).add(text(session.RoleUser, "hi")).add(text(session.RoleAssistant, "hello"))
	if _, err := Prepare(small.entries, DefaultSettings); !errors.Is(err, ErrNothingToCompact) || err.Error() != "Nothing to compact (session too small)" {
		t.Fatalf("small: %v", err)
	}
	done := small.add(session.Entry{Type: session.TypeCompaction, Summary: "s"})
	if _, err := Prepare(done.entries, DefaultSettings); !errors.Is(err, ErrAlreadyCompacted) || err.Error() != "Already compacted" {
		t.Fatalf("compacted: %v", err)
	}
	if _, err := Prepare(nil, DefaultSettings); !errors.Is(err, ErrNothingToCompact) {
		t.Fatalf("empty: %v", err)
	}
}

// A cut never lands on a tool result, over seeded random sessions (0005-PLAN
// F5's property, moved here with the cut-point rules).
func TestNoCutAtAToolResult(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for n := range 500 {
		p := &path{}
		p.add(text(session.RoleUser, chars(1+r.IntN(800))))
		for range 3 + r.IntN(25) {
			switch r.IntN(4) {
			case 0:
				p.add(text(session.RoleUser, chars(1+r.IntN(800))))
			case 1:
				p.add(text(session.RoleAssistant, chars(1+r.IntN(800))))
			case 2:
				p.add(call("read", `{"path":"a.go"}`)).add(text(session.RoleToolResult, chars(1+r.IntN(4000))))
			default:
				p.add(session.Entry{Type: session.TypeModelChange, Provider: "p", ModelID: "m"})
			}
		}
		prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 1 + r.IntN(1500)})
		if err != nil {
			continue
		}
		for _, e := range p.entries {
			if e.ID == prep.FirstKeptEntryID && e.Message != nil && e.Message.Role == session.RoleToolResult {
				t.Fatalf("session %d: cut at tool result %s", n, e.ID)
			}
		}
	}
}

// Context with a compaction is its summary, the kept entries, and the
// entries after it.
func TestContext(t *testing.T) {
	p := (&path{}).
		add(text(session.RoleUser, "old")).
		add(text(session.RoleUser, "kept")).
		add(session.Entry{Type: session.TypeCompaction, Summary: "S"}).
		add(text(session.RoleUser, "new"))
	p.entries[2].FirstKeptEntryID = p.entries[1].ID
	got := Context(p.entries)
	if len(got) != 3 || got[0].Type != session.TypeCompaction || got[1].ID != p.entries[1].ID || got[2].ID != p.entries[3].ID {
		t.Fatalf("context %v", got)
	}
	if SummaryText("S") != "The conversation history before this point was compacted into the following summary:\n\n<summary>\nS\n</summary>" {
		t.Fatal("summary text")
	}
}

// The serialisation is Pi's serializeConversation, written out by hand
// from its code.
func TestSerialize(t *testing.T) {
	asst := &session.Message{Role: session.RoleAssistant}
	if err := asst.SetBlocks([]session.Block{
		{Type: session.BlockThinking, Thinking: "hm"},
		{Type: session.BlockText, Text: "a"},
		{Type: session.BlockText, Text: "b"},
		{Type: session.BlockToolCall, Name: "edit", Arguments: jsontext.Value(`{"path": "x.go", "n": 2}`)},
		{Type: session.BlockToolCall, Name: "bash", Arguments: jsontext.Value(`{"command":"ls"}`)},
	}); err != nil {
		t.Fatal(err)
	}
	long := chars(2005)
	msgs := []*session.Message{text(session.RoleUser, "hi").Message, asst, text(session.RoleToolResult, long).Message}
	want := "[User]: hi\n\n[Assistant thinking]: hm\n\n[Assistant]: a\nb\n\n" +
		`[Assistant tool calls]: edit(path="x.go", n=2); bash(command="ls")` +
		"\n\n[Tool result]: " + chars(2000) + "\n\n[... 5 more characters truncated]"
	if got := serialize(msgs); got != want {
		t.Fatalf("serialize\n%q\nwant\n%q", got, want)
	}
}

func TestCompactRequestAndFiles(t *testing.T) {
	p := (&path{}).
		add(text(session.RoleUser, chars(400))).
		add(call("read", `{"path":"r.go"}`)).
		add(text(session.RoleToolResult, "ok")).
		add(call("edit", `{"path":"w.go"}`)).
		add(text(session.RoleToolResult, "ok")).
		add(text(session.RoleUser, chars(800)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 150})
	if err != nil {
		t.Fatal(err)
	}
	s := llmtest.NewScript(llmtest.Text("## Goal\nG"))
	res, err := Compact(t.Context(), prep, Model{Provider: s, Model: "m1"}, "the API")
	if err != nil {
		t.Fatal(err)
	}
	want := "## Goal\nG\n\n<read-files>\nr.go\n</read-files>\n\n<modified-files>\nw.go\n</modified-files>"
	if res.Summary != want || res.FirstKeptEntryID != p.entries[5].ID {
		t.Fatalf("result %+v", res)
	}
	req := s.Requests()[0]
	llmtest.AssertSystem(t, req, "You are a context summarization assistant.")
	prompt := req.Messages[len(req.Messages)-1].Content[0].Text
	if !strings.HasPrefix(prompt, "<conversation>\n[User]: ") || !strings.HasSuffix(prompt, "Additional focus: the API") ||
		!strings.Contains(prompt, "Use this EXACT format:") || strings.Contains(prompt, "<previous-summary>") || req.Model != "m1" {
		t.Fatalf("prompt %q", prompt)
	}
	b, err := json.Marshal(res.Details)
	if err != nil || string(b) != `{"readFiles":["r.go"],"modifiedFiles":["w.go"]}` {
		t.Fatalf("details %s %v", b, err)
	}
}

// A second compaction updates the first one's summary and carries its
// file lists.
func TestIterativeUpdate(t *testing.T) {
	p := (&path{}).
		add(text(session.RoleUser, chars(400))).
		add(session.Entry{Type: session.TypeCompaction, Summary: "PREV", Details: jsontext.Value(`{"readFiles":["old.go"],"modifiedFiles":[]}`)}).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleUser, chars(400)))
	p.entries[1].FirstKeptEntryID = p.entries[1].ID // keeps nothing before it
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 150})
	if err != nil {
		t.Fatal(err)
	}
	s := llmtest.NewScript(llmtest.Text("NEW"))
	res, err := Compact(t.Context(), prep, Model{Provider: s}, "")
	if err != nil {
		t.Fatal(err)
	}
	prompt := s.Requests()[0].Messages[1].Content[0].Text
	if !strings.Contains(prompt, "<previous-summary>\nPREV\n</previous-summary>") || !strings.Contains(prompt, "incorporate into the existing summary") {
		t.Fatalf("prompt %q", prompt)
	}
	if res.Summary != "NEW\n\n<read-files>\nold.go\n</read-files>" {
		t.Fatalf("summary %q", res.Summary)
	}
}

func TestSplitTurnMerge(t *testing.T) {
	p := (&path{}).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400))).
		add(text(session.RoleToolResult, chars(400))).
		add(text(session.RoleAssistant, chars(400)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 150})
	if err != nil {
		t.Fatal(err)
	}
	s := llmtest.NewScript(llmtest.Text("PREFIX"))
	res, err := Compact(t.Context(), prep, Model{Provider: s}, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary != "No prior history.\n\n---\n\n**Turn Context (split turn):**\n\nPREFIX" {
		t.Fatalf("summary %q", res.Summary)
	}
	if got := s.Requests()[0].Messages[1].Content[0].Text; !strings.HasPrefix(got, "# Conversation\n[User]: ") || !strings.Contains(got, "# Instructions\nThe messages above are earlier context") {
		t.Fatalf("prefix prompt %q", got)
	}
}

func TestSummaryFailures(t *testing.T) {
	p := (&path{}).add(text(session.RoleUser, chars(400))).add(text(session.RoleUser, chars(400)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 50})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		turn llmtest.Turn
		want string
	}{
		{llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "part"}, llm.Done{StopReason: llm.StopMaxTokens}}},
			"Summarization failed: generation hit the token cap and the summary is incomplete"},
		{llmtest.ToolCall("x", "read", `{}`), "Summarization attempted to call a tool"},
		{llmtest.Turn{Err: llm.ErrAuth}, "Summarization failed: llm: authentication failed"},
	}
	for _, c := range cases {
		_, err := Compact(t.Context(), prep, Model{Provider: llmtest.NewScript(c.turn)}, "")
		if err == nil || err.Error() != c.want {
			t.Errorf("err %v, want %q", err, c.want)
		}
	}
}
