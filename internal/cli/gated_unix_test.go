//go:build unix

package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/maccavelli/go-tui-lib/v2/inline"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The terminal input over a POSIX terminal's cancellable reader (0063-MADR
// D22; PLAN P2 steps 16 to 19, A42, A47, A50, A51; rows R127 to R129,
// R138, R140, R141, R145, R147, R150, R151). A pipe stands in for the
// private descriptor.

// gatedPipe is a term.Gated reader over a non-blocking pipe, and the pipe's
// write end; it is stopped in t.Cleanup.
func gatedPipe(t *testing.T, o term.GatedOptions) (*term.Gated, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Dup(int(r.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	if err := unix.SetNonblock(fd, true); err != nil {
		t.Fatal(err)
	}
	g, err := term.NewGated(fd, o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = w.Close()
		g.Stop()
	})
	return g, w
}

// gatedInput is a terminal input over g, as terminalIn makes it on a POSIX
// terminal; flush, when not nil, is its flush.
func gatedInput(g *term.Gated, flush func() error) *terminalInput {
	ti := newTerminalInput(inline.NewReader(g), flush)
	ti.gate = g
	return ti
}

// readCount counts read(2) through the layer's seam.
type readCount struct{ n atomic.Int64 }

func (c *readCount) read(fd int, p []byte) (int, error) {
	c.n.Add(1)
	return unix.Read(fd, p)
}

func writeTo(t *testing.T, w *os.File, s string) {
	t.Helper()
	if _, err := w.WriteString(s); err != nil {
		t.Fatal(err)
	}
}

// A42 (rows R128, R129, R145, R147, R150, R151): give-ups over the layer.
func TestCancelableGiveUp(t *testing.T) {
	t.Run("a program's give-up with nothing read drops nothing", func(t *testing.T) {
		g, w := gatedPipe(t, term.GatedOptions{})
		ti := gatedInput(g, nil)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
		writeTo(t, w, "next\n")
		if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != "next" {
			t.Fatalf("the next line = %q, %v; want \"next\": nothing was read, so nothing is dropped", s, err)
		}
	})
	t.Run("a Ctrl+C give-up with nothing read drops nothing", func(t *testing.T) {
		g, w := gatedPipe(t, term.GatedOptions{})
		ti := gatedInput(g, nil)
		ctx, cancel := context.WithCancel(t.Context())
		q := &questions{ctx: ctx, in: ti, out: io.Discard}
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()
		if _, err := q.line(); !errors.Is(err, errCancelled) {
			t.Fatalf("the interrupted question: %v", err)
		}
		writeTo(t, w, "n\n")
		if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != "n" {
			t.Fatalf("the next line = %q, %v; want \"n\"", s, err)
		}
	})
	// Rows R127, R150: after a prompt returns, nothing reads the terminal
	// until the next prompt waits.
	t.Run("no read between a prompt's return and the next opening", func(t *testing.T) {
		var c readCount
		g, w := gatedPipe(t, term.GatedOptions{Read: c.read})
		ti := gatedInput(g, nil)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
		writeTo(t, w, "x\n")
		time.Sleep(200 * time.Millisecond)
		if n := c.n.Load(); n != 0 {
			t.Fatalf("%d reads of the terminal with no prompt waiting; want none", n)
		}
		if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != "x" || c.n.Load() != 1 {
			t.Fatalf("the next line = %q, %v after %d reads; want \"x\" after one", s, err, c.n.Load())
		}
	})
	// Row R145: with the gate closed and bytes ready, the wait does not
	// return: it cannot spin. (2026-11-21, MADR D23, row R146: this is the
	// case that catches a wake pipe not drained before a wait, plant 91.)
	t.Run("a closed gate's wait does not return for bytes ready", func(t *testing.T) {
		var returns atomic.Int64
		g, w := gatedPipe(t, term.GatedOptions{Returned: func() { returns.Add(1) }})
		ti := gatedInput(g, nil)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
		writeTo(t, w, "ready\n")
		time.Sleep(50 * time.Millisecond)
		before := returns.Load()
		time.Sleep(200 * time.Millisecond)
		if after := returns.Load(); after != before {
			t.Fatalf("the closed gate's wait returned %d times in 200 ms with bytes ready; want none", after-before)
		}
		if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != "ready" {
			t.Fatalf("the next line = %q, %v; want \"ready\"", s, err)
		}
	})
	// Row R147: a parked ask, then a 256-byte read: what it cannot take stays
	// in pending, where discard() reaches it.
	t.Run("the gated ask is lineBuf, so discard reaches the rest", func(t *testing.T) {
		g, w := gatedPipe(t, term.GatedOptions{})
		ti := gatedInput(g, nil)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
		writeTo(t, w, strings.Repeat("p", 400))
		buf := make([]byte, 256)
		if n, err := ti.Read(t.Context(), buf); err != nil || n != 256 {
			t.Fatalf("the 256-byte read gave %d, %v", n, err)
		}
		ti.discard()
		writeTo(t, w, "z")
		bounded, stop := context.WithTimeout(t.Context(), 3*time.Second)
		defer stop()
		if n, err := ti.Read(bounded, buf); err != nil || string(buf[:n]) != "z" {
			t.Fatalf("after discard the read gave %q, %v; want \"z\" alone", buf[:n], err)
		}
	})
	// Row R151: with no gate, D13's drop holds as built.
	t.Run("no gate: a program's give-up drops its completion", func(t *testing.T) {
		ti, pw := pipeInput(t)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
		write(pw, "hello\n")
		time.Sleep(100 * time.Millisecond)
		write(pw, "next\n")
		if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != "next" {
			t.Fatalf("the next line = %q, %v; want \"next\"", s, err)
		}
	})
	// Row R129: a read that completed before the gate closed: a program's
	// give-up drops it; a Ctrl+C give-up gives it to the next prompt.
	for _, c := range []struct {
		name, want string
		program    bool
	}{{"a race completion, the program's give-up", "next", true}, {"a race completion, Ctrl+C", "raced", false}} {
		t.Run(c.name, func(t *testing.T) {
			in := &sentIn{results: make(chan stubResult)}
			ti := newTerminalInput(in, nil)
			ti.gate = tookGate{}
			qctx, qcancel := context.WithCancel(t.Context())
			defer qcancel()
			q := &questions{ctx: qctx, in: ti, out: io.Discard}
			if c.program {
				_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
			} else {
				go func() {
					time.Sleep(100 * time.Millisecond)
					qcancel()
				}()
				_, _ = q.line()
			}
			go func() {
				in.results <- stubResult{b: "raced\n"}
				in.results <- stubResult{b: "next\n"}
			}()
			if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != c.want {
				t.Fatalf("the next line = %q, %v; want %q", s, err, c.want)
			}
		})
	}
	// Row R89: a queued Ctrl+D read in raw mode is one byte; at readSecret
	// 0x04 cancels and 0x00 is ignored.
	t.Run("readSecret and a queued Ctrl+D's byte", func(t *testing.T) {
		q, _, _ := asked(t, "\x04")
		if _, err := q.secret("Key"); !errors.Is(err, errCancelled) {
			t.Fatalf("0x04 at the key prompt: %v; want cancelled", err)
		}
		q, _, _ = asked(t, "\x00k\r")
		if s, err := q.secret("Key"); err != nil || s != "k" {
			t.Fatalf("0x00 at the key prompt: %q, %v; want it ignored", s, err)
		}
	})
}

// A47 (row R140): a timed wait of the same caller over the layer reads
// nothing; the bytes written after come whole to its next read.
func TestCancelableTimedWait(t *testing.T) {
	var c readCount
	g, w := gatedPipe(t, term.GatedOptions{Read: c.read})
	ti := gatedInput(g, nil)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if _, err := ti.Read(ctx, make([]byte, 16)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the timed wait: %v", err)
	}
	writeTo(t, w, "kept")
	time.Sleep(150 * time.Millisecond)
	if n := c.n.Load(); n != 0 {
		t.Fatalf("%d reads after the timed wait ended; want none", n)
	}
	bounded, stop := context.WithTimeout(t.Context(), 3*time.Second)
	defer stop()
	buf := make([]byte, 16)
	if n, err := ti.Read(bounded, buf); err != nil || string(buf[:n]) != "kept" {
		t.Fatalf("the next read = %q, %v; want \"kept\"", buf[:n], err)
	}
}

// A51 (row R141): the permission prompt's timed wait over the layer reads
// nothing, and the key typed after it answers.
func TestPermissionTimedWaitOverLayer(t *testing.T) {
	g, w := gatedPipe(t, term.GatedOptions{})
	ls := permissionSession(t, &countedChunks{}, "")
	ls.keys = gatedInput(g, nil)
	writeTo(t, w, "\x1b")
	go func() {
		time.Sleep(300 * time.Millisecond) // after the Esc timer's wait ended
		_, _ = w.WriteString("1")
	}()
	if a := ls.askPermission(t.Context(), writeRequest()); a.OptionID != "allow_once" {
		t.Fatalf("the answer %+v; want allow_once, from the key typed after the timed wait", a)
	}
}

// warnWords is Q24's warning, as the owner approved it, for err.
func warnWords(err string) string {
	return "Warning: gobble cannot wait on this terminal without reading it (" + err + "); keys typed while a sign-in finishes may be lost."
}

// terminalEnv is a run whose standard input is a pipe taken for a
// terminal, and the pipe's write end.
func terminalEnv(t *testing.T) (*runEnv, *os.File, *syncBuffer) {
	t.Helper()
	e, w, stderr := pipeEnv(t)
	replace(t, &isTerminalFile, func(*os.File) bool { return true })
	return e, w, stderr
}

// A50 (rows R138, R150): a wait that cannot be used falls back to reading
// as built, with Q24's warning in the owner's words, once in the process;
// the prompts still read.
func TestCancelableFallbackWarnsOnce(t *testing.T) {
	t.Run("both opens fail", func(t *testing.T) {
		e, w, stderr := terminalEnv(t)
		replace(t, &openGated, func(*os.File, term.GatedOptions) (*term.Gated, error) {
			return nil, errors.New("neither the path nor /dev/tty opened")
		})
		q := &questions{ctx: t.Context(), in: e.terminalIn(), out: io.Discard}
		writeTo(t, w, "y\n")
		if s, err := q.line(); err != nil || s != "y" {
			t.Fatalf("the prompt read %q, %v; want \"y\"", s, err)
		}
		e.warnWait(errors.New("a second cause"))
		shown := strings.Join(strings.Fields(stderr.String()), " ")
		if n := strings.Count(shown, warnWords("neither the path nor /dev/tty opened")); n != 1 {
			t.Fatalf("the warning shown %d times; want once:\n%s", n, stderr)
		}
		if n := strings.Count(shown, "cannot wait on this terminal"); n != 1 {
			t.Fatalf("%d warnings for two causes; want one", n)
		}
	})
	// (2026-11-21, MADR D23, row R156: the warning of a fallback inside a
	// read is written before the next question's text, so one is asked.)
	t.Run("the wait fails", func(t *testing.T) {
		e, w, stderr := terminalEnv(t)
		replace(t, &openGated, failingWait(t))
		q := &questions{ctx: t.Context(), in: e.terminalIn(), out: io.Discard}
		writeTo(t, w, "y\n")
		if s, err := q.line(); err != nil || s != "y" {
			t.Fatalf("the prompt read %q, %v; want \"y\" from standard input as built", s, err)
		}
		writeTo(t, w, "z\n")
		if s, err := q.input("Next"); err != nil || s != "z" {
			t.Fatalf("the next question read %q, %v; want \"z\"", s, err)
		}
		shown := strings.Join(strings.Fields(stderr.String()), " ")
		if n := strings.Count(shown, warnWords(unix.EBADF.Error())); n != 1 {
			t.Fatalf("the warning shown %d times; want once:\n%s", n, stderr)
		}
	})
}

// failingWait is an openGated whose gated reader's wait fails with EBADF:
// the reader falls back inside its first read, on inline.Reader's
// goroutine.
func failingWait(t *testing.T) func(*os.File, term.GatedOptions) (*term.Gated, error) {
	return func(_ *os.File, o term.GatedOptions) (*term.Gated, error) {
		o.Wait = func([]int) ([]bool, error) { return nil, unix.EBADF }
		g, _ := gatedPipe(t, o)
		return g, nil
	}
}

// A56 (rows R80, R132, R143, R157): the program's give-up flushes a POSIX
// terminal's queue whatever reads it, the gate or standard input as built
// after a fallback at start-up or later; a terminal with no cancellable
// read (Windows) is never flushed there.
func TestGiveUpFlushBySystem(t *testing.T) {
	for _, c := range []struct {
		name   string
		open   func(t *testing.T) func(*os.File, term.GatedOptions) (*term.Gated, error)
		flushs int64
	}{
		{"a gateless input that is not a POSIX terminal", func(*testing.T) func(*os.File, term.GatedOptions) (*term.Gated, error) {
			return func(*os.File, term.GatedOptions) (*term.Gated, error) { return nil, term.ErrNoGate }
		}, 0},
		{"a fallback at start-up", func(*testing.T) func(*os.File, term.GatedOptions) (*term.Gated, error) {
			return func(*os.File, term.GatedOptions) (*term.Gated, error) { return nil, errors.New("no terminal path") }
		}, 1},
		{"a later fallback", failingWait, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			e, _, _ := terminalEnv(t)
			replace(t, &openGated, c.open(t))
			var flushes atomic.Int64
			replace(t, &termFlush, func(*os.File) error { flushes.Add(1); return nil })
			q := &questions{ctx: t.Context(), in: e.terminalIn(), out: io.Discard}
			if err := giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err }); !errors.Is(err, context.Canceled) {
				t.Fatalf("the program's give-up: %v", err)
			}
			if n := flushes.Load(); n != c.flushs {
				t.Fatalf("%d flushes at the program's give-up; want %d", n, c.flushs)
			}
		})
	}
}

