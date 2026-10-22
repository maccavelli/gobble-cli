// Package termtest is a terminal's screen for tests of the native CLI mode:
// what is drawn is written to a Screen, and the test checks the rows and the
// cursor it ends with, not the bytes (0027-MADR D6, as amended).
// Stability: internal
//
// Only tests import it. It imports only the standard library and
// golang.org/x/text.
package termtest

import (
	"strings"

	"golang.org/x/text/width"
)

// Screen is a terminal's screen, enough of one to check what the CLI mode
// draws: a grid of cells with a cursor; printable runes, a wide one taking
// two cells; \r; line feed, scrolling at the bottom; cursor up, down,
// forward and back, and home; erase to the end of the screen and of the
// line, and the whole screen; hiding and showing the cursor; other private
// modes (bracketed paste, synchronized output) and other sequences, which
// show nothing. A rune written in the last column leaves a
// wrap pending, as terminals do: the next rune goes to the next row's start,
// and \r, a line feed or a move cancels it.
//
// A line feed keeps the column, as in raw mode, where the editor draws.
// Cooked makes it also return to the row's start, as a terminal's output
// processing does for what is written outside raw mode.
type Screen struct {
	// Row and Col are the cursor's.
	Row, Col int
	// Cooked makes a line feed a new line.
	Cooked bool
	// Hidden is the cursor hidden, by DECTCEM (CSI ?25l; CSI ?25h shows
	// it).
	Hidden bool

	width, height int
	cells         [][]rune
	pending       bool
}

// wideTail marks the second cell of a wide rune.
const wideTail = -1

// New returns a blank screen of columns by rows, the cursor at the top
// left.
func New(columns, rows int) *Screen {
	s := &Screen{width: columns, height: rows}
	for range rows {
		s.cells = append(s.cells, s.blank())
	}
	return s
}

func (s *Screen) blank() []rune {
	r := make([]rune, s.width)
	for i := range r {
		r[i] = ' '
	}
	return r
}

// Write draws p.
func (s *Screen) Write(p []byte) (int, error) {
	runes := []rune(string(p))
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == '\r':
			s.Col, s.pending = 0, false
		case r == '\n':
			s.pending = false
			if s.Cooked {
				s.Col = 0
			}
			s.down()
		case r == 0x1b && i+1 < len(runes) && runes[i+1] == '[':
			i = s.sequence(runes, i+2)
		case r < 0x20:
		default:
			s.put(r)
		}
	}
	return len(p), nil
}

// sequence applies the control sequence whose parameters start at j and
// returns the index of its final character. A private one (bracketed
// paste) shows nothing.
func (s *Screen) sequence(runes []rune, j int) int {
	private := j < len(runes) && runes[j] == '?'
	if private {
		j++
	}
	start := j
	for j < len(runes) && (runes[j] >= '0' && runes[j] <= '9' || runes[j] == ';') {
		j++
	}
	if j >= len(runes) {
		return j
	}
	if private {
		if string(runes[start:j]) == "25" && (runes[j] == 'h' || runes[j] == 'l') {
			s.Hidden = runes[j] == 'l'
		}
		return j
	}
	n, given := 1, j > start
	if given {
		n = 0
		for _, d := range runes[start:j] {
			if d == ';' {
				break
			}
			n = n*10 + int(d-'0')
		}
	}
	s.csi(runes[j], n, given)
	return j
}

// csi applies one control sequence: count n, given says it was written.
func (s *Screen) csi(final rune, n int, given bool) {
	s.pending = false
	switch final {
	case 'A':
		s.Row = max(s.Row-n, 0)
	case 'B':
		s.Row = min(s.Row+n, s.height-1)
	case 'C':
		s.Col = min(s.Col+n, s.width-1)
	case 'D':
		s.Col = max(s.Col-n, 0)
	case 'H':
		s.Row, s.Col = 0, 0
	case 'J':
		if given && n == 2 {
			for r := range s.cells {
				s.cells[r] = s.blank()
			}
		}
		if !given || n == 0 {
			for c := s.Col; c < s.width; c++ {
				s.cells[s.Row][c] = ' '
			}
			for r := s.Row + 1; r < s.height; r++ {
				s.cells[r] = s.blank()
			}
		}
	case 'K':
		if !given || n == 0 {
			for c := s.Col; c < s.width; c++ {
				s.cells[s.Row][c] = ' '
			}
		}
	}
}

// down moves the cursor a row down, scrolling at the bottom.
func (s *Screen) down() {
	if s.Row == s.height-1 {
		s.cells = append(s.cells[1:], s.blank())
		return
	}
	s.Row++
}

// put writes r at the cursor, taking its pending wrap first.
func (s *Screen) put(r rune) {
	w := runeWidth(r)
	if s.pending || s.Col+w > s.width {
		s.Col, s.pending = 0, false
		s.down()
	}
	s.cells[s.Row][s.Col] = r
	if w == 2 {
		s.cells[s.Row][s.Col+1] = wideTail
	}
	if s.Col+w >= s.width {
		s.Col, s.pending = s.width-1, true
		return
	}
	s.Col += w
}

// Lines are the screen's rows, trailing spaces dropped.
func (s *Screen) Lines() []string {
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

// runeWidth is the columns r takes: two for a wide or fullwidth character,
// one for any other, as the editor counts them.
func runeWidth(r rune) int {
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	}
	return 1
}
