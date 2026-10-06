package mcptest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
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
)

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

// ServeStdioIfAsked serves the fixture on stdin and stdout, then exits,
// when EnvServe is "stdio". Otherwise it returns at once. The process is
// then the fixture, not a test, so it owns its exit code and stderr.
func ServeStdioIfAsked() {
	if os.Getenv(EnvServe) != "stdio" {
		return
	}
	os.Exit(serveStdio()) //nolint:forbidigo // the fixture process's own exit
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
	if err := NewServer().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
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
	srv := NewServer()
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
