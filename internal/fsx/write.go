package fsx

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// newFileMode is a created file's mode: what Pi's writeFile gives under the
// usual umask.
const newFileMode fs.FileMode = 0o644

// fileSystem is the part of os and os.Root that WriteFile uses, so the same
// steps run inside a root and, for an approved path, outside every root.
type fileSystem interface {
	MkdirAll(name string, perm fs.FileMode) error
	Stat(name string) (fs.FileInfo, error)
	OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error)
	Chmod(name string, mode fs.FileMode) error
	Rename(oldname, newname string) error
	Remove(name string) error
}

type osFS struct{}

func (osFS) MkdirAll(name string, perm fs.FileMode) error { return os.MkdirAll(name, perm) }
func (osFS) Stat(name string) (fs.FileInfo, error)        { return os.Stat(name) }
func (osFS) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm) //nolint:gosec // G304: an approved path outside the roots
}
func (osFS) Chmod(name string, mode fs.FileMode) error { return os.Chmod(name, mode) }
func (osFS) Rename(oldname, newname string) error      { return os.Rename(oldname, newname) }
func (osFS) Remove(name string) error                  { return os.Remove(name) }

// WriteFile replaces or creates abs with data, and its parent directories.
// It writes a temporary file beside abs and renames it into place, so a
// reader sees the old content or the new, never part of it. An existing
// file keeps its permission bits; a new one is 0o644. A symlink is followed
// to its target, which must be inside a root, and a file with more than one
// hard link is written in place (0037-MADR D6).
func (w Workspace) WriteFile(abs string, data []byte) error {
	return w.writeFrom(abs, bytes.NewReader(data))
}

// writeFrom is WriteFile with the content read from r.
func (w Workspace) writeFrom(abs string, r io.Reader) error {
	abs, err := follow(abs)
	if err != nil {
		return err
	}
	root, rel, err := w.open(abs)
	if err != nil {
		return err
	}
	if root == nil {
		return writeAtomicFrom(osFS{}, abs, r, newFileMode)
	}
	defer root.Close() //nolint:errcheck // the rename has happened or failed by now
	return writeAtomicFrom(root, rel, r, newFileMode)
}

// WriteFileMode replaces or creates abs with data, as WriteFile does, for a
// file outside any workspace, such as a settings file or mcp.json: written
// beside abs, synced, and renamed into place. An existing file keeps its
// permission bits; a new one gets newMode (0018-PLAN P1 deviation 1). A
// symlink and a hard link are kept, as WriteFile keeps them.
func WriteFileMode(abs string, data []byte, newMode fs.FileMode) error {
	abs, err := follow(abs)
	if err != nil {
		return err
	}
	return writeAtomic(osFS{}, abs, data, newMode)
}

// maxHops is how many dangling symlinks follow reads before it gives up:
// Linux's own limit.
const maxHops = 40

// follow is abs, or, when abs is a symlink, the file it names, so a write
// through the link writes its target and keeps the link. A link whose
// target does not exist yet is read one hop at a time, so the write creates
// the target (0037-PLAN deviation 5).
func follow(abs string) (string, error) {
	for range maxHops {
		info, err := os.Lstat(abs)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return abs, nil
		case err != nil:
			return "", err
		case info.Mode()&fs.ModeSymlink == 0:
			return abs, nil
		}
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			return resolved, nil
		}
		target, err := os.Readlink(abs)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(abs), target)
		}
		abs = target
	}
	return "", fmt.Errorf("%s: more than %d symlinks", abs, maxHops)
}

func writeAtomic(fsys fileSystem, name string, data []byte, newMode fs.FileMode) error {
	return writeAtomicFrom(fsys, name, bytes.NewReader(data), newMode)
}

// writeAtomicFrom is writeAtomic with the content read from r, so a copy
// streams (0037-MADR D7).
func writeAtomicFrom(fsys fileSystem, name string, r io.Reader, newMode fs.FileMode) error {
	if dir := filepath.Dir(name); dir != "." {
		if err := fsys.MkdirAll(dir, 0o750); err != nil {
			return err
		}
	}
	mode := newMode
	switch info, err := fsys.Stat(name); {
	case err == nil:
		mode = info.Mode().Perm()
		// A rename would leave the other names on the old content, so a
		// file with more than one is written in place (0037-MADR Q1 (a)).
		n, err := links(fsys, name, info)
		if err != nil {
			return err
		}
		if n > 1 {
			return writeInPlace(fsys, name, r)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	tmp, err := tempName(name)
	if err != nil {
		return err
	}
	f, err := fsys.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	werr := fill(f, r)
	if werr == nil {
		werr = fsys.Chmod(tmp, mode)
	}
	if werr == nil {
		werr = rename(fsys, tmp, name)
	}
	if werr != nil {
		_ = fsys.Remove(tmp) //nolint:errcheck // the write's error is the one to report
		return werr
	}
	return syncDir(fsys, filepath.Dir(name))
}

// writeInPlace truncates name and writes r's content into it, keeping its
// mode and every name it has.
func writeInPlace(fsys fileSystem, name string, r io.Reader) error {
	f, err := fsys.OpenFile(name, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	return fill(f, r)
}

// fill copies r into f, syncs and closes f, and returns the first error.
func fill(f *os.File, r io.Reader) error {
	_, err := io.Copy(f, r)
	if serr := f.Sync(); err == nil {
		err = serr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// renameWindow is how long a rename keeps retrying an error that another
// process holding the file causes, as Go's cmd/go robustio does.
const renameWindow = 2 * time.Second

// rename moves tmp onto name. Windows refuses to replace a file another
// process has open without FILE_SHARE_DELETE, an editor or a reader in the
// middle of a read, so a transient error is retried with growing pauses
// until renameWindow has passed.
func rename(fsys fileSystem, tmp, name string) error {
	deadline := time.Now().Add(renameWindow)
	pause := time.Millisecond
	for {
		err := fsys.Rename(tmp, name)
		if err == nil || !transient(err) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(pause)
		pause = min(2*pause, 100*time.Millisecond)
	}
}

// tempName is a fresh name beside name: hidden, and marked as gobble's.
func tempName(name string) (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(name), "."+filepath.Base(name)+".gobble-"+hex.EncodeToString(b[:])+".tmp"), nil
}
