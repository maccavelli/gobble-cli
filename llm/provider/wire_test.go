package provider

import (
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// Canned replies in each wire's shape: "hello", or a get_weather call. Every
// one reports 12 tokens in and 3 out (0005-PLAN F4-1a, decision 3).
const (
	wireResponsesText = `{"id":"r","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":12,"output_tokens":3,"total_tokens":15}}`
	wireResponsesTool = `{"id":"r","status":"completed","output":[{"type":"function_call","call_id":"c1","name":"get_weather","arguments":"{\"city\":\"Paris\"}"}],"usage":{"input_tokens":12,"output_tokens":3,"total_tokens":15}}`
	wireMessagesText  = `{"id":"m","type":"message","role":"assistant","stop_reason":"end_turn","content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":12,"output_tokens":3}}`
	wireMessagesTool  = `{"id":"m","type":"message","role":"assistant","stop_reason":"tool_use","content":[{"type":"tool_use","id":"c1","name":"get_weather","input":{"city":"Paris"}}],"usage":{"input_tokens":12,"output_tokens":3}}`
	wireInterText     = `{"id":"i","status":"completed","steps":[{"type":"model_output","content":[{"type":"text","text":"hello"}]}],"usage":{"total_input_tokens":12,"total_output_tokens":3,"total_tokens":15}}`
	wireInterTool     = `{"id":"i","status":"completed","steps":[{"type":"function_call","id":"c1","name":"get_weather","arguments":{"city":"Paris"}}],"usage":{"total_input_tokens":12,"total_output_tokens":3,"total_tokens":15}}`
	wireChatText      = `{"id":"c","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"hello"}}],"usage":{"prompt_tokens":12,"completion_tokens":3,"total_tokens":15}}`
	wireChatTool      = `{"id":"c","choices":[{"index":0,"finish_reason":"tool_calls","message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]}}],"usage":{"prompt_tokens":12,"completion_tokens":3,"total_tokens":15}}`

	wireRateLimited = `{"error":{"message":"rate limited","type":"rate_limit_error"}}`
	wireOverflow    = `{"error":{"message":"prompt is too long: 300000 tokens > 200000 maximum","type":"invalid_request_error","code":"context_length_exceeded"}}`
	wireSplit       = `{"error":{"message":"This model's maximum context length is 1000 tokens. However, you requested 1100 tokens (100 in the messages, 1000 in the completion).","type":"invalid_request_error"}}`
)

// wireIDs are the ten ids with a model each, as gobble's New builds them.
var wireIDs = []struct{ id, model string }{
	{"openai", "gpt-6-luna"}, {"claude", "claude-haiku-5-5"}, {"gemini", "gemini-3.5-flash-lite"}, {"grok", "grok-4.6"},
	{"opencode-zen", "claude-haiku-5-5"}, {"opencode-go", "claude-haiku-5-5"}, {"huggingface", "deepseek-ai/DeepSeek-V4-Flash-0731"},
	{"kilo", "deepseek/deepseek-v4.1-flash"}, {"together", "zai-org/GLM-5.3-Flash"}, {"ollama", "llama3"},
}

// cannedWire answers each request in process: with status and header when
// status is set, else with the text or tool reply for the request's wire,
// found by its path. It records each request's body. No test here reaches a
// service (0005-PLAN F4-1a, decision 3).
type cannedWire struct {
	status int
	header http.Header
	body   string
	tool   bool

	mu     sync.Mutex
	paths  []string
	bodies []string
}

func (c *cannedWire) RoundTrip(r *http.Request) (*http.Response, error) {
	var body []byte
	if r.Body != nil {
		var err error
		if body, err = io.ReadAll(r.Body); err != nil {
			return nil, err
		}
	}
	c.mu.Lock()
	c.paths = append(c.paths, r.URL.Path)
	c.bodies = append(c.bodies, string(body))
	c.mu.Unlock()
	reply := func(status int, h http.Header, b string) (*http.Response, error) {
		out := http.Header{"Content-Type": {"application/json"}}
		for k, v := range h {
			out[http.CanonicalHeaderKey(k)] = v // as a server's headers arrive
		}
		return &http.Response{StatusCode: status, Header: out, Body: io.NopCloser(strings.NewReader(b)), Request: r}, nil
	}
	if c.status != 0 {
		return reply(c.status, c.header, c.body)
	}
	pick := func(text, tool string) (*http.Response, error) {
		if c.tool {
			return reply(http.StatusOK, nil, tool)
		}
		return reply(http.StatusOK, nil, text)
	}
	switch p := r.URL.Path; {
	case strings.HasSuffix(p, "/responses"):
		return pick(wireResponsesText, wireResponsesTool)
	case strings.HasSuffix(p, "/messages"):
		return pick(wireMessagesText, wireMessagesTool)
	case strings.HasSuffix(p, "/interactions"):
		return pick(wireInterText, wireInterTool)
	case strings.HasSuffix(p, "/chat/completions"):
		return pick(wireChatText, wireChatTool)
	}
	return reply(http.StatusNotFound, nil, `{"error":{"message":"no canned reply for this path"}}`)
}

func (c *cannedWire) requests() ([]string, []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paths, c.bodies
}

// wireOutcome is what one Stream gave: its text, call names, usage and error.
type wireOutcome struct {
	text  string
	calls []string
	usage llm.Usage
	err   error
}

// wireStream streams req through provider id over c.
func wireStream(t *testing.T, id, model string, c *cannedWire, req *llm.Request) wireOutcome {
	t.Helper()
	p, err := New(id, Options{APIKey: "k", Model: model, HTTPClient: &http.Client{Transport: c}})
	if err != nil {
		t.Fatal(err)
	}
	req.Model = model
	var o wireOutcome
	for ev, err := range p.Stream(t.Context(), req) {
		if err != nil {
			o.err = err
			break
		}
		switch ev := ev.(type) {
		case llm.TextDelta:
			o.text += ev.Text
		case llm.ToolCallDone:
			o.calls = append(o.calls, ev.Name+" "+ev.Args)
		case llm.Usage:
			o.usage = ev
		}
	}
	return o
}

func wireRequest() *llm.Request {
	return &llm.Request{Messages: []llm.Message{text(llm.RoleSystem, "be brief"), text(llm.RoleUser, "weather?")}}
}

// oneRequest fails the test unless c saw exactly one request.
func oneRequest(t *testing.T, c *cannedWire) {
	t.Helper()
	if paths, _ := c.requests(); len(paths) != 1 {
		t.Fatalf("%d requests (%q), want 1", len(paths), paths)
	}
}

// Each of the ten ids, through gobble's New: a text reply with its usage, a
// tool call, a 429 past the retry cap returned at once with its wait, and an
// overflow, each after one request (0005-PLAN F4-1, F4-1a).
func TestWireCases(t *testing.T) {
	for _, w := range wireIDs {
		t.Run(w.id+"/text", func(t *testing.T) {
			c := &cannedWire{}
			o := wireStream(t, w.id, w.model, c, wireRequest())
			if o.err != nil || o.text != "hello" || o.usage.InputTokens != 12 || o.usage.OutputTokens != 3 {
				t.Fatalf("text %q usage %+v err %v, want hello with 12 in and 3 out", o.text, o.usage, o.err)
			}
			oneRequest(t, c)
		})
		t.Run(w.id+"/tool", func(t *testing.T) {
			c := &cannedWire{tool: true}
			o := wireStream(t, w.id, w.model, c, wireRequest())
			if o.err != nil || len(o.calls) != 1 || o.calls[0] != `get_weather {"city":"Paris"}` {
				t.Fatalf("calls %q err %v, want get_weather for Paris", o.calls, o.err)
			}
			oneRequest(t, c)
		})
		t.Run(w.id+"/retry-after", func(t *testing.T) {
			c := &cannedWire{status: http.StatusTooManyRequests, header: http.Header{"Retry-After": {"120"}}, body: wireRateLimited}
			o := wireStream(t, w.id, w.model, c, wireRequest())
			var ae *llm.APIError
			if !errors.Is(o.err, llm.ErrRateLimited) || !errors.As(o.err, &ae) || ae.RetryAfter != 120*time.Second {
				t.Fatalf("err %v, want llm.ErrRateLimited with RetryAfter 120s", o.err)
			}
			oneRequest(t, c)
		})
		t.Run(w.id+"/overflow", func(t *testing.T) {
			c := &cannedWire{status: http.StatusBadRequest, body: wireOverflow}
			if o := wireStream(t, w.id, w.model, c, wireRequest()); !errors.Is(o.err, llm.ErrContextOverflow) {
				t.Fatalf("err %v, want llm.ErrContextOverflow", o.err)
			}
			oneRequest(t, c)
		})
	}
}

// A 429 with no Retry-After takes its wait from the provider's reset header:
// a Go duration on OpenAI and Hugging Face, an RFC 3339 time on Claude
// (go-llmprovider-sdk v1.4.0; 0005-PLAN F4-1a).
func TestWireRateLimitResets(t *testing.T) {
	for _, c := range []struct {
		id, model, header string
		value             func() string
		min, max          time.Duration
	}{
		{"openai", "gpt-6-luna", "x-ratelimit-reset-requests", func() string { return "90s" }, 90 * time.Second, 90 * time.Second},
		{"huggingface", "deepseek-ai/DeepSeek-V4-Flash-0731", "x-ratelimit-reset-requests", func() string { return "90s" }, 90 * time.Second, 90 * time.Second},
		{"claude", "claude-haiku-5-5", "anthropic-ratelimit-requests-reset",
			func() string { return time.Now().Add(90 * time.Second).UTC().Format(time.RFC3339) }, 89 * time.Second, 90 * time.Second},
	} {
		t.Run(c.id, func(t *testing.T) {
			w := &cannedWire{status: http.StatusTooManyRequests, header: http.Header{c.header: {c.value()}}, body: wireRateLimited}
			o := wireStream(t, c.id, c.model, w, wireRequest())
			var ae *llm.APIError
			if !errors.Is(o.err, llm.ErrRateLimited) || !errors.As(o.err, &ae) || ae.RetryAfter < c.min || ae.RetryAfter > c.max {
				t.Fatalf("err %v, want llm.ErrRateLimited with RetryAfter in [%s, %s]", o.err, c.min, c.max)
			}
			oneRequest(t, w)
		})
	}
}

// An overflow message that states its split, the completion alone at the
// limit, is an invalid request with its status, not an overflow
// (go-llmprovider-sdk v1.4.0; 0005-PLAN F4-1a).
func TestWireOverflowSplit(t *testing.T) {
	c := &cannedWire{status: http.StatusBadRequest, body: wireSplit}
	o := wireStream(t, "openai", "gpt-6-luna", c, wireRequest())
	var ae *llm.APIError
	if errors.Is(o.err, llm.ErrContextOverflow) || !errors.As(o.err, &ae) || ae.Status != http.StatusBadRequest {
		t.Fatalf("err %v, want an *llm.APIError of status 400 that is not llm.ErrContextOverflow", o.err)
	}
	oneRequest(t, c)
}

// A failed tool result reaches the wire as one where the wire has an error
// form (Messages, Interactions), and as its output text elsewhere
// (0005-PLAN F4-1a).
func TestWireToolResultError(t *testing.T) {
	req := func() *llm.Request {
		call := llm.Message{Role: llm.RoleAssistant, Content: []llm.Content{{Type: llm.ContentToolCall, ToolCallID: "c1", ToolName: "get_weather", ToolArgs: `{"city":"Paris"}`}}}
		res := llm.Message{Role: llm.RoleTool, Content: []llm.Content{{Type: llm.ContentToolResult, ToolCallID: "c1", Text: "no such city", IsError: true}}}
		return &llm.Request{Messages: []llm.Message{text(llm.RoleSystem, "be brief"), text(llm.RoleUser, "weather?"), call, res}}
	}
	for _, c := range []struct {
		id, model string
		flagged   bool
	}{
		{"claude", "claude-haiku-5-5", true},
		{"gemini", "gemini-3.5-flash-lite", true},
		{"opencode-zen", "claude-haiku-5-5", true},
		{"openai", "gpt-6-luna", false},
		{"huggingface", "deepseek-ai/DeepSeek-V4-Flash-0731", false},
	} {
		t.Run(c.id, func(t *testing.T) {
			w := &cannedWire{}
			if o := wireStream(t, c.id, c.model, w, req()); o.err != nil {
				t.Fatal(o.err)
			}
			_, bodies := w.requests()
			if len(bodies) != 1 {
				t.Fatalf("%d requests, want 1", len(bodies))
			}
			if got := strings.Contains(bodies[0], `"is_error":true`); got != c.flagged || !strings.Contains(bodies[0], "no such city") {
				t.Fatalf("body flags the failure %v (want %v), or lacks the output:\n%s", got, c.flagged, bodies[0])
			}
		})
	}
}

// Each built-in tool's strict flag reaches every wire that carries one, and
// none reaches Gemini's or Ollama's; true is never sent (0012-PLAN P7).
func TestWireStrict(t *testing.T) {
	var specs []tool.Spec
	for _, tl := range builtin.Tools() {
		specs = append(specs, tl.Spec())
	}
	tools, err := json.Marshal(specs)
	if err != nil {
		t.Fatal(err)
	}
	req := func() *llm.Request {
		r := wireRequest()
		r.Tools = tools
		return r
	}
	sreq, err := toRequest(req())
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, tl := range sreq.Tools {
		if tl.Strict != nil && !*tl.Strict {
			n++
		}
	}
	if n == 0 {
		t.Fatal("no built-in tool is sent with strict false")
	}
	for _, w := range wireIDs {
		t.Run(w.id, func(t *testing.T) {
			c := &cannedWire{}
			if o := wireStream(t, w.id, w.model, c, req()); o.err != nil {
				t.Fatal(o.err)
			}
			oneRequest(t, c)
			_, bodies := c.requests()
			want := n
			if w.id == "gemini" || w.id == "ollama" {
				want = 0
			}
			body := bodies[0]
			if got := strings.Count(body, `"strict":false`); got != want || !strings.Contains(body, `"name":"read"`) {
				t.Errorf(`%d "strict":false, want %d, or no read tool, in:\n%.400s`, got, want, body)
			}
			if strings.Contains(body, `"strict":true`) {
				t.Errorf(`a "strict":true reached the wire:\n%.400s`, body)
			}
		})
	}
}
