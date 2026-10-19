package acpserver

import (
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
)

// A compaction in plan mode ends with the plan-mode reminder, and the
// user's compaction prompts reach the summary request (0005-PLAN F5-4
// decisions 5 and 8).
func TestCompactionModeAndPrompts(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text(big(100)), llmtest.Text(big(100)), llmtest.Text("S"), llmtest.Text("after"))
	client := &acptest.Client{}
	a := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return model, nil },
		Store: session.NewMemoryStore(), Compaction: compaction.Settings{ReserveTokens: 16384, KeepRecentTokens: 150},
		CompactionPrompts: compaction.Prompts{System: "MY SYSTEM", Summary: "MY SUMMARY"}})
	c := acptest.Connect(t, a, client)
	a.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	s, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"one " + big(100), "two " + big(100), "/mode plan", "/compact", "three"} {
		if _, err := prompt(t, c, s.SessionId, p); err != nil {
			t.Fatal(err)
		}
	}
	reqs := model.Requests()
	llmtest.AssertSystem(t, reqs[2], "MY SYSTEM")
	if p := reqs[2].Messages[1].Content[0].Text; !strings.HasSuffix(p, "\n</conversation>\n\nMY SUMMARY") {
		t.Fatalf("summary prompt %q", p)
	}
	if got := requestTexts(reqs[3]); len(got) == 0 || got[0] != compaction.SummaryText("S\n\n<mode>plan</mode>") {
		t.Fatalf("context after compact: %.120q", got)
	}
}
