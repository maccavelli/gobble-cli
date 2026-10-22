package cli

import (
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/internal/cli/term/termtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// reprintReply is a reply with prose, a list and code.
const reprintReply = "Hello again. If you have a specific task—whether it’s writing code, reviewing documentation, " +
	"debugging, or anything else—please describe it in your own words. I’ll be precise, courteous, and to the point.\n\n" +
	"- A list item with enough words in it to need a second row at sixty columns, continued under its text.\n\n" +
	"```go\nfunc main() { fmt.Println(\"a long line of code wider than sixty columns, kept as it is\") }\n```\n"

// afterLastClear is what out shows after its last clear of the screen, on a
// terminal columns wide.
func afterLastClear(out string, columns int) []string {
	if i := strings.LastIndex(out, "\x1b[2J"); i >= 0 {
		out = out[i:]
	}
	s := termtest.New(columns, 400)
	s.Cooked = true
	_, _ = s.Write([]byte(out))
	return s.Lines()
}

// inOrder reports whether words appear among lines' words, in order, one
// after another: none of them cut.
func inOrder(lines []string, words []string) bool {
	var all []string
	for _, l := range lines {
		all = append(all, strings.Fields(l)...)
	}
	for i := 0; i+len(words) <= len(all); i++ {
		if slices.Equal(all[i:i+len(words)], words) {
			return true
		}
	}
	return false
}

// A resize at the prompt, once settled, clears the screen and reprints the
// conversation at the new width: the reply's prose with every word whole,
// its list continued, /help laid out again; then the line (0030-MADR D3, D4).
func TestLineSessionReprintsOnResize(t *testing.T) {
	for _, p := range provider.KeyProviders() { // the no-credential warning, a message, is always shown
		t.Setenv(provider.CredentialVar(p), "")
	}
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk(reprintReply)}}}}
	h := newLineHarness(t, agent)
	h.e.out.caps.Out.TTY, h.e.out.caps.Err.TTY = true, true
	var width atomic.Int64
	width.Store(100)
	h.termWidth = func() int { return int(width.Load()) }
	log := &modeLog{h: h}
	h.e.out.out, h.e.out.err = log, log
	h.start(Prompt{})
	h.press("hi\r")
	// The reply is wrapped, so its words are looked for, not its text.
	waitUntil(t, func() bool { return strings.Contains(strings.Join(strings.Fields(log.String()), " "), "to the point.") })
	h.press("/help\r")
	waitUntil(t, func() bool { return strings.Contains(log.String(), "Commands") })
	time.Sleep(200 * time.Millisecond) // the next prompt drawn
	n := len(log.String())
	width.Store(60)
	h.resize <- struct{}{}
	waitUntil(t, func() bool { return strings.Contains(log.String()[n:], "\x1b[6n") })
	h.press("\x1b[5;3R")
	waitUntil(t, func() bool { return strings.Contains(log.String()[n:], "\x1b[3J") })
	time.Sleep(100 * time.Millisecond)
	lines := afterLastClear(log.String(), 60)
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	prose := strings.Fields(strings.SplitN(reprintReply, "\n", 2)[0])
	if !inOrder(lines, prose) {
		t.Errorf("the reply's words are not whole at 60 after the reprint: %q", lines)
	}
	if n := len(slices.DeleteFunc(slices.Clone(lines), func(l string) bool { return !strings.HasPrefix(l, "Hello again.") })); n != 1 {
		t.Errorf("the reply is reprinted %d times, want once: %q", n, lines)
	}
	item := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "- A list item") })
	if item < 0 || item+1 >= len(lines) || !strings.HasPrefix(lines[item+1], "  ") {
		t.Errorf("the list item is not continued under its text: %q", lines)
	}
	warning := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "Warning: no model credential") })
	if warning < 0 || !strings.HasPrefix(lines[warning+1], "  ") {
		t.Errorf("the warning, a message, is not reprinted wrapped at 60: %q", lines)
	}
	if !slices.Contains(lines, "> hi") {
		t.Errorf("the prompt sent is not reprinted: %q", lines)
	}
	if !slices.ContainsFunc(lines, func(l string) bool { return statusRow.MatchString(l) }) {
		t.Errorf("the turn's status line is not reprinted: %q", lines)
	}
	start := slices.Index(lines, "Commands")
	if start < 0 {
		t.Fatalf("no /help after the reprint: %q", lines)
	}
	for _, l := range lines[start:] {
		if strings.HasPrefix(l, ">") {
			break
		}
		if len([]rune(l)) > 59 {
			t.Errorf("a /help row of %d columns, more than 59: %q", len([]rune(l)), l)
		}
	}
}

