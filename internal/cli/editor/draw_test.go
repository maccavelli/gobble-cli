package editor

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/internal/cli/term/termtest"
)

// drawOn is a renderer drawing on s, below what s already shows.
func drawOn(s *termtest.Screen) *renderer { return &renderer{out: s} }

func mustDraw(t *testing.T, r *renderer, l screenLayout, height int) {
	t.Helper()
	if err := r.draw(l, height); err != nil {
		t.Fatal(err)
	}
}

func checkScreen(t *testing.T, s *termtest.Screen, lines []string, row, col int) {
	t.Helper()
	if got := s.Lines(); !slices.Equal(got, lines) || s.Row != row || s.Col != col {
		t.Fatalf("screen %q, cursor (%d, %d); want %q, (%d, %d)", got, s.Row, s.Col, lines, row, col)
	}
}

// A line is drawn where the cursor stands, below what was written before,
// and drawn again in place: shorter, longer, or the same, nothing of the
// last drawing is left (0027-MADR D3).
func TestDrawRedraws(t *testing.T) {
	s := termtest.New(20, 6)
	if _, err := s.Write([]byte("header\r\n")); err != nil {
		t.Fatal(err)
	}
	r := drawOn(s)
	mustDraw(t, r, screenLayout{rows: []string{"> the quick", "  brown fox", "  jumps"}, row: 2, column: 7}, 6)
	checkScreen(t, s, []string{"header", "> the quick", "  brown fox", "  jumps", "", ""}, 3, 7)
	mustDraw(t, r, screenLayout{rows: []string{"> the"}, row: 0, column: 5}, 6)
	checkScreen(t, s, []string{"header", "> the", "", "", "", ""}, 1, 5)
	mustDraw(t, r, screenLayout{rows: []string{"> one two", "  three", "  four"}, row: 1, column: 4}, 6)
	checkScreen(t, s, []string{"header", "> one two", "  three", "  four", "", ""}, 2, 4)
}

// Rows the full width of the screen leave no stray line between them, and
// the cursor lands where the layout puts it (0027-MADR D3).
func TestDrawFullRows(t *testing.T) {
	s := termtest.New(10, 5)
	r := drawOn(s)
	mustDraw(t, r, layout("> ", []rune("abcdefgh ijklmnop xy"), 20, 10), 5)
	checkScreen(t, s, []string{"> abcdefgh", "  ijklmnop", "  xy", "", ""}, 2, 4)
	mustDraw(t, r, layout("> ", []rune("abcdefgh"), 8, 10), 5)
	checkScreen(t, s, []string{"> abcdefgh", "", "", "", ""}, 1, 2)
}

// A line taller than the screen is drawn from the rows around the cursor,
// so the cursor's row is on the screen (0027-MADR D3).
func TestDrawTallLine(t *testing.T) {
	s := termtest.New(12, 4)
	r := drawOn(s)
	rows := []string{"> r0", "  r1", "  r2", "  r3", "  r4", "  r5", "  r6", "  r7"}
	mustDraw(t, r, screenLayout{rows: rows, row: 7, column: 4}, 4)
	checkScreen(t, s, []string{"  r4", "  r5", "  r6", "  r7"}, 3, 4)
	mustDraw(t, r, screenLayout{rows: rows, row: 6, column: 2}, 4)
	if got := s.Lines(); !slices.Contains(got, "  r6") || s.Lines()[s.Row] != "  r6" {
		t.Fatalf("the cursor's row r6 is not under the cursor: %q, cursor row %d", got, s.Row)
	}
	// The cursor at the line's start: the first rows are drawn, not the
	// last ones scrolled over them.
	mustDraw(t, r, screenLayout{rows: rows, row: 0, column: 2}, 4)
	checkScreen(t, s, []string{"> r0", "  r1", "  r2", "  r3"}, 0, 2)
}

// tokens splits a drawing into what a terminal takes in one step: a control
// sequence whole, or one rune.
func tokens(frame string) []string {
	var out []string
	for i := 0; i < len(frame); {
		if strings.HasPrefix(frame[i:], "\x1b[") {
			j := i + 2
			for j < len(frame) && (frame[j] < 0x40 || frame[j] > 0x7e) {
				j++
			}
			out = append(out, frame[i:min(j+1, len(frame))])
			i = j + 1
			continue
		}
		_, n := utf8.DecodeRuneInString(frame[i:])
		out = append(out, frame[i:i+n])
		i += n
	}
	return out
}

// printed counts the runes a drawing writes on the screen.
func printed(frame string) int {
	n := 0
	for _, tok := range tokens(frame) {
		if !strings.HasPrefix(tok, "\x1b") && tok >= " " {
			n++
		}
	}
	return n
}

func commonPrefix(a, b string) string {
	ra, rb := []rune(a), []rune(b)
	i := 0
	for i < len(ra) && i < len(rb) && ra[i] == rb[i] {
		i++
	}
	return string(ra[:i])
}

