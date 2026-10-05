package complete

import (
	"slices"
	"testing"
)

func TestSplitLine(t *testing.T) {
	cases := []struct {
		name  string
		line  string
		point int
		words []string
		cur   string
	}{
		{"plain", "gobble --model an", -1, []string{"gobble", "--model"}, "an"},
		{"trailing space", "gobble --model ", -1, []string{"gobble", "--model"}, ""},
		{"single quotes", `gobble 'a b' c`, -1, []string{"gobble", "a b"}, "c"},
		{"double quotes and escapes", `gobble "a \"b\" \$c" d`, -1, []string{"gobble", `a "b" $c`}, "d"},
		{"backslash outside quotes", `gobble a\ b`, -1, []string{"gobble"}, "a b"},
		{"open single quote", `gobble --file 'my fi`, -1, []string{"gobble", "--file"}, "my fi"},
		{"open double quote", `gobble "abc`, -1, []string{"gobble"}, "abc"},
		{"no expansion", `gobble $HOME ~/x *`, -1, []string{"gobble", "$HOME", "~/x"}, "*"},
		{"empty quotes are a word", `gobble '' x`, -1, []string{"gobble", ""}, "x"},
		{"cut at the cursor", "gobble --output-format json", 25, []string{"gobble", "--output-format"}, "js"},
		{"point past the end", "gobble x", 99, []string{"gobble"}, "x"},
		// point is a byte offset: "é" is two bytes.
		{"multibyte cut", "gobble --file dés.txt", 17, []string{"gobble", "--file"}, "dé"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			words, cur := SplitLine(tc.line, tc.point)
			if !slices.Equal(words, tc.words) || cur != tc.cur {
				t.Fatalf("SplitLine(%q, %d) = %q, %q; want %q, %q", tc.line, tc.point, words, cur, tc.words, tc.cur)
			}
		})
	}
}
