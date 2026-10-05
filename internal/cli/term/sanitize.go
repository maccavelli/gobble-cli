package term

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	esc = 0x1b
	bel = 0x07
	del = 0x7f
	c1  = 0x80 // first C1 control
	c1N = 0x9f // last C1 control

	c1DCS = 0x90
	c1SOS = 0x98
	c1CSI = 0x9b
	c1ST  = 0x9c
	c1OSC = 0x9d
	c1PM  = 0x9e
	c1APC = 0x9f
)

// Sanitize removes everything from s that a terminal would act on rather
// than print (0008-MADR D8):
//   - ESC-introduced sequences: CSI, OSC up to BEL or ST, DCS, SOS, PM and
//     APC up to ST, and two-character escapes;
//   - their C1 forms (U+009B and the others) and every other C1 control;
//   - C0 controls except tab and newline, and DEL.
//
// An invalid UTF-8 byte becomes U+FFFD, so a raw 0x9B cannot act as an
// 8-bit CSI. An unterminated sequence is dropped to the end of s.
func Sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			b.WriteRune(utf8.RuneError)
			i++
		case r == esc:
			i = skipEscape(s, i+1)
		case r >= c1 && r <= c1N:
			i = skipC1(s, i+size, r)
		case r == '\t' || r == '\n':
			b.WriteRune(r)
			i += size
		case r < 0x20 || r == del:
			i += size
		default:
			b.WriteString(s[i : i+size])
			i += size
		}
	}
	return b.String()
}

// skipEscape returns the index after the sequence whose ESC preceded i.
func skipEscape(s string, i int) int {
	if i >= len(s) {
		return i
	}
	switch s[i] {
	case '[':
		return skipCSI(s, i+1)
	case ']':
		return skipString(s, i+1, true)
	case 'P', 'X', '^', '_':
		return skipString(s, i+1, false)
	}
	// Two-character escape: intermediates 0x20–0x2F, then one final byte.
	for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2f {
		i++
	}
	if i < len(s) && s[i] >= 0x30 && s[i] <= 0x7e {
		i++
	}
	return i
}

// skipC1 returns the index after the sequence a C1 control r introduced.
func skipC1(s string, i int, r rune) int {
	switch r {
	case c1CSI:
		return skipCSI(s, i)
	case c1OSC:
		return skipString(s, i, true)
	case c1DCS, c1SOS, c1PM, c1APC:
		return skipString(s, i, false)
	}
	return i
}

// skipCSI skips parameter bytes 0x30–0x3F, intermediates 0x20–0x2F and the
// final byte 0x40–0x7E. Any other byte ends the sequence unconsumed.
func skipCSI(s string, i int) int {
	for i < len(s) && s[i] >= 0x20 && s[i] <= 0x3f {
		i++
	}
	if i < len(s) && s[i] >= 0x40 && s[i] <= 0x7e {
		i++
	}
	return i
}

// skipString skips a control string up to ST (ESC \ or U+009C), or BEL when
// belEnds is set (OSC).
func skipString(s string, i int, belEnds bool) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == bel && belEnds:
			return i + size
		case r == c1ST:
			return i + size
		case r == esc && i+1 < len(s) && s[i+1] == '\\':
			return i + 2
		}
		i += size
	}
	return i
}

// EscapeBidi rewrites the Unicode bidirectional controls as visible \uXXXX
// text, so tool output cannot reorder what the user reads (Trojan Source).
func EscapeBidi(s string) string {
	if !strings.ContainsFunc(s, isBidi) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		if isBidi(r) {
			fmt.Fprintf(&b, `\u%04X`, r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isBidi(r rune) bool {
	switch {
	case r == 0x061c, r == 0x200e, r == 0x200f:
		return true
	case r >= 0x202a && r <= 0x202e:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	}
	return false
}
