//go:build windows

package term

import (
	"context"
	"os"
	"time"

	xterm "golang.org/x/term"
)

// resizeNotices reads the console's size every 100 ms.
func resizeNotices(ctx context.Context, f *os.File) <-chan struct{} {
	return pollSize(ctx, func() (int, int) {
		w, h, err := xterm.GetSize(int(f.Fd())) //nolint:gosec // G115: a console handle fits an int
		if err != nil {
			return 0, 0 // no size read; the next that is counts as a change
		}
		return w, h
	}, 100*time.Millisecond)
}
