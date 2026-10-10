package acpserver

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// treeAgent is an agent over a store holding one session file, body, on
// model p, with the session resumed. It returns the connection, the agent,
// the store and the session's id.
func treeAgent(t *testing.T, body []byte, p llm.Provider) (*acptest.Conn, *Agent, *jsonl.Store, acp.SessionId) {
	t.Helper()
	var h session.Header
	if err := json.Unmarshal(bytes.SplitN(body, []byte("\n"), 2)[0], &h); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, jsonl.EncodeCwd(h.Cwd), jsonl.FileName(h))
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, body, 0o600); err != nil {
		t.Fatal(err)
	}
	store := jsonl.New(dir, jsonl.Options{})
	ag := New(Options{Version: "1.2.3", Provider: func() (llm.Provider, error) { return p, nil },
		Models: func() ModelChoice { return piModels }, Store: store})
	c := acptest.Connect(t, ag, &acptest.Client{})
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	if _, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: acp.SessionId(h.ID), Cwd: h.Cwd}); err != nil {
		t.Fatal(err)
	}
	return c, ag, store, acp.SessionId(h.ID)
}

func fixtureBody(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(piFixtures, name+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// labelsAtEnd are the label targets of a stored session, which must all
// come after its other entries.
func labelsAtEnd(t *testing.T, store *jsonl.Store, id string) []string {
	t.Helper()
	_, entries, _, err := store.Read(t.Context(), session.ID(id))
	if err != nil {
		t.Fatal(err)
	}
	var targets []string
	for i, e := range entries {
		if e.Type != session.TypeLabel {
			if len(targets) > 0 {
				t.Fatalf("entry %d, a %s, follows a label", i+1, e.Type)
			}
			continue
		}
		targets = append(targets, e.TargetID)
	}
	return targets
}

// /clone writes the path as Pi's createBranchedSession does: its labels
// after it, in Pi's order, each for a target on the path.
func TestCloneKeepsLabels(t *testing.T) {
	c, _, store, id := treeAgent(t, fixtureBody(t, "v3"), llmtest.NewScript())
	resp, err := prompt(t, c, id, "/clone")
	if err != nil {
		t.Fatal(err)
	}
	// v3.jsonl labels its second user message, its first, and its first
	// answer, cleared and set again: Pi's order is the second, the first,
	// then the answer.
	want := []string{"3f8cc253", "8815d46e", "9ba47962"}
	if got := labelsAtEnd(t, store, switchTo(t, resp)); !slices.Equal(got, want) {
		t.Fatalf("clone's labels %v, want %v", got, want)
	}
}

// /fork drops a label whose target is not on the path it copies.
func TestForkDropsOffPathLabels(t *testing.T) {
	c, _, store, id := treeAgent(t, fixtureBody(t, "v3"), llmtest.NewScript())
	resp, err := prompt(t, c, id, "/fork 3f8cc253") // before the second user message, which "second" labels
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"8815d46e", "9ba47962"}
	if got := labelsAtEnd(t, store, switchTo(t, resp)); !slices.Equal(got, want) {
		t.Fatalf("fork's labels %v, want %v", got, want)
	}
}

// A compaction whose first kept entry was a label keeps, in the copy, the
// entry after that label.
func TestCloneMovesFirstKeptPastLabel(t *testing.T) {
	lines := []string{
		`{"type":"session","version":3,"id":"lbl-1","timestamp":"2026-10-10T00:00:00.000Z","cwd":"/work/project"}`,
		`{"type":"message","id":"u1","parentId":null,"timestamp":"2026-10-10T00:00:01.000Z","message":{"role":"user","content":"first","timestamp":1791590401000}}`,
		`{"type":"label","id":"l1","parentId":"u1","timestamp":"2026-10-10T00:00:02.000Z","targetId":"u1","label":"start"}`,
		`{"type":"message","id":"a1","parentId":"l1","timestamp":"2026-10-10T00:00:03.000Z","message":{"role":"assistant","content":[{"type":"text","text":"ok"}],"stopReason":"stop","timestamp":1791590403000}}`,
		`{"type":"compaction","id":"c1","parentId":"a1","timestamp":"2026-10-10T00:00:04.000Z","summary":"s","firstKeptEntryId":"l1","tokensBefore":10}`,
		`{"type":"message","id":"u2","parentId":"c1","timestamp":"2026-10-10T00:00:05.000Z","message":{"role":"user","content":"second","timestamp":1791590405000}}`,
	}
	c, _, store, id := treeAgent(t, []byte(strings.Join(lines, "\n")+"\n"), llmtest.NewScript())
	resp, err := prompt(t, c, id, "/clone")
	if err != nil {
		t.Fatal(err)
	}
	_, entries, _, err := store.Read(t.Context(), session.ID(switchTo(t, resp)))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Type == session.TypeCompaction && e.FirstKeptEntryID != "a1" {
			t.Fatalf("the copy's compaction keeps from %q, want a1", e.FirstKeptEntryID)
		}
		if e.ID == "a1" && (e.ParentID == nil || *e.ParentID != "u1") {
			t.Fatalf("a1's parent %v, want u1 once the label is gone", e.ParentID)
		}
	}
}

// Every path is the tree's: once its leaf moves and the session's history
// is rebuilt from it, as a caller that moves it does (Pi's navigateTree
// refreshes its context; X1's /tree), the next prompt's history is the
// path to the new leaf, and the prompt is that leaf's child.
func TestPromptFollowsTheLeaf(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("from here"))
	c, ag, store, id := treeAgent(t, fixtureBody(t, "v3"), model)
	ag.mu.Lock()
	s := ag.sessions[id]
	ag.mu.Unlock()
	if err := s.tree.Branch("9ba47962"); err != nil { // the first answer
		t.Fatal(err)
	}
	ag.mu.Lock()
	s.history = s.state().history
	ag.mu.Unlock()
	if leaf := mustExt(t, c, "_gobble/get_entries", map[string]any{"sessionId": id})["leafId"]; leaf != "9ba47962" {
		t.Fatalf("get_entries' leaf %v, want the moved leaf", leaf)
	}
	if _, err := prompt(t, c, id, "again"); err != nil {
		t.Fatal(err)
	}
	llmtest.AssertTail(t, model.Requests()[0], llm.RoleUser, llm.RoleAssistant, llm.RoleTool, llm.RoleAssistant, llm.RoleUser)
	_, entries, _, err := store.Read(t.Context(), session.ID(id))
	if err != nil {
		t.Fatal(err)
	}
	var promptID string
	for _, e := range entries {
		if e.Message == nil {
			continue
		}
		switch e.Message.Text() {
		case "again":
			promptID = e.ID
			if e.ParentID == nil || *e.ParentID != "9ba47962" {
				t.Fatalf("the prompt's parent %v, want the moved leaf", e.ParentID)
			}
		case "from here":
			if e.ParentID == nil || *e.ParentID != promptID {
				t.Fatalf("the answer's parent %v, want the prompt %s", e.ParentID, promptID)
			}
		}
	}
	if promptID == "" {
		t.Fatal("the prompt was not written")
	}
	n := 0
	for _, m := range model.Requests()[0].Messages {
		if m.Role != llm.RoleSystem {
			n++
		}
	}
	if n != 5 { // the first prompt, the call, its result, the answer, and "again"
		t.Fatalf("the request has %d messages, want the path to the leaf's 4 and the prompt", n)
	}
}
