//go:build windows

package term

import (
	"os"

	"golang.org/x/sys/windows"
)

// isConsoleInput reports whether f is a console: a handle GetConsoleMode
// accepts (0063-MADR D19).
func isConsoleInput(f *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(f.Fd()), &mode) == nil
}
