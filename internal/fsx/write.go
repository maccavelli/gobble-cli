package fsx

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
// file keeps its permission bits; a new one is 0o644.
func (w Workspace) WriteFile(abs string, data []byte) error {
	root, rel, err := w.open(abs)
	if err != nil {
		return err
	}
	if root == nil {
		return writeAtomic(osFS{}, abs, data)
	}
	defer root.Close() //nolint:errcheck // the rename has happened or failed by now
	return writeAtomic(root, rel, data)
}

func writeAtomic(fsys fileSystem, name string, data []byte) error {
	if dir := filepath.Dir(name); dir != "." {
		if err := fsys.MkdirAll(dir, 0o750); err != nil {
			return err
		}
	}
	mode := newFileMode
	switch info, err := fsys.Stat(name); {
	case err == nil:
		mode = info.Mode().Perm()
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
	_, werr := f.Write(data)
	if serr := f.Sync(); werr == nil {
		werr = serr
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = fsys.Chmod(tmp, mode)
	}
	if werr == nil {
		werr = rename(fsys, tmp, name)
	}
	if werr != nil {
		_ = fsys.Remove(tmp) //nolint:errcheck // the write's error is the one to report
	}
	return werr
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
