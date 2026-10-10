package frontmatter

import (
	"reflect"
	"testing"
)

// Pi's parseFrontmatter cases: a block scalar, quoted strings, a nested
// map, CRLF and a byte-order mark, no frontmatter, an unclosed one, an
// empty one, and YAML that is not a mapping.
func TestParse(t *testing.T) {
	for _, c := range []struct {
		name, text string
		fields     map[string]any
		body       string
		err        bool
	}{
		{"block scalar", "---\ndescription: |\n  one\n  two\nname: x\n---\n\nBody.\n\n", map[string]any{"description": "one\ntwo\n", "name": "x"}, "Body.", false},
		{"quotes and a map", "---\na: \"double 'q'\"\nb: 'single \"q\"'\nmetadata:\n  k: v\n---\nB", map[string]any{"a": "double 'q'", "b": `single "q"`, "metadata": map[string]any{"k": "v"}}, "B", false},
		{"CRLF and BOM", "\ufeff---\r\nname: y\r\n---\r\nBody\r\n", map[string]any{"name": "y"}, "Body", false},
		{"CRLF in the body", "---\r\nname: y\r\n---\r\nLine one\r\nLine two\r\n", map[string]any{"name": "y"}, "Line one\nLine two", false},
		{"CR alone", "---\rname: y\r---\rBody", map[string]any{"name": "y"}, "Body", false},
		{"none", "Just text.\n", map[string]any{}, "Just text.\n", false},
		{"unclosed", "---\nname: z\nno end\n", map[string]any{}, "---\nname: z\nno end\n", false},
		{"empty", "---\n---\nBody", map[string]any{}, "Body", false},
		{"a scalar", "---\njust words\n---\nBody", map[string]any{}, "Body", false},
		{"bad YAML", "---\na: [1\n---\nBody", map[string]any{}, "Body", true},
		{"JavaScript's trim", "---\nname: y\n---\n\ufeff \u0085Body\u0085\u3000\ufeff", map[string]any{"name": "y"}, "\u0085Body\u0085", false},
	} {
		fields, body, err := Parse(c.text)
		if (err != nil) != c.err || body != c.body || !reflect.DeepEqual(fields, c.fields) {
			t.Errorf("%s: fields %v, body %q, err %v", c.name, fields, body, err)
		}
	}
}

// JavaScript's whitespace: its \s set, which has U+FEFF and not U+0085,
// unlike Go's unicode.IsSpace.
func TestIsSpace(t *testing.T) {
	for _, r := range "\t\n\v\f\r \u00a0\u1680\u2000\u2005\u200a\u2028\u2029\u202f\u205f\u3000\ufeff" {
		if !IsSpace(r) {
			t.Errorf("%U is JavaScript whitespace", r)
		}
	}
	for _, r := range "\u0085\u200bxa\u180e" {
		if IsSpace(r) {
			t.Errorf("%U is not JavaScript whitespace", r)
		}
	}
	if got := Trim("\ufeff\t x y \u0085\u3000"); got != "x y \u0085" {
		t.Errorf("Trim: %q", got)
	}
}
