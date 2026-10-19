package acpserver

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/permission"
)

// readCall is a reply that reads f.txt, the call ids alone differing.
func readCall(id string) llmtest.Turn { return llmtest.ToolCall(id, "read", `{"path":"f.txt"}`) }

// The third identical call asks the client over session/request_permission,
// its title the call's with what it asks about; in plan mode it is asked,
// not refused, since it approves no write (0005-PLAN F5-3 decision 1).
func TestRepeatedCallQuestion(t *testing.T) {
	for _, mode := range []string{modeDefault, modePlan} {
		client := &acptest.Client{Allow: true}
		s := llmtest.NewScript(readCall("a"), readCall("b"), readCall("c"), llmtest.Text("done"))
		c, _ := sessionAgent(t, t.TempDir(), s, client)
		cwd := t.TempDir()
		writeFile(t, filepath.Join(cwd, "f.txt"), "hello")
		id := newSession(t, c, cwd, nil).SessionId
		if _, err := c.Client.SetSessionMode(t.Context(), acp.SetSessionModeRequest{SessionId: id, ModeId: acp.SessionModeId(mode)}); err != nil {
			t.Fatal(err)
		}
		if resp, err := prompt(t, c, id, "go"); err != nil || resp.StopReason != acp.StopReasonEndTurn {
			t.Fatalf("%s: %+v, %v", mode, resp, err)
		}
		perms := client.Permissions()
		if len(perms) != 1 || perms[0].ToolCall.ToolCallId != "c" || perms[0].ToolCall.Title == nil ||
			*perms[0].ToolCall.Title != "read f.txt (the same call 3 times in a row)" {
			t.Fatalf("%s: permission requests %+v, want one about c, titled for the repeated call", mode, perms)
		}
		// The question is marked, so --yes leaves it (0005-PLAN F6-2a's
		// deviation 2).
		if n, _ := perms[0].Meta[gobble].(map[string]any)[keyRepeated].(float64); n != 3 {
			t.Fatalf("%s: _meta %v, want gobble.repeated 3", mode, perms[0].Meta)
		}
		// It offers the two "once" options alone (0005-PLAN F6-2b
		// decision D).
		if o := perms[0].Options; len(o) != 2 || o[0].OptionId != allowOnce || o[1].OptionId != rejectOnce {
			t.Fatalf("%s: options %+v", mode, o)
		}
	}
}

// A long call's title is cut to leave the question room, within the title
// bound (0005-PLAN F5-3 decision 1).
func TestRepeatedCallTitle(t *testing.T) {
	got := ruleTitle(permission.Rule{Title: "read " + strings.Repeat("d/", 40) + "f.txt", Repeated: 3})
	if !strings.HasSuffix(got, "… (the same call 3 times in a row)") || utf8.RuneCountInString(got) != maxTitle {
		t.Fatalf("title %q (%d runes), want the call's cut and the question, %d runes", got, utf8.RuneCountInString(got), maxTitle)
	}
	if got := ruleTitle(permission.Rule{Title: "read f.txt"}); got != "read f.txt" {
		t.Fatalf("an ordinary title %q", got)
	}
}

// A reply stopped as a refusal ends the prompt as ACP's refusal
// (0005-PLAN F5-3 decision 5).
func TestRefusalStopReason(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "I can't help with that."}, llm.Done{StopReason: llm.StopRefusal}}})
	c, _ := sessionAgent(t, t.TempDir(), s, &acptest.Client{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if resp, err := prompt(t, c, id, "go"); err != nil || resp.StopReason != acp.StopReasonRefusal {
		t.Fatalf("%+v, %v; want refusal", resp, err)
	}
}
