package render

import (
	"strings"
	"unicode"

	"golang.org/x/text/width"
)

const zwj = 0x200d // zero-width joiner

// displayWidth is the number of terminal columns s occupies: 2 for an East
// Asian Wide or Fullwidth rune, 0 for a combining mark or a zero-width
// joiner, and 1 otherwise. It does not shape emoji sequences, so a joined
// family counts every member.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		switch {
		case r == zwj, unicode.In(r, unicode.Mn, unicode.Me):
		case isWide(r):
			w += 2
		default:
			w++
		}
	}
	return w
}

func isWide(r rune) bool {
	k := width.LookupRune(r).Kind()
	return k == width.EastAsianWide || k == width.EastAsianFullwidth
}

// truncateWidth cuts s to at most w columns, ending it with ellipsis when it
// was cut. ellipsis counts towards w.
func truncateWidth(s string, w int, ellipsis string) string {
	if displayWidth(s) <= w {
		return s
	}
	room := w - displayWidth(ellipsis)
	if room <= 0 {
		return ""
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := displayWidth(string(r))
		if used+rw > room {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String() + ellipsis
}

type align int

const (
	alignLeft align = iota
	alignCenter
	alignRight
)

// Tables re-lays out every GFM table in text: cells are padded to their
// column's display width and aligned as the delimiter row's colons say.
// Lines that are not part of a table pass through unchanged.
func Tables(text string) string {
	lines := strings.SplitAfter(text, "\n")
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(lines); {
		n := tableLen(lines[i:])
		if n == 0 {
			b.WriteString(lines[i])
			i++
			continue
		}
		b.WriteString(layout(lines[i : i+n]))
		i += n
	}
	return b.String()
}

// tableLen returns how many lines from the start of lines form a table: a
// header row, a delimiter row with the same cell count, then body rows.
func tableLen(lines []string) int {
	if len(lines) < 2 || !isRow(lines[0]) || !isRow(lines[1]) {
		return 0
	}
	head, delim := cells(lines[0]), cells(lines[1])
	if len(head) != len(delim) || !isDelimiterRow(delim) {
		return 0
	}
	n := 2
	for n < len(lines) && isRow(lines[n]) {
		n++
	}
	return n
}

func isRow(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "|")
}

func isDelimiterRow(cs []string) bool {
	for _, c := range cs {
		c = strings.TrimSuffix(strings.TrimPrefix(c, ":"), ":")
		if c == "" || strings.Trim(c, "-") != "" {
			return false
		}
	}
	return true
}

// cells splits a row on unescaped pipes, without the outer pipes, trimming
// each cell.
func cells(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	if strings.HasSuffix(s, "|") && !strings.HasSuffix(s, `\|`) {
		s = s[:len(s)-1]
	}
	var out []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '|':
			cur.WriteString(`\|`)
			i++
		case s[i] == '|':
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(s[i])
		}
	}
	return append(out, strings.TrimSpace(cur.String()))
}

func alignOf(delim string) align {
	left, right := strings.HasPrefix(delim, ":"), strings.HasSuffix(delim, ":")
	switch {
	case left && right:
		return alignCenter
	case right:
		return alignRight
	}
	return alignLeft
}

func layout(rows []string) string {
	delim := cells(rows[1])
	cols := len(delim)
	aligns := make([]align, cols)
	widths := make([]int, cols)
	for c, d := range delim {
		aligns[c] = alignOf(d)
		widths[c] = 3
	}
	grid := make([][]string, 0, len(rows)-1)
	for i, r := range rows {
		if i == 1 {
			continue
		}
		cs := cells(r)
		cs = append(cs, make([]string, max(0, cols-len(cs)))...)[:cols]
		for c, v := range cs {
			widths[c] = max(widths[c], displayWidth(v))
		}
		grid = append(grid, cs)
	}
	var b strings.Builder
	for i, cs := range grid {
		writeRow(&b, cs, widths, aligns)
		if i == 0 {
			writeDelimiter(&b, widths, aligns)
		}
	}
	if !strings.HasSuffix(rows[len(rows)-1], "\n") {
		return strings.TrimSuffix(b.String(), "\n")
	}
	return b.String()
}

func writeRow(b *strings.Builder, cs []string, widths []int, aligns []align) {
	b.WriteString("|")
	for c, v := range cs {
		pad := widths[c] - displayWidth(v)
		var l, r int
		switch aligns[c] {
		case alignRight:
			l = pad
		case alignCenter:
			l, r = pad/2, pad-pad/2
		default:
			r = pad
		}
		b.WriteString(" " + strings.Repeat(" ", l) + v + strings.Repeat(" ", r) + " |")
	}
	b.WriteString("\n")
}

func writeDelimiter(b *strings.Builder, widths []int, aligns []align) {
	b.WriteString("|")
	for c, w := range widths {
		d := strings.Repeat("-", w)
		switch aligns[c] {
		case alignRight:
			d = d[:w-1] + ":"
		case alignCenter:
			d = ":" + d[:w-2] + ":"
		}
		b.WriteString(" " + d + " |")
	}
	b.WriteString("\n")
}
