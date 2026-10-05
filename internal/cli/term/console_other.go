//go:build !windows

package term

import "os"

// PrepareConsole is a no-op outside Windows: terminals there interpret VT
// sequences without a mode switch.
func PrepareConsole(_, _ *os.File) (restore func(), _ error) {
	return func() {}, nil
}
