package acpserver

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
	"github.com/maccavelli/gobble-cli/permission"
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
		Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
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
	// The system prompt names no tool; the request carries their specs
	// (0005-PLAN F3-1 deviation 2).
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
	// SSE and ACP are sent raw, since the client's SDK refuses to send a
	// transport the agent did not advertise; the agent's SDK refuses them
	// (0019-MADR D7, D8).
	for _, tc := range []struct {
		s          acp.McpServer
		capability string
	}{
		{acp.McpServer{Sse: &acp.McpServerSseInline{Name: "old", Type: "sse", Url: "https://example.com/sse", Headers: []acp.HttpHeader{}}}, "mcpCapabilities.sse"},
		//lint:ignore SA1019 acp-go-sdk2 keeps this unstable-schema name under v1 (0035-MADR D6)
		{acp.McpServer{Acp: &acp.McpServerAcpInline{Name: "peer", Type: "acp", ServerId: "x"}}, "mcpCapabilities.acp"}, //nolint:staticcheck // as the lint:ignore above
	} {
		_, err := acp.SendRequest[acp.NewSessionResponse](c.Client.Connection(), t.Context(), acp.AgentMethodSessionNew,
			acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{tc.s}})
		re, ok := errors.AsType[*acp.RequestError](err)
		if !ok || re.Code != -32602 || !strings.Contains(fmt.Sprint(re.Data), tc.capability) {
			t.Errorf("%s: err %v, want invalid params naming it", tc.capability, err)
		}
	}
	// gobble's own check refuses a stdio server's bad name.
	_, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{
		{Stdio: &acp.McpServerStdio{Name: "bad name", Command: acptest.Abs("/bin/x"), Args: []string{}, Env: []acp.EnvVariable{}}},
	}})
	if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32602 {
		t.Errorf("bad name: err %v, want invalid params", err)
	}
	_, err = acp.SendRequest[acp.NewSessionResponse](c.Client.Connection(), t.Context(), acp.AgentMethodSessionNew,
		acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{
			{Sse: &acp.McpServerSseInline{Name: "old", Type: "sse", Url: "https://example.com/sse", Headers: []acp.HttpHeader{}}},
		}})
	if re, ok := errors.AsType[*acp.RequestError](err); !ok || !strings.Contains(fmt.Sprint(re.Data), "mcpCapabilities.sse") {
		t.Fatalf("SSE: %v, want the SDK's mcpCapabilities.sse reason", err)
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

// The start-up wait's default is internal/config's mcp.startupWaitMs
// (10000): a change here, or there, fails one of the two tests (0018-PLAN
// P2 deviation 5).
func TestDefaultStartupWait(t *testing.T) {
	if DefaultStartupWait != 10*time.Second {
		t.Fatalf("DefaultStartupWait %v; internal/config's mcp.startupWaitMs says 10s", DefaultStartupWait)
	}
}

// xaiRefusal is an in-process transport that answers every request with
// xAI's recorded 400 for a tool schema it refuses, naming the tool last set.
type xaiRefusal struct {
	mu   sync.Mutex
	tool string
}

func (x *xaiRefusal) name(tool string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.tool = tool
}

func (x *xaiRefusal) RoundTrip(r *http.Request) (*http.Response, error) {
	x.mu.Lock()
	tool := x.tool
	x.mu.Unlock()
	body := fmt.Sprintf(`{"code":"invalid-argument","error":"Failed to start sampling: [invalid_client_tool_schema] %s: `+
		`tool parameter root must be an object type (root schema is an anyOf/oneOf union with a non-object branch)"}`, tool)
	return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

// refusedUnion is the first tool of a request whose root holds a union with
// a branch that is not typed object, the shape xAI refuses (0060-MADR P1,
// X1), or "".
func refusedUnion(t *testing.T, tools jsontext.Value) string {
	t.Helper()
	if len(tools) == 0 {
		return ""
	}
	var specs []struct {
		Name        string         `json:"name"`
		InputSchema map[string]any `json:"inputSchema"`
	}
	if err := json.Unmarshal(tools, &specs); err != nil {
		t.Errorf("request tools: %v", err)
		return ""
	}
	for _, s := range specs {
		for _, kw := range []string{"anyOf", "oneOf"} {
			branches, ok := s.InputSchema[kw].([]any)
			if !ok {
				continue // no list, no union
			}
			for _, b := range branches {
				if m, ok := b.(map[string]any); !ok || m["type"] != "object" {
					return s.Name
				}
			}
		}
	}
	return ""
}

// rootUnionRefuser is a provider over a script that refuses, as xAI does, a
// request whose tools carry a root union with a branch not typed object,
// with the error gobble's grok provider makes of xAI's body, and otherwise
// plays the script. The refusal goes through llm/provider's real adapter and
// the SDK's error mapping over an in-process transport (acpserver may not
// import the SDK, 0004-MADR rule 4).
type rootUnionRefuser struct {
	t     *testing.T
	inner *llmtest.Script
	wire  *xaiRefusal
	grok  llm.Provider

	mu      sync.Mutex
	refused []string
}

var _ llm.Provider = (*rootUnionRefuser)(nil)

// newRootUnionRefuser is a rootUnionRefuser that plays inner.
func newRootUnionRefuser(t *testing.T, inner *llmtest.Script) *rootUnionRefuser {
	t.Helper()
	wire := &xaiRefusal{}
	grok, err := provider.New("grok", provider.Options{APIKey: "k", Model: "grok-4.6", HTTPClient: &http.Client{Transport: wire}})
	if err != nil {
		t.Fatal(err)
	}
	return &rootUnionRefuser{t: t, inner: inner, wire: wire, grok: grok}
}

func (r *rootUnionRefuser) ID() string                     { return "grok" }
func (r *rootUnionRefuser) Capabilities() llm.Capabilities { return r.inner.Capabilities() }

func (r *rootUnionRefuser) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	if name := refusedUnion(r.t, req.Tools); name != "" {
		r.mu.Lock()
		r.refused = append(r.refused, name)
		r.mu.Unlock()
		r.wire.name(name)
		return r.grok.Stream(ctx, &llm.Request{Model: "grok-4.6", Messages: []llm.Message{{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "x"}}}}})
	}
	return r.inner.Stream(ctx, req)
}

