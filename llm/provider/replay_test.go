package provider

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"encoding/json/jsontext"

	"github.com/maccavelli/gobble-cli/llm"
)

var sig = jsontext.Value(`{"signature":"sig-A"}`)

// fromA is an assistant message provider a's model ma wrote: thinking with a
// signature, text, and a tool call with a signature.
func fromA(failed bool) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, Origin: llm.Origin{Provider: "a", Model: "ma", Failed: failed}, Content: []llm.Content{
		{Type: llm.ContentThinking, Text: "a thinks", ProviderData: sig},
		{Type: llm.ContentText, Text: "a says", ProviderData: sig},
		{Type: llm.ContentToolCall, ToolCallID: "c1", ToolName: "read", ToolArgs: "{}", ProviderData: sig},
	}}
}

func result(id string) llm.Message {
	return llm.Message{Role: llm.RoleTool, Content: []llm.Content{{Type: llm.ContentToolResult, ToolCallID: id, ToolName: "read", Text: "ok"}}}
}

func user(text string) llm.Message {
	return llm.Message{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: text}}}
}

// kinds lists each message as role:type,type…
func kinds(msgs []llm.Message) []string {
	var out []string
	for _, m := range msgs {
		var ts []string
		for _, c := range m.Content {
			ts = append(ts, c.Type)
		}
		out = append(out, string(m.Role)+":"+strings.Join(ts, ","))
	}
	return out
}

func hasProviderData(msgs []llm.Message) bool {
	for _, m := range msgs {
		for _, c := range m.Content {
			if len(c.ProviderData) > 0 {
				return true
			}
		}
	}
	return false
}

// To the model that wrote it, a message is sent whole, signatures included
// (0025-MADR D2).
func TestReplaySameModel(t *testing.T) {
	in := []llm.Message{user("q"), fromA(false), result("c1")}
	got := replay(in, "a", "ma")
	if !slices.Equal(kinds(got), kinds(in)) || !hasProviderData(got) {
		t.Fatalf("same model: %q, want %q with its signatures", kinds(got), kinds(in))
	}
}

// To another model, of another provider or the same one, the message's
// thinking is dropped and its text and tool calls lose their signatures;
// tool results are kept (0025-MADR D2, Q1).
func TestReplayOtherModel(t *testing.T) {
	in := []llm.Message{user("q"), fromA(false), result("c1")}
	for _, to := range [][2]string{{"b", "mb"}, {"a", "ma-2"}} {
		got := replay(in, to[0], to[1])
		want := []string{"user:text", "assistant:text,tool_call", "tool:tool_result"}
		if !slices.Equal(kinds(got), want) || hasProviderData(got) {
			t.Errorf("to %s/%s: %q with provider data %v, want %q with none", to[0], to[1], kinds(got), hasProviderData(got), want)
		}
		if got[1].Content[0].Text != "a says" || got[1].Content[1].ToolCallID != "c1" {
			t.Errorf("to %s/%s: the text or the call changed: %+v", to[0], to[1], got[1])
		}
	}
	if !hasProviderData(in) {
		t.Fatal("replay changed its input")
	}
}

// An errored or aborted turn is not replayed, nor the results of its calls;
// a call left with no result gets "No result provided" before the next
// message, and at the end (0025-MADR D2).
func TestReplayFailedAndOrphaned(t *testing.T) {
	in := []llm.Message{user("q"), fromA(true), result("c1"), user("again"), fromA(false), user("and")}
	got := replay(in, "a", "ma")
	want := []string{"user:text", "user:text", "assistant:thinking,text,tool_call", "tool:tool_result", "user:text"}
	if !slices.Equal(kinds(got), want) {
		t.Fatalf("got %q, want %q", kinds(got), want)
	}
	if r := got[3].Content[0]; r.ToolCallID != "c1" || r.Text != "No result provided" || !r.IsError {
		t.Fatalf("the synthetic result is %+v", r)
	}
	end := replay([]llm.Message{user("q"), fromA(false)}, "a", "ma")
	if k := kinds(end); len(k) != 3 || k[2] != "tool:tool_result" {
		t.Fatalf("a call at the end: %q, want a synthetic result last", k)
	}
}

// A message whose origin is unknown, from before 0025, is sent as it was.
func TestReplayUnknownOrigin(t *testing.T) {
	m := fromA(false)
	m.Origin = llm.Origin{}
	got := replay([]llm.Message{user("q"), m, result("c1")}, "b", "mb")
	if !hasProviderData(got) || len(got[1].Content) != 3 {
		t.Fatalf("unknown origin: %q, want it whole", kinds(got))
	}
}

// refuseFirst answers its first request 400 with body, then as next does.
type refuseFirst struct {
	next *cannedWire
	body string
	mu   sync.Mutex
	n    int
}

func (r *refuseFirst) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.n++
	first := r.n == 1
	r.mu.Unlock()
	if !first {
		return r.next.RoundTrip(req)
	}
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	r.next.mu.Lock()
	r.next.paths, r.next.bodies = append(r.next.paths, req.URL.Path), append(r.next.bodies, string(b))
	r.next.mu.Unlock()
	return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(r.body)), Request: req}, nil
}

