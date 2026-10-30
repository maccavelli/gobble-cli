package editor

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
	"unicode/utf8"

	"github.com/maccavelli/go-tui-lib/v2/inline"
	xterm "golang.org/x/term"
)

// fixedHistory is a History that Add leaves as it is, so x/term's Terminal,
// which adds each line it returns, and the editor, which adds each it
// sends, see the same entries. Index 0 is the most recent.
type fixedHistory []string

func (fixedHistory) Add(string)          {}
func (h fixedHistory) Len() int          { return len(h) }
func (h fixedHistory) At(idx int) string { return h[idx] }

// outcome is what a run of keys leaves: the lines it ended, whether Ctrl+D
// ended the input, and the line being typed with its cursor, in runes.
type outcome struct {
	lines []string
	eof   bool
	text  string
	pos   int
}

// probeKey is sent after a run to x/term's Terminal: it reaches
// AutoCompleteCallback with the line and the cursor, and changes nothing.
const probeKey = 0x1c

// probeConn feeds a Terminal its reads in turn, then io.EOF, and discards
// what it draws. The probe comes in a read of its own: x/term drops the rest
// of a read after Ctrl+D or Ctrl+C ends the input.
type probeConn struct{ reads [][]byte }

func (c *probeConn) Read(p []byte) (int, error) {
	if len(c.reads) == 0 {
		return 0, io.EOF
	}
	n := copy(p, c.reads[0])
	if c.reads[0] = c.reads[0][n:]; len(c.reads[0]) == 0 {
		c.reads = c.reads[1:]
	}
	return n, nil
}

func (c *probeConn) Write(p []byte) (int, error) { return len(p), nil }

// xtermRun is what x/term's Terminal makes of in: the parity table's
// reference, measured rather than written down.
func xtermRun(t *testing.T, h History, in []byte) outcome {
	t.Helper()
	term := xterm.NewTerminal(&probeConn{reads: [][]byte{slices.Clone(in), {probeKey}}}, "> ")
	term.History = h
	var out outcome
	probed := false
	term.AutoCompleteCallback = func(line string, pos int, key rune) (string, int, bool) {
		if key == probeKey {
			out.text, out.pos, probed = line, utf8.RuneCountInString(line[:pos]), true
		}
		return "", 0, false
	}
	for range 64 {
		line, err := term.ReadLine()
		switch {
		case err == nil, errors.Is(err, xterm.ErrPasteIndicator):
			if line != "" {
				out.lines = append(out.lines, line)
			}
		case errors.Is(err, io.EOF) && probed:
			return out
		case errors.Is(err, io.EOF):
			// Ctrl+D: x/term keeps the read that held it and ends every
			// later ReadLine the same way, so the line it holds cannot be
			// probed; pos -1 says so.
			out.eof, out.pos = true, -1
			return out
		default:
			t.Fatalf("x/term: %v", err)
		}
	}
	t.Fatal("x/term: no end to the run")
	return out
}

// hostRun is what the editor makes of in, read from r (in by default) at
// columns: the submissions it returns, and the line it holds, which Tab,
// sent after in, gives Options.Complete. Empty lines are skipped, as
// ReadPrompt skips them.
func hostRun(t *testing.T, h History, in string, columns int, r func(io.Reader) io.Reader) outcome {
	t.Helper()
	var out outcome
	probed := false
	var src io.Reader = strings.NewReader(in + "\t")
	if r != nil {
		src = r(src)
	}
	e := New(inline.NewReader(src), io.Discard, io.Discard, Options{History: h, Width: func() int { return columns },
		Complete: func(line string, pos int) (string, int, bool) {
			out.text, out.pos, probed = line, utf8.RuneCountInString(line[:pos]), true
			return "", 0, false
		}})
	for range 64 {
		s, err := e.ReadPrompt(t.Context())
		switch {
		case err == nil:
			out.lines = append(out.lines, s)
		case errors.Is(err, io.EOF) && probed:
			return out
		case errors.Is(err, io.EOF):
			out.eof, out.pos = true, -1
			return out
		default:
			t.Fatalf("editor: %v", err)
		}
	}
	t.Fatal("editor: no end to the run")
	return out
}

