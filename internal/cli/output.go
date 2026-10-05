package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
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
}

// Result writes p to stdout. A broken pipe becomes errBrokenPipe.
func (o *Output) Result(p []byte) error {
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

func (o *Output) diag(prefix, format string, args ...any) {
	if _, err := fmt.Fprintf(o.err, prefix+format+"\n", args...); err != nil && o.errWrite == nil {
		o.errWrite = err
	}
}
