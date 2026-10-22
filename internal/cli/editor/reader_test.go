package editor

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// heldInput is stdin whose reads wait until the test gives them bytes; it
// counts the reads started.
type heldInput struct {
	mu      sync.Mutex
	started int
	give    chan string
}

func (h *heldInput) Read(p []byte) (int, error) {
	h.mu.Lock()
	h.started++
	h.mu.Unlock()
	s, ok := <-h.give
	if !ok {
		return 0, io.EOF
	}
	return copy(p, s), nil
}

func (h *heldInput) reads() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.started
}

// A resize, or the end of a wait, stops ReadUntil with no bytes and the
// read it asked for still asked: the next call is served by it, and no
// second read of stdin starts (0030-MADR D2).
func TestReaderWakes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in := &heldInput{give: make(chan string)}
		ctx, cancel := context.WithCancel(t.Context())
		defer func() {
			cancel()
			close(in.give)
		}()
		var r Reader
		r.Start(ctx, in)
		resize := make(chan struct{}, 1)
		buf := make([]byte, 16)
		done := make(chan error)
		go func() {
			_, err := r.ReadUntil(t.Context(), buf, resize, nil)
			done <- err
		}()
		synctest.Wait()
		resize <- struct{}{}
		if err := <-done; !errors.Is(err, errResized) {
			t.Fatalf("err %v, want errResized", err)
		}
		timeout := make(chan time.Time, 1)
		timeout <- time.Now()
		if _, err := r.ReadUntil(t.Context(), buf, nil, timeout); !errors.Is(err, errWaitOver) {
			t.Fatalf("err %v, want errWaitOver", err)
		}
		got := make(chan string)
		go func() {
			n, _ := r.ReadUntil(t.Context(), buf, nil, nil)
			got <- string(buf[:n])
		}()
		synctest.Wait()
		in.give <- "ab"
		if s := <-got; s != "ab" {
			t.Fatalf("read %q, want %q", s, "ab")
		}
		if n := in.reads(); n != 1 {
			t.Fatalf("%d reads of stdin, want 1", n)
		}
	})
}

// The reader reads stdin only when asked: nothing at Start, one read for a
// ReadContext that finds nothing buffered, and none after it. A read
// started ahead would take the console's mode of that moment, a line read
// on Windows outside raw mode, and keep it (0027-MADR D5, as amended for
// P5).
func TestReaderReadsOnlyWhenAsked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in := &heldInput{give: make(chan string)}
		ctx, cancel := context.WithCancel(t.Context())
		defer func() { // unblocks the reader on any way out, so the bubble ends
			cancel()
			close(in.give)
		}()
		var r Reader
		r.Start(ctx, in)
		synctest.Wait()
		if n := in.reads(); n != 0 {
			t.Fatalf("%d reads started before any was asked for", n)
		}
		got := make(chan string)
		go func() {
			buf := make([]byte, 16)
			n, err := r.ReadContext(t.Context(), buf)
			if err != nil {
				t.Error(err)
			}
			got <- string(buf[:n])
		}()
		synctest.Wait()
		if n := in.reads(); n != 1 {
			t.Fatalf("%d reads started for one ask, want 1", n)
		}
		in.give <- "ab"
		if s := <-got; s != "ab" {
			t.Fatalf("read %q, want %q", s, "ab")
		}
		synctest.Wait()
		if n := in.reads(); n != 1 {
			t.Fatalf("%d reads started after the one asked for was served, want 1", n)
		}
	})
}
