package cli

import (
	"fmt"
	"os"
	"syscall"
)

// Exit codes (0008-MADR D10).
const (
	ExitOK              = 0
	ExitFailure         = 1
	ExitUsage           = 2
	ExitAuth            = 3
	ExitCancelled       = 4
	ExitUpdateAvailable = 10
)

// exitForSignal is 128 plus the signal number: 130 for an interrupt, 129
// for SIGHUP, 143 for SIGTERM.
func exitForSignal(s os.Signal) int {
	if n, ok := s.(syscall.Signal); ok {
		return 128 + int(n)
	}
	return ExitFailure
}

// exitPanic is what Kong's Exit option panics with, so that parsing stops
// at once (after --help, say) and Main recovers the code. Kong's own exit
// would end the process from inside the parser.
type exitPanic struct{ code int }

// exitError is a command's failure: the code to exit with and the message
// Main prints as "Error: <msg>". An empty message prints nothing.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func usageErrorf(format string, args ...any) error {
	return &exitError{code: ExitUsage, msg: fmt.Sprintf(format, args...)}
}

func failf(format string, args ...any) error {
	return &exitError{code: ExitFailure, msg: fmt.Sprintf(format, args...)}
}
