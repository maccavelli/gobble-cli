package mcptest

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The variables the stdio fixture reads.
const (
	// EnvServe set to "stdio" makes ServeStdioIfAsked serve.
	EnvServe = "GOBBLE_MCPTEST"
	// EnvDelay is a time.Duration the server waits before it serves.
	EnvDelay = "GOBBLE_MCPTEST_DELAY"
	// EnvFail is a line the server writes to stderr before it exits 3.
	EnvFail = "GOBBLE_MCPTEST_FAIL"
	// EnvMarker is a file the server creates when it starts.
	EnvMarker = "GOBBLE_MCPTEST_MARKER"
	// EnvStubborn set to "1" makes the server, and the child it starts,
	// ignore SIGTERM, and the server keep running after stdin ends
	// (0002-PLAN Phase 8).
	EnvStubborn = "GOBBLE_MCPTEST_STUBBORN"
	// EnvOrphan set to "1" makes the server start a child and leave it
	// running when it exits.
	EnvOrphan = "GOBBLE_MCPTEST_ORPHAN"
	// EnvPIDs is a file the server writes "<server pid> <child pid>" to, in
	// the stubborn and orphan modes, before it serves.
	EnvPIDs = "GOBBLE_MCPTEST_PIDS"
)

// childLife bounds a fixture child that nothing stops, so a failed test
// leaks it for a while only.
const childLife = 2 * time.Minute

type echoIn struct {
	Text string `json:"text"`
}

type addIn struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type sleepIn struct {
	MS int `json:"ms"`
}

func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

// NewServer is the fixture: echo {text} (read-only), add {a, b},
// sleep {ms}, and dotted.name, whose name needs sanitizing.
func NewServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "gobble-mcptest", Version: "1.0.0"}, nil)
	mcp.AddTool(s, &mcp.Tool{Name: "echo", Description: "Echo the text.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(_ context.Context, _ *mcp.CallToolRequest, in echoIn) (*mcp.CallToolResult, any, error) {
			return text(in.Text), nil, nil
		})
	mcp.AddTool(s, &mcp.Tool{Name: "add", Description: "Add two numbers."},
		func(_ context.Context, _ *mcp.CallToolRequest, in addIn) (*mcp.CallToolResult, any, error) {
			return text(strconv.FormatFloat(in.A+in.B, 'f', -1, 64)), nil, nil
		})
	mcp.AddTool(s, &mcp.Tool{Name: "sleep", Description: "Sleep, then answer."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in sleepIn) (*mcp.CallToolResult, any, error) {
			select {
			case <-time.After(time.Duration(in.MS) * time.Millisecond):
				return text("slept"), nil, nil
			case <-ctx.Done():
				return nil, nil, context.Cause(ctx)
			}
		})
	mcp.AddTool(s, &mcp.Tool{Name: "dotted.name", Description: "A tool whose name has a dot."},
		func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
			return text("dotted"), nil, nil
		})
	return s
}

// ResourceURI, ResourceText and TemplateURI are the resource fixture's
// resource, its text, and its template's uri pattern.
const (
	ResourceURI  = "doc://readme"
	ResourceText = "# Read me"
	TemplateURI  = "doc://page/{n}"
)

// NewResourceServer is NewServer with resources: ResourceURI, read as
// ResourceText, and the template TemplateURI. NewServer offers none.
func NewResourceServer() *mcp.Server {
	s := NewServer()
	s.AddResource(&mcp.Resource{URI: ResourceURI, Name: "readme", MIMEType: "text/markdown", Description: "The project's readme."},
		func(_ context.Context, r *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: r.Params.URI, MIMEType: "text/markdown", Text: ResourceText}}}, nil
		})
	s.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: TemplateURI, Name: "page", Description: "A page, by number."},
		func(_ context.Context, r *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: r.Params.URI, Text: "page " + r.Params.URI}}}, nil
		})
	return s
}

// The union fixture's tools (0060-MADR).
const (
	// ExecutePipelineTool is the tool that carries ExecutePipelineSchema.
	ExecutePipelineTool = "execute_pipeline"
	// PickOneTool is the tool that carries NullableUnionSchema.
	PickOneTool = "pick_one"
)

