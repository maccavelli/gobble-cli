package acpserver

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
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
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/personality"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// projectFiles writes the user's SYSTEM.md, skill and template into cfg,
// and a project's own into cwd, a repository root: each of a name the
// user also has.
func projectFiles(t *testing.T, cfg, cwd string) {
	t.Helper()
	writeFile(t, filepath.Join(cfg, "SYSTEM.md"), "User system.")
	writeFile(t, filepath.Join(cfg, "skills", "lint", "SKILL.md"), "---\nname: lint\ndescription: User lint.\n---\nUser.\n")
	writeFile(t, filepath.Join(cfg, "prompts", "hello.md"), "User hello.\n")
	if err := os.MkdirAll(filepath.Join(cwd, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cwd, ".gobble", "SYSTEM.md"), "Project system.")
	writeFile(t, filepath.Join(cwd, ".gobble", "APPEND_SYSTEM.md"), "Project appended.")
	writeFile(t, filepath.Join(cwd, ".gobble", "skills", "lint", "SKILL.md"), "---\nname: lint\ndescription: Project lint.\n---\nProject.\n")
	writeFile(t, filepath.Join(cwd, ".agents", "skills", "fmt", "SKILL.md"), "---\nname: fmt\ndescription: Agents fmt.\n---\nFmt.\n")
	// A loose file is a skill at .gobble/skills' top, and not at
	// .agents/skills' (skill.Source.Agents).
	writeFile(t, filepath.Join(cwd, ".gobble", "skills", "top.md"), "---\nname: top\ndescription: Top.\n---\nTop.\n")
	writeFile(t, filepath.Join(cwd, ".agents", "skills", "loose.md"), "---\nname: loose\ndescription: Loose.\n---\nLoose.\n")
	writeFile(t, filepath.Join(cwd, ".gobble", "prompts", "hello.md"), "Project hello.\n")
	writeFile(t, filepath.Join(cwd, ".gobble", "personalities", "dry", "messages.md"), "## mode.current\n\n- Project mode: {mode}\n")
}

// trustAgent is an agent over model whose sessions take settings's, with
// the read tool, client answering its questions.
func trustAgent(t *testing.T, model llm.Provider, client acp.Client, logs *bytes.Buffer, change func(*Options)) *acptest.Conn {
	t.Helper()
	opts := Options{Version: "1.2.3", Provider: func(string, llm.Retry) (llm.Provider, error) { return model, nil }, Models: func() ModelChoice { return testModels },
		Store: jsonl.New(t.TempDir(), jsonl.Options{}), Tools: []tool.Tool{builtin.Read()}, Logger: slog.New(slog.NewTextHandler(logs, nil))}
	change(&opts)
	ag := New(opts)
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c
}

// toolDescription is the named tool's description as a request sent it.
func toolDescription(t *testing.T, req *llm.Request, name string) string {
	t.Helper()
	var specs []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(req.Tools, &specs); err != nil {
		t.Fatal(err)
	}
	for _, s := range specs {
		if s.Name == name {
			return s.Description
		}
	}
	t.Fatalf("no tool %s", name)
	return ""
}

