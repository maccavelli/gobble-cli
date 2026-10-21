package editor

import (
	"slices"
	"strings"
	"testing"
)

// The width is the terminal's, to 120 columns, and 80 when it is unknown
// (0027-MADR D2).
func TestLineWidth(t *testing.T) {
	for in, want := range map[int]int{100: 100, 120: 120, 200: 120, 40: 40, 0: 80, -1: 80} {
		if got := lineWidth(in); got != want {
			t.Errorf("lineWidth(%d) = %d, want %d", in, got, want)
		}
	}
}

// The layout breaks rows only at spaces, so a word keeps its punctuation;
// cuts only a word wider than a row; indents continuation rows by the
// prompt's width; measures wide characters as two columns; and places the
// cursor by the same rows (0027-MADR D2).
func TestLayout(t *testing.T) {
	cases := []struct {
		name     string
		prompt   string
		text     string
		cursor   int // -1: the end of the text
		width    int
		rows     []string
		row, col int
	}{
		{"empty", "> ", "", -1, 20, []string{"> "}, 0, 2},
		{"fits on one row", "> ", "hello world", -1, 20, []string{"> hello world"}, 0, 13},
		{"breaks at the last space that fits", "> ", "the quick brown fox jumps", -1, 16,
			[]string{"> the quick", "  brown fox", "  jumps"}, 2, 7},
		{"a word keeps its comma and period", "> ", "alpha beta, gamma delta.", -1, 13,
			[]string{"> alpha beta,", "  gamma", "  delta."}, 2, 8},
		{"a quoted word stays whole", "> ", `say "hello there" now`, -1, 11,
			[]string{`> say`, `  "hello`, `  there"`, `  now`}, 3, 5},
		{"a word wider than a row is cut", "> ", "abcdefghijklmnop x", -1, 10,
			[]string{"> abcdefgh", "  ijklmnop", "  x"}, 2, 3},
		{"spacing inside a row is kept", "> ", "a  b   c", -1, 20, []string{"> a  b   c"}, 0, 10},
		{"a longer prompt indents by its width", "gobble> ", "one two three four", -1, 19,
			[]string{"gobble> one two", "        three four"}, 1, 18},
		{"wide characters take two columns", "> ", "漢字 漢字漢字 x", -1, 10,
			[]string{"> 漢字", "  漢字漢字", "  x"}, 2, 3},
		{"the cursor inside a later row", "> ", "the quick brown fox", 12, 14,
			[]string{"> the quick", "  brown fox"}, 1, 4},
		{"the cursor at the start", "> ", "the quick brown", 0, 12,
			[]string{"> the quick", "  brown"}, 0, 2},
		{"the cursor at a break's space ends its row", "> ", "the quick brown", 9, 12,
			[]string{"> the quick", "  brown"}, 0, 11},
		{"the cursor after a full last row goes below it", "> ", "abcdefgh", -1, 10,
			[]string{"> abcdefgh", "  "}, 1, 2},
		{"the cursor inside a cut word", "> ", "abcdefghijklmnop", 10, 10,
			[]string{"> abcdefgh", "  ijklmnop"}, 1, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := []rune(tc.text)
			cursor := tc.cursor
			if cursor < 0 {
				cursor = len(text)
			}
			got := layout(tc.prompt, text, cursor, tc.width)
			if !slices.Equal(got.rows, tc.rows) || got.row != tc.row || got.column != tc.col {
				t.Fatalf("layout = %q, cursor (%d, %d); want %q, (%d, %d)", got.rows, got.row, got.column, tc.rows, tc.row, tc.col)
			}
			for _, r := range got.rows {
				if w := displayWidth(r); w > tc.width {
					t.Fatalf("row %q is %d columns, wider than %d", r, w, tc.width)
				}
			}
			// The layout changes the screen, never the text: the rows, less
			// the prompt and the indent, joined at the spaces the breaks
			// took, are the text.
			if !strings.Contains(tc.name, "cut") && !strings.Contains(tc.name, "full last row") {
				var parts []string
				for i, r := range got.rows {
					cut := len(tc.prompt)
					if i > 0 {
						cut = len(strings.Repeat(" ", displayWidth(tc.prompt)))
					}
					parts = append(parts, r[cut:])
				}
				if joined := strings.Join(parts, " "); joined != tc.text {
					t.Fatalf("the rows hold %q, want the text %q", joined, tc.text)
				}
			}
		})
	}
}
