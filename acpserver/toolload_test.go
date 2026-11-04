package acpserver

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// toolsCalls are the gobble.tools entries' call ids in session id's file,
// "" for an entry that names none.
func toolsCalls(t *testing.T, dir, id string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*", "*_"+id+".jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("session %s files %v, %v", id, files, err)
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for line := range strings.Lines(string(b)) {
		var e session.Entry
		if json.Unmarshal([]byte(line), &e) != nil || e.CustomType != customTools {
			continue
		}
		var d struct {
			CallID string `json:"callId"`
		}
		if err := json.Unmarshal(e.Data, &d); err != nil {
			t.Fatal(err)
		}
		out = append(out, d.CallID)
	}
	return out
}

// deferredOf are the names a request's Deferred specs carry.
func deferredOf(t *testing.T, req *llm.Request) []string {
	t.Helper()
	if len(req.Deferred) == 0 {
		return nil
	}
	var specs []tool.Spec
	if err := json.Unmarshal(req.Deferred, &specs); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}

// resultLoads is the Loads of call id's result among msgs, and whether it
// was found.
func resultLoads(msgs []llm.Message, id string) ([]string, bool) {
	for _, m := range msgs {
		for _, c := range m.Content {
			if c.Type == llm.ContentToolResult && c.ToolCallID == id {
				return c.Loads, true
			}
		}
	}
	return nil, false
}

// A load is recorded with its call; the next prompt, and a fork resumed
// from the file, send that call's result naming the load, with the tool
// among the deferred ones, so it is loaded natively (0012-MADR D16, D18).
func TestResumeReplaysLoads(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	model := llmtest.NewScript(
		llmtest.ToolCall("s1", builtin.ToolSearchName, `{"query":"widgets"}`),
		llmtest.Text("A1"),
		llmtest.Text("A2"),
		llmtest.Text("A3"),
	)
	c, client := loadAgent(t, dir, model, []tool.Tool{builtin.Read(), widgetTool()})
	id := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, id, "first"); err != nil {
		t.Fatal(err)
	}
	if got := toolsCalls(t, dir, string(id)); !slices.Equal(got, []string{"s1"}) {
		t.Fatalf("gobble.tools call ids %q, want s1", got)
	}
	if _, err := prompt(t, c, id, "second"); err != nil {
		t.Fatal(err)
	}
	list, _ := slash(t, c, client, id, "/fork")
	var second string
	for line := range strings.SplitSeq(list, "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[1] == "second" {
			second = f[0]
		}
	}
	_, resp := slash(t, c, client, id, "/fork "+second)
	forked := acp.SessionId(switchTo(t, resp))
	if _, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: forked, Cwd: cwd}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, forked, "in the fork"); err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{2, 3} {
		req := model.Requests()[i]
		if loads, ok := resultLoads(req.Messages, "s1"); !ok || !slices.Equal(loads, []string{"widget_tool"}) {
			t.Errorf("request %d: s1's result loads %q (found %v), want widget_tool", i, loads, ok)
		}
		if got := deferredOf(t, req); !slices.Contains(got, "widget_tool") {
			t.Errorf("request %d: Deferred %q lacks widget_tool", i, got)
		}
	}
}

// A gobble.tools entry puts its load on its call's result when the history
// is rebuilt; one with no call id, as gobble wrote before 0012-MADR D18,
// puts it on none, so the adapter sends the tool as an ordinary one (D17).
func TestReplayLoadsByCall(t *testing.T) {
	ts := session.Timestamp(time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC))
	for _, c := range []struct {
		name string
		data string
		want []string
	}{
		{"with its call", `{"loaded":["widget_tool"],"callId":"c1"}`, []string{"widget_tool"}},
		{"from before D18", `{"loaded":["widget_tool"]}`, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			entries := []session.Entry{
				{Type: session.TypeMessage, ID: "e1", Message: agentMessageWithCalls("c1")},
				{Type: session.TypeMessage, ID: "e2", ParentID: new("e1"), Message: toolResultMessage("c1", "bash", "ok", false, ts)},
				{Type: session.TypeCustom, ID: "e3", ParentID: new("e2"), CustomType: customTools, Data: jsontext.Value(c.data)},
			}
			loads, ok := resultLoads(replayState(entries).history, "c1")
			if !ok || !slices.Equal(loads, c.want) {
				t.Fatalf("c1's result loads %q (found %v), want %q", loads, ok, c.want)
			}
		})
	}
}

// widgetTool is a deferred tool for the load tests.
func widgetTool() tool.Tool {
	return tool.New("widget_tool", "Turn the widgets over.", func(context.Context, struct{}, tool.Env) (string, error) { return "turned", nil },
		tool.WithExposure(tool.ExposureDeferred), tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true}))
}

// loadAgent is an agent over the JSONL store in dir with tools, on p,
// joined to an approving client, which it returns too.
func loadAgent(t *testing.T, dir string, p llm.Provider, tools []tool.Tool) (*acptest.Conn, *acptest.Client) {
	t.Helper()
	ag := New(Options{Version: "1.2.3", Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil }, Models: func() ModelChoice { return testModels },
		Store: jsonl.New(dir, jsonl.Options{}), Tools: tools})
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

