package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/termkey"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/personality"
)

// ACP's option kinds, as acpclient gives them.
const (
	kindAllowOnce    = "allow_once"
	kindAllowAlways  = "allow_always"
	kindRejectOnce   = "reject_once"
	kindRejectAlways = "reject_always"
)

// permissionKeys are 0008-MADR D16's keys, by ACP option kind, in the order
// shown, for a question that is not a tool call's eight options
// (0029-MADR D2).
var permissionKeys = []struct {
	key   string
	kind  string
	label personality.Key
}{
	{"y", kindAllowOnce, personality.PromptAllowOnce},
	{"a", kindAllowAlways, personality.PromptAlways},
	{"n", kindRejectOnce, personality.PromptDeny},
	{"N", kindRejectAlways, personality.PromptNever},
}

// toolAnswers are a tool call's eight options, by acpserver's option ids
// (acpserver/question.go), as 0029-MADR D1 draws them: the Allow row, then
// the Reject row, each once, session, project and everywhere, with the
// words each is echoed in.
var toolAnswers = [2][4]struct {
	id          string
	label, echo personality.Key
}{
	{
		{"allow_once", personality.PromptOnce, personality.PromptAllowOnce},
		{"allow_session", personality.PromptSession, personality.PromptEchoAllowSession},
		{"allow_project", personality.PromptProject, personality.PromptEchoAllowProject},
		{"allow_always", personality.PromptEverywhere, personality.PromptEchoAllowEverywhere},
	},
	{
		{acpserver.RejectOnceID, personality.PromptOnce, personality.PromptEchoRejectOnce},
		{"reject_session", personality.PromptSession, personality.PromptEchoRejectSession},
		{"reject_project", personality.PromptProject, personality.PromptEchoRejectProject},
		{"reject_always", personality.PromptEverywhere, personality.PromptEchoRejectEverywhere},
	},
}

// permissionQuestion is r as go-tui-lib's ask draws it (0029-MADR D1, D2,
// as amended by 0035-PLAN PQ4): why only "once" is offered, as a note; the
// title with "?"; then, for a tool call's eight options, the patterns an
// "always" saves and the Allow and Reject rows; for any other question,
// its options in one row, numbered by name where two share a kind, else by
// D16's keys. [f] reads a reason when reject_once is offered, and [c]
// cancels. Every word is v's (0035-PLAN P14, deviation 4). ask cleans the
// text by safetext's Line rules.
func permissionQuestion(v *personality.Voice, r acpclient.PermissionRequest) ask.Question {
	say := func(k personality.Key) string { return v.Text(k, nil) }
	q := ask.Question{Cancel: ask.Answer{Key: "c", Label: say(personality.PromptCancel)}}
	if r.OnceOnly != "" {
		q.Notes = []string{"(" + oneLine(r.OnceOnly) + ")"}
	}
	if rows, ok := toolRows(say, r); ok {
		q.Title = capitalized(cmp.Or(oneLine(r.FullTitle), oneLine(r.Tool.Title), say(personality.PromptUntitled))) + "?"
		if len(r.Saves) > 0 {
			q.After = []string{v.Text(personality.PromptPatterns, personality.Args{"patterns": oneLine(strings.Join(r.Saves, ", "))})}
		}
		q.Rows = rows
		return q
	}
	q.Title = cmp.Or(oneLine(r.Tool.Title), say(personality.PromptUntitled)) + "?"
	var answers []ask.Answer
	if kindsRepeat(r.Options) {
		for i, o := range r.Options[:min(len(r.Options), 9)] {
			answers = append(answers, ask.Answer{ID: o.ID, Key: strconv.Itoa(i + 1), Label: oneLine(o.Name)})
		}
	} else {
		for _, k := range permissionKeys {
			if i := slices.IndexFunc(r.Options, func(o acpclient.PermissionOption) bool { return o.Kind == k.kind }); i >= 0 {
				answers = append(answers, ask.Answer{ID: r.Options[i].ID, Key: k.key, Label: say(k.label)})
			}
		}
	}
	if offers(r.Options, acpserver.RejectOnceID) {
		answers = append(answers, ask.Answer{ID: acpserver.RejectOnceID, Key: "f", Label: say(personality.PromptDenyWhy), Reason: true})
	}
	q.Rows = []ask.Row{{Answers: answers}}
	return q
}

// toolRows are 0029-MADR D1's rows, when r offers exactly a tool call's
// eight options: Allow and Reject, keyed 1 to 8, the Reject row ending
// with "(hides the tool)" when a "never" would hide it, then [f].
func toolRows(say func(personality.Key) string, r acpclient.PermissionRequest) ([]ask.Row, bool) {
	if len(r.Options) != 8 {
		return nil, false
	}
	rows := make([]ask.Row, 0, 3)
	for i, label := range []personality.Key{personality.PromptAllow, personality.PromptReject} {
		row := ask.Row{Label: say(label)}
		for j, a := range toolAnswers[i] {
			if !offers(r.Options, a.id) {
				return nil, false
			}
			row.Answers = append(row.Answers, ask.Answer{ID: a.id, Key: strconv.Itoa(4*i + j + 1), Label: say(a.label), Echo: say(a.echo)})
		}
		if i == 1 && r.Hides {
			last := &row.Answers[len(row.Answers)-1]
			last.Label += " " + say(personality.PromptHides)
		}
		rows = append(rows, row)
	}
	why := ask.Answer{ID: acpserver.RejectOnceID, Key: "f", Label: say(personality.PromptSayWhy), Echo: say(personality.PromptEchoSayWhy), Reason: true}
	return append(rows, ask.Row{Answers: []ask.Answer{why}}), true
}

