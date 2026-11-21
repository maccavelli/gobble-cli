//go:build unix

package term

import (
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// The cancellable reader of a POSIX terminal (0063-MADR D22; PLAN P2 steps
// 15 and 19, A41, A48, A52, A54; rows R125 to R127, R134, R138, R142, R145,
// R146, R148). A pipe stands in for the private descriptor: select(2)
// serves it as it serves a terminal.

// pipeGated is a Gated reader over a pipe's read end, made non-blocking,
// and the pipe's write end; it is stopped in t.Cleanup.
func pipeGated(t *testing.T, o GatedOptions) (*Gated, *os.File) {
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
	g, err := NewGated(fd, o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = w.Close()
		g.Stop()
	})
	return g, w
}

// readResult is one Read's result.
type readResult struct {
	s   string
	err error
}

// readAsync reads g once in the background.
func readAsync(g *Gated) <-chan readResult {
	done := make(chan readResult, 1)
	go func() {
		b := make([]byte, 64)
		n, err := g.Read(b)
		done <- readResult{string(b[:n]), err}
	}()
	return done
}

func within(t *testing.T, done <-chan readResult, d time.Duration) readResult {
	t.Helper()
	select {
	case r := <-done:
		return r
	case <-time.After(d):
		t.Fatalf("the read did not return within %v", d)
	}
	return readResult{}
}

func notWithin(t *testing.T, done <-chan readResult, d time.Duration) {
	t.Helper()
	select {
	case r := <-done:
		t.Fatalf("the read returned %q, %v; want it still waiting", r.s, r.err)
	case <-time.After(d):
	}
}

// countingRead counts read(2) through the seam.
type countingRead struct{ n atomic.Int64 }

func (c *countingRead) read(fd int, p []byte) (int, error) {
	c.n.Add(1)
	return unix.Read(fd, p)
}

// A41 (rows R125 to R127, R134), with A52 (row R148).
func TestCancelableWait(t *testing.T) {
	t.Run("a read only after data, with the gate open", func(t *testing.T) {
		var c countingRead
		g, w := pipeGated(t, GatedOptions{Read: c.read})
		g.Open()
		done := readAsync(g)
		notWithin(t, done, 150*time.Millisecond)
		if n := c.n.Load(); n != 0 {
			t.Fatalf("%d reads before any data; want none", n)
		}
		if _, err := w.WriteString("abc"); err != nil {
			t.Fatal(err)
		}
		if r := within(t, done, 2*time.Second); r.s != "abc" || r.err != nil || c.n.Load() != 1 {
			t.Fatalf("read %q, %v after %d reads; want \"abc\" after one", r.s, r.err, c.n.Load())
		}
	})
	t.Run("a close before data reads nothing; what comes after is read whole after the next opening", func(t *testing.T) {
		var c countingRead
		g, w := pipeGated(t, GatedOptions{Read: c.read})
		g.Open()
		done := readAsync(g)
		time.Sleep(50 * time.Millisecond)
		if g.Close() {
			t.Fatal("the close reported a read; none was made")
		}
		if _, err := w.WriteString("xyz"); err != nil {
			t.Fatal(err)
		}
		notWithin(t, done, 200*time.Millisecond)
		if n := c.n.Load(); n != 0 {
			t.Fatalf("%d reads with the gate closed; want none", n)
		}
		g.Open()
		if r := within(t, done, 2*time.Second); r.s != "xyz" || r.err != nil {
			t.Fatalf("read %q, %v; want \"xyz\"", r.s, r.err)
		}
	})
	t.Run("a close wakes the wait and returns after the waiter left it", func(t *testing.T) {
		var returns atomic.Int64
		g, _ := pipeGated(t, GatedOptions{Returned: func() { returns.Add(1) }})
		g.Open()
		done := readAsync(g)
		time.Sleep(50 * time.Millisecond)
		closed := make(chan bool, 1)
		go func() { closed <- g.Close() }()
		select {
		case <-closed:
		case <-time.After(time.Second):
			t.Fatal("the close did not return: the wait was not woken")
		}
		if returns.Load() == 0 {
			t.Fatal("the close returned before the wait did")
		}
		notWithin(t, done, 100*time.Millisecond)
	})
	t.Run("a close during a read waits for it and reports it", func(t *testing.T) {
		started, release := make(chan struct{}), make(chan struct{})
		g, w := pipeGated(t, GatedOptions{Read: func(fd int, p []byte) (int, error) {
			close(started)
			<-release
			return unix.Read(fd, p)
		}})
		g.Open()
		done := readAsync(g)
		if _, err := w.WriteString("q"); err != nil {
			t.Fatal(err)
		}
		<-started
		closed := make(chan bool, 1)
		go func() { closed <- g.Close() }()
		select {
		case <-closed:
			t.Fatal("the close returned while a read was in progress")
		case <-time.After(100 * time.Millisecond):
		}
		close(release)
		if took := <-closed; !took {
			t.Fatal("the close did not report the read")
		}
		if r := within(t, done, 2*time.Second); r.s != "q" {
			t.Fatalf("read %q, %v; want \"q\"", r.s, r.err)
		}
	})
	t.Run("EINTR from the wait is retried", func(t *testing.T) {
		var calls atomic.Int64
		g, w := pipeGated(t, GatedOptions{Wait: func(fds []int) ([]bool, error) {
			if calls.Add(1) == 1 {
				return nil, unix.EINTR
			}
			return selectRead(fds)
		}})
		g.Open()
		done := readAsync(g)
		if _, err := w.WriteString("i"); err != nil {
			t.Fatal(err)
		}
		if r := within(t, done, 2*time.Second); r.s != "i" || r.err != nil {
			t.Fatalf("read %q, %v; want \"i\": EINTR retried", r.s, r.err)
		}
	})
	t.Run("EAGAIN returns to the wait", func(t *testing.T) {
		var calls atomic.Int64
		g, w := pipeGated(t, GatedOptions{Read: func(fd int, p []byte) (int, error) {
			if calls.Add(1) == 1 {
				return -1, unix.EAGAIN
			}
			return unix.Read(fd, p)
		}})
		g.Open()
		done := readAsync(g)
		if _, err := w.WriteString("e"); err != nil {
			t.Fatal(err)
		}
		if r := within(t, done, 2*time.Second); r.s != "e" || r.err != nil {
			t.Fatalf("read %q, %v; want \"e\" after EAGAIN", r.s, r.err)
		}
	})
	t.Run("EINTR from the read returns to the wait", eintrReadWaitsAgain)
	t.Run("the end of input is exactly (0, io.EOF)", func(t *testing.T) {
		g, w := pipeGated(t, GatedOptions{})
		g.Open()
		done := readAsync(g)
		_ = w.Close()
		r := within(t, done, 2*time.Second)
		if r.s != "" || r.err != io.EOF { //nolint:errorlint // exactly io.EOF, as inline.Reader keeps it
			t.Fatalf("read %q, %v; want exactly (0, io.EOF)", r.s, r.err)
		}
	})
}

