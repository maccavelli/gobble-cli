package cli

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
)

// The readers of the terminal besides the questions and the editor, each
// through the one terminal input (0063-PLAN P2 step 8, A23, A27; 0063-MADR
// D2, D13, D15, D17; rows R45, R46, R50, R60).

// ctxChunks gives its chunks one read at a time, then the end of input,
// and records for each read whether its context had a deadline.
type ctxChunks struct {
	mu        sync.Mutex
	c         []string
	deadlines []bool
}

func (r *ctxChunks) Read(ctx context.Context, p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := ctx.Deadline()
	r.deadlines = append(r.deadlines, ok)
	if len(r.c) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.c[0])
	r.c = r.c[1:]
	return n, nil
}

func (r *ctxChunks) reads() []bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]bool(nil), r.deadlines...)
}

// trustEnv is a run on a project that needs trust, its terminal input over
// in, and stdin a file holding fromStdin, which the question must not read.
func trustEnv(t *testing.T, in keysIn, fromStdin string) (*runEnv, string) {
	t.Helper()
	fakeRaw(t)
	t.Setenv("GOBBLE_HOME", t.TempDir())
	proj := trustProject(t)
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, stdin: stdinFile(t, fromStdin)}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	e.termIn = newTerminalInput(in, nil)
	return e, proj
}

// Step 8 (rows R45, R46): the trust question's key is read through the
// process's terminal input, not from stdin itself.
func TestTrustReadsTerminalInput(t *testing.T) {
	in := &ctxChunks{c: []string{"1"}}
	e, proj := trustEnv(t, in, "4")
	e.decideTrustHere(proj)
	if _, d := e.projectTrust(proj); !d.Trusted {
		t.Fatalf("the decision %+v; want trusted, the key the terminal input gave", d)
	}
	if n := len(in.reads()); n != 1 {
		t.Fatalf("the terminal input was read %d times; want once", n)
	}
}

// A27 (row R50, Q12): the trust question reads with the run's context and
// no deadline, even where the asker waits on the clock (a lone Esc).
func TestTrustReadsWithoutDeadline(t *testing.T) {
	in := &ctxChunks{c: []string{"\x1b", "1", "1"}}
	e, proj := trustEnv(t, in, "")
	e.decideTrustHere(proj)
	if _, d := e.projectTrust(proj); !d.Trusted {
		t.Fatalf("the decision %+v; want trusted", d)
	}
	reads := in.reads()
	if len(reads) < 2 {
		t.Fatalf("%d reads; want the Esc's and the answer's", len(reads))
	}
	for i, deadline := range reads {
		if deadline {
			t.Fatalf("read %d had a deadline; want none (Q12)", i+1)
		}
	}
}

// noRaw is a raw mode that does nothing.
func noRaw() (func() error, error) { return func() error { return nil }, nil }

// permissionSession is a line session that asks the permission prompt
// through a terminal input over in, its stdin a file holding fromStdin.
func permissionSession(t *testing.T, in keysIn, fromStdin string) *lineSession {
	t.Helper()
	out := &Output{out: io.Discard, err: io.Discard}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, stdin: stdinFile(t, fromStdin)}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	return &lineSession{e: e, keys: newTerminalInput(in, nil), raw: noRaw}
}

// writeRequest is a tool call's request to write a.txt.
func writeRequest() acpclient.PermissionRequest {
	return acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "write a.txt"}, Options: toolOptions()}
}

// A23 (row R60): the permission prompt's key comes through the session's
// terminal input; and (D13, D17) a request withdrawn while its read waits
// leaves that read to be dropped, where the person's Ctrl+C does not.
func TestPermissionReadsTerminalInput(t *testing.T) {
	t.Run("one key", func(t *testing.T) {
		in := &countedChunks{c: []string{"1"}}
		ls := permissionSession(t, in, "5")
		if a := ls.askPermission(t.Context(), writeRequest()); a.OptionID != "allow_once" || in.count() != 1 {
			t.Fatalf("the answer %+v after %d reads; want allow_once, read once from the terminal input", a, in.count())
		}
	})
	t.Run("withdrawn", func(t *testing.T) {
		in := &sentIn{results: make(chan stubResult)}
		ls := permissionSession(t, in, "")
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel() // the agent withdrew the request
		}()
		if a := ls.askPermission(ctx, writeRequest()); a.OptionID != "" {
			t.Fatalf("a withdrawn request answered %+v", a)
		}
		go func() {
			in.results <- stubResult{b: "1"}      // what completes the read left waiting
			in.results <- stubResult{b: "next\n"} // the next prompt's
		}()
		if s, err := lineWithin(t, ls.keys, 3*time.Second); err != nil || s != "next" {
			t.Fatalf("the next line = %q, %v; want \"next\", the withdrawn read's completion dropped", s, err)
		}
	})
	t.Run("Ctrl+C", func(t *testing.T) {
		in := &countedChunks{c: []string{"\x03", "next\n"}}
		ls := permissionSession(t, in, "")
		if a := ls.askPermission(t.Context(), writeRequest()); a.OptionID != "" {
			t.Fatalf("Ctrl+C answered %+v", a)
		}
		if s, err := ls.keys.line(t.Context()); err != nil || s != "next" {
			t.Fatalf("the next line = %q, %v; want \"next\"", s, err)
		}
	})
}