// loadsOf are the gobble.tools entries' lists in session id's file.
func loadsOf(t *testing.T, dir, id string) [][]string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*", "*_"+id+".jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("session %s files %v, %v", id, files, err)
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var out [][]string
	for line := range strings.Lines(string(b)) {
		var e session.Entry
		if json.Unmarshal([]byte(line), &e) != nil || e.CustomType != customTools {
			continue
		}
		var d toolsData
		if err := json.Unmarshal(e.Data, &d); err != nil {
			t.Fatal(err)
		}
		out = append(out, d.Loaded)
	}
	return out
}

// A tool_search load is recorded, the load reaches the same turn's next
// request, the next prompt starts with the tool sent and tool_search no
// longer offered, and a fork from after the load keeps it (0005-PLAN
// F1d-2).
func TestToolLoadRecordedAndReplayed(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	model := llmtest.NewScript(
		llmtest.ToolCall("s1", builtin.ToolSearchName, `{"query":"widgets"}`),
		llmtest.ToolCall("w1", "widget_tool", `{}`),
		llmtest.Text("A1"),
		llmtest.Text("A2"),
		llmtest.Text("A3"),
	)
	c, client := loadAgent(t, dir, model, []tool.Tool{builtin.Read(), widgetTool()})
	id := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, id, "first"); err != nil {
		t.Fatal(err)
	}
	reqs := model.Requests()
	llmtest.AssertTools(t, reqs[0], "read", builtin.ToolSearchName)
	llmtest.AssertTools(t, reqs[1], "read", builtin.ToolSearchName, "widget_tool")
	// The system prompt names no tool (0005-PLAN F3-1 deviation 2), and so
	// never names the deferred one as usable.
	if strings.Contains(systemOf(reqs[0]), "widget_tool") {
		t.Fatalf("system prompt %q names the deferred tool", systemOf(reqs[0]))
	}
	if got := loadsOf(t, dir, string(id)); len(got) != 1 || !slices.Equal(got[0], []string{"widget_tool"}) {
		t.Fatalf("gobble.tools entries %v", got)
	}
	if _, err := prompt(t, c, id, "second"); err != nil {
		t.Fatal(err)
	}
	// tool_search is offered, and the loaded tool keeps its place after
	// it (0005-PLAN F6-5 deviation 1; 0012-MADR D19).
	llmtest.AssertTools(t, model.Requests()[3], "read", builtin.ToolSearchName, "widget_tool")

	list, _ := slash(t, c, client, id, "/fork")
	var second string
	for line := range strings.SplitSeq(list, "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[1] == "second" {
			second = f[0]
		}
	}
	_, resp := slash(t, c, client, id, "/fork "+second)
	forked := acp.SessionId(switchTo(t, resp))
	if _, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: forked, Cwd: cwd}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, forked, "in the fork"); err != nil {
		t.Fatal(err)
	}
	llmtest.AssertTools(t, model.Requests()[4], "read", builtin.ToolSearchName, "widget_tool")
}

// A server that offers resources makes the three resource tools load at
// the first prompt: they are sent from its first request and named in the
// system prompt, the load is recorded once, and tool_search is offered for
// request_permissions alone, the system prompt naming no other category
// (0005-PLAN F1d-2; F6-5 deviation 1).
func TestResourceToolsLoadAtFirstPrompt(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("ok"), llmtest.Text("again"))
	c, a, _ := connectMCP(t, model, MCPOptions{})
	id := mcpSession(t, c, httpServer("docs", mcptest.ResourceHTTPServer(t)))
	for _, p := range []string{"first", "second"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	for i, req := range model.Requests() {
		names := toolNames(t, req)
		for _, n := range builtin.ResourceNames() {
			if !slices.Contains(names, n) {
				t.Errorf("request %d lacks %s: %v", i, n, names)
			}
		}
		if !slices.Contains(names, builtin.ToolSearchName) || !slices.Contains(names, "mcp__docs__echo") || slices.Contains(names, RequestPermissionsTool) {
			t.Errorf("request %d tools %v; want the server's tools, and tool_search", i, names)
		}
		if sys := req.Messages[0].Content[0].Text; !strings.Contains(sys, "- tool_search can find more tools for: permissions.\n") {
			t.Errorf("request %d system prompt: want only the permissions category deferred", i)
		}
	}
	a.mu.Lock()
	s := a.sessions[id]
	a.mu.Unlock()
	var loads [][]string
	for _, e := range s.log.Entries() {
		if e.CustomType == customTools {
			var d toolsData
			if err := json.Unmarshal(e.Data, &d); err != nil {
				t.Fatal(err)
			}
			loads = append(loads, d.Loaded)
		}
	}
	if len(loads) != 1 || !slices.Equal(loads[0], builtin.ResourceNames()) {
		t.Fatalf("loads %v; want the resource family once", loads)
	}
}

func systemOf(req *llm.Request) string {
	if len(req.Messages) == 0 || req.Messages[0].Role != llm.RoleSystem {
		return ""
	}
	return req.Messages[0].Content[0].Text
}

func toolNames(t *testing.T, req *llm.Request) []string {
	t.Helper()
	var specs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(req.Tools, &specs); err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Name
	}
	return out
}