// equalOutcome compares got with want; the line held is compared only when
// want has one (pos ≥ 0).
func equalOutcome(got, want outcome) bool {
	return slices.Equal(got.lines, want.lines) && got.eof == want.eof &&
		(want.pos < 0 || got.text == want.text && got.pos == want.pos)
}

func times(s string, n int) string { return strings.Repeat(s, n) }

const (
	up, down, left, right = "\x1b[A", "\x1b[B", "\x1b[D", "\x1b[C"
	home, end, del        = "\x1b[H", "\x1b[F", "\x1b[3~"
	altLeft, altRight     = "\x1b[1;3D", "\x1b[1;3C"
	pasteOn, pasteOff     = "\x1b[200~", "\x1b[201~"
)

// parityCases are keys x/term reads, from a state reached by typing. rows
// marks the cases that use ↑ or ↓ (and Ctrl+P/N), whose meaning on a line of
// several rows 0027-MADR D4 changes: they are compared on one row only.
// differs names the cases where lineedit's behaviour is not x/term's, by
// 0035-PLAN PQ7 (a) or the host's own keys, and the test that holds the
// new behaviour instead.
var parityCases = []struct {
	name    string
	in      string
	rows    bool
	differs string
}{
	{"typing", "abc", false, ""},
	{"typing in the middle", "hello world" + times(left, 6) + "big ", false, ""},
	{"left at the start stays", "ab" + times(left, 5), false, ""},
	{"right at the end stays", "ab" + left + times(right, 4), false, ""},
	{"Ctrl+B and Ctrl+F", "abcd\x02\x02\x02\x06", false, ""},
	{"Backspace", "hello" + left + left + "\x7f", false, ""},
	{"Backspace at the start", "ab" + home + "\x7f", false, ""},
	{"Ctrl+H", "hello\x08\x08", false, ""},
	{"Home", "hello world" + left + home, false, ""},
	{"End", "hello world" + home + right + end, false, ""},
	{"Ctrl+A", "hello world" + left + "\x01", false, ""},
	{"Ctrl+E", "hello world" + "\x01\x06\x05", false, ""},
	{"Alt+Left twice", "one two  three" + altLeft + altLeft, false, ""},
	{"Alt+Left over spaces", "one two   " + altLeft, false, ""},
	{"Alt+Left at the start", "one" + home + altLeft, false, ""},
	{"Alt+Right twice", "one two  three" + home + altRight + altRight, false, "TestWordKeys: to a word's end"},
	{"Alt+Right at the end", "one two" + altRight, false, ""},
	{"Alt+Right from a space", "one   two" + home + times(right, 4) + altRight, false, "TestWordKeys: to a word's end"},
	{"Delete in the middle", "hello" + left + left + del, false, ""},
	{"Delete at the end", "hello" + del, false, ""},
	{"Ctrl+D in the middle", "hello" + home + "\x04", false, ""},
	{"Ctrl+D at the end of a line", "hello\x04", false, ""},
	{"Ctrl+K", "hello world" + times(left, 5) + "\x0b", false, ""},
	{"Ctrl+U", "hello world" + times(left, 5) + "\x15", false, ""},
	{"Ctrl+W", "one two three\x17", false, ""},
	{"Ctrl+W over spaces", "one two   \x17", false, ""},
	{"Ctrl+W in the middle", "one two three" + times(left, 3) + "\x17", false, ""},
	{"Ctrl+T in the middle", "abcd" + left + left + "\x14", false, ""},
	{"Ctrl+T at the end", "abcd\x14", false, ""},
	{"Ctrl+T at the start", "abcd" + home + "\x14", false, ""},
	{"Ctrl+L", "hello" + left + "\x0c", false, ""},
	{"Tab without completion", "ab\t", false, ""},
	{"unknown sequences", "ab\x1b[Z\x1b[15~\x1b[1;2A", false, ""},
	{"wide and accented", "héllo 世界" + left + "x", false, ""},
	{"Enter in the middle", "hello world" + times(left, 3) + "\r" + "next", false, ""},
	{"CR LF is one Enter", "ab\r\ncd", false, ""},
	{"LF is Enter", "ab\ncd", false, ""},
	{"Ctrl+D on an empty line", "\x04", false, ""},
	{"Ctrl+D after a line", "ab\r\x04", false, ""},
	{"Ctrl+C", "ab\x03", false, "TestReadPrompt and keys_test.go: the host's Ctrl+C"},
	{"a paste of two lines", pasteOn + "one\ntwo" + pasteOff, false, "TestReadPrompt: one buffer"},
	{"a paste then Enter", pasteOn + "one\ntwo" + pasteOff + "\r", false, "TestReadPrompt: one buffer"},
	{"a paste of three lines", pasteOn + "one\ntwo\nthree" + pasteOff + "\r", false, "TestReadPrompt: one buffer"},
	{"a paste into a typed line", "say " + pasteOn + "one\ntwo" + pasteOff + "\r", false, "TestReadPrompt: one buffer"},
	{"control keys inside a paste", pasteOn + "a\tb\x03c\x04d\x7f" + pasteOff, false, "TestReadPrompt: cleaned"},
	{"the line limit", times("a", 4100), false, ""},
	{"Up recalls", up, true, ""},
	{"Up twice", up + up, true, ""},
	{"Up past the oldest", times(up, 5), true, ""},
	{"Up then Down", "draft" + up + up + down, true, ""},
	{"Down back to the draft", "draft" + up + down, true, ""},
	{"Down with nothing recalled", "draft" + down, true, ""},
	{"Ctrl+P and Ctrl+N", "draft\x10\x10\x0e", true, ""},
	{"Up then edit", up + "\x7f\x7fX", true, ""},
	{"Up after Enter", "a\r" + up, true, ""},
}

