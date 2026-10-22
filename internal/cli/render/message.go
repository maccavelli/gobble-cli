package render

import "strings"

// Message wraps a message of gobble's own to width columns (0028-MADR D6).
// Each line of s that does not fit is broken at spaces, its continuation
// rows indented two columns past the line's own indentation, and a word
// wider than its row is left whole on it. Escape sequences take no columns.
// A line that fits, and every line when width is 0 or less, is left as it
// is.
func Message(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if visibleWidth(line) <= width {
			continue
		}
		body := strings.TrimLeft(line, " ")
		lead := line[:len(line)-len(body)]
		cont := lead + "  "
		var b strings.Builder
		b.WriteString(lead)
		col, rowStart := len(lead), true
		for word := range strings.FieldsSeq(body) {
			w := visibleWidth(word)
			switch {
			case rowStart:
			case col+1+w <= width:
				b.WriteByte(' ')
				col++
			default:
				b.WriteString("\n" + cont)
				col = len(cont)
			}
			b.WriteString(word)
			col, rowStart = col+w, false
		}
		lines[i] = b.String()
	}
	return strings.Join(lines, "\n")
}

// visibleWidth is the columns s takes on a terminal, its escape sequences
// taking none.
func visibleWidth(s string) int {
	n := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			i += escapeLen(s[i:])
			continue
		}
		j := i + 1
		for j < len(s) && s[j] != 0x1b {
			j++
		}
		n += displayWidth(s[i:j])
		i = j
	}
	return n
}
