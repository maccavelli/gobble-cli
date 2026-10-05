//go:build windows

package term

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"golang.org/x/sys/windows"
)

// PrepareConsole enables VT processing on stdout and stderr when they are
// consoles, and returns a func that restores the saved modes. A handle
// that is not a console is skipped, not an error. Provenance:
// mcp-server-magictools internal/ui/init_windows.go (Apache-2.0).
func PrepareConsole(out, err *os.File) (restore func(), _ error) {
	type saved struct {
		h    windows.Handle
		mode uint32
	}
	var done []saved
	var errs []error
	for _, f := range []*os.File{out, err} {
		if f == nil {
			continue
		}
		h := windows.Handle(f.Fd())
		var mode uint32
		if windows.GetConsoleMode(h, &mode) != nil {
			continue // not a console
		}
		want := mode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING | windows.ENABLE_PROCESSED_OUTPUT
		if want == mode {
			continue // already set, or the same console as the previous stream
		}
		if e := windows.SetConsoleMode(h, want); e != nil {
			errs = append(errs, fmt.Errorf("enable virtual terminal processing on %s: %w", f.Name(), e))
			continue
		}
		done = append(done, saved{h: h, mode: mode})
	}
	return func() {
		for _, s := range slices.Backward(done) {
			// Best effort: restore runs as the process exits, with no one left to report to.
			_ = windows.SetConsoleMode(s.h, s.mode) //nolint:errcheck // see above
		}
	}, errors.Join(errs...)
}
