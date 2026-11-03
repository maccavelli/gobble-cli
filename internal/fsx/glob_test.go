package fsx

import (
	"errors"
	"path"
	"strings"
	"testing"
	"time"
)

// The examples of git's PATTERN FORMAT (git-scm.com/docs/gitignore), and
// the rest of its syntax, with {a,b} on top for tool globs.
func TestGlob(t *testing.T) {
	cases := []struct {
		pattern string
		rel     string
		dir     bool
		want    bool
	}{
		// "doc/frotz/ matches doc/frotz directory, but not a/doc/frotz"
		{"doc/frotz/", "doc/frotz", true, true},
		{"doc/frotz/", "a/doc/frotz", true, false},
		{"doc/frotz/", "doc/frotz", false, false},
		// "frotz/ matches frotz and a/frotz that is a directory"
		{"frotz/", "frotz", true, true},
		{"frotz/", "a/frotz", true, true},
		{"frotz/", "a/frotz", false, false},
		// "**/foo matches file or directory foo anywhere"
		{"**/foo", "foo", false, true},
		{"**/foo", "x/y/foo", true, true},
		// "**/foo/bar matches bar anywhere that is directly under directory foo"
		{"**/foo/bar", "foo/bar", false, true},
		{"**/foo/bar", "x/foo/bar", false, true},
		{"**/foo/bar", "foo/x/bar", false, false},
		// "abc/** matches all files inside directory abc"
		{"abc/**", "abc/x", false, true},
		{"abc/**", "abc/x/y", false, true},
		{"abc/**", "abc", true, false},
		// "a/**/b matches a/b, a/x/b, a/x/y/b"
		{"a/**/b", "a/b", false, true},
		{"a/**/b", "a/x/b", false, true},
		{"a/**/b", "a/x/y/b", false, true},
		{"a/**/b", "b", false, false},
		// a pattern without a slash matches at any level
		{"*.go", "main.go", false, true},
		{"*.go", "cmd/x/main.go", false, true},
		// a leading or middle slash anchors it
		{"/main.go", "main.go", false, true},
		{"/main.go", "cmd/main.go", false, false},
		{"cmd/*.go", "cmd/main.go", false, true},
		{"cmd/*.go", "x/cmd/main.go", false, false},
		// "*" and "?" never match "/"
		{"a*b", "a/b", false, false},
		{"a?b", "a/b", false, false},
		{"a?b", "axb", false, true},
		// other consecutive asterisks are regular
		{"a**b", "axxb", false, true},
		{"a**b", "a/b", false, false},
		// ranges, and git's [!…]
		{"file[0-9].txt", "file7.txt", false, true},
		{"file[!0-9].txt", "file7.txt", false, false},
		{"file[!0-9].txt", "filex.txt", false, true},
		// escapes
		{`\*.txt`, "*.txt", false, true},
		{`\*.txt`, "a.txt", false, false},
		{`\#notes`, "#notes", false, true},
		// a trailing lone backslash never matches
		{`abc\`, `abc\`, false, false},
		{`abc\`, "abc", false, false},
		// braces, nested braces, and an unclosed or comma-less group
		{"*.{ts,tsx}", "a.ts", false, true},
		{"*.{ts,tsx}", "a.tsx", false, true},
		{"*.{ts,tsx}", "a.js", false, false},
		{"{src,lib}/**/*.{go,{c,h}}", "lib/x/y.h", false, true},
		{"{src,lib}/**/*.{go,{c,h}}", "test/y.go", false, false},
		{"a{b", "a{b", false, true},
		{"{x}.txt", "{x}.txt", false, true},
	}
	for _, c := range cases {
		g, err := CompileGlob(c.pattern)
		if err != nil {
			t.Fatalf("CompileGlob(%q): %v", c.pattern, err)
		}
		if got := g.Match(c.rel, c.dir); got != c.want {
			t.Errorf("%q matching %q (dir %v) = %v, want %v", c.pattern, c.rel, c.dir, got, c.want)
		}
	}
}

// A malformed class is an error, and so are too many alternatives.
func TestGlobErrors(t *testing.T) {
	for _, p := range []string{"file[.txt", "{a,b}{a,b}{a,b}{a,b}{a,b}{a,b}{a,b}{a,b}{a,b}"} {
		if _, err := CompileGlob(p); !errors.Is(err, ErrBadGlob) {
			t.Errorf("CompileGlob(%q) = %v, want ErrBadGlob", p, err)
		}
	}
}

// F10's patterns match as before, each within 50 ms: a run of "**" is one
// segment once compiled, and a "**" between others is matched once per
// pair of positions (0037-MADR D11).
func TestGlobLinear(t *testing.T) {
	g, err := CompileGlob("**/**/**/**/**/**/**/**/**/z")
	if err != nil {
		t.Fatal(err)
	}
	if segs := g.alts[0].segs; len(segs) != 2 {
		t.Errorf("the run of nine compiled to %q, want [** z]", segs)
	}
	for _, c := range []struct {
		pattern string
		rel     string
		want    bool
	}{
		{"**/**/**/**/**/**/**/**/**/z", strings.Repeat("a/", 30) + "b", false},
		{"**/**/**/**/**/**/**/**/**/z", strings.Repeat("a/", 30) + "z", true},
		{strings.Repeat("**/a/", 6) + "z", strings.Repeat("a/", 40) + "a", false},
		{strings.Repeat("**/a/", 6) + "z", strings.Repeat("a/", 40) + "z", true},
	} {
		g, err := CompileGlob(c.pattern)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		got := g.Match(c.rel, false)
		if d := time.Since(start); d > 50*time.Millisecond {
			t.Errorf("%q against %d segments took %v, want within 50ms", c.pattern, strings.Count(c.rel, "/")+1, d)
		}
		if got != c.want {
			t.Errorf("%q against %d segments = %v, want %v", c.pattern, strings.Count(c.rel, "/")+1, got, c.want)
		}
	}
}

// Runs of "**", a trailing "**" and "**" between others match as they did
// before 0037-PLAN P10: a table, and every short pattern against every
// short path, compared with the recursive match it replaced.
func TestGlobMatchesUnchanged(t *testing.T) {
	cases := []struct {
		pattern string
		rel     string
		dir     bool
		want    bool
	}{
		{"**", "a", false, true},
		{"**", "a/b/c", false, true},
		{"**/", "a", true, true},
		{"**/", "a", false, false},
		{"a/**/**", "a", true, false},
		{"a/**/**", "a/b", false, true},
		{"a/**/**/b", "a/b", false, true},
		{"a/**/**/b", "a/x/y/b", false, true},
		{"**/**/a", "a", false, true},
		{"**/**/a", "x/y/a", false, true},
		{"/**/a", "a", false, true},
		{"x/**/y/**/z", "x/y/z", false, true},
		{"x/**/y/**/z", "x/1/y/2/3/z", false, true},
		{"x/**/y/**/z", "x/1/z/2/y", false, false},
		{"a/**/b/**", "a/b", true, false},
		{"a/**/b/**", "a/q/b/c", false, true},
		{"*/**/*.go", "main.go", false, false},
		{"*/**/*.go", "cmd/main.go", false, true},
		{"*/**/*.go", "cmd/a/b/main.go", false, true},
		{"{a,b}/**/c", "b/x/c", false, true},
	}
	for _, c := range cases {
		g, err := CompileGlob(c.pattern)
		if err != nil {
			t.Fatalf("CompileGlob(%q): %v", c.pattern, err)
		}
		if got := g.Match(c.rel, c.dir); got != c.want {
			t.Errorf("%q matching %q (dir %v) = %v, want %v", c.pattern, c.rel, c.dir, got, c.want)
		}
	}

	segs := []string{"**", "a", "*", "b"}
	var pats [][]string
	var grow func(p []string)
	grow = func(p []string) {
		pats = append(pats, p)
		if len(p) < 5 {
			for _, s := range segs {
				grow(append(append([]string(nil), p...), s))
			}
		}
	}
	grow(nil)
	var names [][]string
	for n := range 7 {
		for bits := range 1 << n {
			name := make([]string, n)
			for i := range name {
				name[i] = string("ab"[bits>>i&1])
			}
			names = append(names, name)
		}
	}
	for _, p := range pats {
		for _, name := range names {
			if got, want := matchSegs(collapse(p), name), recursiveSegs(p, name); got != want {
				t.Fatalf("%q against %q = %v, want %v", p, name, got, want)
			}
		}
	}
}

// recursiveSegs is matchSegs as it was before 0037-PLAN P10, uncollapsed,
// for TestGlobMatchesUnchanged to compare with.
func recursiveSegs(pat, name []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			rest := pat[1:]
			if len(rest) == 0 {
				return len(name) > 0
			}
			for i := 0; i <= len(name); i++ {
				if recursiveSegs(rest, name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if ok, err := path.Match(pat[0], name[0]); err != nil || !ok {
			return false
		}
		pat, name = pat[1:], name[1:]
	}
	return len(name) == 0
}
