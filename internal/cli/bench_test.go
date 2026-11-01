package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	tuitranscript "github.com/maccavelli/go-tui-lib/v2/transcript"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// benchTranscript is a transcript of 4 MB of the benchmark reply, each
// kept as the view keeps a reply: its Printer, given the reply.
func benchTranscript(tb testing.TB) *transcript {
	tb.Helper()
	reply, err := os.ReadFile("render/testdata/reply.txt")
	if err != nil {
		tb.Fatal(err)
	}
	s := term.Stream{TTY: true, Color: true, Width: 100}
	v := newView(&Output{caps: term.Caps{Out: s, Err: s}}, display{}, stepClock())
	tr := newTranscript(100)
	keepReplies(tr, v, string(reply), tuitranscript.DefaultMaxBytes/len(reply))
	return tr
}

// keepReplies keeps n replies in tr, each a Printer of v's given reply
// whole, as the view keeps one.
func keepReplies(tr *transcript, v *view, reply string, n int) {
	for range n {
		p := v.replyPrinter(100)
		tr.reply(p)
		tr.do(func() { p.Write(reply); p.Flush() })
	}
}

// A reprint at 60 columns of the last 10,000 rows of a 4 MB transcript, the
// most D3 writes (0030-MADR D5).
func BenchmarkReprint(b *testing.B) {
	tr := benchTranscript(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		reprintAt(tr, 60)
	}
}

// The line session from its start to its first prompt drawn, in process,
// with a scripted agent: the session's own start-up, the process's being
// measured in a terminal (0030-MADR D5).
func BenchmarkLineSessionStart(b *testing.B) {
	for b.Loop() {
		h := newLineHarness(b, &acptest.ScriptAgent{})
		h.noResize = true
		h.start(Prompt{})
		for !strings.Contains(h.stdout.String(), "> ") {
			time.Sleep(50 * time.Microsecond)
		}
		b.StopTimer()
		h.press("\x04")
		if err := h.wait(); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}
