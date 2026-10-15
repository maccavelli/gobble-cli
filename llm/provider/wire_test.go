package provider

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
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

	mu      sync.Mutex
	paths   []string
	bodies  []string
	headers []http.Header
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
	c.headers = append(c.headers, r.Header.Clone())
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

// models.json's baseUrl replaces the provider's base, the wire's path after
// it (0005-PLAN F4-2b).
func TestWireBaseURL(t *testing.T) {
	for _, c := range []struct{ id, model, base, path string }{
		{"openai", "gpt-6-luna", "https://proxy.example/custom/v1/", "/custom/v1/responses"},
		{"claude", "claude-haiku-5-5", "https://proxy.example/anthropic/v1", "/anthropic/v1/messages"},
		{"huggingface", "deepseek-ai/DeepSeek-V4-Flash-0731", "https://proxy.example/hf", "/hf/chat/completions"},
	} {
		t.Run(c.id, func(t *testing.T) {
			w := &cannedWire{}
			p, err := New(c.id, Options{APIKey: "k", Model: c.model, HTTPClient: &http.Client{Transport: w}, BaseURL: c.base})
			if err != nil {
				t.Fatal(err)
			}
			req := wireRequest()
			req.Model = c.model
			for _, err := range p.Stream(t.Context(), req) {
				if err != nil {
					t.Fatal(err)
				}
			}
			if paths, _ := w.requests(); len(paths) != 1 || paths[0] != c.path {
				t.Errorf("paths %q, want %q", paths, c.path)
			}
		})
	}
}

// models.json's headers reach each request, resolved for the request's
// model; a header that cannot be resolved ends the request unsent, and is
// not retried (0005-PLAN F4-2b, deviation 1).
func TestWireHeaders(t *testing.T) {
	var calls atomic.Int32
	headers := func(_ context.Context, model string) (map[string]string, error) {
		calls.Add(1)
		if model == "bad" {
			return nil, errors.New("the environment variable TOKEN is not set")
		}
		return map[string]string{"X-Gobble-Test": "for " + model}, nil
	}
	w := &cannedWire{}
	p, err := New("openai", Options{APIKey: "k", Model: "gpt-6-luna", HTTPClient: &http.Client{Transport: w}, Headers: headers})
	if err != nil {
		t.Fatal(err)
	}
	stream := func(model string) error {
		req := wireRequest()
		req.Model = model
		for _, err := range p.Stream(t.Context(), req) {
			if err != nil {
				return err
			}
		}
		return nil
	}
	if err := stream("other-model"); err != nil {
		t.Fatal(err)
	}
	if w.headers[0].Get("X-Gobble-Test") != "for other-model" {
		t.Errorf("header %q, want the request's model's", w.headers[0].Get("X-Gobble-Test"))
	}
	calls.Store(0)
	err = stream("bad")
	if err == nil || !strings.Contains(err.Error(), "TOKEN is not set") {
		t.Fatalf("err %v, want the header's failure", err)
	}
	if paths, _ := w.requests(); len(paths) != 1 || calls.Load() != 1 {
		t.Errorf("%d requests reached the wire and Headers ran %d times; want none sent and no retry", len(paths)-1, calls.Load())
	}
}

// A key the provider's response refuses is llm.ErrAuth naming the provider
// and where the key came from; with no source, or for a permission refusal
// (403), nothing is added (0005-PLAN F4-3a decision 9).
func TestWireRejectedKey(t *testing.T) {
	const rejected = `{"error":{"message":"Incorrect API key provided.","type":"invalid_request_error","code":"invalid_api_key"}}`
	stream := func(source string, status int) error {
		w := &cannedWire{status: status, body: rejected}
		p, err := New("openai", Options{APIKey: "k", CredentialSource: source, Model: "gpt-6-luna", HTTPClient: &http.Client{Transport: w}})
		if err != nil {
			t.Fatal(err)
		}
		for _, err := range p.Stream(t.Context(), wireRequest()) {
			if err != nil {
				return err
			}
		}
		return nil
	}
	if err := stream("auth.json", http.StatusUnauthorized); !errors.Is(err, llm.ErrAuth) ||
		!strings.HasPrefix(err.Error(), "openai rejected the API key from auth.json: ") {
		t.Errorf("401 with a source: %v, want llm.ErrAuth naming openai and auth.json", err)
	}
	if err := stream("", http.StatusUnauthorized); !errors.Is(err, llm.ErrAuth) || strings.Contains(err.Error(), "rejected the API key") {
		t.Errorf("401 with no source: %v, want llm.ErrAuth alone", err)
	}
	if err := stream("auth.json", http.StatusForbidden); err == nil || strings.Contains(err.Error(), "rejected the API key") {
		t.Errorf("403: %v, want a refusal that does not say the key was rejected", err)
	}
}

