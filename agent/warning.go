package agent

import (
	"log/slog"
	"sync"

	"github.com/maccavelli/gobble-cli/llm"
)

// WarningLog is the warnings one session has already said. A caller that
// builds an Agent for each prompt of one session passes the same log, so a
// kind and model are logged once (0036-MADR D3).
type WarningLog struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

// first reports that kind and model have not been logged yet, and records
// them.
func (w *WarningLog) first(kind, model string) bool {
	if w == nil {
		return true
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.seen == nil {
		w.seen = map[string]struct{}{}
	}
	key := kind + "\x00" + model
	if _, ok := w.seen[key]; ok {
		return false
	}
	w.seen[key] = struct{}{}
	return true
}

// note logs w once per kind and model. deferral_emulated is Debug; every
// other kind is Warn (0036-MADR D3, 0012-MADR D21).
func (t *turn) note(w llm.Warning) {
	if !t.a.cfg.Warned.first(w.Kind, w.Model) {
		return
	}
	log := t.a.cfg.Logger
	if log == nil {
		return
	}
	level := slog.LevelWarn
	if w.Kind == llm.WarningDeferralEmulated {
		level = slog.LevelDebug
	}
	log.LogAttrs(t.ctx, level, "model warning", slog.String("kind", w.Kind), slog.String("model", w.Model), slog.String("detail", w.Detail))
}
