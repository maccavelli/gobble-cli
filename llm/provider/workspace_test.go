package provider

import (
	"errors"
	"net/http"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// A Claude key not scoped to a workspace is refused as llm.ErrWorkspace,
// classed "workspace"; with Options.Workspace each request names it in
// anthropic-workspace-id, and a provider other than Claude sends none
// (0035-MADR D5).
func TestWireWorkspace(t *testing.T) {
	refused := &cannedWire{status: http.StatusBadRequest,
		body: `{"type":"error","error":{"type":"invalid_request_error","message":"This API key is not scoped to a workspace; name one with anthropic-workspace-id."}}`}
	p, err := New("claude", Options{APIKey: "k", Model: "claude-haiku-5-5", HTTPClient: &http.Client{Transport: refused}, Retry: &llm.Retry{Off: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := drain(t, p, wireRequest()); !errors.Is(err, llm.ErrWorkspace) || FailureKind(err) != "workspace" {
		t.Fatalf("err %v (%s), want llm.ErrWorkspace", err, FailureKind(err))
	}
	for _, c := range []struct{ id, model, want string }{
		{"claude", "claude-haiku-5-5", "wrkspc_1"},
		{"openai", "gpt-6-luna", ""},
	} {
		w := &cannedWire{}
		p, err := New(c.id, Options{APIKey: "k", Model: c.model, Workspace: "wrkspc_1", HTTPClient: &http.Client{Transport: w},
			Retry: &llm.Retry{Off: true}})
		if err != nil {
			t.Fatal(err)
		}
		drain(t, p, wireRequest()) //nolint:errcheck // only the request's header matters
		w.mu.Lock()
		headers := w.headers
		w.mu.Unlock()
		if len(headers) == 0 || headers[0].Get("anthropic-workspace-id") != c.want {
			t.Errorf("%s: anthropic-workspace-id %q, want %q", c.id, firstHeader(headers), c.want)
		}
	}
}

// firstHeader is the first request's anthropic-workspace-id, or "" with no
// request.
func firstHeader(hs []http.Header) string {
	if len(hs) == 0 {
		return ""
	}
	return hs[0].Get("anthropic-workspace-id")
}
