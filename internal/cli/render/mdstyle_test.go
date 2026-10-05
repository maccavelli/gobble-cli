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
