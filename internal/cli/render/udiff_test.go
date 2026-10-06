package render

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

func lines(n int, f func(i int) string) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString(f(i) + "\n")
	}
	return b.String()
}

func TestUnifiedDiff(t *testing.T) {
	old := lines(20, func(i int) string { return fmt.Sprintf("line %d", i) })
	changed := strings.Replace(strings.Replace(old, "line 3\n", "line three\n", 1), "line 18\n", "", 1)
	cases := []struct {
		name     string
		old      *string
		new      string
		filename string
	}{
		{"two hunks", &old, changed, "udiff-two-hunks"},
		{"new file", nil, "a\nb\n", "udiff-new-file"},
		{"no final newline", new("a\nb"), "a\nc", "udiff-no-newline"},
		{"neighbours share a hunk", &old, strings.Replace(strings.Replace(old, "line 5\n", "five\n", 1), "line 9\n", "nine\n", 1), "udiff-one-hunk"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			golden(t, tc.filename, UnifiedDiff("src/a.go", tc.old, tc.new))
		})
	}
	if got := UnifiedDiff("a", &old, old); got != "" {
		t.Fatalf("equal texts gave %q", got)
	}
}

// apply rebuilds new from old and a diff's ops, so any script myers gives
// can be checked against both texts.
func apply(ops []op) (a, b string) {
	var x, y strings.Builder
	for _, o := range ops {
		if o.kind != opIns {
			x.WriteString(o.text)
		}
		if o.kind != opDel {
			y.WriteString(o.text)
		}
	}
	return x.String(), y.String()
}

// The edit script turns old into new, for random texts, and is minimal
// against a known case.
func TestMyersScripts(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	word := func() string { return []string{"a\n", "b\n", "c\n", "d\n"}[r.IntN(4)] }
	for range 500 {
		var x, y strings.Builder
		for range r.IntN(12) {
			x.WriteString(word())
		}
		for range r.IntN(12) {
			y.WriteString(word())
		}
		ga, gb := apply(myers(splitLines(x.String()), splitLines(y.String())))
		if ga != x.String() || gb != y.String() {
			t.Fatalf("script for %q → %q rebuilds %q → %q", x.String(), y.String(), ga, gb)
		}
	}
	// ABCABBA → CBABAC has an edit distance of 5 (Myers's own example).
	split := func(s string) []string { return strings.Split(s, "") }
	edits := 0
	for _, o := range myers(split("ABCABBA"), split("CBABAC")) {
		if o.kind != opEqual {
			edits++
		}
	}
	if edits != 5 {
		t.Fatalf("edit distance %d, want 5", edits)
	}
}