func (r *rootUnionRefuser) refusals() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.refused)
}

// refusalText is the tail of the owner's paste of 2026-11-16 (0060-MADR P1),
// with tool in place of the magictools one.
func refusalText(tool string) string {
	return "grok HTTP 400 invalid-argument: Failed to start sampling: [invalid_client_tool_schema] " + tool +
		": tool parameter root must be an object type (root schema is an anyOf/oneOf union with a non-object branch)"
}

// unionAgent is an agent over p with the union fixture served on HTTP as
// server fx, settings ss and MCP options mo, whose client offers notices,
// logging at debug level, with a session that has the fixture.
func unionAgent(t *testing.T, p llm.Provider, ss SessionSettings, mo MCPOptions) (*acptest.Conn, *acptest.Client, *syncBuffer, acp.SessionId, *mcptest.UnionCalls) {
	t.Helper()
	r := newUnionRig(t, p, ss, mo)
	return r.conn, r.client, r.logs, r.id, r.calls
}

// unionRig is unionAgent's parts, and the agent, for a test that changes the
// session's rules while it runs.
type unionRig struct {
	conn   *acptest.Conn
	client *acptest.Client
	logs   *syncBuffer
	id     acp.SessionId
	calls  *mcptest.UnionCalls
	agent  *Agent
}

func newUnionRig(t *testing.T, p llm.Provider, ss SessionSettings, mo MCPOptions) unionRig {
	t.Helper()
	url, calls := mcptest.UnionHTTPServer(t)
	logs := &syncBuffer{}
	a := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Settings: func(string) SessionSettings { return ss }, MCP: mo,
		Logger: slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))})
	client := &acptest.Client{Allow: true}
	c := acptest.Connect(t, a, client)
	a.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Logf("close: %v", err)
		}
	})
	if _, err := c.Client.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber, ClientCapabilities: offersNotices,
		ClientInfo: &acp.Implementation{Name: "test", Version: "0.0.0"}}); err != nil {
		t.Fatal(err)
	}
	return unionRig{conn: c, client: client, logs: logs, id: mcpSession(t, c, httpServer("fx", url)), calls: calls, agent: a}
}

// schemaOf is the input schema a request carries for tool name.
func schemaOf(t *testing.T, req *llm.Request, name string) map[string]any {
	t.Helper()
	var specs []struct {
		Name        string         `json:"name"`
		InputSchema map[string]any `json:"inputSchema"`
	}
	if err := json.Unmarshal(req.Tools, &specs); err != nil {
		t.Fatal(err)
	}
	for _, s := range specs {
		if s.Name == name {
			return s.InputSchema
		}
	}
	t.Fatalf("no tool %s in the request", name)
	return nil
}

