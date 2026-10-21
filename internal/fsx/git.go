package fsx

import (
	"os"
	"path/filepath"
	"strings"
)

// GitRepo finds the repository holding dir, as Pi's findGitPaths does
// (footer-data-provider.ts:16-48): the directory with .git, and the common
// git directory, through a .git file's gitdir and commondir. ok is false
// outside a repository, and for a .git that git would not read. prompt's
// context files and internal/config's project settings both find their
// root with it (0018-MADR D18).
func GitRepo(dir string) (repoDir, commonGitDir string, ok bool) {
	repoDir, _, commonGitDir, ok = gitDirs(dir)
	return repoDir, commonGitDir, ok
}

// gitDirs is GitRepo's walk, with the git directory that holds the
// checkout's own HEAD: .git, or a worktree's gitdir.
func gitDirs(dir string) (repoDir, gitDir, commonGitDir string, ok bool) {
	for d := filepath.Clean(dir); ; {
		gp := filepath.Join(d, ".git")
		if info, err := os.Stat(gp); err == nil {
			if info.IsDir() {
				if _, err := os.Stat(filepath.Join(gp, "HEAD")); err != nil {
					return "", "", "", false
				}
				return d, gp, gp, true
			}
			b, err := os.ReadFile(gp) //nolint:gosec // G304: a .git file of the user's repository
			content := strings.TrimSpace(string(b))
			if err != nil || !strings.HasPrefix(content, "gitdir: ") {
				return "", "", "", false
			}
			gitDir := absUnder(d, strings.TrimSpace(content[len("gitdir: "):]))
			if _, err := os.Stat(filepath.Join(gitDir, "HEAD")); err != nil { //nolint:gosec // G703: the user's own .git file names it, as git reads it
				return "", "", "", false
			}
			common := gitDir
			if c, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil { //nolint:gosec // G304: as above
				common = absUnder(gitDir, strings.TrimSpace(string(c)))
			}
			return d, gitDir, common, true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", "", "", false
		}
		d = parent
	}
}

// branchRef starts a HEAD that names a branch.
const branchRef = "ref: refs/heads/"

// GitBranch is the branch checked out in the repository holding dir, read
// from its HEAD, a worktree's own, as Pi's footer reads it
// (footer-data-provider.ts): "ref: refs/heads/<name>" gives the name,
// slashes kept, and any other HEAD "detached". It is "" outside a
// repository, and for a reftable repository, whose HEAD names
// refs/heads/.invalid; no git command is run (0024-MADR D8).
func GitBranch(dir string) string {
	_, gitDir, _, ok := gitDirs(dir)
	if !ok {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(gitDir, "HEAD")) //nolint:gosec // G304: the user's repository's HEAD, read-only
	if err != nil {
		return ""
	}
	head := strings.TrimSpace(string(b))
	name, ok := strings.CutPrefix(head, branchRef)
	switch {
	case !ok:
		return "detached"
	case name == ".invalid":
		return ""
	}
	return name
}

func absUnder(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}