// ExecutePipelineSchema is the inputSchema of the execute_pipeline tool of
// mcp-server-magictools, copied verbatim from its internal/handler/inventory.go,
// lines 289-336, at commit 368ac6a: a root anyOf of two required-only
// branches beside a root "type": "object", properties and
// additionalProperties false. mcpclient's testdata holds the same copy as a
// file, and a test there keeps the two equal.
const ExecutePipelineSchema = `{
  "type": "object",
  "properties": {
    "session_id": {
      "type": "string",
      "description": "Optional CSSA session correlation ID. Auto-generated if omitted. For CONTINUATION MODE: pass session_id WITHOUT target/intent to resume a paused pipeline after human approval."
    },
    "target": {
      "type": "string",
      "description": "Absolute path to the project root or package to analyze and refactor."
    },
    "intent": {
      "type": "string",
      "description": "The goal/objective of the pipeline run."
    },
    "plan_hash": {
      "type": "string",
      "description": "Optional SHA-256 hash of an approved implementation plan for MUTATOR integrity verification."
    },
    "dry_run": {
      "type": "boolean",
      "description": "When true, return the DAG plan as markdown without executing. Equivalent to the old compose_pipeline preview."
    },
    "target_roles": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Optional role filter (e.g. ['ANALYZER', 'CRITIC']). Only tools matching these roles enter the DAG."
    },
    "reject": {
      "type": "boolean",
      "description": "Set to true when continuing a paused pipeline to reject the plan and abort without mutations."
    }
  },
  "additionalProperties": false,
  "anyOf": [
    {
      "required": [
        "target",
        "intent"
      ]
    },
    {
      "required": [
        "session_id"
      ]
    }
  ]
}
`

// NullableUnionSchema is a schema the rule of 0060-MADR D2 does not
// rewrite: a root typed ["object", "null"] over a union of required-only
// branches (the MADR's S8). The go-sdk's Server.AddTool refuses a root that
// is not exactly "object", so the union server offers PickOneTool with an
// object root and swaps this schema in when it lists its tools.
const NullableUnionSchema = `{
  "type": ["object", "null"],
  "properties": {"a": {"type": "string"}, "b": {"type": "string"}},
  "anyOf": [{"required": ["a"]}, {"required": ["b"]}]
}`

// UnionCalls records the arguments each call of ExecutePipelineTool carried
// to a union server, as the server received them.
type UnionCalls struct {
	mu   sync.Mutex
	args []string
}

// Args returns the arguments received, as JSON text, in order.
func (u *UnionCalls) Args() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]string(nil), u.args...)
}

// NewUnionServer is a server whose tools carry root-level unions, sent as
// written: ExecutePipelineTool with ExecutePipelineSchema, and PickOneTool
// with NullableUnionSchema. Both are added through Server.AddTool, so the
// go-sdk validates nothing against them. NewServer's tools are not here.
//
// ExecutePipelineTool is registered with its schema. PickOneTool cannot be:
// AddTool panics unless a root's type is exactly "object", and
// NullableUnionSchema's is ["object", "null"]. It is registered with the
// schema {"type":"object"}, and a receiving middleware swaps
// NullableUnionSchema in, on a copy of the tool, in each tools/list result,
// so that a client is sent the schema as written. The server's own
// registration is never changed.
func NewUnionServer() (*mcp.Server, *UnionCalls) {
	calls := &UnionCalls{}
	s := mcp.NewServer(&mcp.Implementation{Name: "gobble-mcptest-union", Version: "1.0.0"}, nil)
	s.AddTool(&mcp.Tool{Name: ExecutePipelineTool, Description: "Run a pipeline.", InputSchema: jsontext.Value(ExecutePipelineSchema)},
		func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			calls.mu.Lock()
			calls.args = append(calls.args, string(req.Params.Arguments))
			calls.mu.Unlock()
			return text("ran"), nil
		})
	s.AddTool(&mcp.Tool{Name: PickOneTool, Description: "Pick a or b.", InputSchema: jsontext.Value(`{"type":"object"}`)},
		func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return text("picked"), nil
		})
	s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			res, err := next(ctx, method, req)
			list, ok := res.(*mcp.ListToolsResult)
			if err != nil || !ok {
				return res, err
			}
			out := *list
			out.Tools = make([]*mcp.Tool, len(list.Tools))
			for i, t := range list.Tools {
				c := *t
				if c.Name == PickOneTool {
					c.InputSchema = jsontext.Value(NullableUnionSchema)
				}
				out.Tools[i] = &c
			}
			return &out, nil
		}
	})
	return s, calls
}

