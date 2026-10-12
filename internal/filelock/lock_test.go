package filelock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// A second TryLock, through a second open in this process, finds the lock
// held, with this process's pid: flock and LockFileEx lock per open file.
// Releasing it lets the next one in.
func TestTryLockHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	release, err := TryLock(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = TryLock(path)
	held, ok := errors.AsType[*HeldError](err)
	if !ok || held.PID != os.Getpid() {
		t.Fatalf("second TryLock: %v, want *HeldError with pid %d", err, os.Getpid())
	}
	if err.Error() != "locked by pid "+strconv.Itoa(os.Getpid()) {
		t.Fatalf("message %q", err.Error())
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	again, err := TryLock(path)
	if err != nil {
		t.Fatalf("after release: %v", err)
	}
	if err := again(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the lock file should stay in place: %v", err)
	}
}

// Lock waits for a holder that lets go within its attempts, and gives up
// with *HeldError when the holder keeps it.
func TestLockRetries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	release, err := TryLock(path)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = release() //nolint:errcheck // the test's assertion is Lock's success
	}()
	got, err := Lock(t.Context(), path, 10, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("Lock after the holder let go: %v", err)
	}
	if _, err := Lock(t.Context(), path, 3, time.Millisecond); err == nil {
		t.Fatal("Lock while held: want *HeldError")
	} else if held := new(HeldError); !errors.As(err, &held) {
		t.Fatalf("Lock while held: %v, want *HeldError", err)
	}
	if err := got(); err != nil {
		t.Fatal(err)
	}
}

// A context that ends while Lock waits ends it with the context's error.
func TestLockContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	release, err := TryLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer release() //nolint:errcheck // cleanup
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Lock(ctx, path, 10, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("Lock with a cancelled context: %v, want context.Canceled", err)
	}
}
