package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	xterm "golang.org/x/term"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// ErrExit is returned by ReadPrompt for a second Ctrl+C on an empty line
// inside the exit window.
var ErrExit = errors.New("editor: exit requested")

// History stores submissions. Index 0 of At is the most recent. The editor
// browses it with ↑ and ↓; ReadPrompt adds each whole submission.
type History interface {
	Add(entry string)
	Len() int
	At(idx int) string
}

// Options configure an Editor. Empty prompts take the defaults "> " and
// "… "; pass "... " for ASCII glyphs. Complete, when set, is called for Tab
// with the line and the cursor's byte offset. Header, when set, is read
// before each new prompt and written on its own line above it, never above
// a continuation prompt (0024-MADR D8). Width, when set, is the terminal's
// width, read at each drawing; the line is laid out to it, at most to
// MaxWidth, and to DefaultWidth when it is unknown (0024-MADR D9, 0027-MADR
// D2). Height, when set, is the terminal's height, read at each drawing: a
// line taller than it is drawn from the rows around the cursor's
// (0027-MADR D3); 0 or unset draws every row.
type Options struct {
	Prompt, Continue string
	History          History
	Complete         func(line string, pos int) (newLine string, newPos int, ok bool)
	Header           func() string
	Width            func() int
	Height           func() int
}

// The prompt's widths (0024-MADR D9): the most columns a line takes, and
// the width when the terminal's is unknown.
const (
	MaxWidth     = 120
	DefaultWidth = 80
)

// Bracketed paste on and off.
const (
	pasteModeOn  = "\x1b[?2004h"
	pasteModeOff = "\x1b[?2004l"
)

// Editor reads one submission at a time from a terminal in raw mode: the
// line in a buffer, laid out in whole words and drawn again after each key
// (0027-MADR D1–D3).
type Editor struct {
	keys *keyReader
	out  io.Writer
	errw io.Writer
	opts Options
	sub  coalescer

	line *lineBuffer
	draw renderer
	// in holds bytes read and not yet used; readErr, the error of the
	// last read, returned once they are.
	in      []byte
	readErr error
	// pasting says bracketed paste has been turned on.
	pasting bool
}

// New returns an Editor reading in (normally a started *Reader) and drawing
// on out. Notices go to errw. The first ReadPrompt turns bracketed paste
// on; Restore turns it off.
func New(in io.Reader, out, errw io.Writer, opts Options) *Editor {
	if opts.Prompt == "" {
		opts.Prompt = "> "
	}
	if opts.Continue == "" {
		opts.Continue = "… "
	}
	e := &Editor{keys: &keyReader{src: in}, out: out, errw: errw, opts: opts, draw: renderer{out: out}}
	e.line = newLineBuffer(opts.History)
	e.line.complete = e.onKey
	return e
}

// Restore turns bracketed paste off. Call it before the process exits or
// hands the terminal to another program.
func (e *Editor) Restore() error {
	if _, err := io.WriteString(e.out, pasteModeOff); err != nil {
		return fmt.Errorf("bracketed paste off: %w", err)
	}
	return nil
}

// ReadPrompt returns the next submission. Empty submissions are skipped.
// It returns io.EOF for Ctrl+D on an empty line, /exit and /quit, and
// ErrExit for a second Ctrl+C. When ctx is done it returns ctx's error.
func (e *Editor) ReadPrompt(ctx context.Context) (string, error) {
	e.keys.ctx = ctx
	if !e.pasting {
		if _, err := io.WriteString(e.out, pasteModeOn); err != nil {
			return "", fmt.Errorf("bracketed paste on: %w", err)
		}
		e.pasting = true
	}
	if err := e.header(); err != nil {
		return "", err
	}
	for {
		line, pasted, err := e.readLine(e.prompt())
		switch {
		case errors.Is(err, io.EOF):
			e.sub = coalescer{}
			if e.keys.takeExit() {
				return "", ErrExit
			}
			return "", io.EOF
		case err != nil:
			return "", err
		case pasted:
			e.sub.addPasted(line)
			continue
		}
		if e.keys.takeNotice() {
			if err := e.notify("Press Ctrl+C again to exit"); err != nil {
				return "", err
			}
			continue
		}
		if !e.sub.addTyped(line) {
			continue
		}
		text, pastes := e.sub.take()
		if pastes > 0 {
			if err := e.notify(fmt.Sprintf("%s %d pasted lines", strings.TrimSpace(e.opts.Continue), pastes+1)); err != nil {
				return "", err
			}
		}
		switch strings.TrimSpace(text) {
		case "":
			continue
		case "/exit", "/quit":
			return "", io.EOF
		}
		if e.opts.History != nil {
			e.opts.History.Add(text)
		}
		return text, nil
	}
}

