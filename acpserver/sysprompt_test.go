package acpserver

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// promptAgent is an agent over the store in dir, on model p, with prompt
// options o and the read tool, and any changes to its options. Its warnings
// go to logs.
func promptAgent(t *testing.T, dir string, p llm.Provider, o PromptOptions, logs *bytes.Buffer, change ...func(*Options)) *acptest.Conn {
	t.Helper()
	opts := Options{Version: "1.2.3", Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil }, Models: func() ModelChoice { return testModels },
		Store: jsonl.New(dir, jsonl.Options{}), Tools: []tool.Tool{builtin.Read()}, Prompt: o,
		Logger: slog.New(slog.NewTextHandler(logs, nil))}
	for _, f := range change {
		f(&opts)
	}
	ag := New(opts)
	c := acptest.Connect(t, ag, &acptest.Client{Allow: true})
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c
}

// promptEntries are the gobble.prompt and gobble.context entries of
// session id's file, in order, as "baseline", "patch" or "context:<paths>".
func promptEntries(t *testing.T, dir, id string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*", "*_"+id+".jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("session %s files %v, %v", id, files, err)
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n")[1:] {
		var e session.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		switch {
		case e.CustomType == session.CustomPrompt && strings.Contains(string(e.Data), `"baseline"`):
			out = append(out, "baseline")
		case e.CustomType == session.CustomPrompt:
			out = append(out, "patch")
		case e.CustomType == session.CustomContext:
			out = append(out, "context:"+string(e.Data))
		case e.Type == session.TypeCompaction:
			out = append(out, "compaction")
		}
	}
	return out
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// userTextsOf are a request's user messages' texts.
func userTextsOf(req *llm.Request) []string {
	var out []string
	for _, m := range req.Messages {
		if m.Role == llm.RoleUser {
			for _, c := range m.Content {
				out = append(out, c.Text)
			}
		}
	}
	return out
}

// The first prompt writes a baseline, which is the system prompt; an
// AGENTS.md edit makes the next prompt write a patch, whose update message
// joins the history while the system prompt stays the same; a resumed
// session sends that prompt again and writes nothing new (0005-PLAN F3-1).
func TestPromptBaselineAndPatch(t *testing.T) {
	dir, cwd, cfg := t.TempDir(), t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(cfg, "AGENTS.md"), "Config rule.")
	writeFile(t, filepath.Join(cwd, "AGENTS.md"), "First rule.")
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"), llmtest.Text("c"))
	var logs bytes.Buffer
	c := promptAgent(t, dir, model, PromptOptions{ConfigDir: cfg}, &logs)
	id := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, id, "one"); err != nil {
		t.Fatal(err)
	}
	first := systemOf(model.Requests()[0])
	if !strings.Contains(first, "Config rule.") || !strings.Contains(first, "First rule.") || !strings.HasPrefix(first, "You are gobble") {
		t.Fatalf("system prompt %q", first)
	}
	writeFile(t, filepath.Join(cwd, "AGENTS.md"), "Second rule.")
	if _, err := prompt(t, c, id, "two"); err != nil {
		t.Fatal(err)
	}
	req := model.Requests()[1]
	if systemOf(req) != first {
		t.Fatalf("the system prompt changed:\n%s", systemOf(req))
	}
	users := userTextsOf(req)
	if n := len(users); n < 2 || !strings.HasPrefix(users[n-2], `Updated system prompt section "project_context":`) ||
		!strings.Contains(users[n-2], "Second rule.") || users[n-1] != "two" {
		t.Fatalf("user messages %q; want the update before the prompt", users)
	}
	if got := strings.Join(promptEntries(t, dir, string(id)), ","); got != "baseline,patch" {
		t.Fatalf("entries %s", got)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	c2 := promptAgent(t, dir, model, PromptOptions{ConfigDir: cfg}, &logs)
	if _, err := c2.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c2, id, "three"); err != nil {
		t.Fatal(err)
	}
	resumed := model.Requests()[2]
	if systemOf(resumed) != first || !strings.Contains(strings.Join(userTextsOf(resumed), "\n"), `Updated system prompt section "project_context"`) {
		t.Fatalf("resumed: system %q, users %q", systemOf(resumed), userTextsOf(resumed))
	}
	if got := strings.Join(promptEntries(t, dir, string(id)), ","); got != "baseline,patch" {
		t.Fatalf("entries after resume %s; want nothing new", got)
	}
}

