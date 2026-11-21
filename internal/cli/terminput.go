package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"

	xterm "golang.org/x/term"

	"github.com/maccavelli/go-tui-lib/v2/inline"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// lineBuf is the buffer a question reads the terminal with, one line at a
// time out of raw mode or one key at a time in it.
const lineBuf = 512

// terminalInput is the one reader of the terminal's input in a process:
// every prompt reads through it, so what one answer leaves is the next
// one's, and a line end is read one way everywhere (0063-MADR D1 to D4).
//
// A line ends at \r, at \n, or at \r\n, which is one end. After a \r that
// ended a line read out of raw mode, the next \n any prompt is given is that
// end's and is dropped, whether it is already read or arrives in a later
// read (lfOwed); after a raw read's \r, only a \n already read is dropped;
// \n then \r is two ends; any byte given clears lfOwed (D3).
//
// It never reads ahead: in is read only for a prompt that needs a byte and
// finds pending empty, with that prompt's buffer, so each read starts in the
// mode of whoever asked (D1; 0027-MADR D5).
type terminalInput struct {
	in keysIn // the inline.Reader over stdin, or a test's
	// flush empties the terminal's own input queue; nil for a pipe, a file
	// or a test's input with no terminal (D12, D18).
	flush func() error
	// turn is held by the prompt reading, so bytes are given in the order
	// they were read; a read of in is made under it.
	turn chan struct{}

	mu      sync.Mutex // guards the fields below; never held across a read of in
	pending []byte     // read from in and not yet given to any prompt
	lfOwed  bool       // a line read out of raw mode ended at \r: a \n next is that end's
	closed  bool       // in gave an error that is not a context's: the input is closed for good
	err     error      // that error, given to every read after
	// waiting is the size of a read of in that ended on its context and may
	// still wait in inline.Reader; 0 when none (D13).
	waiting int
	// drop is the size of a read a program gave up: its completion is taken
	// with that size and dropped, whoever reads next (D13, D17, D18).
	drop int
	// flushDue says the next read of in is preceded by flush (D12).
	flushDue bool
	// signedIn is the mark of D17 (Q20): a sign-in round or the first run
	// ended, so the next session prompt flushes.
	signedIn bool
	// gate is a POSIX terminal's cancellable reader under in, opened for each
	// read of in and closed when it returns; nil where there is none: a
	// Windows console, a pipe, a test's input (0063-MADR D22, rows R127,
	// R151).
	gate inputGate
	// posix says standard input is a POSIX terminal, gated or fallen back to
	// reading as built: a program's give-up flushes its queue (0063-MADR
	// D23, rows R80, R157).
	posix bool
	// tell writes Q24's warning when a fallback queued it; nil when nothing
	// can queue one (D23, row R156).
	tell func()
}

// inputGate lets the reader under in read the terminal only while a prompt
// waits: term.Gated (0063-MADR D22). Close reports whether a read completed
// since Open.
type inputGate interface {
	Open()
	Close() (took bool)
}

// newTerminalInput is the terminal input over in; flush, when not nil,
// empties the terminal's own queue.
func newTerminalInput(in keysIn, flush func() error) *terminalInput {
	return &terminalInput{in: in, flush: flush, turn: make(chan struct{}, 1)}
}

// terminalIn is the process's one terminal input, made on first use over
// standard input (0063-MADR D1): on a POSIX terminal through term.Gated, so
// no read of it is left waiting after a prompt returns (D22); on Windows
// through term.ConsoleInput (D19); a pipe or a file read as it is. On a
// terminal it can flush the terminal's queue (D18).
func (e *runEnv) terminalIn() *terminalInput {
	e.termInMu.Lock()
	defer e.termInMu.Unlock()
	if e.termIn == nil {
		var flush func() error
		if e.onTerminal() {
			flush = e.flushTerminal
		}
		src, gate := e.terminalSource()
		e.termIn = newTerminalInput(inline.NewReader(src), flush)
		e.termIn.gate = gate
		e.termIn.posix = e.posixTerm
		e.termIn.tell = e.tellWaitWarn
	}
	return e.termIn
}

// openGated opens the POSIX terminal's cancellable reader; tests replace it
// (0063-MADR D22).
var openGated = term.OpenGated