// eintrReadWaitsAgain is A62 (row R153), a case of TestCancelableWait:
// EINTR from read(2) returns to the wait, as EAGAIN does; never an error and
// never the end of input.
func eintrReadWaitsAgain(t *testing.T) {
	var calls atomic.Int64
	g, w := pipeGated(t, GatedOptions{Read: func(fd int, p []byte) (int, error) {
		if calls.Add(1) == 1 {
			return -1, unix.EINTR
		}
		return unix.Read(fd, p)
	}})
	g.Open()
	done := readAsync(g)
	if _, err := w.WriteString("r"); err != nil {
		t.Fatal(err)
	}
	if r := within(t, done, 2*time.Second); r.s != "r" || r.err != nil || calls.Load() != 2 {
		t.Fatalf("read %q, %v after %d reads; want \"r\" after EINTR, then a read", r.s, r.err, calls.Load())
	}
}

// A52 (row R148): a hang-up reads 0, the end of input (above); EIO is the
// input's error.
func TestCancelableHangUp(t *testing.T) {
	t.Run("a hang-up's EIO is the read's error", func(t *testing.T) {
		g, w := pipeGated(t, GatedOptions{Read: func(int, []byte) (int, error) { return -1, unix.EIO }})
		g.Open()
		done := readAsync(g)
		if _, err := w.WriteString("h"); err != nil {
			t.Fatal(err)
		}
		if r := within(t, done, 2*time.Second); !errors.Is(r.err, unix.EIO) {
			t.Fatalf("read %q, %v; want EIO", r.s, r.err)
		}
	})
}

