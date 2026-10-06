package mcpclient

import (
	"context"
	"encoding/json/jsontext"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
	"github.com/maccavelli/gobble-cli/tool"
)

func TestMain(m *testing.M) {
	mcptest.ServeStdioIfAsked()
	os.Exit(m.Run())
}

// stdioFixture is the stdio fixture as a server named name, with extra
// environment.
func stdioFixture(t *testing.T, name string, extra ...Pair) Server {
	t.Helper()
	cmd, env := mcptest.StdioServer(t)
	s := Server{Name: name, Command: cmd, Timeout: DefaultTimeout, Enabled: true, Literal: true}
	for k, v := range env {
		s.Env = append(s.Env, Pair{Name: k, Value: v})
	}
	s.Env = append(s.Env, extra...)
	return s
}

func httpFixture(name, url string, headers ...Pair) Server {
	return Server{Name: name, URL: url, Headers: headers, Timeout: DefaultTimeout, Enabled: true}
}

func connectAll(t *testing.T, o Options, servers ...Server) *Manager {
	t.Helper()
	m := Start(t.Context(), servers, o)
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Logf("close: %v", err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := m.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	return m
}

func names(tools []tool.Tool) []string {
	out := make([]string, len(tools))
	for i, tl := range tools {
		out[i] = tl.Spec().Name
	}
	return out
}

func byName(t *testing.T, tools []tool.Tool, name string) tool.Tool {
	t.Helper()
	for _, tl := range tools {
		if tl.Spec().Name == name {
			return tl
		}
	}
	t.Fatalf("no tool %s in %v", name, names(tools))
	return nil
}

func call(t *testing.T, tl tool.Tool, args string) tool.Result {
	t.Helper()
	res, err := tl.Run(t.Context(), tool.Call{ID: "c1", Name: tl.Spec().Name, Args: jsontext.Value(args)}, tool.Env{})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// A stdio and an HTTP server both connect, and their tools run.
func TestStdioAndHTTP(t *testing.T) {
	m := connectAll(t, Options{Cwd: t.TempDir(), ClientName: "gobble", ClientVersion: "0.0.0-test"},
		stdioFixture(t, "fx"), httpFixture("web", mcptest.HTTPServer(t, "")))
	tools := m.Tools(nil)
	for _, want := range []string{"mcp__fx__echo", "mcp__fx__add", "mcp__fx__sleep", "mcp__fx__dotted_name", "mcp__web__echo"} {
		byName(t, tools, want)
	}
	echo := byName(t, tools, "mcp__fx__echo")
	if !echo.Spec().Annotations.ReadOnlyHint || echo.Spec().Kind != tool.KindOther {
		t.Fatalf("echo spec %+v", echo.Spec())
	}
	if d, ok := echo.(tool.Describer); !ok || d.Describe(tool.Call{}, tool.Env{}) != "fx/echo" {
		t.Fatal("echo is not titled fx/echo")
	}
	if res := call(t, echo, `{"text":"hi"}`); res.IsError || res.Text() != "hi" {
		t.Fatalf("echo = %+v", res)
	}
	if res := call(t, byName(t, tools, "mcp__web__add"), `{"a":2,"b":3}`); res.IsError || res.Text() != "5" {
		t.Fatalf("add = %+v", res)
	}
	if res := call(t, echo, `[1]`); !res.IsError {
		t.Fatalf("array arguments = %+v", res)
	}
	got := m.Tools(func(n string) bool { return n == "mcp__fx__add" })
	if !slices.Equal(names(got), []string{"mcp__fx__add"}) {
		t.Fatalf("allow kept %v", names(got))
	}
	for _, st := range m.Statuses() {
		if st.State != StateConnected || len(st.Tools) != 4 || st.Exposure != "direct" {
			t.Errorf("status %+v", st)
		}
	}
}

// A call that outlives the server's timeout is an error result naming it.
func TestCallTimeout(t *testing.T) {
	s := stdioFixture(t, "fx")
	s.Timeout = 500 * time.Millisecond
	m := connectAll(t, Options{}, s)
	begin := time.Now()
	res := call(t, byName(t, m.Tools(nil), "mcp__fx__sleep"), `{"ms":5000}`)
	if !res.IsError || res.Text() != "MCP tool fx/sleep timed out after 0.5 s" {
		t.Fatalf("sleep = %+v", res)
	}
	if d := time.Since(begin); d > 3*time.Second {
		t.Fatalf("the timeout took %v", d)
	}
}

// A bearer header resolves from the environment; unset, the server fails
// with Pi's message; with no Authorization header, a 401 means sign-in.
func TestBearerAndSignIn(t *testing.T) {
	url := mcptest.HTTPServer(t, "sekrit")
	auth := Pair{Name: "Authorization", Value: "Bearer ${FX_TOKEN}"}
	with := connectAll(t, Options{Getenv: func(k string) string { return map[string]string{"FX_TOKEN": "sekrit"}[k] }},
		httpFixture("web", url, auth))
	if st := with.Statuses()[0]; st.State != StateConnected {
		t.Fatalf("with the token: %+v", st)
	}
	without := connectAll(t, Options{Getenv: func(string) string { return "" }}, httpFixture("web", url, auth), httpFixture("open", url))
	sts := without.Statuses()
	if sts[0].State != StateFailed || sts[0].Error != `Failed to resolve MCP server "web" header "Authorization" from environment variable: FX_TOKEN` {
		t.Fatalf("unset token: %+v", sts[0])
	}
	if sts[1].State != StateNeedsAuth || sts[1].Error != `MCP server "open" requires sign-in, which arrives with 0005-PLAN F8 (gobble mcp login)` {
		t.Fatalf("no header: %+v", sts[1])
	}
	if len(without.Tools(nil)) != 0 {
		t.Fatal("servers that did not connect offered tools")
	}
}

// A stdio server that exits shows its stderr in the error.
func TestStderrInTheError(t *testing.T) {
	m := connectAll(t, Options{}, stdioFixture(t, "bad", Pair{Name: mcptest.EnvFail, Value: "boom: no config"}))
	st := m.Statuses()[0]
	if st.State != StateFailed || !strings.Contains(st.Error, "boom: no config") {
		t.Fatalf("status %+v", st)
	}
}

// A disabled server is listed and never started.
func TestDisabled(t *testing.T) {
	marker := t.TempDir() + "/started"
	s := stdioFixture(t, "off", Pair{Name: mcptest.EnvMarker, Value: marker})
	s.Enabled = false
	m := connectAll(t, Options{}, s)
	if st := m.Statuses()[0]; st.State != StateDisabled || st.Enabled {
		t.Fatalf("status %+v", st)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("a disabled server was started")
	}
}

// HTTP requests carry the User-Agent and the configured headers.
func TestHTTPHeaders(t *testing.T) {
	var (
		mu  sync.Mutex
		uas []string
		xs  []string
	)
	srv := mcptest.NewServer()
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		uas, xs = append(uas, r.UserAgent()), append(xs, r.Header.Get("X-Team"))
		mu.Unlock()
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(func() { ts.CloseClientConnections(); ts.Close() })
	connectAll(t, Options{UserAgent: "gobble/1.2.3 (test)"}, httpFixture("web", ts.URL, Pair{Name: "X-Team", Value: "blue"}))
	mu.Lock()
	defer mu.Unlock()
	if len(uas) == 0 || uas[0] != "gobble/1.2.3 (test)" || xs[0] != "blue" {
		t.Fatalf("user agents %q, headers %q", uas, xs)
	}
}

// Results convert as Pi converts them.
func TestResultText(t *testing.T) {
	size := int64(2048)
	r := &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: "one"},
		&mcp.ImageContent{MIMEType: "image/png", Data: []byte{1}},
		&mcp.AudioContent{MIMEType: "audio/wav", Data: []byte{1}},
		&mcp.ResourceLink{URI: "file:///a.txt", Name: "a", MIMEType: "text/plain", Size: &size, Description: "notes"},
		&mcp.EmbeddedResource{Resource: &mcp.ResourceContents{URI: "file:///b", Text: "embedded"}},
		&mcp.EmbeddedResource{Resource: &mcp.ResourceContents{URI: "file:///c.json", MIMEType: "application/json", Blob: []byte(`{"x":1}`)}},
		&mcp.EmbeddedResource{Resource: &mcp.ResourceContents{URI: "file:///d.bin", Blob: []byte{0}}},
	}}
	want := strings.Join([]string{
		"one", "[image image/png omitted]", "[audio audio/wav omitted]",
		`[Resource file:///a.txt "a" (text/plain, 2.0KB): notes]`, "embedded", `{"x":1}`,
		"[binary resource file:///d.bin (unknown type) omitted]",
	}, "\n")
	if got := resultText(r); got != want {
		t.Fatalf("text\n%s\nwant\n%s", got, want)
	}
	structured := &mcp.CallToolResult{StructuredContent: map[string]any{"b": 1, "a": "x"}}
	if got := resultText(structured); got != "{\n  \"a\": \"x\",\n  \"b\": 1\n}" {
		t.Fatalf("structured %q", got)
	}
}
