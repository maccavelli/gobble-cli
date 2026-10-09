package fsx

import (
	"errors"
	"testing"
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