// isTerminalFile reports whether f is a terminal; tests replace it.
var isTerminalFile = func(f *os.File) bool { return xterm.IsTerminal(int(f.Fd())) } //nolint:gosec // G115: a descriptor fits an int

// terminalSource is what the terminal input reads, and its gate: on a POSIX
// terminal the gated reader; where it cannot be had, standard input as
// built, with Q24's warning once (rows R137 to R139, R143). It notes in
// e.posixTerm whether standard input is a POSIX terminal: any terminal
// but one where the system has no cancellable read (D23, row R157). A
// fallback on inline.Reader's goroutine queues the warning (row R156).
func (e *runEnv) terminalSource() (io.Reader, inputGate) {
	console := term.ConsoleInput(e.stdin, e.intr.consoleMark())
	if !e.onTerminal() || !isTerminalFile(e.stdin) {
		return console, nil
	}
	g, err := openGated(e.stdin, term.GatedOptions{Fallback: e.stdin, OnFallback: e.queueWaitWarn})
	e.posixTerm = !errors.Is(err, term.ErrNoGate)
	switch {
	case errors.Is(err, term.ErrNoGate):
		return console, nil
	case err != nil:
		e.warnWait(err)
		return e.stdin, nil
	}
	return g, g
}

// waitWarning is Q24's warning, in the owner's approved words.
const waitWarning = "gobble cannot wait on this terminal without reading it (%v); keys typed while a sign-in finishes may be lost."

// warnWait is Q24's warning, written now, once in the process: gobble
// reads the terminal as built from here on (0063-MADR D22, row R138). It is
// for the main flow, where the input is made.
func (e *runEnv) warnWait(err error) {
	e.waitWarn.Do(func() {
		e.out.Warnf(waitWarning, err)
	})
}

// queueWaitWarn is Q24's warning when the reader falls back on
// inline.Reader's goroutine: never written there, but queued, once in the
// process with warnWait, for tellWaitWarn (0063-MADR D23, row R156).
func (e *runEnv) queueWaitWarn(err error) {
	e.waitWarn.Do(func() {
		e.waitMu.Lock()
		e.waitQueued = err
		e.waitMu.Unlock()
	})
}

// waitPending reports whether a warning is queued for tellWaitWarn.
func (e *runEnv) waitPending() bool {
	e.waitMu.Lock()
	defer e.waitMu.Unlock()
	return e.waitQueued != nil
}

// tellWaitWarnOutOfRaw is tellWaitWarn at the line session's prompt, where
// the terminal is raw: when a warning is queued, it is written through
// outOfRaw, so its line end keeps its carriage return (0063-MADR D23, row
// R156; build pass 5). With none, raw mode is left alone.
func (e *runEnv) tellWaitWarnOutOfRaw(outOfRaw func(func() error) error) error {
	if !e.waitPending() {
		return nil
	}
	return outOfRaw(func() error { e.tellWaitWarn(); return nil })
}

// tellWaitWarn writes a queued warning on the main flow, once: before the
// next prompt's or question's text, with the terminal in its cooked state,
// or at the run's end (row R156).
func (e *runEnv) tellWaitWarn() {
	e.waitMu.Lock()
	err := e.waitQueued
	e.waitQueued = nil
	e.waitMu.Unlock()
	if err != nil {
		e.out.Warnf(waitWarning, err)
	}
}

// termFlush empties the terminal's input queue on f; tests replace it
// (0063-MADR D18).
var termFlush = term.FlushInput

// flushTerminal empties standard input's terminal queue. A failure is
// skipped, and the first in the process is warned of on stderr, once
// (0063-MADR D17, rows R78, R101).
func (e *runEnv) flushTerminal() error {
	err := termFlush(e.stdin)
	if err != nil {
		e.flushWarn.Do(func() {
			e.out.Warnf("could not clear what was typed ahead on this terminal (%v); it may answer the next prompt.", err)
		})
	}
	return err
}

