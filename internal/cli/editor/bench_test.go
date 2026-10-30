package editor

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// keyLine is a Composer at 80 columns, 24 rows tall, holding text, its
// first drawing written.
func keyLine(text string) *lineedit.Composer {
	r := inline.NewRegion(inline.WithHeight(24))
	c := lineedit.NewComposer(lineedit.WithRegion(r), lineedit.WithPrompt("> "))
	now := time.Now()
	c.Resize(80, now)
	c.SetText(text)
	drainTo(io.Discard, c)
	return c
}

// drainTo writes what c asks to w, and returns how many bytes it wrote.
func drainTo(w io.Writer, c *lineedit.Composer) int {
	n := 0
	for ev := c.NextEvent(); ev.Kind != lineedit.EditNone; ev = c.NextEvent() {
		if ev.Kind == lineedit.EditWrite {
			m, _ := io.WriteString(w, ev.Data) //nolint:errcheck // a builder or io.Discard
			n += m
		}
	}
	return n
}

// keys decodes s as a terminal's input.
func keys(s string) []termkey.Input {
	var d termkey.Decoder
	return d.Flush(d.Decode([]byte(s), nil))
}

// A key typed at the end of a three-row line, drawn (0030-MADR D5): the
// Composer's draw of one key (0035-PLAN P17 step 5), typed and taken back
// in turn.
func BenchmarkKeyDraw(b *testing.B) {
	c := keyLine(strings.Repeat("words fill the prompt ", 10))
	in := keys("x\x7f")
	var out strings.Builder
	now := time.Now()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		out.Reset()
		c.HandleInput(in[i%2], now)
		drainTo(&out, c)
	}
}
