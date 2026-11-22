package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
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
			ctx, stop := watchSignals(t.Context(), []os.Signal{tc.sig}, fakeNotify(tc.sig), nil)
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
	ctx, stop := watchSignals(t.Context(), nil, func(chan<- os.Signal, ...os.Signal) {}, nil)
	stop()
	if _, ok := signalExit(ctx); ok {
		t.Fatal("stop without a signal reported a signal exit")
	}
	parent, cancel := context.WithCancel(t.Context())
	ctx, stop = watchSignals(parent, nil, func(chan<- os.Signal, ...os.Signal) {}, nil)
	cancel()
	<-ctx.Done()
	stop()
	if _, ok := signalExit(ctx); ok {
		t.Fatal("a cancelled parent reported a signal exit")
	}
}

// captureNotify records the channel watchSignals registers, so a test can
// send signals later.
func captureNotify() (func(chan<- os.Signal, ...os.Signal), func() chan<- os.Signal) {
	got := make(chan chan<- os.Signal, 1)
	var ch chan<- os.Signal
	return func(c chan<- os.Signal, _ ...os.Signal) { got <- c }, func() chan<- os.Signal {
		if ch == nil {
			ch = <-got
		}
		return ch
	}
}

// During a turn the first interrupt cancels the turn and the process goes
// on; a second, before the turn ends, exits 130 (0008-MADR D5).
func TestInterruptDuringTurn(t *testing.T) {
	intr := &interrupts{}
	notify, ch := captureNotify()
	ctx, stop := watchSignals(t.Context(), []os.Signal{os.Interrupt}, notify, intr)
	defer stop()
	cancelled := make(chan struct{}, 2)
	release := intr.during(func() { cancelled <- struct{}{} })
	ch() <- os.Interrupt
	<-cancelled
	if ctx.Err() != nil {
		t.Fatal("the first interrupt during a turn cancelled the process")
	}
	ch() <- os.Interrupt
	<-ctx.Done()
	if code, ok := signalExit(ctx); !ok || code != 130 {
		t.Fatalf("second interrupt: signalExit = %d, %v; want 130", code, ok)
	}
	if len(cancelled) != 0 {
		t.Fatal("the second interrupt was given to the turn")
	}
	release()
}

// A released turn takes nothing, and a new turn takes its own first
// interrupt again.
func TestInterruptReleased(t *testing.T) {
	intr := &interrupts{}
	n := 0
	release := intr.during(func() { n++ })
	if !intr.take() || intr.take() {
		t.Fatal("a turn must take exactly the first interrupt")
	}
	release()
	if intr.take() {
		t.Fatal("an interrupt with no turn was taken")
	}
	release = intr.during(func() { n++ })
	if !intr.take() {
		t.Fatal("the next turn did not take its first interrupt")
	}
	release()
	if n != 2 {
		t.Fatalf("cancel called %d times, want 2", n)
	}
}

// blockedConsole is a console whose read says it started, then gives what
// the test sends.
type blockedConsole struct {
	started chan struct{}
	results chan stubResult
}

func (c *blockedConsole) Read(p []byte) (int, error) {
	c.started <- struct{}{}
	r := <-c.results
	return copy(p, r.b), r.err
}

// A31 (row R110): watchSignals marks each interrupt for the console read
// that waits, before it hands it on, whether a prompt takes it or not: the
// read that then gives nothing is read again (0063-MADR D19).
func TestWatchSignalsMarksInterrupt(t *testing.T) {
	for _, taken := range []bool{true, false} {
		t.Run(map[bool]string{true: "taken", false: "not taken"}[taken], func(t *testing.T) {
			intr := &interrupts{}
			notify, ch := captureNotify()
			ctx, stop := watchSignals(t.Context(), []os.Signal{os.Interrupt}, notify, intr)
			defer stop()
			handed := make(chan struct{}, 1)
			if taken {
				defer intr.during(func() { handed <- struct{}{} })()
			}
			console := &blockedConsole{started: make(chan struct{}, 4), results: make(chan stubResult)}
			r := term.NewConsoleReader(console, true, intr.consoleMark(), term.MarkWait)
			done := make(chan string, 1)
			go func() {
				buf := make([]byte, 16)
				n, err := r.Read(buf)
				done <- fmt.Sprintf("%q, %v", buf[:n], err)
			}()
			<-console.started
			ch() <- os.Interrupt
			if taken {
				<-handed
			} else {
				<-ctx.Done() // the interrupt nobody took: handed on after the mark
			}
			console.results <- stubResult{err: io.EOF}
			select {
			case <-console.started:
			case s := <-done:
				t.Fatalf("the empty read after the interrupt gave %s; want it read again", s)
			case <-time.After(2 * time.Second):
				t.Fatal("the console was not read again")
			}
			console.results <- stubResult{b: "n\r\n"}
			if s := <-done; s != `"n\r\n", <nil>` {
				t.Fatalf("the reader gave %s; want \"n\\r\\n\"", s)
			}
		})
	}
}

// TERM during a turn stops the process at once: only an interrupt is the
// turn's.
func TestTermDuringTurn(t *testing.T) {
	intr := &interrupts{}
	notify, ch := captureNotify()
	ctx, stop := watchSignals(t.Context(), []os.Signal{syscall.SIGTERM}, notify, intr)
	defer stop()
	defer intr.during(func() { t.Error("TERM was given to the turn") })()
	ch() <- syscall.SIGTERM
	<-ctx.Done()
	if code, ok := signalExit(ctx); !ok || code != 143 {
		t.Fatalf("signalExit = %d, %v; want 143", code, ok)
	}
}
