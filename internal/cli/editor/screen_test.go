package editor

import (
	"slices"
	"strings"
	"testing"
)

// screen is a terminal's screen, enough of one to check what the editor
// draws (0027-MADR D6): a grid of cells with a cursor; printable runes,
// a wide one taking two cells; \r; line feed, scrolling at the bottom;
// cursor up, down, forward and back; erase to the end of the screen and of
// the line. A rune written in the last column leaves a wrap pending, as
// terminals do: the next rune goes to the next row's start, and \r, a line
// feed or a move cancels it.
type screen struct {
	width, height int
	cells         [][]rune
	row, col      int
	pending       bool
}

// wideTail marks the second cell of a wide rune.
const wideTail = -1

func newScreen(width, height int) *screen {
	s := &screen{width: width, height: height}
	for range height {
		s.cells = append(s.cells, s.blank())
	}
	return s
}

func (s *screen) blank() []rune {
	r := make([]rune, s.width)
	for i := range r {
		r[i] = ' '
	}
	return r
}

func (s *screen) Write(p []byte) (int, error) {
	runes := []rune(string(p))
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == '\r':
			s.col, s.pending = 0, false
		case r == '\n':
			s.pending = false
			s.down()
		case r == 0x1b && i+1 < len(runes) && runes[i+1] == '[':
			j := i + 2
			for j < len(runes) && runes[j] >= '0' && runes[j] <= '9' {
				j++
			}
			n := 1
			if j > i+2 {
				n = 0
				for _, d := range runes[i+2 : j] {
					n = n*10 + int(d-'0')
				}
			}
			if j < len(runes) {
				s.csi(runes[j], n, j > i+2)
			}
			i = j
		case r < 0x20:
		default:
			s.put(r)
		}
	}
	return len(p), nil
}

// csi applies one control sequence: count n, given says it was written.
func (s *screen) csi(final rune, n int, given bool) {
	s.pending = false
	switch final {
	case 'A':
		s.row = max(s.row-n, 0)
	case 'B':
		s.row = min(s.row+n, s.height-1)
	case 'C':
		s.col = min(s.col+n, s.width-1)
	case 'D':
		s.col = max(s.col-n, 0)
	case 'J':
		if !given || n == 0 {
			for c := s.col; c < s.width; c++ {
				s.cells[s.row][c] = ' '
			}
			for r := s.row + 1; r < s.height; r++ {
				s.cells[r] = s.blank()
			}
		}
	case 'K':
		if !given || n == 0 {
			for c := s.col; c < s.width; c++ {
				s.cells[s.row][c] = ' '
			}
		}
	}
}

// down moves the cursor a row down, scrolling at the bottom.
func (s *screen) down() {
	if s.row == s.height-1 {
		s.cells = append(s.cells[1:], s.blank())
		return
	}
	s.row++
}

// put writes r at the cursor, taking its pending wrap first.
func (s *screen) put(r rune) {
	w := runeWidth(r)
	if s.pending || s.col+w > s.width {
		s.col, s.pending = 0, false
		s.down()
	}
	s.cells[s.row][s.col] = r
	if w == 2 {
		s.cells[s.row][s.col+1] = wideTail
	}
	if s.col+w >= s.width {
		s.col, s.pending = s.width-1, true
		return
	}
	s.col += w
}

// lines are the screen's rows, trailing spaces dropped.
func (s *screen) lines() []string {
	out := make([]string, s.height)
	for i, row := range s.cells {
		var b strings.Builder
		for _, r := range row {
			if r != wideTail {
				b.WriteRune(r)
			}
		}
		out[i] = strings.TrimRight(b.String(), " ")
	}
	return out
}

// The screen model, checked before anything is checked with it.
func TestScreenModel(t *testing.T) {
	type want struct {
		lines    []string
		row, col int
	}
	cases := []struct {
		name          string
		width, height int
		in            string
		want          want
	}{
		{"text and a line end", 10, 3, "ab\r\ncd", want{[]string{"ab", "cd", ""}, 1, 2}},
		{"\\r goes to the row's start", 10, 2, "abc\rX", want{[]string{"Xbc", ""}, 0, 1}},
		{"a line feed keeps the column", 10, 3, "ab\nc", want{[]string{"ab", "  c", ""}, 1, 3}},
		{"moves up, down, forward, back", 10, 4, "a\r\nb\r\nc\x1b[2A\x1b[3C\x1b[1DX\x1b[1BY", want{[]string{"a  X", "b   Y", "c", ""}, 1, 5}},
		{"a move with no count moves one", 10, 3, "ab\r\ncd\x1b[AX", want{[]string{"abX", "cd", ""}, 0, 3}},
		{"erase to the end of the screen", 10, 3, "abcd\r\nefgh\r\nijkl\x1b[2A\r\x1b[2C\x1b[J", want{[]string{"ab", "", ""}, 0, 2}},
		{"erase to the end of the line", 10, 2, "abcdef\r\x1b[3C\x1b[K", want{[]string{"abc", ""}, 0, 3}},
		{"a full row leaves a wrap pending", 4, 3, "abcdX", want{[]string{"abcd", "X", ""}, 1, 1}},
		{"\\r cancels the pending wrap", 4, 3, "abcd\rX", want{[]string{"Xbcd", "", ""}, 0, 1}},
		{"a line feed at the bottom scrolls", 10, 2, "a\r\nb\r\nc", want{[]string{"b", "c"}, 1, 1}},
		{"wide characters take two cells", 10, 2, "漢x", want{[]string{"漢x", ""}, 0, 3}},
		{"up stops at the top", 10, 2, "a\x1b[5AX", want{[]string{"aX", ""}, 0, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newScreen(tc.width, tc.height)
			if _, err := s.Write([]byte(tc.in)); err != nil {
				t.Fatal(err)
			}
			if got := s.lines(); !slices.Equal(got, tc.want.lines) || s.row != tc.want.row || s.col != tc.want.col {
				t.Fatalf("screen %q, cursor (%d, %d); want %q, (%d, %d)", got, s.row, s.col, tc.want.lines, tc.want.row, tc.want.col)
			}
		})
	}
	if strings.Join(newScreen(3, 1).lines(), "") != "" {
		t.Fatal("a new screen is not blank")
	}
}
