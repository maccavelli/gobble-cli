package render

import (
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// The bar turns at the phone's thresholds, 75% and 90% (0008-MADR F27).
func TestStatusLineThresholds(t *testing.T) {
	cases := []struct {
		name       string
		used, size int64
		sgr        string
	}{
		{"74.9% is green", 749, 1000, "\x1b[32m"},
		{"75% is yellow", 750, 1000, "\x1b[33m"},
		{"89.9% is yellow", 899, 1000, "\x1b[33m"},
		{"90% is red", 900, 1000, "\x1b[31m"},
		{"over 100% is red", 1200, 1000, "\x1b[31m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StatusLine(time.Second, tc.used, tc.size, nil, colorOn, term.Unicode)
			if !strings.Contains(got, tc.sgr) {
				t.Fatalf("StatusLine = %q, want colour %q", got, tc.sgr)
			}
		})
	}
}

func TestStatusLine(t *testing.T) {
	cost := 0.0123
	cases := []struct {
		name       string
		elapsed    time.Duration
		used, size int64
		cost       *float64
		g          term.Glyphs
		want       string
	}{
		{"half, with cost", 62*time.Second + 400*time.Millisecond, 100_000, 200_000, &cost, term.Unicode,
			"1m2s • ━━━━━━━━━━╌╌╌╌╌╌╌╌╌╌ 50% 100.0k/200.0k • $0.0123"},
		{"ASCII glyphs", 3 * time.Second, 950, 1000, nil, term.ASCII,
			"3s * ###################- 95% 950/1.0k"},
		{"no size, no bar", 1500 * time.Millisecond, 10, 0, nil, term.Unicode, "2s"},
		{"millions", time.Second, 1_500_000, 2_000_000, nil, term.ASCII,
			"1s * ###############----- 75% 1.5M/2.0M"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StatusLine(tc.elapsed, tc.used, tc.size, tc.cost, colorOff, tc.g)
			if got != tc.want {
				t.Fatalf("StatusLine = %q\nwant        %q", got, tc.want)
			}
		})
	}
}
