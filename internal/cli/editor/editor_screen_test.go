package editor

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
)

// report is the owner's report that asked for this editor (0027-MADR), a
// paragraph of words and punctuation to type.
const report = "The prompt is being cut and wrapped at around 80 chars, including in the middle of words. " +
	"My terminal is much wider than 80 chars. The line wrap needs to be dynamic, based on the available width " +
	"of the user's terminal to a max of 120 chars, ensuring not to cut words or punctuation such as a trailing " +
	"comma when wrapping the prompt text."

// onScreen runs a new Editor on in, drawing on s, until an error, and
// returns the submissions.
func onScreen(t *testing.T, s *inlinetest.Screen, in io.Reader, opts Options) []string {
	t.Helper()
	e := New(inline.NewReader(in), s, io.Discard, opts)
	var subs []string
	for {
		text, err := e.ReadPrompt(t.Context())
		if err != nil {
			return subs
		}
		subs = append(subs, text)
	}
}

// trimmedLines is the screen's rows without their trailing spaces.
func trimmedLines(s *inlinetest.Screen) []string {
	lines := s.Lines()
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return lines
}

// lineRows are the screen's rows from the first that starts with prompt,
// through the last before an empty one or the next prompt, and the row
// after them. A run's last read draws a new prompt, so after a line sent
// that row holds the bare prompt.
func lineRows(t *testing.T, s *inlinetest.Screen, prompt string) (rows []string, next int) {
	t.Helper()
	lines := trimmedLines(s)
	bare := strings.TrimSpace(prompt)
	for i, l := range lines {
		if strings.HasPrefix(l, prompt) {
			end := i + 1
			for end < len(lines) && lines[end] != "" && !strings.HasPrefix(lines[end], bare) {
				end++
			}
			return lines[i:end], end
		}
	}
	t.Fatalf("no row starts with %q: %q", prompt, lines)
	return nil, 0
}

// checkNextPrompt fails unless the row after a sent line holds the next
// prompt, alone, with the cursor after it: Enter left the line as drawn and
// moved below it.
func checkNextPrompt(t *testing.T, s *inlinetest.Screen, next int) {
	t.Helper()
	lines := trimmedLines(s)
	if next >= len(lines) || lines[next] != ">" || s.Row != next || s.Col != 2 {
		t.Errorf("after the sent line, the cursor at (%d, %d), want the next prompt's (%d, 2) alone; screen %q",
			s.Row, s.Col, next, lines)
	}
}

// checkWholeWords fails unless rows hold text with every word whole: the
// rows, prompt and indent taken off, joined by single spaces, are the text;
// none is wider than columns.
func checkWholeWords(t *testing.T, rows []string, text string, columns int) {
	t.Helper()
	var parts []string
	for i, r := range rows {
		if w := ansi.StringWidth(r); w > columns {
			t.Errorf("row %d is %d columns, more than %d: %q", i, w, columns, r)
		}
		if i == 0 {
			r = strings.TrimPrefix(r, "> ")
		} else if !strings.HasPrefix(r, "  ") {
			t.Errorf("row %d is not indented under the text: %q", i, r)
		}
		parts = append(parts, strings.TrimSpace(r))
	}
	if got := strings.Join(parts, " "); got != text {
		t.Errorf("a word was split, or the text changed: rows %q", rows)
	}
}

// A long prompt typed at 60, 100 and 200 columns is drawn in rows of the
// terminal's width, to 120 columns, every word whole with its punctuation,
// and stays so once sent; the text sent is as typed (0027-MADR D2, D3).
func TestTypingKeepsWordsWhole(t *testing.T) {
	for _, width := range []int{60, 100, 200} {
		s := newScreen(width, 20)
		subs := onScreen(t, s, strings.NewReader(report+"\r"), Options{Width: func() int { return width }})
		if !slices.Equal(subs, []string{report}) {
			t.Fatalf("at %d columns, sent %q", width, subs)
		}
		columns := min(width, MaxWidth)
		rows, next := lineRows(t, s, "> ")
		checkWholeWords(t, rows, report, columns)
		if want, _, _ := laidOut("> ", report, columns); !slices.Equal(rows, want) {
			t.Errorf("at %d columns, the screen's rows %q, want lineedit's %q", width, rows, want)
		}
		checkNextPrompt(t, s, next)
	}
}

// chunkReader returns its chunks one read at a time, calling before(i)
// before it returns chunk i.
type chunkReader struct {
	chunks []string
	before func(i int)
	next   int
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if c.next == len(c.chunks) {
		return 0, io.EOF
	}
	if c.before != nil {
		c.before(c.next)
	}
	n := copy(p, c.chunks[c.next])
	c.next++
	return n, nil
}

