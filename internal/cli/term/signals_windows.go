//go:build windows

package term

import "os"

// ShutdownSignals returns the signals that cancel a run.
//
// Windows never delivers SIGTERM. os.Interrupt covers CTRL_C_EVENT,
// CTRL_BREAK_EVENT and CTRL_CLOSE_EVENT for a console process, so it stands
// alone. Provenance: magic-cli-remote internal/cli/signals_windows.go.
func ShutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
