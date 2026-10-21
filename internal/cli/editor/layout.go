package editor

import (
	"strings"

	"golang.org/x/text/width"
)

// screenLayout is the line as it is drawn: its rows, each the prompt or
// the indent and its part of the text, and the cursor's row and column
// (0027-MADR D2).
type screenLayout struct {
	rows        []string
	row, column int
}

// lineWidth is the width a line is laid out to: the terminal's, to
// MaxWidth, or DefaultWidth when it is unknown (0024-MADR D9; 0027-MADR D2).
func lineWidth(terminal int) int {
	if terminal <= 0 {
		return DefaultWidth
	}
	return min(terminal, MaxWidth)
}

// layout lays out text after prompt in rows of at most columns, with
// the cursor at cursor, a rune index into text (0027-MADR D2). The first
// row follows the prompt; the others are indented by its width. A row ends
// only at a space, which the break takes and the row does not draw, so a
// word keeps its punctuation; a word wider than a whole row is cut at the
// row's end. The text is unchanged: only where it is drawn. A cursor that
// would stand past the last column goes to the start of the next row,
// which is added, empty, when it is the last.
func layout(prompt string, text []rune, cursor, columns int) screenLayout {
	indent := displayWidth(prompt)
	room := max(columns-indent, 1)
	pad := strings.Repeat(" ", indent)
	var out screenLayout
	found := false
	for start := 0; ; {
		end, next := rowEnd(text, start, room)
		head := pad
		if start == 0 {
			head = prompt
		}
		out.rows = append(out.rows, head+string(text[start:end]))
		last := next >= len(text) && end == len(text)
		if !found && (cursor < next || last) {
			col := indent + runesWidth(text[start:min(cursor, end)])
			if col >= columns {
				if last {
					out.rows = append(out.rows, pad)
				}
				out.row, out.column = len(out.rows)-1, indent
				if !last {
					out.row = len(out.rows)
				}
			} else {
				out.row, out.column = len(out.rows)-1, col
			}
			found = true
		}
		if last {
			return out
		}
		start = next
	}
}

// rowEnd is where the row that starts at start ends in text, and where the
// next row starts: at the last space the room reaches, the space taken by
// the break, or, with no such space, the room's end, cutting the word.
func rowEnd(text []rune, start, room int) (end, next int) {
	fit, used := start, 0
	for fit < len(text) && used+runeWidth(text[fit]) <= room {
		used += runeWidth(text[fit])
		fit++
	}
	if fit == len(text) {
		return fit, fit
	}
	for k := fit; k > start; k-- {
		if text[k] == ' ' {
			return k, k + 1
		}
	}
	return max(fit, start+1), max(fit, start+1)
}

// runesWidth is the columns text takes on a terminal.
func runesWidth(text []rune) int {
	n := 0
	for _, r := range text {
		n += runeWidth(r)
	}
	return n
}

// runeWidth is the columns r takes on a terminal: two for a wide or
// fullwidth character, one for any other.
func runeWidth(r rune) int {
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	}
	return 1
}

// displayWidth is the columns s takes on a terminal.
func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		n += runeWidth(r)
	}
	return n
}