// take waits for the turn to read, until ctx ends.
func (t *terminalInput) take(ctx context.Context) error {
	select {
	case t.turn <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *terminalInput) give() { <-t.turn }

// owedLocked applies D3's rule to the next byte given: an owed \n is
// dropped; any byte clears what was owed. t.mu is held.
func (t *terminalInput) owedLocked() {
	if !t.lfOwed || len(t.pending) == 0 {
		return
	}
	t.lfOwed = false
	if t.pending[0] == '\n' {
		t.pending = t.pending[1:]
	}
}

// isContextErr reports whether err is a context's: an ended wait, not the
// input's end.
func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// fill makes one read of in for a prompt that found nothing to take, with
// a buffer of size, and keeps what it gives in pending. The turn is held.
// Before it: the terminal's queue flushed when due (D12), and the
// completion of a read a program gave up taken and dropped (D13). With a
// gate, the gate is open only while this read waits, and the read asks for
// lineBuf, whatever size, so what the caller cannot take stays in pending
// (0063-MADR D22, rows R127, R147).
func (t *terminalInput) fill(ctx context.Context, size int) error {
	if err := ctx.Err(); err != nil {
		return err // no read of in is started on an ended context
	}
	t.mu.Lock()
	due, drop := t.flushDue, t.drop
	t.flushDue = false
	t.mu.Unlock()
	if due && t.flush != nil {
		_ = t.flush() //nolint:errcheck // a failed flush is skipped; flushTerminal warns once (row R78)
	}
	ask := size
	if t.gate != nil {
		ask = lineBuf
		t.gate.Open()
	}
	if drop > 0 {
		n, err := t.in.Read(ctx, make([]byte, drop))
		if isContextErr(err) {
			t.closeGate()
			return err // the drop stays owed, and the read waits on
		}
		t.mu.Lock()
		t.drop, t.waiting = 0, 0
		t.mu.Unlock()
		if err != nil && n == 0 {
			t.closeGate()
			return t.closeWith(err) // the end of input is not dropped (row R89)
		}
	}
	buf := make([]byte, ask)
	n, err := t.in.Read(ctx, buf)
	took := t.closeGate()
	t.mu.Lock()
	defer t.mu.Unlock()
	if isContextErr(err) {
		// With no gate, the read may still wait in inline.Reader, as built
		// (D13). With one, it waits only when a read(2) completed before the
		// gate closed; otherwise nothing was read (rows R128, R129, R151).
		if took {
			t.waiting = ask
		}
		return err
	}
	t.waiting = 0
	t.pending = append(t.pending, buf[:n]...)
	if err != nil {
		t.closed, t.err = true, err
		if n == 0 {
			return err
		}
	}
	return nil
}

// closeGate closes the gate, when there is one, and reports whether a
// read completed while it was open; with no gate it reports one, so D13's
// drop holds as built (row R151).
func (t *terminalInput) closeGate() bool {
	if t.gate == nil {
		return true
	}
	return t.gate.Close()
}

// flushGiveUp empties a POSIX terminal's queue at the program's give-up of
// a prompt, so what was typed into it is never a turn, an answer, or the
// shell's input after gobble exits (0063-MADR D22, Q23, rows R80, R132):
// gated or fallen back to reading as built (D23, row R157). A Windows
// console keeps D13's drop; a pipe has no queue.
func (t *terminalInput) flushGiveUp() {
	if t.posix && t.flush != nil {
		_ = t.flush() //nolint:errcheck // a failed flush is skipped; flushTerminal warns once (row R78)
	}
}

// warnQueued writes Q24's warning when a fallback on inline.Reader's
// goroutine queued it: called on the main flow before a prompt's or a
// question's text, with the terminal cooked (0063-MADR D23, row R156).
func (t *terminalInput) warnQueued() {
	if t != nil && t.tell != nil {
		t.tell()
	}
}

// closeWith records err as the input's end and returns it.
func (t *terminalInput) closeWith(err error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed, t.err = true, err
	return err
}

// Read gives what pending holds, D3's rule applied, else one read of in
// with p's size: the line editor, the permission prompt and the trust
// question read here (0063-MADR D2). The end of input, or the read's
// error, is returned as inline.Reader returns it, once the bytes before it
// are given.
func (t *terminalInput) Read(ctx context.Context, p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err := t.take(ctx); err != nil {
		return 0, err
	}
	defer t.give()
	for {
		t.mu.Lock()
		t.owedLocked()
		if len(t.pending) > 0 {
			n := copy(p, t.pending)
			t.pending = t.pending[n:]
			t.mu.Unlock()
			return n, nil
		}
		closed, cerr := t.closed, t.err
		t.mu.Unlock()
		if closed {
			return 0, cerr
		}
		if err := t.fill(ctx, len(p)); err != nil {
			return 0, err
		}
	}
}

// line is the text up to the next line end, read out of raw mode a line at
// a time: \r, \n or \r\n, and after \r the next \n is owed (D3). A line with
// no end before the end of input is no answer and stays in pending.
func (t *terminalInput) line(ctx context.Context) (string, error) {
	if err := t.take(ctx); err != nil {
		return "", err
	}
	defer t.give()
	for {
		t.mu.Lock()
		t.owedLocked()
		for i, c := range t.pending {
			if c != '\r' && c != '\n' {
				continue
			}
			s := string(t.pending[:i])
			t.pending = t.pending[i+1:]
			t.lfOwed = c == '\r'
			t.mu.Unlock()
			return s, nil
		}
		closed, cerr := t.closed, t.err
		t.mu.Unlock()
		if closed {
			return "", cerr
		}
		if err := t.fill(ctx, lineBuf); err != nil {
			return "", err
		}
	}
}

// key is the next byte, for a secret read in raw mode (D2).
func (t *terminalInput) key(ctx context.Context) (byte, error) {
	if err := t.take(ctx); err != nil {
		return 0, err
	}
	defer t.give()
	for {
		t.mu.Lock()
		t.owedLocked()
		if len(t.pending) > 0 {
			c := t.pending[0]
			t.pending = t.pending[1:]
			t.mu.Unlock()
			return c, nil
		}
		closed, cerr := t.closed, t.err
		t.mu.Unlock()
		if closed {
			return 0, cerr
		}
		if err := t.fill(ctx, lineBuf); err != nil {
			return 0, err
		}
	}
}

// rawEnd is a raw read's line end, c: after \r, a \n already read directly
// after it is that end's and is dropped (a pasted key ending \r\n); nothing
// is owed, since in raw mode Enter is \r alone (D3).
func (t *terminalInput) rawEnd(c byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lfOwed = false
	if c == '\r' && len(t.pending) > 0 && t.pending[0] == '\n' {
		t.pending = t.pending[1:]
	}
}

// discard drops what was read and not given, keeping lfOwed: before each
// session prompt, nothing typed or pasted during a sign-in becomes a turn
// (D4; Q3).
func (t *terminalInput) discard() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pending = nil
}

