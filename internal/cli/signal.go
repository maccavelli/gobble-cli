package cli

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
)

// signalCause is the cancellation cause a shutdown signal leaves on the
// context, so the exit code can be 128 plus the signal.
type signalCause struct{ sig os.Signal }

func (c signalCause) Error() string { return "received " + c.sig.String() }

// watchSignals returns a context cancelled with signalCause on the first of
// sigs. notify is signal.Notify in production. After the first signal the
// channel is released, so a second signal gets the default behaviour and
// ends a process that does not stop. stop releases everything.
func watchSignals(parent context.Context, sigs []os.Signal, notify func(chan<- os.Signal, ...os.Signal)) (ctx context.Context, stop func()) {
	ctx, cancel := context.WithCancelCause(parent)
	ch := make(chan os.Signal, 1)
	notify(ch, sigs...)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		select {
		case s := <-ch:
			signal.Stop(ch)
			cancel(signalCause{sig: s})
		case <-done:
		case <-ctx.Done():
		}
	})
	return ctx, func() {
		close(done)
		wg.Wait()
		signal.Stop(ch)
		cancel(context.Canceled)
	}
}

// signalExit reports the exit code of a signal that cancelled ctx.
func signalExit(ctx context.Context) (int, bool) {
	if c, ok := errors.AsType[signalCause](context.Cause(ctx)); ok {
		return exitForSignal(c.sig), true
	}
	return 0, false
}
