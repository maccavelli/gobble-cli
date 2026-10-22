package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/render"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/personality"
)

// display is how the line session shows a turn (0008-MADR D7).
type display struct {
	showThinking, verbose, stats bool
	home                         string
	// voice speaks the view's own lines; nil is boring (0022-PLAN P1).
	voice *personality.Voice
	// width is the width replies and thinking wrap to, read as they
	// arrive; nil, or 0, wraps nothing (0028-MADR D1, D2).
	width func() int
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

	mu    sync.Mutex
	md    render.Stream
	wrap  render.Wrap // the reply's rows (0028-MADR)
	think render.Wrap // the thinking text's rows
	// reprintWidth is the terminal's width for a reprint (0030-MADR D3).
	reprintWidth func() int
	// afterTurn hands a reprint still due when the turn ends to the
	// editor.
	afterTurn func()
	// reply and thinking are the transcript's pieces still growing.
	reply, thinking *piece
	// settle times the wait for resizes to settle; reprintDue says one
	// settled and its reprint waits for a line's end; ended, the turn's.
	settle     *time.Timer
	reprintDue bool
	ended      bool
	clock      turnClock
	tools      map[string]*toolState
	usage      *acpclient.ContextUsage
	wrote      bool // anything is on stdout
	atBOL      bool // the cursor is at the start of a line
	thought    bool // the last text written was thinking
	err        error
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

// setVoice is the session's new voice, for the rest of the turn
// (0022-PLAN P3).
func (v *view) setVoice(voice *personality.Voice) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.opts.voice = voice
}

// on draws one update.
func (v *view) on(u acpclient.Update) {
	v.mu.Lock()
	defer v.mu.Unlock()
	defer v.tryReprint()
	switch u.Kind {
	case acpclient.KindAgentText:
		v.clock.token(v.now())
		if u.Block != "" {
			v.line("[" + u.Block + "]")
			return
		}
		if v.thought {
			v.thought = false
			v.show(v.think.Flush())
			v.thinking = nil
			v.newline()
		}
		v.text(v.md.Push(term.Sanitize(u.Text)))
	case acpclient.KindThought:
		v.clock.token(v.now())
		if !v.opts.showThinking || u.Text == "" {
			return
		}
		v.flushReply() // not the thinking: its next chunk continues the row
		v.thought = true
		v.keep(&v.thinking, v.thinkingAt, term.Sanitize(u.Text))
		s := term.Dim.Wrap(term.Sanitize(u.Text), v.caps.Out.Color)
		if w := v.wrapWidth(); w > 0 {
			s = v.think.Push(s, w)
		}
		v.show(s)
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
	case acpclient.KindRetry:
		if line, ok := retryLine(v.opts.voice, u.Retry); ok {
			v.out.Notef("%s", line)
		}
	case acpclient.KindCompaction:
		if line, ok := compactionLine(v.opts.voice, u.Compaction); ok {
			v.out.Notef("%s", line)
		}
	case acpclient.KindOther:
	}
}

// The placeholders the view's lines fill, by the boring catalog's names
// (0022-PLAN P1).
const (
	argAttempt = "attempt"
	argMax     = "max"
	argCause   = "cause"
	argReason  = "reason"
	argError   = "error"
)

// retryLine is what the CLI shows on stderr for a retry wait's start, in
// voice's words: "waiting for the network", or "retrying: <cause> (attempt
// <n> of <m>)"; an end shows nothing, since a failure ends the turn with
// its error (0005-PLAN F5-1 decision 5).
func retryLine(voice *personality.Voice, r acpclient.Retry) (string, bool) {
	attempts := personality.Args{argAttempt: strconv.Itoa(r.Attempt), argMax: strconv.Itoa(r.MaxAttempts)}
	switch {
	case r.Phase != "start":
		return "", false
	case r.Cause == "offline":
		return voice.Text(personality.RetryOffline, nil), true
	case r.Cause == "empty_reply":
		return voice.Text(personality.RetryEmptyReply, attempts), true
	}
	attempts[argCause] = strings.ReplaceAll(r.Cause, "_", " ")
	return voice.Text(personality.RetryAttempt, attempts), true
}

