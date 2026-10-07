package acpserver

import (
	"encoding/json/v2"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// big is a message of n tokens by Pi's estimate.
func big(n int) string { return strings.Repeat("x", 4*n) }

// compactAgent is an agent over store with a small KeepRecentTokens, so a
// test session can be compacted.
func compactAgent(t *testing.T, store session.Store, p llm.Provider, client *acptest.Client) *acptest.Conn {
	t.Helper()
	a := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func() (llm.Provider, error) { return p, nil },
		Store: store, Compaction: compaction.Settings{ReserveTokens: 16384, KeepRecentTokens: 150}})
	c := acptest.Connect(t, a, client)
	a.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c
}

// requestTexts are the texts of a request's messages after the system
// prompt, in order.
func requestTexts(req *llm.Request) []string {
	var out []string
	for _, m := range req.Messages {
		if m.Role == llm.RoleSystem {
			continue
		}
		for _, c := range m.Content {
			if c.Type == llm.ContentText {
				out = append(out, c.Text)
			}
		}
	}
	return out
}

// /compact summarises the history before the kept range, streams its
// progress and the summary, sends the new estimate, and the next prompt
// sees the summary and the kept messages only (0002-PLAN Phase 6 Accept).
func TestCompactShortensContext(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text(big(100)), llmtest.Text(big(100)), llmtest.Text("## Goal\nSUMMARY"), llmtest.Text("after"))
	client := &acptest.Client{}
	c := compactAgent(t, session.NewMemoryStore(), model, client)
	s, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	u1, u2 := "one "+big(100), "two "+big(100)
	for _, p := range []string{u1, u2} {
		if _, err := prompt(t, c, s.SessionId, p); err != nil {
			t.Fatal(err)
		}
	}
	n := len(client.Updates())
	resp, err := prompt(t, c, s.SessionId, "/compact keep the API")
	if err != nil || resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("compact = %+v, %v", resp, err)
	}
	after := client.Updates()[n:]
	text := agentText(after)
	if !strings.HasPrefix(text, "Compacting the conversation…") || !strings.Contains(text, "Compacted from ") || !strings.Contains(text, "## Goal\nSUMMARY") {
		t.Fatalf("compact text %q", text)
	}
	if last := after[len(after)-1].Update.UsageUpdate; last == nil || last.Used <= 0 {
		t.Fatal("compact did not end with a usage_update")
	}
	if p := model.Requests()[2].Messages[1].Content[0].Text; !strings.Contains(p, "Additional focus: keep the API") || !strings.Contains(p, "[User]: one ") {
		t.Fatalf("summary prompt %q", p)
	}
	if _, err := prompt(t, c, s.SessionId, "three"); err != nil {
		t.Fatal(err)
	}
	got := requestTexts(model.Requests()[3])
	if len(got) != 4 || got[0] != compaction.SummaryText("## Goal\nSUMMARY") || got[1] != u2 || got[3] != "three" {
		t.Fatalf("context after compact: %d messages, first %.80q", len(got), got)
	}
}

// A compaction written by gobble, or by Pi in Pi's shape, is the summary
// a loaded session starts from, and load replays it.
func TestCompactionLoads(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	store := jsonl.New(dir, jsonl.Options{})
	h := session.Header{Type: session.TypeHeader, Version: session.Version, ID: "pi-made", Timestamp: "2026-10-06T10:00:00.000Z", Cwd: cwd}
	log, err := store.Create(t.Context(), h)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		`{"type":"message","id":"a0000001","parentId":null,"timestamp":"2026-10-06T10:00:01.000Z","message":{"role":"user","content":"old work","timestamp":1}}`,
		`{"type":"message","id":"a0000002","parentId":"a0000001","timestamp":"2026-10-06T10:00:02.000Z","message":{"role":"user","content":"kept work","timestamp":2}}`,
		`{"type":"compaction","id":"a0000003","parentId":"a0000002","timestamp":"2026-10-06T10:00:03.000Z","summary":"PI SUMMARY","firstKeptEntryId":"a0000002","tokensBefore":42}`,
	} {
		var e session.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		if err := log.Append(t.Context(), e); err != nil {
			t.Fatal(err)
		}
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	model := llmtest.NewScript(llmtest.Text("ok"))
	client := &acptest.Client{}
	c := compactAgent(t, store, model, client)
	if _, err := c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: "pi-made", Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if replay := agentText(client.Updates()); !strings.HasPrefix(replay, "Compacted from 42 tokens:\n\nPI SUMMARY") {
		t.Fatalf("replay %q", replay)
	}
	if _, err := prompt(t, c, "pi-made", "next"); err != nil {
		t.Fatal(err)
	}
	got := requestTexts(model.Requests()[0])
	if len(got) != 3 || got[0] != compaction.SummaryText("PI SUMMARY") || got[1] != "kept work" || got[2] != "next" {
		t.Fatalf("context %q", got)
	}
}
