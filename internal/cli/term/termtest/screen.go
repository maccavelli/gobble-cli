// Package termtest is a terminal's screen for tests of the native CLI mode:
// what is drawn is written to a Screen, and the test checks the rows and the
// cursor it ends with, not the bytes (0027-MADR D6, as amended).
// Stability: internal
//
// Only tests import it. It imports only the standard library and
// golang.org/x/text.
package termtest

import (
	"fmt"
	"slices"
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
	// Answer, when set, is given the terminal's reply to a cursor position
	// query (CSI 6 n): CSI row ; column R, 1-based.
	Answer func(reply string)

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
	case 'n':
		if n == 6 && s.Answer != nil {
			s.Answer(fmt.Sprintf("\x1b[%d;%dR", s.Row+1, s.Col+1))
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

// Narrowing is what a terminal does to its rows when its window gets
// narrower, the three ways 0030-PLAN P1 measured.
type Narrowing int

const (
	// ReflowDown re-wraps each row at the new width and moves the cursor
	// with its cell, down (WezTerm, kitty, Terminal.app, Windows).
	ReflowDown Narrowing = iota
	// ReflowKeepRow re-wraps each row but keeps the cursor on its screen
	// row, pushing the rows above it out of the top (iTerm2, VTE).
	ReflowKeepRow
	// Cut keeps each row on one row, cut at the new width (xterm).
	Cut
)

// Resize makes the screen columns wide, its rows changed as how says.
// Each row is taken as a line of its own, as the line session writes
// them, and re-wrapped by runes, which is enough for the ASCII the tests
// draw: a wide character is not kept whole.
func (s *Screen) Resize(columns int, how Narrowing) {
	text := make([][]rune, len(s.cells))
	for i, row := range s.cells {
		text[i] = []rune(strings.TrimRight(string(slices.DeleteFunc(slices.Clone(row), func(r rune) bool { return r == wideTail })), " "))
	}
	row, col := s.Row, s.Col
	s.width, s.pending = columns, false
	if how == Cut {
		for i := range s.cells {
			s.cells[i] = s.blank()
			copy(s.cells[i], text[i][:min(len(text[i]), columns)])
		}
		s.Col = min(col, columns-1)
		return
	}
	var rows [][]rune
	at := 0
	for i, line := range text {
		if i == row {
			at = len(rows) + col/columns
		}
		for len(line) > columns {
			rows = append(rows, line[:columns])
			line = line[columns:]
		}
		rows = append(rows, line)
	}
	start := max(at-(s.height-1), 0) // the cursor's row on screen, the rows above it scrolled off as needed
	if how == ReflowKeepRow {
		start = max(at-row, 0)
	}
	for i := range s.cells {
		s.cells[i] = s.blank()
		if start+i < len(rows) {
			copy(s.cells[i], rows[start+i])
		}
	}
	s.Row, s.Col = at-start, col%columns
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
