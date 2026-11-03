package fsx

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

// ErrBadGlob is the error of a glob with a malformed character class.
var ErrBadGlob = errors.New("malformed glob")

// maxAlternatives bounds how many patterns a glob's {a,b} groups may expand
// to, so a pattern cannot make matching exponential.
const maxAlternatives = 256

// Glob is a compiled pattern for a tool's globs, in gitignore's syntax
// (git-scm.com/docs/gitignore): "*" and "?" never match "/", "[…]" is a
// class, "\" escapes, and "**" matches across directories. On top of git's
// rules, {a,b} alternatives are expanded first. A pattern without a "/"
// matches a name at any depth; one with a "/" matches the path from the
// search root (0005-PLAN, entry "F1c made executable").
type Glob struct {
	alts []matcher
}

// CompileGlob compiles pattern. A malformed character class is an error.
func CompileGlob(pattern string) (Glob, error) {
	budget := maxAlternatives
	alts := expandBraces(pattern, &budget)
	if budget < 0 {
		return Glob{}, fmt.Errorf("%w: %q has more than %d alternatives", ErrBadGlob, pattern, maxAlternatives)
	}
	g := Glob{alts: make([]matcher, 0, len(alts))}
	for _, p := range alts {
		m, err := compile(p)
		if err != nil {
			return Glob{}, err
		}
		g.alts = append(g.alts, m)
	}
	return g, nil
}

// Match reports whether rel, a slash-separated relative path, matches. dir
// says rel is a directory, which a pattern ending in "/" requires.
func (g Glob) Match(rel string, dir bool) bool {
	for _, m := range g.alts {
		if m.match(rel, dir) {
			return true
		}
	}
	return false
}

// matcher is one pattern, as path segments. A segment "**" matches any
// number of whole segments; a trailing one matches at least one, since
// "abc/**" matches what is inside abc and not abc itself.
type matcher struct {
	segs    []string
	dirOnly bool
	never   bool // git: a pattern ending in a lone "\" never matches
}

// compile is one pattern, with git's rules and no {a,b} expansion: the form
// gitignore files use.
func compile(p string) (matcher, error) {
	if trailingEscape(p) {
		return matcher{never: true}, nil
	}
	var m matcher
	if strings.HasSuffix(p, "/") {
		m.dirOnly = true
		p = strings.TrimRight(p, "/")
	}
	anchored := strings.Contains(p, "/")
	p = strings.TrimPrefix(p, "/")
	segs := strings.Split(p, "/")
	if !anchored {
		segs = append([]string{"**"}, segs...)
	}
	for i, s := range segs {
		if s == "**" {
			continue
		}
		s = gitClass(s)
		if _, err := path.Match(s, ""); err != nil {
			return matcher{}, fmt.Errorf("%w: %q", ErrBadGlob, p)
		}
		segs[i] = s
	}
	m.segs = collapse(segs)
	return m, nil
}

// collapse is segs with each run of "**" made one, as git's wildmatch does:
// a run matches what one does (0037-MADR D11).
func collapse(segs []string) []string {
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if s == "**" && len(out) > 0 && out[len(out)-1] == "**" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (m matcher) match(rel string, dir bool) bool {
	if m.never || (m.dirOnly && !dir) {
		return false
	}
	return matchSegs(m.segs, strings.Split(rel, "/"))
}

// matchSegs matches pattern segments against name segments, "**" taking
// any number of them and a trailing one at least one. It fills a table of
// whether pat[i:] matches name[j:] from the end, so each pair is decided
// once: O(len(pat)·len(name)) (0037-MADR D11).
func matchSegs(pat, name []string) bool {
	np, nn := len(pat), len(name)
	w := nn + 1
	t := make([]bool, (np+1)*w)
	t[np*w+nn] = true
	for i := np - 1; i >= 0; i-- {
		row, next := t[i*w:(i+1)*w], t[(i+1)*w:(i+2)*w]
		for j := nn; j >= 0; j-- {
			switch {
			case pat[i] == "**" && i == np-1:
				row[j] = j < nn
			case pat[i] == "**":
				row[j] = next[j] || (j < nn && row[j+1])
			case j < nn && next[j+1]:
				// compile checked every segment, so Match cannot fail here.
				ok, err := path.Match(pat[i], name[j])
				row[j] = err == nil && ok
			}
		}
	}
	return t[0]
}

// trailingEscape reports whether p ends in a backslash that escapes
// nothing.
func trailingEscape(p string) bool {
	n := 0
	for i := len(p) - 1; i >= 0 && p[i] == '\\'; i-- {
		n++
	}
	return n%2 == 1
}

// gitClass rewrites git's "[!…]" negation to the "[^…]" path.Match reads.
func gitClass(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		b.WriteByte(c)
		switch {
		case c == '\\' && i+1 < len(s):
			i++
			b.WriteByte(s[i])
		case c == '[' && i+1 < len(s) && s[i+1] == '!':
			i++
			b.WriteByte('^')
		}
	}
	return b.String()
}

// expandBraces is p with its first {a,b,…} group expanded, recursively, so
// every group is. A group with no top-level comma, and an unclosed "{", are
// literal. budget counts the patterns made, and goes negative past
// maxAlternatives.
func expandBraces(p string, budget *int) []string {
	open, depth := -1, 0
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case '\\':
			i++
		case '{':
			if depth == 0 {
				open = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth > 0 {
				continue
			}
			parts := splitTop(p[open+1 : i])
			if len(parts) < 2 {
				var out []string
				for _, rest := range expandBraces(p[i+1:], budget) {
					out = append(out, p[:i+1]+rest)
				}
				return out
			}
			var out []string
			for _, part := range parts {
				if *budget < 0 {
					return out
				}
				out = append(out, expandBraces(p[:open]+part+p[i+1:], budget)...)
			}
			return out
		}
	}
	*budget--
	return []string{p}
}

// splitTop splits s at the commas outside any nested group.
func splitTop(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}
