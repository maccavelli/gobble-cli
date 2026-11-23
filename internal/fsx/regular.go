package fsx

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"time"
)

// fileError is an error about what a file is, or how big: fs.ErrInvalid,
// with its own text.
type fileError string

func (e fileError) Error() string { return string(e) }
func (fileError) Unwrap() error   { return fs.ErrInvalid }

var (
	// ErrNotRegular is the error of a file opened for reading that is not a
	// regular file: a FIFO, a device, a socket or a directory (0037-MADR
	// D7).
	ErrNotRegular error = fileError("not a regular file")
	// ErrTooLarge is the error of a file over the limit it was read with.
	ErrTooLarge error = fileError("file too large")
)

// ConfigLimit is the most a settings or configuration file may hold
// (0037-MADR D10).
const ConfigLimit = 1 << 20

// openFunc is os.OpenFile's shape.
type openFunc func(name string, flag int, perm fs.FileMode) (*os.File, error)

// The read's seams (0067-MADR D6): how ReadRegularFile opens a file, which
// open errors it retries, and for how long.
var (
	openFile   openFunc = os.OpenFile
	retriable           = sharingViolation
	readWindow          = 2 * time.Second
)

// SetOpenFile makes ReadRegularFile open files through open until restore
// is called. It is a test seam (0067-MADR D6), for a test in another
// package that must see a refused open and release the file's holder on
// it; nothing else calls it, and no two tests may use it at once.
func SetOpenFile(open func(name string, flag int, perm fs.FileMode) (*os.File, error)) (restore func()) {
	old := openFile
	openFile = open
	return func() { openFile = old }
}

// ReadRegularFile reads abs, a file outside any workspace, when it is a
// regular file of at most limit bytes. On Windows an open refused by a
// sharing violation, a replace of abs in the middle of its move, is tried
// again until readWindow has passed (0067-MADR D1).
func ReadRegularFile(abs string, limit int) ([]byte, error) {
	f, err := openRetrying(abs)
	if err != nil {
		return nil, err
	}
	info, err := regular(f, abs)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only
	return readAtMost(f, abs, info.Size(), limit)
}

// openRetrying opens abs for reading. An error retriable reports is tried
// again with growing pauses, from 1 ms to 100 ms, as rename's are, until
// readWindow has passed, and the last one is returned then; any other error
// is returned at once, so a file that went away in the window reads as
// absent (0067-MADR D1, D2).
func openRetrying(abs string) (*os.File, error) {
	deadline := time.Now().Add(readWindow)
	pause := time.Millisecond
	for {
		f, err := openFile(abs, os.O_RDONLY|nonblock, 0)
		if err == nil || !retriable(err) || time.Now().After(deadline) {
			return f, err
		}
		time.Sleep(pause)
		pause = min(2*pause, 100*time.Millisecond)
	}
}

// regular is the info of f, opened as name, when it is a regular file.
// Otherwise f is closed, and the error is ErrNotRegular.
func regular(f *os.File, name string) (fs.FileInfo, error) {
	info, err := f.Stat()
	if err == nil && !info.Mode().IsRegular() {
		err = &fs.PathError{Op: "open", Path: name, Err: ErrNotRegular}
	}
	if err != nil {
		f.Close() //nolint:errcheck,gosec // the check's error is the one to report
		return nil, err
	}
	return info, nil
}

// readAtMost reads f, opened as name with size bytes, to its end, or fails
// with ErrTooLarge past limit bytes. A file that grows while it is read is
// stopped at limit+1.
func readAtMost(f *os.File, name string, size int64, limit int) ([]byte, error) {
	if size > int64(limit) {
		return nil, tooLarge(name, limit)
	}
	var buf bytes.Buffer
	buf.Grow(int(size) + bytes.MinRead) //nolint:gosec // G115: size is at most limit, an int
	if _, err := buf.ReadFrom(io.LimitReader(f, int64(limit)+1)); err != nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: err}
	}
	if buf.Len() > limit {
		return nil, tooLarge(name, limit)
	}
	return buf.Bytes(), nil
}

func tooLarge(name string, limit int) error {
	return &fs.PathError{Op: "read", Path: name, Err: fmt.Errorf("%w: over %s", ErrTooLarge, sizeText(limit))}
}

// sizeText is n bytes in the largest binary unit that divides it.
func sizeText(n int) string {
	switch {
	case n > 0 && n%(1<<20) == 0:
		return strconv.Itoa(n>>20) + " MiB"
	case n > 0 && n%(1<<10) == 0:
		return strconv.Itoa(n>>10) + " KiB"
	}
	return strconv.Itoa(n) + " bytes"
}
