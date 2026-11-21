package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// errCancelled ends a sign-in the user left: Ctrl+C, the end of input, or
// an empty answer where one is needed.
var errCancelled = errors.New("cancelled")

// keysIn is the reader that owns stdin: go-tui-lib's inline.Reader, which
// the terminal input reads (0063-MADR D1).
type keysIn interface {
	Read(ctx context.Context, p []byte) (int, error)
}

// questions asks the user what a sign-in needs, on the terminal (0025-MADR
// D6, deviations 6 and 7): a choice, a yes or no, or a line, each read a
// line at a time out of raw mode; a secret, read in raw mode with each
// character echoed as *. Input comes through in, the process's one
// terminal input, which keeps what was read past an answer for the next
// question, across asks, and reads every line end one way (0063-MADR D2,
// D3); questions are written to out.
type questions struct {
	ctx context.Context
	in  *terminalInput
	out io.Writer
	// raw puts the terminal in raw mode and returns the restore; nil reads
	// a secret as a line, as when stdin is not a terminal.
	raw func() (restore func() error, err error)
}

// option is one answer to a choice: its label and a detail shown beside it.
type option struct{ label, detail string }

// prompter is a sign-in's prompter over questions (0031-MADR D7): what the
// SDK's wizard asks is asked there, and an empty secret ends the sign-in.
type prompter struct{ q *questions }

// Choose implements provider.Prompter.
func (p prompter) Choose(title string, choices []provider.Choice, def int) (int, error) {
	opts := make([]option, len(choices))
	for i, c := range choices {
		opts[i] = option{c.Label, c.Detail}
	}
	return p.q.choose(title, opts, def)
}

// Confirm implements provider.Prompter.
func (p prompter) Confirm(question string, def bool) (bool, error) { return p.q.confirm(question, def) }

// Input implements provider.Prompter: the read gives up when ctx ends.
// When ctx ends while the question's own context goes on, the program gave
// the read up (a browser sign-in's loopback came first), and what completes
// it is dropped, never an answer or a turn (0063-MADR D13, D17).
func (p prompter) Input(ctx context.Context, prompt string) (string, error) {
	return p.q.inputContext(ctx, prompt)
}

// Secret implements provider.Prompter; an empty answer cancels.
func (p prompter) Secret(prompt string) (string, error) {
	s, err := p.q.secret(prompt)
	if err == nil && s == "" {
		return "", errCancelled
	}
	return s, err
}

// Notify implements provider.Prompter.
func (p prompter) Notify(format string, args ...any) { p.q.notify(format, args...) }

// unasked is a sign-in's prompter where nothing may be asked: no terminal,
// or --json. Its notices go to out.
type unasked struct{ out io.Writer }

// errNoTerminal is a sign-in that would ask a question with no terminal.
var errNoTerminal = errors.New("this sign-in asks a question, and there is no terminal to ask it on")

func (unasked) Choose(string, []provider.Choice, int) (int, error) { return 0, errNoTerminal }
func (unasked) Confirm(string, bool) (bool, error)                 { return false, errNoTerminal }
func (unasked) Input(context.Context, string) (string, error)      { return "", errNoTerminal }
func (unasked) Secret(string) (string, error)                      { return "", errNoTerminal }
func (u unasked) Notify(format string, args ...any) {
	fmt.Fprintf(u.out, format+"\n", args...) //nolint:errcheck // a notice is best effort
}

var (
	_ provider.Prompter = prompter{}
	_ provider.Prompter = unasked{}
)

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
	q.in.warnQueued() // before the question's text (0063-MADR D23, row R156)
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
		q.in.warnQueued()
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
		// Before the question's text (0063-MADR D23, row R156).
		q.in.warnQueued()
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
func (q *questions) input(prompt string) (string, error) { return q.inputContext(q.ctx, prompt) }

// inputContext is input, read until ctx ends.
func (q *questions) inputContext(ctx context.Context, prompt string) (string, error) {
	// Before the question's text (0063-MADR D23, row R156).
	q.in.warnQueued()
	fmt.Fprintf(q.out, "%s: ", prompt) //nolint:errcheck // the answer is read all the same
	return q.lineContext(ctx)
}

// line is the next line typed, trimmed, with any bracketed-paste markers
// taken out. The end of input and ctx ending are errCancelled.
func (q *questions) line() (string, error) { return q.lineContext(q.ctx) }

// lineContext is line, read until ctx ends.
func (q *questions) lineContext(ctx context.Context) (string, error) {
	s, err := q.in.line(ctx)
	if err != nil {
		return "", q.cancelled(ctx, err)
	}
	return plainLine(s), nil
}

// cancelled is a question's read that ended with err: errCancelled, with
// err. A read that ended on ctx while the question's own context goes on
// was given up by the program (a browser sign-in's loopback came first), so
// its completion is dropped when it comes; one the person gave up with
// Ctrl+C ends the question's own context and is not (0063-MADR D13, D17).
// On a POSIX terminal the program's give-up also flushes the terminal's
// queue, so what was typed into the prompt is never a later prompt's, nor
// the shell's after gobble exits (0063-MADR D22, Q23).
func (q *questions) cancelled(ctx context.Context, err error) error {
	if ctx.Err() != nil && q.ctx.Err() == nil {
		q.in.abandon()
		q.in.flushGiveUp()
	}
	return fmt.Errorf("%w (%w)", errCancelled, err)
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
// prompt is written, so nothing typed after it is echoed by the terminal;
// it is restored in a deferred call, so a panic restores it before it goes
// on (0063-MADR D15). When raw mode cannot be switched on, its error is
// returned, no prompt is written and nothing is read (row R28).
func (q *questions) secret(prompt string) (s string, err error) {
	q.in.warnQueued() // before raw mode and the question's text (0063-MADR D23, row R156)
	if q.raw == nil {
		fmt.Fprintf(q.out, "%s: ", prompt) //nolint:errcheck // the answer is read all the same
		return q.line()
	}
	restore, err := q.raw()
	if err != nil {
		return "", err
	}
	defer func() {
		if rerr := restore(); rerr != nil && err == nil {
			err = fmt.Errorf("restore the terminal: %w", rerr)
		}
		fmt.Fprintln(q.out) //nolint:errcheck // the line end is best effort
	}()
	fmt.Fprintf(q.out, "%s: ", prompt) //nolint:errcheck // the answer is read all the same
	return q.readSecret()
}

// readSecret reads the secret's bytes up to Enter; q.raw's mode is on. The
// Enter that ends it is the terminal input's raw line end (0063-MADR D3).
func (q *questions) readSecret() (string, error) {
	var b []byte
	esc := 0 // 1 after ESC, 2 inside a control sequence
	for {
		c, err := q.in.key(q.ctx)
		if err != nil {
			return "", q.cancelled(q.ctx, err)
		}
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
			q.in.rawEnd(c)
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
