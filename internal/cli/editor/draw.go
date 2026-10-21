package editor

import (
	"fmt"
	"io"
	"strings"
)

// renderer draws the line on the terminal and draws it again after each
// key, by relative moves only, from where the cursor stands (0027-MADR D3).
// It remembers the rows it drew last and the cursor's row among them.
type renderer struct {
	out io.Writer
	// at is the cursor's row within the rows drawn last; rows is how many
	// there were, 0 before the first drawing and after finish.
	at, rows int
}

// draw draws l in place of the line drawn last: up to that line's first
// row, erase to the end of the screen, l's rows, then the cursor to its
// row and column. Every move to a column starts at the row's start, so a
// wrap a full row left pending cannot shift it. A line taller than height
// is drawn from the rows around the cursor's, so that row is on screen; a
// height of 0 draws every row.
func (r *renderer) draw(l screenLayout, height int) error {
	rows, at := l.rows, l.row
	if height > 0 && len(rows) > height {
		from := min(max(at-height+1, 0), len(rows)-height)
		rows, at = rows[from:from+height], at-from
	}
	var b strings.Builder
	if r.at > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", r.at)
	}
	b.WriteString("\r\x1b[J")
	b.WriteString(strings.Join(rows, "\r\n"))
	if up := len(rows) - 1 - at; up > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", up)
	}
	b.WriteByte('\r')
	if l.column > 0 {
		fmt.Fprintf(&b, "\x1b[%dC", l.column)
	}
	r.at, r.rows = at, len(rows)
	if _, err := io.WriteString(r.out, b.String()); err != nil {
		return fmt.Errorf("draw the line: %w", err)
	}
	return nil
}

// finish leaves the line as drawn and moves the cursor to the start of the
// row below it; the next line is drawn from there.
func (r *renderer) finish() error {
	var b strings.Builder
	if down := r.rows - 1 - r.at; down > 0 {
		fmt.Fprintf(&b, "\x1b[%dB", down)
	}
	b.WriteString("\r\n")
	r.at, r.rows = 0, 0
	if _, err := io.WriteString(r.out, b.String()); err != nil {
		return fmt.Errorf("finish the line: %w", err)
	}
	return nil
}
