package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// errCancelled ends a sign-in the user left: Ctrl+C, the end of input, or
// an empty answer where one is needed.
var errCancelled = errors.New("cancelled")

// keysIn is the reader that owns stdin: editor.Reader.
type keysIn interface {
	ReadContext(ctx context.Context, p []byte) (int, error)
}

// questions asks the user what a sign-in needs, on the terminal (0025-MADR
// D6, deviations 6 and 7): a choice, a yes or no, or a line, each read a
// line at a time out of raw mode; a secret, read in raw mode with each
// character echoed as *. Input comes through in, the reader that owns
// stdin; questions are written to out.
type questions struct {
	ctx context.Context
	in  keysIn
	out io.Writer
	// raw puts the terminal in raw mode and returns the restore; nil reads
	// a secret as a line, as when stdin is not a terminal.
	raw func() (restore func() error, err error)
	// pending is what was read past the last answer; afterCR is whether
	// that answer ended with \r, so a \n after it is not an empty answer.
	pending []byte
	afterCR bool
}

// option is one answer to a choice: its label and a detail shown beside it.
type option struct{ label, detail string }

// notify tells the user something, on a line of its own.
func (q *questions) notify(format string, args ...any) {
	fmt.Fprintf(q.out, format+"\n", args...) //nolint:errcheck // a question's notice is best effort
}

// choose asks for one of opts by its number, def's when the answer is
// empty, and asks again when it names none. One option is chosen without
// asking.
func (q *questions) choose(title string, opts []option, def int) (int, error) {
	if len(opts) == 1 {
		return 0, nil
	}
	width := 0
	for _, o := range opts {
		width = max(width, utf8.RuneCountInString(o.label))
	}
	var b strings.Builder
	b.WriteString(title + "\n")
	for i, o := range opts {
		line := fmt.Sprintf("  %*d  %-*s  %s", len(strconv.Itoa(len(opts))), i+1, width, o.label, o.detail)
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}
	if _, err := io.WriteString(q.out, b.String()); err != nil {
		return 0, fmt.Errorf("write the question: %w", err)
	}
	for {
		fmt.Fprintf(q.out, "Number [%d]: ", def+1) //nolint:errcheck // the answer is read all the same
		s, err := q.line()
		if err != nil {
			return 0, err
		}
		if s == "" {
			return def, nil
		}
		if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(opts) {
			return n - 1, nil
		}
		q.notify("No choice %q: enter a number from 1 to %d.", s, len(opts))
	}
}

// confirm asks a yes or no question, def when the answer is empty.
func (q *questions) confirm(question string, def bool) (bool, error) {
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	for {
		fmt.Fprintf(q.out, "%s %s: ", question, hint) //nolint:errcheck // the answer is read all the same
		s, err := q.line()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(s) {
		case "":
			return def, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		q.notify("Answer y or n.")
	}
}

// input asks for a line of visible text.
func (q *questions) input(prompt string) (string, error) {
	fmt.Fprintf(q.out, "%s: ", prompt) //nolint:errcheck // the answer is read all the same
	return q.line()
}

// line is the next line typed, trimmed, with any bracketed-paste markers
// taken out. The end of input and ctx ending are errCancelled.
func (q *questions) line() (string, error) {
	for {
		if q.afterCR && len(q.pending) > 0 && q.pending[0] == '\n' {
			q.pending = q.pending[1:]
		}
		if len(q.pending) > 0 {
			q.afterCR = false
		}
		if i := bytes.IndexAny(q.pending, "\r\n"); i >= 0 {
			s := string(q.pending[:i])
			q.afterCR = q.pending[i] == '\r'
			q.pending = q.pending[i+1:]
			return plainLine(s), nil
		}
		if err := q.fill(); err != nil {
			return "", err
		}
	}
}

// fill reads the next chunk into pending.
func (q *questions) fill() error {
	buf := make([]byte, 512)
	n, err := q.in.ReadContext(q.ctx, buf)
	q.pending = append(q.pending, buf[:n]...)
	if n == 0 && err != nil {
		return fmt.Errorf("%w (%w)", errCancelled, err)
	}
	return nil
}

// plainLine is s trimmed, without bracketed-paste markers.
func plainLine(s string) string {
	s = strings.ReplaceAll(s, "\x1b[200~", "")
	return strings.TrimSpace(strings.ReplaceAll(s, "\x1b[201~", ""))
}

// Keys a secret reads in raw mode.
const (
	keyCtrlC     = 0x03
	keyCtrlD     = 0x04
	keyBackspace = 0x08
	keyCtrlU     = 0x15
	keyEscape    = 0x1b
	keyDelete    = 0x7f
)

// secret asks for a credential in raw mode, echoing * for each character
// typed or pasted: Backspace erases one, Ctrl+U all, Enter ends, Ctrl+C
// cancels, as Ctrl+D does on an empty answer. An escape sequence, a
// bracketed-paste marker among them, is not part of it. The value is never
// written anywhere (0025-MADR D6, deviation 7). Raw mode is on before the
// prompt is written, so nothing typed after it is echoed by the terminal.
func (q *questions) secret(prompt string) (string, error) {
	if q.raw == nil {
		fmt.Fprintf(q.out, "%s: ", prompt) //nolint:errcheck // the answer is read all the same
		return q.line()
	}
	restore, err := q.raw()
	if err != nil {
		return "", err
	}
	fmt.Fprintf(q.out, "%s: ", prompt) //nolint:errcheck // the answer is read all the same
	s, err := q.readSecret()
	if rerr := restore(); rerr != nil && err == nil {
		err = fmt.Errorf("restore the terminal: %w", rerr)
	}
	fmt.Fprintln(q.out) //nolint:errcheck // the line end is best effort
	return s, err
}

// readSecret reads the secret's bytes up to Enter; q.raw's mode is on.
func (q *questions) readSecret() (string, error) {
	var b []byte
	esc := 0 // 1 after ESC, 2 inside a control sequence
	for {
		if len(q.pending) == 0 {
			if err := q.fill(); err != nil {
				return "", err
			}
		}
		c := q.pending[0]
		q.pending = q.pending[1:]
		switch {
		case esc == 1:
			esc = 0
			if c == '[' {
				esc = 2
			}
		case esc == 2:
			if c >= 0x40 && c <= 0x7e {
				esc = 0
			}
		case c == '\r' || c == '\n':
			q.afterCR = c == '\r'
			return strings.TrimSpace(string(b)), nil
		case c == keyCtrlC, c == keyCtrlD && len(b) == 0:
			return "", errCancelled
		case c == keyEscape:
			esc = 1
		case c == keyBackspace || c == keyDelete:
			if _, size := utf8.DecodeLastRune(b); size > 0 {
				b = b[:len(b)-size]
				io.WriteString(q.out, "\b \b") //nolint:errcheck,gosec // the echo is best effort
			}
		case c == keyCtrlU:
			io.WriteString(q.out, strings.Repeat("\b \b", utf8.RuneCount(b))) //nolint:errcheck,gosec // the echo is best effort
			b = b[:0]
		case c >= 0x20:
			b = append(b, c)
			if c < 0x80 || c >= 0xc0 { // a character's first byte
				io.WriteString(q.out, "*") //nolint:errcheck,gosec // the echo is best effort
			}
		}
	}
}
