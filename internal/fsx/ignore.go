package fsx

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Ignore is a repository's ignore rules for one walk: info/exclude, and
// each .gitignore read so far, in git's precedence (git-scm.com/docs/
// gitignore). Paths are slash-separated, from the repository's top level.
type Ignore struct {
	sets []ruleSet // lowest precedence first
}

// ruleSet is one ignore file's rules, relative to base: "" for the top
// level. depth orders the sets, a lower .gitignore overriding a higher one,
// and info/exclude (depth -1) below them all.
type ruleSet struct {
	base  string
	depth int
	rules []rule
}

type rule struct {
	m      matcher
	negate bool
}

// Ignored reports whether rel is ignored: by the last rule that matches it,
// unless a directory above it is ignored, which no rule can undo.
func (ig *Ignore) Ignored(rel string, dir bool) bool {
	parts := strings.Split(rel, "/")
	for i := 1; i < len(parts); i++ {
		if ig.match(strings.Join(parts[:i], "/"), true) {
			return true
		}
	}
	return ig.match(rel, dir)
}

func (ig *Ignore) match(rel string, dir bool) bool {
	ignored := false
	for _, s := range ig.sets {
		sub, ok := under(s.base, rel)
		if !ok {
			continue
		}
		for _, r := range s.rules {
			if r.m.match(sub, dir) {
				ignored = !r.negate
			}
		}
	}
	return ignored
}

// add adds the rules of an ignore file read at base.
func (ig *Ignore) add(base string, depth int, text string) {
	rules := parseIgnore(text)
	if len(rules) == 0 {
		return
	}
	// After every set of the same or a lower depth, so equal depths keep
	// the order they were added in.
	i := len(ig.sets)
	for i > 0 && ig.sets[i-1].depth > depth {
		i--
	}
	ig.sets = slices.Insert(ig.sets, i, ruleSet{base: base, depth: depth, rules: rules})
}

// under is rel relative to base, when rel is inside it.
func under(base, rel string) (string, bool) {
	if base == "" {
		return rel, true
	}
	if rest, ok := strings.CutPrefix(rel, base+"/"); ok {
		return rest, true
	}
	return "", false
}

// parseIgnore is an ignore file's rules, by git's PATTERN FORMAT: blank
// lines and "#" comments are skipped, trailing spaces are dropped unless
// escaped, "!" negates, and "\#" and "\!" begin a pattern literally. A
// malformed pattern matches nothing, as git treats it.
func parseIgnore(text string) []rule {
	text = strings.TrimPrefix(text, "\ufeff")
	var out []rule
	for line := range strings.SplitSeq(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = trimTrailingSpaces(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		negate := false
		switch {
		case strings.HasPrefix(line, "!"):
			negate, line = true, line[1:]
		case strings.HasPrefix(line, `\#`), strings.HasPrefix(line, `\!`):
			line = line[1:]
		}
		m, err := compile(line)
		if err != nil {
			continue
		}
		out = append(out, rule{m: m, negate: negate})
	}
	return out
}

// trimTrailingSpaces drops the spaces that end line, except one a
// backslash escapes.
func trimTrailingSpaces(line string) string {
	end := len(line)
	for end > 0 && line[end-1] == ' ' {
		if trailingEscape(line[:end-1]) {
			break // "\ ": an escaped space stays
		}
		end--
	}
	return line[:end]
}

// repoTop is the nearest directory at or above dir holding a .git entry, a
// directory or a worktree's file, or "" when there is none.
func repoTop(dir string) string {
	for d := filepath.Clean(dir); ; {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// excludeFile is the repository's info/exclude: under .git, or for a
// worktree, whose .git is a file naming its git directory, under the common
// directory that directory's commondir file names, as git reads
// $GIT_COMMON_DIR/info/exclude (0005-PLAN, F1c-1 deviation 1).
func excludeFile(top string) string {
	dotGit := filepath.Join(top, ".git")
	info, err := os.Stat(dotGit)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return filepath.Join(dotGit, "info", "exclude")
	}
	b, err := os.ReadFile(dotGit) //nolint:gosec // G304: the repository's own .git file, read-only
	if err != nil {
		return ""
	}
	gitDir, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "gitdir:")
	if !ok {
		return ""
	}
	gitDir = strings.TrimSpace(gitDir)
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(top, gitDir)
	}
	common := gitDir
	if c, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil { //nolint:gosec // G304: git's own metadata, read-only
		common = strings.TrimSpace(string(c))
		if !filepath.IsAbs(common) {
			common = filepath.Join(gitDir, common)
		}
	}
	return filepath.Join(common, "info", "exclude")
}

// readIgnoreFile is an ignore file's text, or "" when it cannot be read.
// The files above a walk's directory are read this way, outside any root:
// read-only, and their text only filters a walk; it never reaches a model
// (0005-PLAN, entry "F1c made executable").
func readIgnoreFile(name string) string {
	b, err := os.ReadFile(name) //nolint:gosec // G304: an ignore file, read-only, never shown
	if err != nil {
		return ""
	}
	return string(b)
}