// A50's part here (row R138): a wait that fails falls back, once, to
// reading the fallback as built, and every close then reports a read.
func TestCancelableFallsBack(t *testing.T) {
	var told []error
	var mu sync.Mutex
	g, _ := pipeGated(t, GatedOptions{
		Wait:       func([]int) ([]bool, error) { return nil, unix.EBADF },
		Fallback:   strings.NewReader("fb1fb2"),
		OnFallback: func(err error) { mu.Lock(); told = append(told, err); mu.Unlock() },
	})
	g.Open()
	b := make([]byte, 3)
	for _, want := range []string{"fb1", "fb2"} {
		n, err := g.Read(b)
		if err != nil || string(b[:n]) != want {
			t.Fatalf("read %q, %v; want %q from the fallback", b[:n], err, want)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(told) != 1 || !errors.Is(told[0], unix.EBADF) {
		t.Fatalf("told %v; want the wait's error once", told)
	}
	if !g.Close() {
		t.Fatal("after the fallback a close must report a read (D13's drop)")
	}
}

// A54 (row R146): the wake pipe's ends are non-blocking and close-on-exec;
// closes with no waiter never block; Stop ends a waiting Read with io.EOF.
func TestCancelableWakePipe(t *testing.T) {
	g, _ := pipeGated(t, GatedOptions{})
	for _, fd := range []int{g.wakeR, g.wakeW} {
		fl, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
		if err != nil || fl&unix.O_NONBLOCK == 0 {
			t.Fatalf("wake pipe end %d: flags %#x, %v; want O_NONBLOCK", fd, fl, err)
		}
		fd2, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
		if err != nil || fd2&unix.FD_CLOEXEC == 0 {
			t.Fatalf("wake pipe end %d: descriptor flags %#x, %v; want FD_CLOEXEC", fd, fd2, err)
		}
	}
	closed := make(chan struct{})
	go func() {
		for range 70000 { // more than a pipe holds
			g.Close()
		}
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("closes with no waiter blocked on the wake pipe")
	}
	g.Open()
	done := readAsync(g)
	time.Sleep(50 * time.Millisecond)
	g.Stop()
	if r := within(t, done, 2*time.Second); r.err != io.EOF { //nolint:errorlint // exactly io.EOF
		t.Fatalf("a stopped Read gave %q, %v; want io.EOF", r.s, r.err)
	}
}

// highDescriptor is a copy of fd at FD_SETSIZE plus off, closed in
// t.Cleanup. Go raises the open-files limit to the hard limit at start-up,
// so the copy fits; a copy that cannot be made fails the test.
func highDescriptor(t *testing.T, fd, off int) int {
	t.Helper()
	hi := unix.FD_SETSIZE + off
	if err := unix.Dup2(fd, hi); err != nil {
		t.Fatalf("copy descriptor %d to %d: %v", fd, hi, err)
	}
	t.Cleanup(func() { _ = unix.Close(hi) })
	return hi
}

// nonBlockingPipe is a pipe's read end as a descriptor, non-blocking when
// nb, and its write end; both are closed in t.Cleanup.
func nonBlockingPipe(t *testing.T, nb bool) (int, *os.File) {
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
	return fd, w
}

// A57 and A63 (rows R138, R154): NewGated refuses, with an error and never
// a panic, a wake pipe that cannot be made, a descriptor at or above
// FD_SETSIZE (the one given, or the wake pipe's read end), and a blocking
// descriptor.
func TestGatedRefusals(t *testing.T) {
	t.Run("the wake pipe cannot be made", func(t *testing.T) {
		fd, _ := nonBlockingPipe(t, true)
		g, err := NewGated(fd, GatedOptions{Pipe: func([]int) error { return unix.EMFILE }})
		if g != nil || !errors.Is(err, unix.EMFILE) {
			t.Fatalf("NewGated = %v, %v; want the wake pipe's error", g, err)
		}
	})
	t.Run("a descriptor at FD_SETSIZE", func(t *testing.T) {
		fd, _ := nonBlockingPipe(t, true)
		hi := highDescriptor(t, fd, 0)
		g, err := NewGated(hi, GatedOptions{})
		if g != nil || err == nil || !strings.Contains(err.Error(), "FD_SETSIZE") {
			t.Fatalf("NewGated(%d) = %v, %v; want it refused at FD_SETSIZE", hi, g, err)
		}
	})
	t.Run("the wake pipe's read end at FD_SETSIZE", func(t *testing.T) {
		fd, _ := nonBlockingPipe(t, true)
		g, err := NewGated(fd, GatedOptions{Pipe: func(p []int) error {
			if err := unix.Pipe(p); err != nil {
				return err
			}
			hi := unix.FD_SETSIZE + 1
			highDescriptor(t, p[0], 1)
			_ = unix.Close(p[0])
			p[0] = hi
			return nil
		}})
		if g != nil || err == nil || !strings.Contains(err.Error(), "FD_SETSIZE") {
			t.Fatalf("NewGated = %v, %v; want it refused at FD_SETSIZE", g, err)
		}
	})
	t.Run("a blocking descriptor", func(t *testing.T) {
		fd, _ := nonBlockingPipe(t, false)
		g, err := NewGated(fd, GatedOptions{})
		if g != nil || !errors.Is(err, errBlocking) {
			t.Fatalf("NewGated of a blocking pipe = %v, %v; want it refused", g, err)
		}
	})
}

// A48 (row R142): opens, closes and reads from several goroutines; every
// byte written is read once. Run under -race in P4 step 5.
func TestCancelableConcurrent(t *testing.T) {
	g, w := pipeGated(t, GatedOptions{})
	const total = 2000
	var got strings.Builder
	var gotMu sync.Mutex
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			g.Open()
			time.Sleep(time.Millisecond)
			g.Close()
		}
	})
	wg.Go(func() {
		b := make([]byte, 7)
		for {
			n, err := g.Read(b)
			if err != nil {
				return
			}
			gotMu.Lock()
			got.Write(b[:n])
			full := got.Len() >= total
			gotMu.Unlock()
			if full {
				return
			}
		}
	})
	var want strings.Builder
	for i := range total {
		want.WriteByte(byte('a' + i%26))
	}
	if _, err := w.WriteString(want.String()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		gotMu.Lock()
		n := got.Len()
		gotMu.Unlock()
		if n >= total || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(stop)
	g.Open()
	_ = w.Close()
	wg.Wait()
	if s := got.String(); s != want.String() {
		t.Fatalf("read %d bytes; want the %d written, each once and in order", len(s), total)
	}
}
