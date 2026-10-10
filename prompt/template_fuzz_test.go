package prompt

import (
	"strings"
	"testing"
)

// FuzzTemplate: ParseArgs and Substitute never panic; ParseArgs gives no
// empty argument, as Pi's skips an empty quote; content without "$" comes
// back unchanged; and an argument is placed as it is, its own $1 or $@
// never substituted in turn (0005-PLAN F3-3).
func FuzzTemplate(f *testing.F) {
	for _, s := range []struct{ content, args string }{
		{"$1 $2 ${@:2} ${@:1:1} ${1:-d} ${@:-all} $ARGUMENTS $@", `a "b c" 'd e' f`},
		{"${@:0:99999999999999999999} $99999999999999999999", "$1 $@ ${@:2}"},
		{"no placeholders", "x y"},
		{"${ ${1 ${@: $ $$", "\"unterminated 'quote"},
		{"$1$2", "\"\"  x  ''"},
		{"\u3000$1\u0085", "a\u3000b\u0085c\ufeffd"},
	} {
		f.Add(s.content, s.args)
	}
	f.Fuzz(func(t *testing.T, content, argText string) {
		args := ParseArgs(argText)
		for _, a := range args {
			if a == "" {
				t.Fatalf("ParseArgs(%q) gave an empty argument: %q", argText, args)
			}
		}
		got := Substitute(content, args)
		if !strings.Contains(content, "$") && got != content {
			t.Fatalf("Substitute(%q) = %q, with no placeholder", content, got)
		}
		if got := Substitute("[$@]", args); got != "["+strings.Join(args, " ")+"]" {
			t.Fatalf("$@ of %q = %q", args, got)
		}
		first := ""
		if len(args) > 0 {
			first = args[0]
		}
		if got := Substitute("$1", args); got != first {
			t.Fatalf("$1 of %q = %q", args, got)
		}
	})
}