// A57 and A63 (rows R138, R154), through terminalIn: each refusal of
// NewGated is a fallback to standard input as built, with Q24's warning
// once, in the refusal's words.
func TestGatedRefusals(t *testing.T) {
	for _, c := range []struct {
		name string
		gate func(t *testing.T, o term.GatedOptions) (*term.Gated, error)
	}{
		{"the wake pipe cannot be made", func(t *testing.T, o term.GatedOptions) (*term.Gated, error) {
			o.Pipe = func([]int) error { return unix.EMFILE }
			return term.NewGated(pipeDescriptor(t, true), o)
		}},
		{"a descriptor at FD_SETSIZE", func(t *testing.T, o term.GatedOptions) (*term.Gated, error) {
			return term.NewGated(highCopy(t, pipeDescriptor(t, true)), o)
		}},
		{"a blocking descriptor", func(t *testing.T, o term.GatedOptions) (*term.Gated, error) {
			return term.NewGated(pipeDescriptor(t, false), o)
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e, w, stderr := terminalEnv(t)
			var refusal error
			replace(t, &openGated, func(_ *os.File, o term.GatedOptions) (*term.Gated, error) {
				g, err := c.gate(t, o)
				if g != nil {
					t.Cleanup(g.Stop)
				}
				refusal = err
				return g, err
			})
			q := &questions{ctx: t.Context(), in: e.terminalIn(), out: io.Discard}
			if refusal == nil {
				t.Fatal("NewGated took the descriptor; want it refused")
			}
			writeTo(t, w, "y\n")
			if s, err := q.line(); err != nil || s != "y" {
				t.Fatalf("the prompt read %q, %v; want \"y\" from standard input as built", s, err)
			}
			shown := strings.Join(strings.Fields(stderr.String()), " ")
			if n := strings.Count(shown, warnWords(refusal.Error())); n != 1 {
				t.Fatalf("the warning shown %d times; want once:\n%s", n, stderr)
			}
		})
	}
}

