package cli

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// Q23's flush at the program's give-up (0063-MADR D22; PLAN P2 step 17, A49;
// rows R80, R81, R84, R132). A gate stands in for a POSIX terminal's
// cancellable reader; the flush is counted.

// tookGate is a gate whose close always reports a completed read: the race
// of a give-up with a byte's arrival (row R129), or any gated input whose
// reads are a test's.
type tookGate struct{}

func (tookGate) Open()       {}
func (tookGate) Close() bool { return true }

// countedFlush counts a terminal input's flushes.
type countedFlush struct{ n atomic.Int64 }

func (c *countedFlush) flush() error { c.n.Add(1); return nil }

// gatedStub is a terminal input over in with a gate, as on a POSIX
// terminal, and a counted flush. (2026-11-21, MADR D23: the give-up flush
// is keyed on a POSIX terminal, so the stub says it is one.)
func gatedStub(in keysIn, c *countedFlush) *terminalInput {
	ti := newTerminalInput(in, c.flush)
	ti.gate = tookGate{}
	ti.posix = true
	return ti
}

// A49 (rows R80, R81, R84, R132).
func TestFlushAtGiveUp(t *testing.T) {
	t.Run("a withdrawn permission request", func(t *testing.T) {
		var c countedFlush
		ls := permissionSession(t, &countedChunks{}, "")
		ls.keys = gatedStub(&sentIn{results: make(chan stubResult)}, &c)
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel() // the agent withdrew the request
		}()
		if a := ls.askPermission(ctx, writeRequest()); a.OptionID != "" {
			t.Fatalf("a withdrawn request answered %+v", a)
		}
		if n := c.n.Load(); n != 1 {
			t.Fatalf("%d flushes at the withdrawal; want one", n)
		}
	})
	t.Run("gobble auth login's paste prompt, ended by a loopback", func(t *testing.T) {
		var c countedFlush
		e, _, _ := pipeEnv(t)
		e.termIn = gatedStub(&sentIn{results: make(chan stubResult)}, &c)
		replace(t, &signInWith, func(ctx context.Context, _ string, p provider.Prompter, _ provider.SignInOptions, _ provider.LoginStore) (provider.Credential, error) {
			loop, cancel := context.WithCancel(ctx)
			defer cancel()
			go func() {
				time.Sleep(100 * time.Millisecond)
				cancel() // the loopback arrives first
			}()
			_, _ = p.Input(loop, "Paste the redirect URL")
			return provider.Credential{}, errLoopback
		})
		if err := e.loginWith(e.ctx, "grok", false, nil); !errors.Is(err, errLoopback) {
			t.Fatalf("the sign-in: %v", err)
		}
		if n := c.n.Load(); n != 1 {
			t.Fatalf("%d flushes at the paste prompt's give-up; want one", n)
		}
	})
	t.Run("a Ctrl+C give-up flushes nothing", func(t *testing.T) {
		var c countedFlush
		ti := gatedStub(&sentIn{results: make(chan stubResult)}, &c)
		ctx, cancel := context.WithCancel(t.Context())
		q := &questions{ctx: ctx, in: ti, out: io.Discard}
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel() // ls.ask's interrupt cancel
		}()
		if _, err := q.choose("Which?", []option{{"A", ""}, {"B", ""}}, 0); !errors.Is(err, errCancelled) {
			t.Fatalf("the interrupted menu: %v", err)
		}
		if n := c.n.Load(); n != 0 {
			t.Fatalf("%d flushes at the person's give-up; want none", n)
		}
	})
}

// errLoopback is the stubbed loopback's sign-in result.
var errLoopback = errors.New("the stubbed loopback finished first")