// A reply still arriving when resizes settle is reprinted at its next line
// end, when nothing is held back, and goes on at the new width with no word
// twice (0030-MADR D3).
func TestViewReprintsAtTheNextLineEnd(t *testing.T) {
	var out, errw strings.Builder
	s := term.Stream{TTY: true, Width: 100}
	o := &Output{out: &out, err: &errw, caps: term.Caps{In: true, Out: s, Err: s, Unicode: true}, rec: &transcript{}}
	var width atomic.Int64
	width.Store(100)
	v := newView(o, display{width: func() int { return int(width.Load()) }}, stepClock())
	v.reprintWidth = func() int { return int(width.Load()) }
	first := strings.Repeat("alpha beta gamma delta ", 6)
	v.on(acpclient.Update{Kind: acpclient.KindAgentText, Text: first})
	width.Store(40)
	v.resized()
	time.Sleep(300 * time.Millisecond)
	if strings.Contains(out.String(), "\x1b[3J") {
		t.Fatal("reprinted in the middle of a line")
	}
	v.on(acpclient.Update{Kind: acpclient.KindAgentText, Text: "end of the first.\n"})
	if !strings.Contains(out.String(), "\x1b[3J") {
		t.Fatal("not reprinted at the line's end")
	}
	second := strings.Repeat("epsilon zeta eta theta ", 5) + "end.\n"
	v.on(acpclient.Update{Kind: acpclient.KindAgentText, Text: second})
	if err := v.finish(acpclient.Result{StopReason: "end_turn"}); err != nil {
		t.Fatal(err)
	}
	lines := afterLastClear(out.String(), 40)
	want := strings.Fields(first + "end of the first. " + second)
	if !inOrder(lines, want) {
		t.Errorf("the reply after the reprint is not its words, whole, once: %q", lines)
	}
	if n := strings.Count(strings.Join(lines, " "), "end of the first."); n != 1 {
		t.Errorf("the reply's first paragraph shown %d times, want once: %q", n, lines)
	}
	for _, l := range lines {
		if len(l) > 40 {
			t.Errorf("a row of %d columns, more than 40: %q", len(l), l)
		}
	}
}

// statusRow is a turn's status line as the harness shows it: its time.
var statusRow = regexp.MustCompile(`^\d+(\.\d+)?m?s\b`)

// A reprint still due when the turn ends, the reply ending mid-line, is
// handed to the editor, which reprints at the prompt (0030-MADR D3).
func TestLateReprintGoesToTheEditor(t *testing.T) {
	var out, errw strings.Builder
	s := term.Stream{TTY: true, Width: 100}
	o := &Output{out: &out, err: &errw, caps: term.Caps{In: true, Out: s, Err: s, Unicode: true}, rec: &transcript{}}
	v := newView(o, display{width: func() int { return 100 }}, stepClock())
	v.reprintWidth = func() int { return 100 }
	var handed atomic.Int64
	v.afterTurn = func() { handed.Add(1) }
	v.on(acpclient.Update{Kind: acpclient.KindAgentText, Text: "a reply that ends in the middle of a line"})
	v.resized()
	time.Sleep(300 * time.Millisecond)
	if err := v.finish(acpclient.Result{StopReason: "end_turn"}); err != nil {
		t.Fatal(err)
	}
	if n := handed.Load(); n != 1 {
		t.Fatalf("handed to the editor %d times, want 1", n)
	}
}

// The chat command keeps what is shown from its start in the line session
// on a terminal, so a warning written before the session opens, such as the
// model's, is kept; not in print mode, and not off a terminal (0030-MADR D4,
// as amended).
func TestKeepShownFromTheStart(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mode      Mode
		tty, kept bool
	}{
		{"the line session on a terminal", ModeLine, true, true},
		{"print mode", ModePrint, true, false},
		{"off a terminal", ModeLine, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errw strings.Builder
			s := term.Stream{TTY: tc.tty, Width: 80}
			o := &Output{out: &out, err: &errw, caps: term.Caps{In: tc.tty, Out: s, Err: s}}
			keepShown(o, tc.mode)
			o.Warnf("Model %q not found for provider %q. Using custom model id.", "stub-model", "kilo")
			if kept := o.rec != nil && strings.Contains(reprintText(o.rec, 80, false), `Warning: Model "stub-model"`); kept != tc.kept {
				t.Fatalf("kept %v, want %v", kept, tc.kept)
			}
		})
	}
}

