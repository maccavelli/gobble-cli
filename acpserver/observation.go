package acpserver

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"strings"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/personality"
)

// callWatch is a provider whose every call's end is reported: done with the
// call's done event and the usage before it; failed with the error that
// ended it, unless the call was cancelled (0071-MADR P22, P25, P32, P34).
type callWatch struct {
	llm.Provider
	done   func(req *llm.Request, d llm.Done, u llm.Usage)
	failed func(req *llm.Request, err error)
}

// Stream is the provider's stream, watched.
func (w callWatch) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	return func(yield func(llm.Event, error) bool) {
		var usage llm.Usage
		for ev, err := range w.Provider.Stream(ctx, req) {
			switch e := ev.(type) {
			case llm.Usage:
				usage = e
			case llm.Done:
				if err == nil && w.done != nil {
					w.done(req, e, usage)
				}
			}
			if err != nil && w.failed != nil && ctx.Err() == nil && !errors.Is(err, context.Canceled) {
				w.failed(req, err)
			}
			if !yield(ev, err) {
				return
			}
		}
	}
}

// probingCallWatch is a callWatch over a provider that probes, so a turn's
// wait for the network still probes it (llm.Prober).
type probingCallWatch struct {
	callWatch
	prober llm.Prober
}

func (p probingCallWatch) Probe(ctx context.Context) error { return p.prober.Probe(ctx) }

// watchCalls is p with each call's end reported to done and failed, either
// nil for none; a Prober stays one.
func watchCalls(p llm.Provider, done func(*llm.Request, llm.Done, llm.Usage), failed func(*llm.Request, error)) llm.Provider {
	w := callWatch{Provider: p, done: done, failed: failed}
	if pr, ok := p.(llm.Prober); ok {
		return probingCallWatch{callWatch: w, prober: pr}
	}
	return w
}

// turnProvider is model as a turn calls it: watched for a cut tool call
// (0071-MADR P29), and at each call's end for what it carried, which limits
// records, and its output, for the saved caps (P22, P25, P32).
func (a *Agent) turnProvider(model llm.Provider, provider string, limits *turnLimits) (llm.Provider, *cutWatch) {
	seen := a.outputSeen(provider)
	return watchCuts(watchCalls(model, func(req *llm.Request, d llm.Done, u llm.Usage) {
		limits.sent(req, d, u)
		if seen != nil {
			seen(req, d, u)
		}
	}, limits.callFailed))
}

// waitTurnDiscovery is waitDiscovery for a turn's first call: when a
// discovery of the model ended and the turn goes on, an output limit it left
// unknown is said once (0066-MADR S10).
func (a *Agent) waitTurnDiscovery(ctx context.Context, s *liveSession, provider, model string) {
	if a.waitDiscovery(ctx, s, provider, model) && ctx.Err() == nil {
		a.noteUnknown(s, provider, model)
	}
}

// outputSeen is provider's calls' done report for the run's saved caps: a
// completed call's output tokens, which drop a saved cap of its model they
// exceed, under the model a cap the call's model met was saved under; an
// estimated usage, which the service did not report, is not reported
// (0071-MADR P25; 0071-PLAN's review deviation of 2026-11-25, R3, R6).
func (a *Agent) outputSeen(provider string) func(*llm.Request, llm.Done, llm.Usage) {
	if a.callOutput == nil {
		return nil
	}
	return func(req *llm.Request, _ llm.Done, u llm.Usage) {
		if u.OutputTokens > 0 && !u.Estimated {
			a.callOutput(provider, a.savedUnder(provider, req.Model), int(u.OutputTokens))
		}
	}
}

// savedUnder is the model provider's caps for a request naming model are
// saved under: the model of the cap one such request learned, which the SDK
// names cmp.Or(req.Model, its own model), else model (0071-PLAN's review
// deviation of 2026-11-25, R6).
func (a *Agent) savedUnder(provider, model string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if m, ok := a.savedAs[refOf(provider, model)]; ok {
		return m
	}
	return model
}

