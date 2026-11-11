package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

// ceilingDir is one directory of the walk ceiling: its absolute path, and
// its stat, which a walked directory is compared with.
type ceilingDir struct {
	path string
	info os.FileInfo
}

// ceiling is where the upward walks stop, unread: none, so the filesystem
// root, unless a test binary set it (0053-MADR D1, A1).
var ceiling atomic.Pointer[[]ceilingDir]

// SetWalkCeiling makes dirs the walk ceiling. GitRepo, the ignore rules'
// repository finder, prompt.ContextFiles and prompt.ProjectSkillDirs stop
// at the first of them they reach, before reading it, as they stop at the
// filesystem root. An empty dir is skipped, and with none the ceiling is
// cleared; a dir that is not a directory is an error, and leaves the
// ceiling as it was.
//
// Only a test binary sets it, to its temp roots, so a host's files above
// them change no test (0053-MADR D1, D2, A1); a forbidigo rule keeps it out
// of every other file.
func SetWalkCeiling(dirs ...string) error {
	var set []ceilingDir
	for _, d := range dirs {
		if d == "" {
			continue
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			return fmt.Errorf("walk ceiling %q: %w", d, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return fmt.Errorf("walk ceiling: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("walk ceiling %s: not a directory", abs)
		}
		set = append(set, ceilingDir{path: abs, info: info})
	}
	if len(set) == 0 {
		ceiling.Store(nil)
		return nil
	}
	ceiling.Store(&set)
	return nil
}

// WalkCeiling is the walk ceiling's directories, absolute, or none.
func WalkCeiling() []string {
	set := ceiling.Load()
	if set == nil {
		return nil
	}
	out := make([]string, len(*set))
	for i, c := range *set {
		out[i] = c.path
	}
	return out
}

// AtWalkCeiling reports whether dir is one of the walk ceiling's
// directories. It compares with os.SameFile, so any spelling of one
// matches: a link to it, a case-folded name, a Windows short name. It is
// false when no ceiling is set, at the cost of one atomic load, or when dir
// cannot be stat'ed.
func AtWalkCeiling(dir string) bool {
	set := ceiling.Load()
	if set == nil {
		return false
	}
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	for _, c := range *set {
		if os.SameFile(c.info, info) {
			return true
		}
	}
	return false
}
