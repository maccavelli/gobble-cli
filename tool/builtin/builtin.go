package builtin

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/tool"
)

const (
	// maxLines and maxBytes bound what a read or a command's output gives
	// the model, as Pi does (0005-MADR "Tools").
	maxLines = 2000
	maxBytes = 50 * 1024
	// maxTitle is a title's length in runes, and maxSummary a summary's in
	// bytes (0008-MADR D19 item 3).
	maxTitle   = 60
	maxSummary = 400
)

// Tools returns the built-in tools: read, write, edit and bash. 0005-PLAN
// F1 brings them to Pi's parity and adds the rest.
func Tools() []tool.Tool {
	return []tool.Tool{Read(), Write(), Edit(), Bash()}
}

// Names are the built-in tools' names, in Tools' order.
func Names() []string {
	ts := Tools()
	names := make([]string, len(ts))
	for i, t := range ts {
		names[i] = t.Spec().Name
	}
	return names
}

// title is s on one line, cut to maxTitle runes.
func title(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= maxTitle {
		return s
	}
	r := []rune(s)
	return string(r[:maxTitle-1]) + "…"
}

// summary is s cut to maxSummary bytes on a rune boundary.
func summary(s string) string {
	if len(s) <= maxSummary {
		return s
	}
	cut := maxSummary - len("…")
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}

// tail keeps the last maxLines lines and maxBytes bytes of s, and says what
// it cut. 0005-PLAN F1b replaces it with Pi's truncateTail and notices.
func tail(s string) string {
	lines := strings.SplitAfter(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	size, start := 0, len(lines)
	for start > 0 && len(lines)-start < maxLines && size+len(lines[start-1]) <= maxBytes {
		start--
		size += len(lines[start])
	}
	out := strings.Join(lines[start:], "")
	if start > 0 {
		return fmt.Sprintf("[truncated: showing the last %d of %d lines]\n%s", len(lines)-start, len(lines), out)
	}
	return out
}

// count is n with its noun: "1 line", "3 lines".
func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimSuffix(s, "\n"), "\n") + 1
}
