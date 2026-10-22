//go:build unix

package term

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// resizeNotices notices SIGWINCH.
func resizeNotices(ctx context.Context, _ *os.File) <-chan struct{} {
	ch := make(chan struct{}, 1)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGWINCH)
	go func() {
		defer signal.Stop(sig)
		for {
			select {
			case <-ctx.Done():
				return
			case <-sig:
				notify(ch)
			}
		}
	}()
	return ch
}
