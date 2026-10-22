package editor

import (
	"bytes"
	"context"
	"io"
	"time"
)

const (
	keyCtrlC = 0x03
	keyCtrlD = 0x04
	keyEnter = '\r'

	// keyInterrupt is the private key Ctrl+C becomes outside a paste. The
	// buffer gives 0x1C no meaning, so it reaches the completion callback
	// with the whole line (lineBuffer.other).
	keyInterrupt = 0x1c

	// exitWindow is how long a second Ctrl+C on an empty line exits.
	exitWindow = 2 * time.Second
)

var (
	pasteStart = []byte("\x1b[200~")
	pasteEnd   = []byte("\x1b[201~")
)

// contextReader is satisfied by *Reader.
type contextReader interface {
	ReadContext(ctx context.Context, p []byte) (int, error)
}

// keyReader sits between stdin and the editor. Outside a bracketed paste
// it rewrites Ctrl+C to keyInterrupt and returns at once, so the callback
// decides what Ctrl+C means before any later byte is read. The decision is
// carried out by the one key the callback may queue: Enter, which ends the
// empty line so the notice can be printed between prompts, or Ctrl+D, which
// ends the input on the empty line.
type keyReader struct {
	src     io.Reader
	ctx     context.Context
	pending []byte
	tail    []byte // the last bytes seen, to find paste markers across reads
	inPaste bool

	inject     byte
	notice     bool
	exit       bool
	armedUntil time.Time

	// resize and timeout, when set, stop a read with no bytes: the
	// terminal was resized, or the wait for its cursor's place is over
	// (0030-MADR D2).
	resize  <-chan struct{}
	timeout <-chan time.Time
}

// untilReader is satisfied by *Reader.
type untilReader interface {
	ReadUntil(ctx context.Context, p []byte, resize <-chan struct{}, timeout <-chan time.Time) (int, error)
}

func (k *keyReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if k.inject != 0 {
		p[0], k.inject = k.inject, 0
		return 1, nil
	}
	if len(k.pending) == 0 {
		buf := make([]byte, chunkSize)
		n, err := k.readSrc(buf)
		if n == 0 {
			return 0, err
		}
		k.pending = buf[:n]
	}
	n := 0
	for n < len(p) && len(k.pending) > 0 {
		c := k.pending[0]
		k.pending = k.pending[1:]
		if c == keyCtrlC && !k.inPaste {
			p[n] = keyInterrupt
			return n + 1, nil
		}
		k.track(c)
		p[n] = c
		n++
	}
	return n, nil
}

func (k *keyReader) readSrc(p []byte) (int, error) {
	if ur, ok := k.src.(untilReader); ok && k.ctx != nil && (k.resize != nil || k.timeout != nil) {
		return ur.ReadUntil(k.ctx, p, k.resize, k.timeout)
	}
	if cr, ok := k.src.(contextReader); ok && k.ctx != nil {
		return cr.ReadContext(k.ctx, p)
	}
	return k.src.Read(p)
}

// track follows bracketed-paste markers, which may be split across reads.
func (k *keyReader) track(c byte) {
	k.tail = append(k.tail, c)
	if len(k.tail) > len(pasteStart) {
		k.tail = k.tail[len(k.tail)-len(pasteStart):]
	}
	switch {
	case bytes.HasSuffix(k.tail, pasteStart):
		k.inPaste = true
	case bytes.HasSuffix(k.tail, pasteEnd):
		k.inPaste = false
	}
}

// interrupt is called from the completion callback for keyInterrupt. A
// non-empty line is cleared. On an empty line the first press arms the exit
// window and asks for the notice; a second press inside it exits.
func (k *keyReader) interrupt(line string) (newLine string, newPos int, ok bool) {
	if line != "" {
		k.armedUntil = time.Time{}
		return "", 0, true
	}
	now := time.Now()
	if now.Before(k.armedUntil) {
		k.exit, k.inject = true, keyCtrlD
		return "", 0, false
	}
	k.armedUntil = now.Add(exitWindow)
	k.notice, k.inject = true, keyEnter
	return "", 0, false
}

// takeNotice reports, once, that the last empty line was Ctrl+C's.
func (k *keyReader) takeNotice() bool {
	n := k.notice
	k.notice = false
	return n
}

// takeExit reports, once, that io.EOF came from a second Ctrl+C.
func (k *keyReader) takeExit() bool {
	e := k.exit
	k.exit = false
	return e
}
