package render

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// wrapCases are replies and the rows a width makes of them (0028-MADR D3,
// D4).
var wrapCases = []struct {
	name, in string
	width    int
	want     string
}{
	{"prose breaks at spaces", "The quick brown fox jumps over the lazy dog.\n", 20,
		"The quick brown fox\njumps over the lazy\ndog.\n"},
	{"a trailing comma stays with its word", "aaaa bbbb cccc dddd, eeee\n", 20, "aaaa bbbb cccc dddd,\neeee\n"},
	{"a word wider than the row is whole on its own row", "see https://example.com/a/very/long/path ok\n", 20,
		"see\nhttps://example.com/a/very/long/path\nok\n"},
	{"wide characters take two columns", "漢字漢字 漢字漢字 漢字漢字\n", 18, "漢字漢字 漢字漢字\n漢字漢字\n"},
	{"a list item continues under its text", "- alpha beta gamma delta epsilon\n", 20,
		"- alpha beta gamma\n  delta epsilon\n"},
	{"an ordered item continues under its text", "12. alpha beta gamma delta\n", 20,
		"12. alpha beta gamma\n    delta\n"},
	{"a nested item keeps its indent", "  * alpha beta gamma delta\n", 20, "  * alpha beta gamma\n    delta\n"},
	{"a quote continues with its marker", "> alpha beta gamma delta epsilon\n", 20,
		"> alpha beta gamma\n> delta epsilon\n"},
	{"a list in a quote", "> - alpha beta gamma delta\n", 20, "> - alpha beta gamma\n>   delta\n"},
	{"a lone dash does not start a row", "aaaa bbbb cccc dddd - eeee\n", 20, "aaaa bbbb cccc\ndddd - eeee\n"},
	{"a number and a dot do not start a row", "aaaa bbbb cccc dddd 1990. eeee\n", 20,
		"aaaa bbbb cccc\ndddd 1990. eeee\n"},
	{"hashes do not start a row", "aaaa bbbb cccc dddd ## eeee\n", 20, "aaaa bbbb cccc\ndddd ## eeee\n"},
	{"a quote marker does not start a row", "aaaa bbbb cccc dddd >x eeee\n", 20, "aaaa bbbb cccc\ndddd >x eeee\n"},
	{"a pipe does not start a row", "aaaa bbbb cccc dddd |x| eeee\n", 20, "aaaa bbbb cccc\ndddd |x| eeee\n"},
	{"a fence does not start a row", "aaaa bbbb cccc dddd ```x eeee\n", 20, "aaaa bbbb cccc\ndddd ```x eeee\n"},
	{"a rule does not start a row", "aaaa bbbb cccc dddd *** eeee\n", 20, "aaaa bbbb cccc\ndddd *** eeee\n"},
	{"fenced code passes through", "```go\nfunc main() { fmt.Println(\"a long line of code\") }\n```\nalpha beta gamma delta\n", 20,
		"```go\nfunc main() { fmt.Println(\"a long line of code\") }\n```\nalpha beta gamma\ndelta\n"},
	{"a tilde fence, longer run", "~~~~\nlong code line here that exceeds\n~~~\nstill code inside here ok\n~~~~\n", 20,
		"~~~~\nlong code line here that exceeds\n~~~\nstill code inside here ok\n~~~~\n"},
	{"backticks closed on their line are inline code, not a fence", "```a``` then some words that wrap here\n", 20,
		"```a``` then some\nwords that wrap here\n"},
	{"an unclosed fence passes through", "```\nlong code line here that exceeds the width", 20,
		"```\nlong code line here that exceeds the width"},
	{"indented code passes through", "    indented code line that is long enough\n", 20,
		"    indented code line that is long enough\n"},
	{"a tab-indented line passes through", "\tindented code line that is long enough\n", 20,
		"\tindented code line that is long enough\n"},
	{"a table passes through", "| a | b |\n|---|---|\n| long cell value here | other long cell |\n", 20,
		"| a | b |\n|---|---|\n| long cell value here | other long cell |\n"},
	{"a heading passes through", "## A heading that is longer than twenty\n", 20, "## A heading that is longer than twenty\n"},
	{"blank lines and line ends are kept", "one two\n\nthree four\nfive\n", 20, "one two\n\nthree four\nfive\n"},
	{"a hard line end is not joined", "alpha beta gamma delta\nepsilon\n", 20, "alpha beta gamma\ndelta\nepsilon\n"},
	{"no final line end", "alpha beta gamma delta", 20, "alpha beta gamma\ndelta"},
	{"spaces between words are kept", "a  b   c\n", 20, "a  b   c\n"},
	{"a row exactly the width", "aaaaaaaaa bbbbbbbbbb\n", 20, "aaaaaaaaa bbbbbbbbbb\n"},
	{"no width, no wrapping", "The quick brown fox jumps over the lazy dog.\n", 0,
		"The quick brown fox jumps over the lazy dog.\n"},
}

