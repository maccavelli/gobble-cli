package acpserver

import (
	"bytes"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/personality"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// A session whose settings give dry sends its brief and the house rules in
// the system prompt, and answers its slash commands in its voice; a
// session with none is boring, as before; a name no personality has is
// boring too, and logged (0022-PLAN P2).
func TestSessionPersonality(t *testing.T) {
	dry, err := personality.New(personality.Options{Name: personality.Dry})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, setting string
		dry           bool
	}{
		{"dry", personality.Dry, true},
		{"unset", "", false},
		{"unknown", "zany", false},
	} {
		model := llmtest.NewScript(llmtest.Text("hi"))
		var logs bytes.Buffer
		client := &acptest.Client{}
		ag := New(Options{Version: "1.2.3", Store: jsonl.New(t.TempDir(), jsonl.Options{}), Models: func() ModelChoice { return testModels },
			Provider: func(string, llm.Retry) (llm.Provider, error) { return model, nil },
			Settings: func(string) SessionSettings { return SessionSettings{Personality: c.setting} },
			Logger:   slog.New(slog.NewTextHandler(&logs, nil))})
		conn := acptest.Connect(t, ag, client)
		ag.SetConnection(conn.Agent)
		t.Cleanup(func() {
			if err := ag.Close(); err != nil {
				t.Error(err)
			}
		})
		initialize(t, conn.Client)
		id := newSession(t, conn, t.TempDir(), nil).SessionId
		if _, err := prompt(t, conn, id, "hello"); err != nil {
			t.Fatal(err)
		}
		sys := systemOf(model.Requests()[0])
		hasBrief := strings.Contains(sys, "<personality>\n"+dry.Brief()+"\n</personality>") &&
			strings.Contains(sys, "<personality_rules>\n"+dry.Rules()+"\n</personality_rules>")
		if hasBrief != c.dry || (!c.dry && strings.Contains(sys, "<personality")) {
			t.Errorf("%s: the system prompt has the personality %v, want %v", c.name, hasBrief, c.dry)
		}
		before := len(client.Updates())
		if _, err := prompt(t, conn, id, "/mode plan"); err != nil {
			t.Fatal(err)
		}
		got := agentText(client.Updates()[before:])
		const boring = "Mode: plan. Files cannot change and commands cannot run until the user approves a plan."
		if c.dry && (!slices.Contains(dry.Variants(personality.ModePlanOn, nil), got) || got == boring) {
			t.Errorf("%s: /mode plan answered %q, want one of dry's", c.name, got)
		}
		if !c.dry && got != boring {
			t.Errorf("%s: /mode plan answered %q, want boring's", c.name, got)
		}
		if c.setting == "zany" && !strings.Contains(logs.String(), `no personality named \"zany\"`) {
			t.Errorf("unknown: logs %q, want the unknown name noted", logs.String())
		}
	}
}
