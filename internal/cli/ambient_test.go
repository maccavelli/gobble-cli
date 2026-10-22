package cli

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/personality"
)

func dryVoice(t *testing.T) *personality.Voice {
	t.Helper()
	v, err := personality.New(personality.Options{Name: personality.Dry})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// The welcome follows the calendar: Thanksgiving on the fourth Thursday of
// November, not the last; late at night, midnight to five; Monday; the
// weekend (0022-MADR D12).
func TestWelcomeKey(t *testing.T) {
	at := func(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, time.Local) }
	for _, c := range []struct {
		t    time.Time
		want personality.Key
	}{
		{at(2026, time.November, 26, 10), personality.SessionWelcomeThanksgiving},
		{at(2026, time.November, 26, 2), personality.SessionWelcomeThanksgiving},
		{at(2026, time.November, 19, 10), personality.SessionWelcome},
		{at(2029, time.November, 22, 10), personality.SessionWelcomeThanksgiving},
		{at(2029, time.November, 29, 10), personality.SessionWelcome},
		{at(2026, time.October, 21, 3), personality.SessionWelcomeLateNight},
		{at(2026, time.October, 21, 4), personality.SessionWelcomeLateNight},
		{at(2026, time.October, 21, 5), personality.SessionWelcome},
		{at(2026, time.October, 19, 9), personality.SessionWelcomeMonday},
		{at(2026, time.October, 17, 12), personality.SessionWelcomeWeekend},
		{at(2026, time.October, 18, 12), personality.SessionWelcomeWeekend},
		{at(2026, time.October, 21, 14), personality.SessionWelcome},
	} {
		if got := welcomeKey(c.t); got != c.want {
			t.Errorf("%s: %s, want %s", c.t.Format("Mon 2006-01-02 15:04"), got, c.want)
		}
	}
}