// UnionHTTPServer serves NewUnionServer over streamable HTTP until the
// test ends, and returns its URL and the calls it receives.
func UnionHTTPServer(t testing.TB) (url string, calls *UnionCalls) {
	t.Helper()
	srv, calls := NewUnionServer()
	return serveHTTP(t, srv, ""), calls
}

// ServeStdioIfAsked serves the fixture on stdin and stdout, then exits,
// when EnvServe is "stdio". Otherwise it returns at once. The process is
// then the fixture, not a test, so it owns its exit code and stderr.
func ServeStdioIfAsked() {
	switch os.Getenv(EnvServe) {
	case "stdio":
		os.Exit(serveStdio()) //nolint:forbidigo // the fixture process's own exit
	case "sleep":
		if os.Getenv(EnvStubborn) == "1" {
			signal.Ignore(syscall.SIGTERM)
		}
		time.Sleep(childLife)
		os.Exit(0) //nolint:forbidigo // as above
	}
}

// startChild starts this binary as a sleeping child, with no stdio, and
// writes both pids to EnvPIDs.
func startChild() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	child := exec.Command(exe) //nolint:gosec,noctx // G204: the test binary itself; the child outlives any context, which is the point
	child.Env = append(os.Environ(), EnvServe+"=sleep")
	if err := child.Start(); err != nil {
		return err
	}
	return os.WriteFile(os.Getenv(EnvPIDs), fmt.Appendf(nil, "%d %d", os.Getpid(), child.Process.Pid), 0o600) //nolint:gosec // G703: the test names the pid file
}

func serveStdio() int {
	if m := os.Getenv(EnvMarker); m != "" {
		if err := os.WriteFile(m, nil, 0o600); err != nil { //nolint:gosec // G703: the test names the marker file
			return 4
		}
	}
	if msg := os.Getenv(EnvFail); msg != "" {
		fmt.Fprintln(os.Stderr, msg) //nolint:forbidigo // the fixture process's own stderr
		return 3
	}
	if d, err := time.ParseDuration(os.Getenv(EnvDelay)); err == nil {
		time.Sleep(d)
	}
	stubborn := os.Getenv(EnvStubborn) == "1"
	if stubborn {
		signal.Ignore(syscall.SIGTERM)
	}
	if stubborn || os.Getenv(EnvOrphan) == "1" {
		if err := startChild(); err != nil {
			return 5
		}
	}
	err := NewServer().Run(context.Background(), &mcp.StdioTransport{})
	if stubborn {
		time.Sleep(childLife) // stdin has ended; a stubborn server stays
	}
	if err != nil {
		return 1
	}
	return 0
}

// StdioServer is the command that runs this test binary as the stdio
// fixture, and the environment that asks it to serve.
func StdioServer(t testing.TB) (command string, env map[string]string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return exe, map[string]string{EnvServe: "stdio"}
}

// HTTPServer serves the fixture over streamable HTTP until the test ends,
// and returns its URL. With a bearer, a request without "Authorization:
// Bearer <bearer>" gets 401.
func HTTPServer(t testing.TB, bearer string) string {
	t.Helper()
	return serveHTTP(t, NewServer(), bearer)
}

// ResourceHTTPServer serves NewResourceServer over streamable HTTP until
// the test ends, and returns its URL.
func ResourceHTTPServer(t testing.TB) string {
	t.Helper()
	return serveHTTP(t, NewResourceServer(), "")
}

func serveHTTP(t testing.TB, srv *mcp.Server, bearer string) string {
	t.Helper()
	var h http.Handler = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
	if bearer != "" {
		next := h
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+bearer {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	ts := httptest.NewServer(h)
	t.Cleanup(func() {
		ts.CloseClientConnections()
		ts.Close()
	})
	return ts.URL
}
