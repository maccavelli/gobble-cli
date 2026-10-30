package term

import (
	"testing"
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
