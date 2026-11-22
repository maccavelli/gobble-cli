//go:build unix

package term

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

// privateFlags open the terminal's own path as a descriptor of gobble's:
// its own open file description, so non-blocking reads leave standard
// input as the shell set it (0063-MADR D22, Q25, row R126).
const privateFlags = unix.O_RDONLY | unix.O_NONBLOCK | unix.O_CLOEXEC | unix.O_NOCTTY

// Gated reads a POSIX terminal only while a prompt waits (0063-MADR D22): a
// read(2) of a private non-blocking descriptor is made only with the gate
// open and once select(2) reports the terminal ready, so no read of the
// terminal is ever left waiting after a prompt returns. While the gate is
// closed, Read waits on the wake pipe alone and reads nothing (rows R125 to
// R127, R145, R146). It is the layer under inline.Reader, beside
// ConsoleInput.
type Gated struct {
	fd         int // the private descriptor
	wakeR      int
	wakeW      int
	o          GatedOptions
	mu         sync.Mutex
	changed    *sync.Cond // inSelect or active changed
	open       bool
	took       bool // a read(2) completed since the gate opened
	inSelect   bool // the waiter is in a wait that includes the terminal
	active     int  // Reads in progress
	stopped    bool
	fellBack   bool
	fallbackMu sync.Mutex
}

// OpenGated opens the private descriptor on stdin's terminal: its own path
// (o.Path, or the path read from stdin), opened with privateFlags, when it
// is the same terminal as stdin's (the same device); when that open fails,
// or opens another terminal, or gives a blocking descriptor, and stdin is
// the controlling terminal (the same foreground process group on stdin and
// on /dev/tty), /dev/tty with the same flags (0063-MADR D22, Q25; D23, rows
// R154, R155). Its error is Q24's case: the caller reads stdin as built.
func OpenGated(stdin *os.File, o GatedOptions) (*Gated, error) {
	if stdin == nil {
		return nil, errors.New("no standard input")
	}
	path := o.Path
	var err error
	if path == "" {
		if path, err = terminalPath(int(stdin.Fd())); err != nil { //nolint:gosec // G115: a descriptor fits an int
			path = ""
		}
	}
	if path != "" {
		var g *Gated
		g, err = gatedOn(func() (int, error) { return sameTerminal(stdin, path) }, o)
		if !isNotOpened(err) {
			return g, err
		}
	}
	g, cerr := gatedOn(func() (int, error) { return controllingTTY(stdin) }, o)
	if isNotOpened(cerr) {
		return nil, fmt.Errorf("open the terminal: %w", errors.Join(err, cerr))
	}
	return g, cerr
}

// notOpened is a descriptor OpenGated could not have, or refused: it tries
// the next, /dev/tty, and then gives up (0063-MADR D23, rows R154, R155).
type notOpened struct{ err error }

func (e notOpened) Error() string { return e.err.Error() }
func (e notOpened) Unwrap() error { return e.err }

func isNotOpened(err error) bool {
	var no notOpened
	return errors.As(err, &no)
}

// gatedOn is a Gated reader of the descriptor open gives. A failed open,
// or a blocking descriptor, is notOpened; any other refusal of NewGated's
// is Q24's case at once (row R138).
func gatedOn(open func() (int, error), o GatedOptions) (*Gated, error) {
	fd, err := open()
	if err != nil {
		return nil, notOpened{err}
	}
	g, err := NewGated(fd, o)
	if err != nil {
		closeFDs(fd)
		if errors.Is(err, errBlocking) {
			return nil, notOpened{err}
		}
		return nil, err
	}
	return g, nil
}

// sameTerminal opens path with privateFlags when it is stdin's own
// terminal: the same device, by fstat(2)'s st_rdev on both (0063-MADR D23,
// row R155).
func sameTerminal(stdin *os.File, path string) (int, error) {
	fd, err := unix.Open(path, privateFlags, 0)
	if err != nil {
		return -1, err
	}
	var in, opened unix.Stat_t
	if err := unix.Fstat(int(stdin.Fd()), &in); err != nil { //nolint:gosec // G115: a descriptor fits an int
		closeFDs(fd)
		return -1, err
	}
	if err := unix.Fstat(fd, &opened); err != nil {
		closeFDs(fd)
		return -1, err
	}
	if in.Rdev != opened.Rdev {
		closeFDs(fd)
		return -1, fmt.Errorf("%s is not standard input's terminal", path)
	}
	return fd, nil
}

