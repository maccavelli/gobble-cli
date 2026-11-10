package cli

import (
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/acpext"
)

// limitWords are the line session's words for each limit a failed turn
// names (0049-MADR Q3 (a)).
var limitWords = map[acpext.LimitScope]string{
	acpext.LimitRate:     "Rate limit",
	acpext.LimitWindow:   "Usage window limit",
	acpext.LimitDaily:    "Daily quota",
	acpext.LimitWeekly:   "Weekly quota",
	acpext.LimitMonthly:  "Monthly quota",
	acpext.LimitCredits:  "Credits used up",
	acpext.LimitFreeTier: "Free tier limit",
	acpext.LimitPlan:     "Plan usage limit",
}

// limitLine is the line a failed turn's data adds under its error when it
// names a limit or when the limit lifts: "<limit words> on <provider>;
// resets at HH:MM, in <duration>", the time on now's clock, the reset part
// left out when there is no time and its wait when the time has passed. A
// limit the table does not know is named as it is. ok is false when the
// data names neither.
func limitLine(d acpext.ErrorData, now time.Time) (line string, ok bool) {
	reset, rerr := time.Parse(time.RFC3339, d.ResetAt)
	hasReset := d.ResetAt != "" && rerr == nil
	if d.Limit == "" && !hasReset {
		return "", false
	}
	words := "Limit"
	if d.Limit != "" {
		words = limitWords[d.Limit]
		if words == "" {
			words = string(d.Limit)
		}
	}
	if d.Provider != "" {
		words += " on " + d.Provider
	}
	if !hasReset {
		return words, true
	}
	words += "; resets at " + reset.In(now.Location()).Format("15:04")
	if wait := reset.Sub(now); wait > 0 {
		words += ", in " + shortWait(wait)
	}
	return words, true
}

// shortWait is d to the minute, as 1h33m, 2h or 2m, or to the second under
// a minute.
func shortWait(d time.Duration) string {
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	s := strings.TrimSuffix(d.Round(time.Minute).String(), "0s")
	if strings.HasSuffix(s, "h0m") {
		s = strings.TrimSuffix(s, "0m")
	}
	return s
}
