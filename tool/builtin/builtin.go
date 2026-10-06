package builtin

import (
	"fmt"
	"path/filepath"
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

// resolve makes path absolute against the session's working directory.
func resolve(env tool.Env, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(env.Cwd, path)
}

// shown is path as a title shows it: relative to cwd when it lies under it.
func shown(env tool.Env, path string) string {
	abs := resolve(env, path)
	if env.Cwd != "" {
		if rel, err := filepath.Rel(env.Cwd, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(abs)
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

// head keeps the first maxLines lines and maxBytes bytes of s, and says
// what it cut.
func head(s string) string {
	lines := strings.SplitAfter(s, "\n")
	total := len(lines)
	if total > 0 && lines[total-1] == "" {
		total--
	}
	var b strings.Builder
	n := 0
	for _, l := range lines[:total] {
		if n == maxLines || b.Len()+len(l) > maxBytes {
			fmt.Fprintf(&b, "\n[truncated: showing the first %d of %d lines]\n", n, total)
			return b.String()
		}
		b.WriteString(l)
		n++
	}
	return b.String()
}

// tail keeps the last maxLines lines and maxBytes bytes of s, and says what
// it cut.
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

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimSuffix(s, "\n"), "\n") + 1
}
