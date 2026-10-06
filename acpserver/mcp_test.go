package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
)

// fileServer is one mcp.json entry a test writes.
type fileServer struct {
	name    string
	Type    string            `json:"type,omitzero"`
	Command string            `json:"command,omitzero"`
	Env     map[string]string `json:"env,omitzero"`
	URL     string            `json:"url,omitzero"`
}

// stdioEntry is the stdio fixture as an mcp.json entry, with extra
// environment.
func stdioEntry(t *testing.T, name string, extra map[string]string) fileServer {
	t.Helper()
	cmd, env := mcptest.StdioServer(t)
	maps.Copy(env, extra)
	return fileServer{name: name, Command: cmd, Env: env}
}

// writeMCP writes an mcp.json holding servers, in order.
func writeMCP(t *testing.T, servers ...fileServer) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(`{"mcpServers": {`)
	for i, s := range servers {
		v, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 {
			b.WriteString(",")
		}
		n, err := json.Marshal(s.name)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(n)
		b.WriteString(":")
		b.Write(v)
	}
	b.WriteString("}}")
	p := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// connectMCP joins an agent with MCP options to an approving client, and
// returns its log.
func connectMCP(t *testing.T, p llm.Provider, mo MCPOptions) (*acptest.Conn, *Agent, *syncBuffer) {
	t.Helper()
	logs := &syncBuffer{}
	a := New(Options{
		Version: "1.2.3", NewID: counterIDs(), Provider: func() (llm.Provider, error) { return p, nil },
		MCP: mo, Logger: slog.New(slog.NewTextHandler(logs, nil)),
	})
	c := acptest.Connect(t, a, &acptest.Client{Allow: true})
	a.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Logf("close: %v", err)
		}
	})
	initialize(t, c.Client)
	return c, a, logs
}

func httpServer(name, url string) acp.McpServer {
	return acp.McpServer{Http: &acp.McpServerHttpInline{Name: name, Url: url, Type: "http", Headers: []acp.HttpHeader{}}}
}

func mcpSession(t *testing.T, c *acptest.Conn, servers ...acp.McpServer) acp.SessionId {
	t.Helper()
	if servers == nil {
		servers = []acp.McpServer{}
	}
	s, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: servers})
	if err != nil {
		t.Fatal(err)
	}
	return s.SessionId
}

// requestTools are the tool names a request declares.
func requestTools(t *testing.T, req *llm.Request) []string {
	t.Helper()
	var specs []struct {
		Name string `json:"name"`
	}
	if len(req.Tools) > 0 {
		if err := json.Unmarshal(req.Tools, &specs); err != nil {
			t.Fatal(err)
		}
	}
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Name
	}
	return out
}

func hasAll(got []string, want ...string) bool {
	for _, w := range want {
		if !slices.Contains(got, w) {
			return false
		}
	}
	return true
}

func lastText(req *llm.Request) string {
	m := req.Messages[len(req.Messages)-1]
	return m.Content[0].Text
}

// A stdio server from mcp.json and an HTTP server from session/new both
// appear as tools during session/prompt, and their calls reach the model
// (0002-PLAN Phase 5 Accept).
func TestMCPToolsInPrompt(t *testing.T) {
	model := llmtest.NewScript(
		llmtest.ToolCall("c1", "mcp__fx__echo", `{"text":"hi"}`),
		llmtest.ToolCall("c2", "mcp__web__add", `{"a":2,"b":3}`),
		llmtest.Text("done"),
	)
	c, _, _ := connectMCP(t, model, MCPOptions{Config: writeMCP(t, stdioEntry(t, "fx", nil))})
	id := mcpSession(t, c, httpServer("web", mcptest.HTTPServer(t, "")))
	if resp, err := prompt(t, c, id, "use the tools"); err != nil || resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("prompt = %+v, %v", resp, err)
	}
	reqs := model.Requests()
	if len(reqs) != 3 {
		t.Fatalf("%d requests", len(reqs))
	}
	if got := requestTools(t, reqs[0]); !hasAll(got, "read", "bash", "mcp__fx__echo", "mcp__fx__add", "mcp__fx__dotted_name", "mcp__web__echo", "mcp__web__add") {
		t.Fatalf("request tools %q", got)
	}
	if got := lastText(reqs[1]); got != "hi" {
		t.Fatalf("echo result %q", got)
	}
	if got := lastText(reqs[2]); got != "5" {
		t.Fatalf("add result %q", got)
	}
	llmtest.AssertSystem(t, reqs[0], "mcp__fx__echo")
}

