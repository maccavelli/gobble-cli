//go:build !windows

package cli

import (
	"errors"
	"syscall"
)

// isBrokenPipe reports a write to a pipe whose reader has gone.
func isBrokenPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE)
}