// signed is a conversation whose assistant turn came from provider/model,
// with signed thinking.
func signed(provider, model string) *llm.Request {
	return &llm.Request{Messages: []llm.Message{text(llm.RoleUser, "q"),
		{Role: llm.RoleAssistant, Origin: llm.Origin{Provider: provider, Model: model}, Content: []llm.Content{
			{Type: llm.ContentThinking, Text: "a thinks", ProviderData: jsontext.Value(`{"signature":"sig-A","format":"messages"}`)},
			{Type: llm.ContentText, Text: "a says"}}},
		text(llm.RoleUser, "q2")}}
}

// Through the adapter, another model's signed thinking never reaches the
// request; the same model's does (0025-MADR D2).
func TestWireReplay(t *testing.T) {
	c := &cannedWire{}
	if o := wireStream(t, "claude", "claude-haiku-5-5", c, signed("openai", "gpt-6-luna")); o.err != nil {
		t.Fatal(o.err)
	}
	c2 := &cannedWire{}
	if o := wireStream(t, "claude", "claude-haiku-5-5", c2, signed("claude", "claude-haiku-5-5")); o.err != nil {
		t.Fatal(o.err)
	}
	_, foreignBodies := c.requests()
	_, sameBodies := c2.requests()
	if strings.Contains(foreignBodies[0], "sig-A") || strings.Contains(foreignBodies[0], "a thinks") || !strings.Contains(foreignBodies[0], "a says") {
		t.Fatalf("another model's thinking was sent: %s", foreignBodies[0])
	}
	if !strings.Contains(sameBodies[0], "sig-A") {
		t.Fatalf("the same model's signature was not sent: %s", sameBodies[0])
	}
}

// A request refused over a thinking signature is sent once more with no
// thinking, and answers; any other 400 is not retried (0025-MADR Q4).
func TestWireSignatureRetry(t *testing.T) {
	c := &cannedWire{}
	rt := &refuseFirst{next: c, body: `{"type":"error","error":{"type":"invalid_request_error","message":"messages.1.content.0: Invalid ` + "`signature`" + ` in ` + "`thinking`" + ` block"}}`}
	p, err := New("claude", Options{APIKey: "k", Model: "claude-haiku-5-5", HTTPClient: &http.Client{Transport: rt}})
	if err != nil {
		t.Fatal(err)
	}
	req := signed("claude", "claude-haiku-5-5")
	req.Model = "claude-haiku-5-5"
	var b strings.Builder
	for ev, err := range p.Stream(t.Context(), req) {
		if err != nil {
			t.Fatalf("the retried request failed: %v", err)
		}
		if d, ok := ev.(llm.TextDelta); ok {
			b.WriteString(d.Text)
		}
	}
	said := b.String()
	_, bodies := c.requests()
	if said != "hello" || len(bodies) != 2 || !strings.Contains(bodies[0], "sig-A") || strings.Contains(bodies[1], "sig-A") || strings.Contains(bodies[1], "a thinks") {
		t.Fatalf("said %q after %d requests: %q", said, len(bodies), bodies)
	}

	other := &refuseFirst{next: &cannedWire{}, body: `{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: too large"}}`}
	p, err = New("claude", Options{APIKey: "k", Model: "claude-haiku-5-5", HTTPClient: &http.Client{Transport: other}})
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range p.Stream(t.Context(), req) {
		if err == nil {
			t.Fatal("another 400 was answered")
		}
		break
	}
	if other.n != 1 {
		t.Fatalf("another 400 was sent %d times, want 1", other.n)
	}
}

// stripThinking removes every thinking block and every signature, and
// signatureRefused knows the refusals (0025-MADR Q4).
func TestStripThinkingAndSignatureRefused(t *testing.T) {
	got := stripThinking([]llm.Message{user("q"), fromA(false)})
	if want := []string{"user:text", "assistant:text,tool_call"}; !slices.Equal(kinds(got), want) || hasProviderData(got) {
		t.Fatalf("stripped: %q, provider data %v", kinds(got), hasProviderData(got))
	}
	// The providers' own words, capitalized as they send them.
	openaiText, geminiText := "The encrypted content for item rs_1 could not be verified.", "Thought signature is not valid."
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{&llm.APIError{Status: 400, Err: errors.New("messages.1.content.0: Invalid `signature` in `thinking` block")}, true},
		{&llm.APIError{Status: 400, Err: errors.New(openaiText)}, true},
		{&llm.APIError{Status: 400, Err: errors.New(geminiText)}, true},
		{&llm.APIError{Status: 400, Err: errors.New("max_tokens is too large")}, false},
		{&llm.APIError{Status: 401, Err: fmt.Errorf("%w: signature", llm.ErrAuth)}, false},
		{errors.New("invalid signature"), false},
	} {
		if got := signatureRefused(tc.err); got != tc.want {
			t.Errorf("signatureRefused(%v) = %v", tc.err, got)
		}
	}
}
