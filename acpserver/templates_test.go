package acpserver

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// templateAgent is promptAgent with the client returned, for its updates:
// an agent over the store in dir, with the config directory cfg, whose
// warnings go to logs.
func templateAgent(t *testing.T, dir, cfg string, p llm.Provider, logs *bytes.Buffer) (*acptest.Conn, *acptest.Client) {
	t.Helper()
	ag := New(Options{Version: "1.2.3", Provider: func() (llm.Provider, error) { return p, nil }, Models: func() ModelChoice { return testModels },
		Store: jsonl.New(dir, jsonl.Options{}), Tools: []tool.Tool{builtin.Read()}, Prompt: PromptOptions{ConfigDir: cfg},
		Logger: slog.New(slog.NewTextHandler(logs, nil))})
	client := &acptest.Client{Allow: true}
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c, client
}

// templateConfig is a config directory with Pi's documented review
// template, a fix template, a template whose name is not a command name,
// a help template a built-in shadows, and a review skill.
func templateConfig(t *testing.T) string {
	t.Helper()
	cfg := t.TempDir()
	writeFile(t, filepath.Join(cfg, "prompts", "review.md"), "---\ndescription: Review staged git changes\nargument-hint: \"[focus]\"\n---\n"+
		"Review the staged changes. Focus on ${1:-correctness, security, and error handling}.\n")
	writeFile(t, filepath.Join(cfg, "prompts", "fix.md"), "Fix $1 then ${@:2}\n")
	writeFile(t, filepath.Join(cfg, "prompts", "a.b.md"), "Dotted $1\n")
	writeFile(t, filepath.Join(cfg, "prompts", "help.md"), "Not the help\n")
	writeFile(t, filepath.Join(cfg, "skills", "review", "SKILL.md"), "---\nname: review\ndescription: Review a change.\n---\nCheck the diff.\n")
	return cfg
}

// A template runs as a model prompt with its arguments, Pi's expansion the
// user's message and the session's entry (a body without frontmatter
// untrimmed, as Pi reads it), a template whose name is no
// command name included; text before the "/" is dropped as Parse drops
// it; a command neither built in nor a template is unknown (0005-PLAN
// F3-3).
func TestTemplateRuns(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"), llmtest.Text("c"), llmtest.Text("d"))
	var logs bytes.Buffer
	c, client := templateAgent(t, dir, templateConfig(t), model, &logs)
	id := newSession(t, c, cwd, nil).SessionId
	want := []string{
		"Review the staged changes. Focus on concurrency.",
		"Review the staged changes. Focus on API compatibility.",
		"Fix a then b c d\n",
		"Dotted x\n",
	}
	for i, text := range []string{"/review concurrency", ` /review "API compatibility"`, `/fix a "b c" d`, "/a.b x"} {
		if _, err := prompt(t, c, id, text); err != nil {
			t.Fatal(err)
		}
		if n := len(model.Requests()); n != i+1 {
			t.Fatalf("%s: %d model requests", text, n)
		}
		if got := lastUser(model.Requests()[i]); got != want[i] {
			t.Errorf("%s: the user's message %q, want %q", text, got, want[i])
		}
	}
	n := len(client.Updates())
	if _, err := prompt(t, c, id, "/nothing here"); err != nil {
		t.Fatal(err)
	}
	if got := textSince(client, n); !strings.Contains(got, "Unknown command /nothing") || len(model.Requests()) != 4 {
		t.Fatalf("/nothing: %q, %d requests", got, len(model.Requests()))
	}
	var users []string
	for _, m := range fileMessages(t, dir, id) {
		if text, ok := strings.CutPrefix(m, "user: "); ok {
			users = append(users, text)
		}
	}
	if !slices.Equal(users, want) {
		t.Fatalf("the session's user entries %q", users)
	}
}