// observedSaved is provider's calls' failure report for s's summaries: a
// refusal whose Observed states a cap below the limit sent, which the
// ladder sends the request again at, is learned (0071-MADR P34).
func (a *Agent) observedSaved(s *liveSession, provider string) func(*llm.Request, error) {
	return func(req *llm.Request, err error) {
		if ob, ok := observedBelow(err, req.MaxTokens); ok {
			a.learnedCap(s, provider, req.Model, ob)
		}
	}
}

// observedBelow is err's llm.APIError.Observed, when it states a cap of at
// least 1 below sent, the limit the call carried.
func observedBelow(err error, sent int) (llm.LimitObservation, bool) {
	ae, ok := errors.AsType[*llm.APIError](err)
	if !ok || ae.Observed.Tokens < 1 || ae.Observed.Tokens >= sent {
		return llm.LimitObservation{}, false
	}
	return ae.Observed, true
}

// learnedCap records the cap provider's service stated for a model, which a
// call asking for model was sent again at: saved through the run's store of
// saved caps, when it keeps one, and said once per session and model with
// model.limit.learned, before the turn's next reply (0066-MADR D7, D10,
// S12; 0071-MADR P19, P34). When the cap names another model, as it does
// for a request that names none, later calls asking for model report their
// output under the cap's (R6).
func (a *Agent) learnedCap(s *liveSession, provider, model string, o llm.LimitObservation) {
	if o.Model != "" && !strings.EqualFold(o.Model, model) {
		a.mu.Lock()
		if a.savedAs == nil {
			a.savedAs = map[string]string{}
		}
		a.savedAs[refOf(provider, model)] = o.Model
		a.mu.Unlock()
	}
	if a.saveObservation != nil {
		a.saveObservation(provider, o)
	}
	text := a.voice(s).Text(personality.ModelLimitLearned, personality.Args{argModel: o.Model, "stated": thousands(int64(o.Tokens))})
	a.noteOnce(s, "learned:"+refOf(provider, o.Model), text)
}

// noteUnknown keeps model.limit.unknown for s's next reply, once per session
// and model, when the discovery of provider's model ended with its output
// limit unknown (0066-MADR D10 as amended 2026-11-25, S10; 0071-MADR V7).
func (a *Agent) noteUnknown(s *liveSession, provider, model string) {
	if c, _, _ := outputCap(a.modelOf(provider, model)); c > 0 {
		return
	}
	a.noteOnce(s, "unknown:"+refOf(provider, model), a.voice(s).Text(personality.ModelLimitUnknown, personality.Args{argModel: model}))
}

// noteOnce keeps text for s's next reply unless key was kept before.
func (a *Agent) noteOnce(s *liveSession, key, text string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s.said[key] {
		return
	}
	if s.said == nil {
		s.said = map[string]bool{}
	}
	s.said[key] = true
	s.notes = append(s.notes, text)
}

// sayNotes sends the notes kept for s, each once, as agent messages, live
// only: before the reply's next update (0066-MADR D10).
func (a *Agent) sayNotes(out *updates, s *liveSession) {
	a.mu.Lock()
	notes := s.notes
	s.notes = nil
	a.mu.Unlock()
	for _, n := range notes {
		out.flush()
		out.send(acp.UpdateAgentMessageText(n + "\n\n"))
	}
}

// warnSDKBasis logs, once per session and model, a basis the SDK reported
// for a call gobble named its own limit for: shown raw, as sdk:<basis>
// (0071-MADR P31, the owner's "gobble's basis, wire value (Recommended)").
func (a *Agent) warnSDKBasis(s *liveSession, provider, model, basis string) {
	ref := refOf(provider, model)
	a.mu.Lock()
	logged := s.sdkBasis[ref]
	if s.sdkBasis == nil {
		s.sdkBasis = map[string]bool{}
	}
	s.sdkBasis[ref] = true
	a.mu.Unlock()
	if !logged {
		a.logger.Warn("output limit: the SDK reports a basis gobble did not ask for", slog.String("session", string(s.id)),
			slog.String("provider", provider), slog.String("model", model), slog.String("basis", basis))
	}
}
