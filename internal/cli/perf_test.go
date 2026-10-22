//go:build perf

package cli

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/render"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// perfSlack is how many times its baseline a benchmark may take before
// `make perf` fails: timing varies by host and by load (0030-MADR D5).
const perfSlack = 3

// perfBaselines are each benchmark's median time per operation, of five
// runs, on the hosts 0030-PLAN P4 measured after its rework, by GOOS.
var perfBaselines = map[string]map[string]time.Duration{
	"darwin": {
		"ReplyRender":      30 * time.Microsecond,
		"Reprint":          9400 * time.Microsecond,
		"LineSessionStart": 830 * time.Microsecond,
	},
	"linux": {
		"ReplyRender":      110 * time.Microsecond,
		"Reprint":          29 * time.Millisecond,
		"LineSessionStart": 2900 * time.Microsecond,
	},
	"windows": {
		"ReplyRender":      54 * time.Microsecond,
		"Reprint":          12 * time.Millisecond,
		"LineSessionStart": 6900 * time.Microsecond,
	},
}

// perfCheck runs bench and fails t if it takes more than perfSlack times
// its baseline for this GOOS; with no baseline it reports the time only.
func perfCheck(t *testing.T, baselines map[string]map[string]time.Duration, name string, bench func(*testing.B)) {
	t.Helper()
	r := testing.Benchmark(bench)
	per := time.Duration(r.NsPerOp())
	base, ok := baselines[runtime.GOOS][name]
	if !ok {
		t.Logf("%s: %v per op, %d allocs; no baseline for %s", name, per, r.AllocsPerOp(), runtime.GOOS)
		return
	}
	t.Logf("%s: %v per op (baseline %v), %d allocs", name, per, base, r.AllocsPerOp())
	if per > perfSlack*base {
		t.Errorf("%s: %v per op, more than %d times its baseline %v", name, per, perfSlack, base)
	}
}

// benchReplyRender is a reply rendered as the view renders it while it
// streams: chunks of 64 bytes through render.Stream, styled, wrapped at
// 100 columns.
func benchReplyRender(b *testing.B) {
	reply, err := os.ReadFile("render/testdata/reply.txt")
	if err != nil {
		b.Fatal(err)
	}
	on := term.Stream{Color: true}
	b.ReportAllocs()
	for b.Loop() {
		var md render.Stream
		var w render.Wrap
		var out strings.Builder
		push := func(s string) {
			if s != "" {
				s = render.Style(render.Tables(s), on)
			}
			out.WriteString(w.Push(s, 100))
		}
		for i := 0; i < len(reply); i += 64 {
			push(md.Push(string(reply[i:min(i+64, len(reply))])))
		}
		push(md.Flush())
		out.WriteString(w.Flush())
	}
}

// The CLI mode's timings held to their baselines (0030-MADR D5), run by
// `make perf` on each host.
func TestPerfBudget(t *testing.T) {
	perfCheck(t, perfBaselines, "ReplyRender", benchReplyRender)
	perfCheck(t, perfBaselines, "Reprint", BenchmarkReprint)
	perfCheck(t, perfBaselines, "LineSessionStart", BenchmarkLineSessionStart)
}
