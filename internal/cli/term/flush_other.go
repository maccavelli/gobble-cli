//go:build !linux && !darwin && !windows

package term

import (
	"errors"
	"os"
)

// FlushInput cannot empty a terminal's input on this system: it reports so,
// and gobble goes on without the flush (0063-MADR D18, row R78).
func FlushInput(*os.File) error {
	return errors.New("this system has no call to clear a terminal's input")
}