// /compact writes a new baseline, holding the patch, just before its
// compaction entry, and the next request's system prompt is that baseline.
func TestCompactRebases(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(cwd, "AGENTS.md"), "Old rule.")
	model := llmtest.NewScript(llmtest.Text(strings.Repeat("x", 4000)), llmtest.Text("y"), llmtest.Text("## Goal\nS"+strings.Repeat(" detail", 80)), llmtest.Text("z"))
	var logs bytes.Buffer
	c := promptAgent(t, dir, model, PromptOptions{}, &logs, func(o *Options) { o.Compaction = compaction.Settings{KeepRecentTokens: 50} })
	id := newSession(t, c, cwd, nil).SessionId
	for _, p := range []string{strings.Repeat("a ", 2000), "second"} {
		if p == "second" {
			writeFile(t, filepath.Join(cwd, "AGENTS.md"), "New rule.")
		}
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ext(t, c, "_gobble/compact", map[string]any{"sessionId": id}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(promptEntries(t, dir, string(id)), ","); got != "baseline,patch,baseline,compaction" {
		t.Fatalf("entries %s", got)
	}
	if _, err := prompt(t, c, id, "after"); err != nil {
		t.Fatal(err)
	}
	reqs := model.Requests()
	if sys := systemOf(reqs[len(reqs)-1]); !strings.Contains(sys, "New rule.") || strings.Contains(sys, "Old rule.") {
		t.Fatalf("system prompt after compaction %q", sys)
	}
}

// A read below cwd brings in the AGENTS.md between them once: a second
// read adds nothing, and nor does a resumed session (0005-MADR, amendment
// of 2026-10-10, item 5).
func TestNestedInstructionsOnce(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(cwd, "pkg", "AGENTS.md"), "Package rule.")
	writeFile(t, filepath.Join(cwd, "pkg", "a.go"), "package pkg\n")
	model := llmtest.NewScript(
		llmtest.ToolCall("r1", "read", `{"path":"pkg/a.go"}`), llmtest.Text("one"),
		llmtest.ToolCall("r2", "read", `{"path":"pkg/a.go"}`), llmtest.Text("two"),
		llmtest.ToolCall("r3", "read", `{"path":"pkg/a.go"}`), llmtest.Text("three"))
	var logs bytes.Buffer
	c := promptAgent(t, dir, model, PromptOptions{}, &logs)
	id := newSession(t, c, cwd, nil).SessionId
	for _, p := range []string{"read it", "again"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	results := func(req *llm.Request) string {
		var out []string
		for _, m := range req.Messages {
			for _, ct := range m.Content {
				if ct.Type == llm.ContentToolResult {
					out = append(out, ct.Text)
				}
			}
		}
		return strings.Join(out, "\n---\n")
	}
	if r := results(model.Requests()[1]); !strings.Contains(r, "<project_instructions path=") || !strings.Contains(r, "Package rule.") {
		t.Fatalf("first read's result %q", r)
	}
	if r := results(model.Requests()[3]); strings.Count(r, "Package rule.") != 1 {
		t.Fatalf("second read added the file again: %q", r)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	c2 := promptAgent(t, dir, model, PromptOptions{}, &logs)
	if _, err := c2.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c2, id, "once more"); err != nil {
		t.Fatal(err)
	}
	if r := results(model.Requests()[5]); strings.Count(r, "Package rule.") != 1 {
		t.Fatalf("a resumed read added the file again: %q", r)
	}
	var contexts int
	for _, e := range promptEntries(t, dir, string(id)) {
		if strings.HasPrefix(e, "context:") {
			contexts++
		}
	}
	if contexts != 1 {
		t.Fatalf("%d gobble.context entries, want 1", contexts)
	}
}

// The options' texts replace SYSTEM.md's and APPEND_SYSTEM.md's; with no
// context files, none is read; a project's own SYSTEM.md waits for trust,
// and says so once.
func TestPromptOptionsAndTrust(t *testing.T) {
	dir, cwd, cfg := t.TempDir(), t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(cfg, "SYSTEM.md"), "From the file.")
	writeFile(t, filepath.Join(cwd, "AGENTS.md"), "Project rule.")
	writeFile(t, filepath.Join(cwd, ".gobble", "SYSTEM.md"), "Untrusted.")
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"))
	var logs bytes.Buffer
	c := promptAgent(t, dir, model, PromptOptions{ConfigDir: cfg, Append: "Appended.", NoContextFiles: true}, &logs)
	id := newSession(t, c, cwd, nil).SessionId
	for _, p := range []string{"one", "two"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	sys := systemOf(model.Requests()[0])
	if !strings.HasPrefix(sys, "From the file.") || !strings.Contains(sys, "<addendum>\nAppended.\n</addendum>") ||
		strings.Contains(sys, "Project rule.") || strings.Contains(sys, "Untrusted.") || strings.Contains(sys, "<rules>") {
		t.Fatalf("system prompt %q", sys)
	}
	if n := strings.Count(logs.String(), "needs a trusted project"); n != 1 {
		t.Fatalf("the trust note was logged %d times:\n%s", n, logs.String())
	}
	flagged := llmtest.NewScript(llmtest.Text("c"))
	c2 := promptAgent(t, t.TempDir(), flagged, PromptOptions{ConfigDir: cfg, System: "From the flag."}, &logs)
	if _, err := prompt(t, c2, newSession(t, c2, cwd, nil).SessionId, "x"); err != nil {
		t.Fatal(err)
	}
	if sys := systemOf(flagged.Requests()[0]); !strings.HasPrefix(sys, "From the flag.") || strings.Contains(sys, "From the file.") {
		t.Fatalf("with System set, the system prompt is %q", sys)
	}
}
