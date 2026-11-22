package term

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

// MarkWait is how long a console read that gave nothing, with no interrupt
// marked, waits for one before it is the end of input (0063-MADR D20, Q22).
const MarkWait = 100 * time.Millisecond

// InterruptMark is the process's note that gobble received Ctrl+C while a
// read of the console was waiting (0063-MADR D19, D20, Q21). The signal
// watcher calls Interrupted; ConsoleInput's reader takes the mark. The zero
// value is ready to use.
type InterruptMark struct {
	mu      sync.Mutex
	waiting int  // console reads waiting, given back to no caller yet
	set     bool // an interrupt came while one waited, and excused no read yet
	wake    chan struct{}
}

// Interrupted notes an interrupt gobble received: a mark, only when a read
// of the console is waiting (Q21); an interrupt with none waiting excuses
// nothing (row R114).
func (m *InterruptMark) Interrupted() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.waiting == 0 {
		return
	}
	m.set = true
	select {
	case m.wakeLocked() <- struct{}{}:
	default:
	}
}

// wakeLocked is the channel a waiting read is woken on; m.mu is held.
func (m *InterruptMark) wakeLocked() chan struct{} {
	if m.wake == nil {
		m.wake = make(chan struct{}, 1)
	}
	return m.wake
}

func (m *InterruptMark) begin() {
	m.mu.Lock()
	m.waiting++
	m.mu.Unlock()
}

func (m *InterruptMark) end() {
	m.mu.Lock()
	m.waiting--
	m.mu.Unlock()
}

// clear drops the mark: a read that returns bytes clears it (Q21).
func (m *InterruptMark) clear() {
	m.mu.Lock()
	m.set = false
	m.mu.Unlock()
}

// take reports whether a mark excuses a read that gave nothing, clearing
// it; with none, it waits up to wait for one (Q22).
func (m *InterruptMark) take(wait time.Duration) bool {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for {
		m.mu.Lock()
		if m.set {
			m.set = false
			m.mu.Unlock()
			return true
		}
		wake := m.wakeLocked()
		m.mu.Unlock()
		select {
		case <-wake:
		case <-timer.C:
			m.mu.Lock()
			ok := m.set
			m.set = false
			m.mu.Unlock()
			return ok
		}
	}
}

// consoleReader is standard input with D19's rule when it is a console.
type consoleReader struct {
	r       io.Reader
	console bool
	mark    *InterruptMark
	wait    time.Duration
}

// NewConsoleReader is r with 0063-MADR D19's rule when console is true: a
// read that gives nothing, the end of input, is read again when an
// interrupt was marked while it waited, or within wait after it returned
// (D20); otherwise the end of input is passed on. A read that returns bytes
// clears the mark. When console is false, r is read unchanged (row R117).
// It is ConsoleInput's portable part, which tests drive over a stub.
func NewConsoleReader(r io.Reader, console bool, mark *InterruptMark, wait time.Duration) io.Reader {
	if mark == nil {
		mark = &InterruptMark{}
	}
	return &consoleReader{r: r, console: console, mark: mark, wait: wait}
}

func (c *consoleReader) Read(p []byte) (int, error) {
	if !c.console {
		return c.r.Read(p)
	}
	c.mark.begin()
	defer c.mark.end()
	for {
		n, err := c.r.Read(p)
		if n > 0 {
			c.mark.clear()
			return n, err
		}
		if !errors.Is(err, io.EOF) || !c.mark.take(c.wait) {
			return n, err
		}
	}
}

// ConsoleInput is what gobble's one terminal input reads (0063-MADR D19):
// on Windows, when f is a console, f with D19's rule over mark; elsewhere,
// and for a pipe or a file, f itself.
func ConsoleInput(f *os.File, mark *InterruptMark) io.Reader {
	if f == nil || !isConsoleInput(f) {
		return f
	}
	return NewConsoleReader(f, true, mark, MarkWait)
}
