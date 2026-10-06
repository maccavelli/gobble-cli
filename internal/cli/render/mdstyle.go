package render

import (
	"strings"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// Style colours Markdown in place: headings bold, fence lines and inline
// code dim with the fence's language label in cyan, and list and quote
// markers in cyan. Every marker is kept and nothing is reflowed. With
// s.Color false the input is returned byte for byte.
//
// lines is text released by a Stream. The first byte of lines is taken to
// be a line start; a Stream releases mid-line only after an inline
// construct closes, so a block marker cannot appear there.
func Style(lines string, s term.Stream) string {
	if !s.Color || lines == "" {
		return lines
	}
	var b strings.Builder
	b.Grow(len(lines) + len(lines)/4)
	var fence string // the open fence's marker run, or ""
	for line := range strings.Lines(lines) {
		body, nl := strings.CutSuffix(line, "\n")
		switch {
		case fence != "":
			if isFenceClose(body, fence) {
				b.WriteString(term.Dim.Wrap(body, true))
				fence = ""
			} else {
				b.WriteString(body)
			}
		case fenceRun(body) != "":
			fence = fenceRun(body)
			b.WriteString(styleFenceOpen(body, fence))
		case isHeading(body):
			b.WriteString(term.Bold.Wrap(body, true))
		default:
			prefix, rest := splitMarker(body)
			b.WriteString(prefix + styleInline(rest))
		}
		if nl {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// fenceRun returns the run of three or more backticks or tildes that opens a
// fence on line, after at most three spaces of indentation, or "".
func fenceRun(line string) string {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 || t == "" || (t[0] != '`' && t[0] != '~') {
		return ""
	}
	run := t[:len(t)-len(strings.TrimLeft(t, t[:1]))]
	if len(run) < 3 || (run[0] == '`' && strings.Contains(t[len(run):], "`")) {
		return ""
	}
	return run
}

func isFenceClose(line, open string) bool {
	t := strings.TrimSpace(line)
	return len(t) >= len(open) && strings.Trim(t, open[:1]) == ""
}

func styleFenceOpen(line, run string) string {
	i := strings.Index(line, run) + len(run)
	marker, info := line[:i], line[i:]
	lang := strings.TrimSpace(info)
	if lang == "" {
		return term.Dim.Wrap(line, true)
	}
	j := strings.Index(info, lang)
	return term.Dim.Wrap(marker+info[:j], true) + term.Cyan.Wrap(lang, true) + info[j+len(lang):]
}

func isHeading(line string) bool {
	h := len(line) - len(strings.TrimLeft(line, "#"))
	return h >= 1 && h <= 6 && (len(line) == h || line[h] == ' ' || line[h] == '\t')
}

// splitMarker colours a list marker (-, *, +, 1. or 1)) or a quote marker
// (>) that starts line after indentation, and returns it with the
// indentation, apart from the rest of the line, so inline styling never
// reads a * marker as emphasis.
func splitMarker(line string) (prefix, rest string) {
	body := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(body)]
	n := markerLen(body)
	if n == 0 {
		return "", line
	}
	return indent + term.Cyan.Wrap(body[:n], true), body[n:]
}

func markerLen(s string) int {
	switch {
	case s == "":
		return 0
	case s[0] == '>':
		return 1
	case (s[0] == '-' || s[0] == '*' || s[0] == '+') && len(s) > 1 && s[1] == ' ':
		return 1
	}
	d := len(s) - len(strings.TrimLeft(s, "0123456789"))
	if d >= 1 && d <= 9 && len(s) > d+1 && (s[d] == '.' || s[d] == ')') && s[d+1] == ' ' {
		return d + 1
	}
	return 0
}

// styleInline dims inline code spans, backticks included, and styles
// emphasis in the text between them (styleEmphasis). A span closes with a
// backtick run of the same length; an unclosed run is left as is.
func styleInline(line string) string {
	var b strings.Builder
	var text strings.Builder // text outside code spans, not yet styled
	for line != "" {
		i := strings.IndexByte(line, '`')
		if i < 0 {
			text.WriteString(line)
			break
		}
		text.WriteString(line[:i])
		rest := line[i:]
		run := rest[:len(rest)-len(strings.TrimLeft(rest, "`"))]
		end := closingRun(rest[len(run):], len(run))
		if end < 0 {
			text.WriteString(run)
			line = rest[len(run):]
			continue
		}
		b.WriteString(styleEmphasis(text.String()))
		text.Reset()
		span := rest[:len(run)+end+len(run)]
		b.WriteString(term.Dim.Wrap(span, true))
		line = rest[len(span):]
	}
	b.WriteString(styleEmphasis(text.String()))
	return b.String()
}

// emphasisStyle is the style of a delimiter run of n c's, and whether the
// run is one: three * or _ bold italic, two bold, one italic, and ~~ dim
// (0008-PLAN P8).
func emphasisStyle(c byte, n int) (term.Style, bool) {
	switch {
	case c == '~':
		return term.Dim, n == 2
	case n == 1:
		return term.Italic, true
	case n == 2:
		return term.Bold, true
	case n == 3:
		return term.Bold | term.Italic, true
	}
	return 0, false
}

// styleEmphasis styles emphasis in s, which holds no code span. Markers are
// kept and styled with the text. An opener is followed by a non-space, its
// closer (the same run) is preceded by one, and _ opens and closes only at a
// word boundary, so snake_case stays plain. Escaped and unclosed runs stay
// as they are, and spans are not nested: a nested reset would end the outer
// style early.
func styleEmphasis(s string) string {
	if !strings.ContainsAny(s, "*_~") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			b.WriteString(s[i : i+2])
			i += 2
			continue
		}
		if c != '*' && c != '_' && c != '~' {
			b.WriteByte(c)
			i++
			continue
		}
		n := runLen(s, i)
		delim := s[i : i+n]
		style, known := emphasisStyle(c, n)
		if end := closer(s, i, n); known && opens(s, i, n) && end > 0 {
			b.WriteString(style.Wrap(s[i:end+n], true))
			i = end + n
			continue
		}
		b.WriteString(delim)
		i += n
	}
	return b.String()
}

// runLen is the length of the run of s[i] starting at i.
func runLen(s string, i int) int {
	n := 1
	for i+n < len(s) && s[i+n] == s[i] {
		n++
	}
	return n
}

// opens reports whether the run s[i:i+n] can open emphasis.
func opens(s string, i, n int) bool {
	if i+n >= len(s) || isSpace(s[i+n]) {
		return false
	}
	return s[i] != '_' || i == 0 || !isWord(s[i-1])
}

// closer finds the run that closes s[i:i+n]: the same character, exactly n
// long, after a non-space, and for _ before a word boundary. It returns the
// closer's offset, or -1.
func closer(s string, i, n int) int {
	c := s[i]
	for j := i + n + 1; j < len(s); {
		if s[j] == '\\' {
			j += 2
			continue
		}
		if s[j] != c {
			j++
			continue
		}
		m := runLen(s, j)
		if m == n && !isSpace(s[j-1]) && (c != '_' || j+m == len(s) || !isWord(s[j+m])) {
			return j
		}
		j += m
	}
	return -1
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' }

// isWord reports a letter or digit, ASCII or the lead byte of a multibyte
// rune, which is taken as a letter.
func isWord(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}

// closingRun returns the offset in s of a backtick run of exactly n, or -1.
func closingRun(s string, n int) int {
	for off := 0; off < len(s); {
		i := strings.IndexByte(s[off:], '`')
		if i < 0 {
			return -1
		}
		start := off + i
		end := start
		for end < len(s) && s[end] == '`' {
			end++
		}
		if end-start == n {
			return start
		}
		off = end
	}
	return -1
}
