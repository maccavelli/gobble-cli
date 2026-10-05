//go:build windows

package cli

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

// isBrokenPipe reports a write to a pipe whose reader has gone. Windows
// reports it as ERROR_BROKEN_PIPE, or ERROR_NO_DATA while the pipe is being
// closed; Go maps neither to EPIPE.
func isBrokenPipe(err error) bool {
	return errors.Is(err, windows.ERROR_BROKEN_PIPE) || errors.Is(err, windows.ERROR_NO_DATA) ||
		errors.Is(err, syscall.EPIPE)
}
