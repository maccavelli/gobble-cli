package prompt

import (
	"os"
	"path/filepath"
)

// ProjectSkillDirs are the project's own skill directories that exist: its
// .gobble/skills, and each .agents/skills from cwd up to the repository
// root, or the filesystem root outside one, but not home's, which is the
// user's (Pi's collectAncestorAgentsSkillDirs, package-manager.ts:468-487).
func ProjectSkillDirs(cwd, home string) []string {
	var out []string
	if p := filepath.Join(cwd, ".gobble", "skills"); isDir(p) {
		out = append(out, p)
	}
	stop := ""
	for d := filepath.Clean(cwd); ; {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			stop = d
			break
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	for d := filepath.Clean(cwd); ; {
		if p := filepath.Join(d, ".agents", "skills"); isDir(p) && (home == "" || filepath.Clean(d) != filepath.Clean(home)) {
			out = append(out, p)
		}
		parent := filepath.Dir(d)
		if d == stop || parent == d {
			break
		}
		d = parent
	}
	return out
}

// TrustFiles are the files and directories of cwd's project that exist and
// load only for a trusted project, its settings file aside, which
// internal/config finds (0005-PLAN F6-1 decision 1): .gobble's SYSTEM.md,
// APPEND_SYSTEM.md, prompts and personalities, and ProjectSkillDirs.
func TrustFiles(cwd, home string) []string {
	var out []string
	for _, name := range []string{"SYSTEM.md", "APPEND_SYSTEM.md", "prompts", "personalities"} {
		p := filepath.Join(cwd, ".gobble", name)
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
	}
	return append(out, ProjectSkillDirs(cwd, home)...)
}

func isDir(p string) bool { info, err := os.Stat(p); return err == nil && info.IsDir() }