// One ambient line a turn, the welcome sharing the first turn's: the slow
// labels, very_slow in slow's place and steady; milestones in their order,
// once each, one held back by a spent budget said at a later turn's end;
// none without a terminal, and none under boring (0022-MADR D12).
func TestAmbientBudget(t *testing.T) {
	dry := dryVoice(t)
	in := func(key personality.Key, args personality.Args, line string) bool {
		return slices.Contains(dry.Variants(key, args), line)
	}
	a := newAmbient(true)
	if line, ok := a.welcome(dry, time.Date(2026, time.October, 21, 14, 0, 0, 0, time.Local)); !ok || !in(personality.SessionWelcome, nil, line) {
		t.Fatalf("welcome %q, %v", line, ok)
	}
	a.startTurn()
	if got := a.label(dry, 40*time.Second); !in(personality.TurnWorking, nil, got) {
		t.Fatalf("the first turn's label %q: the welcome spent its budget", got)
	}
	a.note(acpclient.Update{Kind: acpclient.KindCompaction, Compaction: acpclient.Compaction{Phase: "end"}})
	if line, ok := a.endTurn(dry, time.Second); ok {
		t.Fatalf("the first turn's end said %q", line)
	}
	a.startTurn()
	if got := a.label(dry, 10*time.Second); !in(personality.TurnWorking, nil, got) {
		t.Fatalf("at 10 s: %q", got)
	}
	slow := a.label(dry, 31*time.Second)
	if !in(personality.TurnSlow, nil, slow) || a.label(dry, 40*time.Second) != slow {
		t.Fatalf("at 31 s: %q, then %q", slow, a.label(dry, 40*time.Second))
	}
	verySlow := a.label(dry, 181*time.Second)
	if !in(personality.TurnVerySlow, nil, verySlow) || a.label(dry, 200*time.Second) != verySlow {
		t.Fatalf("at 181 s: %q", verySlow)
	}
	if line, ok := a.endTurn(dry, 11*time.Minute); ok {
		t.Fatalf("a slow turn's end said %q", line)
	}
	a.note(acpclient.Update{Kind: acpclient.KindUsage, Context: acpclient.ContextUsage{Cost: &acpclient.Cost{Amount: 1.25, Currency: "USD"}}})
	for range callMilestone {
		a.note(acpclient.Update{Kind: acpclient.KindToolCall})
	}
	for _, want := range []struct {
		key  personality.Key
		args personality.Args
	}{
		{personality.MilestoneFirstCompaction, nil},
		{personality.MilestoneLongTurn, nil},
		{personality.MilestoneCost, personality.Args{"cost": "$1.25"}},
		{personality.MilestoneHundredthCall, nil},
	} {
		a.startTurn()
		if line, ok := a.endTurn(dry, time.Second); !ok || !in(want.key, want.args, line) {
			t.Fatalf("%s: %q, %v", want.key, line, ok)
		}
	}
	a.startTurn()
	if line, ok := a.endTurn(dry, 11*time.Minute); ok {
		t.Fatalf("a milestone said twice: %q", line)
	}
	var goodbyes []string
	for range dry.Variants(personality.SessionGoodbye, nil) {
		line, ok := a.goodbye(dry)
		if !ok || slices.Contains(goodbyes, line) {
			t.Fatalf("goodbye %q again, after %q", line, goodbyes)
		}
		goodbyes = append(goodbyes, line)
	}

	for _, c := range []struct {
		name  string
		on    bool
		voice *personality.Voice
	}{{"no terminal", false, dry}, {"boring", true, nil}} {
		a := newAmbient(c.on)
		_, welcomed := a.welcome(c.voice, time.Now())
		a.startTurn()
		label := a.label(c.voice, 5*time.Minute)
		a.note(acpclient.Update{Kind: acpclient.KindCompaction, Compaction: acpclient.Compaction{Phase: "end"}})
		_, milestone := a.endTurn(c.voice, time.Second)
		_, bye := a.goodbye(c.voice)
		if welcomed || milestone || bye || !slices.Contains(c.voice.Variants(personality.TurnWorking, nil), label) {
			t.Errorf("%s: welcome %v, label %q, milestone %v, goodbye %v", c.name, welcomed, label, milestone, bye)
		}
	}
	if got := costText(acpclient.Cost{Amount: 2.5, Currency: "EUR"}); got != "2.50 EUR" {
		t.Errorf("cost %q", got)
	}
}

// The spinner's label is the session's ambient label at the time since
// the turn started (0022-MADR D12).
func TestSpinnerLabel(t *testing.T) {
	dry := dryVoice(t)
	now := time.Date(2026, time.October, 21, 14, 0, 0, 0, time.Local)
	start := now
	ls := &lineSession{e: &runEnv{now: func() time.Time { return now }}, voice: dry, amb: newAmbient(true)}
	ls.amb.turns = 1 // a later turn: the welcome's budget is not this one's
	label := ls.spinnerLabel(start)
	if got := label(); !slices.Contains(dry.Variants(personality.TurnWorking, nil), got) {
		t.Fatalf("at the start: %q", got)
	}
	now = now.Add(31 * time.Second)
	if got := label(); !slices.Contains(dry.Variants(personality.TurnSlow, nil), got) {
		t.Fatalf("at 31 s: %q", got)
	}
}

