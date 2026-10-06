package render

import (
	"os"
	"strings"
	"testing"
)

func TestStyleGolden(t *testing.T) {
	in, err := os.ReadFile("testdata/style.input.txt")
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "style", Style(string(in), colorOn))
}

func TestStyleColourOffIsByteForByte(t *testing.T) {
	in, err := os.ReadFile("testdata/style.input.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got := Style(string(in), colorOff); got != string(in) {
		t.Fatalf("colour off changed the text:\n%q", got)
	}
}

func TestStyleKeepsEveryByte(t *testing.T) {
	in, err := os.ReadFile("testdata/style.input.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got := stripSGR(Style(string(in), colorOn)); got != string(in) {
		t.Fatalf("styled text minus SGR differs from the input:\n%q", got)
	}
}

func TestStyleCases(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"heading", "## Title\n", "\x1b[1m## Title\x1b[0m\n"},
		{"not a heading", "#hashtag\n", "#hashtag\n"},
		{"list marker", "- item\n", "\x1b[36m-\x1b[0m item\n"},
		{"ordered marker", "  12. item\n", "  \x1b[36m12.\x1b[0m item\n"},
		{"quote marker", "> said\n", "\x1b[36m>\x1b[0m said\n"},
		{"inline code", "run `go test` now\n", "run \x1b[2m`go test`\x1b[0m now\n"},
		{"unclosed inline code", "a `b\n", "a `b\n"},
		{"fence with label", "```go\nx := 1\n```\n", "\x1b[2m```\x1b[0m\x1b[36mgo\x1b[0m\nx := 1\n\x1b[2m```\x1b[0m\n"},
		{"code inside a fence is not styled", "```\n# not a heading\n- nor a list\n```\n",
			"\x1b[2m```\x1b[0m\n# not a heading\n- nor a list\n\x1b[2m```\x1b[0m\n"},
		{"no trailing newline", "# T", "\x1b[1m# T\x1b[0m"},
		// Emphasis (0008-PLAN P8).
		{"bold", "a **b c** d\n", "a \x1b[1m**b c**\x1b[0m d\n"},
		{"bold underscores", "__b__\n", "\x1b[1m__b__\x1b[0m\n"},
		{"italic", "an *it* and _it_\n", "an \x1b[3m*it*\x1b[0m and \x1b[3m_it_\x1b[0m\n"},
		{"bold italic", "***both***\n", "\x1b[1;3m***both***\x1b[0m\n"},
		{"strikethrough", "~~gone~~ ~not~\n", "\x1b[2m~~gone~~\x1b[0m ~not~\n"},
		{"snake_case stays plain", "call snake_case_name now\n", "call snake_case_name now\n"},
		{"spaced stars stay plain", "2 * 3 * 4\n", "2 * 3 * 4\n"},
		{"unclosed", "**open and *half\n", "**open and *half\n"},
		{"escaped", "\\*not\\* *yes*\n", "\\*not\\* \x1b[3m*yes*\x1b[0m\n"},
		{"not nested", "**a *b* c**\n", "\x1b[1m**a *b* c**\x1b[0m\n"},
		{"beside a code span", "`a*b*` and *c*\n", "\x1b[2m`a*b*`\x1b[0m and \x1b[3m*c*\x1b[0m\n"},
		{"not across a code span (a subset of CommonMark)", "*a `b` c*\n", "*a \x1b[2m`b`\x1b[0m c*\n"},
		{"a star list marker then emphasis", "* *it*\n", "\x1b[36m*\x1b[0m \x1b[3m*it*\x1b[0m\n"},
		{"a lone star marker is not an opener", "* item\n", "\x1b[36m*\x1b[0m item\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Style(tc.in, colorOn); got != tc.want {
				t.Fatalf("Style(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// stripSGR removes ESC [ … m sequences.
func stripSGR(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "\x1b[")
		if i < 0 {
			return b.String() + s
		}
		b.WriteString(s[:i])
		j := strings.IndexByte(s[i:], 'm')
		if j < 0 {
			return b.String() + s[i:]
		}
		s = s[i+j+1:]
	}
}
