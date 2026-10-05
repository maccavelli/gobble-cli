package term

import (
	"reflect"
	"testing"
	"unicode/utf8"
)

func TestStyleWrap(t *testing.T) {
	cases := []struct {
		name  string
		style Style
		on    bool
		want  string
	}{
		{"off", Bold | Red, false, "x"},
		{"empty style", 0, true, "x"},
		{"bold red", Bold | Red, true, "\x1b[1;31mx\x1b[0m"},
		{"dim", Dim, true, "\x1b[2mx\x1b[0m"},
		{"underline white", Underline | White, true, "\x1b[4;37mx\x1b[0m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.style.Wrap("x", tc.on); got != tc.want {
				t.Fatalf("Wrap = %q, want %q", got, tc.want)
			}
		})
	}
	if got := (Bold).Wrap("", true); got != "" {
		t.Fatalf("empty text wrapped: %q", got)
	}
}

func TestGlyphSets(t *testing.T) {
	for name, g := range map[string]Glyphs{"Unicode": Unicode, "ASCII": ASCII} {
		v := reflect.ValueOf(g)
		for i := range v.NumField() {
			s := v.Field(i).String()
			if s == "" {
				t.Errorf("%s.%s is empty", name, v.Type().Field(i).Name)
			}
			if name == "ASCII" {
				for _, r := range s {
					if r >= utf8.RuneSelf {
						t.Errorf("ASCII.%s = %q is not ASCII", v.Type().Field(i).Name, s)
					}
				}
			}
		}
	}
}
