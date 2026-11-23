package fsx

import (
	"errors"

	"golang.org/x/sys/windows"
)

// sharingViolation reports whether an open was refused because another
// handle holds the file without sharing what the open needs: a rename
// replacing it, in the middle of its move (0067-MADR D1). Access denied is
// not one: on a read it comes only from a file pending deletion.
func sharingViolation(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}
