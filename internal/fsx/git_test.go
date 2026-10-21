package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func mkfile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// GitRepo finds a repository by its .git directory, a linked worktree by
// its .git file's gitdir and commondir, and nothing for a .git without HEAD
// or a directory in no repository.
func TestGitRepo(t *testing.T) {
	root := t.TempDir()

	main := filepath.Join(root, "main")
	mkfile(t, filepath.Join(main, ".git", "HEAD"), "ref: refs/heads/main\n")
	sub := filepath.Join(main, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	if repo, common, ok := GitRepo(sub); !ok || repo != main || common != filepath.Join(main, ".git") {
		t.Fatalf("directory .git: %q %q %v", repo, common, ok)
	}

	wt := filepath.Join(root, "wt")
	wtGit := filepath.Join(main, ".git", "worktrees", "wt")
	mkfile(t, filepath.Join(wtGit, "HEAD"), "ref: refs/heads/topic\n")
	mkfile(t, filepath.Join(wtGit, "commondir"), "../..\n")
	mkfile(t, filepath.Join(wt, ".git"), "gitdir: "+wtGit+"\n")
	if repo, common, ok := GitRepo(wt); !ok || repo != wt || common != filepath.Join(main, ".git") {
		t.Fatalf("worktree: %q %q %v, want %q and the main .git", repo, common, ok, wt)
	}

	broken := filepath.Join(root, "broken")
	if err := os.MkdirAll(filepath.Join(broken, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := GitRepo(broken); ok {
		t.Fatal("a .git without HEAD is not a repository")
	}

	plain := filepath.Join(root, "plain")
	if err := os.MkdirAll(plain, 0o750); err != nil {
		t.Fatal(err)
	}
	if repo, _, ok := GitRepo(plain); ok {
		t.Fatalf("a plain directory found %q", repo)
	}
}

// GitBranch reads the branch from the repository's HEAD, a worktree's own,
// as Pi's footer does: a branch's name, slashes kept; "detached" for a
// commit; "" for a reftable HEAD and outside a repository (0024-MADR D8).
func TestGitBranch(t *testing.T) {
	root := t.TempDir()
	main := filepath.Join(root, "main")
	mkfile(t, filepath.Join(main, ".git", "HEAD"), "ref: refs/heads/feature/x\n")
	sub := filepath.Join(main, "a")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(root, "wt")
	wtGit := filepath.Join(main, ".git", "worktrees", "wt")
	mkfile(t, filepath.Join(wtGit, "HEAD"), "ref: refs/heads/topic\n")
	mkfile(t, filepath.Join(wtGit, "commondir"), "../..\n")
	mkfile(t, filepath.Join(wt, ".git"), "gitdir: "+wtGit+"\n")
	detached := filepath.Join(root, "detached")
	mkfile(t, filepath.Join(detached, ".git", "HEAD"), "0123456789abcdef0123456789abcdef01234567\n")
	reftable := filepath.Join(root, "reftable")
	mkfile(t, filepath.Join(reftable, ".git", "HEAD"), "ref: refs/heads/.invalid\n")
	plain := filepath.Join(root, "plain")
	if err := os.MkdirAll(plain, 0o750); err != nil {
		t.Fatal(err)
	}
	for dir, want := range map[string]string{
		sub: "feature/x", wt: "topic", detached: "detached", reftable: "", plain: "",
	} {
		if got := GitBranch(dir); got != want {
			t.Errorf("GitBranch(%s) = %q, want %q", filepath.Base(dir), got, want)
		}
	}
}
