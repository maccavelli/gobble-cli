package prompt

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// File is one context file: its path and its text.
type File struct {
	Path, Content string
}

// Diagnostic is a file gobble found and did not use, and why.
type Diagnostic struct {
	Path, Message string
	// Winner is what a name collision kept in the file's place, or "".
	Winner string
}

// candidates are a directory's context-file names, the first found winning
// (Pi's resource-loader.ts:185).
var candidates = []string{"AGENTS.override.md", "AGENTS.md", "AGENTS.MD", "CLAUDE.md", "CLAUDE.MD"}

// TrustNote is the diagnostic of a project file that loads only for a
// trusted project, which gobble has none of before 0005-PLAN F6.
const TrustNote = "not loaded: it needs a trusted project (0005-PLAN F6)"

// readFile is the context files' reader; tests replace it.
var readFile = os.ReadFile

// fromDir is dir's context file, or none: the first candidate that is a
// regular file. One that cannot be read is a diagnostic, and the next is
// tried.
func fromDir(dir string) (File, []Diagnostic, bool) {
	var diags []Diagnostic
	for _, name := range candidates {
		p := filepath.Join(dir, name)
		info, err := os.Stat(p)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		b, err := readFile(p)
		if err != nil {
			diags = append(diags, Diagnostic{Path: p, Message: "could not read: " + err.Error()})
			continue
		}
		return File{Path: p, Content: string(bytes.TrimPrefix(b, []byte("\ufeff")))}, diags, true
	}
	return File{}, diags, false
}

// ContextFiles are the context files for cwd, as Pi's
// loadProjectContextFiles finds them (resource-loader.ts:232-270): the
// config directory's, then each directory's from the filesystem root down
// to cwd, a path seen twice kept once, and the main repository's file left
// out where a nested worktree has its own. No trust is needed.
func ContextFiles(configDir, cwd string) ([]File, []Diagnostic) {
	var out []File
	var diags []Diagnostic
	seen := map[string]bool{}
	if configDir != "" {
		f, d, ok := fromDir(configDir)
		diags = append(diags, d...)
		if ok {
			out, seen[f.Path] = append(out, f), true
		}
	}
	shadowed := shadowedFile(cwd)
	var ancestors []File
	for dir := filepath.Clean(cwd); ; {
		f, d, ok := fromDir(dir)
		diags = append(diags, d...)
		if ok && !seen[f.Path] && (shadowed == "" || canonical(f.Path) != shadowed) {
			ancestors, seen[f.Path] = append([]File{f}, ancestors...), true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return append(out, ancestors...), diags
}

// canonical is p with its links resolved, or p itself.
func canonical(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// shadowedFile is the main repository's context file that a nested linked
// worktree's own copy shadows, canonical, or "" (Pi's
// findShadowedContextFile, resource-loader.ts:214-230): only for a worktree
// under the main repository's root, whose root is checked out from it.
func shadowedFile(cwd string) string {
	repoDir, commonGitDir, ok := gitPaths(cwd)
	if !ok {
		return ""
	}
	common := canonical(commonGitDir)
	worktree := canonical(repoDir)
	main := filepath.Dir(common)
	if !strings.HasPrefix(worktree, main+string(filepath.Separator)) {
		return ""
	}
	if canonical(filepath.Join(main, ".git")) != common {
		return ""
	}
	f, _, ok := fromDir(worktree)
	if !ok {
		return ""
	}
	return filepath.Join(main, filepath.Base(f.Path))
}

// gitPaths finds the repository holding dir, as Pi's findGitPaths does
// (footer-data-provider.ts:16-48): the directory with .git, and the common
// git directory, through a .git file's gitdir and commondir.
func gitPaths(dir string) (repoDir, commonGitDir string, ok bool) {
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

// renderContext is Pi's project context (system-prompt.ts:72-79).
func renderContext(files []File) string {
	parts := []string{"Project-specific instructions and guidelines:"}
	for _, f := range files {
		parts = append(parts, Instruction(f))
	}
	return strings.Join(parts, "\n\n")
}

// Instruction is one context file as the model reads it, Pi's
// <project_instructions> block.
func Instruction(f File) string {
	return `<project_instructions path="` + f.Path + `">` + "\n" + f.Content + "\n</project_instructions>"
}

// Nested are the context files in each directory below cwd, down to the
// one holding path, root-most first (opencode's instructions on read,
// 0005-MADR choice 3). cwd's own, and its ancestors', are already in the
// project context. A path outside cwd has none.
func Nested(cwd, path string) []File {
	cwd = filepath.Clean(cwd)
	rel, err := filepath.Rel(cwd, filepath.Dir(path))
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil
	}
	var out []File
	dir := cwd
	for part := range strings.SplitSeq(rel, string(filepath.Separator)) {
		dir = filepath.Join(dir, part)
		if f, _, ok := fromDir(dir); ok {
			out = append(out, f)
		}
	}
	return out
}

// SystemFiles are SYSTEM.md's and APPEND_SYSTEM.md's texts from the config
// directory, and the project's own copies under .gobble, which wait for
// trust (Pi's resource-loader.ts:1201-1227; 0005-MADR, amendment of
// 2026-10-10, item 3).
func SystemFiles(configDir, cwd string) (system, appended string, waiting []Diagnostic) {
	read := func(dir, name string) string {
		if dir == "" {
			return ""
		}
		b, err := readFile(filepath.Join(dir, name))
		if err != nil {
			return ""
		}
		return string(bytes.TrimPrefix(b, []byte("\ufeff")))
	}
	system, appended = read(configDir, "SYSTEM.md"), read(configDir, "APPEND_SYSTEM.md")
	for _, name := range []string{"SYSTEM.md", "APPEND_SYSTEM.md"} {
		p := filepath.Join(cwd, ".gobble", name)
		if _, err := os.Stat(p); err == nil || !errors.Is(err, fs.ErrNotExist) {
			waiting = append(waiting, Diagnostic{Path: p, Message: TrustNote})
		}
	}
	return system, appended, waiting
}