var parityHistory = fixedHistory{"newest entry", "middle", "oldest one"}

// Every key x/term reads does what it does in x/term's Terminal, through
// the host and termkey's decoding, on one row and on a line of several
// rows (0027-PLAN P3, step 2; 0035-PLAN P17 step 4), but where differs
// names the behaviour PQ7 (a) took instead.
func TestKeysMatchXterm(t *testing.T) {
	for _, c := range parityCases {
		if c.differs != "" {
			continue
		}
		want := xtermRun(t, parityHistory, []byte(c.in))
		if got := hostRun(t, parityHistory, c.in, 200, nil); !equalOutcome(got, want) {
			t.Errorf("%s, one row: got %+v, want %+v", c.name, got, want)
		}
		if c.rows {
			continue
		}
		if got := hostRun(t, parityHistory, c.in, 12, nil); !equalOutcome(got, want) {
			t.Errorf("%s, rows of 12: got %+v, want %+v", c.name, got, want)
		}
	}
}

// A key's bytes may arrive over several reads.
func TestKeysSplitAcrossReads(t *testing.T) {
	in := "héllo" + left + altLeft + del + "\r" + "ne" + up + down + "xt"
	want := xtermRun(t, parityHistory, []byte(in))
	if got := hostRun(t, parityHistory, in, 12, iotest.OneByteReader); !equalOutcome(got, want) {
		t.Errorf("a byte at a time: got %+v, want %+v", got, want)
	}
}