// The line view, on a terminal: the welcome first; the milestones the
// session's updates reach, at later turns' ends, the welcome having spent
// the first turn's; /gobble's turkey and line, hidden from /help and
// completion; the hidden /humor sent to the agent; the goodbye last
// (0022-MADR D12; 0022-PLAN P3, P4).
func TestLineAmbient(t *testing.T) {
	var calls []string
	for i := range callMilestone {
		calls = append(calls, fmt.Sprintf(`{"sessionUpdate":"tool_call","toolCallId":"t%d","title":"read","status":"completed"}`, i))
	}
	priced := `{"sessionUpdate":"usage_update","used":1500,"size":200000,"cost":{"amount":1.25,"currency":"USD"}}`
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{
		{Updates: append(calls, priced, chunk("one done"))},
		{Updates: []string{chunk("two done")}},
		{Updates: []string{chunk("three done")}},
		{Updates: []string{chunk("humored")}},
	}}
	h := newLineHarness(t, agent)
	dry := dryVoice(t)
	h.voice = dry
	h.e.out.caps.Err.TTY = true
	any := func(key personality.Key, args personality.Args) func() bool {
		return func() bool {
			return slices.ContainsFunc(dry.Variants(key, args), func(l string) bool { return strings.Contains(h.stderr.String(), l+"\n") })
		}
	}
	h.start(Prompt{})
	waitUntil(t, any(welcomeKey(time.Now()), nil))
	h.press("one\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "one done") })
	h.press("two\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "two done") })
	waitUntil(t, any(personality.MilestoneCost, personality.Args{"cost": "$1.25"}))
	h.press("three\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "three done") })
	waitUntil(t, any(personality.MilestoneHundredthCall, nil))
	h.press("/gobble\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), turkeyASCII) })
	waitUntil(t, func() bool {
		return slices.ContainsFunc(dry.Variants(personality.EasterGobble, nil), func(l string) bool { return strings.Contains(h.stdout.String(), turkeyASCII+l+"\n") })
	})
	h.press("/help\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Commands\n") })
	h.press("/humor 4\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "humored") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if !any(personality.SessionGoodbye, nil)() {
		t.Fatalf("no goodbye in %q", h.stderr.String())
	}
	// /help's rows, up to the next prompt's, on the screen (0027-PLAN P4,
	// deviation 3).
	lines := screenOf(h.stdout.String(), 80).Lines()
	start := slices.Index(lines, "Commands")
	if start < 0 {
		t.Fatalf("no Commands row on the screen: %q", lines)
	}
	end := start + 1
	for end < len(lines) && !strings.HasPrefix(lines[end], ">") {
		end++
	}
	help := strings.Join(lines[start:end], "\n")
	if strings.Contains(help, "/gobble") || strings.Contains(help, "/humor") {
		t.Fatalf("/help shows a hidden command:\n%s", help)
	}
	if !slices.ContainsFunc(agent.Prompts(), func(p string) bool { return strings.Contains(p, `"text":"/humor 4"`) }) {
		t.Fatalf("the agent was sent %q", agent.Prompts())
	}
	if line, _, _ := (&lineSession{}).complete("/gob", 4); strings.Contains(line, "gobble") {
		t.Fatalf("/gob completes to %q", line)
	}
}

// Without a terminal on stderr, and in print mode, there is no ambient
// line (0022-MADR D12).
func TestNoAmbientOffTerminal(t *testing.T) {
	dry := dryVoice(t)
	lines := func(stderr string) []string {
		var out []string
		for _, k := range personality.AmbientKeys {
			for _, l := range dry.Variants(k, nil) {
				if strings.Contains(stderr, l) {
					out = append(out, l)
				}
			}
		}
		return out
	}
	h := newLineHarness(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("ok")}}}})
	h.voice = dry
	h.start(Prompt{})
	h.press("hi\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "ok") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if got := lines(h.stderr.String()); len(got) > 0 {
		t.Fatalf("the line view without a terminal said %q", got)
	}
}

// Print mode says no ambient line (0022-MADR D12).
func TestNoAmbientInPrint(t *testing.T) {
	dry := dryVoice(t)
	withModel(t, llmtest.NewScript(llmtest.Text("fine")))
	home := t.TempDir()
	t.Chdir(t.TempDir())
	r := mustRun(t, home, "-p", "hi")
	for _, k := range personality.AmbientKeys {
		for _, l := range dry.Variants(k, nil) {
			if strings.Contains(r.stderr, l) || r.stdout != "fine\n" {
				t.Fatalf("print mode said %q; stdout %q", l, r.stdout)
			}
		}
	}
}
