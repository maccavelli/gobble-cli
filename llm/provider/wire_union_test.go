package provider

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/mcpclient"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
	"github.com/maccavelli/gobble-cli/tool"
)

// xaiWire is an in-process xAI endpoint for tool schemas (0060-MADR P1, X1):
// it answers 400 in xAI's recorded body when a tool's parameters hold a root
// anyOf or oneOf with a branch not typed object, naming the tool, and
// otherwise answers as cannedWire does, with its canned text.
type xaiWire struct {
	inner *cannedWire
}

// refusedTool is the first tool of body whose parameters are a root union
// with a branch that is not typed object, or "".
func refusedTool(body []byte) (string, error) {
	var req struct {
		Tools []struct {
			Name       string         `json:"name"`
			Parameters map[string]any `json:"parameters"`
			Function   struct {
				Name       string         `json:"name"`
				Parameters map[string]any `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", err
	}
	for _, tl := range req.Tools {
		name, params := tl.Name, tl.Parameters
		if params == nil {
			name, params = tl.Function.Name, tl.Function.Parameters
		}
		for _, kw := range []string{"anyOf", "oneOf"} {
			branches, ok := params[kw].([]any)
			if !ok {
				continue
			}
			for _, b := range branches {
				if m, ok := b.(map[string]any); !ok || m["type"] != "object" {
					return name, nil
				}
			}
		}
	}
	return "", nil
}

func (x *xaiWire) RoundTrip(r *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	name, err := refusedTool(body)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return x.inner.RoundTrip(r)
	}
	x.inner.mu.Lock()
	x.inner.paths = append(x.inner.paths, r.URL.Path)
	x.inner.bodies = append(x.inner.bodies, string(body))
	x.inner.mu.Unlock()
	refusal := fmt.Sprintf(`{"code":"invalid-argument","error":"Failed to start sampling: [invalid_client_tool_schema] %s: `+
		`tool parameter root must be an object type (root schema is an anyOf/oneOf union with a non-object branch)"}`, name)
	return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(refusal)), Request: r}, nil
}

// toolsOf is a tools array of one tool, named name, with schema as its
// input schema, compacted.
func toolsOf(t *testing.T, name string, schema []byte) jsontext.Value {
	t.Helper()
	c := jsontext.Value(bytes.Clone(schema))
	if err := c.Compact(); err != nil {
		t.Fatal(err)
	}
	return jsontext.Value(`[{"name":"` + name + `","description":"Run a pipeline.","inputSchema":` + string(c) + `}]`)
}

// sentParameters is the parameters of the first tool of a request body, as
// the wire carried them.
func sentParameters(t *testing.T, body string) jsontext.Value {
	t.Helper()
	var req struct {
		Tools []struct {
			Parameters jsontext.Value `json:"parameters"`
			Function   struct {
				Parameters jsontext.Value `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal([]byte(body), &req); err != nil || len(req.Tools) != 1 {
		t.Fatalf("%d tools in the body (%v):\n%.600s", len(req.Tools), err, body)
	}
	if p := req.Tools[0].Parameters; len(p) > 0 {
		return p
	}
	return req.Tools[0].Function.Parameters
}

// Through gobble's grok provider, over an endpoint that refuses a root
// union with a branch that is not typed object as xAI does: the real schema
// as the server sends it is refused with the owner's text after one request,
// and the expected rewrite (mcpclient/testdata, 0060-MADR D2) is sent
// unchanged and answered (0060-MADR Confirmation 2).
func TestWireRootUnionSchema(t *testing.T) {
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("..", "..", "mcpclient", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	const name = "mcp__magictools__execute_pipeline"
	run := func(tools jsontext.Value) (wireOutcome, *cannedWire) {
		c := &cannedWire{}
		p, err := New("grok", Options{APIKey: "k", Model: "grok-4.6", HTTPClient: &http.Client{Transport: &xaiWire{inner: c}}})
		if err != nil {
			t.Fatal(err)
		}
		req := wireRequest()
		req.Model, req.Tools = "grok-4.6", tools
		var o wireOutcome
		for ev, err := range p.Stream(t.Context(), req) {
			if err != nil {
				o.err = err
				break
			}
			if d, ok := ev.(llm.TextDelta); ok {
				o.text += d.Text
			}
		}
		return o, c
	}

	// The schema as the server sends it, handed to the provider as is.
	o, c := run(toolsOf(t, name, read("magictools-execute-pipeline.json")))
	const refused = "llm: api status 400: llmprovider: invalid request: grok HTTP 400 invalid-argument: Failed to start sampling: " +
		"[invalid_client_tool_schema] " + name + ": tool parameter root must be an object type " +
		"(root schema is an anyOf/oneOf union with a non-object branch)"
	if o.err == nil || o.err.Error() != refused {
		t.Fatalf("the original schema: %v\nwant %q", o.err, refused)
	}
	oneRequest(t, c)

	// The tool as gobble loads it from that server: the spec's schema is
	// the expected rewrite, and it is sent unchanged and answered.
	spec := loadedSpec(t, "magictools", "execute_pipeline")
	specs := jsontext.Value(nil)
	if b, err := json.Marshal([]tool.Spec{spec}); err != nil {
		t.Fatal(err)
	} else {
		specs = b
	}
	o, c = run(specs)
	if o.err != nil || o.text != "hello" {
		t.Fatalf("the loaded tool: text %q, err %v, want it answered (the schema is\n%s)", o.text, o.err, spec.InputSchema)
	}
	oneRequest(t, c)
	_, bodies := c.requests()
	// The schema on the wire is the spec's, byte for byte: mcpclient writes
	// it with its keys sorted, and so does the SDK.
	sent := sentParameters(t, bodies[0])
	if string(sent) != string(spec.InputSchema) {
		t.Fatalf("the schema on the wire is not the spec's, byte for byte:\n got %s\nwant %s", sent, spec.InputSchema)
	}
	// The hand-indented file is the same schema once both are canonical
	// (RFC 8785).
	canon := func(v jsontext.Value) string {
		v = bytes.Clone(v)
		if err := v.Canonicalize(); err != nil {
			t.Fatal(err)
		}
		return string(v)
	}
	if got, want := canon(sent), canon(read("magictools-execute-pipeline.rewritten.json")); got != want {
		t.Fatalf("the schema on the wire is not the expected rewrite:\n got %s\nwant %s", got, want)
	}
}

// loadedSpec is the spec of one tool of the union fixture, served as server
// name, as mcpclient loads it.
func loadedSpec(t *testing.T, server, name string) tool.Spec {
	t.Helper()
	url, _ := mcptest.UnionHTTPServer(t)
	m := mcpclient.Start(t.Context(), []mcpclient.Server{{Name: server, URL: url, Timeout: mcpclient.DefaultTimeout, Enabled: true}}, mcpclient.Options{})
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
	want := "mcp__" + server + "__" + name
	for _, tl := range m.Tools(nil) {
		if tl.Spec().Name == want {
			return tl.Spec()
		}
	}
	t.Fatalf("no tool %s", want)
	return tool.Spec{}
}
