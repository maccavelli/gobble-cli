package termtest

import (
	"slices"
	"strings"
	"testing"
)

// The screen model, checked before anything is checked with it.
func TestScreen(t *testing.T) {
	type want struct {
		lines    []string
		row, col int
	}
	cases := []struct {
		name          string
		width, height int
		cooked        bool
		in            string
		want          want
	}{
		{"text and a line end", 10, 3, false, "ab\r\ncd", want{[]string{"ab", "cd", ""}, 1, 2}},
		{"\\r goes to the row's start", 10, 2, false, "abc\rX", want{[]string{"Xbc", ""}, 0, 1}},
		{"a line feed keeps the column", 10, 3, false, "ab\nc", want{[]string{"ab", "  c", ""}, 1, 3}},
		{"moves up, down, forward, back", 10, 4, false, "a\r\nb\r\nc\x1b[2A\x1b[3C\x1b[1DX\x1b[1BY", want{[]string{"a  X", "b   Y", "c", ""}, 1, 5}},
		{"a move with no count moves one", 10, 3, false, "ab\r\ncd\x1b[AX", want{[]string{"abX", "cd", ""}, 0, 3}},
		{"erase to the end of the screen", 10, 3, false, "abcd\r\nefgh\r\nijkl\x1b[2A\r\x1b[2C\x1b[J", want{[]string{"ab", "", ""}, 0, 2}},
		{"erase to the end of the line", 10, 2, false, "abcdef\r\x1b[3C\x1b[K", want{[]string{"abc", ""}, 0, 3}},
		{"a full row leaves a wrap pending", 4, 3, false, "abcdX", want{[]string{"abcd", "X", ""}, 1, 1}},
		{"\\r cancels the pending wrap", 4, 3, false, "abcd\rX", want{[]string{"Xbcd", "", ""}, 0, 1}},
		{"a line feed at the bottom scrolls", 10, 2, false, "a\r\nb\r\nc", want{[]string{"b", "c"}, 1, 1}},
		{"wide characters take two cells", 10, 2, false, "漢x", want{[]string{"漢x", ""}, 0, 3}},
		{"up stops at the top", 10, 2, false, "a\x1b[5AX", want{[]string{"aX", ""}, 0, 2}},
		{"a private mode changes nothing on screen", 10, 2, false, "a\x1b[?2004hb\x1b[?2004l", want{[]string{"ab", ""}, 0, 2}},
		{"erase the whole screen keeps the cursor", 10, 3, false, "ab\r\ncd\x1b[2JX", want{[]string{"", "  X", ""}, 1, 3}},
		{"home is the top left", 10, 3, false, "ab\r\ncd\x1b[HX", want{[]string{"Xb", "cd", ""}, 0, 1}},
		{"a sequence with parameters shows nothing", 10, 2, false, "a\x1b[1;31mb\x1b[0m", want{[]string{"ab", ""}, 0, 2}},
		{"cooked, a line feed is a new line", 10, 3, true, "ab\ncd\r\ne", want{[]string{"ab", "cd", "e"}, 2, 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := New(tc.width, tc.height)
			s.Cooked = tc.cooked
			if _, err := s.Write([]byte(tc.in)); err != nil {
				t.Fatal(err)
			}
			if got := s.Lines(); !slices.Equal(got, tc.want.lines) || s.Row != tc.want.row || s.Col != tc.want.col {
				t.Fatalf("screen %q, cursor (%d, %d); want %q, (%d, %d)", got, s.Row, s.Col, tc.want.lines, tc.want.row, tc.want.col)
			}
		})
	}
	if strings.Join(New(3, 1).Lines(), "") != "" {
		t.Fatal("a new screen is not blank")
	}
}

// A cursor position query is answered with the cursor's place, 1-based.
func TestScreenAnswersCursorQuery(t *testing.T) {
	s := New(20, 5)
	var got string
	s.Answer = func(reply string) { got = reply }
	if _, err := s.Write([]byte("ab\r\ncdef\x1b[6n")); err != nil {
		t.Fatal(err)
	}
	if got != "\x1b[2;5R" {
		t.Fatalf("reply %q, want %q", got, "\x1b[2;5R")
	}
	if l := s.Lines(); l[1] != "cdef" {
		t.Fatalf("the query showed on screen: %q", l)
	}
}

// Narrowed, the screen's rows change as each kind of terminal measured in
// 0030-PLAN P1 changes them.
func TestScreenResize(t *testing.T) {
	draw := func() *Screen {
		s := New(20, 10)
		if _, err := s.Write([]byte("header\r\n0123456789abcdefghij\r\nshort\r\nKLMNOPQRSTUVWX")); err != nil {
			t.Fatal(err)
		}
		return s
	}
	blank := func(n int) []string { return make([]string, n) }
	for _, tc := range []struct {
		name     string
		how      Narrowing
		lines    []string
		row, col int
	}{
		{"re-wrapped, the cursor down", ReflowDown,
			slices.Concat([]string{"header", "01234567", "89abcdef", "ghij", "short", "KLMNOPQR", "STUVWX"}, blank(3)), 6, 6},
		{"re-wrapped, the cursor kept", ReflowKeepRow,
			slices.Concat([]string{"ghij", "short", "KLMNOPQR", "STUVWX"}, blank(6)), 3, 6},
		{"cut", Cut, slices.Concat([]string{"header", "01234567", "short", "KLMNOPQR"}, blank(6)), 3, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := draw()
			s.Resize(8, tc.how)
			if got := s.Lines(); !slices.Equal(got, tc.lines) || s.Row != tc.row || s.Col != tc.col {
				t.Fatalf("screen %q, cursor (%d, %d); want %q, (%d, %d)", got, s.Row, s.Col, tc.lines, tc.row, tc.col)
			}
		})
	}
}

// The cursor is hidden by DECTCEM off and shown by on; any other private
// mode leaves it as it is.
func TestScreenHidesTheCursor(t *testing.T) {
	s := New(10, 2)
	for _, step := range []struct {
		in     string
		hidden bool
	}{
		{"a", false}, {"\x1b[?25l", true}, {"\x1b[?2026h", true}, {"b\x1b[?25h", false}, {"\x1b[?2026l", false},
	} {
		if _, err := s.Write([]byte(step.in)); err != nil {
			t.Fatal(err)
		}
		if s.Hidden != step.hidden {
			t.Fatalf("after %q, hidden %v, want %v", step.in, s.Hidden, step.hidden)
		}
	}
	if got := s.Lines()[0]; got != "ab" {
		t.Fatalf("row %q, want %q", got, "ab")
	}
}