// A trusted project's files load for its session, each winning over the
// user's: SYSTEM.md and APPEND_SYSTEM.md, skills, templates, personality
// messages, sections and tool descriptions; an untrusted project's are held
// back in a session of the same agent, each with a note (0005-PLAN F6-1
// decision 8).
func TestTrustedProjectLoads(t *testing.T) {
	cfg, home, trusted, untrusted := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	projectFiles(t, cfg, trusted)
	projectFiles(t, cfg, untrusted)
	set, errs := (*personality.Set)(nil).WithProject(filepath.Join(trusted, ".gobble", "personalities"))
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	project := &ProjectOverrides{Personalities: set, Descriptions: map[string]string{"read": "Project read."}}
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"), llmtest.Text("c"), llmtest.Text("d"))
	client := &acptest.Client{}
	var logs bytes.Buffer
	c := trustAgent(t, model, client, &logs, func(o *Options) {
		o.Prompt = PromptOptions{ConfigDir: cfg, Home: home}
		o.Trust.Quiet = true
		o.Settings = func(cwd string) SessionSettings {
			if cwd == trusted {
				return SessionSettings{Personality: personality.Dry, ProjectRoot: cwd,
					Trust: permission.TrustDecision{Trusted: true, Source: permission.SourceSaved, At: cwd}, Project: project}
			}
			return SessionSettings{Personality: personality.Dry, ProjectRoot: cwd, Trust: permission.TrustDecision{Source: permission.SourceAsk}}
		}
	})
	for _, cwd := range []string{trusted, untrusted} {
		id := newSession(t, c, cwd, nil).SessionId
		for _, p := range []string{"hi", "/hello"} {
			if _, err := prompt(t, c, id, p); err != nil {
				t.Fatal(err)
			}
		}
		mode, _ := slash(t, c, client, id, "/mode")
		if got := strings.HasPrefix(mode, "Project mode: default"); got != (cwd == trusted) {
			t.Errorf("%s: /mode says %q", cwd, mode)
		}
		// A switch takes the session's personalities, its project's among
		// them.
		slash(t, c, client, id, "/personality boring")
		slash(t, c, client, id, "/personality dry")
		if mode, _ := slash(t, c, client, id, "/mode"); strings.HasPrefix(mode, "Project mode: default") != (cwd == trusted) {
			t.Errorf("%s: after a switch, /mode says %q", cwd, mode)
		}
	}
	reqs := model.Requests()
	sys := systemOf(reqs[0])
	if strings.Contains(sys, "<name>loose</name>") {
		t.Errorf("trusted: a loose file at .agents/skills' top is a skill:\n%s", sys)
	}
	for _, want := range []string{"Project appended.", "<name>lint</name>", "Project lint.", "<name>fmt</name>", "<name>top</name>"} {
		if !strings.HasPrefix(sys, "Project system.") || !strings.Contains(sys, want) || strings.Contains(sys, "User lint.") {
			t.Errorf("trusted: the system prompt lacks %q, or keeps the user's:\n%s", want, sys)
		}
	}
	if got := lastUser(reqs[1]); !strings.HasPrefix(got, "Project hello.") {
		t.Errorf("trusted: /hello sent %q", got)
	}
	if d := toolDescription(t, reqs[0], "read"); d != "Project read." {
		t.Errorf("trusted: read's description %q", d)
	}
	sys = systemOf(reqs[2])
	for _, unwanted := range []string{"Project system.", "Project appended.", "Project lint.", "Agents fmt."} {
		if !strings.HasPrefix(sys, "User system.") || strings.Contains(sys, unwanted) || !strings.Contains(sys, "User lint.") {
			t.Errorf("untrusted: the system prompt has %q, or lacks the user's:\n%s", unwanted, sys)
		}
	}
	if got := lastUser(reqs[3]); !strings.HasPrefix(got, "User hello.") {
		t.Errorf("untrusted: /hello sent %q", got)
	}
	if d := toolDescription(t, reqs[2], "read"); d == "Project read." {
		t.Error("untrusted: the trusted session's description reached another session")
	}
	var notes int
	for line := range strings.Lines(logs.String()) {
		if !strings.Contains(line, sysprompt.TrustNote) {
			continue
		}
		notes++
		if !strings.Contains(line, untrusted) || strings.Contains(line, trusted+string(filepath.Separator)) {
			t.Errorf("a note for the wrong project: %s", line)
		}
	}
	// SYSTEM.md, APPEND_SYSTEM.md, personalities, two skill directories and prompts.
	if notes != 6 {
		t.Errorf("%d notes, want 6:\n%s", notes, logs.String())
	}
}

// A trusted project's tools and rules sections replace the user's in its
// session alone (decision 8).
func TestTrustedProjectSections(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"))
	var logs bytes.Buffer
	trusted := t.TempDir()
	c := trustAgent(t, model, &acptest.Client{}, &logs, func(o *Options) {
		o.Prompt = PromptOptions{ToolsSection: "User tools.", RulesSection: "User rules."}
		o.Settings = func(cwd string) SessionSettings {
			if cwd != trusted {
				return SessionSettings{ProjectRoot: cwd}
			}
			return SessionSettings{ProjectRoot: cwd, Trust: permission.TrustDecision{Trusted: true, Source: permission.SourceSaved},
				Project: &ProjectOverrides{ToolsSection: "Project tools.", RulesSection: "Project rules."}}
		}
	})
	for _, cwd := range []string{trusted, t.TempDir()} {
		if _, err := prompt(t, c, newSession(t, c, cwd, nil).SessionId, "hi"); err != nil {
			t.Fatal(err)
		}
	}
	reqs := model.Requests()
	if sys := systemOf(reqs[0]); !strings.Contains(sys, "Project tools.") || !strings.Contains(sys, "Project rules.") || strings.Contains(sys, "User tools.") {
		t.Errorf("trusted:\n%s", sys)
	}
	if sys := systemOf(reqs[1]); !strings.Contains(sys, "User tools.") || !strings.Contains(sys, "User rules.") || strings.Contains(sys, "Project tools.") || strings.Contains(sys, "Project rules.") {
		t.Errorf("another session:\n%s", sys)
	}
}

