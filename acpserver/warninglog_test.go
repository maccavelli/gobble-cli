package acpserver

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// warnRecords is a slog handler that counts the "model warning" records
// the agent writes at Warn.
type warnRecords struct {
	mu sync.Mutex
	n  int
}

func (h *warnRecords) Enabled(context.Context, slog.Level) bool { return true }
func (h *warnRecords) Handle(_ context.Context, r slog.Record) error {
	if r.Level == slog.LevelWarn && r.Message == "model warning" {
		h.mu.Lock()
		h.n++
		h.mu.Unlock()
	}
	return nil
}
func (h *warnRecords) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *warnRecords) WithGroup(string) slog.Handler      { return h }

func (h *warnRecords) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.n
}

// effortTurn is a turn whose response warns effort_degraded for model m1,
// then answers.
func effortTurn() llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{
		llm.Warning{Kind: llm.WarningEffortDegraded, Model: "m1", Detail: "sent low"},
		llm.TextDelta{Text: "ok"},
		llm.Done{StopReason: llm.StopEndTurn},
	}}
}

// Each session keeps one warning log across its prompts, each of which
// builds an Agent: two prompts of one session that warn effort_degraded for
// one model give one Warn record, and a second session gives its own
// (0036-MADR D3, 0036-PLAN gap G4).
func TestSessionWarningLogIsPerSession(t *testing.T) {
	logs := &warnRecords{}
	script := llmtest.NewScript(effortTurn(), effortTurn(), effortTurn())
	ag := New(Options{Version: "1.2.3", Models: func() ModelChoice { return testModels },
		Provider: func(*provider.ModelData, string, llm.Retry) (llm.Provider, error) { return script, nil },
		Logger:   slog.New(logs)})
	c := acptest.Connect(t, ag, &acptest.Client{})
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	first := newSession(t, c, t.TempDir(), nil).SessionId
	for _, text := range []string{"one", "two"} {
		if _, err := prompt(t, c, first, text); err != nil {
			t.Fatalf("%s: %v", text, err)
		}
	}
	if got := logs.count(); got != 1 {
		t.Fatalf("after two prompts of one session: %d Warn records, want 1", got)
	}
	second := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, second, "three"); err != nil {
		t.Fatal(err)
	}
	if got := logs.count(); got != 2 {
		t.Fatalf("after a second session's prompt: %d Warn records, want 2", got)
	}
}
