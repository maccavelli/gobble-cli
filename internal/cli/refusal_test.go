package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// refusalTurn is a turn whose response warns reasoning_learned for model m
// at high, then answers.
func refusalTurn() llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{
		llm.Warning{
			Kind:    llm.WarningReasoningLearned,
			Model:   "m",
			Refused: llm.ReasoningRefusal{Model: "m", Effort: "high"},
		},
		llm.TextDelta{Text: "kept-refusal"},
		llm.Done{StopReason: llm.StopEndTurn},
	}}
}

// savedHigh are the refusals of model m at high in the state directory's file.
func savedHigh(t *testing.T) []llm.ReasoningRefusal {
	t.Helper()
	path, err := stateFile(refusalFileName)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f refusalFile
	if err := jsonx.UnmarshalStrict(b, &f); err != nil {
		t.Fatalf("refusals file: %v\n%s", err, b)
	}
	var got []llm.ReasoningRefusal
	for _, list := range f.Providers {
		for _, r := range list {
			if r.Model == "m" && r.Effort == "high" {
				got = append(got, r)
			}
		}
	}
	return got
}

// TestRefusalsSaved: a turn whose response warns reasoning_learned with
// Refused {Model: m, Effort: high} leaves that refusal in the state
// directory's file, once, however many turns repeat it (0036-MADR D4).
func TestRefusalsSaved(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	h := newModelHarness(t, llmtest.NewScript(refusalTurn(), refusalTurn()), t.TempDir())
	h.start(Prompt{})
	h.press("one\r")
	waitUntil(t, func() bool { return strings.Count(h.stdout.String(), "kept-refusal") >= 1 })
	h.press("two\r")
	waitUntil(t, func() bool { return strings.Count(h.stdout.String(), "kept-refusal") >= 2 })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if got := savedHigh(t); len(got) != 1 {
		t.Fatalf("saved %d refusals of m at high, want 1: %+v", len(got), got)
	}
}

// TestRefusalsFileUnreadable: a corrupt file is ignored with one warning,
// and the provider is built (0036-MADR D4).
func TestRefusalsFileUnreadable(t *testing.T) {
	e, _ := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-key")
	path, err := stateFile(refusalFileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := modelProvider(e)("s", llm.Retry{Off: true})
	if err != nil || p == nil {
		t.Fatalf("provider: %v, %v; want it built", p, err)
	}
	stderr := e.out.err.(*bytes.Buffer)
	if n := strings.Count(stderr.String(), "Warning: "+refusalFileName); n != 1 {
		t.Fatalf("%d warnings, want 1; stderr %q", n, stderr.String())
	}
}
