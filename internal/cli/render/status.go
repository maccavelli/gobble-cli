package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

const (
	barCells = 20
	// warnAt and alertAt are the phone's context-chip thresholds, in percent
	// (0008-MADR F27): the bar turns yellow, then red.
	warnAt  = 75
	alertAt = 90
)

// StatusLine is the context line after a turn: elapsed time, a 20-cell bar
// of used against size, the percentage and token counts, and the cost when
// known. size 0 leaves out the bar.
func StatusLine(elapsed time.Duration, used, size int64, cost *float64, s term.Stream, g term.Glyphs) string {
	parts := []string{elapsed.Round(time.Second).String()}
	if size > 0 {
		pct := float64(used) * 100 / float64(size)
		filled := min(barCells, max(0, int(pct*barCells/100+0.5)))
		bar := strings.Repeat(g.Bar, filled) + strings.Repeat(g.BarEmpty, barCells-filled)
		parts = append(parts, fmt.Sprintf("%s %.0f%% %s/%s",
			barStyle(pct).Wrap(bar, s.Color), pct, tokens(used), tokens(size)))
	}
	if cost != nil {
		parts = append(parts, fmt.Sprintf("$%.4f", *cost))
	}
	return strings.Join(parts, " "+g.Bullet+" ")
}

// Stats is the --stats suffix of the status line: the time to the first
// token, then tokens per second when both the output tokens and the time
// spent generating them are known.
func Stats(ttft time.Duration, outputTokens int64, gen time.Duration, g term.Glyphs) string {
	s := "ttft " + ttft.Round(100*time.Millisecond).String()
	if outputTokens > 0 && gen > 0 {
		s += fmt.Sprintf(" %s %.0f tok/s", g.Bullet, float64(outputTokens)/gen.Seconds())
	}
	return s
}

func barStyle(pct float64) term.Style {
	switch {
	case pct >= alertAt:
		return term.Red
	case pct >= warnAt:
		return term.Yellow
	}
	return term.Green
}

// tokens formats a token count: 950, 12.3k, 1.2M.
func tokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	}
	return strconv.FormatInt(n, 10)
}
