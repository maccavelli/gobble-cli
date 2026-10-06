package cli

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/render"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// display is how the line session shows a turn (0008-MADR D7).
type display struct {
	showThinking, verbose, stats bool
	home                         string
}

// turnClock times a turn: the start, the first token and the end.
type turnClock struct {
	start, first, end time.Time
}

func (c *turnClock) token(now time.Time) {
	if c.first.IsZero() {
		c.first = now
	}
}

// view draws one turn's updates on stdout, and its status line on stderr.
// on runs on the connection's notification goroutine and finish on the
// caller's, so the state is locked.
type view struct {
	out    *Output
	caps   term.Caps
	glyphs term.Glyphs
	opts   display
	now    func() time.Time
	// beforeWrite stops the spinner; it is called before the first write.
	beforeWrite func()

	mu      sync.Mutex
	md      render.Stream
	clock   turnClock
	tools   map[string]*toolState
	usage   *acpclient.ContextUsage
	wrote   bool // anything is on stdout
	atBOL   bool // the cursor is at the start of a line
	thought bool // the last text written was thinking
	err     error
}

// toolState is a tool call as last drawn: its fields, and the header line
// shown, so a change that draws the same header (pending, then
// in_progress) prints nothing.
type toolState struct{ title, status, header string }

// header draws the call's header line if it differs from the one shown.
func (v *view) header(st *toolState) {
	h := render.ToolHeader(st.title, st.status, v.caps.Out, v.glyphs)
	if h != st.header {
		st.header = h
		v.line(h)
	}
}

func newView(out *Output, opts display, now func() time.Time) *view {
	return &view{
		out: out, caps: out.caps, glyphs: glyphsFor(out.caps), opts: opts, now: now,
		tools: map[string]*toolState{}, atBOL: true, clock: turnClock{start: now()},
	}
}

func glyphsFor(c term.Caps) term.Glyphs {
	if c.Unicode {
		return term.Unicode
	}
	return term.ASCII
}

// on draws one update.
func (v *view) on(u acpclient.Update) {
	v.mu.Lock()
	defer v.mu.Unlock()
	switch u.Kind {
	case acpclient.KindAgentText:
		v.clock.token(v.now())
		if u.Block != "" {
			v.line("[" + u.Block + "]")
			return
		}
		if v.thought {
			v.thought = false
			v.newline()
		}
		v.text(v.md.Push(term.Sanitize(u.Text)))
	case acpclient.KindThought:
		v.clock.token(v.now())
		if !v.opts.showThinking || u.Text == "" {
			return
		}
		v.flush()
		v.thought = true
		v.write(term.Dim.Wrap(term.Sanitize(u.Text), v.caps.Out.Color))
	case acpclient.KindToolCall:
		st := &toolState{title: u.Tool.Title, status: u.Tool.Status}
		v.tools[u.Tool.ID] = st
		v.header(st)
		v.content(u.Tool.Content)
	case acpclient.KindToolCallUpdate:
		st, ok := v.tools[u.Tool.ID]
		if !ok {
			st = &toolState{}
			v.tools[u.Tool.ID] = st
		}
		if u.Tool.Title != "" {
			st.title = u.Tool.Title
		}
		if u.Tool.Status != "" {
			st.status = u.Tool.Status
		}
		v.header(st)
		v.content(u.Tool.Content)
	case acpclient.KindPlan:
		items := make([]render.PlanItem, 0, len(u.Plan))
		for _, e := range u.Plan {
			items = append(items, render.PlanItem{Content: e.Content, Status: e.Status})
		}
		v.flush()
		v.newline()
		v.write(render.Plan(items, v.caps.Out, v.glyphs))
	case acpclient.KindUsage:
		c := u.Context
		v.usage = &c
	case acpclient.KindOther:
	}
}