// A terminal may paint at any point in a drawing. At every point, each row
// keeps the text its old and new versions share, and the cursor is hidden
// unless it stands where the drawing found it or where it leaves it; the
// line never shows blank, and the cursor never visibly jumps (0027-MADR D3,
// deviation 4).
func TestDrawNeverBlanks(t *testing.T) {
	const width, height = 30, 10
	s := termtest.New(width, height)
	if _, err := s.Write([]byte("header\r\n")); err != nil {
		t.Fatal(err)
	}
	const top = 1
	var frame strings.Builder
	r := &renderer{out: &frame}
	steps := []struct {
		text   string
		cursor int
	}{
		{"", 0}, {"h", 1}, {"hello", 5}, {"hello there", 11}, {"hello there", 3}, {"hello thee", 9},
		{"hello there my friend, the line is long", 39}, {"hello there my friend, the line is longer", 41},
		{"hello there my friendly, the line is longer", 22}, {"hello there my friend, the", 26},
		{"hello there", 11}, {"", 0},
	}
	var old screenLayout
	for i, st := range steps {
		l := layout("> ", []rune(st.text), st.cursor, width)
		frame.Reset()
		mustDraw(t, r, l, height)
		fromRow, fromCol := s.Row, s.Col
		toRow, toCol := top+l.row, l.column
		for _, tok := range tokens(frame.String()) {
			if _, err := s.Write([]byte(tok)); err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				continue
			}
			lines := s.Lines()
			for k := range max(len(old.rows), len(l.rows)) {
				var was, now string
				if k < len(old.rows) {
					was = old.rows[k]
				}
				if k < len(l.rows) {
					now = l.rows[k]
				}
				if keep := strings.TrimRight(commonPrefix(was, now), " "); !strings.HasPrefix(lines[top+k], keep) {
					t.Fatalf("step %d (%q): after %q, row %d shows %q, losing %q", i, st.text, tok, k, lines[top+k], keep)
				}
			}
			at := [2]int{s.Row, s.Col}
			if !s.Hidden && at != [2]int{fromRow, fromCol} && at != [2]int{toRow, toCol} {
				t.Fatalf("step %d (%q): after %q, the cursor shows at %v, neither %v nor %v", i, st.text, tok, at,
					[2]int{fromRow, fromCol}, [2]int{toRow, toCol})
			}
		}
		want := []string{"header"}
		for _, row := range l.rows {
			want = append(want, strings.TrimRight(row, " "))
		}
		want = append(want, make([]string, height-len(want))...)
		checkScreen(t, s, want, toRow, toCol)
		if s.Hidden {
			t.Fatalf("step %d (%q): the cursor is left hidden", i, st.text)
		}
		old = l
	}
}

// A drawing writes only what changed: a key typed at the end of the line
// writes that key, a cursor move writes no text; each drawing that writes
// is one synchronized update, which a terminal paints whole (0027-MADR D3,
// deviation 4).
func TestDrawWritesOnlyWhatChanged(t *testing.T) {
	var frame strings.Builder
	r := &renderer{out: &frame}
	mustDraw(t, r, layout("> ", []rune("hello"), 5, 40), 10)
	for _, st := range []struct {
		name   string
		text   string
		cursor int
		runes  int
	}{
		{"a key typed at the end", "hellox", 6, 1},
		{"the cursor moved", "hellox", 2, 0},
		{"a key typed in the middle", "heZllox", 3, 5},
		{"nothing changed", "heZllox", 3, 0},
	} {
		frame.Reset()
		mustDraw(t, r, layout("> ", []rune(st.text), st.cursor, 40), 10)
		got := frame.String()
		if n := printed(got); n != st.runes {
			t.Fatalf("%s: wrote %d runes (%q), want %d", st.name, n, got, st.runes)
		}
		if st.runes > 0 && (!strings.HasPrefix(got, syncOn) || !strings.HasSuffix(got, syncOff)) {
			t.Fatalf("%s: %q is not one synchronized update", st.name, got)
		}
	}
}

// After Enter the line stays as drawn and the cursor goes below it; the
// next line is drawn there (0027-MADR D3).
func TestDrawFinish(t *testing.T) {
	s := termtest.New(20, 6)
	r := drawOn(s)
	mustDraw(t, r, screenLayout{rows: []string{"> the quick", "  brown fox"}, row: 0, column: 4}, 6)
	if err := r.finish(); err != nil {
		t.Fatal(err)
	}
	checkScreen(t, s, []string{"> the quick", "  brown fox", "", "", "", ""}, 2, 0)
	mustDraw(t, r, screenLayout{rows: []string{"> next"}, row: 0, column: 6}, 6)
	checkScreen(t, s, []string{"> the quick", "  brown fox", "> next", "", "", ""}, 2, 6)
}