// loginWire answers a refresh at /oauth2/token with refresh, or with
// invalid_grant when refresh is "", the first API request with a 401, and
// every later one through c. It records each API request's bearer.
type loginWire struct {
	c       *cannedWire
	refresh string

	mu        sync.Mutex
	bearers   []string
	refreshes int
}

func (w *loginWire) RoundTrip(r *http.Request) (*http.Response, error) {
	reply := func(status int, body string) (*http.Response, error) {
		if r.Body != nil {
			_ = r.Body.Close() //nolint:errcheck // a canned reply reads nothing of the request
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}
	w.mu.Lock()
	if r.URL.Path == "/oauth2/token" {
		w.refreshes++
		w.mu.Unlock()
		if w.refresh == "" {
			return reply(http.StatusBadRequest, `{"error":"invalid_grant"}`)
		}
		return reply(http.StatusOK, w.refresh)
	}
	w.bearers = append(w.bearers, r.Header.Get("Authorization"))
	first := len(w.bearers) == 1
	w.mu.Unlock()
	if first {
		return reply(http.StatusUnauthorized, `{"error":{"message":"token expired","type":"invalid_request_error"}}`)
	}
	return w.c.RoundTrip(r)
}

// A provider built from a stored login sends its access token; a 401
// refreshes it once under the refresh lock, saves the rotation and retries;
// a refused refresh is LoginRejected's error; a key wins over the login
// (0005-PLAN F4-3b decisions 4, 5 and 8).
func TestWireLogin(t *testing.T) {
	stream := func(w *loginWire, store *memLogins, key string) wireOutcome {
		t.Helper()
		p, err := New("grok", Options{APIKey: key, Model: "grok-4.6", Logins: store, CredentialSource: "auth.json",
			HTTPClient: &http.Client{Transport: w}})
		if err != nil {
			t.Fatal(err)
		}
		var o wireOutcome
		for ev, err := range p.Stream(t.Context(), wireRequest()) {
			if err != nil {
				o.err = err
				break
			}
			if ev, ok := ev.(llm.TextDelta); ok {
				o.text += ev.Text
			}
		}
		return o
	}
	login := func() *memLogins {
		return &memLogins{logins: map[string]*Login{"grok": {Access: "access-0", Refresh: "refresh-0",
			Expires: time.Now().Add(time.Hour), Issuer: "https://auth.x.ai", ClientID: "cid", TokenURL: "https://auth.test/oauth2/token"}}}
	}

	store := login()
	w := &loginWire{c: &cannedWire{}, refresh: `{"access_token":"access-1","refresh_token":"refresh-1","expires_in":3600}`}
	if o := stream(w, store, ""); o.err != nil || o.text != "hello" {
		t.Fatalf("stream: %q, %v", o.text, o.err)
	}
	if !slices.Equal(w.bearers, []string{"Bearer access-0", "Bearer access-1"}) || w.refreshes != 1 {
		t.Errorf("bearers %q after %d refreshes; want the stored token, then the refreshed one", w.bearers, w.refreshes)
	}
	if l := store.logins["grok"]; l == nil || l.Access != "access-1" || l.Refresh != "refresh-1" || store.locks == 0 {
		t.Errorf("saved %v under %d refresh locks; want the rotation, under the lock", l, store.locks)
	}

	refused := &loginWire{c: &cannedWire{}}
	o := stream(refused, login(), "")
	if !errors.Is(o.err, llm.ErrAuth) || !strings.HasPrefix(o.err.Error(), LoginRejected("grok")+": ") {
		t.Errorf("a refused refresh: %v, want llm.ErrAuth led by %q", o.err, LoginRejected("grok"))
	}

	keyed := &loginWire{c: &cannedWire{}, refresh: `{"access_token":"never"}`}
	stream(keyed, login(), "k")
	if len(keyed.bearers) == 0 || keyed.bearers[0] != "Bearer k" {
		t.Errorf("with a key: bearers %q, want the key's", keyed.bearers)
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
