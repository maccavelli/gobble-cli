package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Walker reads the tree under one directory of the workspace, through its
// os.Root: .git and node_modules are skipped, and so is what the
// repository's ignore files exclude (0005-PLAN, entry "F1c made
// executable"). Hidden entries are kept, and a symbolic link is never
// followed into a directory. Paths are slash-separated, from the walk's
// directory, "." being the directory itself. Close it when done.
type Walker struct {
	fsys   fs.FS
	close  func() error
	ig     *Ignore
	base   string          // the walk's directory, from the top level the ignore rules use
	loaded map[string]bool // directories whose .gitignore has been read
}

// Walker opens dir for walking. The ignore rules come from the repository
// dir is in: its info/exclude, and every .gitignore from its top level down;
// the files above dir are read read-only and only filter the walk.
// Without a repository, only the .gitignore files under dir apply.
func (w Workspace) Walker(dir string) (*Walker, error) {
	root, rel, err := w.open(dir)
	if err != nil {
		return nil, err
	}
	k := &Walker{close: func() error { return nil }, ig: &Ignore{}, loaded: map[string]bool{}}
	if root == nil {
		k.fsys = os.DirFS(dir)
	} else {
		sub, err := fs.Sub(root.FS(), filepath.ToSlash(rel))
		if err != nil {
			root.Close() //nolint:errcheck,gosec // the error is the Sub's
			return nil, err
		}
		k.fsys, k.close = sub, root.Close
	}
	if top := repoTop(dir); top != "" {
		k.base = slash(top, dir)
		k.ig.add("", -1, readIgnoreFile(excludeFile(top)))
		// The .gitignore files above dir, from the top level down, each at
		// its depth; dir's own is read by load, at the next.
		if k.base != "" {
			parts := strings.Split(k.base, "/")
			for i := range parts {
				base := strings.Join(parts[:i], "/")
				k.ig.add(base, i, readIgnoreFile(filepath.Join(top, filepath.FromSlash(base), ".gitignore")))
			}
		}
	}
	return k, nil
}

// slash is to relative to from, slash-separated, "" when they are the same.
func slash(from, to string) string {
	rel, err := filepath.Rel(from, to)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

// Close releases the walk's root.
func (k *Walker) Close() error { return k.close() }

// ReadFile reads rel.
func (k *Walker) ReadFile(rel string) ([]byte, error) { return fs.ReadFile(k.fsys, rel) }

// Stat is fs.Stat of rel.
func (k *Walker) Stat(rel string) (fs.FileInfo, error) { return fs.Stat(k.fsys, rel) }

// ReadDir is rel's entries, sorted by name in byte order, without .git,
// node_modules and what the ignore rules exclude.
func (k *Walker) ReadDir(rel string) ([]fs.DirEntry, error) {
	k.load(rel)
	entries, err := fs.ReadDir(k.fsys, rel)
	if err != nil {
		return nil, err
	}
	out := entries[:0]
	for _, e := range entries {
		name := e.Name()
		if name == ".git" || name == "node_modules" {
			continue
		}
		if k.ig.Ignored(k.top(path.Join(rel, name)), e.IsDir()) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// Ignored reports whether rel is excluded by the ignore rules read so far,
// or is .git or node_modules or under one.
func (k *Walker) Ignored(rel string, dir bool) bool {
	for p := range strings.SplitSeq(rel, "/") {
		if p == ".git" || p == "node_modules" {
			return true
		}
	}
	k.load(path.Dir(rel))
	return k.ig.Ignored(k.top(rel), dir)
}

// Walk visits every entry under the walk's directory, depth first, each
// directory's entries in ReadDir's order. visit returning fs.SkipDir skips
// a directory's contents, or, for a file, the rest of its directory;
// fs.SkipAll ends the walk. A subdirectory that cannot be read is skipped.
func (k *Walker) Walk(visit func(rel string, d fs.DirEntry) error) error {
	err := k.walkDir(".", visit, true)
	if errors.Is(err, fs.SkipAll) {
		return nil
	}
	return err
}

func (k *Walker) walkDir(dir string, visit func(string, fs.DirEntry) error, top bool) error {
	entries, err := k.ReadDir(dir)
	if err != nil {
		if top {
			return err
		}
		return nil
	}
	for _, e := range entries {
		rel := path.Join(dir, e.Name())
		err := visit(rel, e)
		switch {
		case errors.Is(err, fs.SkipDir):
			if !e.IsDir() {
				return nil
			}
			continue
		case err != nil:
			return err
		}
		if e.IsDir() {
			if err := k.walkDir(rel, visit, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// load reads the .gitignore of rel and of each directory above it, up to
// the walk's directory, that has not been read yet.
func (k *Walker) load(rel string) {
	dirs := []string{"."}
	if rel != "." && rel != "" {
		parts := strings.Split(rel, "/")
		for i := range parts {
			dirs = append(dirs, strings.Join(parts[:i+1], "/"))
		}
	}
	for _, d := range dirs {
		if k.loaded[d] {
			continue
		}
		k.loaded[d] = true
		b, err := fs.ReadFile(k.fsys, path.Join(d, ".gitignore"))
		if err != nil {
			continue
		}
		base := k.top(d)
		depth := 0
		if base != "" {
			depth = strings.Count(base, "/") + 1
		}
		k.ig.add(base, depth, string(b))
	}
}

// top is rel, from the walk's directory, as a path from the top level.
func (k *Walker) top(rel string) string {
	if rel == "." || rel == "" {
		return k.base
	}
	if k.base == "" {
		return rel
	}
	return k.base + "/" + rel
}
