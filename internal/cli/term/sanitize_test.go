package term

import (
	"strings"
	"testing"
)

// The bidi controls and U+FFFD are built from code points, so this file
// carries no invisible or reordering characters itself.
var (
	rlo  = string(rune(0x202e))
	lre  = string(rune(0x202a))
	rle  = string(rune(0x202b))
	pdf  = string(rune(0x202c))
	lro  = string(rune(0x202d))
	lri  = string(rune(0x2066))
	pdi  = string(rune(0x2069))
	lrm  = string(rune(0x200e))
	rlm  = string(rune(0x200f))
	alm  = string(rune(0x061c))
	repl = string(rune(0xfffd))
)

// escaped returns the visible form EscapeBidi writes for a code point.
func escaped(hex string) string { return "\\u" + hex }

func TestSanitize(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"clean text passes unchanged", "hello, wörld\tok\nnext 世界", "hello, wörld\tok\nnext 世界"},
		{"CSI clear screen", "a\x1b[2Jb", "ab"},
		{"CSI SGR with params", "\x1b[1;31mred\x1b[0m", "red"},
		{"OSC title ended by BEL", "x\x1b]0;title\x07y", "xy"},
		{"OSC hyperlink ended by ST", "\x1b]8;;http://e.invalid\x1b\\link\x1b]8;;\x1b\\", "link"},
		{"DCS up to ST", "a\x1bPq#0;2;0;0;0\x1b\\b", "ab"},
		{"APC up to ST", "a\x1b_payload\x1b\\b", "ab"},
		{"C1 CSI U+009B", "a\u009b2Jb", "ab"},
		{"C1 OSC up to C1 ST", "a\u009d0;t\u009cb", "ab"},
		{"other C1 dropped", "a\u0085b", "ab"},
		{"raw 0x9B byte is not a CSI", "a\x9b2Jb", "a" + repl + "2Jb"},
		{"C0 controls and DEL dropped", "a\rb\x07c\x08d\x7fe", "abcde"},
		{"two-character escape", "a\x1bcb\x1b(Bc", "abc"},
		{"unterminated OSC dropped", "a\x1b]0;never ends", "a"},
		{"lone ESC at end", "a\x1b", "a"},
		{"bidi is not Sanitize's job", "a" + rlo + "b", "a" + rlo + "b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Sanitize(tc.in); got != tc.want {
				t.Fatalf("Sanitize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestEscapeBidi(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"clean", "plain text", "plain text"},
		{"RLO", "a" + rlo + "b", "a" + escaped("202E") + "b"},
		{"isolates and marks", lri + "x" + pdi + lrm + rlm + alm,
			escaped("2066") + "x" + escaped("2069") + escaped("200E") + escaped("200F") + escaped("061C")},
		{"embeddings", lre + rle + pdf + lro,
			strings.Join([]string{escaped("202A"), escaped("202B"), escaped("202C"), escaped("202D")}, "")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EscapeBidi(tc.in); got != tc.want {
				t.Fatalf("EscapeBidi(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