// pipeDescriptor is a pipe's read end as a descriptor, non-blocking when
// nb; the pipe is closed in t.Cleanup.
func pipeDescriptor(t *testing.T, nb bool) int {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Dup(int(r.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	if err := unix.SetNonblock(fd, nb); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = w.Close()
		_ = unix.Close(fd)
	})
	return fd
}

// highCopy is a copy of fd at FD_SETSIZE, closed in t.Cleanup. Go raises
// the open-files limit to the hard limit at start-up, so the copy fits; a
// copy that cannot be made fails the test.
func highCopy(t *testing.T, fd int) int {
	t.Helper()
	hi := unix.FD_SETSIZE
	if err := unix.Dup2(fd, hi); err != nil {
		t.Fatalf("copy descriptor %d to %d: %v", fd, hi, err)
	}
	t.Cleanup(func() { _ = unix.Close(hi) })
	return hi
}

// A58 (row R124), in the default suite: on a terminal, terminalIn's input
// has the gated reader openGated gives as its gate, and is a POSIX
// terminal's.
func TestTerminalInWiresGate(t *testing.T) {
	e, _, stderr := terminalEnv(t)
	var made *term.Gated
	replace(t, &openGated, func(_ *os.File, o term.GatedOptions) (*term.Gated, error) {
		made, _ = gatedPipe(t, o)
		return made, nil
	})
	ti := e.terminalIn()
	if made == nil {
		t.Fatal("terminalIn did not open the gated reader")
	}
	if g, ok := ti.gate.(*term.Gated); !ok || g != made {
		t.Fatalf("the input's gate is %#v; want the gated reader openGated gave", ti.gate)
	}
	if !ti.posix {
		t.Fatal("a gated input is not marked a POSIX terminal's")
	}
	if strings.Contains(stderr.String(), "cannot wait") {
		t.Fatalf("a gated input warned:\n%s", stderr)
	}
}

