package render

import (
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

func TestPlan(t *testing.T) {
	items := []PlanItem{
		{Content: "read the code", Status: PlanCompleted},
		{Content: "fix\x1b[2J the\n test", Status: PlanInProgress},
		{Content: "run the suite", Status: PlanPending},
	}
	golden(t, "plan-color", Plan(items, colorOn, term.Unicode))
	golden(t, "plan-plain", Plan(items, colorOff, term.ASCII))
}

func TestStats(t *testing.T) {
	cases := []struct {
		ttft   time.Duration
		tokens int64
		gen    time.Duration
		want   string
	}{
		{420 * time.Millisecond, 104, 2 * time.Second, "ttft 400ms • 52 tok/s"},
		{1500 * time.Millisecond, 0, 2 * time.Second, "ttft 1.5s"},
		{time.Second, 10, 0, "ttft 1s"},
	}
	for _, c := range cases {
		if got := Stats(c.ttft, c.tokens, c.gen, term.Unicode); got != c.want {
			t.Errorf("Stats(%v, %d, %v) = %q, want %q", c.ttft, c.tokens, c.gen, got, c.want)
		}
	}
}
