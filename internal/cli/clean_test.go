package cli

import (
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// Model text bound for a terminal is cleaned by go-tui-lib's safetext, its
// Text rules: a zero-width space and an OSC 52 clipboard write are gone,
// and a bidi override is kept, as term.Sanitize keeps it. Off a terminal it
// keeps term.Sanitize's bytes, the zero-width space included, as a script
// read them before (0035-PLAN PQ5, deviation 10).
func TestTerminalCleaning(t *testing.T) {
	text := "a\u200bb\u202ec\x1b]52;c;ZGF0YQ==\x07d end"
	for _, tc := range []struct {
		tty  bool
		gone []string
		kept []string
	}{
		{true, []string{"\u200b", "\x1b]52", "ZGF0YQ=="}, []string{"\u202e", "end"}},
		{false, []string{"\x1b]52", "ZGF0YQ=="}, []string{"\u200b", "\u202e", "end"}},
	} {
		var out, errw strings.Builder
		s := term.Stream{TTY: tc.tty, Width: 80}
		v := newView(&Output{out: &out, err: &errw, caps: term.Caps{Out: s, Err: s}}, display{}, stepClock())
		v.on(acpclient.Update{Kind: acpclient.KindAgentText, Text: text})
		if err := v.finish(acpclient.Result{StopReason: "end_turn"}); err != nil {
			t.Fatal(err)
		}
		for _, g := range tc.gone {
			if strings.Contains(out.String(), g) {
				t.Errorf("terminal %v: %q shown in %q", tc.tty, g, out.String())
			}
		}
		for _, k := range tc.kept {
			if !strings.Contains(out.String(), k) {
				t.Errorf("terminal %v: %q missing from %q", tc.tty, k, out.String())
			}
		}
	}
}
