//go:build unix

package term

import (
	"os"
	"syscall"
)

// ShutdownSignals returns the signals that cancel a run. Each one exits with
// 128 plus its number (0008-MADR D10).
func ShutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}
