package fsx

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
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

// ReadRegularFile reads abs, a file outside any workspace, when it is a
// regular file of at most limit bytes.
func ReadRegularFile(abs string, limit int) ([]byte, error) {
	f, err := os.OpenFile(abs, os.O_RDONLY|nonblock, 0) //nolint:gosec // G304: a file the caller names, outside any workspace
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
