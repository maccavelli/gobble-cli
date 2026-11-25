package cli

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/logging"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// downThenHello answers a 503, then the Responses wire's "hello" stream.
type downThenHello struct {
	mu sync.Mutex
	n  int
}

func (d *downThenHello) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, r.Body) //nolint:errcheck // not read here
	}
	d.mu.Lock()
	d.n++
	first := d.n == 1
	d.mu.Unlock()
	if first {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"error":{"message":"overloaded","type":"server_error"}}`)), Request: r}, nil
	}
	sse := func(name, data string) string { return "event: " + name + "\ndata: " + data + "\n\n" }
	body := sse("response.output_text.delta", `{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"hello"}`) +
		sse("response.output_item.done", `{"type":"response.output_item.done","output_index":0,"item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}}`) +
		sse("response.completed", `{"type":"response.completed","response":{"id":"r","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":12,"output_tokens":3,"total_tokens":15}}}`)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body)),
		Request: r}, nil
}

// Through the real log file's handler, which hides a key that names a
// credential, every field of the model call record keeps its value
// (0070-MADR D17, F14; T12).
func TestModelCallRecordSurvivesRedaction(t *testing.T) {
	file := filepath.Join(t.TempDir(), "gobble.log")
	logger, closer, err := logging.Open(logging.Options{Level: slog.LevelDebug, File: file})
	if err != nil {
		t.Fatal(err)
	}
	p, err := provider.New("openai", provider.Options{APIKey: "k", Model: "gpt-6-luna", HTTPClient: &http.Client{Transport: &downThenHello{}},
		Logger: logger, SessionID: "s1", Retry: &llm.Retry{MaxRetries: 2, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}})
	if err != nil {
		t.Fatal(err)
	}
	req := &llm.Request{Model: "gpt-6-luna", MaxTokens: 4096, Messages: []llm.Message{{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "hi"}}}}}
	for _, err := range p.Stream(t.Context(), req) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var rec map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(string(b)), "\n") {
		var m map[string]any
		if jsonx.Unmarshal([]byte(line), &m) == nil && m["msg"] == "model call" {
			rec = m
		}
	}
	if rec == nil {
		t.Fatalf("no model call record in %s", b)
	}
	for k, v := range rec {
		if v == "[REDACTED]" {
			t.Errorf("%s was redacted", k)
		}
	}
	kinds, _ := rec["retry_kinds"].([]any)
	if len(kinds) != 1 || kinds[0] != "unavailable" || rec["session"] != "s1" || rec["outcome"] != "done" {
		t.Fatalf("record %v, want retry_kinds [unavailable], session s1, outcome done", rec)
	}
}