// A trusted project's compaction prompts reach its summary request, each
// over the user's (decision 8).
func TestTrustedProjectCompactionPrompts(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text(big(100)), llmtest.Text(big(100)), llmtest.Text("S"))
	var logs bytes.Buffer
	c := trustAgent(t, model, &acptest.Client{}, &logs, func(o *Options) {
		o.Compaction = compaction.Settings{ReserveTokens: 16384, KeepRecentTokens: 150}
		o.CompactionPrompts = compaction.Prompts{System: "USER SYSTEM", Summary: "USER SUMMARY"}
		o.Settings = func(cwd string) SessionSettings {
			return SessionSettings{ProjectRoot: cwd, Trust: permission.TrustDecision{Trusted: true, Source: permission.SourceSaved},
				Project: &ProjectOverrides{CompactionPrompts: compaction.Prompts{System: "PROJECT SYSTEM"}}}
		}
	})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	for _, p := range []string{"one " + big(100), "two " + big(100), "/compact"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	req := model.Requests()[2]
	llmtest.AssertSystem(t, req, "PROJECT SYSTEM")
	if p := req.Messages[1].Content[0].Text; !strings.HasSuffix(p, "USER SUMMARY") {
		t.Fatalf("summary prompt %q, want the user's", p)
	}
}

// chooser answers each permission request with the option it names, or
// cancels, and records each request.
type chooser struct {
	*acptest.Client
	choose string
}

func (c *chooser) RequestPermission(ctx context.Context, r acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	if _, err := c.Client.RequestPermission(ctx, r); err != nil {
		return acp.RequestPermissionResponse{}, err
	}
	if c.choose == "" {
		return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}, nil
	}
	return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(acp.PermissionOptionId(c.choose))}, nil
}

// /trust says the session's decision, asks, and saves each answer for the
// project's next session: the root, its parent, or "not trusted"; a cancel
// and a failed save save nothing (decision 7).
func TestTrustCommand(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "work", "repo")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	type save struct {
		dir     string
		trusted bool
	}
	var saves []save
	fail := false
	client := &chooser{Client: &acptest.Client{}}
	var logs bytes.Buffer
	c := trustAgent(t, llmtest.NewScript(), client, &logs, func(o *Options) {
		o.Trust = TrustOptions{Home: base, Quiet: true, Save: func(_ context.Context, dir string, trusted bool) (string, error) {
			if fail {
				return "", errors.New("disk full")
			}
			saves = append(saves, save{dir, trusted})
			return dir, nil
		}}
		o.Settings = func(cwd string) SessionSettings {
			return SessionSettings{ProjectRoot: root, Trust: permission.TrustDecision{Source: permission.SourceAsk}}
		}
	})
	// The session runs in a directory of the project, which /trust names
	// by its root.
	id := newSession(t, c, filepath.Join(root, "sub"), nil).SessionId
	parent := filepath.Dir(root)
	for _, tc := range []struct {
		choose, want string
		saved        []save
	}{
		{trustYes, "Saved: " + root + " is trusted. It applies from this project's next session.", []save{{root, true}}},
		{trustParent, "Saved: " + parent + " is trusted.", []save{{parent, true}}},
		{trustNo, "Saved: " + root + " is not trusted.", []save{{root, false}}},
		{"", "Nothing was saved.", nil},
	} {
		saves = nil
		client.choose = tc.choose
		text, _ := slash(t, c, client.Client, id, "/trust")
		if !strings.HasPrefix(text, "This project, "+root+", is not trusted: nothing is saved") || !strings.Contains(text, tc.want) {
			t.Errorf("%q: /trust said %q", tc.choose, text)
		}
		if len(saves) != len(tc.saved) || (len(saves) == 1 && saves[0] != tc.saved[0]) {
			t.Errorf("%q: saved %v, want %v", tc.choose, saves, tc.saved)
		}
	}
	reqs := client.Permissions()
	last := reqs[len(reqs)-1]
	if last.ToolCall.ToolCallId != TrustCallID || last.ToolCall.Title == nil || *last.ToolCall.Title != "Trust "+root || len(last.Options) != 3 ||
		last.Options[1].Name != "Trust the parent folder ("+parent+")" || last.Options[1].Kind != acp.PermissionOptionKindAllowAlways ||
		last.Options[2].Kind != acp.PermissionOptionKindRejectAlways {
		t.Fatalf("the question %+v", last)
	}
	fail, client.choose = true, trustYes
	if text, _ := slash(t, c, client.Client, id, "/trust"); !strings.Contains(text, "The decision was not saved: disk full") {
		t.Errorf("a failed save said %q", text)
	}
	// A root whose parent is home is not offered its parent (decision 3).
	home := filepath.Join(base, "work")
	o := trustAgent(t, llmtest.NewScript(), client, &logs, func(o *Options) {
		o.Trust = TrustOptions{Home: home, Save: func(context.Context, string, bool) (string, error) { return "", nil }}
		o.Settings = func(string) SessionSettings { return SessionSettings{ProjectRoot: root} }
	})
	client.choose = ""
	slash(t, o, client.Client, newSession(t, o, root, nil).SessionId, "/trust")
	reqs = client.Permissions()
	if opts := reqs[len(reqs)-1].Options; len(opts) != 2 || opts[1].OptionId != trustNo {
		t.Fatalf("home offered as the parent: %+v", opts)
	}
}

