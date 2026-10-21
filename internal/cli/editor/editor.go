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

// History stores submissions. Index 0 of At is the most recent. It is the
// shape of x/term's History, which FileHistory implements.
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
// width, read before each prompt; the line breaks there, at most at
// MaxWidth, and at DefaultWidth when it is unknown (0024-MADR D9).
type Options struct {
	Prompt, Continue string
	History          History
	Complete         func(line string, pos int) (newLine string, newPos int, ok bool)
	Header           func() string
	Width            func() int
}

// The prompt's widths (0024-MADR D9): the most columns a line takes, and
// the width when the terminal's is unknown.
const (
	MaxWidth     = 120
	DefaultWidth = 80
)

// Editor reads one submission at a time from a terminal in raw mode.
type Editor struct {
	term *xterm.Terminal
	keys *keyReader
	errw io.Writer
	opts Options
	sub  coalescer
}

// New returns an Editor reading in (normally a started *Reader) and drawing
// on out. Notices go to errw. It turns bracketed paste on; Restore turns it
// off.
func New(in io.Reader, out, errw io.Writer, opts Options) *Editor {
	if opts.Prompt == "" {
		opts.Prompt = "> "
	}
	if opts.Continue == "" {
		opts.Continue = "… "
	}
	k := &keyReader{src: in}
	t := xterm.NewTerminal(readWriter{Reader: k, Writer: out}, opts.Prompt)
	t.History = viewHistory{h: opts.History}
	e := &Editor{term: t, keys: k, errw: errw, opts: opts}
	t.AutoCompleteCallback = e.onKey
	t.SetBracketedPasteMode(true)
	return e
}

// Restore turns bracketed paste off. Call it before the process exits or
// hands the terminal to another program.
func (e *Editor) Restore() {
	e.term.SetBracketedPasteMode(false)
}

// ReadPrompt returns the next submission. Empty submissions are skipped.
// It returns io.EOF for Ctrl+D on an empty line, /exit and /quit, and
// ErrExit for a second Ctrl+C. When ctx is done it returns ctx's error.
func (e *Editor) ReadPrompt(ctx context.Context) (string, error) {
	e.keys.ctx = ctx
	if err := e.size(); err != nil {
		return "", err
	}
	if err := e.header(); err != nil {
		return "", err
	}
	for {
		e.term.SetPrompt(e.prompt())
		line, err := e.term.ReadLine()
		switch {
		case errors.Is(err, xterm.ErrPasteIndicator):
			e.sub.addPasted(line)
			continue
		case errors.Is(err, io.EOF):
			e.sub = coalescer{}
			if e.keys.takeExit() {
				return "", ErrExit
			}
			return "", io.EOF
		case err != nil:
			return "", err
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
		text, pasted := e.sub.take()
		if pasted > 0 {
			if err := e.notify(fmt.Sprintf("%s %d pasted lines", strings.TrimSpace(e.opts.Continue), pasted+1)); err != nil {
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

// size gives the terminal the width the line breaks at: Options.Width's,
// to MaxWidth, or DefaultWidth when it is unknown.
func (e *Editor) size() error {
	w := DefaultWidth
	if e.opts.Width != nil {
		w = lineWidth(e.opts.Width())
	}
	if err := e.term.SetSize(w, 24); err != nil {
		return fmt.Errorf("editor width: %w", err)
	}
	return nil
}

// header writes Options.Header above the new prompt, through the terminal,
// whose count of the cursor it keeps: one line, with escape sequences and
// control characters dropped and line breaks made spaces.
func (e *Editor) header() error {
	if e.opts.Header == nil || e.sub.open() {
		return nil
	}
	h := strings.Join(strings.Fields(term.Sanitize(e.opts.Header())), " ")
	if h == "" {
		return nil
	}
	if _, err := e.term.Write([]byte(h + "\n")); err != nil {
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

// notify writes one line to errw. ReadLine has just ended a line, so the
// cursor is at column 0 and the next prompt is drawn below it.
func (e *Editor) notify(msg string) error {
	if _, err := fmt.Fprintf(e.errw, "%s\r\n", msg); err != nil {
		return fmt.Errorf("editor notice: %w", err)
	}
	return nil
}

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

type readWriter struct {
	io.Reader
	io.Writer
}

// viewHistory lets the Terminal browse History without adding to it: the
// Terminal would add each pasted line separately, and ReadPrompt adds the
// whole submission instead.
type viewHistory struct{ h History }

func (viewHistory) Add(string) {}

func (v viewHistory) Len() int {
	if v.h == nil {
		return 0
	}
	return v.h.Len()
}

func (v viewHistory) At(idx int) string { return v.h.At(idx) }
