package cli

import (
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// The turn's last line names a stop from the catalog, and one the catalog
// does not know as the agent gave it (0022-PLAN P1).
func TestStopText(t *testing.T) {
	for reason, want := range map[string]string{
		"max_tokens": "max_tokens", "max_turn_requests": "max_turn_requests", "refusal": "refusal", "cancelled": "cancelled",
		"something_new": "something_new",
	} {
		if got := stopText(nil, reason); got != want {
			t.Errorf("%s: %q, want %q", reason, got, want)
		}
		line := statusLine(turnClock{}, nil, acpclient.Result{StopReason: reason}, false, term.Stream{}, term.ASCII, nil)
		if !strings.HasSuffix(line, " "+want) {
			t.Errorf("%s: the line %q does not end with it", reason, line)
		}
	}
	if line := statusLine(turnClock{}, nil, acpclient.Result{StopReason: "end_turn"}, false, term.Stream{}, term.ASCII, nil); strings.Contains(line, "end_turn") {
		t.Fatalf("an ordinary end named: %q", line)
	}
}
