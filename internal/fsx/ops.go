package fsx

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// ErrCrossDevice is the error of a rename that cannot be one: the two paths
// are in different roots, or on different devices. A move then copies and
// deletes (0005-PLAN, entry "F1c made executable", F1c-2).
var ErrCrossDevice = errors.New("not on one device")

// rootRename is os.Root.Rename; tests replace it to make a rename fail as
// it does across devices.
var rootRename = func(r *os.Root, from, to string) error { return r.Rename(from, to) }

// Lstat is os.Lstat of abs, through its root: a symbolic link is described,
// not followed.
func (w Workspace) Lstat(abs string) (fs.FileInfo, error) {
	root, rel, err := w.open(abs)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return os.Lstat(abs)
	}
	defer root.Close() //nolint:errcheck // read-only
	return root.Lstat(rel)
}

// MkdirAll creates the directory abs and its parents, through its root.
func (w Workspace) MkdirAll(abs string) error {
	root, rel, err := w.open(abs)
	if err != nil {
		return err
	}
	if root == nil {
		return os.MkdirAll(abs, 0o750)
	}
	defer root.Close() //nolint:errcheck // the directories exist or not by now
	return root.MkdirAll(rel, 0o750)
}

// Remove removes the file, or the empty directory, abs.
func (w Workspace) Remove(abs string) error {
	root, rel, err := w.open(abs)
	if err != nil {
		return err
	}
	if root == nil {
		return os.Remove(abs)
	}
	defer root.Close() //nolint:errcheck // removed or not by now
	return root.Remove(rel)
}

// RemoveAll removes abs and everything under it, through its root.
func (w Workspace) RemoveAll(abs string) error {
	root, rel, err := w.open(abs)
	if err != nil {
		return err
	}
	if root == nil {
		return os.RemoveAll(abs)
	}
	defer root.Close() //nolint:errcheck // removed or not by now
	return root.RemoveAll(rel)
}

// Rename moves from to to when both are in one root and on one device. It
// returns ErrCrossDevice when they are not, and the caller copies instead.
func (w Workspace) Rename(from, to string) error {
	fromRoot, fromRel, err := w.open(from)
	if err != nil {
		return err
	}
	toRoot, toRel, err := w.open(to)
	if err != nil {
		closeRoot(fromRoot)
		return err
	}
	defer closeRoot(fromRoot)
	defer closeRoot(toRoot)
	switch {
	case fromRoot == nil && toRoot == nil:
		err = os.Rename(from, to)
	case fromRoot == nil || toRoot == nil || fromRoot.Name() != toRoot.Name():
		return ErrCrossDevice
	default:
		err = rootRename(fromRoot, fromRel, toRel)
	}
	if errors.Is(err, syscall.EXDEV) {
		return ErrCrossDevice
	}
	return err
}

// CopyFile writes a copy of the file from at to, atomically, with from's
// permission bits.
func (w Workspace) CopyFile(from, to string) error {
	info, err := w.Stat(from)
	if err != nil {
		return err
	}
	data, err := w.ReadFile(from)
	if err != nil {
		return err
	}
	if err := w.WriteFile(to, data); err != nil {
		return err
	}
	return w.chmod(to, info.Mode().Perm())
}

// CopyLink makes to a symbolic link with the same target text as the link
// from. The target is not followed or checked.
func (w Workspace) CopyLink(from, to string) error {
	target, err := w.readlink(from)
	if err != nil {
		return err
	}
	root, rel, err := w.open(to)
	if err != nil {
		return err
	}
	if root == nil {
		return os.Symlink(target, to)
	}
	defer root.Close() //nolint:errcheck // linked or not by now
	return root.Symlink(target, rel)
}

func (w Workspace) readlink(abs string) (string, error) {
	root, rel, err := w.open(abs)
	if err != nil {
		return "", err
	}
	if root == nil {
		return os.Readlink(abs)
	}
	defer root.Close() //nolint:errcheck // read-only
	return root.Readlink(rel)
}

func (w Workspace) chmod(abs string, mode fs.FileMode) error {
	root, rel, err := w.open(abs)
	if err != nil {
		return err
	}
	if root == nil {
		return os.Chmod(abs, mode)
	}
	defer root.Close() //nolint:errcheck // changed or not by now
	return root.Chmod(rel, mode)
}

// IsRoot reports whether abs is a workspace root, or a directory above one,
// which delete refuses.
func (w Workspace) IsRoot(abs string) bool {
	for _, r := range w.Roots {
		if within(abs, r) {
			return true
		}
	}
	return false
}

// LockPair holds the locks of a and b, taken in the order of their keys so
// two callers locking the same pair in opposite orders cannot deadlock.
func LockPair(a, b string) (unlock func()) {
	ka, kb := lockKey(a), lockKey(b)
	if ka == kb {
		return Lock(a)
	}
	if kb < ka {
		a, b = b, a
	}
	first := Lock(a)
	second := Lock(b)
	return func() {
		second()
		first()
	}
}

func closeRoot(r *os.Root) {
	if r != nil {
		r.Close() //nolint:errcheck,gosec // the operation has finished by now
	}
}

// Inside reports whether to is from or under it: a move or a copy of a
// directory into itself.
func Inside(from, to string) bool { return within(from, to) }
