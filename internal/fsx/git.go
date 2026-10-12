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
	for d := filepath.Clean(dir); ; {
		gp := filepath.Join(d, ".git")
		if info, err := os.Stat(gp); err == nil {
			if info.IsDir() {
				if _, err := os.Stat(filepath.Join(gp, "HEAD")); err != nil {
					return "", "", false
				}
				return d, gp, true
			}
			b, err := os.ReadFile(gp) //nolint:gosec // G304: a .git file of the user's repository
			content := strings.TrimSpace(string(b))
			if err != nil || !strings.HasPrefix(content, "gitdir: ") {
				return "", "", false
			}
			gitDir := absUnder(d, strings.TrimSpace(content[len("gitdir: "):]))
			if _, err := os.Stat(filepath.Join(gitDir, "HEAD")); err != nil { //nolint:gosec // G703: the user's own .git file names it, as git reads it
				return "", "", false
			}
			common := gitDir
			if c, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil { //nolint:gosec // G304: as above
				common = absUnder(gitDir, strings.TrimSpace(string(c)))
			}
			return d, common, true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", "", false
		}
		d = parent
	}
}

func absUnder(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}
