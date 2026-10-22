package editor

import (
	"strings"
	"testing"
)

// A key typed at the end of a three-row line, drawn (0030-MADR D5).
func BenchmarkKeyDraw(b *testing.B) {
	text := []rune(strings.Repeat("words fill the prompt ", 10))
	more := append(append([]rune{}, text...), 'x')
	short, long := layout("> ", text, len(text), 80), layout("> ", more, len(more), 80)
	var out strings.Builder
	r := &renderer{out: &out}
	if err := r.draw(short, 24); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		out.Reset()
		l := long
		if i%2 == 1 {
			l = short
		}
		if err := r.draw(l, 24); err != nil {
			b.Fatal(err)
		}
	}
}
