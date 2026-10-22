package term

import (
	"context"
	"os"
	"time"

	xterm "golang.org/x/term"
)

// WatchResize notices when the terminal f is resized, until ctx is done:
// a value on the channel for each resize, resizes coalesced while one
// waits to be taken (0030-MADR D1). On macOS and Linux it is SIGWINCH; on
// Windows the console's size, read every 100 ms, since a reader in
// virtual-terminal input mode never receives the console's size event
// (0030-PLAN P1 step 2). A file that is not a terminal is never resized:
// the channel is nil.
func WatchResize(ctx context.Context, f *os.File) <-chan struct{} {
	if f == nil || !xterm.IsTerminal(int(f.Fd())) { //nolint:gosec // G115: a terminal's descriptor fits an int
		return nil
	}
	return resizeNotices(ctx, f)
}

// pollSize reads size every interval until ctx is done, and notices each
// change.
func pollSize(ctx context.Context, size func() (int, int), every time.Duration) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		w, h := size()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			if nw, nh := size(); nw != w || nh != h {
				w, h = nw, nh
				notify(ch)
			}
		}
	}()
	return ch
}

// notify puts a notice on ch unless one already waits there.
func notify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
