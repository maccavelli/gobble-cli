package editor

import (
	"fmt"
	"io"
	"strings"
)

// A drawing that writes is one synchronized update (DEC mode 2026), with the
// cursor hidden while it moves (DECTCEM); a terminal without mode 2026
// ignores it.
const (
	syncOn     = "\x1b[?2026h"
	syncOff    = "\x1b[?2026l"
	cursorHide = "\x1b[?25l"
	cursorShow = "\x1b[?25h"
)

// renderer draws the line on the terminal and draws it again after each
// key, by relative moves only, from where the cursor stands (0027-MADR D3).
// It remembers the rows it drew last, the cursor's row among them and its
// column.
type renderer struct {
	out io.Writer
	// drawn are the rows drawn last, none before the first drawing and
	// after finish; at and column are the cursor's place among them.
	drawn      []string
	at, column int
	// up, for a first drawing after a resize, is how many rows above the
	// cursor the line's first row now is (lift).
	up int
	// prefix, for a first drawing, is written first in the same
	// synchronized update: a reprint's clear and conversation.
	prefix string
}

// lift is how many rows above the cursor the line drawn last starts, now
// that the terminal is terminal columns wide (0030-MADR D2, as amended).
// Terminals re-wrap the line's rows at the new width, unless they cut them
// off, as xterm (cut) does; the column of a cursor report (col, 1-based,
// when known) tells which when the cursor's column was past the new width.
// A terminal that keeps the cursor's row pushes the rows above it out of
// the top; moving up stops at the screen's first row, which is where the
// line is then drawn from.
func (r *renderer) lift(terminal, col int, known, cut bool) int {
	if len(r.drawn) == 0 || terminal <= 0 {
		return 0
	}
	if known && r.column >= terminal && r.column%terminal != terminal-1 {
		cut = col-1 == terminal-1
	}
	if cut {
		return r.at
	}
	up := r.column / terminal
	for _, d := range r.drawn[:r.at] {
		up += max((displayWidth(d)+terminal-1)/terminal, 1)
	}
	return up
}

// draw draws l in place of the line drawn last, never erasing first: each
// row that changed is written from its first changed column, what is left
// of a row that got shorter is erased, and so are rows l no longer has; the
// cursor then goes to its row and column. A terminal that paints in the
// middle shows each row with at least what its old and new text share, and
// the drawing is one synchronized update with the cursor hidden (0027-MADR
// D3, deviation 4). A first drawing starts at the row's start and erases to
// the end of the screen. Every move to a column starts at the row's start,
// so a wrap a full row left pending cannot shift it. A line taller than
// height is drawn from the rows around the cursor's, so that row is on
// screen; a height of 0 draws every row.
func (r *renderer) draw(l screenLayout, height int) error {
	rows, at := l.rows, l.row
	if height > 0 && len(rows) > height {
		from := min(max(at-height+1, 0), len(rows)-height)
		rows, at = rows[from:from+height], at-from
	}
	var b strings.Builder
	cur := r.at
	if len(r.drawn) == 0 {
		b.WriteString(r.prefix)
		if r.up > 0 {
			fmt.Fprintf(&b, "\x1b[%dA", r.up)
		}
		b.WriteString("\r\x1b[J")
	}
	r.up, r.prefix = 0, ""
	for i, row := range rows {
		var was string
		if i < len(r.drawn) {
			if was = r.drawn[i]; was == row {
				continue
			}
		}
		cur = moveTo(&b, cur, i)
		same := sharedPrefix(was, row)
		b.WriteByte('\r')
		if w := displayWidth(same); w > 0 {
			fmt.Fprintf(&b, "\x1b[%dC", w)
		}
		b.WriteString(row[len(same):])
		if displayWidth(row) < displayWidth(was) {
			b.WriteString("\x1b[K")
		}
	}
	if len(r.drawn) > len(rows) {
		cur = moveTo(&b, cur, len(rows))
		b.WriteString("\r\x1b[J")
	}
	if b.Len() > 0 || cur != at || r.column != l.column {
		moveTo(&b, cur, at)
		b.WriteByte('\r')
		if l.column > 0 {
			fmt.Fprintf(&b, "\x1b[%dC", l.column)
		}
	}
	r.drawn, r.at, r.column = rows, at, l.column
	if b.Len() == 0 {
		return nil
	}
	if _, err := io.WriteString(r.out, syncOn+cursorHide+b.String()+cursorShow+syncOff); err != nil {
		return fmt.Errorf("draw the line: %w", err)
	}
	return nil
}

// moveTo moves the cursor from row from of the line to row to: up by a
// sequence, down by line feeds, which scroll when the line grows past the
// screen's last row. The column is left to the caller.
func moveTo(b *strings.Builder, from, to int) int {
	switch {
	case to < from:
		fmt.Fprintf(b, "\x1b[%dA", from-to)
	case to > from:
		b.WriteString(strings.Repeat("\n", to-from))
	}
	return to
}

// sharedPrefix is the runes a and b start with alike.
func sharedPrefix(a, b string) string {
	ra, rb := []rune(a), []rune(b)
	i := 0
	for i < len(ra) && i < len(rb) && ra[i] == rb[i] {
		i++
	}
	return string(ra[:i])
}

// forget drops the last drawing, after the screen was cleared: the next
// drawing is a first one.
func (r *renderer) forget() {
	r.drawn, r.at, r.column = nil, 0, 0
}

// finish leaves the line as drawn and moves the cursor to the start of the
// row below it; the next line is drawn from there.
func (r *renderer) finish() error {
	var b strings.Builder
	if down := len(r.drawn) - 1 - r.at; down > 0 {
		fmt.Fprintf(&b, "\x1b[%dB", down)
	}
	b.WriteString("\r\n")
	r.forget()
	if _, err := io.WriteString(r.out, b.String()); err != nil {
		return fmt.Errorf("finish the line: %w", err)
	}
	return nil
}
