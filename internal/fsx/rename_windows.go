package fsx

import (
	"errors"

	"golang.org/x/sys/windows"
)

// transient reports whether a rename failed only because another process
// holds the file: access denied or a sharing violation.
func transient(err error) bool {
	return errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}
