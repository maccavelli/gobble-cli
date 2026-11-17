//go:build live_grok

package provider_test

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// TestLiveGrokUnionSchema is the owner's step of 0060-PLAN P3: it sends one
// short prompt to xAI's grok-4.6 twice, each with one tool, the magictools
// execute_pipeline tool, once with the schema as the server sends it
// (mcpclient/testdata) and once with the schema 0060-MADR D2 makes of it. It
// fails when the rewrite is refused; whether the original is refused it only
// logs, with the status. Nothing but statuses and the model's reply is
// printed, and the key is redacted from every failure.
func TestLiveGrokUnionSchema(t *testing.T) {
	key := llmtest.LiveCredential(t, "XAI_API_KEY")
	p, err := provider.New("grok", provider.Options{APIKey: key, Model: "grok-4.6", ClientName: "gobble-live-test"})
	if err != nil {
		t.Fatal(liveRedact(err.Error(), key))
	}
	// turn sends the prompt with the tool whose schema is in file, and
	// returns the HTTP status of a refusal (0 when it is not an API error),
	// and the error, or nil when the model answered.
	turn := func(file string) (int, error) {
		schema, err := os.ReadFile(filepath.Join("..", "..", "mcpclient", "testdata", file))
		if err != nil {
			t.Fatal(err)
		}
		c := jsontext.Value(bytes.Clone(schema))
		if err := c.Compact(); err != nil {
			t.Fatal(err)
		}
		req := &llm.Request{
			Model:    "grok-4.6",
			Messages: []llm.Message{{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "Reply with the single word: ok. Do not call a tool."}}}},
			Tools:    jsontext.Value(`[{"name":"mcp__magictools__execute_pipeline","description":"Run a pipeline.","inputSchema":` + string(c) + `}]`),
		}
		for _, err := range p.Stream(t.Context(), req) {
			if err != nil {
				if ae, ok := errors.AsType[*llm.APIError](err); ok {
					return ae.Status, err
				}
				return 0, err
			}
		}
		return 0, nil
	}

	status, err := turn("magictools-execute-pipeline.json")
	switch {
	case err == nil:
		t.Logf("the schema as the server sends it: accepted")
	case status != 0:
		t.Logf("the schema as the server sends it: refused, status %d", status)
	default:
		t.Fatalf("the schema as the server sends it: %s", liveRedact(err.Error(), key))
	}

	status, err = turn("magictools-execute-pipeline.rewritten.json")
	if err != nil {
		t.Fatalf("the rewritten schema was refused (status %d): %s", status, liveRedact(err.Error(), key))
	}
	t.Logf("the rewritten schema: accepted")
}
