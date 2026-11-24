package render

import (
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
)

func TestPlan(t *testing.T) {
	items := []PlanItem{
		{Content: "read the code", Status: PlanCompleted},
		{Content: "fix\x1b[2J the\n test", Status: PlanInProgress},
		{Content: "run the suite", Status: PlanPending},
	}
	golden(t, "plan-color", Plan(items, colorOn, glyph.Unicode()))
	golden(t, "plan-plain", Plan(items, colorOff, glyph.ASCII()))
}

func TestStats(t *testing.T) {
	cases := []struct {
		ttft   time.Duration
		tokens int64
		gen    time.Duration
		limit  int64
		g      glyph.Set
		want   string
	}{
		{420 * time.Millisecond, 104, 2 * time.Second, 0, glyph.Unicode(), "ttft 400ms • 52 tok/s"},
		{1500 * time.Millisecond, 0, 2 * time.Second, 0, glyph.Unicode(), "ttft 1.5s"},
		{time.Second, 10, 0, 0, glyph.Unicode(), "ttft 1s"},
		// The output limit's suffix, in render.tokens's form (0071-MADR D10, L27).
		{420 * time.Millisecond, 104, 2 * time.Second, 64000, glyph.Unicode(), "ttft 400ms • 52 tok/s • out-limit 64.0k"},
		{time.Second, 0, 0, 950, glyph.ASCII(), "ttft 1s " + glyph.ASCII().Bullet + " out-limit 950"},
	}
	for _, c := range cases {
		if got := Stats(c.ttft, c.tokens, c.gen, c.limit, c.g); got != c.want {
			t.Errorf("Stats(%v, %d, %v, %d) = %q, want %q", c.ttft, c.tokens, c.gen, c.limit, got, c.want)
		}
	}
}