// controllingTTY opens /dev/tty with privateFlags when it is stdin's
// terminal: the same foreground process group on both.
func controllingTTY(stdin *os.File) (int, error) {
	fd, err := unix.Open("/dev/tty", privateFlags, 0)
	if err != nil {
		return -1, err
	}
	a, err1 := unix.IoctlGetInt(int(stdin.Fd()), unix.TIOCGPGRP) //nolint:gosec // G115: a descriptor fits an int
	b, err2 := unix.IoctlGetInt(fd, unix.TIOCGPGRP)
	if err1 != nil || err2 != nil || a != b {
		closeFDs(fd)
		return -1, errors.New("/dev/tty is not standard input's terminal")
	}
	return fd, nil
}

// errBlocking is NewGated's refusal of a descriptor without O_NONBLOCK: a
// read(2) of it could wait with the gate closed (0063-MADR D23, row R154).
var errBlocking = errors.New("the descriptor is blocking")

// NewGated is a Gated reader of fd, a non-blocking descriptor the caller
// opened, which the reader then owns: a terminal's private descriptor, or a
// test's pipe. A descriptor without O_NONBLOCK is refused (row R154), and
// one at or above FD_SETSIZE, as select(2) cannot wait on it (row R138);
// a refused descriptor stays the caller's.
func NewGated(fd int, o GatedOptions) (*Gated, error) {
	fl, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0) //nolint:gosec // G115: a descriptor fits a uintptr
	if err != nil {
		return nil, fmt.Errorf("descriptor %d: %w", fd, err)
	}
	if fl&unix.O_NONBLOCK == 0 {
		return nil, fmt.Errorf("descriptor %d: %w", fd, errBlocking)
	}
	pipe := o.Pipe
	if pipe == nil {
		pipe = unix.Pipe
	}
	var p [2]int
	syscall.ForkLock.RLock()
	err = pipe(p[:])
	if err == nil {
		unix.CloseOnExec(p[0])
		unix.CloseOnExec(p[1])
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("the wake pipe: %w", err)
	}
	for _, d := range p {
		if err := unix.SetNonblock(d, true); err != nil {
			closeFDs(p[0])
			closeFDs(p[1])
			return nil, fmt.Errorf("the wake pipe: %w", err)
		}
	}
	if fd >= unix.FD_SETSIZE || p[0] >= unix.FD_SETSIZE {
		closeFDs(p[0])
		closeFDs(p[1])
		return nil, fmt.Errorf("descriptor %d is at or above FD_SETSIZE", max(fd, p[0]))
	}
	g := &Gated{fd: fd, wakeR: p[0], wakeW: p[1], o: o}
	g.changed = sync.NewCond(&g.mu)
	if g.o.Wait == nil {
		g.o.Wait = selectRead
	}
	if g.o.Read == nil {
		g.o.Read = unix.Read
	}
	return g, nil
}

// selectRead is select(2) on fds for reading, with no timeout.
func selectRead(fds []int) ([]bool, error) {
	var s unix.FdSet
	s.Zero()
	top := 0
	for _, fd := range fds {
		s.Set(fd)
		top = max(top, fd)
	}
	if _, err := unix.Select(top+1, &s, nil, nil, nil); err != nil {
		return nil, err
	}
	ready := make([]bool, len(fds))
	for i, fd := range fds {
		ready[i] = s.IsSet(fd)
	}
	return ready, nil
}

// wake makes a wait on the wake pipe return; a full pipe is already awake.
func (g *Gated) wake() {
	_, _ = unix.Write(g.wakeW, []byte{1}) //nolint:errcheck // EAGAIN: a wake is already pending (row R146)
}

// drain empties the wake pipe before a wait.
func (g *Gated) drain() {
	b := make([]byte, 64)
	for {
		if n, err := unix.Read(g.wakeR, b); n <= 0 || err != nil {
			return
		}
	}
}

// Open lets the next read of the terminal happen, for a prompt that waits.
func (g *Gated) Open() {
	g.mu.Lock()
	g.open, g.took = true, false
	g.mu.Unlock()
	g.wake()
}

