package fsx

import (
	"io/fs"
	"os"

	"golang.org/x/sys/windows"
)

// links is the number of names name has. Windows's os.Stat does not
// carry the count, so the file is opened to read it.
func links(fsys fileSystem, name string, _ fs.FileInfo) (uint64, error) {
	f, err := fsys.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}
	defer f.Close() //nolint:errcheck // read-only
	var d windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &d); err != nil {
		return 0, err
	}
	return uint64(d.NumberOfLinks), nil
}

// syncDir does nothing: Windows has no directory sync to call.
func syncDir(fileSystem, string) error { return nil }
