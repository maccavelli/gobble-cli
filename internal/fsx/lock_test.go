package fsx

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// A second holder of one path waits for the first; a holder of another
// path does not.
func TestLockSerialises(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	unlock := Lock(a)

	got := make(chan string, 2)
	var wg sync.WaitGroup
	wg.Go(func() {
		u := Lock(a)
		got <- "a"
		u()
	})
	wg.Go(func() {
		u := Lock(b)
		got <- "b"
		u()
	})
	select {
	case s := <-got:
		if s != "b" {
			t.Fatalf("%s acquired while a was held", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a lock on another path waited for this one")
	}
	select {
	case s := <-got:
		t.Fatalf("%s acquired while a was held", s)
	case <-time.After(100 * time.Millisecond):
	}
	unlock()
	select {
	case s := <-got:
		if s != "a" {
			t.Fatalf("got %s, want a after the unlock", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the waiter never acquired a after the unlock")
	}
	unlock() // a second call is harmless
	wg.Wait()
	locksMu.Lock()
	n := len(locks)
	locksMu.Unlock()
	if n != 0 {
		t.Fatalf("%d locks left in the map, want none", n)
	}
}