// offers reports whether opts holds the option id.
func offers(opts []acpclient.PermissionOption, id string) bool {
	return slices.ContainsFunc(opts, func(o acpclient.PermissionOption) bool { return o.ID == id })
}

// kindsRepeat reports whether two options share a kind.
func kindsRepeat(opts []acpclient.PermissionOption) bool {
	seen := map[string]bool{}
	for _, o := range opts {
		if seen[o.Kind] {
			return true
		}
		seen[o.Kind] = true
	}
	return false
}

// oneLine is s on one line, its runs of spaces one space.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// capitalized is s with its first letter upper-cased (0029-MADR D1).
func capitalized(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// promptWidth is the width a question is drawn at: stderr's, at most 120
// columns (0029-MADR D5), or 120 when stderr is not a terminal.
func promptWidth(o *Output) int {
	if w := o.errWidth(); w > 0 {
		return w
	}
	return editor.MaxWidth
}

// askOn asks q on w, at width, in raw mode, reading the keys with read
// until it is answered, and gives the answer and the rows it leaves in the
// scrollback. A key no answer shows is ignored; c, Ctrl+C and the end of
// input cancel. The region asks the terminal for no cursor report, so no
// answer waits on one. read is given a deadline when the asker or the key
// decoder waits on the clock; a read that cannot take one waits for the
// next key instead.
func askOn(ctx context.Context, q ask.Question, width int, unicodeOK bool, w io.Writer,
	read func(context.Context, []byte) (int, error)) (ask.Response, []string, error) {
	a := ask.NewAsker(q, ask.WithGlyphs(glyph.For(unicodeOK)), ask.WithRegion(inline.NewRegion(inline.WithoutCursorQuery())))
	var resp ask.Response
	done := false
	var werr error
	drain := func() {
		for e := a.NextEvent(); e.Kind != ask.AskNone; e = a.NextEvent() {
			switch e.Kind {
			case ask.AskWrite:
				if _, err := io.WriteString(w, e.Data); err != nil && werr == nil {
					werr = err
				}
			case ask.AskDone:
				resp, done = e.Response, true
			}
		}
	}
	a.Resize(width, time.Now())
	drain()
	var dec termkey.Decoder
	buf := make([]byte, 256)
	for !done {
		rctx, cancel := ctx, context.CancelFunc(func() {})
		if at, ok := a.Timeout(); ok {
			rctx, cancel = context.WithDeadline(ctx, at)
		} else if dec.Pending() {
			rctx, cancel = context.WithTimeout(ctx, termkey.DefaultEscTimeout)
		}
		n, err := read(rctx, buf)
		cancel()
		now := time.Now()
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			for _, in := range dec.Flush(nil) {
				a.HandleInput(in, now)
			}
			a.HandleTimeout(now)
			drain()
			continue
		}
		for _, in := range dec.Decode(buf[:n], nil) {
			a.HandleInput(in, now)
			drain()
			if done {
				break
			}
		}
		if err != nil && !done {
			a.Cancel()
			drain()
		}
	}
	rows, _, _ := a.View(width)
	return resp, rows, werr
}

// askPermission is the line session's answer to session/request_permission:
// the question drawn by go-tui-lib's ask, in the session's voice, on stderr,
// with its keys read in raw mode (0008-MADR D16, as 0029-MADR D1 and D2 lay
// it out). [f] reads a reason up to Enter, and rejects once with it;
// Ctrl+C or the end of input cancels instead (0005-PLAN F6-2b decisions A
// and F).
func (ls *lineSession) askPermission(ctx context.Context, r acpclient.PermissionRequest) acpclient.PermissionAnswer {
	ls.mu.Lock()
	v, keysIn := ls.view, ls.keys
	ls.mu.Unlock()
	if v != nil {
		v.quiet()
	}
	if keysIn == nil {
		return acpclient.PermissionAnswer{}
	}
	restore, err := ls.raw()
	if err != nil {
		return acpclient.PermissionAnswer{}
	}
	out := ls.e.out
	resp, rows, werr := askOn(ctx, permissionQuestion(ls.spoken(), r), promptWidth(out), out.caps.Unicode, out.err, keysIn.ReadContext)
	rerr := restore()
	if rec := out.rec; rec != nil { // the question and its answer, as shown (0030-MADR D4)
		rec.fixed(strings.Join(rows, "\n") + "\n")
	}
	if werr != nil || rerr != nil {
		ls.e.log().WarnContext(ctx, "permission prompt", slog.String("write_error", fmt.Sprint(werr)), slog.String("restore_error", fmt.Sprint(rerr)))
	}
	if resp.Canceled {
		return acpclient.PermissionAnswer{}
	}
	return acpclient.PermissionAnswer{OptionID: resp.Answer.ID, Feedback: resp.Reason}
}
