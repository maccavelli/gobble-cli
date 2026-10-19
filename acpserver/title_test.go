package acpserver

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"strings"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// gated is a provider whose streams wait for open, or for their context.
type gated struct {
	llm.Provider
	open chan struct{}
}

func (g gated) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	return func(yield func(llm.Event, error) bool) {
		select {
		case <-g.open:
		case <-ctx.Done():
			yield(nil, ctx.Err())
			return
		}
		for ev, err := range g.Provider.Stream(ctx, req) {
			if !yield(ev, err) {
				return
			}
		}
	}
}

// titleHarness is an agent whose small model is small, over the store in
// dir.
type titleHarness struct {
	c      *acptest.Conn
	ag     *Agent
	client *acptest.Client
	main   *llmtest.Script
	logs   *syncBuffer
}

func newTitleHarness(t *testing.T, dir string, main *llmtest.Script, small llm.Provider) *titleHarness {
	t.Helper()
	h := &titleHarness{client: &acptest.Client{}, main: main, logs: &syncBuffer{}}
	var title TitleOptions
	if small != nil {
		title = TitleOptions{Provider: func(string, llm.Retry) (llm.Provider, error) { return small, nil }, ProviderID: "small", Model: "s1"}
	}
	h.ag = New(Options{Version: "1.2.3", Store: jsonl.New(dir, jsonl.Options{}), Models: func() ModelChoice { return testModels },
		Provider: func(string, llm.Retry) (llm.Provider, error) { return main, nil }, Title: title,
		Logger: slog.New(slog.NewTextHandler(h.logs, nil))})
	h.c = acptest.Connect(t, h.ag, h.client)
	h.ag.SetConnection(h.c.Agent)
	t.Cleanup(func() {
		if err := h.ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, h.c.Client)
	return h
}

// waitFor polls cond for up to 5 s.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); !cond(); {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 5 s")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// titled reports whether a session_info_update gave title.
func (h *titleHarness) titled(title string) bool {
	for _, u := range h.client.Updates() {
		if si := u.Update.SessionInfoUpdate; si != nil && si.Title == acp.PatchValue(title) {
			return true
		}
	}
	return false
}

func (h *titleHarness) usageEntries(id acp.SessionId) []session.Entry {
	h.ag.mu.Lock()
	s := h.ag.sessions[id]
	h.ag.mu.Unlock()
	s.appendMu.Lock()
	defer s.appendMu.Unlock()
	var out []session.Entry
	for _, e := range s.tree.Entries() {
		if e.Type == session.TypeUsage {
			out = append(out, e)
		}
	}
	return out
}

// A new session's first prompt names it: the small model is asked, with
// gobble's prompt and the message in its block, at its lowest level and
// a small cap; the reply, cleaned, is the session's name, reported, and
// its usage an entry of its own, counted by /usage. A later prompt asks
// nothing (0005-PLAN F5-5 decisions 4 to 8).
func TestTitleFirstPrompt(t *testing.T) {
	small := llmtest.NewScript(llmtest.Text("<think>hm</think>\n\n  \"Fix the flaky login test.\"  \nmore"))
	h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok"), llmtest.Text("ok again")), small)
	id := newSession(t, h.c, t.TempDir(), nil).SessionId
	first := "Fix the flaky login test in auth_test.go " + strings.Repeat("x", 2500)
	if _, err := prompt(t, h.c, id, first); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return h.titled("Fix the flaky login test") })
	req := small.Requests()[0]
	llmtest.AssertSystem(t, req, titlePrompt)
	if got := req.Messages[1].Content[0].Text; got != "<user_message>\n"+first[:2000]+"\n</user_message>" ||
		req.Model != "s1" || req.Thinking != "off" || req.MaxTokens != 128 || len(req.Tools) != 0 {
		t.Fatalf("title request %+v", req)
	}
	if es := h.usageEntries(id); len(es) != 1 || es[0].Kind != usageKindTitle || es[0].Provider != "small" || es[0].Model != "s1" ||
		es[0].Usage == nil || es[0].Usage.Input != 10 {
		t.Fatalf("usage entries %+v", es)
	}
	if got := mustExt(t, h.c, "_gobble/get_state", map[string]any{"sessionId": id})["sessionName"]; got != "Fix the flaky login test" {
		t.Fatalf("name %v", got)
	}
	if _, err := prompt(t, h.c, id, "and again"); err != nil {
		t.Fatal(err)
	}
	if usage, _ := slash(t, h.c, h.client, id, "/usage"); !strings.Contains(usage, "Input: 30\n") {
		t.Fatalf("/usage does not count the title's 10 input tokens with the two turns' 20:\n%s", usage)
	}
	if n := len(small.Requests()); n != 1 {
		t.Fatalf("%d title requests, want 1", n)
	}
}

