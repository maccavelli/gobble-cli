package prompt

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TrustFiles finds each of the project's files that wait for trust, and
// only those that exist; home's .agents/skills is the user's (0005-PLAN
// F6-1 decision 1).
func TestTrustFiles(t *testing.T) {
	home := t.TempDir()
	cwd := filepath.Join(home, "repo", "sub")
	if err := os.MkdirAll(filepath.Join(home, "repo", ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if got := TrustFiles(cwd, home); len(got) != 0 {
		t.Fatalf("an empty project: %v", got)
	}
	var want []string
	for _, f := range []string{"SYSTEM.md", "APPEND_SYSTEM.md", "prompts/a.md", "personalities/dry/messages.md", "skills/x/SKILL.md"} {
		write(t, filepath.Join(cwd, ".gobble", f), "x")
	}
	for _, name := range []string{"SYSTEM.md", "APPEND_SYSTEM.md", "prompts", "personalities", "skills"} {
		want = append(want, filepath.Join(cwd, ".gobble", name))
	}
	write(t, filepath.Join(home, "repo", ".agents", "skills", "y", "SKILL.md"), "x")
	write(t, filepath.Join(home, ".agents", "skills", "z", "SKILL.md"), "x")
	want = append(want, filepath.Join(home, "repo", ".agents", "skills"))
	if got := TrustFiles(cwd, home); !slices.Equal(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

// Outside any repository the walk goes up to the filesystem's root: it
// keeps an ancestor's .agents/skills, and passes home's own by, which is
// the user's (0024-MADR D7).
func TestProjectSkillDirsOutsideRepository(t *testing.T) {
	root := t.TempDir()
	for d := filepath.Dir(root); ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			t.Skipf("the temporary directory is inside a repository, at %s", d)
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	ancestor := filepath.Join(root, "outer")
	home := filepath.Join(ancestor, "home")
	cwd := filepath.Join(home, "proj", "sub")
	write(t, filepath.Join(ancestor, ".agents", "skills", "a", "SKILL.md"), "x")
	write(t, filepath.Join(home, ".agents", "skills", "b", "SKILL.md"), "x")
	write(t, filepath.Join(home, "proj", ".agents", "skills", "c", "SKILL.md"), "x")
	if err := os.MkdirAll(cwd, 0o700); err != nil {
		t.Fatal(err)
	}
	var got []string // only this test's: a real ancestor of root may hold its own
	for _, d := range ProjectSkillDirs(cwd, home) {
		if rel, err := filepath.Rel(root, d); err == nil && filepath.IsLocal(rel) {
			got = append(got, d)
		}
	}
	want := []string{filepath.Join(home, "proj", ".agents", "skills"), filepath.Join(ancestor, ".agents", "skills")}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}