// wrapAll feeds chunks to a new Wrap in turn and flushes it.
func wrapAll(chunks []string, width int) string {
	var w Wrap
	var b strings.Builder
	for _, c := range chunks {
		b.WriteString(w.Push(c, width))
	}
	b.WriteString(w.Flush())
	return b.String()
}

func TestWrap(t *testing.T) {
	for _, tc := range wrapCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := wrapAll([]string{tc.in}, tc.width); got != tc.want {
				t.Errorf("got\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}

// Any split of a reply into chunks gives the rows the whole reply gives:
// every split in two, and random splits into many (0028-MADR D5).
func TestWrapAnySplit(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for _, tc := range wrapCases {
		var cuts []int // rune boundaries
		for i := range tc.in {
			cuts = append(cuts, i)
		}
		for _, c := range cuts[1:] {
			if got := wrapAll([]string{tc.in[:c], tc.in[c:]}, tc.width); got != tc.want {
				t.Fatalf("%s, split at %d: got %q, want %q", tc.name, c, got, tc.want)
			}
		}
		for range 50 {
			var chunks []string
			for s := tc.in; s != ""; {
				n := 1 + r.IntN(6)
				end := 0
				for k := 0; k < n && end < len(s); k++ {
					_, size := utf8.DecodeRuneInString(s[end:])
					end += size
				}
				chunks = append(chunks, s[:end])
				s = s[end:]
			}
			if got := wrapAll(chunks, tc.width); got != tc.want {
				t.Fatalf("%s, chunks %q: got %q, want %q", tc.name, chunks, got, tc.want)
			}
		}
	}
}

// splitEscapes parts s into its escape sequences, in order, and the rest.
func splitEscapes(s string) (escapes, plain string) {
	var e, p strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			e.WriteString(s[i:min(j+1, len(s))])
			i = j
			continue
		}
		p.WriteByte(s[i])
	}
	return e.String(), p.String()
}

// Styled text wraps as its plain text does: with the escapes taken out, the
// rows are the plain text's, and every escape is kept, in order, whatever
// the chunks (0028-MADR D7, as amended).
func TestWrapStyled(t *testing.T) {
	on := term.Stream{Color: true}
	cases := append(slices.Clip(wrapCases), struct {
		name, in string
		width    int
		want     string
	}{"emphasis across a break", "this is **very bold** and `some code` here\n", 16, ""})
	for _, tc := range cases {
		plainIn := Tables(tc.in)
		styled := Style(plainIn, on)
		want := wrapAll([]string{plainIn}, tc.width)
		wantEsc, _ := splitEscapes(styled)
		splits := [][]string{{styled}}
		for c := range styled {
			if c > 0 {
				splits = append(splits, []string{styled[:c], styled[c:]})
			}
		}
		for _, chunks := range splits {
			got := wrapAll(chunks, tc.width)
			esc, plain := splitEscapes(got)
			if plain != want || esc != wantEsc {
				t.Fatalf("%s, chunks %q:\ngot   %q\nplain %q, want %q\nescapes %q, want %q", tc.name, chunks, got, plain,
					want, esc, wantEsc)
			}
		}
	}
}

// A push releases all it can: only the word still arriving and the last
// finished word on the row are held (0028-MADR D5, as amended).
func TestWrapStreams(t *testing.T) {
	var w Wrap
	for _, st := range []struct{ in, want string }{
		{"alpha beta gam", "alpha"},
		{"ma ", " beta"},
		{"delta\n", " gamma delta\n"},
		{"- one two", "- "},
	} {
		if got := w.Push(st.in, 40); got != st.want {
			t.Fatalf("Push(%q) = %q, want %q", st.in, got, st.want)
		}
	}
	if got := w.Flush(); got != "one two" {
		t.Fatalf("Flush() = %q, want %q", got, "one two")
	}
}
