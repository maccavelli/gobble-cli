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
	got, err := Lock(t.Context(), path, EditPolicy)
	if err != nil {
		t.Fatalf("Lock after the holder let go: %v", err)
	}
	if _, err := Lock(t.Context(), path, Policy{First: time.Millisecond, Max: time.Millisecond, Deadline: 3 * time.Millisecond}); err == nil {
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
	if _, err := Lock(ctx, path, Policy{First: time.Hour, Max: time.Hour, Deadline: 2 * time.Hour}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Lock with a cancelled context: %v, want context.Canceled", err)
	}
}

// EditPolicy outlasts a holder that keeps the lock past Pi's budget of 10
// tries 20 ms apart, about 200 ms: the failure TestAddConcurrent showed
// once under load, made deterministic (0018-PLAN P7).
func TestLockOutlastsPiBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	release, err := TryLock(path)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = release() //nolint:errcheck // the test's assertion is Lock's success
	}()
	got, err := Lock(t.Context(), path, EditPolicy)
	if err != nil {
		t.Fatalf("Lock with EditPolicy while held for 300 ms: %v", err)
	}
	if err := got(); err != nil {
		t.Fatal(err)
	}
}

// Lock gives up with *HeldError once its deadline has passed, and not
// before it.
func TestLockDeadline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	release, err := TryLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer release() //nolint:errcheck // cleanup
	p := Policy{First: 10 * time.Millisecond, Max: 40 * time.Millisecond, Deadline: 150 * time.Millisecond}
	start := time.Now()
	_, err = Lock(t.Context(), path, p)
	took := time.Since(start)
	if held := new(HeldError); !errors.As(err, &held) {
		t.Fatalf("Lock while held: %v, want *HeldError", err)
	}
	// The last wait is cut to end at Deadline, where Lock tries once more;
	// the upper bound adds one gap at Max and scheduling slack.
	if lo, hi := p.Deadline, p.Deadline+p.Max+500*time.Millisecond; took < lo || took > hi {
		t.Fatalf("Lock gave up after %v, want between %v and %v", took, lo, hi)
	}
}

// Each wait but the last is drawn from half to all of its gap, and the
// draws vary; the wait that reaches the deadline is exactly the time left
// (0037-PLAN deviation 9, deviation 10).
func TestLockJitter(t *testing.T) {
	const d, left = 100 * time.Millisecond, time.Second
	seen := map[time.Duration]bool{}
	for range 1000 {
		w := spread(d, left)
		if w < d/2 || w > d {
			t.Fatalf("spread(%v, %v) = %v, want within [%v, %v]", d, left, w, d/2, d)
		}
		seen[w] = true
	}
	if len(seen) < 100 {
		t.Errorf("1000 draws gave %d distinct waits, want them spread", len(seen))
	}
	for _, d := range []time.Duration{left, 2 * left} {
		if w := spread(d, left); w != left {
			t.Errorf("spread(%v, %v) = %v, want exactly %v", d, left, w, left)
		}
	}
}