// No title without a small model, for a session named at its start, for a
// resumed session with a conversation, or for a first prompt that is a
// command; the first prompt after such a command names it (decision 4).
func TestTitleNotWanted(t *testing.T) {
	plainDir, plainCwd := t.TempDir(), t.TempDir()
	h := newTitleHarness(t, plainDir, llmtest.NewScript(llmtest.Text("ok")), nil)
	id := newSession(t, h.c, plainCwd, nil).SessionId
	if _, err := prompt(t, h.c, id, "hello"); err != nil {
		t.Fatal(err)
	}
	if h.ag.sessions[id].titleCancel != nil {
		t.Fatal("a title started with no small model")
	}
	if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	// Unnamed, with a conversation, resumed where a small model is set.
	never := llmtest.NewScript(llmtest.Text("Never"))
	h3 := newTitleHarness(t, plainDir, llmtest.NewScript(llmtest.Text("ok")), never)
	if _, err := h3.c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: plainCwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, h3.c, id, "more"); err != nil {
		t.Fatal(err)
	}
	if h3.ag.sessions[id].titleCancel != nil || len(never.Requests()) != 0 {
		t.Fatal("an unnamed resumed session with a conversation was titled")
	}

	small := llmtest.NewScript(llmtest.Text("Named by the model"))
	dir, cwd := t.TempDir(), t.TempDir()
	h = newTitleHarness(t, dir, llmtest.NewScript(llmtest.Text("ok"), llmtest.Text("ok"), llmtest.Text("ok")), small)
	named := newSession(t, h.c, cwd, map[string]any{"gobble": map[string]any{"name": "Mine"}}).SessionId
	if _, err := prompt(t, h.c, named, "hello"); err != nil {
		t.Fatal(err)
	}
	other := newSession(t, h.c, cwd, nil).SessionId
	if _, err := prompt(t, h.c, other, "/help"); err != nil {
		t.Fatal(err)
	}
	if n := len(small.Requests()); n != 0 {
		t.Fatalf("%d title requests for a named session and a command", n)
	}
	if _, err := prompt(t, h.c, other, "now the work"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return h.titled("Named by the model") })
	if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: named}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: other, Cwd: cwd, McpServers: []acp.McpServer{}}); err == nil {
		t.Fatal("loading an open session")
	}
	if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: other}); err != nil {
		t.Fatal(err)
	}
	h2 := newTitleHarness(t, dir, llmtest.NewScript(llmtest.Text("ok")), llmtest.NewScript(llmtest.Text("Never")))
	if _, err := h2.c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: named, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, h2.c, named, "more"); err != nil {
		t.Fatal(err)
	}
	if h2.ag.sessions[named].titleCancel != nil {
		t.Fatal("a resumed session with a conversation was titled")
	}
}

