package personality

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// typical are the values a status line is measured with: the widths it
// shows in an ordinary session.
var typical = Args{
	"attempt": "1", "max": "2", "cause": "rate limited", "reason": "threshold", "error": "Summarization failed: llm: offline",
	"n": "42", "cost": "$1.00",
}

// statusKeys are the lines gobble's CLI draws on one terminal line
// (0022-MADR D17): its status lines, the spinner, the turn's stop, the cut
// output's note.
var statusKeys = []Key{
	RetryOffline, RetryEmptyReply, RetryAttempt, CompactionStart, CompactionFailed, CompactionFailedTurn,
	CompactionFailedSession, TurnWorking, TurnStopMaxTokens, TurnStopMaxRequests, TurnStopRefusal, TurnStopCancelled,
	LinesHidden,
	SessionWelcome, SessionWelcomeLateNight, SessionWelcomeMonday, SessionWelcomeWeekend, SessionWelcomeThanksgiving,
	SessionGoodbye, TurnSlow, TurnVerySlow, MilestoneFirstCompaction, MilestoneLongTurn, MilestoneCost,
	MilestoneHundredthCall, EasterGobble,
}

// spinnerKeys are the lines the spinner draws after its frame.
var spinnerKeys = []Key{TurnWorking, TurnSlow, TurnVerySlow}

// leftBoring are the keys a built-in personality leaves to boring on
// purpose, by personality (0022-MADR D17): trust's, whose facts a person
// acts on to let a project's files load (0005-PLAN F6-1).
var leftBoring = map[string][]Key{Dry: factKeys, Snarky: factKeys, Bombastic: factKeys}

// factKeys are /trust's and /permissions' replies, whose facts a person
// acts on (0005-PLAN F6-1, F6-2b decision H).
var factKeys = []Key{TrustUntrusted, TrustCurrent, TrustSaved, TrustNotSaved, TrustFailed, TrustUnavailable,
	PermissionsDefaults, PermissionsRule, PermissionsNone, PermissionsLimits, PermissionsRemoved, PermissionsFixed,
	PermissionsNoRule, PermissionsAdded, PermissionsBad, PermissionsUsage, PermissionsFailed}

// The words no personality but bombastic aims at the user, and the
// profanity each allows (0022-MADR D7 to D9, D17). The lists are a cheap
// guard, not a judgement.
var (
	insults   = []string{"idiot", "stupid", "moron", "dumb", "fool", "foolish", "clueless", "incompetent", "pathetic", "loser", "dimwit", "imbecile"}
	mild      = []string{"damn", "hell", "crap"}
	strong    = []string{"shit", "fuck", "bastard", "bitch", "arse", "ass", "piss", "bollocks", "wanker"}
	borrowed  = []string{"Jeeves", "Wooster", "Wodehouse", "Sheldon", "Bazinga", "Brannigan", "Skippy", "Gilfoyle", "GLaDOS", "Marvin", "K-2SO", "TARS", "Reticulating splines", "Don't Panic"}
	wordsOnly = func(words []string) *regexp.Regexp {
		return regexp.MustCompile(`(?i)\b(` + strings.Join(words, "|") + `)\b`)
	}
)

// texts are every variant, lead and brief line of a built-in personality.
func texts(t *testing.T, name string) []string {
	t.Helper()
	d := builtins()[name]
	var out []string
	for _, k := range d.catalog.Keys() {
		out = append(out, d.catalog.Variants(k)...)
	}
	return append(out, strings.Split(d.brief.Body, "\n")...)
}