// An untrusted project is told once, at the session's first reply, and
// neither the session file nor the model has it; a quiet agent, a trusted
// project and a session whose settings decided nothing are told nothing
// (decision 9).
func TestUntrustedNotice(t *testing.T) {
	const notice = "this project is not trusted: its .gobble files and skills are not loaded; gobble trust trusts it"
	for _, tc := range []struct {
		name  string
		trust permission.TrustDecision
		quiet bool
		told  bool
	}{
		{"untrusted", permission.TrustDecision{Source: permission.SourceAsk}, false, true},
		{"saved untrusted", permission.TrustDecision{Source: permission.SourceSaved}, false, true},
		{"quiet", permission.TrustDecision{Source: permission.SourceAsk}, true, false},
		{"trusted", permission.TrustDecision{Trusted: true, Source: permission.SourceSaved}, false, false},
		{"undecided", permission.TrustDecision{}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := llmtest.NewScript(llmtest.Text("one"), llmtest.Text("two"))
			client := &acptest.Client{}
			var logs bytes.Buffer
			dir := t.TempDir()
			c := trustAgent(t, model, client, &logs, func(o *Options) {
				o.Store = jsonl.New(dir, jsonl.Options{})
				o.Trust.Quiet = tc.quiet
				o.Settings = func(cwd string) SessionSettings { return SessionSettings{ProjectRoot: cwd, Trust: tc.trust} }
			})
			id := newSession(t, c, t.TempDir(), nil).SessionId
			first, _ := slash(t, c, client, id, "hi")
			second, _ := slash(t, c, client, id, "again")
			if strings.HasPrefix(first, notice+"\n\none") != tc.told || strings.Contains(second, notice) {
				t.Fatalf("replies %q and %q", first, second)
			}
			if !tc.told && strings.Contains(first, notice) {
				t.Fatalf("told: %q", first)
			}
			if r := strings.Join(requestTexts(model.Requests()[1]), "\n"); strings.Contains(r, notice) {
				t.Fatalf("the model saw it: %q", r)
			}
			files, err := filepath.Glob(filepath.Join(dir, "*", "*_"+string(id)+".jsonl"))
			if err != nil || len(files) != 1 {
				t.Fatalf("session files %v, %v", files, err)
			}
			if b, err := os.ReadFile(files[0]); err != nil || strings.Contains(string(b), notice) || !strings.Contains(string(b), "two") {
				t.Fatalf("the session file has it, or lacks the replies (%v):\n%s", err, b)
			}
		})
	}
}