// ended reports whether the input is closed for good: the end of input, or
// a read failure; never an ended context (D8).
func (t *terminalInput) ended() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

// abandon marks a read a program gave up (D13, as D17 narrows it): when a
// read of in ended on its context and may still wait, its completion,
// whatever it is, is taken with the read's own size and dropped, whoever
// reads next (D18). A read the person gave up with Ctrl+C is not abandoned.
func (t *terminalInput) abandon() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.waiting > 0 {
		t.drop = t.waiting
	}
}

// flushNext marks that the next read of in is preceded by a flush of the
// terminal's queue (D12): for a prompt whose text must be written before the
// flush, as P2b's repeated offer; with no terminal it marks nothing.
func (t *terminalInput) flushNext() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flushDue = t.flush != nil
}

// signedInRound sets D17's mark: a sign-in round or the first run ended,
// whatever its outcome, so the next session prompt flushes (rows R103,
// R104).
func (t *terminalInput) signedInRound() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.signedIn = true
}

// beforePrompt is what the line session does before each ReadPrompt:
// pending dropped, lfOwed kept (D4); and, at the first prompt after a
// sign-in round or the first run, the mark cleared and the terminal's
// queue flushed at once, whether or not the editor then reads the terminal:
// keys the editor holds from before the sign-in can be that prompt's
// submission (D12, D17, D21; rows R82, R103 to R108, R123). A failed flush
// is skipped; flushTerminal warns once (row R78). A queued Q24 warning is
// not written here, where the terminal is raw: the session writes it out of
// raw mode first (D23, row R156; build pass 5).
func (t *terminalInput) beforePrompt() {
	t.mu.Lock()
	marked := t.signedIn
	t.signedIn = false
	t.mu.Unlock()
	t.discard()
	if marked && t.flush != nil {
		_ = t.flush() //nolint:errcheck // a failed flush is skipped; flushTerminal warns once (row R78)
	}
}
