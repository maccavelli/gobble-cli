package term

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

// A change of size is noticed once, a steady size not at all, and changes
// made while one notice waits are coalesced into it (0030-MADR D1).
func TestPollSizeNotices(t *testing.T) {
	var mu sync.Mutex
	w, h := 80, 24
	size := func() (int, int) {
		mu.Lock()
		defer mu.Unlock()
		return w, h
	}
	set := func(nw, nh int) {
		mu.Lock()
		w, h = nw, nh
		mu.Unlock()
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ch := pollSize(ctx, size, 5*time.Millisecond)
	if ch == nil {
		t.Fatal("no channel")
	}
	select {
	case <-ch:
		t.Fatal("noticed a resize while the size stood still")
	case <-time.After(50 * time.Millisecond):
	}
	set(43, 24)
	time.Sleep(20 * time.Millisecond)
	set(60, 30)
	time.Sleep(20 * time.Millisecond)
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("a resize was not noticed")
	}
	select {
	case <-ch:
		t.Fatal("two resizes made while a notice waited gave two notices")
	case <-time.After(50 * time.Millisecond):
	}
}

// A file that is not a terminal is never resized.
func TestWatchResizeNotATerminal(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck // test cleanup
	if WatchResize(t.Context(), f) != nil {
		t.Fatal("a regular file has a resize channel")
	}
}
