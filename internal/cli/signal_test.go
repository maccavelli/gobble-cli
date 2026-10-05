package cli

import (
	"context"
	"os"
	"syscall"
	"testing"
)

// fakeNotify delivers sig on the channel watchSignals registers.
func fakeNotify(sig os.Signal) func(chan<- os.Signal, ...os.Signal) {
	return func(ch chan<- os.Signal, _ ...os.Signal) { ch <- sig }
}

// A signal cancels the context with its cause, and the exit code is 128
// plus the signal: 130, 129, 143 (0008-MADR D10).
func TestWatchSignals(t *testing.T) {
	for _, tc := range []struct {
		sig  os.Signal
		want int
	}{
		{os.Interrupt, 130},
		{syscall.SIGHUP, 129},
		{syscall.SIGTERM, 143},
	} {
		t.Run(tc.sig.String(), func(t *testing.T) {
			ctx, stop := watchSignals(t.Context(), []os.Signal{tc.sig}, fakeNotify(tc.sig))
			<-ctx.Done()
			code, ok := signalExit(ctx)
			stop()
			if !ok || code != tc.want {
				t.Fatalf("signalExit = %d, %v; want %d, true", code, ok, tc.want)
			}
		})
	}
}

func TestNoSignalNoExitCode(t *testing.T) {
	ctx, stop := watchSignals(t.Context(), nil, func(chan<- os.Signal, ...os.Signal) {})
	stop()
	if _, ok := signalExit(ctx); ok {
		t.Fatal("stop without a signal reported a signal exit")
	}
	parent, cancel := context.WithCancel(t.Context())
	ctx, stop = watchSignals(parent, nil, func(chan<- os.Signal, ...os.Signal) {})
	cancel()
	<-ctx.Done()
	stop()
	if _, ok := signalExit(ctx); ok {
		t.Fatal("a cancelled parent reported a signal exit")
	}
}