// A60 (row R156): a fallback inside a read on inline.Reader's goroutine
// writes nothing then; Q24's warning comes once, on the main flow, before
// the next question's text, or at the run's end when no text comes.
func TestFallbackWarningOnMainFlow(t *testing.T) {
	t.Run("before the next question's text", func(t *testing.T) {
		e, w, stderr := terminalEnv(t)
		replace(t, &openGated, failingWait(t))
		q := &questions{ctx: t.Context(), in: e.terminalIn(), out: e.out.err}
		writeTo(t, w, "first\n")
		if s, err := q.line(); err != nil || s != "first" {
			t.Fatalf("the read that fell back gave %q, %v; want \"first\"", s, err)
		}
		if strings.Contains(stderr.String(), "cannot wait") {
			t.Fatalf("the warning was written while the read that fell back ran:\n%s", stderr)
		}
		writeTo(t, w, "second\n")
		if s, err := q.input("Next"); err != nil || s != "second" {
			t.Fatalf("the next question read %q, %v; want \"second\"", s, err)
		}
		shown := strings.Join(strings.Fields(stderr.String()), " ")
		warn := warnWords(unix.EBADF.Error())
		if n := strings.Count(shown, warn); n != 1 {
			t.Fatalf("the warning shown %d times; want once:\n%s", n, stderr)
		}
		if i, j := strings.Index(shown, warn), strings.Index(shown, "Next:"); i > j {
			t.Fatalf("the warning came after the question's text:\n%s", stderr)
		}
	})
	// (2026-11-21, build pass 5: a fallback, then two later questions: the
	// warning once, so its queue is cleared when it is written.)
	t.Run("written once over two later questions", func(t *testing.T) {
		e, w, stderr := terminalEnv(t)
		replace(t, &openGated, failingWait(t))
		q := &questions{ctx: t.Context(), in: e.terminalIn(), out: e.out.err}
		writeTo(t, w, "first\n")
		if s, err := q.line(); err != nil || s != "first" {
			t.Fatalf("the read that fell back gave %q, %v; want \"first\"", s, err)
		}
		for _, want := range []string{"second", "third"} {
			writeTo(t, w, want+"\n")
			if s, err := q.input("Next"); err != nil || s != want {
				t.Fatalf("a later question read %q, %v; want %q", s, err, want)
			}
		}
		shown := strings.Join(strings.Fields(stderr.String()), " ")
		if n := strings.Count(shown, warnWords(unix.EBADF.Error())); n != 1 {
			t.Fatalf("the warning shown %d times over two later questions; want once:\n%s", n, stderr)
		}
	})
	// (2026-11-21, build pass 5: at the session prompt the terminal is raw;
	// the warning is written out of raw mode, before the editor's prompt.)
	t.Run("at the session prompt, out of raw mode", func(t *testing.T) {
		h := newLineHarness(t, &acptest.ScriptAgent{})
		t.Setenv(provider.CredentialVar(provider.KeyProviders()[0]), "test-key") // no sign-in offer: the session prompt comes first
		rw := &rawAtWarning{h: h, w: h.stderr}
		h.e.out.err = rw
		h.e.queueWaitWarn(unix.EBADF) // as a fallback on inline.Reader's goroutine queues it
		h.start(Prompt{})
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "> ") })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatalf("session ended with %v", err)
		}
		depths := rw.seen()
		if len(depths) != 1 {
			t.Fatalf("the warning written %d times; want once:\n%s", len(depths), h.stderr)
		}
		if depths[0] != 0 {
			t.Fatalf("the warning was written with the terminal raw (%d raw modes not restored); want it out of raw mode", depths[0])
		}
	})
	t.Run("at the run's end", func(t *testing.T) {
		t.Setenv("GOBBLE_HOME", t.TempDir())
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = w.Close()
			_ = r.Close()
		})
		replace(t, &termRaw, func(*os.File) (func() error, error) { return func() error { return nil }, nil })
		replace(t, &isTerminalFile, func(*os.File) bool { return true })
		replace(t, &openGated, failingWait(t))
		replace(t, &signInWith, func(ctx context.Context, _ string, p provider.Prompter, _ provider.SignInOptions, _ provider.LoginStore) (provider.Credential, error) {
			_, _ = p.Input(ctx, "Paste the redirect URL")
			return provider.Credential{}, errLoopback
		})
		writeTo(t, w, "pasted\n")
		stdout, stderr := &syncBuffer{}, &syncBuffer{}
		var root Root
		k, err := newParser(&root, stdout, stderr)
		if err != nil {
			t.Fatal(err)
		}
		caps := term.Caps{In: true, Out: term.Stream{TTY: true, Width: 80}, Err: term.Stream{Width: 80}}
		out := &Output{out: stdout, err: stderr, caps: caps}
		_ = run(t.Context(), k, &root, []string{"auth", "login", "grok"}, r, out, &interrupts{})
		shown := strings.Join(strings.Fields(stderr.String()), " ")
		warn := warnWords(unix.EBADF.Error())
		if n := strings.Count(shown, warn); n != 1 {
			t.Fatalf("the warning shown %d times at the run's end; want once:\n%s", n, stderr)
		}
		if i, j := strings.Index(shown, warn), strings.Index(shown, "Paste the redirect URL:"); j < 0 || i < j {
			t.Fatalf("the warning came before the question's text, not at the run's end:\n%s", stderr)
		}
	})
}

