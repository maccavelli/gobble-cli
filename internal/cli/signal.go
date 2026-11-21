package cli

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// signalCause is the cancellation cause a shutdown signal leaves on the
// context, so the exit code can be 128 plus the signal.
type signalCause struct{ sig os.Signal }

func (c signalCause) Error() string { return "received " + c.sig.String() }

// interrupts lets a running turn take the first interrupt (0008-MADR D5):
// Ctrl+C during a turn cancels the turn, and a second Ctrl+C before it ends
// stops the process. console is the process's interrupt mark, which a
// Windows console read that gives nothing right after an interrupt reads
// (0063-MADR D19, D20).
type interrupts struct {
	mu      sync.Mutex
	turn    func()
	taken   bool
	console term.InterruptMark
}

// consoleMark is the process's interrupt mark, or nil with no interrupts.
func (i *interrupts) consoleMark() *term.InterruptMark {
	if i == nil {
		return nil
	}
	return &i.console
}

// during hands the next interrupt to cancel until release is called.
func (i *interrupts) during(cancel func()) (release func()) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.turn, i.taken = cancel, false
	return func() {
		i.mu.Lock()
		defer i.mu.Unlock()
		i.turn, i.taken = nil, false
	}
}

// take reports whether a turn took the interrupt, calling its cancel. A
// turn takes only the first one.
func (i *interrupts) take() bool {
	if i == nil {
		return false
	}
	i.mu.Lock()
	cancel := i.turn
	ok := cancel != nil && !i.taken
	i.taken = i.taken || ok
	i.mu.Unlock()
	if ok {
		cancel()
	}
	return ok
}

// watchSignals returns a context cancelled with signalCause on the first of
// sigs that intr does not take. notify is signal.Notify in production.
// After the cancelling signal the channel is released, so a further signal
// gets the default behaviour and ends a process that does not stop. stop
// releases everything. Each interrupt is marked for the console read that
// may be waiting, before it is handed on, whether a prompt takes it or not
// (0063-MADR D19, row R110).
func watchSignals(parent context.Context, sigs []os.Signal, notify func(chan<- os.Signal, ...os.Signal), intr *interrupts) (ctx context.Context, stop func()) {
	ctx, cancel := context.WithCancelCause(parent)
	ch := make(chan os.Signal, 1)
	notify(ch, sigs...)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			select {
			case s := <-ch:
				if s == os.Interrupt {
					intr.consoleMark().Interrupted()
					if intr.take() {
						continue
					}
				}
				signal.Stop(ch)
				cancel(signalCause{sig: s})
				return
			case <-done:
				return
			case <-ctx.Done():
				return
			}
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
