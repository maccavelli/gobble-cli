package render

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// spinInterval is the time between frames.
const spinInterval = 100 * time.Millisecond

var (
	unicodeFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	asciiFrames   = []string{"|", "/", "-", `\`}
)

// Spinner draws a frame and a label on w every spinInterval until stopped.
type Spinner struct {
	wg     sync.WaitGroup
	cancel context.CancelFunc
	err    error
}

// StartSpinner starts a spinner on w, the error writer, whose stream is s.
// When s is not a terminal it draws nothing. It stops when ctx is done or
// Stop is called, and then clears its line.
func StartSpinner(ctx context.Context, w io.Writer, s term.Stream, unicode bool, label string) *Spinner {
	ctx, cancel := context.WithCancel(ctx)
	sp := &Spinner{cancel: cancel}
	if !s.TTY {
		return sp
	}
	frames := asciiFrames
	if unicode {
		frames = unicodeFrames
	}
	sp.wg.Go(func() {
		t := time.NewTicker(spinInterval)
		defer t.Stop()
		for i := 0; ; i++ {
			if _, sp.err = io.WriteString(w, "\r"+frames[i%len(frames)]+" "+label+"\x1b[K"); sp.err != nil {
				return
			}
			select {
			case <-ctx.Done():
				_, sp.err = io.WriteString(w, "\r\x1b[K")
				return
			case <-t.C:
			}
		}
	})
	return sp
}

// Stop stops the spinner, waits for it to clear its line, and returns the
// first write error, if any.
func (sp *Spinner) Stop() error {
	sp.cancel()
	sp.wg.Wait()
	return sp.err
}
