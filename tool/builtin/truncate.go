package builtin

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Output bounds shared by the tools (0005-MADR, Tools; amendment of
// 2026-10-07): at most maxLines lines and maxBytes bytes reach the model,
// whichever comes first. read keeps the head; a command's output keeps the
// tail (0005-PLAN F1b).

// cutBy is the limit that cut the content.
type cutBy int

const (
	cutNone cutBy = iota
	cutLines
	cutBytes
)

// truncation is what truncateTail kept, and what it saw.
type truncation struct {
	content         string
	by              cutBy
	totalLines      int
	outputLines     int
	lastLinePartial bool
}

func (t truncation) truncated() bool { return t.by != cutNone }

// countingLines splits s into lines: none for "", and a final newline does
// not start another line.
func countingLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// truncateTail keeps the last whole lines of s within maxLines lines and
// maxBytes bytes. When the last line alone is over maxBytes it keeps that
// line's end, cut on a UTF-8 boundary, and sets lastLinePartial.
func truncateTail(s string, maxLines, maxBytes int) truncation {
	lines := countingLines(s)
	t := truncation{totalLines: len(lines)}
	if len(lines) <= maxLines && len(s) <= maxBytes {
		t.content, t.outputLines = s, len(lines)
		return t
	}
	t.by = cutLines
	var kept []string
	size := 0
	for i := len(lines) - 1; i >= 0 && len(kept) < maxLines; i-- {
		cost := len(lines[i])
		if len(kept) > 0 {
			cost++
		}
		if size+cost > maxBytes {
			t.by = cutBytes
			if len(kept) == 0 {
				kept = []string{bytesFromEnd(lines[i], maxBytes)}
				t.lastLinePartial = true
			}
			break
		}
		kept = append(kept, lines[i])
		size += cost
	}
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}
	t.content = strings.Join(kept, "\n")
	t.outputLines = len(kept)
	return t
}

// bytesFromEnd is the last maxBytes bytes of s, starting on a rune.
func bytesFromEnd(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	start := len(s) - maxBytes
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
}

// truncateLine cuts a line to maxChars characters and says so.
func truncateLine(line string, maxChars int) (string, bool) {
	n := 0
	for i := range line {
		if n == maxChars {
			return line[:i] + "\u2026 (line cut at " + strconv.Itoa(maxChars) + " characters)", true
		}
		n++
	}
	return line, false
}