// Option+←/→ in macOS Terminal (ESC b, ESC f) and Ctrl+←/→ move by word,
// as Alt+←/→ do (0027-MADR D4, amended by 0027-PLAN P3's deviation 1). By
// 0035-PLAN PQ7 (a) the words are lineedit's, letters and digits:
// rightward to a word's end, and over punctuation, where gobble's went by
// spaces to the next word's start.
func TestWordKeys(t *testing.T) {
	for _, s := range []string{"one two  three", "one two  three" + times(left, 3), "one" + home, "  a  "} {
		for key, same := range map[string]string{
			"\x1bb": altLeft, "\x1bf": altRight, "\x1b[1;5D": altLeft, "\x1b[1;5C": altRight,
		} {
			for _, n := range []int{1, 2} {
				for _, columns := range []int{200, 12} {
					want := hostRun(t, nil, s+times(same, n), columns, nil)
					if got := hostRun(t, nil, s+times(key, n), columns, nil); !equalOutcome(got, want) {
						t.Errorf("%q ×%d after %q at %d columns: got %+v, want %+v", key, n, s, columns, got, want)
					}
				}
			}
		}
	}
	for _, tc := range []struct {
		name, in string
		pos      int
	}{
		{"alt+→ to the end of the word, not the next's start", "one two" + home + altRight, 3},
		{"alt+→ from a space to the next word's end", "one   two" + home + times(right, 4) + altRight, 9},
		{"alt+← stops at punctuation", "path/to-file" + altLeft, 8},
		{"alt+→ stops at punctuation", "path/to-file" + home + altRight, 4},
	} {
		if got := hostRun(t, nil, tc.in, 200, nil); got.pos != tc.pos {
			t.Errorf("%s: cursor at %d in %q, want %d", tc.name, got.pos, got.text, tc.pos)
		}
	}
}

// ↑ and ↓ move between the line's rows, to the column nearest the cursor's
// without passing it, and recall history from its first and last row
// (0027-MADR D4, Q1). Ctrl+P and Ctrl+N are ↑ and ↓.
func TestRowKeys(t *testing.T) {
	// At 12 columns, "the quick brown fox jumps" is "> the quick" (runes 0-9),
	// "  brown fox" (10-19) and "  jumps" (20-25).
	const fox = "the quick brown fox jumps"
	cases := []struct {
		name    string
		hist    fixedHistory
		in      string
		columns int
		text    string
		pos     int
	}{
		{"up from the last row", nil, fox + up, 12, fox, 15},
		{"up twice", nil, fox + up + up, 12, fox, 5},
		{"up from the first row recalls", fixedHistory{"old"}, fox + up + up + up, 12, "old", 3},
		{"Ctrl+P is up", nil, fox + "\x10\x10", 12, fox, 5},
		{"down", nil, fox + up + up + down, 12, fox, 15},
		{"down to the last row", nil, fox + up + up + down + down, 12, fox, 25},
		{"Ctrl+N is down", nil, fox + "\x10\x10\x0e", 12, fox, 15},
		{"down from the last row with nothing recalled", nil, fox + down, 12, fox, 25},
		{"down keeps the column", nil, fox + "\x01" + times(right, 9) + down, 12, fox, 19},
		{"a longer row above", nil, fox + up + times(right, 4) + down, 12, fox, 25},
		{"up to a row's end", nil, fox + up + times(right, 4) + up, 12, fox, 9},
		{"a recalled entry is edited by row", fixedHistory{fox, "older"}, "x" + up + up, 12, fox, 15},
		{"down from a recalled entry's last row", fixedHistory{fox}, "x" + up + down, 12, "x", 1},
		{"down from a recalled entry's first row moves", fixedHistory{fox}, "x" + up + up + up + down, 12, fox, 15},
		// At 10 columns, "世界世界世界 ab" is "> 世界世界" (runes 0-3, columns 2-8) and
		// "  世界 ab" (4-9).
		{"wide characters, up", nil, "世界世界世界 ab" + up, 10, "世界世界世界 ab", 3},
		{"wide characters, between", nil, "世界世界世界 ab" + times(left, 2) + up, 10, "世界世界世界 ab", 2},
		// At 7 columns, "abcde" fills its row, and the cursor at its end
		// stands on the next.
		{"from the cursor's own row past a full one", nil, "abcde" + up, 7, "abcde", 0},
		{"down onto it", nil, "abcde" + up + down, 7, "abcde", 5},
	}
	for _, c := range cases {
		var h History
		if c.hist != nil {
			h = c.hist
		}
		got := hostRun(t, h, c.in, c.columns, nil)
		if got.text != c.text || got.pos != c.pos {
			t.Errorf("%s: text %q pos %d, want %q pos %d", c.name, got.text, got.pos, c.text, c.pos)
		}
	}
}
