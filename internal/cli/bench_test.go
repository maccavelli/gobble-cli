package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// benchTranscript is a transcript of 4 MB of the benchmark reply, each a
// piece drawn as the view draws a reply.
func benchTranscript(tb testing.TB) *transcript {
	tb.Helper()
	reply, err := os.ReadFile("render/testdata/reply.txt")
	if err != nil {
		tb.Fatal(err)
	}
	s := term.Stream{TTY: true, Color: true, Width: 100}
	v := newView(&Output{caps: term.Caps{Out: s, Err: s}}, display{}, stepClock())
	tr := &transcript{}
	for range maxTranscript / len(reply) {
		tr.open(v.replyAt).add(tr, string(reply))
	}
	return tr
}

// A reprint at 60 columns of the last 10,000 rows of a 4 MB transcript, the
// most D3 writes (0030-MADR D5).
func BenchmarkReprint(b *testing.B) {
	tr := benchTranscript(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		tr.render(60, maxReprintRows)
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
