package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
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
// before each new prompt and drawn on its own line above it, never above a
// continuation (0024-MADR D8). Width, when set, is the terminal's width,
// read at each prompt and resize; the line is laid out to it, at most to
// MaxWidth, and to DefaultWidth when it is unknown (0024-MADR D9, 0027-MADR
// D2). Height, when set, is the terminal's height, read at each key: a
// line taller than it is drawn from the rows around the cursor's
// (0027-MADR D3); 0 or unset draws every row.
type Options struct {
	Prompt, Continue string
	History          History
	Complete         func(line string, pos int) (newLine string, newPos int, ok bool)
	Header           func() string
	Width            func() int
	Height           func() int
	// Resize, when set, notices that the terminal was resized: the line is
	// drawn again at the new width without waiting for a key (0030-MADR D2).
	Resize <-chan struct{}
	// Shown, when set, is given what the editor leaves on the screen, the
	// header, each line sent and each notice, as its size in bytes and how
	// it is drawn at a terminal's width (0030-MADR D4).
	Shown func(size int, at func(width int) string)
	// Reprint, when set, is what the screen showed before the line, drawn
	// at a width: once resizes settle, the screen and its scrollback are
	// cleared and it is written again, then the line (0030-MADR D3).
	Reprint func(width int) string
	// External, when set, edits the line's text in the user's editor, for
	// ctrl+x ctrl+e, and returns what it holds afterwards; on an error the
	// line is left as it was (0035-PLAN P17).
	External func(ctx context.Context, text string) (string, error)
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

const (
	// exitWindow is how long a second Ctrl+C on an empty line exits.
	exitWindow = 2 * time.Second
	// settleWait is how long after the last resize's redraw the
	// conversation is reprinted (0030-MADR D3).
	settleWait = 150 * time.Millisecond
	// chunkSize is the most bytes one read of the terminal takes.
	chunkSize = 512
)

// Editor reads one submission at a time from a terminal in raw mode. Each
// prompt is a go-tui-lib lineedit.Composer, drawn in an inline.Region
// below the scrollback; the Editor is its host and keeps what is gobble's:
// the Ctrl+C exit window, /exit and /quit, skipping an empty submission,
// the paste notice, the history, the reprint once resizes settle, and the
// 120-column cap (0035-MADR D14 step 4; 0035-PLAN P17, PQ7).
type Editor struct {
	in   *inline.Reader
	out  io.Writer
	errw io.Writer
	opts Options

	keys   *keymap.Keymap
	region *inline.Region
	dec    termkey.Decoder
	hist   *submissions

	// line is the prompt being read, and head the header drawn above it;
	// line is nil between prompts.
	line *lineedit.Composer
	head string
	// held is what was read and decoded and not yet given to a prompt;
	// readErr, the read's error, returned once held is used.
	held    []termkey.Input
	readErr error
	// escDue is when the decoder gives up a held escape, the input quiet
	// since the last bytes (termkey.DefaultEscTimeout).
	escDue time.Time
	// width is the width the prompt is laid out to.
	width int
	// pasting says bracketed paste has been turned on.
	pasting bool
	// armedUntil ends the exit window.
	armedUntil time.Time
	// resized says the terminal was resized and the region's redraw is
	// awaited; settleAt is when the reprint is due, zero for none.
	resized  bool
	settleAt time.Time
	// pending is what the prompt asked to write, written at once before
	// each read, each notice and the external editor, and on return.
	pending strings.Builder
	// interrupted, notice and external are what the last input asked of
	// the host, done once the Composer's writes before them are written.
	interrupted *string
	notice      string
	external    *string
}

// New returns an Editor reading in, the process's one reader of the
// terminal, and drawing on out. Notices go to errw. The first ReadPrompt
// turns bracketed paste on; Restore turns it off.
func New(in *inline.Reader, out, errw io.Writer, opts Options) *Editor {
	if opts.Prompt == "" {
		opts.Prompt = "> "
	}
	if opts.Continue == "" {
		opts.Continue = "… "
	}
	reflow := inline.ReflowDown // re-wraps (0030-PLAN P1)
	if os.Getenv("XTERM_VERSION") != "" {
		reflow = inline.ReflowCut // xterm cuts a narrowed row
	}
	keys, err := keymap.New(keymap.Settings{}, lineedit.DefaultRules())
	if err != nil {
		panic(fmt.Sprintf("editor: lineedit's default keys: %v", err)) // the library's own rules
	}
	return &Editor{in: in, out: out, errw: errw, opts: opts, keys: keys,
		region: inline.NewRegion(inline.WithReflow(reflow)), hist: &submissions{h: opts.History}}
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
	text, err := e.nextSubmission(ctx)
	if ferr := e.flush(); ferr != nil && err == nil {
		return "", ferr
	}
	return text, err
}

func (e *Editor) nextSubmission(ctx context.Context) (string, error) {
	if !e.pasting {
		if _, err := io.WriteString(e.out, pasteModeOn); err != nil {
			return "", fmt.Errorf("bracketed paste on: %w", err)
		}
		e.pasting = true
	}
	if e.line == nil {
		e.open()
	}
	if text, done, err := e.drain(ctx); done || err != nil {
		return text, err
	}
	buf := make([]byte, chunkSize)
	for {
		for len(e.held) > 0 {
			in := e.held[0]
			e.held = e.held[1:]
			e.input(in, time.Now())
			if text, done, err := e.drain(ctx); done || err != nil {
				return text, err
			}
		}
		if err := e.readErr; err != nil {
			e.readErr = nil
			return "", err // the input's end or error; the prompt stays drawn
		}
		n, resized, err := e.read(ctx, buf)
		if err != nil && ctx.Err() != nil {
			return "", ctx.Err()
		}
		now := time.Now()
		e.held = e.dec.Decode(buf[:n], e.held)
		if err != nil {
			e.held = e.dec.Flush(e.held)
			e.readErr = err
		} else if n == 0 && e.dec.Pending() && !time.Now().Before(e.escDue) {
			e.held = e.dec.Flush(e.held) // a lone escape, the input quiet since
		}
		if n > 0 {
			e.escDue = now.Add(termkey.DefaultEscTimeout)
		}
		if resized {
			e.resize(now)
		}
		e.timeout(now)
		if text, done, err := e.drain(ctx); done || err != nil {
			return text, err
		}
	}
}

// open starts a new prompt at the terminal's width now, with the header
// read for it.
func (e *Editor) open() {
	e.head = ""
	if e.opts.Header != nil {
		e.head = strings.Join(strings.Fields(term.CleanLine(e.opts.Header(), true)), " ") // the editor draws on a terminal
	}
	head := e.head
	o := []lineedit.Option{lineedit.WithRegion(e.region), lineedit.WithKeymap(e.keys),
		lineedit.WithPrompt(e.opts.Prompt), lineedit.WithContinue(e.opts.Continue), lineedit.WithGlyphs(glyphsOf(e.opts.Continue)),
		lineedit.WithHistory(e.hist), lineedit.WithPaste(lineedit.PasteInsert), lineedit.WithLimit(lineedit.DefaultLimit),
		lineedit.WithClearScreen(), lineedit.WithHeader(func() string { return head })}
	if e.opts.Complete != nil {
		o = append(o, lineedit.WithCompleter(lineedit.CompleterFunc(e.opts.Complete)))
	}
	if e.opts.External != nil {
		o = append(o, lineedit.WithExternalEditor())
	}
	e.line = lineedit.NewComposer(o...)
	e.setHeight()
	e.width = e.terminalWidth()
	e.line.Resize(e.width, time.Now())
}

// glyphsOf is the glyph set the continuation prompt is drawn from: ASCII's
// for "... ".
func glyphsOf(cont string) glyph.Set { return glyph.For(cont != "... ") }

// input gives one decoded input to the prompt, at the terminal's width
// now (0027-MADR D3). A paste's line breaks are LF, as Console Host's CR
// are not (0008-MADR measurement 10), and a paste of more than one line
// asks for the notice.
func (e *Editor) input(in termkey.Input, now time.Time) {
	if e.terminalWidth() != e.width {
		e.resize(now)
	}
	if in.Kind == termkey.InputPaste {
		in.Text = strings.ReplaceAll(strings.ReplaceAll(in.Text, "\r\n", "\n"), "\r", "\n")
		if n := strings.Count(in.Text, "\n"); n > 0 {
			e.notice = fmt.Sprintf("%s %d pasted lines", strings.TrimSpace(e.opts.Continue), n+1)
		}
	}
	e.setHeight()
	e.line.HandleInput(in, now)
}

// resize lays the prompt out at the terminal's new width; the region asks
// where the cursor is and redraws from there (0030-MADR D2).
func (e *Editor) resize(now time.Time) {
	e.setHeight()
	e.width = e.terminalWidth()
	e.line.Resize(e.width, now)
	e.resized, e.settleAt = true, time.Time{}
}

// timeout runs the prompt's wait, for the cursor report or a key
// sequence, once the clock has passed it.
func (e *Editor) timeout(now time.Time) {
	if at, ok := e.line.Timeout(); ok && !now.Before(at) {
		e.line.HandleTimeout(now)
	}
}

// settle starts the wait for the reprint once a resize's redraw is done,
// and reprints when it is over (0030-MADR D3).
func (e *Editor) settle(now time.Time) {
	if e.resized {
		if _, waiting := e.region.Timeout(); waiting {
			return
		}
		e.resized = false
		if e.opts.Reprint != nil {
			e.settleAt = now.Add(settleWait)
		}
	}
	if e.settleAt.IsZero() || now.Before(e.settleAt) {
		return
	}
	e.settleAt = time.Time{}
	text := e.opts.Reprint(e.width)
	if text == "" {
		e.write(e.region.Clear() + "\x1b[2J\x1b[3J\x1b[H")
		e.redraw()
		return
	}
	e.write(e.region.Print("\x1b[2J\x1b[3J\x1b[H" + text)) // one synchronized update, the prompt redrawn below
}

// drain writes what the prompt asks and acts on what it gives: a
// submission is returned, and /exit, /quit, the end of input and a second
// Ctrl+C end the read. A prompt that ends leaves what it committed in the
// scrollback, the cursor at the start of the row below, where the
// caller's output comes next: the empty prompt the Composer draws below a
// submission, and the region's clearing of it, are both dropped.
func (e *Editor) drain(ctx context.Context) (string, bool, error) {
	var text string
	submitted, sent := false, false
	var end error
	for {
		for ev := e.line.NextEvent(); ev.Kind != lineedit.EditNone; ev = e.line.NextEvent() {
			switch ev.Kind {
			case lineedit.EditWrite:
				if !sent {
					e.write(ev.Data)
				}
			case lineedit.EditSubmit:
				switch strings.TrimSpace(ev.Text) {
				case "":
				case "/exit", "/quit":
					end, sent = io.EOF, true
				default:
					text, submitted, sent = ev.Text, true, true
				}
			case lineedit.EditEOF:
				if end == nil {
					end = io.EOF
				}
			case lineedit.EditInterrupt:
				t := ev.Text
				e.interrupted = &t
			case lineedit.EditExternal:
				t := ev.Text
				e.external = &t
			}
		}
		acted, err := e.act(ctx)
		if err != nil {
			end = err
		}
		if !acted { // else the writes it asked for come next
			break
		}
	}
	if !submitted && end == nil {
		e.settle(time.Now())
		return "", false, nil
	}
	if c := e.region.Clear(); !sent {
		e.write(c)
	}
	e.line, e.resized, e.settleAt = nil, false, time.Time{}
	if end != nil {
		return "", true, end
	}
	e.showSubmission(text)
	e.hist.record(text)
	return text, true, nil
}

// act does what the last input asked of the host once the Composer's
// writes before it are written, and says whether it did anything.
func (e *Editor) act(ctx context.Context) (bool, error) {
	acted := false
	if msg := e.notice; msg != "" {
		e.notice, acted = "", true
		if err := e.noticeAbove(msg); err != nil {
			return true, err
		}
	}
	if t := e.interrupted; t != nil {
		e.interrupted, acted = nil, true
		if err := e.interrupt(*t); err != nil {
			return true, err
		}
	}
	if t := e.external; t != nil {
		e.external, acted = nil, true
		e.write(e.region.Clear())
		if err := e.flush(); err != nil {
			return true, err
		}
		if edited, err := e.opts.External(ctx, *t); err == nil {
			e.line.SetText(edited)
		}
		e.redraw()
	}
	return acted, nil
}

// interrupt is Ctrl+C: a non-empty line is cleared; on an empty line the
// first press arms the exit window and says so above the prompt, and a
// second inside it exits, the prompt left on the screen.
func (e *Editor) interrupt(text string) error {
	if text != "" {
		e.armedUntil = time.Time{}
		e.line.SetText("")
		return nil
	}
	now := time.Now()
	if now.Before(e.armedUntil) {
		e.line.Cancel()
		return ErrExit
	}
	e.armedUntil = now.Add(exitWindow)
	return e.noticeAbove("Press Ctrl+C again to exit")
}

// noticeAbove writes msg to errw on its own line above the prompt, which
// is drawn again below it.
func (e *Editor) noticeAbove(msg string) error {
	e.write(e.region.Clear())
	if err := e.flush(); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(e.errw, "%s\r\n", msg); err != nil {
		return fmt.Errorf("editor notice: %w", err)
	}
	if e.opts.Shown != nil {
		e.opts.Shown(len(msg), func(int) string { return msg + "\n" })
	}
	e.redraw()
	return nil
}

// redraw draws the prompt in a cleared region.
func (e *Editor) redraw() {
	rows, row, col := e.line.View(e.width)
	e.write(e.region.Draw(rows, row, col))
}

// read reads the terminal until bytes come, the terminal is resized, or
// the clock makes something due: the prompt's wait, the decoder's held
// escape, or the reprint. resized reports the resize; a wait that is over
// returns no bytes and no error.
func (e *Editor) read(ctx context.Context, p []byte) (n int, resized bool, err error) {
	if err := e.flush(); err != nil {
		return 0, false, err
	}
	var due time.Time
	if at, ok := e.line.Timeout(); ok {
		due = at
	}
	if e.dec.Pending() {
		due = earliest(due, e.escDue)
	}
	if e.resized && e.opts.Reprint != nil {
		due = earliest(due, time.Now().Add(settleWait))
	}
	due = earliest(due, e.settleAt)
	if due.IsZero() && e.opts.Resize == nil {
		n, err = e.in.Read(ctx, p)
		return n, false, err
	}
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var timer <-chan time.Time
	if !due.IsZero() {
		t := time.NewTimer(time.Until(due))
		defer t.Stop()
		timer = t.C
	}
	var gotResize atomic.Bool
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-e.opts.Resize:
			gotResize.Store(true)
		case <-timer:
		case <-stop:
			return
		}
		cancel()
	}()
	n, err = e.in.Read(rctx, p)
	close(stop)
	<-done
	if err != nil && ctx.Err() == nil && rctx.Err() != nil {
		err = nil // a resize or a wait over, not the caller's end
	}
	return n, gotResize.Load(), err
}