// The width is read at each key: narrowed between keys, the line is drawn
// again at the new width, nothing of the wider drawing left (0027-MADR D3).
func TestResizeBetweenKeys(t *testing.T) {
	s := newScreen(100, 20)
	width := 100
	half := len(report) / 2
	in := &chunkReader{chunks: []string{report[:half], report[half:], "\r"}, before: func(i int) {
		if i == 1 {
			width = 60
		}
	}}
	subs := onScreen(t, s, in, Options{Width: func() int { return width }})
	if !slices.Equal(subs, []string{report}) {
		t.Fatalf("sent %q", subs)
	}
	rows, next := lineRows(t, s, "> ")
	checkWholeWords(t, rows, report, 60)
	checkNextPrompt(t, s, next)
	var shown []string
	for _, l := range trimmedLines(s) {
		if l != "" {
			shown = append(shown, l)
		}
	}
	if !slices.Equal(shown, append(rows, ">")) {
		t.Errorf("the screen holds more than the line and the next prompt: %q", trimmedLines(s))
	}
}

// Editing in the middle of a line of several rows draws it again whole:
// the screen holds the layout of the edited text. By PQ7 (a), alt+→ goes
// to the end of this or the next word, lineedit's, so the insertion
// starts with its space; gobble's went to the next word's start
// ("INSERTED " before).
func TestEditingMidLineOnScreen(t *testing.T) {
	s := newScreen(30, 12)
	typed := "alpha beta gamma delta epsilon zeta eta theta"
	in := typed + "\x01" + "\x1b[1;3C\x1b[1;3C" + " INSERTED" + "\x1b[F" + " iota" + "\r"
	want := "alpha beta INSERTED gamma delta epsilon zeta eta theta iota"
	subs := onScreen(t, s, strings.NewReader(in), Options{Width: func() int { return 30 }})
	if !slices.Equal(subs, []string{want}) {
		t.Fatalf("sent %q, want %q", subs, want)
	}
	rows, next := lineRows(t, s, "> ")
	if l, _, _ := laidOut("> ", want, 30); !slices.Equal(rows, l) {
		t.Errorf("screen rows %q, want %q", rows, l)
	}
	checkNextPrompt(t, s, next)
}

// A line taller than the screen is drawn from the rows around the cursor:
// at the line's start, its first row is on screen with the cursor on it
// (0027-MADR D3).
func TestTallLineShowsTheCursorRow(t *testing.T) {
	s := newScreen(20, 5)
	in := strings.NewReader(strings.Repeat("word ", 30) + "\x01")
	onScreen(t, s, in, Options{Width: func() int { return 20 }, Height: func() int { return 5 }})
	if got := s.Lines()[s.Row]; !strings.HasPrefix(got, "> word") || s.Col != 2 {
		t.Errorf("the cursor at (%d, %d) on %q, want at column 2 of the line's first row; screen %q",
			s.Row, s.Col, got, s.Lines())
	}
}

// Ctrl+D on an empty line and a second Ctrl+C leave the line as Enter
// does, the cursor at the start of the row below it, so what is written
// next, the goodbye, has a row of its own (0027-MADR D3, as amended for
// P5).
func TestEndOfInputLeavesTheLine(t *testing.T) {
	for _, tc := range []struct{ name, keys string }{
		{"Ctrl+D", "\x04"},
		{"a second Ctrl+C", "\x03\x03"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newScreen(20, 6)
			if subs := onScreen(t, s, strings.NewReader(tc.keys), Options{Width: func() int { return 20 }}); len(subs) > 0 {
				t.Fatalf("sent %q", subs)
			}
			lines := trimmedLines(s)
			last := -1
			for i, l := range lines {
				if strings.HasPrefix(l, ">") {
					last = i
				}
			}
			if last < 0 || lines[last] != ">" || s.Row != last+1 || s.Col != 0 {
				t.Fatalf("screen %q, cursor (%d, %d); want the bare prompt drawn, the cursor at the start of the row below it",
					lines, s.Row, s.Col)
			}
		})
	}
}

// Ctrl+L clears the screen and draws the line again at its top.
func TestClearScreenOnScreen(t *testing.T) {
	s := newScreen(20, 6)
	if _, err := s.Write([]byte("old output\r\nmore\r\n")); err != nil {
		t.Fatal(err)
	}
	subs := onScreen(t, s, strings.NewReader("ab\x0cc\r"), Options{Width: func() int { return 20 }})
	if !slices.Equal(subs, []string{"abc"}) {
		t.Fatalf("sent %q", subs)
	}
	if got := trimmedLines(s); !slices.Equal(got, []string{"> abc", ">", "", "", "", ""}) || s.Row != 1 {
		t.Errorf("screen %q, cursor row %d; want only \"> abc\" and the next prompt, the cursor on it", got, s.Row)
	}
}

// The Ctrl+C notice and the paste notice go above the prompt, which is
// drawn again below them, nothing of it left above.
func TestNoticesAboveThePrompt(t *testing.T) {
	s := newScreen(30, 6)
	e := New(inline.NewReader(strings.NewReader("ab\x1b[200~x\ny\x1b[201~\x03\x03")), s, s, Options{Width: func() int { return 30 }})
	if _, err := e.ReadPrompt(t.Context()); err == nil {
		t.Fatal("ReadPrompt returned a submission")
	}
	want := []string{"… 2 pasted lines", "Press Ctrl+C again to exit", ">", "", "", ""}
	if got := trimmedLines(s); !slices.Equal(got, want) {
		t.Fatalf("screen %q, want %q", got, want)
	}
}
