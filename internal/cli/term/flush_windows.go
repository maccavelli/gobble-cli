//go:build windows

package term

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// FlushInput empties what the console on f holds of its input, typed or
// pasted and not yet read: FlushConsoleInputBuffer (0063-MADR D12, D18). On
// a handle that is not a console's input it fails, at once.
func FlushInput(f *os.File) error {
	if f == nil {
		return errors.New("no standard input")
	}
	return windows.FlushConsoleInputBuffer(windows.Handle(f.Fd()))
}
