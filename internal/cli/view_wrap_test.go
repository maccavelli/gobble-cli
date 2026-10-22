package cli

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/internal/cli/term/termtest"
)

// ownersReply is the reply the owner saw cut mid-word in WezTerm at 177
// columns (0028-MADR).
const ownersReply = "Hello again. If you have a specific task—whether it’s writing code, reviewing documentation, " +
	"debugging, or anything else—please describe it in your own words. I’ll be precise, courteous, and to the point."

// wrapTurn runs a turn's updates through a view writing to a terminal,
// colour as given, replies wrapped to width, and returns stdout.
func wrapTurn(t *testing.T, color bool, width int, opts display, updates ...acpclient.Update) string {
	t.Helper()
	var out, errw strings.Builder
	s := term.Stream{TTY: true, Color: color, Width: width}
	o := &Output{out: &out, err: &errw, caps: term.Caps{In: true, Out: s, Err: s, Unicode: true}}
	opts.width = func() int { return width }
	v := newView(o, opts, stepClock())
	for _, u := range updates {
		v.on(u)
	}
	if err := v.finish(acpclient.Result{StopReason: "end_turn"}); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// textChunks streams s as text updates of n runes each.
func textChunks(s string, n int) []acpclient.Update {
	var us []acpclient.Update
	for s != "" {
		end := 0
		for k := 0; k < n && end < len(s); k++ {
			_, size := utf8.DecodeRuneInString(s[end:])
			end += size
		}
		us = append(us, acpclient.Update{Kind: acpclient.KindAgentText, Text: s[:end]})
		s = s[end:]
	}
	return us
}

// checkWrapped fails unless the screen's rows are at most width columns and
// hold text's words whole, in order.
func checkWrapped(t *testing.T, rows []string, text string, width int) {
	t.Helper()
	var words []string
	for _, r := range rows {
		if n := utf8.RuneCountInString(r); n > width {
			t.Errorf("row of %d columns, more than %d: %q", n, width, r)
		}
		words = append(words, strings.Fields(r)...)
	}
	if want := strings.Fields(text); !slices.Equal(words, want) {
		t.Errorf("words on screen %q, want %q", words, want)
	}
}

// escapeSeq is a control sequence, which shows nothing.
var escapeSeq = regexp.MustCompile(`\x1b\[[0-9;?]*[@-~]`)

// shownWrapped reports whether out shows text's words, in order, across
// the rows replies wrap into at width, and that no row is wider than width
// unless it is one word, which wrapping leaves whole (0028-MADR D3).
func shownWrapped(out, text string, width int) bool {
	for row := range strings.SplitSeq(escapeSeq.ReplaceAllString(out, ""), "\n") {
		row = row[strings.LastIndex(row, "\r")+1:] // what the row shows after the editor's moves
		if utf8.RuneCountInString(row) > width && len(strings.Fields(row)) > 1 {
			return false
		}
	}
	return strings.Contains(strings.Join(strings.Fields(out), " "), strings.Join(strings.Fields(text), " "))
}

// screenRows are the non-empty rows out shows on a terminal columns wide.
func screenRows(out string, columns int) []string {
	s := termtest.New(columns, 60)
	s.Cooked = true
	_, _ = s.Write([]byte(out))
	var rows []string
	for _, l := range s.Lines() {
		if l != "" {
			rows = append(rows, l)
		}
	}
	return rows
}

// A reply streamed in small chunks keeps its words whole at the width it
// wraps to, styled or not, on a terminal wider than that width
// (0028-MADR D1–D3).
func TestReplyWrapsOnScreen(t *testing.T) {
	for _, tc := range []struct{ terminal, width int }{{60, 60}, {100, 100}, {177, 120}} {
		for _, color := range []bool{false, true} {
			out := wrapTurn(t, color, tc.width, display{}, textChunks(ownersReply+"\n", 7)...)
			checkWrapped(t, screenRows(out, tc.terminal), ownersReply, tc.width)
		}
	}
}

// Code, tables and headings are written as they came; a list item wraps
// under its text (0028-MADR D3, D4).
func TestReplyKeepsCodeAndTables(t *testing.T) {
	code := "    return strings.Join(append(rows, \"a long line of code that is wider than forty columns\"), sep)"
	reply := "## A heading that is longer than forty columns, kept whole\n\n" +
		"```go\n" + code + "\n```\n\n" +
		"| name | value |\n|---|---|\n| a cell that is long enough to pass forty columns | b |\n\n" +
		"- a list item with enough words in it to need a second row here\n"
	out := wrapTurn(t, false, 40, display{}, textChunks(reply, 5)...)
	for _, line := range []string{
		"## A heading that is longer than forty columns, kept whole",
		code,
		"| a cell that is long enough to pass forty columns | b |",
		"- a list item with enough words in it to",
		"  need a second row here",
	} {
		if !slices.Contains(strings.Split(out, "\n"), line) {
			t.Errorf("no line %q in\n%s", line, out)
		}
	}
}

// Text still held when a tool call arrives is written, wrapped, before the
// tool's header (0028-MADR D5).
func TestReplyFlushedBeforeTool(t *testing.T) {
	out := wrapTurn(t, false, 20, display{},
		acpclient.Update{Kind: acpclient.KindAgentText, Text: "alpha beta gamma delta epsilon"},
		acpclient.Update{Kind: acpclient.KindToolCall, Tool: acpclient.ToolCall{ID: "t1", Title: "read notes.md", Status: "completed"}})
	if !strings.HasPrefix(out, "alpha beta gamma\ndelta epsilon\n") || !strings.Contains(out, "read notes.md") {
		t.Errorf("stdout %q, want the text wrapped, then the tool's header", out)
	}
}

// Thinking text, shown dimmed, wraps as a reply does (0028-MADR D3).
func TestThinkingWraps(t *testing.T) {
	out := wrapTurn(t, true, 40, display{showThinking: true},
		acpclient.Update{Kind: acpclient.KindThought, Text: ownersReply})
	checkWrapped(t, screenRows(out, 100), ownersReply, 40)
}

// A reply still arriving when the terminal narrows wraps at the new width
// from the next text on (0030-MADR D2).
func TestReplyAfterResizeWrapsAtNewWidth(t *testing.T) {
	var out, errw strings.Builder
	s := term.Stream{TTY: true, Width: 100}
	o := &Output{out: &out, err: &errw, caps: term.Caps{In: true, Out: s, Err: s, Unicode: true}}
	width := 100
	v := newView(o, display{width: func() int { return width }}, stepClock())
	half := strings.Repeat("alpha beta gamma delta ", 8) + "\n"
	v.on(acpclient.Update{Kind: acpclient.KindAgentText, Text: half})
	width = 40
	v.on(acpclient.Update{Kind: acpclient.KindAgentText, Text: half})
	if err := v.finish(acpclient.Result{StopReason: "end_turn"}); err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	k, words := 0, 0 // the first paragraph's rows end where its words do
	for k < len(rows) && words < len(strings.Fields(half)) {
		words += len(strings.Fields(rows[k]))
		k++
	}
	if k == len(rows) {
		t.Fatalf("no rows after the first paragraph: %q", rows)
	}
	for i, r := range rows {
		limit := 40
		if i < k {
			limit = 100
		}
		if len(r) > limit {
			t.Fatalf("row %d is %d columns, more than %d: %q", i, len(r), limit, rows)
		}
	}
}

// Thinking text still held when the reply starts is written before the
// reply, not after it (0028-MADR D5).
func TestThinkingFlushedBeforeReply(t *testing.T) {
	out := wrapTurn(t, false, 40, display{showThinking: true},
		acpclient.Update{Kind: acpclient.KindThought, Text: "weighing the options here"},
		acpclient.Update{Kind: acpclient.KindAgentText, Text: "Answer.\n"})
	if !strings.HasPrefix(out, "weighing the options here\nAnswer.\n") {
		t.Errorf("stdout %q, want the thinking, then the reply", out)
	}
}

// The line session wraps a reply on a terminal, to the terminal's width up
// to 120 columns, and writes it as it came off a terminal (0028-MADR D1,
// D2; Q1, Q2).
func TestLineSessionWrapsReplies(t *testing.T) {
	long := strings.Repeat("word ", 40) + "end."
	for _, tc := range []struct {
		name  string
		tty   bool
		width int
		want  int // the widest row; 0 is the reply on one line
	}{
		{"a terminal 80 wide", true, 80, 80},
		{"a terminal 200 wide", true, 200, 120},
		{"not a terminal", false, 80, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk(long + "\n")}}}}
			h := newLineHarness(t, agent)
			h.e.out.caps.Out.TTY, h.e.out.caps.Out.Width = tc.tty, tc.width
			h.start(Prompt{})
			h.press("hello\r")
			waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "end.") })
			h.press("\x04")
			if err := h.wait(); err != nil {
				t.Fatal(err)
			}
			out := h.stdout.String()
			if tc.want == 0 {
				if !strings.Contains(out, long+"\n") {
					t.Errorf("the reply is not as it came: %q", out)
				}
				return
			}
			var rows []string
			for l := range strings.SplitSeq(out, "\n") {
				if strings.Contains(l, "word") {
					rows = append(rows, l)
				}
			}
			widest := 0
			for _, r := range rows {
				widest = max(widest, len(r))
			}
			if len(rows) < 2 || widest > tc.want || widest < tc.want-5 {
				t.Errorf("reply rows %q: widest %d, want at most %d and close to it", rows, widest, tc.want)
			}
		})
	}
}
