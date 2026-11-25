package provider_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/tool"
)

// emptyThenHello answers a completed Responses stream with no output, which
// the SDK fails as its empty answer, then a "hello" stream.
type emptyThenHello struct {
	mu sync.Mutex
	n  int
}

func sseEvent(name, data string) string { return "event: " + name + "\ndata: " + data + "\n\n" }

func (e *emptyThenHello) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, r.Body) //nolint:errcheck // not read here
	}
	e.mu.Lock()
	e.n++
	first := e.n == 1
	e.mu.Unlock()
	body := sseEvent("response.completed", `{"type":"response.completed","response":{"id":"r","status":"completed","output":[],"usage":{"input_tokens":12,"output_tokens":0,"total_tokens":12}}}`)
	if !first {
		body = sseEvent("response.output_text.delta", `{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"hello"}`) +
			sseEvent("response.output_item.done", `{"type":"response.output_item.done","output_index":0,"item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}}`) +
			sseEvent("response.completed", `{"type":"response.completed","response":{"id":"r","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":12,"output_tokens":3,"total_tokens":15}}}`)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body)),
		Request: r}, nil
}

// kept is an slog.Handler that keeps each "model call" record's outcome and
// kind.
type kept struct {
	mu   sync.Mutex
	recs []string
}

func (k *kept) Enabled(context.Context, slog.Level) bool { return true }
func (k *kept) Handle(_ context.Context, r slog.Record) error {
	if r.Message != "model call" {
		return nil
	}
	s := ""
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "outcome" || a.Key == "kind" {
			s += a.Key + "=" + a.Value.String() + " "
		}
		return true
	})
	k.mu.Lock()
	defer k.mu.Unlock()
	k.recs = append(k.recs, strings.TrimSpace(s))
	return nil
}
func (k *kept) WithAttrs([]slog.Attr) slog.Handler { return k }
func (k *kept) WithGroup(string) slog.Handler      { return k }

// The turn's reruns are each their own record: an empty answer, run again
// after its wait, is an error record of kind empty_answer, then the rerun's
// done (0070-MADR D20; T15).
func TestModelCallRecordPerRerun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		k := &kept{}
		p, err := provider.New("openai", provider.Options{APIKey: "k", Model: "gpt-6-luna", HTTPClient: &http.Client{Transport: &emptyThenHello{}},
			Logger: slog.New(k), Retry: &llm.Retry{Off: false}})
		if err != nil {
			t.Fatal(err)
		}
		a := agent.New(agent.Config{Provider: p, Model: "gpt-6-luna", Retry: &llm.Retry{}})
		prompt := llm.Message{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "hi"}}}
		for _, err := range a.Run(t.Context(), tool.Env{}, nil, prompt) {
			if err != nil {
				t.Fatal(err)
			}
		}
		k.mu.Lock()
		defer k.mu.Unlock()
		if len(k.recs) != 2 || k.recs[0] != "outcome=error kind=empty_answer" || k.recs[1] != "outcome=done" {
			t.Fatalf("records %q, want an empty_answer error, then done", k.recs)
		}
	})
}
