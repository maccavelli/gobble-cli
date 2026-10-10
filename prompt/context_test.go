package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func paths(fs []File) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Path
	}
	return out
}

// The five candidates in Pi's order, the first found winning, a byte-order
// mark stripped; a directory with a candidate's name is no file.
func TestCandidates(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "AGENTS.override.md"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "CLAUDE.md"), "claude")
	if f, _, ok := fromDir(dir); !ok || filepath.Base(f.Path) != "CLAUDE.md" {
		t.Fatalf("got %+v, want CLAUDE.md", f)
	}
	write(t, filepath.Join(dir, "AGENTS.md"), "\ufeffagents")
	if f, _, ok := fromDir(dir); !ok || filepath.Base(f.Path) != "AGENTS.md" || f.Content != "agents" {
		t.Fatalf("got %+v, want AGENTS.md without its BOM", f)
	}
}

// A nested linked worktree's own AGENTS.md shadows the main repository's,
// which is then left out; a sibling worktree's does not (Pi's
// findShadowedContextFile).
func TestWorktreeShadowing(t *testing.T) {
	main := t.TempDir()
	write(t, filepath.Join(main, ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(main, "AGENTS.md"), "main")
	link := func(wt string) {
		gitdir := filepath.Join(main, ".git", "worktrees", filepath.Base(wt))
		write(t, filepath.Join(gitdir, "HEAD"), "ref: refs/heads/wt\n")
		write(t, filepath.Join(gitdir, "commondir"), "../..\n")
		write(t, filepath.Join(wt, ".git"), "gitdir: "+gitdir+"\n")
		write(t, filepath.Join(wt, "AGENTS.md"), "worktree")
	}
	nested := filepath.Join(main, "wt")
	link(nested)
	got, _ := ContextFiles("", nested)
	if len(got) != 1 || got[0].Content != "worktree" {
		t.Fatalf("nested worktree: %v, want its own file only", paths(got))
	}
	sibling := filepath.Join(filepath.Dir(main), filepath.Base(main)+"-sibling")
	t.Cleanup(func() { _ = os.RemoveAll(sibling) }) //nolint:errcheck // best effort
	link(sibling)
	if got, _ := ContextFiles("", sibling); len(got) != 1 || got[0].Content != "worktree" {
		t.Fatalf("sibling worktree: %v", paths(got))
	}
	plain := filepath.Join(main, "sub")
	write(t, filepath.Join(plain, "AGENTS.md"), "sub")
	if got, _ := ContextFiles("", plain); len(got) != 2 || got[0].Content != "main" || got[1].Content != "sub" {
		t.Fatalf("plain sub-directory: %v, want main's then its own", paths(got))
	}
}

// The config directory's file comes first, once, even when cwd is the
// config directory; an unreadable file is a diagnostic, and the walk goes
// on.
func TestConfigFirstAndUnreadable(t *testing.T) {
	cfg := t.TempDir()
	write(t, filepath.Join(cfg, "AGENTS.md"), "config")
	if got, _ := ContextFiles(cfg, cfg); len(got) != 1 {
		t.Fatalf("cwd at the config directory: %v", paths(got))
	}
	dir := t.TempDir()
	write(t, filepath.Join(dir, "AGENTS.md"), "x")
	write(t, filepath.Join(dir, "CLAUDE.md"), "claude")
	old := readFile
	// The file is refused under either spelling: on a case-insensitive
	// file system AGENTS.MD is the same file, as it is for Pi.
	readFile = func(p string) ([]byte, error) {
		if strings.EqualFold(filepath.Base(p), "AGENTS.md") && filepath.Dir(p) == dir {
			return nil, os.ErrPermission
		}
		return old(p)
	}
	t.Cleanup(func() { readFile = old })
	got, diags := ContextFiles(cfg, dir)
	if len(got) != 2 || got[1].Content != "claude" || len(diags) == 0 {
		t.Fatalf("files %v, diagnostics %+v", paths(got), diags)
	}
}

// Nested files are those below cwd down to the read file's directory,
// root-most first; none for a file at cwd or outside it.
func TestNested(t *testing.T) {
	cwd := t.TempDir()
	write(t, filepath.Join(cwd, "AGENTS.md"), "top")
	write(t, filepath.Join(cwd, "a", "AGENTS.md"), "a")
	write(t, filepath.Join(cwd, "a", "b", "CLAUDE.md"), "b")
	got := Nested(cwd, filepath.Join(cwd, "a", "b", "c", "f.go"))
	if len(got) != 2 || got[0].Content != "a" || got[1].Content != "b" {
		t.Fatalf("nested %v", paths(got))
	}
	if Nested(cwd, filepath.Join(cwd, "f.go")) != nil || Nested(cwd, filepath.Join(filepath.Dir(cwd), "x", "f.go")) != nil {
		t.Fatal("a file at or outside cwd has nested files")
	}
}

// SYSTEM.md and APPEND_SYSTEM.md come from the config directory; the
// project's copies wait for trust.
func TestSystemFiles(t *testing.T) {
	cfg, cwd := t.TempDir(), t.TempDir()
	write(t, filepath.Join(cfg, "SYSTEM.md"), "system")
	write(t, filepath.Join(cfg, "APPEND_SYSTEM.md"), "\ufeffappended")
	write(t, filepath.Join(cwd, ".gobble", "SYSTEM.md"), "project")
	sys, app, waiting := SystemFiles(cfg, cwd)
	if sys != "system" || app != "appended" || len(waiting) != 1 || waiting[0].Message != TrustNote {
		t.Fatalf("system %q, append %q, waiting %+v", sys, app, waiting)
	}
}