// readLine reads one line after prompt, drawing it after each read's keys:
// the line and whether it was pasted (a line ended inside a bracketed
// paste), or io.EOF for Ctrl+D on an empty line, or the read's error. A
// line ended stays drawn, the cursor moved below it.
func (e *Editor) readLine(prompt string) (string, bool, error) {
	e.line.prompt = prompt
	if err := e.redraw(); err != nil {
		return "", false, err
	}
	buf := make([]byte, chunkSize)
	for {
		before := len(e.in)
		rest, ev := e.line.feed(e.in)
		e.in = rest
		switch ev {
		case lineEnded:
			e.line.pos = len(e.line.text)
			if err := e.redraw(); err != nil {
				return "", false, err
			}
			if err := e.draw.finish(); err != nil {
				return "", false, err
			}
			line, pasted := e.line.take()
			return line, pasted, nil
		case endOfInput:
			return "", false, io.EOF
		case clearScreen:
			if _, err := io.WriteString(e.out, "\x1b[2J\x1b[H"); err != nil {
				return "", false, fmt.Errorf("clear the screen: %w", err)
			}
			e.draw.forget()
			if err := e.redraw(); err != nil {
				return "", false, err
			}
			continue
		}
		if len(e.in) != before {
			if err := e.redraw(); err != nil {
				return "", false, err
			}
		}
		if e.readErr != nil {
			err := e.readErr
			e.readErr = nil
			return "", false, err
		}
		n, err := e.keys.Read(buf)
		e.in = append(e.in, buf[:n]...)
		e.readErr = err
	}
}

// redraw lays the line out at the terminal's width now and draws it in
// place of the last drawing.
func (e *Editor) redraw() error {
	l := e.line
	l.columns = DefaultWidth
	if e.opts.Width != nil {
		l.columns = lineWidth(e.opts.Width())
	}
	height := 0
	if e.opts.Height != nil {
		height = max(e.opts.Height(), 0)
	}
	return e.draw.draw(layout(l.prompt, l.text, l.pos, l.columns), height)
}

// header writes Options.Header above the new prompt: one line, with escape
// sequences and control characters dropped and line breaks made spaces.
func (e *Editor) header() error {
	if e.opts.Header == nil || e.sub.open() {
		return nil
	}
	h := strings.Join(strings.Fields(term.Sanitize(e.opts.Header())), " ")
	if h == "" {
		return nil
	}
	if _, err := io.WriteString(e.out, h+"\r\n"); err != nil {
		return fmt.Errorf("editor header: %w", err)
	}
	return nil
}

func (e *Editor) prompt() string {
	if e.sub.open() {
		return e.opts.Continue
	}
	return e.opts.Prompt
}

// notify writes one line to errw. A line has just ended, so the cursor is
// at column 0 and the next prompt is drawn below it.
func (e *Editor) notify(msg string) error {
	if _, err := fmt.Fprintf(e.errw, "%s\r\n", msg); err != nil {
		return fmt.Errorf("editor notice: %w", err)
	}
	return nil
}

// onKey is the buffer's completion callback: Ctrl+C's key and Tab.
func (e *Editor) onKey(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
	switch {
	case key == keyInterrupt:
		return e.keys.interrupt(line)
	case key == '\t' && e.opts.Complete != nil:
		return e.opts.Complete(line, pos)
	}
	return "", 0, false
}

// Raw puts the terminal on f into raw mode and returns the func that
// restores it. On Windows this also enables virtual-terminal input, which
// is what makes Console Host send bracketed paste (measurement 10).
func Raw(f *os.File) (restore func() error, err error) {
	fd := int(f.Fd()) //nolint:gosec // a console handle or fd fits in int
	state, err := xterm.MakeRaw(fd)
	if err != nil {
		return nil, fmt.Errorf("raw mode: %w", err)
	}
	return func() error { return xterm.Restore(fd, state) }, nil
}
