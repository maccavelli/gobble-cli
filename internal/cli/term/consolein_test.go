package term

import (
	"errors"
	"io"
	"os"
	"runtime"
	"testing"
	"time"
)

// The console rule of 0063-MADR D19 and D20, over a stubbed console: rows
// R109, R111, R112, R114 to R117 and R122 (0063-PLAN P2 step 13, A30, A32,
// A34 to A39). They run on every system: the rule is portable, and only the
// question "is standard input a console" is Windows's.

// stubRead is what one read of the stub gives.
type stubRead struct {
	b   string
	err error
}

// consoleStub is a console whose reads wait for the test: each Read says it
// started, then gives what the test sends.
type consoleStub struct {
	started chan struct{}
	results chan stubRead
}

func newConsoleStub() *consoleStub {
	return &consoleStub{started: make(chan struct{}, 8), results: make(chan stubRead)}
}

func (s *consoleStub) Read(p []byte) (int, error) {
	s.started <- struct{}{}
	r := <-s.results
	return copy(p, r.b), r.err
}

// readDone is one Read of the reader under test: what it gave, and when.
type readDone struct {
	b   string
	err error
	at  time.Time
}

// startRead reads r once, in the background.
func startRead(r io.Reader) <-chan readDone {
	done := make(chan readDone, 1)
	go func() {
		buf := make([]byte, 64)
		n, err := r.Read(buf)
		done <- readDone{string(buf[:n]), err, time.Now()}
	}()
	return done
}

// waitStarted waits for the stub's next read to start.
func (s *consoleStub) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-s.started:
	case <-time.After(2 * time.Second):
		t.Fatal("no read of the console started")
	}
}

// give answers the read that started with b and err.
func (s *consoleStub) give(t *testing.T, b string, err error) {
	t.Helper()
	select {
	case s.results <- stubRead{b, err}:
	case <-time.After(2 * time.Second):
		t.Fatal("no read of the console took the answer")
	}
}

// result waits for a read's result, or fails when the stub's console is
// read again first: the empty read was excused where it must not be.
func (s *consoleStub) result(t *testing.T, done <-chan readDone) readDone {
	t.Helper()
	select {
	case r := <-done:
		return r
	case <-s.started:
		t.Fatal("the console was read again: an empty read was excused")
	case <-time.After(3 * time.Second):
		t.Fatal("the read did not return")
	}
	return readDone{}
}

// A30 (row R109): with the mark set while the read waits, the read that
// gives nothing is read again, and the reader gives what comes next, with
// no end of input.
func TestConsoleReadAgainAfterInterrupt(t *testing.T) {
	stub, mark := newConsoleStub(), &InterruptMark{}
	done := startRead(NewConsoleReader(stub, true, mark, MarkWait))
	stub.waitStarted(t)
	mark.Interrupted()
	stub.give(t, "", io.EOF)
	stub.waitStarted(t)
	stub.give(t, "n\r\n", nil)
	if r := stub.result(t, done); r.b != "n\r\n" || r.err != nil {
		t.Fatalf("the reader gave %q, %v; want \"n\\r\\n\" and no error", r.b, r.err)
	}
}

// A32 (row R112): Ctrl+C, then Ctrl+Z at the same read: the first empty
// read is excused, the second, with no new interrupt, is the end of input.
func TestConsoleCtrlZAfterInterruptEnds(t *testing.T) {
	stub, mark := newConsoleStub(), &InterruptMark{}
	done := startRead(NewConsoleReader(stub, true, mark, MarkWait))
	stub.waitStarted(t)
	mark.Interrupted()
	stub.give(t, "", io.EOF)
	stub.waitStarted(t)
	stub.give(t, "", io.EOF)
	if r := stub.result(t, done); !errors.Is(r.err, io.EOF) || r.b != "" {
		t.Fatalf("the reader gave %q, %v; want the end of input", r.b, r.err)
	}
}

// A34 (row R115): with no mark, the byte 0x03 a raw read gives is given as
// it is, in one read.
func TestConsoleRawCtrlCIsAByte(t *testing.T) {
	stub := newConsoleStub()
	done := startRead(NewConsoleReader(stub, true, &InterruptMark{}, MarkWait))
	stub.waitStarted(t)
	stub.give(t, "\x03", nil)
	if r := stub.result(t, done); r.b != "\x03" || r.err != nil {
		t.Fatalf("the reader gave %q, %v; want the byte 0x03", r.b, r.err)
	}
}

// A35 (row R117): standard input that is not a console: an interrupt while
// its read waits excuses nothing, and a read that gives nothing is the end
// of input, at once.
func TestConsoleRuleOnlyOnAConsole(t *testing.T) {
	stub, mark := newConsoleStub(), &InterruptMark{}
	done := startRead(NewConsoleReader(stub, false, mark, MarkWait))
	stub.waitStarted(t)
	mark.Interrupted()
	start := time.Now()
	stub.give(t, "", io.EOF)
	r := stub.result(t, done)
	if !errors.Is(r.err, io.EOF) {
		t.Fatalf("the reader gave %q, %v; want the end of input", r.b, r.err)
	}
	if d := r.at.Sub(start); d >= MarkWait {
		t.Fatalf("the end of input took %v: a pipe or a file waits for no mark", d)
	}
}