// dataSchema is a schema of mcpclient's testdata, decoded.
func dataSchema(t *testing.T, name string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "mcpclient", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

const unionWarning = "mcp: tool schema sent as the server gave it"

// The provider double refuses the original real schema with the owner's
// text and accepts the expected rewrite: the test of the test double, with no
// MCP server, so the end-to-end tests below mean what they say.
func TestMCPUnionSchemaProviderRefusal(t *testing.T) {
	spec := func(schema map[string]any) jsontext.Value {
		b, err := json.Marshal([]map[string]any{{"name": "mcp__fx__execute_pipeline", "inputSchema": schema}})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	model := newRootUnionRefuser(t, llmtest.NewScript(llmtest.Text("ok")))
	var got error
	for _, err := range model.Stream(t.Context(), &llm.Request{Tools: spec(dataSchema(t, "magictools-execute-pipeline.json"))}) {
		got = err
	}
	// The owner's paste reads "llm: api status 400: … grok HTTP 400 …"; the
	// ellipsis is the SDK's own words.
	want := "llm: api status 400: llmprovider: invalid request: " + refusalText("mcp__fx__execute_pipeline")
	if got == nil || got.Error() != want {
		t.Fatalf("the original schema: %v\nwant text %q", got, want)
	}
	got = nil
	for _, err := range model.Stream(t.Context(), &llm.Request{Tools: spec(dataSchema(t, "magictools-execute-pipeline.rewritten.json"))}) {
		got = err
	}
	if got != nil || len(model.refusals()) != 1 {
		t.Fatalf("the expected rewrite: %v, %d refusals, want none and the one", got, len(model.refusals()))
	}
}

// 0060-MADR Confirmation 2: a session with the real schema from an MCP
// server, over a provider that refuses the un-normalised shape as xAI does,
// completes its prompt; the request carries the rewrite; a call of the tool
// reaches the server with its arguments intact; the rewrite is noted once at
// debug level, and nothing is warned. The tool filter leaves out the tool
// the rule cannot rewrite.
func TestMCPUnionSchemaRewritten(t *testing.T) {
	const args = `{"target":"/tmp/x","intent":"preview","dry_run":true}`
	model := newRootUnionRefuser(t, llmtest.NewScript(
		llmtest.ToolCall("c1", "mcp__fx__execute_pipeline", args), llmtest.Text("done"), llmtest.Text("again")))
	c, client, logs, id, calls := unionAgent(t, model, SessionSettings{}, MCPOptions{Allow: func(n string) bool { return n != "mcp__fx__pick_one" }})
	for _, p := range []string{"run it", "once more"} {
		resp, err := prompt(t, c, id, p)
		if err != nil || resp.StopReason != acp.StopReasonEndTurn {
			t.Fatalf("prompt %q = %+v, %v", p, resp, err)
		}
	}
	reqs := model.inner.Requests()
	if len(reqs) != 3 || len(model.refusals()) != 0 {
		t.Fatalf("%d requests, refused %q", len(reqs), model.refusals())
	}
	if got := requestTools(t, reqs[0]); !hasAll(got, "mcp__fx__execute_pipeline") || slices.Contains(got, "mcp__fx__pick_one") {
		t.Fatalf("request tools %q", got)
	}
	want := dataSchema(t, "magictools-execute-pipeline.rewritten.json")
	for i, r := range reqs {
		if got := schemaOf(t, r, "mcp__fx__execute_pipeline"); !reflect.DeepEqual(got, want) {
			t.Errorf("request %d: the schema is\n%v\nwant the expected rewrite\n%v", i, got, want)
		}
	}
	if got := calls.Args(); len(got) != 1 || got[0] != args {
		t.Errorf("the server received %q, want %q", got, args)
	}
	if got := lastText(reqs[1]); got != "ran" {
		t.Errorf("the tool's result %q", got)
	}
	if n := strings.Count(logs.String(), "mcp: tool schema rewritten"); n != 1 || !strings.Contains(logs.String(), "tool=execute_pipeline") {
		t.Errorf("%d rewrite records over two prompts, want 1 naming the tool:\n%s", n, logs)
	}
	if strings.Contains(logs.String(), unionWarning) || len(notices(client)) != 0 {
		t.Errorf("a warning or notice for a tool the rule rewrote:\n%s\n%+v", logs, notices(client))
	}
}

// A tool the rule cannot rewrite, offered, is flagged: the log gets one
// warning and the client one notice for the session however many prompts,
// each naming the server and tool, the reason and the remedy, and the
// provider's refusal that follows is the one the flag predicts (0060-MADR
// D3, D4).
func TestMCPUnionSchemaFlaggedOnce(t *testing.T) {
	model := newRootUnionRefuser(t, llmtest.NewScript())
	c, client, logs, id, _ := unionAgent(t, model, SessionSettings{}, MCPOptions{})
	for _, p := range []string{"one", "two"} {
		_, err := prompt(t, c, id, p)
		var re *acp.RequestError
		if !errors.As(err, &re) || !strings.Contains(fmt.Sprint(re.Data), refusalText("mcp__fx__pick_one")) {
			t.Fatalf("prompt %q: %v, want the provider's refusal of mcp__fx__pick_one", p, err)
		}
	}
	if got := model.refusals(); !slices.Equal(got, []string{"mcp__fx__pick_one", "mcp__fx__pick_one"}) {
		t.Fatalf("refused %q: the real schema was rewritten, the other is what xAI refuses", got)
	}
	if n := strings.Count(logs.String(), unionWarning); n != 1 {
		t.Fatalf("%d warning records over two prompts, want 1:\n%s", n, logs)
	}
	for _, w := range []string{"level=WARN", "server=fx", "tool=pick_one", "its root anyOf has a branch that is not an object", "--exclude-tools mcp__fx__pick_one", "a permission rule that denies it with no pattern"} {
		if !strings.Contains(logs.String(), w) {
			t.Errorf("the log lacks %q:\n%s", w, logs)
		}
	}
	got := notices(client)
	if len(got) != 1 || got[0].Severity != acp.NoticeSeverityWarning || got[0].Title != unionWarning {
		t.Fatalf("notices %+v, want one warning titled %q", got, unionWarning)
	}
	for _, w := range []string{"fx/pick_one", "its root anyOf has a branch that is not an object, which some providers refuse", "--exclude-tools mcp__fx__pick_one", "a permission rule that denies it with no pattern"} {
		if !strings.Contains(descriptionOf(got[0]), w) {
			t.Errorf("the notice lacks %q: %q", w, descriptionOf(got[0]))
		}
	}
}

// A tool a deny rule hides is not warned of, and when the rule is lifted
// within the session, the first prompt that sends the tool gets exactly one
// warning and one notice, and the next prompts none: hidden is not reported,
// and so not counted as reported (0060-MADR D4; the owner's ruling R-2 of
// 2026-11-17).
func TestMCPUnionSchemaHiddenThenShown(t *testing.T) {
	model := newRootUnionRefuser(t, llmtest.NewScript(llmtest.Text("done")))
	r := newUnionRig(t, model, SessionSettings{Rules: []permission.Spec{rule(t, permission.ModeDeny, "mcp__fx__pick_one", 3)}}, MCPOptions{})
	if resp, err := prompt(t, r.conn, r.id, "hidden"); err != nil || resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("the prompt with the tool hidden = %+v, %v", resp, err)
	}
	if strings.Contains(r.logs.String(), unionWarning) || len(notices(r.client)) != 0 {
		t.Fatalf("a warning while the tool is hidden:\n%s\n%+v", r.logs, notices(r.client))
	}
	// Lift the rule, as /permissions would.
	r.agent.mu.Lock()
	s := r.agent.sessions[r.id]
	s.rules = engineOf(SessionSettings{}, t.TempDir(), nil)
	r.agent.mu.Unlock()
	for _, p := range []string{"shown", "shown again"} {
		_, err := prompt(t, r.conn, r.id, p)
		var re *acp.RequestError
		if !errors.As(err, &re) || !strings.Contains(fmt.Sprint(re.Data), refusalText("mcp__fx__pick_one")) {
			t.Fatalf("prompt %q: %v, want the provider's refusal of mcp__fx__pick_one now that it is sent", p, err)
		}
		if n := strings.Count(r.logs.String(), unionWarning); n != 1 {
			t.Fatalf("after prompt %q: %d warning records, want exactly 1:\n%s", p, n, r.logs)
		}
		if got := notices(r.client); len(got) != 1 || got[0].Title != unionWarning {
			t.Fatalf("after prompt %q: notices %+v, want exactly one", p, got)
		}
	}
}

// The remedies the warning names work: a tool the filter leaves out, and a
// tool a deny rule hides, are neither sent, nor warned of, so the prompt
// completes (0060-MADR D4).
func TestMCPUnionSchemaRemedies(t *testing.T) {
	for _, tc := range []struct {
		name string
		ss   SessionSettings
		mo   MCPOptions
	}{
		{"the tool filter", SessionSettings{}, MCPOptions{Allow: func(n string) bool { return n != "mcp__fx__pick_one" }}},
		{"a deny rule", SessionSettings{Rules: []permission.Spec{rule(t, permission.ModeDeny, "mcp__fx__pick_one", 3)}}, MCPOptions{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := newRootUnionRefuser(t, llmtest.NewScript(llmtest.Text("done")))
			c, client, logs, id, _ := unionAgent(t, model, tc.ss, tc.mo)
			if resp, err := prompt(t, c, id, "go"); err != nil || resp.StopReason != acp.StopReasonEndTurn {
				t.Fatalf("prompt = %+v, %v", resp, err)
			}
			if got := requestTools(t, model.inner.Requests()[0]); slices.Contains(got, "mcp__fx__pick_one") || !slices.Contains(got, "mcp__fx__execute_pipeline") {
				t.Fatalf("request tools %q", got)
			}
			if strings.Contains(logs.String(), unionWarning) || len(notices(client)) != 0 {
				t.Errorf("a warning for a tool that is not sent:\n%s\n%+v", logs, notices(client))
			}
		})
	}
}
