package fsx

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// setCeiling makes dirs the walk ceiling for one test, and puts back the
// one before it, the TestMain's, when the test ends. A test that calls it
// does not run in parallel.
func setCeiling(t *testing.T, dirs ...string) {
	t.Helper()
	prev := WalkCeiling()
	if err := SetWalkCeiling(dirs...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := SetWalkCeiling(prev...); err != nil {
			t.Error(err)
		}
	})
}

// A ceiling is directories, each matched by any spelling of it; a path that
// is not a directory is refused and leaves the ceiling as it was; an empty
// one is skipped, and none clears it (0053-MADR D1, A1).
func TestWalkCeiling(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	mkfile(t, filepath.Join(a, "f"), "x")
	if err := os.Mkdir(filepath.Join(a, "x"), 0o750); err != nil {
		t.Fatal(err)
	}
	setCeiling(t, a, "", b)
	if got := WalkCeiling(); !slices.Equal(got, []string{a, b}) {
		t.Fatalf("WalkCeiling = %q, want %q", got, []string{a, b})
	}
	for _, at := range []string{a, b, filepath.Join(a, "x", ".."), a + string(filepath.Separator)} {
		if !AtWalkCeiling(at) {
			t.Errorf("AtWalkCeiling(%q) = false, want true", at)
		}
	}
	for _, not := range []string{filepath.Join(a, "x"), filepath.Dir(a), filepath.Join(a, "none")} {
		if AtWalkCeiling(not) {
			t.Errorf("AtWalkCeiling(%q) = true, want false", not)
		}
	}
	for _, bad := range []string{filepath.Join(a, "f"), filepath.Join(a, "none")} {
		if err := SetWalkCeiling(bad); err == nil {
			t.Errorf("SetWalkCeiling(%q) accepted", bad)
		}
		if got := WalkCeiling(); !slices.Equal(got, []string{a, b}) {
			t.Errorf("after refusing %q, WalkCeiling = %q", bad, got)
		}
	}
	if err := SetWalkCeiling(); err != nil {
		t.Fatal(err)
	}
	if got := WalkCeiling(); got != nil || AtWalkCeiling(a) {
		t.Errorf("cleared: WalkCeiling = %q, AtWalkCeiling(a) = %v", got, AtWalkCeiling(a))
	}
}

// GitRepo stops at the ceiling as at the root: a repository above the
// ceiling, or at it, is not found from below it (0053-MADR D1).
func TestGitRepoStopsAtCeiling(t *testing.T) {
	root := t.TempDir()
	mkfile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/main\n")
	ceil := filepath.Join(root, "c")
	mkfile(t, filepath.Join(ceil, "at", ".git", "HEAD"), "ref: refs/heads/main\n")
	below := filepath.Join(ceil, "p")
	if err := os.MkdirAll(below, 0o750); err != nil {
		t.Fatal(err)
	}
	setCeiling(t, ceil)
	if repo, _, ok := GitRepo(below); ok {
		t.Fatalf("above the ceiling: found %q", repo)
	}
	if b := GitBranch(below); b != "" {
		t.Fatalf("above the ceiling: branch %q", b)
	}
	setCeiling(t, filepath.Join(ceil, "at"))
	if repo, _, ok := GitRepo(filepath.Join(ceil, "at")); ok {
		t.Fatalf("at the ceiling: found %q", repo)
	}
	setCeiling(t, filepath.Dir(root))
	if repo, _, ok := GitRepo(below); !ok || repo != root {
		t.Fatalf("with the ceiling above it: %q %v, want %q", repo, ok, root)
	}
}

// The ignore rules' repository finder stops at the ceiling as at the root
// (0053-MADR D1).
func TestRepoTopStopsAtCeiling(t *testing.T) {
	root := t.TempDir()
	mkfile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/main\n")
	ceil := filepath.Join(root, "c")
	mkfile(t, filepath.Join(ceil, ".git"), "gitdir: elsewhere\n")
	below := filepath.Join(ceil, "p")
	if err := os.MkdirAll(below, 0o750); err != nil {
		t.Fatal(err)
	}
	setCeiling(t, ceil)
	if top := repoTop(below); top != "" {
		t.Fatalf("with the ceiling: repoTop = %q, want none", top)
	}
	setCeiling(t, filepath.Dir(root))
	if top := repoTop(below); top != ceil {
		t.Fatalf("with the ceiling above it: repoTop = %q, want %q", top, ceil)
	}
}
