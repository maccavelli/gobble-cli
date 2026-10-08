package builtin

import (
	"strings"
)

// The forgiving strategies of layer 3 (editmatch.go), from opencode's edit
// tool (packages/opencode/src/tool/edit.ts at ecc4916, which credits Cline
// and gemini-cli): each returns every region of content that find could
// have meant, as byte spans of content. A strategy is used only when it
// returns exactly one.
var strategies = []func(content, find string) []span{
	lineTrimmed,
	indentFlexible,
	escapeNormalized,
}

// lineBlock is a run of whole lines of content, as a span. It includes the
// newline after the last line when find ends with one, so a newText that
// also ends with one replaces it.
func lineBlock(lines []span, content string, first, n int, find string) span {
	s := span{start: lines[first].start, end: lines[first+n-1].end}
	if !strings.HasSuffix(find, "\n") && strings.HasSuffix(content[s.start:s.end], "\n") {
		s.end--
	}
	return s
}

// findLines are find's lines, without the empty one after a final newline.
func findLines(find string) []string {
	return strings.Split(strings.TrimSuffix(find, "\n"), "\n")
}

// contentLine is line i of content, without its newline.
func contentLine(content string, l span) string {
	return strings.TrimSuffix(content[l.start:l.end], "\n")
}

// blocks returns the line blocks of content whose lines equal find's under
// eq, which compares a run of content lines with find's lines.
func blocks(content, find string, eq func(got, want []string) bool) []span {
	want := findLines(find)
	lines := spans(content)
	var out []span
	for first := 0; first+len(want) <= len(lines); first++ {
		got := make([]string, len(want))
		for j := range want {
			got[j] = contentLine(content, lines[first+j])
		}
		if eq(got, want) {
			out = append(out, lineBlock(lines, content, first, len(want), find))
		}
	}
	return out
}

// lineTrimmed matches line by line, ignoring each line's leading and
// trailing whitespace.
func lineTrimmed(content, find string) []span {
	return blocks(content, find, func(got, want []string) bool {
		for j := range want {
			if strings.TrimSpace(got[j]) != strings.TrimSpace(want[j]) {
				return false
			}
		}
		return true
	})
}

// indentFlexible matches lines that are equal once the indentation common
// to all of them is removed, so a block indented one level deeper or
// shallower still matches.
func indentFlexible(content, find string) []span {
	return blocks(content, find, func(got, want []string) bool {
		g, w := dedent(got), dedent(want)
		for j := range w {
			if g[j] != w[j] {
				return false
			}
		}
		return true
	})
}

// dedent removes the leading whitespace common to every non-blank line.
func dedent(lines []string) []string {
	common := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if common < 0 || n < common {
			common = n
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		switch {
		case strings.TrimSpace(l) == "":
			out[i] = ""
		case common > 0:
			out[i] = l[common:]
		default:
			out[i] = l
		}
	}
	return out
}

// escapes are the sequences a model escapes once too often, and what it
// meant.
var escapes = strings.NewReplacer(
	`\n`, "\n", `\t`, "\t", `\r`, "\r", `\'`, "'", `\"`, `"`, "\\`", "`", `\$`, "$", `\\`, `\`,
)

// escapeNormalized matches find with one level of escaping removed, for
// oldText written as a string literal's source rather than its value.
func escapeNormalized(content, find string) []span {
	plain := escapes.Replace(find)
	if plain == find || plain == "" {
		return nil
	}
	var out []span
	for off := 0; ; {
		i := strings.Index(content[off:], plain)
		if i < 0 {
			return out
		}
		out = append(out, span{start: off + i, end: off + i + len(plain)})
		off += i + len(plain)
	}
}
