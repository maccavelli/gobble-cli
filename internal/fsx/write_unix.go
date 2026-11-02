//go:build !windows

package fsx

import (
	"io/fs"
	"os"
	"syscall"
)

// links is the number of names the file info describes has.
func links(_ fileSystem, _ string, info fs.FileInfo) (uint64, error) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 1, nil
	}
	return uint64(st.Nlink), nil //nolint:unconvert // uint16 on darwin, uint64 on linux
}

// syncDir syncs the directory dir, so a rename into it outlasts a crash.
func syncDir(fsys fileSystem, dir string) error {
	d, err := fsys.OpenFile(dir, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	err = d.Sync()
	if cerr := d.Close(); err == nil {
		err = cerr
	}
	return err
}