// A warning written before the session opened, and a log record shown
// during it, are reprinted with the conversation, wrapped at the new width
// (0030-MADR D4, as amended).
func TestLineSessionReprintsWhatCameBefore(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("reply-1\n")}}}}
	h := newLineHarness(t, agent)
	h.e.out.caps.Out.TTY, h.e.out.caps.Err.TTY = true, true
	h.e.out.caps.Err.Width = 100
	var width atomic.Int64
	width.Store(100)
	h.termWidth = func() int { return int(width.Load()) }
	log := &modeLog{h: h}
	h.e.out.out, h.e.out.err = log, log
	h.e.logs = &lazyLog{level: slog.LevelInfo, out: h.e.out, stderr: log}
	keepShown(h.e.out, ModeLine)
	h.e.out.Warnf("Model %q not found for provider %q. Using custom model id.", "stub-model", "kilo")
	h.start(Prompt{})
	h.press("hi\r")
	waitUntil(t, func() bool { return strings.Contains(log.String(), "reply-1") })
	h.e.log().WarnContext(t.Context(), "model listing failed; offering the curated static list",
		slog.String("provider", "kilo"), slog.String("kind", "other"))
	time.Sleep(200 * time.Millisecond) // the next prompt drawn
	n := len(log.String())
	width.Store(60)
	h.resize <- struct{}{}
	waitUntil(t, func() bool { return strings.Contains(log.String()[n:], "\x1b[6n") })
	h.press("\x1b[5;3R")
	waitUntil(t, func() bool { return strings.Contains(log.String()[n:], "\x1b[3J") })
	time.Sleep(100 * time.Millisecond)
	lines := afterLastClear(log.String(), 60)
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	for _, first := range []string{`Warning: Model "stub-model" not found`, "Warning: model listing failed;"} {
		i := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, first) })
		if i < 0 || i+1 >= len(lines) || !strings.HasPrefix(lines[i+1], "  ") {
			t.Errorf("%q is not reprinted, wrapped at 60: %q", first, lines)
			continue
		}
		for _, l := range lines[i : i+2] {
			if len([]rune(l)) > 60 {
				t.Errorf("a row of %d columns, more than 60: %q", len([]rune(l)), l)
			}
		}
	}
}

// A log record is shown wrapped at stderr's width when it is logged, not
// when the log opened, and kept to be drawn again (0030-MADR D4, as amended).
func TestLogRecordWrapsAtTheWidthNow(t *testing.T) {
	t.Setenv("GOBBLE_HOME", t.TempDir())
	var out, errw strings.Builder
	o := &Output{out: &out, err: &errw, caps: term.Caps{In: true, Out: term.Stream{TTY: true, Width: 100},
		Err: term.Stream{TTY: true, Width: 100}}, rec: &transcript{}}
	l := &lazyLog{level: slog.LevelInfo, out: o, stderr: &errw}
	t.Cleanup(func() {
		if err := l.close(); err != nil {
			t.Error(err)
		}
	})
	const msg = "model listing failed; offering the curated static list"
	l.get().WarnContext(t.Context(), msg, slog.String("provider", "kilo"))
	if got := errw.String(); got != "Warning: "+msg+" (provider: kilo)\n" {
		t.Fatalf("at 100 columns: %q", got)
	}
	errw.Reset()
	o.caps.Err.Width = 40
	l.get().WarnContext(t.Context(), msg, slog.String("provider", "kilo"))
	rows := strings.Split(strings.TrimSuffix(errw.String(), "\n"), "\n")
	if len(rows) < 2 {
		t.Fatalf("not wrapped at 40: %q", errw.String())
	}
	for _, r := range rows {
		if len(r) > 39 {
			t.Errorf("a row of %d columns, more than 39: %q", len(r), r)
		}
	}
	if s := reprintText(o.rec, 50, false); strings.Count(s, "Warning: model listing failed;") != 2 {
		t.Fatalf("the records are not kept: %q", s)
	}
}

// Off a terminal nothing is kept (0030-MADR D6).
func TestNoTranscriptOffATerminal(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("reply-1\n")}}}}
	h := newLineHarness(t, agent)
	h.noResize = true
	h.start(Prompt{})
	h.press("hi\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "reply-1") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if h.ls.tr != nil || h.e.out.rec != nil {
		t.Fatal("a transcript was kept off a terminal")
	}
}
