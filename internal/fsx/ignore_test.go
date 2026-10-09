package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

// ignoreOf is an Ignore of one .gitignore at the top level.
func ignoreOf(text string) *Ignore {
	ig := &Ignore{}
	ig.add("", 0, text)
	return ig
}

// git's precedence (git-scm.com/docs/gitignore): the last match wins, "!"
// re-includes, and nothing re-includes a file under an ignored directory.
func TestIgnoreRules(t *testing.T) {
	cases := []struct {
		name, text, rel string
		dir             bool
		want            bool
	}{
		{"a match ignores", "*.log\n", "a/b.log", false, true},
		{"no match keeps", "*.log\n", "a/b.txt", false, false},
		{"the last match wins", "*.log\n!keep.log\n", "keep.log", false, false},
		{"a later rule ignores again", "*.log\n!keep.log\nkeep.log\n", "keep.log", false, true},
		{"no re-include under an ignored directory", "build/\n!build/keep.txt\n", "build/keep.txt", false, true},
		{"a directory-only rule leaves a file", "out/\n", "out", false, false},
		{"comments and blank lines are nothing", "# *.log\n\n", "a.log", false, false},
		{"an escaped hash is a pattern", "\\#tmp\n", "#tmp", false, true},
		{"an escaped bang is a pattern", "\\!x\n", "!x", false, true},
		{"trailing spaces are dropped", "*.log   \n", "a.log", false, true},
		{"an escaped trailing space stays", "a\\ \n", "a ", false, true},
		{"an escaped trailing space is not dropped", "a\\ \n", "a", false, false},
		{"a malformed pattern matches nothing", "[\n", "[", false, false},
		{"a byte order mark is not part of the first pattern", "\ufeff*.log\n", "a.log", false, true},
		{"CRLF line ends", "*.log\r\n*.tmp\r\n", "a.tmp", false, true},
	}
	for _, c := range cases {
		if got := ignoreOf(c.text).Ignored(c.rel, c.dir); got != c.want {
			t.Errorf("%s: Ignored(%q) = %v, want %v", c.name, c.rel, got, c.want)
		}
	}
}

// A lower .gitignore overrides a higher one, and info/exclude is below both.
func TestIgnorePrecedence(t *testing.T) {
	ig := &Ignore{}
	ig.add("", 0, "*.gen\n")
	ig.add("", -1, "!*.gen\nsecret.txt\n")
	ig.add("sub", 1, "!*.gen\n")
	for _, c := range []struct {
		rel  string
		want bool
	}{
		{"a.gen", true},             // the top .gitignore beats info/exclude
		{"sub/a.gen", false},        // sub's .gitignore beats the top's
		{"secret.txt", true},        // info/exclude applies everywhere
		{"sub/x/secret.txt", true},  // at any depth
		{"subway/a.gen", true},      // sub's rules are sub's alone
		{"sub/deep/a.gen", false},   // and below it
		{"other/sub/a.gen", true},   // not another sub
		{"sub", false},              // the directory itself is not matched
		{"sub/deep/b.txt", false},   // nothing matches
		{"sub/deep/x.gen/y", false}, // a.gen re-included, so its contents are too
	} {
		if got := ig.Ignored(c.rel, false); got != c.want {
			t.Errorf("Ignored(%q) = %v, want %v", c.rel, got, c.want)
		}
	}
}

// info/exclude is under .git; for a worktree, under the common directory
// its git directory's commondir names (F1c-1 deviation 1).
func TestExcludeFile(t *testing.T) {
	base := t.TempDir()
	plain := filepath.Join(base, "plain")
	mkdir(t, filepath.Join(plain, ".git", "info"))
	if got, want := excludeFile(plain), filepath.Join(plain, ".git", "info", "exclude"); got != want {
		t.Errorf("a repository's exclude file is %q, want %q", got, want)
	}

	main := filepath.Join(base, "main")
	gitDir := filepath.Join(main, ".git", "worktrees", "wt")
	mkdir(t, gitDir)
	write(t, filepath.Join(gitDir, "commondir"), "../..\n")
	wt := filepath.Join(base, "wt")
	mkdir(t, wt)
	write(t, filepath.Join(wt, ".git"), "gitdir: "+gitDir+"\n")
	if got, want := excludeFile(wt), filepath.Join(main, ".git", "info", "exclude"); got != want {
		t.Errorf("a worktree's exclude file is %q, want the common directory's %q", got, want)
	}

	noCommon := filepath.Join(base, "nocommon")
	other := filepath.Join(base, "othergit")
	mkdir(t, other)
	mkdir(t, noCommon)
	write(t, filepath.Join(noCommon, ".git"), "gitdir: ../othergit\n")
	if got, want := excludeFile(noCommon), filepath.Join(other, "info", "exclude"); got != want {
		t.Errorf("a .git file with no commondir gives %q, want %q", got, want)
	}

	if got := excludeFile(filepath.Join(base, "none")); got != "" {
		t.Errorf("no repository gives %q, want none", got)
	}
}

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
}
