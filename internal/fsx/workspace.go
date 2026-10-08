package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrOutside is the error of a path outside every root, opened without
// permission.
var ErrOutside = errors.New("outside the workspace")

// Workspace is the directories file tools may use: the session's working
// directory and its additional directories, all absolute. A path inside
// one is opened through an os.Root for it, so a symlink or .. that leaves
// the root is refused by the runtime as well as by Outside. A path outside
// every root is opened only when AllowOutside is set, which is a call the
// user approved. A Workspace with no roots confines nothing.
type Workspace struct {
	Roots        []string
	AllowOutside bool
}

// Outside reports whether abs lies outside every root. Symlinks are
// resolved first, in the deepest part of abs that exists, so a link inside
// a root that points out of it is outside.
func (w Workspace) Outside(abs string) bool {
	if len(w.Roots) == 0 {
		return false
	}
	resolved := realPath(abs)
	for _, r := range w.Roots {
		if within(realPath(r), resolved) {
			return false
		}
	}
	return true
}

// ReadFile reads abs.
func (w Workspace) ReadFile(abs string) ([]byte, error) {
	root, rel, err := w.open(abs)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return os.ReadFile(abs) //nolint:gosec // G304: outside the roots only when the user approved the call
	}
	defer root.Close() //nolint:errcheck // read-only
	return root.ReadFile(rel)
}

// Stat is os.Stat of abs, through its root.
func (w Workspace) Stat(abs string) (fs.FileInfo, error) {
	root, rel, err := w.open(abs)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return os.Stat(abs)
	}
	defer root.Close() //nolint:errcheck // read-only
	return root.Stat(rel)
}

// ReadDir lists the directory abs, through its root.
func (w Workspace) ReadDir(abs string) ([]fs.DirEntry, error) {
	root, rel, err := w.open(abs)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return os.ReadDir(abs)
	}
	defer root.Close() //nolint:errcheck // read-only
	return fs.ReadDir(root.FS(), filepath.ToSlash(rel))
}

// open finds the root abs lies in and opens it, returning the path
// relative to it. It returns no root for an approved path outside them all,
// and ErrOutside for one that is not approved.
func (w Workspace) open(abs string) (*os.Root, string, error) {
	if len(w.Roots) == 0 {
		return nil, "", nil
	}
	outside := w.Outside(abs)
	if outside && w.AllowOutside {
		return nil, "", nil
	}
	if outside {
		return nil, "", fmt.Errorf("%s: %w", abs, ErrOutside)
	}
	for _, r := range w.Roots {
		rel, ok := relative(r, abs)
		if !ok {
			continue
		}
		root, err := os.OpenRoot(r)
		if err != nil {
			return nil, "", err
		}
		return root, rel, nil
	}
	// Inside only through a symlink: a root reached by resolving a link,
	// such as a root under a linked directory. Lexically abs is under no
	// root, so open it by its real path.
	resolved := realPath(abs)
	for _, r := range w.Roots {
		if rel, ok := relative(realPath(r), resolved); ok {
			root, err := os.OpenRoot(realPath(r))
			if err != nil {
				return nil, "", err
			}
			return root, rel, nil
		}
	}
	return nil, "", fmt.Errorf("%s: %w", abs, ErrOutside)
}

// realPath is abs with the symlinks of its deepest existing ancestor
// resolved, and the part that does not exist yet appended as it is.
func realPath(abs string) string {
	p := filepath.Clean(abs)
	var rest []string
	for {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(append([]string{r}, rest...)...)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return filepath.Clean(abs)
		}
		rest = append([]string{filepath.Base(p)}, rest...)
		p = parent
	}
}

// within reports whether path is root or under it.
func within(root, path string) bool {
	_, ok := relative(root, path)
	return ok
}

// relative is path relative to root, when path is root or under it. On
// Windows names compare without regard to case, as the file system does.
func relative(root, path string) (string, bool) {
	r, p := filepath.Clean(root), filepath.Clean(path)
	if runtime.GOOS == goosWindows {
		return relativeFold(r, p)
	}
	rel, err := filepath.Rel(r, p)
	if err != nil || !local(rel) {
		return "", false
	}
	return rel, true
}

// relativeFold is relative for Windows: the volume and each element of root
// must match p's without regard to case, and the rest of p, as spelled, is
// the relative path when it is local.
func relativeFold(r, p string) (string, bool) {
	rv, pv := filepath.VolumeName(r), filepath.VolumeName(p)
	if !strings.EqualFold(rv, pv) {
		return "", false
	}
	split := func(s string) []string {
		return strings.FieldsFunc(s, func(c rune) bool { return c == '\\' || c == '/' })
	}
	re, pe := split(r[len(rv):]), split(p[len(pv):])
	if len(pe) < len(re) {
		return "", false
	}
	for i := range re {
		if !strings.EqualFold(re[i], pe[i]) {
			return "", false
		}
	}
	if len(pe) == len(re) {
		return ".", true
	}
	// IsLocal refuses what os.Root refuses on Windows, such as a reserved
	// name or a ':' that names a stream, so the two agree.
	rel := filepath.Join(pe[len(re):]...)
	return rel, local(rel)
}

// foldKey is p as a map key: on Windows, case does not tell two paths apart.
func foldKey(p string) string {
	if runtime.GOOS == goosWindows {
		return strings.ToLower(p)
	}
	return p
}

func local(rel string) bool {
	return rel == "." || filepath.IsLocal(rel)
}
