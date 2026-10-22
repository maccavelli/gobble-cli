//go:build perf

package editor

import (
	"runtime"
	"testing"
	"time"
)

// keyDrawBaseline is BenchmarkKeyDraw's time per key on the hosts 0030-PLAN
// P4 measured after its rework, by GOOS; `make perf` fails past three times
// it (0030-MADR D5).
var keyDrawBaseline = map[string]time.Duration{
	"darwin":  1500 * time.Nanosecond,
	"linux":   5000 * time.Nanosecond,
	"windows": 1400 * time.Nanosecond,
}

// A key's drawing held to its baseline, run by `make perf` on each host.
func TestPerfBudget(t *testing.T) {
	r := testing.Benchmark(BenchmarkKeyDraw)
	per := time.Duration(r.NsPerOp())
	base, ok := keyDrawBaseline[runtime.GOOS]
	if !ok {
		t.Logf("KeyDraw: %v per op, %d allocs; no baseline for %s", per, r.AllocsPerOp(), runtime.GOOS)
		return
	}
	t.Logf("KeyDraw: %v per op (baseline %v), %d allocs", per, base, r.AllocsPerOp())
	if per > 3*base {
		t.Errorf("KeyDraw: %v per op, more than 3 times its baseline %v", per, base)
	}
}
