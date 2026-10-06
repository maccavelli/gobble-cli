package acpserver

import (
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	// coalesceWait and coalesceMax bound a streamed chunk: at most one per
	// 50 ms, of at most 4 KiB (0008-MADR D19 item 2).
	coalesceWait = 50 * time.Millisecond
	coalesceMax  = 4 << 10
)

// coalescer gathers streamed text and sends it at most every coalesceWait,
// in chunks of at most coalesceMax bytes. send runs under the coalescer's
// lock, so a timed send has finished before flush returns, and nothing the
// caller sends after flush can overtake it.
type coalescer struct {
	mu    sync.Mutex
	buf   strings.Builder
	timer *time.Timer
	send  func(string)
}

func newCoalescer(send func(string)) *coalescer { return &coalescer{send: send} }

// add buffers text, sending full chunks at once and arming the timer for
// the rest.
func (c *coalescer) add(text string) {
	if text == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buf.WriteString(text)
	for c.buf.Len() >= coalesceMax {
		s := c.buf.String()
		cut := coalesceMax
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		c.buf.Reset()
		c.buf.WriteString(s[cut:])
		c.send(s[:cut])
	}
	if c.buf.Len() > 0 && c.timer == nil {
		c.timer = time.AfterFunc(coalesceWait, c.fire)
	}
}

func (c *coalescer) fire() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timer = nil
	c.sendLocked()
}

// flush sends what is buffered now.
func (c *coalescer) flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.sendLocked()
}

func (c *coalescer) sendLocked() {
	if c.buf.Len() == 0 {
		return
	}
	s := c.buf.String()
	c.buf.Reset()
	c.send(s)
}
