package compaction

import (
	"testing"

	"github.com/maccavelli/gobble-cli/session"
)

// A compaction summary reads a failed entry as quoted history, as it reads
// an aborted one: its thinking and text are serialised (0070-MADR D9; G27).
func TestSerializeKeepsFailedEntry(t *testing.T) {
	u := &session.Message{Role: session.RoleUser}
	u.SetText("go")
	failed := &session.Message{Role: session.RoleAssistant, StopReason: "error", ErrorMessage: "cut"}
	if err := failed.SetBlocks([]session.Block{{Type: session.BlockThinking, Thinking: "so"}, {Type: session.BlockText, Text: "half"}}); err != nil {
		t.Fatal(err)
	}
	const want = "[User]: go\n\n[Assistant thinking]: so\n\n[Assistant]: half"
	if got := serialize([]*session.Message{u, failed}); got != want {
		t.Fatalf("serialised %q, want %q", got, want)
	}
}
