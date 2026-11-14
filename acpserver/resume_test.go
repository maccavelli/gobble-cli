package acpserver

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// bodyRecorder is an HTTP transport that keeps each request's path and
// body and answers 400, so a provider built over it renders a request and
// sends it nowhere (0037-MADR D42).
type bodyRecorder struct {
	mu     sync.Mutex
	paths  []string
	bodies [][]byte
}

func (r *bodyRecorder) RoundTrip(q *http.Request) (*http.Response, error) {
	b, err := io.ReadAll(q.Body)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.paths = append(r.paths, q.URL.Path)
	r.bodies = append(r.bodies, b)
	r.mu.Unlock()
	return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
		Body:    io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"invalid_request_error","message":"recorded, not sent"}}`)),
		Request: q}, nil
}

// wireCase is a provider and a model id whose request body has one of the
// shapes the providers' wires come in, and the end of the path that shape
// is sent to. The path is asserted, so a model id that a provider's route
// table later sends to another shape fails the test instead of covering a
// shape twice.
type wireCase struct{ shape, provider, model, pathEnd string }

// resumeWires is one wire per body shape of the 16 wires and routes
// measured by 0037-PLAN-resume-prefix-and-failed-turn-history.md P1. Each
// model id is the one least tied to a catalog or a route table:
//   - claude and gemini: a model the provider serves itself, with one wire
//     whatever the id;
//   - openai: the same, for the Responses wire;
//   - ollama: a local engine with no catalog, so the id is free text, and
//     the one chat-completions wire that no catalog, route table or prefix
//     rule reaches;
//   - opencode-zen: the only provider with the Google route, which its
//     embedded route table gives to a Gemini model id; the path check
//     above is what holds that table to this shape.
var resumeWires = []wireCase{
	{"Messages", "claude", "claude-haiku-5-5", "/v1/messages"},
	{"Responses", "openai", "gpt-6-luna", "/v1/responses"},
	{"chat-completions", "ollama", "llama3", "/v1/chat/completions"},
	{"Gemini interactions", "gemini", "gemini-3.8-flash", "/v1beta/interactions"},
	{"OpenCode Google route", "opencode-zen", "gemini-3.5-flash", ":streamGenerateContent"},
}

// wireBodies renders each request through w's real wire and with no retry,
// and returns the bodies it sent.
func wireBodies(t *testing.T, w wireCase, reqs []*llm.Request) [][]byte {
	t.Helper()
	tr := &bodyRecorder{}
	p, err := provider.New(w.provider, provider.Options{APIKey: "not-a-real-key", SessionID: "resume-prefix", HTTPClient: &http.Client{Transport: tr},
		Retry: &llm.Retry{Off: true}})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reqs {
		r := *r
		r.Model = w.model
		msgs := make([]llm.Message, len(r.Messages))
		for i, m := range r.Messages {
			if m.Origin.Provider != "" {
				m.Origin.Provider, m.Origin.Model = w.provider, w.model
			}
			msgs[i] = m
		}
		r.Messages = msgs
		var failure error
		for _, err := range p.Stream(t.Context(), &r) {
			failure = err
		}
		// The recorder's 400 is the only way a request ends here.
		if ae, ok := errors.AsType[*llm.APIError](failure); !ok || ae.Status != http.StatusBadRequest {
			t.Fatalf("%s: the rendered request ended with %v, want the recorder's 400", w.shape, failure)
		}
	}
	if len(tr.bodies) != len(reqs) {
		t.Fatalf("%s: %d requests rendered %d bodies", w.shape, len(reqs), len(tr.bodies))
	}
	for _, path := range tr.paths {
		if !strings.HasSuffix(path, w.pathEnd) {
			t.Fatalf("%s (%s, %s): a request went to %s, want a path ending %s", w.shape, w.provider, w.model, path, w.pathEnd)
		}
	}
	return tr.bodies
}

// withoutMarkers is v with every cache_control member removed: the moving
// marker is not part of the prefix a cache reads.
func withoutMarkers(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, y := range x {
			if k != "cache_control" {
				out[k] = withoutMarkers(y)
			}
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, y := range x {
			out[i] = withoutMarkers(y)
		}
		return out
	}
	return v
}

// sequenceKeys are the members that hold a wire body's message sequence: the
// Messages, chat-completions and Gemini interactions wires' "messages" or
// "input", the Responses wire's "input", and the Google route's "contents".
var sequenceKeys = []string{"messages", "input", "contents"}

// wireShape is a wire body split into its message items, each as the bytes
// it encodes to with its markers removed, and everything else in the body,
// the tools and the system prompt among it, likewise.
func wireShape(t *testing.T, body []byte) (items []string, head string) {
	t.Helper()
	var m map[string]any
	if err := jsonx.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	var list []any
	for _, k := range sequenceKeys {
		if l, ok := m[k].([]any); ok {
			list = l
			delete(m, k)
			break
		}
	}
	if list == nil {
		t.Fatalf("no message sequence in the body %.200s", body)
	}
	for _, it := range list {
		b, err := jsonx.Marshal(withoutMarkers(it))
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, string(b))
	}
	h, err := jsonx.Marshal(withoutMarkers(m))
	if err != nil {
		t.Fatal(err)
	}
	return items, string(h)
}

// A session's requests keep every earlier request of it as a prefix, live
// and after a close and a load, with the cache markers removed: the cache
// reads a request's prefix, and a resumed session that rewrites it pays
// full price once (0037-MADR D42). Four scripted sessions, each with a
// tool call, two replies, a close, a load into a second Agent over the same
// store and one more prompt, are rendered through one wire of each body
// shape (resumeWires):
//   - mention: the prompt's @path adds a files block, which the live
//     message once held as a block of its own and the file holds joined;
//   - two text blocks: the same join, with no mention;
//   - spaced arguments: the model's spacing in the call's arguments, which
//     the file drops;
//   - empty arguments: a call whose argument text is empty, which the file
//     gives back as "{}".
func TestResumeKeepsPrefix(t *testing.T) {
	cases := []struct {
		name   string
		prompt []acp.ContentBlock
		args   string
	}{
		{"mention", []acp.ContentBlock{acp.TextBlock("look at @notes.md please")}, `{"path":"notes.md"}`},
		{"two text blocks", []acp.ContentBlock{acp.TextBlock("first block"), acp.TextBlock("second block")}, `{"path":"notes.md"}`},
		{"spaced arguments", []acp.ContentBlock{acp.TextBlock("read notes.md")}, `{"path": "notes.md"}`},
		{"empty arguments", []acp.ContentBlock{acp.TextBlock("read notes.md")}, ``},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reqs := resumedRequests(t, tc.prompt, tc.args)
			for _, w := range resumeWires {
				checkPrefix(t, w.shape, wireBodies(t, w, reqs))
			}
		})
	}
}

// resumedRequests runs the scripted session of TestResumeKeepsPrefix with
// the given first prompt and the arguments of its one read call, and
// returns the model requests it sent: three before the close, one after the
// load.
func resumedRequests(t *testing.T, first []acp.ContentBlock, args string) []*llm.Request {
	t.Helper()
	dir, cwd := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "notes.md"), []byte("forty-two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	before := llmtest.NewScript(llmtest.ToolCall("toolu_1", "read", args), llmtest.Text("it says forty-two"), llmtest.Text("second answer"))
	c := promptAgent(t, dir, before, PromptOptions{}, &logs)
	sid := newSession(t, c, cwd, nil).SessionId
	if _, err := c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: sid, Prompt: first}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, sid, "and again"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: sid}); err != nil {
		t.Fatal(err)
	}
	after := llmtest.NewScript(llmtest.Text("third answer"))
	c2 := promptAgent(t, dir, after, PromptOptions{}, &logs)
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: sid, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c2, sid, "third"); err != nil {
		t.Fatal(err)
	}
	reqs := append(before.Requests(), after.Requests()...)
	if len(reqs) != 4 {
		t.Fatalf("%d model requests, want 4", len(reqs))
	}
	return reqs
}

// checkPrefix fails for each earlier request whose tools or system differ
// from the last request's, or whose message items are not a prefix of the
// last request's, naming the first item that is not.
func checkPrefix(t *testing.T, id string, bodies [][]byte) {
	t.Helper()
	last, lastHead := wireShape(t, bodies[len(bodies)-1])
	for i, b := range bodies[:len(bodies)-1] {
		items, head := wireShape(t, b)
		if head != lastHead {
			t.Errorf("%s: request %d's tools or system differ from the post-reload request's", id, i)
		}
		for j, it := range items {
			if j < len(last) && it == last[j] {
				continue
			}
			got := "<missing>"
			if j < len(last) {
				got = last[j]
			}
			t.Errorf("%s: request %d's item %d of %d is not the post-reload request's:\n  before: %.160s\n  after:  %.160s", id, i, j, len(items), it, got)
			break
		}
	}
}