// One title a session: a first turn that fails before writing anything
// leaves the session without a conversation, and with its title failed it
// has no name either; the next prompt does not try again (decision 4).
func TestTitleOnce(t *testing.T) {
	small := llmtest.NewScript(llmtest.Turn{Err: errors.New("the small model failed")}, llmtest.Text("Twice"))
	client := &acptest.Client{}
	ag := New(Options{Version: "1.2.3", Store: jsonl.New(t.TempDir(), jsonl.Options{}), Models: func() ModelChoice { return testModels },
		Provider: func(string, llm.Retry) (llm.Provider, error) { return nil, errors.New("no provider") },
		Title:    TitleOptions{Provider: func(string, llm.Retry) (llm.Provider, error) { return small, nil }, ProviderID: "small", Model: "s1"}})
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	s := ag.sessions[id]
	var first chan struct{}
	for i, text := range []string{"first", "second"} {
		if _, err := prompt(t, c, id, text); err == nil {
			t.Fatalf("%s: the turn did not fail", text)
		}
		ag.mu.Lock()
		done := s.titleDone
		ag.mu.Unlock()
		if i == 0 {
			first = done
		} else if done != first {
			t.Fatal("the second prompt started a second title")
		}
	}
	<-first
	if n := len(small.Requests()); n != 1 {
		t.Fatalf("%d title requests, want 1", n)
	}
}

// A name given while the title runs wins; the turn does not wait for the
// title; and closing the session stops a title still running, writing
// nothing (decisions 4, 7 and 9).
func TestTitleRace(t *testing.T) {
	open := make(chan struct{})
	small := llmtest.NewScript(llmtest.Text("The model's title"))
	h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), gated{Provider: small, open: open})
	id := newSession(t, h.c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, h.c, id, "do the work"); err != nil { // returns while the title waits
		t.Fatal(err)
	}
	if _, err := prompt(t, h.c, id, "/name Mine"); err != nil {
		t.Fatal(err)
	}
	close(open)
	waitFor(t, func() bool { return len(h.usageEntries(id)) == 1 })
	h.ag.closeTitle(h.ag.sessions[id])
	if h.titled("The model's title") || mustExt(t, h.c, "_gobble/get_state", map[string]any{"sessionId": id})["sessionName"] != "Mine" {
		t.Fatal("the title replaced the user's name")
	}

	never := make(chan struct{})
	h = newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), gated{Provider: llmtest.NewScript(llmtest.Text("T")), open: never})
	id = newSession(t, h.c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, h.c, id, "do the work"); err != nil {
		t.Fatal(err)
	}
	s := h.ag.sessions[id]
	start := time.Now()
	if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.titleDone:
	default:
		t.Fatal("the title still runs after session/close")
	}
	if time.Since(start) > closeWait || h.titled("T") || strings.Contains(h.logs.String(), "was not named") {
		t.Fatalf("closing: %v, logs %q", time.Since(start), h.logs.String())
	}
}

// A reply with no title, or a failed call, names nothing and is logged
// (decisions 4 and 6).
func TestTitleFailure(t *testing.T) {
	for _, c := range []struct {
		name  string
		turn  llmtest.Turn
		usage int
		why   string
	}{
		{"empty", llmtest.Text("<think>only thinking</think>\n  \n"), 1, "the small model gave no title"},
		{"error", llmtest.Turn{Err: errors.New("boom")}, 0, "boom"},
	} {
		h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), llmtest.NewScript(c.turn))
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		if _, err := prompt(t, h.c, id, "do the work"); err != nil {
			t.Fatal(err)
		}
		waitFor(t, func() bool { return strings.Contains(h.logs.String(), "title: the session was not named") })
		if !strings.Contains(h.logs.String(), c.why) || len(h.usageEntries(id)) != c.usage ||
			mustExt(t, h.c, "_gobble/get_state", map[string]any{"sessionId": id})["sessionName"] != nil {
			t.Errorf("%s: logs %q, %d usage entries", c.name, h.logs.String(), len(h.usageEntries(id)))
		}
	}
}

// A reply's first line, without a leading <think> block, quotes, a final
// mark or extra spaces, cut to 60 characters (decision 6).
func TestCleanTitle(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Fix login test", "Fix login test"},
		{"<think>\nreasoning\n</think>\nFix login test", "Fix login test"},
		{"\n\n  “Fix   login\ttest?”  \nsecond line", "Fix login test"},
		{"'Rename the parser'.", "Rename the parser"},
		{"Wow!!!", "Wow"},
		{strings.Repeat("abcdefghij", 8), strings.Repeat("abcdefghij", 6)},
		{"   \n", ""},
	} {
		if got := cleanTitle(c.in); got != c.want {
			t.Errorf("cleanTitle(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