// rawAtWarning passes stderr's writes on, and notes, at each write of Q24's
// warning, how many of the harness's raw modes are not yet restored.
type rawAtWarning struct {
	h      *lineHarness
	w      io.Writer
	mu     sync.Mutex
	depths []int
}

func (r *rawAtWarning) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "gobble cannot wait") {
		r.h.mu.Lock()
		d := r.h.raws - r.h.restores
		r.h.mu.Unlock()
		r.mu.Lock()
		r.depths = append(r.depths, d)
		r.mu.Unlock()
	}
	return r.w.Write(p)
}

func (r *rawAtWarning) seen() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int(nil), r.depths...)
}

// A61 (row R147): a read that ended on its context is marked with the size
// asked, lineBuf: a race completion of a 256-byte read the program gave up
// is dropped whole, and the next 256-byte read gets none of it.
func TestGatedRaceDropSize(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })
	ti := newTerminalInput(inline.NewReader(pr), nil)
	ti.gate = tookGate{}
	if err := giveUp(t, func(ctx context.Context) error { _, err := ti.Read(ctx, make([]byte, 256)); return err }); !errors.Is(err, context.Canceled) {
		t.Fatalf("the given-up read: %v", err)
	}
	ti.abandon()
	go func() {
		_, _ = io.WriteString(pw, strings.Repeat("p", 400))
		_, _ = io.WriteString(pw, "next")
	}()
	bounded, stop := context.WithTimeout(t.Context(), 3*time.Second)
	defer stop()
	buf := make([]byte, 256)
	if n, err := ti.Read(bounded, buf); err != nil || string(buf[:n]) != "next" {
		t.Fatalf("the next 256-byte read gave %q, %v; want \"next\": the 400 bytes dropped whole", buf[:n], err)
	}
}

// A52, as amended (row R148): through the terminal input, the read's EIO
// is the line's error, and the input has ended.
func TestCancelableHangUp(t *testing.T) {
	g, w := gatedPipe(t, term.GatedOptions{Read: func(int, []byte) (int, error) { return -1, unix.EIO }})
	ti := gatedInput(g, nil)
	writeTo(t, w, "h")
	if _, err := lineWithin(t, ti, 3*time.Second); !errors.Is(err, unix.EIO) {
		t.Fatalf("the line's error is %v; want EIO", err)
	}
	if !ti.ended() {
		t.Fatal("after EIO the input has not ended")
	}
}

// A46 (rows R124, R137): a pipe is read as it is, with no gate and no
// warning.
func TestTerminalInChoosesLayer(t *testing.T) {
	e, _, stderr := pipeEnv(t)
	if ti := e.terminalIn(); ti.gate != nil {
		t.Fatal("a pipe was given the cancellable reader")
	}
	if strings.Contains(stderr.String(), "cannot wait") {
		t.Fatalf("a pipe warned:\n%s", stderr)
	}
}