// content draws a tool call's content blocks. A diff is a unified diff,
// coloured (0002-PLAN Phase 3 step 6), bounded like other tool output.
func (v *view) content(cs []acpclient.ToolContent) {
	for _, c := range cs {
		switch {
		case c.Diff != nil:
			path := render.ShortPath(term.Sanitize(c.Diff.Path), v.opts.home)
			var old *string
			if c.Diff.OldText != nil {
				o := term.Sanitize(*c.Diff.OldText)
				old = &o
			}
			diff := render.UnifiedDiff(path, old, term.Sanitize(c.Diff.NewText))
			v.flush()
			v.newline()
			v.write(render.Diff(render.ToolOutput(diff, v.opts.verbose, v.glyphs), v.caps.Out))
		case c.Terminal != "":
			v.line("terminal " + term.Sanitize(c.Terminal))
		default:
			v.flush()
			v.newline()
			v.write(render.ToolOutput(c.Text, v.opts.verbose, v.glyphs))
		}
	}
}

// text writes Markdown the stream released: styled when colour is on, raw
// when it is off (D7).
func (v *view) text(s string) {
	if s == "" {
		return
	}
	if v.caps.Out.Color {
		s = render.Style(render.Tables(s), v.caps.Out)
	}
	v.write(s)
}

// flush writes what the Markdown stream still holds. It runs before every
// non-text event and at the turn's end.
func (v *view) flush() { v.text(v.md.Flush()) }

// line writes s as a line of its own.
func (v *view) line(s string) {
	v.flush()
	v.newline()
	v.write(s + "\n")
}

// newline ends a partly written line.
func (v *view) newline() {
	if !v.atBOL {
		v.write("\n")
	}
}

func (v *view) write(s string) {
	if s == "" || v.err != nil {
		return
	}
	v.stopSpinner()
	v.err = v.out.Result([]byte(s))
	v.wrote = true
	v.atBOL = strings.HasSuffix(s, "\n")
}

// quiet stops the spinner, if it still runs.
func (v *view) quiet() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.stopSpinner()
}

// stopSpinner runs beforeWrite once. v.mu is held.
func (v *view) stopSpinner() {
	if v.beforeWrite != nil {
		v.beforeWrite()
		v.beforeWrite = nil
	}
}

// finish ends the turn's output and prints the status line on stderr. It
// returns the first stdout write error.
func (v *view) finish(res acpclient.Result) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.flush()
	v.newline()
	v.clock.end = v.now()
	v.stopSpinner()
	line := statusLine(v.clock, v.usage, res, v.opts.stats, v.caps.Err, v.glyphs)
	if _, err := io.WriteString(v.out.err, dimmed(line, v.caps.Err.Color)+"\n"); err != nil && v.err == nil {
		v.err = fmt.Errorf("write stderr: %w", err)
	}
	return v.err
}

// statusLine is the line after a turn: render.StatusLine from the last
// usage_update, the --stats figures, and the stop reason unless the turn
// simply ended.
func statusLine(c turnClock, u *acpclient.ContextUsage, res acpclient.Result, stats bool, s term.Stream, g term.Glyphs) string {
	var used, size int64
	var cost *float64
	if u != nil {
		used, size = u.Used, u.Size
		if u.Cost != nil && (u.Cost.Currency == "" || u.Cost.Currency == "USD") {
			cost = &u.Cost.Amount
		}
	}
	line := render.StatusLine(c.end.Sub(c.start), used, size, cost, s, g)
	if stats && !c.first.IsZero() {
		var out int64
		if res.Usage != nil {
			out = int64(res.Usage.OutputTokens)
		}
		line += " " + g.Bullet + " " + render.Stats(c.first.Sub(c.start), out, c.end.Sub(c.first), g)
	}
	if res.StopReason != "" && res.StopReason != "end_turn" {
		line += " " + g.Bullet + " " + res.StopReason
	}
	return line
}

// dimmed dims a line that may already hold SGR sequences, re-entering dim
// after each reset so the whole line stays dim.
func dimmed(line string, on bool) string {
	if !on {
		return line
	}
	return term.Dim.Wrap(strings.ReplaceAll(line, "\x1b[0m", "\x1b[0;2m"), true)
}
