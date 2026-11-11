package prompt

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// setCeiling makes dirs the walk ceiling for one test, and puts back the
// one before it when the test ends. A test that calls it does not run in
// parallel.
func setCeiling(t *testing.T, dirs ...string) {
	t.Helper()
	prev := fsx.WalkCeiling()
	if err := fsx.SetWalkCeiling(dirs...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fsx.SetWalkCeiling(prev...); err != nil {
			t.Error(err)
		}
	})
}

// The context files' walk stops at the ceiling, unread: a file above it or
// at it is not loaded (0053-MADR D1).
func TestContextFilesStopAtCeiling(t *testing.T) {
	root := t.TempDir()
	ceil := filepath.Join(root, "c")
	cwd := filepath.Join(ceil, "p")
	write(t, filepath.Join(root, "AGENTS.md"), "above")
	write(t, filepath.Join(ceil, "AGENTS.md"), "at")
	write(t, filepath.Join(cwd, "AGENTS.md"), "own")
	setCeiling(t, ceil)
	if got, _ := ContextFiles("", cwd); len(got) != 1 || got[0].Content != "own" {
		t.Fatalf("with the ceiling: %v, want the cwd's file only", paths(got))
	}
	setCeiling(t, filepath.Dir(root))
	got, _ := ContextFiles("", cwd)
	var texts []string
	for _, f := range got {
		texts = append(texts, f.Content)
	}
	if !slices.Equal(texts, []string{"above", "at", "own"}) {
		t.Fatalf("with the ceiling above them: %q, want above, at, own", texts)
	}
}

// The project's skills walk stops at the ceiling, unread: skills above it
// or at it are not listed. With the ceiling above them, the walk takes the
// .git directory's own skills and stops there (0053-MADR D1, A2).
func TestProjectSkillDirsStopAtCeiling(t *testing.T) {
	root := t.TempDir()
	ceil := filepath.Join(root, "c")
	cwd := filepath.Join(ceil, "p", "q")
	write(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(root, ".agents", "skills", "a", "SKILL.md"), "x")
	write(t, filepath.Join(ceil, ".agents", "skills", "b", "SKILL.md"), "x")
	write(t, filepath.Join(ceil, "p", ".agents", "skills", "c", "SKILL.md"), "x")
	if err := os.MkdirAll(cwd, 0o700); err != nil {
		t.Fatal(err)
	}
	own := filepath.Join(ceil, "p", ".agents", "skills")
	setCeiling(t, ceil)
	if got := ProjectSkillDirs(cwd, ""); !slices.Equal(got, []string{own}) {
		t.Fatalf("with the ceiling: %q, want %q", got, []string{own})
	}
	setCeiling(t, filepath.Dir(root))
	want := []string{own, filepath.Join(ceil, ".agents", "skills"), filepath.Join(root, ".agents", "skills")}
	if got := ProjectSkillDirs(cwd, ""); !slices.Equal(got, want) {
		t.Fatalf("with the ceiling above them: %q, want %q", got, want)
	}
}
