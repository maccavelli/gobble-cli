package acpserver

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"
)

// discoverBound is how long a call waits for a discovery, whatever the hook
// does with its context: the SDK's bound on a listing read (0066-MADR S8).
var discoverBound = 10 * time.Second

// discovery is a session's discovery of its models' limits (0066-MADR D1,
// D2; 0071-MADR P37, P40): closed is closed at the session's close, which
// ends every discovery still running; pending are the discoveries a call
// waits for, each by its provider/model and closed when it ends or its
// bound passes; running counts the hooks still running, which the close
// waits for. Its fields but running are under the agent's mu.
type discovery struct {
	closed  chan struct{}
	ended   bool
	pending []pendingDiscovery
	running sync.WaitGroup
}

// pendingDiscovery is one discovery a call may wait for.
type pendingDiscovery struct {
	ref  string
	done chan struct{}
}

// discoverModels starts Options.Discover in the background for provider's
// model and, when s starts, the title model, each once; a call of a model
// waits for its own model's (waitDiscovery). A failure is logged at debug,
// and the call goes on with what is cached. A call waits at most
// discoverBound, whatever the hook does. Nil Options.Discover discovers
// nothing. ctx is the request's that set the model: a discovery keeps its
// values, not its cancel, so it outlives that request, and ends at the
// session's close.
func (a *Agent) discoverModels(ctx context.Context, s *liveSession, provider, model string, withTitle bool) {
	if a.discover == nil {
		return
	}
	refs := [][2]string{{provider, model}}
	if withTitle && a.titles.Provider != nil && a.titles.Model != "" && !slices.Contains(refs, [2]string{a.titles.ProviderID, a.titles.Model}) {
		refs = append(refs, [2]string{a.titles.ProviderID, a.titles.Model})
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if s.disc.closed == nil {
		s.disc.closed = make(chan struct{})
	}
	closed := s.disc.closed
	for _, ref := range refs {
		if ref[1] == "" || s.disc.ended {
			continue
		}
		done, hookDone := make(chan struct{}), make(chan struct{})
		s.disc.pending = append(s.disc.pending, pendingDiscovery{ref: ref[0] + "/" + ref[1], done: done})
		dctx, stop := context.WithTimeout(context.WithoutCancel(ctx), discoverBound)
		go func() {
			defer close(done)
			defer stop()
			select {
			case <-hookDone:
			case <-dctx.Done(): // the bound, whatever the hook does
			case <-closed:
			}
		}()
		s.disc.running.Go(func() {
			defer close(hookDone)
			if err := a.discover(dctx, ref[0], ref[1]); err != nil {
				a.logger.DebugContext(dctx, "model limits discovery", slog.String("session", string(s.id)),
					slog.String("provider", ref[0]), slog.String("model", ref[1]), slog.Any("error", err))
			}
		})
	}
}

// waitDiscovery waits for the discoveries of provider's model that s
// started before its call, or for ctx, the call's, to end; a cancel then
// ends the call as it does today. A discovery that has ended is dropped, so
// a later call finds none and does not wait (0066-MADR D1, S9; 0071-MADR
// P40).
func (a *Agent) waitDiscovery(ctx context.Context, s *liveSession, provider, model string) {
	ref := provider + "/" + model
	a.mu.Lock()
	var mine []chan struct{}
	s.disc.pending = slices.DeleteFunc(s.disc.pending, func(p pendingDiscovery) bool {
		select {
		case <-p.done:
			return true
		default:
		}
		if p.ref == ref {
			mine = append(mine, p.done)
		}
		return false
	})
	a.mu.Unlock()
	for _, done := range mine {
		select {
		case <-done:
		case <-ctx.Done():
			return
		}
	}
}

// endDiscovery ends s's discoveries, at its close, and waits for their hooks
// to return, up to closeWait, as a close waits for a title (0066-MADR S40;
// 0046-MADR D1).
func (a *Agent) endDiscovery(s *liveSession) {
	a.mu.Lock()
	if s.disc.closed != nil && !s.disc.ended {
		close(s.disc.closed)
	}
	s.disc.ended = true
	a.mu.Unlock()
	returned := make(chan struct{})
	go func() {
		s.disc.running.Wait()
		close(returned)
	}()
	t := time.NewTimer(closeWait)
	defer t.Stop()
	select {
	case <-returned:
	case <-t.C:
		a.logger.Warn("model limits discovery: it did not stop in time; closing the session", slog.String("session", string(s.id)))
	}
}
