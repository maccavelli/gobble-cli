package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	xterm "golang.org/x/term"

	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/internal/cli/render"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/personality"
)

// errBrokenPipe ends the process quietly with status 0: the reader of
// stdout has gone, as with `gobble version | head -c1` (0008-MADR D3).
var errBrokenPipe = errors.New("broken pipe")

// Output is the only way commands write. Results go to out; diagnostics go
// to err, prefixed "Error:" or "Warning:" (0008-MADR D3).
type Output struct {
	out, err io.Writer
	caps     term.Caps
	// errWrite is the first failed write to err. Nothing can report it, so
	// it is kept only for tests.
	errWrite error
	// rec, in the line session on a terminal, keeps what is shown for a
	// reprint (0030-MADR D4); nil elsewhere.
	rec *transcript
}

// Result writes p to stdout. A broken pipe becomes errBrokenPipe. In the
// line session on a terminal, p is kept as written for a reprint
// (0030-MADR D4).
func (o *Output) Result(p []byte) error {
	if o.rec != nil {
		o.rec.fixed(string(p))
	}
	return o.emit(p)
}

// emit writes p to stdout without keeping it: for output the caller keeps
// in a form it can draw again, such as a reply's Markdown.
func (o *Output) emit(p []byte) error {
	if _, err := o.out.Write(p); err != nil {
		if isBrokenPipe(err) {
			return errBrokenPipe
		}
		return fmt.Errorf("write stdout: %w", err)
	}
	return nil
}

// Resultf formats a result line.
func (o *Output) Resultf(format string, args ...any) error {
	return o.Result(fmt.Appendf(nil, format, args...))
}

// Errorf prints "Error: …" on stderr.
func (o *Output) Errorf(format string, args ...any) { o.diag("Error: ", format, args...) }

// Warnf prints "Warning: …" on stderr.
func (o *Output) Warnf(format string, args ...any) { o.diag("Warning: ", format, args...) }

// Lead writes voice's line for key before an error or a warning, when the
// voice has one; boring has none (0022-MADR D13).
func (o *Output) Lead(voice *personality.Voice, key personality.Key) {
	if line, ok := voice.Lead(key); ok {
		o.diag("", "%s", line)
	}
}

// Notef writes a status line to stderr, unprefixed, such as a retry's.
func (o *Output) Notef(format string, args ...any) { o.diag("", format, args...) }

// diag writes a message to stderr, wrapped on a terminal (0028-MADR D6).
func (o *Output) diag(prefix, format string, args ...any) {
	src := prefix + fmt.Sprintf(format, args...)
	if o.rec != nil {
		o.rec.message(src)
	}
	msg := render.Message(src, o.errWidth())
	if _, err := io.WriteString(o.err, msg+"\n"); err != nil && o.errWrite == nil {
		o.errWrite = err
	}
}

// errWidth is the width gobble's messages wrap to: stderr's, read now, at
// most 120 columns, or 0, no wrapping, when stderr is not a terminal
// (0028-MADR D6).
func (o *Output) errWidth() int {
	if !o.caps.Err.TTY {
		return 0
	}
	w := o.caps.Err.Width
	if f, ok := o.err.(*os.File); ok {
		if c, _, err := xterm.GetSize(int(f.Fd())); err == nil && c > 0 { //nolint:gosec // G115: a terminal's descriptor fits an int
			w = c
		}
	}
	return min(w, editor.MaxWidth)
}