// (2026-11-21, P3-3, the owner's "As above, plus tighten the P1 test"
// ruling on plant 70; row R118): off Windows, isConsoleInput is false for any
// file, so ConsoleInput gives the file itself, with no rule over it.
func TestIsConsoleInputOffWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a Windows handle may be a console")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pr.Close(); _ = pw.Close() })
	plain, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = plain.Close() })
	for name, f := range map[string]*os.File{"a pipe": pr, "a file": plain} {
		if isConsoleInput(f) {
			t.Fatalf("isConsoleInput(%s) is true off Windows", name)
		}
		if got := ConsoleInput(f, &InterruptMark{}); got != io.Reader(f) {
			t.Fatalf("ConsoleInput(%s) = %T; want the file itself", name, got)
		}
	}
}

// A36 (row R111): the mark excuses one empty read; and a read that returns
// bytes clears it, so an empty read after them is the end of input.
func TestConsoleMarkExcusesOneRead(t *testing.T) {
	t.Run("one", func(t *testing.T) {
		stub, mark := newConsoleStub(), &InterruptMark{}
		done := startRead(NewConsoleReader(stub, true, mark, MarkWait))
		stub.waitStarted(t)
		mark.Interrupted()
		stub.give(t, "", io.EOF)
		stub.waitStarted(t) // excused, read again
		stub.give(t, "", io.EOF)
		if r := stub.result(t, done); !errors.Is(r.err, io.EOF) {
			t.Fatalf("the second empty read gave %q, %v; want the end of input", r.b, r.err)
		}
	})
	t.Run("bytes clear it", func(t *testing.T) {
		stub, mark := newConsoleStub(), &InterruptMark{}
		r := NewConsoleReader(stub, true, mark, MarkWait)
		done := startRead(r)
		stub.waitStarted(t)
		mark.Interrupted()
		stub.give(t, "x", nil)
		if got := stub.result(t, done); got.b != "x" || got.err != nil {
			t.Fatalf("the reader gave %q, %v; want \"x\"", got.b, got.err)
		}
		done = startRead(r)
		stub.waitStarted(t)
		stub.give(t, "", io.EOF)
		if got := stub.result(t, done); !errors.Is(got.err, io.EOF) {
			t.Fatalf("an empty read after bytes gave %q, %v; want the end of input", got.b, got.err)
		}
	})
}

// A37 (row R114): an interrupt while no read of the console waits sets no
// mark: a later read that gives nothing is the end of input, after the
// wait.
func TestConsoleNoMarkWithoutWaitingRead(t *testing.T) {
	stub, mark := newConsoleStub(), &InterruptMark{}
	r := NewConsoleReader(stub, true, mark, MarkWait)
	mark.Interrupted()
	done := startRead(r)
	stub.waitStarted(t)
	start := time.Now()
	stub.give(t, "", io.EOF)
	got := stub.result(t, done)
	if !errors.Is(got.err, io.EOF) {
		t.Fatalf("the reader gave %q, %v; want the end of input", got.b, got.err)
	}
	if d := got.at.Sub(start); d < MarkWait {
		t.Fatalf("the end of input came after %v; want the %v wait first", d, MarkWait)
	}
}

// A38 (row R116): the read gives nothing before the interrupt is marked;
// a mark 30 ms later excuses it, and the next bytes come.
func TestConsoleWaitsForLateMark(t *testing.T) {
	stub, mark := newConsoleStub(), &InterruptMark{}
	done := startRead(NewConsoleReader(stub, true, mark, MarkWait))
	stub.waitStarted(t)
	stub.give(t, "", io.EOF)
	time.Sleep(30 * time.Millisecond)
	mark.Interrupted()
	select {
	case <-stub.started:
	case r := <-done:
		t.Fatalf("the reader gave %q, %v before the late mark; want the read made again", r.b, r.err)
	case <-time.After(2 * time.Second):
		t.Fatal("the console was not read again")
	}
	stub.give(t, "n\r\n", nil)
	if r := stub.result(t, done); r.b != "n\r\n" || r.err != nil {
		t.Fatalf("the reader gave %q, %v; want \"n\\r\\n\"", r.b, r.err)
	}
}

// A39 (rows R116, R122): an empty read with no mark is the end of input
// after at least 100 ms and well under a second; a mark at 600 ms, with no
// read waiting, excuses nothing.
func TestConsoleWaitBound(t *testing.T) {
	stub, mark := newConsoleStub(), &InterruptMark{}
	r := NewConsoleReader(stub, true, mark, MarkWait)
	done := startRead(r)
	stub.waitStarted(t)
	start := time.Now()
	stub.give(t, "", io.EOF)
	got := stub.result(t, done)
	if !errors.Is(got.err, io.EOF) {
		t.Fatalf("the reader gave %q, %v; want the end of input", got.b, got.err)
	}
	if d := got.at.Sub(start); d < 100*time.Millisecond || d >= 500*time.Millisecond {
		t.Fatalf("the end of input came after %v; want at least 100 ms and under 500 ms", d)
	}
	time.Sleep(time.Until(start.Add(600 * time.Millisecond)))
	mark.Interrupted()
	done = startRead(r)
	stub.waitStarted(t)
	stub.give(t, "", io.EOF)
	if got := stub.result(t, done); !errors.Is(got.err, io.EOF) {
		t.Fatalf("after a mark with no read waiting, an empty read gave %q, %v; want the end of input", got.b, got.err)
	}
}
