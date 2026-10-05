package render

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestSpinnerFrames(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var w syncBuffer
		sp := StartSpinner(t.Context(), &w, colorOn, false, "thinking")
		time.Sleep(350 * time.Millisecond)
		if err := sp.Stop(); err != nil {
			t.Fatal(err)
		}
		want := "\r| thinking\x1b[K\r/ thinking\x1b[K\r- thinking\x1b[K\r\\ thinking\x1b[K\r\x1b[K"
		if got := w.String(); got != want {
			t.Fatalf("spinner wrote %q\nwant          %q", got, want)
		}
	})
}

func TestSpinnerStopsOnCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var w syncBuffer
		ctx, cancel := context.WithCancel(t.Context())
		sp := StartSpinner(ctx, &w, colorOn, true, "x")
		time.Sleep(150 * time.Millisecond)
		cancel()
		synctest.Wait()
		n := len(w.String())
		time.Sleep(time.Second)
		if len(w.String()) != n {
			t.Fatal("spinner kept drawing after its context was cancelled")
		}
		if !strings.HasSuffix(w.String(), "\r\x1b[K") {
			t.Fatalf("line not cleared: %q", w.String())
		}
		if err := sp.Stop(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSpinnerSilentOffTerminal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var w syncBuffer
		sp := StartSpinner(t.Context(), &w, colorOff, true, "x")
		time.Sleep(time.Second)
		if err := sp.Stop(); err != nil {
			t.Fatal(err)
		}
		if w.String() != "" {
			t.Fatalf("spinner drew on a non-terminal: %q", w.String())
		}
	})
}
