package acpserver

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

func modeUpdates(client *acptest.Client) []string {
	var out []string
	for _, u := range client.Updates() {
		if m := u.Update.CurrentModeUpdate; m != nil {
			out = append(out, string(m.CurrentModeId))
		}
	}
	return out
}

// session/new advertises default and plan; set_mode to plan then default
// confirms each with current_mode_update (0002-PLAN Phase 6 Accept).
func TestSetModeUpdates(t *testing.T) {
	c, client := connectWith(t, llmtest.NewScript(), &acptest.Client{})
	initialize(t, c.Client)
	s, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	if s.Modes == nil || s.Modes.CurrentModeId != modeDefault || len(s.Modes.AvailableModes) != 2 || s.Modes.AvailableModes[1].Id != modePlan {
		t.Fatalf("modes %+v", s.Modes)
	}
	for _, m := range []acp.SessionModeId{modePlan, modeDefault} {
		if _, err := c.Client.SetSessionMode(t.Context(), acp.SetSessionModeRequest{SessionId: s.SessionId, ModeId: m}); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Join(modeUpdates(client), ","); got != "default,plan,default" {
		t.Fatalf("mode updates %s, want the start's default, then plan, then default", got)
	}
	_, err = c.Client.SetSessionMode(t.Context(), acp.SetSessionModeRequest{SessionId: s.SessionId, ModeId: "bypass"})
	if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32602 {
		t.Fatalf("unknown mode: %v", err)
	}
}

// In plan mode a mutating call is refused without asking, and nothing is
// written.
func TestPlanModeRefusesWrite(t *testing.T) {
	dir := t.TempDir()
	model := llmtest.NewScript(llmtest.ToolCall("w1", "write", toolArgs(t, map[string]string{"path": "a.txt", "content": "x"})), llmtest.Text("ok"))
	c, client := connectWith(t, model, &acptest.Client{Allow: true})
	id := openSession(t, c, dir)
	if text, _ := slash(t, c, client, id, "/plan"); !strings.HasPrefix(text, "Mode: plan.") {
		t.Fatalf("/plan: %q", text)
	}
	if _, err := prompt(t, c, id, "write it"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err == nil {
		t.Fatal("plan mode let write run")
	}
	if len(client.Permissions()) != 0 {
		t.Fatal("plan mode asked instead of refusing")
	}
	sent := model.Requests()[1].Messages
	if r := sent[len(sent)-1].Content[0]; !r.IsError || !strings.HasPrefix(r.Text, "Plan mode is read-only, so write did not run.") {
		t.Fatalf("result %+v", r)
	}
	if !hasAll(requestTools(t, model.Requests()[0]), exitPlanTool) {
		t.Fatal("plan mode does not offer exit_plan_mode")
	}
}

// exit_plan_mode publishes the plan, asks with the plan as the request's
// first content, and on allow returns to the default mode, where a write
// asks as usual.
func TestExitPlanMode(t *testing.T) {
	dir := t.TempDir()
	plan := "1. Add the flag\n- Test it\n\n* [ ] Ship"
	model := llmtest.NewScript(
		llmtest.ToolCall("x1", exitPlanTool, toolArgs(t, map[string]string{"plan": plan})),
		llmtest.Text("leaving"),
		llmtest.ToolCall("w1", "write", toolArgs(t, map[string]string{"path": "a.txt", "content": "x"})),
		llmtest.Text("done"),
	)
	c, client := connectWith(t, model, &acptest.Client{Allow: true})
	id := openSession(t, c, dir)
	if _, err := c.Client.SetSessionMode(t.Context(), acp.SetSessionModeRequest{SessionId: id, ModeId: modePlan}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, id, "plan it"); err != nil {
		t.Fatal(err)
	}
	var entries []string
	for _, u := range client.Updates() {
		if p := u.Update.Plan; p != nil {
			for _, e := range p.Entries {
				entries = append(entries, e.Content)
			}
		}
	}
	if strings.Join(entries, "|") != "Add the flag|Test it|Ship" {
		t.Fatalf("plan entries %q", entries)
	}
	perms := client.Permissions()
	if len(perms) != 1 || perms[0].ToolCall.ToolCallId != "x1" || perms[0].ToolCall.Content[0].Content.Content.Text.Text != plan {
		t.Fatalf("permission %+v", perms)
	}
	if got := modeUpdates(client); got[len(got)-1] != modeDefault {
		t.Fatalf("mode updates %v", got)
	}
	if _, err := prompt(t, c, id, "do it"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatalf("after the plan was approved, write did not run: %v", err)
	}
	if len(client.Permissions()) != 2 {
		t.Fatal("the write after plan mode did not ask")
	}
}
