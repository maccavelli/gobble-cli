//go:build !windows

package fsx

import (
	"errors"
	"io/fs"
	"syscall"
	"testing"
)

// Off Windows no open error is a sharing violation, so a read is tried
// once (0067-MADR R8).
func TestSharingViolationNever(t *testing.T) {
	for _, err := range []error{
		syscall.EBUSY, syscall.EACCES, fs.ErrNotExist, fs.ErrPermission, errors.New("plain"),
		&fs.PathError{Op: "open", Path: "x", Err: syscall.ETXTBSY},
	} {
		if sharingViolation(err) {
			t.Errorf("sharingViolation(%v) = true; want false off Windows", err)
		}
	}
}