// compactionLine is what the CLI shows on stderr for an automatic
// compaction: "compacting the conversation (<reason>)" at its start, and
// "compaction failed: <error>" at a failed end, with what it paused. A
// manual one shows nothing, since /compact's reply says so (0005-PLAN F5-2
// decision 9).
func compactionLine(voice *personality.Voice, c acpclient.Compaction) (string, bool) {
	switch {
	case c.Reason == "manual":
		return "", false
	case c.Phase == "start":
		return voice.Text(personality.CompactionStart, personality.Args{argReason: c.Reason}), true
	case c.Error == "":
		return "", false
	}
	key := personality.CompactionFailed
	switch c.Paused {
	case "turn":
		key = personality.CompactionFailedTurn
	case "session":
		key = personality.CompactionFailedSession
	}
	return voice.Text(key, personality.Args{argError: c.Error}), true
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
			v.write(render.Diff(render.ToolOutput(diff, v.opts.verbose, v.glyphs, v.opts.voice), v.caps.Out))
		case c.Terminal != "":
			v.line("terminal " + term.Sanitize(c.Terminal))
		default:
			v.flush()
			v.newline()
			v.write(render.ToolOutput(c.Text, v.opts.verbose, v.glyphs, v.opts.voice))
		}
	}
}

// text writes Markdown the stream released: styled when colour is on, raw
// when it is off (D7), then wrapped to the terminal's width, when there is
// one (0028-MADR D1, D7 as amended).
func (v *view) text(s string) {
	v.keep(&v.reply, v.replyAt, s)
	if s != "" && v.caps.Out.Color {
		s = render.Style(render.Tables(s), v.caps.Out)
	}
	if w := v.wrapWidth(); w > 0 {
		s = v.wrap.Push(s, w)
	}
	v.show(s)
}

// wrapWidth is the width replies wrap to now; 0 wraps nothing.
func (v *view) wrapWidth() int {
	if v.opts.width == nil {
		return 0
	}
	return v.opts.width()
}

// flush writes what the Markdown stream and the wrappers still hold. It
// runs before every non-text event and at the turn's end.
func (v *view) flush() {
	v.flushReply()
	v.show(v.think.Flush())
	v.thinking = nil
}

// flushReply writes what the Markdown stream and the reply's wrapper hold.
func (v *view) flushReply() {
	v.text(v.md.Flush())
	v.show(v.wrap.Flush())
	v.reply = nil
}

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

func (v *view) write(s string) { v.put(s, true) }

// show writes rows the transcript keeps in another form: a reply's and
// thinking's, kept as their source (0030-MADR D4).
func (v *view) show(s string) { v.put(s, false) }

func (v *view) put(s string, keep bool) {
	if s == "" || v.err != nil {
		return
	}
	v.stopSpinner()
	if keep {
		v.err = v.out.Result([]byte(s))
	} else {
		v.err = v.out.emit([]byte(s))
	}
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
	v.ended = true
	if v.settle != nil {
		v.settle.Stop()
	}
	if v.reprintDue && v.afterTurn != nil {
		v.afterTurn() // the editor reprints at the prompt
	}
	line := statusLine(v.clock, v.usage, res, v.opts.stats, v.caps.Err, v.glyphs, v.opts.voice)
	if v.out.rec != nil {
		v.out.rec.fixed(dimmed(line, v.caps.Err.Color) + "\n")
	}
	if _, err := io.WriteString(v.out.err, dimmed(line, v.caps.Err.Color)+"\n"); err != nil && v.err == nil {
		v.err = fmt.Errorf("write stderr: %w", err)
	}
	return v.err
}

