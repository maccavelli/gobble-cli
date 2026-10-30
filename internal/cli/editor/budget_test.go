package editor

import (
	"strings"
	"testing"
	"time"
)

// A key's drawing writes at most the text it changed and 40 bytes besides:
// the synchronized update, the hidden cursor, and the moves (0030-MADR D5).
// A key typed at a long line's end, in its middle, and one that only moves
// the cursor, are each held to it, as the Composer draws them (0035-PLAN
// P17 step 5).
func TestKeyDrawBudget(t *testing.T) {
	text := strings.Repeat("words fill the prompt ", 10)
	for _, tc := range []struct {
		name  string
		to    string // keys from the line's end, to the place the key is typed
		key   string
		after string
	}{
		{"a key at the end", "", "x", text + "x"},
		{"a key in the middle", "\x01" + strings.Repeat("\x1b[C", 40), "x", text[:40] + "x" + text[40:]},
		{"the cursor moved", "", "\x1b[D", text},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := keyLine(text)
			now := time.Now()
			for _, in := range keys(tc.to) {
				c.HandleInput(in, now)
			}
			drainTo(&strings.Builder{}, c)
			old, _, _ := c.View(80)
			var out strings.Builder
			for _, in := range keys(tc.key) {
				c.HandleInput(in, now)
			}
			n := drainTo(&out, c)
			if c.Text() != tc.after {
				t.Fatalf("the text is %q, want %q", c.Text(), tc.after)
			}
			now2, _, _ := c.View(80)
			changed := 0 // the bytes of each row after what its old and new text share
			for i, row := range now2 {
				was := ""
				if i < len(old) {
					was = old[i]
				}
				changed += len(row) - len(commonPrefix(was, row))
			}
			if n > changed+40 {
				t.Fatalf("%d bytes for %d bytes changed, more than %d: %q", n, changed, changed+40, out.String())
			}
		})
	}
}

// commonPrefix is the bytes a and b start with alike.
func commonPrefix(a, b string) string {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return a[:n]
}