// Close stops reading the terminal: it wakes the wait, returns once the
// waiter has left a wait that includes the terminal, and reports whether a
// read completed since the gate opened. Once the reader has fallen back to
// reading standard input as built, every Close reports a read, so D13's
// drop applies (row R151).
func (g *Gated) Close() bool {
	g.mu.Lock()
	g.open = false
	g.mu.Unlock()
	g.wake()
	g.mu.Lock()
	defer g.mu.Unlock()
	for g.inSelect {
		g.changed.Wait()
	}
	took := g.took || g.fellBack
	g.took = false
	return took
}

// Read gives the terminal's next bytes once the gate is open, the end of
// input as io.EOF, or a read error; never because of a close (row R145).
func (g *Gated) Read(p []byte) (int, error) {
	g.mu.Lock()
	g.active++
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		g.active--
		g.changed.Broadcast()
		g.mu.Unlock()
	}()
	for {
		// Drained before the state is read: a wake written after this, by
		// Open, Close or Stop, stays in the pipe for the wait below.
		g.drain()
		g.mu.Lock()
		stopped, open, fellBack := g.stopped, g.open, g.fellBack
		if open && !stopped && !fellBack {
			g.inSelect = true
		}
		g.mu.Unlock()
		switch {
		case stopped:
			return 0, io.EOF
		case fellBack:
			return g.fallbackRead(p)
		case !open:
			// The terminal is not in this wait, so bytes ready there
			// cannot make it return (row R145).
			if _, err := g.wait([]int{g.wakeR}); err != nil {
				return g.fallBack(p, err)
			}
			continue
		}
		ready, err := g.wait([]int{g.fd, g.wakeR})
		g.mu.Lock()
		g.inSelect = false
		g.changed.Broadcast()
		if err != nil {
			g.mu.Unlock()
			return g.fallBack(p, err)
		}
		if !ready[0] || !g.open || g.stopped {
			g.mu.Unlock()
			continue
		}
		n, rerr := g.o.Read(g.fd, p)
		if errors.Is(rerr, unix.EINTR) {
			g.mu.Unlock()
			continue // a signal before any byte: wait again (row R153)
		}
		if errors.Is(rerr, unix.EAGAIN) {
			g.mu.Unlock()
			continue // nothing after all: wait again (row R126)
		}
		g.took = true
		g.mu.Unlock()
		switch {
		case n > 0:
			return n, nil
		case rerr == nil:
			return 0, io.EOF // Ctrl+D, or a hang-up (rows R134, R148)
		}
		return 0, rerr
	}
}

// wait is the Wait seam, retried on EINTR (row R125: a guard; none was
// seen, M6, M9), and then Returned.
func (g *Gated) wait(fds []int) ([]bool, error) {
	for {
		ready, err := g.o.Wait(fds)
		if g.o.Returned != nil {
			g.o.Returned()
		}
		if !errors.Is(err, unix.EINTR) {
			return ready, err
		}
	}
}

// fallBack makes the reader read standard input as built for the rest of
// the process, tells OnFallback once, and reads (0063-MADR D22, Q24).
func (g *Gated) fallBack(p []byte, err error) (int, error) {
	g.mu.Lock()
	first := !g.fellBack
	g.fellBack = true
	g.inSelect = false
	g.changed.Broadcast()
	g.mu.Unlock()
	if first && g.o.OnFallback != nil {
		g.o.OnFallback(err)
	}
	return g.fallbackRead(p)
}

func (g *Gated) fallbackRead(p []byte) (int, error) {
	if g.o.Fallback == nil {
		return 0, io.EOF
	}
	g.fallbackMu.Lock()
	defer g.fallbackMu.Unlock()
	return g.o.Fallback.Read(p)
}

// Stop ends the reader: a waiting Read returns io.EOF, and once none is
// left, the private descriptor and the wake pipe are closed. Tests stop
// theirs in t.Cleanup; the process's lives as long as it does (row R146).
func (g *Gated) Stop() {
	g.mu.Lock()
	g.stopped = true
	g.mu.Unlock()
	g.wake()
	g.mu.Lock()
	for g.active > 0 && !g.fellBack {
		g.changed.Wait()
	}
	g.mu.Unlock()
	closeFDs(g.fd)
	closeFDs(g.wakeR)
	closeFDs(g.wakeW)
}

// closeFDs closes descriptors the reader opened; a close that fails leaves
// nothing more to do.
func closeFDs(fds ...int) {
	for _, fd := range fds {
		_ = unix.Close(fd) //nolint:errcheck // nothing more can be done with a descriptor that will not close
	}
}

// Descriptor is the private descriptor's number, for a test.
func (g *Gated) Descriptor() int { return g.fd }
