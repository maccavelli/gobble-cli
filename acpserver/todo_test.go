package acpserver

import (
	"encoding/json/v2"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/tool"
)

const todoArgs = `{"todos":[` +
	`{"content":"read the code","status":"completed","priority":"high"},` +
	`{"content":"write the tool","status":"in_progress"},` +
	`{"content":"test it","status":"pending","priority":"low"}]}`

var todoWant = []acp.PlanEntry{
	{Content: "read the code", Status: acp.PlanEntryStatusCompleted, Priority: acp.PlanEntryPriorityHigh},
	{Content: "write the tool", Status: acp.PlanEntryStatusInProgress, Priority: acp.PlanEntryPriorityMedium},
	{Content: "test it", Status: acp.PlanEntryStatusPending, Priority: acp.PlanEntryPriorityLow},
}

// samePlan reports whether two plans have the same entries in order.
func samePlan(a, b []acp.PlanEntry) bool {
	return slices.EqualFunc(a, b, func(x, y acp.PlanEntry) bool {
		return x.Content == y.Content && x.Status == y.Status && x.Priority == y.Priority
	})
}

// plans are the plan updates among us, and the index each one has.
func plans(us []acp.SessionNotification) (out [][]acp.PlanEntry, at []int) {
	for i, n := range us {
		if p := n.Update.Plan; p != nil {
			out, at = append(out, p.Entries), append(at, i)
		}
	}
	return out, at
}

// todoEntries are the session file's gobble.todo entries' lists.
func todoEntries(t *testing.T, dir string) [][]tool.PlanItem {
	t.Helper()
	files := sessionFiles(t, dir)
	if len(files) != 1 {
		t.Fatalf("%d session files", len(files))
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var out [][]tool.PlanItem
	for line := range strings.Lines(string(b)) {
		var e session.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // the header
		}
		if todos, ok := todosOf(e); ok {
			out = append(out, todos)
		}
	}
	return out
}

// A todo call sends one plan update, after the call's terminal update, with
// the items in order; the session records it as a gobble.todo entry;
// /todos prints it; and session/load sends it again after the history
// (0005-PLAN F1d-1).
func TestTodoPlan(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	model := llmtest.NewScript(llmtest.ToolCall("t1", "todo", todoArgs), llmtest.Text("planned"))
	client := &acptest.Client{}
	c, _ := sessionAgent(t, dir, model, client)
	id := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, id, "plan it"); err != nil {
		t.Fatal(err)
	}
	us := client.Updates()
	got, at := plans(us)
	if len(got) != 1 || !samePlan(got[0], todoWant) {
		t.Fatalf("plan updates %+v, want one with %+v", got, todoWant)
	}
	end := slices.IndexFunc(us, func(n acp.SessionNotification) bool {
		u := n.Update.ToolCallUpdate
		return u != nil && u.ToolCallId == "t1" && u.Status != nil && *u.Status == acp.ToolCallStatusCompleted
	})
	if end < 0 || end > at[0] {
		t.Fatalf("the call's completed update is at %d, the plan at %d; want the plan after it", end, at[0])
	}
	recorded := todoEntries(t, dir)
	if len(recorded) != 1 || len(recorded[0]) != 3 || recorded[0][1] != (tool.PlanItem{Content: "write the tool", Status: tool.PlanInProgress, Priority: tool.PriorityMedium}) {
		t.Fatalf("gobble.todo entries %+v", recorded)
	}
	if text, _ := slash(t, c, client, id, "/todos"); text != "[x] read the code\n[>] write the tool\n[ ] test it" {
		t.Fatalf("/todos = %q", text)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}

	loader := &acptest.Client{}
	c2, _ := sessionAgent(t, dir, llmtest.NewScript(), loader)
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	us = loader.Updates()
	got, at = plans(us)
	last := slices.IndexFunc(us, func(n acp.SessionNotification) bool {
		u := n.Update.AgentMessageChunk
		return u != nil && u.Content.Text != nil && u.Content.Text.Text == "planned"
	})
	if len(got) != 1 || !samePlan(got[0], todoWant) || last < 0 || at[0] < last {
		t.Fatalf("load: plan updates %+v at %v, the last agent message at %d; want the plan once, after it", got, at, last)
	}
}

// An empty list clears: the second plan update has no entries, the session
// records the empty list, /todos says so, and session/load sends no plan.
func TestTodoCleared(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	model := llmtest.NewScript(llmtest.ToolCall("t1", "todo", todoArgs), llmtest.ToolCall("t2", "todo", `{"todos":[]}`), llmtest.Text("cleared"))
	client := &acptest.Client{}
	c, _ := sessionAgent(t, dir, model, client)
	id := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, id, "plan, then clear"); err != nil {
		t.Fatal(err)
	}
	got, _ := plans(client.Updates())
	if len(got) != 2 || got[1] == nil || len(got[1]) != 0 {
		t.Fatalf("plan updates %+v, want two, the second empty", got)
	}
	if recorded := todoEntries(t, dir); len(recorded) != 2 || recorded[1] == nil || len(recorded[1]) != 0 {
		t.Fatalf("gobble.todo entries %+v, want two, the second empty", recorded)
	}
	if text, _ := slash(t, c, client, id, "/todos"); text != "no todo list" {
		t.Fatalf("/todos = %q", text)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	loader := &acptest.Client{}
	c2, _ := sessionAgent(t, dir, llmtest.NewScript(), loader)
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := plans(loader.Updates()); len(got) != 0 {
		t.Fatalf("load sent plans %+v for a cleared list", got)
	}
}