// earliest is the earlier of a and b, a zero time being none.
func earliest(a, b time.Time) time.Time {
	switch {
	case a.IsZero():
		return b
	case b.IsZero() || a.Before(b):
		return a
	}
	return b
}

func (e *Editor) setHeight() {
	if e.opts.Height != nil {
		e.region.SetHeight(e.opts.Height())
	}
}

// terminalWidth is the width the prompt is laid out to (0024-MADR D9).
func (e *Editor) terminalWidth() int {
	if e.opts.Width == nil {
		return DefaultWidth
	}
	return lineWidth(e.opts.Width())
}

// lineWidth is the width a prompt is laid out to on a terminal that many
// columns wide: at most MaxWidth, and DefaultWidth when it is unknown.
func lineWidth(terminal int) int {
	if terminal <= 0 {
		return DefaultWidth
	}
	return min(terminal, MaxWidth)
}

// showSubmission gives Options.Shown the header drawn above the line sent,
// and the line, laid out again at whatever width it is drawn at.
func (e *Editor) showSubmission(text string) {
	if e.opts.Shown == nil {
		return
	}
	if h := e.head; h != "" {
		e.opts.Shown(len(h), func(int) string { return h + "\n" })
	}
	prompt, cont := e.opts.Prompt, e.opts.Continue
	e.opts.Shown(len(prompt)+len(text), func(width int) string {
		c := lineedit.NewComposer(lineedit.WithPrompt(prompt), lineedit.WithContinue(cont), lineedit.WithGlyphs(glyphsOf(cont)))
		c.SetText(text)
		rows, row, _ := c.View(lineWidth(width))
		if row == len(rows)-1 && len(rows) > 1 && strings.TrimSpace(rows[row]) == "" {
			rows = rows[:row] // the row the cursor would wrap to, not the text's
		}
		var b strings.Builder
		for _, r := range rows {
			b.WriteString(strings.TrimRight(r, " ") + "\n")
		}
		return b.String()
	})
}

// write keeps s to write with the rest of what the prompt asks.
func (e *Editor) write(s string) { e.pending.WriteString(s) }

// flush writes what the prompt asked, in one write.
func (e *Editor) flush() error {
	if e.pending.Len() == 0 {
		return nil
	}
	_, err := io.WriteString(e.out, e.pending.String())
	e.pending.Reset()
	if err != nil {
		return fmt.Errorf("draw the prompt: %w", err)
	}
	return nil
}

// submissions is the History a prompt recalls. The Composer adds each
// submission it gives; ReadPrompt adds only those it returns, so an empty
// line, /exit and /quit stay out (0027-MADR D4).
type submissions struct{ h History }

// Add is the Composer's, and adds nothing: see record.
func (s *submissions) Add(string) {}

func (s *submissions) record(text string) {
	if s.h != nil {
		s.h.Add(text)
	}
}

func (s *submissions) Len() int {
	if s.h == nil {
		return 0
	}
	return s.h.Len()
}

func (s *submissions) At(idx int) string { return s.h.At(idx) }

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
