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
			b.WriteString(styleInline(styleMarker(body)))
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

// styleMarker colours a list marker (-, *, +, 1. or 1)) or a quote marker
// (>) that starts line after indentation.
func styleMarker(line string) string {
	body := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(body)]
	n := markerLen(body)
	if n == 0 {
		return line
	}
	return indent + term.Cyan.Wrap(body[:n], true) + body[n:]
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

// styleInline dims inline code spans, backticks included. A span closes
// with a backtick run of the same length; an unclosed run is left as is.
func styleInline(line string) string {
	if !strings.Contains(line, "`") {
		return line
	}
	var b strings.Builder
	for line != "" {
		i := strings.IndexByte(line, '`')
		if i < 0 {
			b.WriteString(line)
			break
		}
		b.WriteString(line[:i])
		rest := line[i:]
		run := rest[:len(rest)-len(strings.TrimLeft(rest, "`"))]
		end := closingRun(rest[len(run):], len(run))
		if end < 0 {
			b.WriteString(run)
			line = rest[len(run):]
			continue
		}
		span := rest[:len(run)+end+len(run)]
		b.WriteString(term.Dim.Wrap(span, true))
		line = rest[len(span):]
	}
	return b.String()
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
