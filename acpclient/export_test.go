package acpclient

import (
	"testing"
	"time"
)

// SetCloseTimeout makes Close wait at most d for the rest of t.
func SetCloseTimeout(t testing.TB, d time.Duration) {
	old := closeTimeout
	closeTimeout = d
	t.Cleanup(func() { closeTimeout = old })
}