// Every built-in loads, with its level; boring has no brief.
func TestBuiltinsLoad(t *testing.T) {
	levels := map[string]int{Boring: 0, Dry: 3, Snarky: 4, Bombastic: 5}
	for _, name := range Builtins {
		v, err := New(Options{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		if got := builtins()[name].brief.Level; got != levels[name] {
			t.Errorf("%s: level %d, want %d", name, got, levels[name])
		}
		if (v.Brief() == "") != (name == Boring) || (v.Rules() == "") != (name == Boring) {
			t.Errorf("%s: brief %d bytes, rules %d bytes", name, len(v.Brief()), len(v.Rules()))
		}
	}
	if Canonical("plain") != Boring || Canonical("normal") != Boring || Canonical("dry") != Dry || Canonical("zany") != "" {
		t.Fatal("Canonical")
	}
}

// Every variant of a built-in personality keeps boring's placeholders;
// its keys are boring's, leads or ambient, a lead has none, and an ambient
// line keeps its own; it covers every key but those left to boring on
// purpose, and every ambient key with at least three lines, which boring
// has none of (0022-MADR D5, D12, D17).
func TestBuiltinsFactsRule(t *testing.T) {
	for _, name := range Builtins[1:] {
		c := builtins()[name].catalog
		for _, k := range c.Keys() {
			switch {
			case slices.Contains(LeadKeys, k):
				for _, v := range c.Variants(k) {
					if ps := Placeholders(v); len(ps) > 0 {
						t.Errorf("%s %s: a lead %q has placeholders %v", name, k, v, ps)
					}
				}
			case slices.Contains(Keys, k):
				for _, v := range c.Variants(k) {
					if err := CheckPlaceholders(boringCat().Variants(k)[0], v); err != nil {
						t.Errorf("%s %s: %v", name, k, err)
					}
				}
			case slices.Contains(AmbientKeys, k):
				for _, v := range c.Variants(k) {
					if err := CheckPlaceholders(ambientFacts[k], v); err != nil {
						t.Errorf("%s %s: %v", name, k, err)
					}
				}
			default:
				t.Errorf("%s: unknown key %s", name, k)
			}
		}
		for _, k := range AmbientKeys {
			if n := len(c.Variants(k)); n < 3 {
				t.Errorf("%s: %d %s lines, want at least 3", name, n, k)
			}
		}
		for _, k := range Keys {
			if len(c.Variants(k)) == 0 && !slices.Contains(leftBoring[name], k) {
				t.Errorf("%s: no %s", name, k)
			}
		}
		if n := len(c.Variants(TurnWorking)); n < 40 {
			t.Errorf("%s: %d spinner lines, want at least 40", name, n)
		}
	}
	for _, k := range AmbientKeys {
		if vs := boringCat().Variants(k); len(vs) > 0 {
			t.Errorf("boring has %s: %q", k, vs)
		}
	}
}

// A status line fits 80 columns at typical widths, or boring's own width
// where boring's is longer; a spinner line leaves room for its frame
// (0022-MADR D17).
func TestBuiltinsWidths(t *testing.T) {
	for _, name := range Builtins[1:] {
		c := builtins()[name].catalog
		for _, k := range statusKeys {
			limit := 80
			if b := boringCat().Variants(k); len(b) > 0 {
				limit = max(limit, utf8.RuneCountInString(fill(b[0], typical)))
			}
			if slices.Contains(spinnerKeys, k) {
				limit = 60
			}
			for _, v := range c.Variants(k) {
				if n := utf8.RuneCountInString(fill(v, typical)); n > limit {
					t.Errorf("%s %s: %d columns, want at most %d: %q", name, k, n, limit, v)
				}
			}
		}
	}
}

// Dry and Snarky aim no insult at the user, and keep to their profanity;
// no built-in borrows a character's name or catchphrase (0022-MADR D7 to
// D9, D19).
func TestBuiltinsGuards(t *testing.T) {
	allowed := map[string][]string{Boring: nil, Dry: nil, Snarky: mild, Bombastic: append(slices.Clone(mild), strong...)}
	for _, name := range Builtins {
		for _, line := range texts(t, name) {
			if name != Bombastic {
				if m := wordsOnly(insults).FindString(line); m != "" {
					t.Errorf("%s: %q insults (%s)", name, line, m)
				}
			}
			for _, w := range append(slices.Clone(mild), strong...) {
				if !slices.Contains(allowed[name], w) && wordsOnly([]string{w}).MatchString(line) {
					t.Errorf("%s: %q swears (%s)", name, line, w)
				}
			}
			for _, b := range borrowed {
				if strings.Contains(line, b) {
					t.Errorf("%s: %q borrows %q", name, line, b)
				}
			}
		}
	}
}