// Each template whose name is a command name is advertised after the
// built-ins, with its description and its argument-hint as the input
// hint; one a built-in shadows is not, and is logged once in Pi's
// collision wording (F3-3 deviation 1), the built-in running.
func TestTemplateAdvertised(t *testing.T) {
	model := llmtest.NewScript()
	var logs bytes.Buffer
	c, client := templateAgent(t, t.TempDir(), templateConfig(t), model, &logs)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	var got []acp.AvailableCommand
	for _, u := range client.Updates() {
		if a := u.Update.AvailableCommandsUpdate; a != nil {
			got = a.AvailableCommands
		}
	}
	if len(got) != len(commands)+2 {
		t.Fatalf("advertised %v", got)
	}
	fix, review := got[len(commands)], got[len(commands)+1]
	if fix.Name != "fix" || fix.Description != "Fix $1 then ${@:2}" || fix.Input != nil {
		t.Errorf("fix: %+v", fix)
	}
	if review.Name != "review" || review.Description != "Review staged git changes" ||
		review.Input == nil || review.Input.Unstructured == nil || review.Input.Unstructured.Hint != "[focus]" {
		t.Errorf("review: %+v", review)
	}
	text, _ := slash(t, c, client, id, "/help")
	if !strings.HasPrefix(text, "Commands:") || len(model.Requests()) != 0 {
		t.Fatalf("/help: %q, %d requests", text, len(model.Requests()))
	}
	note := `reason="name \"/help\" collision (the built-in command /help is loaded)"`
	if n := strings.Count(logs.String(), note); n != 1 {
		t.Fatalf("%q logged %d times:\n%s", note, n, logs.String())
	}
}

// The set is the one found when the session started: a template added
// later is an unknown command until the next session (0005-PLAN F3-3).
func TestTemplatesFixedAtStart(t *testing.T) {
	cfg := templateConfig(t)
	model := llmtest.NewScript(llmtest.Text("a"))
	var logs bytes.Buffer
	c, client := templateAgent(t, t.TempDir(), cfg, model, &logs)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	writeFile(t, filepath.Join(cfg, "prompts", "late.md"), "Late $1\n")
	if text, _ := slash(t, c, client, id, "/late x"); !strings.Contains(text, "Unknown command /late") || len(model.Requests()) != 0 {
		t.Fatalf("/late in the first session: %q", text)
	}
	id = newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "/late x"); err != nil {
		t.Fatal(err)
	}
	if reqs := model.Requests(); len(reqs) != 1 || lastUser(reqs[0]) != "Late x\n" {
		t.Fatalf("/late in the next session: %d requests", len(reqs))
	}
}

// The project's .gobble/prompts is not loaded before F6: it is noted once,
// and its template is an unknown command.
func TestProjectTemplatesWait(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, filepath.Join(cwd, ".gobble", "prompts", "local.md"), "Local $1\n")
	model := llmtest.NewScript(llmtest.Text("a")) // answers a project template, should one run
	var logs bytes.Buffer
	c, client := templateAgent(t, t.TempDir(), templateConfig(t), model, &logs)
	id := newSession(t, c, cwd, nil).SessionId
	if text, _ := slash(t, c, client, id, "/local x"); !strings.Contains(text, "Unknown command /local") || len(model.Requests()) != 0 {
		t.Fatalf("/local: %q", text)
	}
	dir := filepath.Join(cwd, ".gobble", "prompts")
	if n := strings.Count(logs.String(), "file="+dir+" "); n != 1 {
		t.Fatalf("%s noted %d times:\n%s", dir, n, logs.String())
	}
}

// A queued message is expanded as it is queued, a skill and then a
// template, as Pi queues it (F3-3 deviation 2); other text is queued as
// typed.
func TestQueuedExpansion(t *testing.T) {
	cfg := templateConfig(t)
	var logs bytes.Buffer
	c, _ := templateAgent(t, t.TempDir(), cfg, llmtest.NewScript(), &logs)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "/review concurrency"})
	mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "/skill:review now"})
	mustExt(t, c, "_gobble/follow_up", map[string]any{"sessionId": id, "message": "/fix a b"})
	mustExt(t, c, "_gobble/follow_up", map[string]any{"sessionId": id, "message": "/nothing x"})
	got := mustExt(t, c, "_gobble/clear_queue", map[string]any{"sessionId": id})
	path := filepath.Join(cfg, "skills", "review", "SKILL.md")
	block := `<skill name="review" location="` + path + `">` + "\nReferences are relative to " + filepath.Dir(path) + ".\n\nCheck the diff.\n</skill>\n\nnow"
	want := map[string][]any{
		"steering": {"Review the staged changes. Focus on concurrency.", block},
		"followUp": {"Fix a then b\n", "/nothing x"},
	}
	for key, w := range want {
		if q, _ := got[key].([]any); !slices.Equal(q, w) {
			t.Errorf("%s: %q, want %q", key, got[key], w)
		}
	}
}
