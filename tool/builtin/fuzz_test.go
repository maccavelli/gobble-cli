package builtin

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzEditMatch: applyEdits never panics, and an edit it accepts leaves
// every newText in the result and changes the content.
func FuzzEditMatch(f *testing.F) {
	for _, s := range [][3]string{
		{"hello world\n", "world", "there"},
		{"a\nb\nc\n", "b\n", "B\n"},
		{"line one   \nline two\n", "line one\n", "x\n"},
		{"console.log(\u2018hi\u2019);\n", "console.log('hi');", "y"},
		{"\uFF21\uFF22\n", "AB", "ab"},
		{"x x", "x", "y"},
		{"", "a", "b"},
	} {
		f.Add(s[0], s[1], s[2])
	}
	f.Fuzz(func(t *testing.T, content, oldText, newText string) {
		if !utf8.ValidString(content) || !utf8.ValidString(oldText) || !utf8.ValidString(newText) {
			return
		}
		lf := toLF(content)
		out, err := applyEdits(lf, []edit{{OldText: oldText, NewText: newText}}, "f")
		if err != nil {
			return
		}
		if out == lf {
			t.Fatalf("accepted an edit that changed nothing: %q", out)
		}
		if !strings.Contains(out, toLF(newText)) {
			t.Fatalf("result %q lacks newText %q", out, newText)
		}
	})
}