// A session server replaces the file's server of the same name.
func TestSessionServerWins(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("ok"))
	cfg := writeMCP(t, fileServer{name: "web", URL: "http://127.0.0.1:1/mcp"})
	c, _, logs := connectMCP(t, model, MCPOptions{Config: cfg})
	id := mcpSession(t, c, httpServer("web", mcptest.HTTPServer(t, "")))
	if _, err := prompt(t, c, id, "hi"); err != nil {
		t.Fatal(err)
	}
	if got := requestTools(t, model.Requests()[0]); !hasAll(got, "mcp__web__echo") {
		t.Fatalf("request tools %q, want the session's web", got)
	}
	if strings.Contains(logs.String(), "needs attention") {
		t.Fatalf("the file's web was connected:\n%s", logs)
	}
}

// SSE and ACP-transport servers are refused on session/new: the agent does
// not offer them.
func TestUnofferedTransportsRefused(t *testing.T) {
	c, _, _ := connectMCP(t, llmtest.NewScript(), MCPOptions{})
	for _, s := range []acp.McpServer{
		{Sse: &acp.McpServerSseInline{Name: "old", Type: "sse", Url: "https://example.com/sse", Headers: []acp.HttpHeader{}}},
		{Acp: &acp.McpServerAcpInline{Name: "peer", Type: "acp", Id: "x"}},
		{Stdio: &acp.McpServerStdio{Name: "bad name", Command: "x", Args: []string{}, Env: []acp.EnvVariable{}}},
	} {
		_, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{s}})
		re, ok := errors.AsType[*acp.RequestError](err)
		if !ok || re.Code != -32602 {
			t.Errorf("%+v: err %v, want invalid params", s, err)
		}
	}
	_, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{
		{Sse: &acp.McpServerSseInline{Name: "old", Type: "sse", Url: "https://example.com/sse", Headers: []acp.HttpHeader{}}},
	}})
	if err == nil || !strings.Contains(err.Error(), "the SSE transport is not offered (mcpCapabilities.sse is false)") {
		t.Fatalf("SSE: %v", err)
	}
}

// The first prompt waits a bounded time; a server still connecting is
// logged, and its tools join a later prompt.
func TestStartupWait(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("first"), llmtest.Text("second"))
	cfg := writeMCP(t, stdioEntry(t, "fx", map[string]string{mcptest.EnvDelay: "1500ms"}))
	c, a, logs := connectMCP(t, model, MCPOptions{Config: cfg, StartupWait: 200 * time.Millisecond})
	id := mcpSession(t, c)
	begin := time.Now()
	if _, err := prompt(t, c, id, "one"); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(begin); d > 1200*time.Millisecond {
		t.Fatalf("the first prompt waited %v", d)
	}
	if got := requestTools(t, model.Requests()[0]); slices.ContainsFunc(got, func(n string) bool { return strings.HasPrefix(n, "mcp__") }) {
		t.Fatalf("first request had MCP tools %q", got)
	}
	if !strings.Contains(logs.String(), "still connecting") || !strings.Contains(logs.String(), "server=fx") {
		t.Fatalf("log:\n%s", logs)
	}
	a.mu.Lock()
	m := a.sessions[id].mcp.m
	a.mu.Unlock()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := m.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, id, "two"); err != nil {
		t.Fatal(err)
	}
	if got := requestTools(t, model.Requests()[1]); !hasAll(got, "mcp__fx__echo") {
		t.Fatalf("second request tools %q", got)
	}
}

// session/resume connects the servers it carries.
func TestResumeConnects(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("one"), llmtest.Text("two"))
	c, _, _ := connectMCP(t, model, MCPOptions{})
	id := mcpSession(t, c)
	if _, err := prompt(t, c, id, "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	_, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: t.TempDir(),
		McpServers: []acp.McpServer{httpServer("web", mcptest.HTTPServer(t, ""))}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, id, "two"); err != nil {
		t.Fatal(err)
	}
	if got := requestTools(t, model.Requests()[1]); !hasAll(got, "mcp__web__echo") {
		t.Fatalf("request tools %q", got)
	}
}

// The file's invalid entries are logged once, at the first prompt.
func TestConfigErrorsLogged(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("one"), llmtest.Text("two"))
	cfg := writeMCP(t, fileServer{name: "old", Type: "sse", URL: "https://example.com/sse"})
	c, _, logs := connectMCP(t, model, MCPOptions{Config: cfg})
	id := mcpSession(t, c)
	for _, p := range []string{"one", "two"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	if n := strings.Count(logs.String(), "legacy SSE transport is not supported"); n != 1 {
		t.Fatalf("logged %d times:\n%s", n, logs)
	}
}
