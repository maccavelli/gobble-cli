package editor

import (
	"strings"
	"testing"
)

// A key's drawing writes at most the text it changed and 40 bytes besides:
// the synchronized update, the hidden cursor, and the moves (0030-MADR D5).
// A key typed at a long line's end, in its middle, and one that only moves
// the cursor, are each held to it.
func TestKeyDrawBudget(t *testing.T) {
	text := []rune(strings.Repeat("words fill the prompt ", 10))
	for _, tc := range []struct {
		name   string
		before []rune
		pos    int
		after  []rune
		cursor int
	}{
		{"a key at the end", text, len(text), append(append([]rune{}, text...), 'x'), len(text) + 1},
		{"a key in the middle", text, 40, append(append(append([]rune{}, text[:40]...), 'x'), text[40:]...), 41},
		{"the cursor moved", text, len(text), text, len(text) - 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			r := &renderer{out: &out}
			old, now := layout("> ", tc.before, tc.pos, 80), layout("> ", tc.after, tc.cursor, 80)
			mustDraw(t, r, old, 24)
			out.Reset()
			mustDraw(t, r, now, 24)
			changed := 0 // the bytes of each row after what its old and new text share
			for i, row := range now.rows {
				was := ""
				if i < len(old.rows) {
					was = old.rows[i]
				}
				changed += len(row) - len(commonPrefix(was, row))
			}
			if n := len(out.String()); n > changed+40 {
				t.Fatalf("%d bytes for %d runes changed, more than %d: %q", n, changed, changed+40, out.String())
			}
		})
	}
}
