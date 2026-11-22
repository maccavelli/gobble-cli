package term

import (
	"errors"
	"io"
)

// ErrNoGate is OpenGated's answer on a system with no cancellable read: a
// Windows console keeps D19's rule, and any other system reads standard
// input as built (0063-MADR D22, rows R139, R143).
var ErrNoGate = errors.New("this system has no cancellable read of a terminal")

// GatedOptions are a Gated reader's choices and its seams.
type GatedOptions struct {
	// Path is the terminal's path; "" reads it from standard input.
	Path string
	// Fallback is read, as built, for the rest of the process once the
	// readiness wait cannot be used (0063-MADR D22, Q24); nil gives the end
	// of input instead.
	Fallback io.Reader
	// OnFallback is told the wait's error, once, when it falls back.
	OnFallback func(error)
	// Wait, Read, Returned and Pipe are seams for tests. Wait waits until
	// one of fds is ready to read and reports which; nil is select(2). Read
	// reads the private descriptor; nil is read(2). Returned is called after
	// each return from Wait. Pipe makes the wake pipe; nil is pipe(2)
	// (0063-MADR D23, row R138).
	Wait     func(fds []int) (ready []bool, err error)
	Read     func(fd int, p []byte) (int, error)
	Returned func()
	Pipe     func(p []int) error
}
