package cli

import (
	"fmt"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/personality"
)

// The ambient lines' thresholds (0022-MADR D12).
const (
	slowAfter     = 30 * time.Second
	verySlowAfter = 3 * time.Minute
	longTurn      = 10 * time.Minute
	costMilestone = 1.0
	callMilestone = 100
	lateNightEnds = 5 // the late-night welcome runs from midnight to 5 a.m.
)

// ambient is the line view's ambient text (0022-MADR D12): the welcome and
// the goodbye, the slow turn's spinner label, and the milestones, once each
// a session. Its budget is one line a turn, the welcome sharing the first
// turn's; it says nothing when stderr is not a terminal, where the spinner
// draws nothing either; and its deck repeats no line while unused ones
// remain. Updates arrive on the connection's goroutine and the spinner asks
// on its own, so it is locked.
type ambient struct {
	on   bool
	deck personality.Deck

	mu    sync.Mutex
	turns int
	spent bool
	// slow is the slow key the spinner shows this turn, and line its text.
	slow personality.Key
	line string
	// The session's milestones: those said, and what they are made of.
	shown     map[personality.Key]bool
	compacted bool
	long      bool
	calls     int
	cost      *acpclient.Cost
}

func newAmbient(on bool) *ambient { return &ambient{on: on, shown: map[personality.Key]bool{}} }

// welcomeKey is the welcome for the time t: Thanksgiving on the fourth
// Thursday of November, late at night from midnight to five, Monday, the
// weekend, else the plain one.
func welcomeKey(t time.Time) personality.Key {
	switch {
	case t.Month() == time.November && t.Weekday() == time.Thursday && (t.Day()-1)/7 == 3:
		return personality.SessionWelcomeThanksgiving
	case t.Hour() < lateNightEnds:
		return personality.SessionWelcomeLateNight
	case t.Weekday() == time.Monday:
		return personality.SessionWelcomeMonday
	case t.Weekday() == time.Saturday, t.Weekday() == time.Sunday:
		return personality.SessionWelcomeWeekend
	}
	return personality.SessionWelcome
}

// welcome is the line view's first line, for the day, or the plain one
// when the voice has none for the day.
func (a *ambient) welcome(v *personality.Voice, now time.Time) (string, bool) {
	if !a.on {
		return "", false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.spent {
		return "", false
	}
	line, ok := a.deck.Deal(v, welcomeKey(now), nil)
	if !ok {
		line, ok = a.deck.Deal(v, personality.SessionWelcome, nil)
	}
	a.spent = ok
	return line, ok
}

// goodbye is the line view's last line.
func (a *ambient) goodbye(v *personality.Voice) (string, bool) {
	if !a.on {
		return "", false
	}
	return a.deck.Deal(v, personality.SessionGoodbye, nil)
}

// startTurn opens a turn's budget; the first turn's is the welcome's too.
func (a *ambient) startTurn() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.turns > 0 {
		a.spent = false
	}
	a.turns++
	a.slow, a.line = "", ""
}

// label is the spinner's label, elapsed into a turn with no output yet:
// turn.slow after 30 seconds and turn.very_slow, in its place, after 3
// minutes, when the budget allows and the voice has them; else a working
// line. The spinner runs only on a terminal, before the turn's first
// output, and stops while a question waits.
func (a *ambient) label(v *personality.Voice, elapsed time.Duration) string {
	var key personality.Key
	switch {
	case elapsed >= verySlowAfter:
		key = personality.TurnVerySlow
	case elapsed >= slowAfter:
		key = personality.TurnSlow
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.on && key != "" && a.slow != key && (a.slow != "" || !a.spent) {
		if line, ok := a.deck.Deal(v, key, nil); ok {
			a.slow, a.line, a.spent = key, line, true
		}
	}
	if a.slow != "" {
		return a.line
	}
	return v.Text(personality.TurnWorking, nil)
}

// note keeps what the milestones are made of, from the session's updates:
// its tool calls, a compaction that ended well, and its cost.
func (a *ambient) note(u acpclient.Update) {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch u.Kind {
	case acpclient.KindToolCall:
		a.calls++
	case acpclient.KindCompaction:
		if u.Compaction.Phase == "end" && u.Compaction.Error == "" {
			a.compacted = true
		}
	case acpclient.KindUsage:
		if c := u.Context.Cost; c != nil {
			a.cost = c
		}
	default:
	}
}

// endTurn is the milestone a turn that took d ends with, when the budget
// allows: the first compaction, a turn past ten minutes, a cost past a
// dollar, the hundredth tool call, in that order, each once a session. One
// the budget holds back is said at a later turn's end.
func (a *ambient) endTurn(v *personality.Voice, d time.Duration) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if d >= longTurn {
		a.long = true
	}
	if !a.on || a.spent {
		return "", false
	}
	for _, m := range []struct {
		key     personality.Key
		reached bool
	}{
		{personality.MilestoneFirstCompaction, a.compacted},
		{personality.MilestoneLongTurn, a.long},
		{personality.MilestoneCost, a.cost != nil && a.cost.Amount >= costMilestone},
		{personality.MilestoneHundredthCall, a.calls >= callMilestone},
	} {
		if !m.reached || a.shown[m.key] {
			continue
		}
		a.shown[m.key] = true
		var args personality.Args
		if m.key == personality.MilestoneCost {
			args = personality.Args{"cost": costText(*a.cost)}
		}
		if line, ok := a.deck.Deal(v, m.key, args); ok {
			a.spent = true
			return line, true
		}
	}
	return "", false
}

// newSession starts the milestones again, for the session /new or a
// switch opens.
func (a *ambient) newSession() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.shown = map[personality.Key]bool{}
	a.compacted, a.long, a.calls, a.cost = false, false, 0, nil
}

// costText is a cost as the milestone says it: dollars, or the amount and
// its currency.
func costText(c acpclient.Cost) string {
	if c.Currency == "" || c.Currency == "USD" {
		return fmt.Sprintf("$%.2f", c.Amount)
	}
	return fmt.Sprintf("%.2f %s", c.Amount, c.Currency)
}

// The turkey /gobble draws, in the terminal's glyph set (0022-MADR D12).
const (
	turkeyUnicode = `   ╲ │ ╱
  ─ (◉▸ ─
   ╱(▓▓)╲
     ╯╰
`
	turkeyASCII = `   \ | /
  -- (o> --
   /(##)\
     ^^
`
)

// gobbleText is /gobble's answer: the turkey, and a line in the voice;
// boring has none, so its turkey stands alone.
func (a *ambient) gobbleText(v *personality.Voice, unicode bool) string {
	text := turkeyASCII
	if unicode {
		text = turkeyUnicode
	}
	if line, ok := a.deck.Deal(v, personality.EasterGobble, nil); ok {
		text += line + "\n"
	}
	return text
}
