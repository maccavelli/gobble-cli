package acpserver

import (
	"context"
	"iter"
	"sync"

	"github.com/maccavelli/gobble-cli/llm"
)

// cutWatch is a turn's provider, watching each model call for the SDK's
// report that the output limit cut a tool call, which the reply then
// dropped: a call that ends max_tokens after a tool_call_cut warning. The
// turn then ends max_tokens with its text, and the agent says why after it
// (0071-MADR P29, the owner's "End the turn, keep the text, notify
// (Recommended)"; 0065-MADR D7, D8).
type cutWatch struct {
	llm.Provider
	mu  sync.Mutex
	cut bool // the last call's reply ended at the limit with a call dropped
}

// Stream is the provider's stream, watched.
func (w *cutWatch) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	return func(yield func(llm.Event, error) bool) {
		w.set(false)
		dropped := false
		for ev, err := range w.Provider.Stream(ctx, req) {
			switch e := ev.(type) {
			case llm.Warning:
				dropped = dropped || e.Kind == llm.WarningToolCallCut
			case llm.Done:
				w.set(dropped && e.StopReason == llm.StopMaxTokens)
			}
			if !yield(ev, err) {
				return
			}
		}
	}
}

func (w *cutWatch) set(cut bool) {
	w.mu.Lock()
	w.cut = cut
	w.mu.Unlock()
}

// take reports whether the last call's reply ended at the output limit with
// a tool call dropped, and forgets it, so its notice is sent once.
func (w *cutWatch) take() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	cut := w.cut
	w.cut = false
	return cut
}

// probingCutWatch is a cutWatch over a provider that probes, so the turn's
// wait for the network still probes it (llm.Prober).
type probingCutWatch struct {
	*cutWatch
	prober llm.Prober
}

func (p probingCutWatch) Probe(ctx context.Context) error { return p.prober.Probe(ctx) }

// watchCuts is p watched for a cut tool call, as the provider the turn
// calls, and the watch.
func watchCuts(p llm.Provider) (llm.Provider, *cutWatch) {
	w := &cutWatch{Provider: p}
	if pr, ok := p.(llm.Prober); ok {
		return probingCutWatch{cutWatch: w, prober: pr}, w
	}
	return w, w
}
