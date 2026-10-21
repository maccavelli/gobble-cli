package cli

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/personality"
)

// lineBefore is the line of s just before the first line that starts with
// prefix, or "" when there is none.
func lineBefore(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, prefix) && i > 0 {
			return lines[i-1]
		}
	}
	return ""
}

// In a dry session, a failed turn's error, and an authentication error,
// are led by a line in the session's voice; the error itself is unchanged.
// A boring session's errors stand alone (0022-MADR D13).
func TestLineSessionLeads(t *testing.T) {
	dry, err := personality.New(personality.Options{Name: personality.Dry, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name  string
		turn  acptest.Turn
		error string
		lead  personality.Key
	}{
		{"a failed turn", acptest.Turn{ErrCode: -32603, ErrMessage: "provider down"}, "Error: session/prompt:", personality.LeadTurnFailed},
		{"authentication", acptest.Turn{ErrCode: -32000, ErrMessage: "Authentication required"}, "Error: no model credential", personality.LeadAuth},
	} {
		for _, voice := range []*personality.Voice{dry, nil} {
			h := newLineHarness(t, &acptest.ScriptAgent{Turns: []acptest.Turn{c.turn}})
			h.voice = voice
			h.start(Prompt{})
			h.press("hi\r")
			waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), c.error) })
			h.press("\x04")
			if err := h.wait(); err != nil {
				t.Fatal(err)
			}
			got := lineBefore(h.stderr.String(), c.error)
			leads := dry.Variants(c.lead, nil)
			// The harness has no credential, so the session opens with the
			// warning, led in dry, once the sign-in offered first is declined;
			// the harness's terminal echoes no answer, so the lead follows the
			// question on its line (0025-MADR D7).
			w := strings.TrimPrefix(lineBefore(h.stderr.String(), "Warning: no model credential"), firstRunQuestion+" [Y/n]: ")
			if voice != nil &&
				!slices.Contains(dry.Variants(personality.LeadNoCredential, nil), w) {
				t.Errorf("%s, dry: the line before the credential warning is %q", c.name, w)
			}
			if voice != nil && !slices.Contains(leads, got) {
				t.Errorf("%s, dry: the line before the error is %q, want one of %q", c.name, got, leads)
			}
			if voice == nil && slices.Contains(leads, got) {
				t.Errorf("%s, boring: a lead %q", c.name, got)
			}
		}
	}
}

// Lead writes the voice's line, unprefixed, and nothing for boring.
func TestOutputLead(t *testing.T) {
	dry, err := personality.New(personality.Options{Name: personality.Dry, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	var errw bytes.Buffer
	o := &Output{out: &bytes.Buffer{}, err: &errw}
	o.Lead(nil, personality.LeadImages)
	if errw.Len() != 0 {
		t.Fatalf("boring led with %q", errw.String())
	}
	o.Lead(dry, personality.LeadImages)
	if got := strings.TrimSuffix(errw.String(), "\n"); !slices.Contains(dry.Variants(personality.LeadImages, nil), got) {
		t.Fatalf("dry led with %q", got)
	}
}

// gobble's CLI speaks the voice its sessions start with, in its line view
// and in print mode: dry by default (0022-PLAN P2, deviation 1), or the
// setting's (P3).
func TestCLIVoice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	for _, c := range []struct{ setting, want string }{{"", personality.Dry}, {"personality: snarky\n", personality.Snarky}} {
		writeFile(t, filepath.Join(home, "settings.yaml"), c.setting)
		out := &Output{}
		e := &runEnv{out: out, logs: &lazyLog{out: out}}
		if got := newLineSession(e, &ChatCmd{}, t.TempDir(), "").spoken().Name(); got != c.want {
			t.Fatalf("%q: the line view speaks %s, want %s", c.setting, got, c.want)
		}
		if got := newPrintSink(e, &ChatCmd{}, t.TempDir()).voice.Name(); got != c.want {
			t.Fatalf("%q: print mode speaks %s, want %s", c.setting, got, c.want)
		}
	}
}
