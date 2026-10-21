package editor

import (
	"slices"
	"testing"
)

// drawOn is a renderer drawing on s, below what s already shows.
func drawOn(s *screen) *renderer { return &renderer{out: s} }

func mustDraw(t *testing.T, r *renderer, l screenLayout, height int) {
	t.Helper()
	if err := r.draw(l, height); err != nil {
		t.Fatal(err)
	}
}

func checkScreen(t *testing.T, s *screen, lines []string, row, col int) {
	t.Helper()
	if got := s.lines(); !slices.Equal(got, lines) || s.row != row || s.col != col {
		t.Fatalf("screen %q, cursor (%d, %d); want %q, (%d, %d)", got, s.row, s.col, lines, row, col)
	}
}

// A line is drawn where the cursor stands, below what was written before,
// and drawn again in place: shorter, longer, or the same, nothing of the
// last drawing is left (0027-MADR D3).
func TestDrawRedraws(t *testing.T) {
	s := newScreen(20, 6)
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
	s := newScreen(10, 5)
	r := drawOn(s)
	mustDraw(t, r, layout("> ", []rune("abcdefgh ijklmnop xy"), 20, 10), 5)
	checkScreen(t, s, []string{"> abcdefgh", "  ijklmnop", "  xy", "", ""}, 2, 4)
	mustDraw(t, r, layout("> ", []rune("abcdefgh"), 8, 10), 5)
	checkScreen(t, s, []string{"> abcdefgh", "", "", "", ""}, 1, 2)
}

// A line taller than the screen is drawn from the rows around the cursor,
// so the cursor's row is on the screen (0027-MADR D3).
func TestDrawTallLine(t *testing.T) {
	s := newScreen(12, 4)
	r := drawOn(s)
	rows := []string{"> r0", "  r1", "  r2", "  r3", "  r4", "  r5", "  r6", "  r7"}
	mustDraw(t, r, screenLayout{rows: rows, row: 7, column: 4}, 4)
	checkScreen(t, s, []string{"  r4", "  r5", "  r6", "  r7"}, 3, 4)
	mustDraw(t, r, screenLayout{rows: rows, row: 6, column: 2}, 4)
	if got := s.lines(); !slices.Contains(got, "  r6") || s.lines()[s.row] != "  r6" {
		t.Fatalf("the cursor's row r6 is not under the cursor: %q, cursor row %d", got, s.row)
	}
	// The cursor at the line's start: the first rows are drawn, not the
	// last ones scrolled over them.
	mustDraw(t, r, screenLayout{rows: rows, row: 0, column: 2}, 4)
	checkScreen(t, s, []string{"> r0", "  r1", "  r2", "  r3"}, 0, 2)
}

// After Enter the line stays as drawn and the cursor goes below it; the
// next line is drawn there (0027-MADR D3).
func TestDrawFinish(t *testing.T) {
	s := newScreen(20, 6)
	r := drawOn(s)
	mustDraw(t, r, screenLayout{rows: []string{"> the quick", "  brown fox"}, row: 0, column: 4}, 6)
	if err := r.finish(); err != nil {
		t.Fatal(err)
	}
	checkScreen(t, s, []string{"> the quick", "  brown fox", "", "", "", ""}, 2, 0)
	mustDraw(t, r, screenLayout{rows: []string{"> next"}, row: 0, column: 6}, 6)
	checkScreen(t, s, []string{"> the quick", "  brown fox", "> next", "", "", ""}, 2, 6)
}
