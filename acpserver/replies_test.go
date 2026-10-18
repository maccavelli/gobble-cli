package acpserver

import (
	"regexp"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// unfilled is a placeholder a reply left in braces.
var unfilled = regexp.MustCompile(`\{[a-z_]+\}`)

// Every slash command's replies come from the boring catalog with their
// placeholders filled, and the replies no other test reads are today's
// text (0022-PLAN P1).
func TestRepliesFromTheCatalog(t *testing.T) {
	estimated := llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "hi"}, llm.Usage{InputTokens: 3, OutputTokens: 1, Estimated: true},
		llm.Done{StopReason: llm.StopEndTurn}}}
	client := &acptest.Client{}
	c, _ := sessionAgent(t, t.TempDir(), llmtest.NewScript(estimated, llmtest.Text("SUMMARY")), client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	reply := func(text string) string {
		t.Helper()
		before := len(client.Updates())
		if _, err := prompt(t, c, id, text); err != nil {
			t.Fatalf("%s: %v", text, err)
		}
		got := agentText(client.Updates()[before:])
		if m := unfilled.FindString(got); m != "" {
			t.Errorf("%s: %q left %s unfilled", text, got, m)
		}
		return got
	}
	wants := []struct{ prompt, want string }{
		{"/fork", "No messages to fork from yet."},
		{"/clone", "Nothing to clone yet"},
		{"/todos", "no todo list"},
		{"/mode plan", "Mode: plan. Files cannot change and commands cannot run until the user approves a plan."},
		{"/mode plan", "Already in plan mode."},
		{"/mode default", "Mode: default"},
		{"/name first\nsecond", "Session name was normalized from \"first\\nsecond\" to \"first second\".\nSession name set: first second"},
	}
	for _, w := range wants {
		if got := reply(w.prompt); got != w.want {
			t.Errorf("%q: %q, want %q", w.prompt, got, w.want)
		}
	}
	reply("hello") // a turn whose usage is estimated
	if got := reply("/clone"); !strings.HasPrefix(got, "Cloned to session ") || !strings.HasSuffix(got, ".") {
		t.Errorf("/clone: %q", got)
	}
	if got := reply("/usage"); !strings.Contains(got, "\nThese counts are estimated: the provider did not report usage.\n") ||
		!strings.Contains(got, "Account and rate-limit figures are not reported by fake.\n") || !strings.HasPrefix(got, "Session usage\n\n") {
		t.Errorf("/usage: %q", got)
	}
	for _, p := range []string{"/help", "/session", "/model", "/model m2", "/model nope", "/thinking", "/thinking nope", "/thinking off",
		"/mode", "/mode nope", "/name", "/fork nope", "/compact", "/nope"} {
		if got := reply(p); got == "" {
			t.Errorf("%s: no reply", p)
		}
	}
	if got := reply("/compact"); !strings.HasPrefix(got, "Compacting the conversation…\n\nCompaction failed: ") {
		t.Errorf("a second /compact: %q", got)
	}
}
