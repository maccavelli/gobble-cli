//go:build live_openai

package provider_test

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// recorder keeps the request path and a copy of the response body.
type recorder struct {
	base http.RoundTripper
	path string
	body bytes.Buffer
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	r.path = req.URL.Path
	resp.Body = struct {
		io.Reader
		io.Closer
	}{io.TeeReader(resp.Body, &r.body), resp.Body}
	return resp, nil
}

// echoed is the masked key an OpenAI 401 repeats back.
var echoed = regexp.MustCompile(`(?i)(api key provided:)\s*\S+`)

// redact removes the key, its last four characters and OpenAI's masked echo
// of it from an error's text.
func redact(s, key string) string {
	s = strings.ReplaceAll(s, key, "[redacted]")
	if len(key) > 8 {
		s = strings.ReplaceAll(s, key[len(key)-4:], "[redacted]")
	}
	return echoed.ReplaceAllString(s, "$1 [redacted]")
}

// TestStrictProbe sends gobble's built-in tools to an OpenAI Responses model
// through llm/provider, and logs the strict value the reply reports for each
// (0012-MADR D3, Probe D). It records; it does not judge.
func TestStrictProbe(t *testing.T) {
	key := llmtest.LiveCredential(t, "OPENAI_API_KEY")
	rec := &recorder{base: http.DefaultTransport}
	p, err := provider.New("openai", provider.Options{APIKey: key, Model: "gpt-4.1-mini", HTTPClient: &http.Client{Transport: rec}})
	if err != nil {
		t.Fatal(redact(err.Error(), key))
	}
	var specs []tool.Spec
	for _, tl := range builtin.Tools() {
		specs = append(specs, tl.Spec())
	}
	tools, err := json.Marshal(specs)
	if err != nil {
		t.Fatal(err)
	}
	req := &llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "Reply with the word ok. Do not call a tool."}}}},
		Tools:    tools,
	}
	for _, err := range p.Stream(t.Context(), req) {
		if err != nil {
			t.Fatalf("stream: %s", redact(err.Error(), key))
		}
	}
	if !strings.HasSuffix(rec.path, "/responses") {
		t.Fatalf("the request went to %s, not the Responses API", rec.path)
	}
	strict, err := strictOf(rec.body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range slices.Sorted(maps.Keys(strict)) {
		t.Logf("strict=%v tool=%s", strict[name], name)
	}
}
