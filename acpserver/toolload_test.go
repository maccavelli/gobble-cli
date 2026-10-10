package acpserver

import (
	"context"
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
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// widgetTool is a deferred tool for the load tests.
func widgetTool() tool.Tool {
	return tool.New("widget_tool", "Turn the widgets over.", func(context.Context, struct{}, tool.Env) (string, error) { return "turned", nil },
		tool.WithExposure(tool.ExposureDeferred), tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true}))
}

// loadAgent is an agent over the JSONL store in dir with tools, on p,
// joined to an approving client, which it returns too.
func loadAgent(t *testing.T, dir string, p llm.Provider, tools []tool.Tool) (*acptest.Conn, *acptest.Client) {
	t.Helper()
	ag := New(Options{Version: "1.2.3", Provider: func() (llm.Provider, error) { return p, nil }, Models: func() ModelChoice { return testModels },
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
	if !strings.Contains(systemOf(reqs[0]), "read, tool_search") || strings.Contains(systemOf(reqs[0]), "widget_tool") {
		t.Fatalf("system prompt %q; want read and tool_search named, the deferred tool not", systemOf(reqs[0]))
	}
	if got := loadsOf(t, dir, string(id)); len(got) != 1 || !slices.Equal(got[0], []string{"widget_tool"}) {
		t.Fatalf("gobble.tools entries %v", got)
	}
	if _, err := prompt(t, c, id, "second"); err != nil {
		t.Fatal(err)
	}
	llmtest.AssertTools(t, model.Requests()[3], "read", "widget_tool")

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
	llmtest.AssertTools(t, model.Requests()[4], "read", "widget_tool")
}

// A server that offers resources makes the three resource tools load at
// the first prompt: they are sent from its first request and named in the
// system prompt, the load is recorded once, and tool_search is not offered
// (0005-PLAN F1d-2).
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
		if slices.Contains(names, builtin.ToolSearchName) || !slices.Contains(names, "mcp__docs__echo") {
			t.Errorf("request %d tools %v; want the server's tools, and no tool_search", i, names)
		}
		if !strings.Contains(systemOf(req), builtin.ReadResourceName) {
			t.Errorf("request %d's system prompt does not name %s", i, builtin.ReadResourceName)
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
