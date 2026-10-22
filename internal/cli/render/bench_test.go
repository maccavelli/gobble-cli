package render

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// benchReply is a reply with prose, a heading, a list, a quote, code, a
// table and a long URL (0028-PLAN P4's captures).
//
//go:embed testdata/reply.txt
var benchReply string

// renderReply is the view's path for one reply as it streams: chunks of 64
// bytes through Stream, styled, then wrapped at width.
func renderReply(reply string, width int) string {
	var md Stream
	var w Wrap
	var b strings.Builder
	on := term.Stream{Color: true}
	push := func(s string) {
		if s != "" {
			s = Style(Tables(s), on)
		}
		b.WriteString(w.Push(s, width))
	}
	for i := 0; i < len(reply); i += 64 {
		push(md.Push(reply[i:min(i+64, len(reply))]))
	}
	push(md.Flush())
	b.WriteString(w.Flush())
	return b.String()
}

// One reply as it streams (0030-MADR D5).
func BenchmarkReplyRender(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		renderReply(benchReply, 100)
	}
}

// One reply drawn again whole at a new width, as a reprint draws each of its
// pieces (0030-MADR D5).
func BenchmarkReplyRedraw(b *testing.B) {
	on := term.Stream{Color: true}
	b.ReportAllocs()
	for b.Loop() {
		var w Wrap
		_ = w.Push(Style(Tables(benchReply), on), 60) + w.Flush()
	}
}
