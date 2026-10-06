package acpserver

import (
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
	"unicode/utf8"
)

type sent struct {
	mu     sync.Mutex
	chunks []string
}

func (s *sent) add(c string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chunks = append(s.chunks, c)
}

func (s *sent) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.chunks...)
}

// 200 deltas inside 10 ms are one chunk, sent 50 ms after the first.
func TestCoalesceByTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var s sent
		c := newCoalescer(s.add)
		for range 200 {
			c.add("x")
			time.Sleep(50 * time.Microsecond)
		}
		if got := s.all(); len(got) != 0 {
			t.Fatalf("sent %d chunks before the wait", len(got))
		}
		time.Sleep(coalesceWait)
		synctest.Wait()
		if got := s.all(); len(got) != 1 || got[0] != strings.Repeat("x", 200) {
			t.Fatalf("chunks %q, want one of 200 bytes", got)
		}
	})
}

// A 10 KiB delta is three chunks of at most 4 KiB, two at once and the rest
// after the wait; a multibyte rune is never split.
func TestCoalesceBySize(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var s sent
		c := newCoalescer(s.add)
		c.add(strings.Repeat("é", 5*1024)) // 10 KiB
		if got := s.all(); len(got) != 2 {
			t.Fatalf("sent %d chunks at once, want 2", len(got))
		}
		time.Sleep(coalesceWait)
		synctest.Wait()
		got := s.all()
		total := 0
		for _, ch := range got {
			if len(ch) > coalesceMax || !utf8.ValidString(ch) {
				t.Fatalf("chunk of %d bytes, valid %v", len(ch), utf8.ValidString(ch))
			}
			total += len(ch)
		}
		if len(got) != 3 || total != 10*1024 {
			t.Fatalf("%d chunks, %d bytes", len(got), total)
		}
	})
}

// flush sends at once and disarms the timer.
func TestCoalesceFlush(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var s sent
		c := newCoalescer(s.add)
		c.add("a")
		c.flush()
		time.Sleep(2 * coalesceWait)
		synctest.Wait()
		if got := s.all(); len(got) != 1 || got[0] != "a" {
			t.Fatalf("chunks %q", got)
		}
	})
}
