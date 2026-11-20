package agent_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/tool"
)

// records is a slog handler that keeps the records it is given.
type records struct {
	level slog.Level
	recs  []slog.Record
}

func (h *records) Enabled(_ context.Context, l slog.Level) bool { return l >= h.level }
func (h *records) Handle(_ context.Context, r slog.Record) error {
	h.recs = append(h.recs, r.Clone())
	return nil
}
func (h *records) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *records) WithGroup(string) slog.Handler      { return h }

func (h *records) count(level slog.Level) int {
	n := 0
	for _, r := range h.recs {
		if r.Level == level {
			n++
		}
	}
	return n
}

func warnTurn(kind, model, detail string) llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{
		llm.Warning{Kind: kind, Model: model, Detail: detail},
		llm.TextDelta{Text: "ok"},
		llm.Done{StopReason: llm.StopEndTurn},
	}}
}

// TestWarningLoggedOnce: two turns that warn effort_degraded for one model
// give one Warn record. A different model, or a different kind, gives
// another (0036-MADR D3).
func TestWarningLoggedOnce(t *testing.T) {
	s := llmtest.NewScript(
		warnTurn(llm.WarningEffortDegraded, "m", "sent low"),
		warnTurn(llm.WarningEffortDegraded, "m", "sent low"),
		warnTurn(llm.WarningEffortDegraded, "other", "sent low"),
		warnTurn(llm.WarningImageDropped, "m", "dropped one"),
	)
	h := &records{level: slog.LevelDebug}
	a := agent.New(agent.Config{Provider: s, Logger: slog.New(h)})
	for range 2 {
		if _, _, err := collect(t.Context(), t, a, tool.Env{}, "hi"); err != nil {
			t.Fatal(err)
		}
	}
	if got := h.count(slog.LevelWarn); got != 1 {
		t.Fatalf("after two turns of one model: %d Warn records, want 1; %d records", got, len(h.recs))
	}
	if _, _, err := collect(t.Context(), t, a, tool.Env{}, "hi"); err != nil {
		t.Fatal(err)
	}
	if got := h.count(slog.LevelWarn); got != 2 {
		t.Fatalf("after a different model: %d Warn records, want 2", got)
	}
	if _, _, err := collect(t.Context(), t, a, tool.Env{}, "hi"); err != nil {
		t.Fatal(err)
	}
	if got := h.count(slog.LevelWarn); got != 3 {
		t.Fatalf("after a different kind: %d Warn records, want 3", got)
	}
}

// TestDeferralLoggedAtDebug: deferral_emulated is one Debug record per
// model, and not a Warn record (0012-MADR D21, carried by 0036-PLAN P2).
func TestDeferralLoggedAtDebug(t *testing.T) {
	s := llmtest.NewScript(
		warnTurn(llm.WarningDeferralEmulated, "m", "appended"),
		warnTurn(llm.WarningDeferralEmulated, "m", "appended"),
		warnTurn(llm.WarningDeferralEmulated, "other", "appended"),
	)
	h := &records{level: slog.LevelDebug}
	a := agent.New(agent.Config{Provider: s, Logger: slog.New(h)})
	for range 2 {
		if _, _, err := collect(t.Context(), t, a, tool.Env{}, "hi"); err != nil {
			t.Fatal(err)
		}
	}
	if h.count(slog.LevelWarn) != 0 || h.count(slog.LevelDebug) != 1 {
		t.Fatalf("after two turns: %d Warn, %d Debug; want 0 and 1", h.count(slog.LevelWarn), h.count(slog.LevelDebug))
	}
	if _, _, err := collect(t.Context(), t, a, tool.Env{}, "hi"); err != nil {
		t.Fatal(err)
	}
	if h.count(slog.LevelWarn) != 0 || h.count(slog.LevelDebug) != 2 {
		t.Fatalf("after a different model: %d Warn, %d Debug; want 0 and 2", h.count(slog.LevelWarn), h.count(slog.LevelDebug))
	}
}