// statusLine is the line after a turn: render.StatusLine from the last
// usage_update, the --stats figures, and the stop reason, in voice's
// words, unless the turn simply ended.
func statusLine(c turnClock, u *acpclient.ContextUsage, res acpclient.Result, stats bool, s term.Stream, g term.Glyphs,
	voice *personality.Voice) string {
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
		line += " " + g.Bullet + " " + stopText(voice, res.StopReason)
	}
	return line
}

// stopKeys are the stop reasons the turn's last line names, by key; any
// other is shown as the agent gave it.
var stopKeys = map[string]personality.Key{
	"max_tokens":        personality.TurnStopMaxTokens,
	"max_turn_requests": personality.TurnStopMaxRequests,
	"refusal":           personality.TurnStopRefusal,
	"cancelled":         personality.TurnStopCancelled,
}

func stopText(voice *personality.Voice, reason string) string {
	if key, ok := stopKeys[reason]; ok {
		return voice.Text(key, nil)
	}
	return reason
}

// dimmed dims a line that may already hold SGR sequences, re-entering dim
// after each reset so the whole line stays dim.
func dimmed(line string, on bool) string {
	if !on {
		return line
	}
	return term.Dim.Wrap(strings.ReplaceAll(line, "\x1b[0m", "\x1b[0;2m"), true)
}

// reprintSettle is how long after the last resize the conversation is
// reprinted (0030-MADR D3).
const reprintSettle = 150 * time.Millisecond

// resized notices a resize while the turn runs: once resizes settle,
// the conversation is reprinted at the next line's end (0030-MADR D3).
func (v *view) resized() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.ended {
		if v.afterTurn != nil {
			v.afterTurn()
		}
		return
	}
	if v.settle != nil {
		v.settle.Stop()
	}
	v.settle = time.AfterFunc(reprintSettle, func() {
		v.mu.Lock()
		defer v.mu.Unlock()
		v.reprintDue = true
		v.tryReprint()
	})
}

// tryReprint makes a due reprint when the view is at a line's start with
// nothing held back, so that everything shown so far can be drawn again
// and the reply goes on from there, no word lost or doubled. v.mu is held.
func (v *view) tryReprint() {
	if !v.reprintDue || v.ended || v.out.rec == nil || !v.atBOL || !v.wrap.Idle() || !v.think.Idle() {
		return
	}
	v.reprintDue = false
	width := 0
	if v.reprintWidth != nil {
		width = v.reprintWidth()
	}
	if width <= 0 {
		return
	}
	text := reprintText(v.out.rec, width, v.caps.Out.Color)
	v.stopSpinner()
	if err := v.out.emit([]byte("\x1b[?2026h\x1b[2J\x1b[3J\x1b[H" + text + "\x1b[?2026l")); err != nil && v.err == nil {
		v.err = err
	}
	// The wrappers are idle at a line's start, and a Stream releases a code
	// fence only once it is closed, so they go on as they are.
	v.atBOL = text == "" || strings.HasSuffix(text, "\n")
}

// keep adds s to the transcript piece *p, opening it, drawn by at.
func (v *view) keep(p **piece, at func(src string, width int) string, s string) {
	if v.out.rec == nil || s == "" {
		return
	}
	if *p == nil {
		*p = v.out.rec.open(at)
	}
	(*p).add(v.out.rec, s)
}

// replyAt draws a reply's Markdown at a terminal's width as the view
// does: styled when colour is on, then wrapped, to 120 columns.
func (v *view) replyAt(src string, width int) string {
	if v.caps.Out.Color {
		src = render.Style(render.Tables(src), v.caps.Out)
	}
	var w render.Wrap
	return w.Push(src, min(width, 120)) + w.Flush()
}

// thinkingAt draws thinking text at a terminal's width, dimmed.
func (v *view) thinkingAt(src string, width int) string {
	var w render.Wrap
	return w.Push(term.Dim.Wrap(src, v.caps.Out.Color), min(width, 120)) + w.Flush()
}
