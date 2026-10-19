package acpserver

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// rulesAgent joins an agent on p whose sessions take ss, with tools (nil is
// the built-ins), to client, and returns its log.
func rulesAgent(t *testing.T, p llm.Provider, client *acptest.Client, ss SessionSettings, tools []tool.Tool) (*acptest.Conn, *syncBuffer) {
	t.Helper()
	logs := &syncBuffer{}
	a := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Tools: tools, Settings: func(string) SessionSettings { return ss }, Logger: slog.New(slog.NewTextHandler(logs, nil))})
	c := acptest.Connect(t, a, client)
	a.SetConnection(c.Agent)
	return c, logs
}

// rule is text's rule of effect, from line of the global settings.yaml.
func rule(t *testing.T, effect permission.Mode, text string, line int) permission.Spec {
	t.Helper()
	s, err := permission.ParseSpec(effect, text)
	if err != nil {
		t.Fatal(err)
	}
	s.Source = permission.RuleSource{Layer: permission.LayerGlobal, File: "settings.yaml", Line: line}
	return s
}

// A call is decided by the session's rules over the defaults, then by
// gobble's limits, which no rule lifts; a deny reaches a read; a path
// outside the roots is external_directory's; plan mode refuses what is not
// read-only whatever the rules say; each decision is logged with its rule
// and source (0005-PLAN F6-2 decisions 3 to 7 and 12).
func TestRulesDecide(t *testing.T) {
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret.txt"), "the secret is 42")
	home := t.TempDir()
	for _, tc := range []struct {
		name  string
		rules func(cwd string) []permission.Spec
		call  func(cwd string) (string, map[string]string)
		allow bool // the client's answer to a question
		plan  bool
		asked int
		limit bool   // the question carries _meta.gobble.limit
		reply string // in the reply
		log   string // in the log
		check func(t *testing.T, cwd string)
	}{
		{name: "a read inside runs unasked",
			call:  func(string) (string, map[string]string) { return "read", map[string]string{"path": "f.txt"} },
			reply: "hello", log: `effect=allow rule="allow: read" from=default`},
		{name: "a deny reaches a read, at any depth",
			rules: func(string) []permission.Spec {
				return []permission.Spec{rule(t, permission.ModeDeny, "read(.env)", 3)}
			},
			call:  func(string) (string, map[string]string) { return "read", map[string]string{"path": "app/.env"} },
			reply: `read did not run: the permission rule "deny: read(.env)" (settings.yaml:3 (global)) denies it.`,
			log:   `effect=deny rule="deny: read(.env)" from="settings.yaml:3 (global)"`},
		{name: "no rule asks for a write",
			call: func(string) (string, map[string]string) {
				return "write", map[string]string{"path": "a.txt", "content": "x"}
			},
			asked: 1, reply: "declined", log: `effect=ask rule="ask: write" from=default`,
			check: func(t *testing.T, cwd string) { noFile(t, filepath.Join(cwd, "a.txt")) }},
		{name: "an allow runs a write",
			rules: func(string) []permission.Spec { return []permission.Spec{rule(t, permission.ModeAllow, "write", 4)} },
			call: func(string) (string, map[string]string) {
				return "write", map[string]string{"path": "a.txt", "content": "x"}
			},
			check: func(t *testing.T, cwd string) { hasFile(t, filepath.Join(cwd, "a.txt")) }},
		{name: "the last match wins",
			rules: func(string) []permission.Spec {
				return []permission.Spec{rule(t, permission.ModeAllow, "write", 4), rule(t, permission.ModeDeny, "write(*.txt)", 5)}
			},
			call: func(string) (string, map[string]string) {
				return "write", map[string]string{"path": "a.txt", "content": "x"}
			},
			reply: `the permission rule "deny: write(*.txt)" (settings.yaml:5 (global)) denies it`,
			check: func(t *testing.T, cwd string) { noFile(t, filepath.Join(cwd, "a.txt")) }},
		{name: "a configuration path asks though allowed",
			rules: func(string) []permission.Spec { return []permission.Spec{rule(t, permission.ModeAllow, "write", 4)} },
			call: func(string) (string, map[string]string) {
				return "write", map[string]string{"path": "AGENTS.md", "content": "x"}
			},
			allow: true, asked: 1, limit: true, log: "limit=",
			check: func(t *testing.T, cwd string) { hasFile(t, filepath.Join(cwd, "AGENTS.md")) }},
		{name: "a delete asks though allowed",
			rules: func(string) []permission.Spec { return []permission.Spec{rule(t, permission.ModeAllow, "delete", 4)} },
			call:  func(string) (string, map[string]string) { return "delete", map[string]string{"path": "f.txt"} },
			asked: 1, limit: true, reply: "declined",
			check: func(t *testing.T, cwd string) { hasFile(t, filepath.Join(cwd, "f.txt")) }},
		{name: "a protected path is denied though allowed",
			rules: func(string) []permission.Spec { return []permission.Spec{rule(t, permission.ModeAllow, "write", 4)} },
			call: func(string) (string, map[string]string) {
				return "write", map[string]string{"path": ".git/config", "content": "x"}
			},
			allow: true, reply: "write did not run: gobble never changes files under .git", log: "from=limit",
			check: func(t *testing.T, cwd string) { noFile(t, filepath.Join(cwd, ".git", "config")) }},
		{name: "~/.ssh is denied though both rules allow",
			rules: func(string) []permission.Spec {
				return []permission.Spec{rule(t, permission.ModeAllow, "write", 4), rule(t, permission.ModeAllow, "external_directory", 5)}
			},
			call: func(string) (string, map[string]string) {
				return "write", map[string]string{"path": filepath.Join(home, ".ssh", "config"), "content": "x"}
			},
			allow: true, reply: "gobble never changes files under .git, under ~/.ssh",
			check: func(t *testing.T, _ string) { noFile(t, filepath.Join(home, ".ssh", "config")) }},
		{name: "an outside read asks",
			call: func(string) (string, map[string]string) {
				return "read", map[string]string{"path": filepath.Join(outside, "secret.txt")}
			},
			asked: 1, reply: "declined", log: `effect=ask rule="ask: external_directory" from=default`},
		{name: "external_directory allowed by a rule",
			rules: func(string) []permission.Spec {
				return []permission.Spec{rule(t, permission.ModeAllow, "external_directory("+filepath.ToSlash(outside)+"/**)", 6)}
			},
			call: func(string) (string, map[string]string) {
				return "read", map[string]string{"path": filepath.Join(outside, "secret.txt")}
			},
			reply: "the secret is 42", log: "from=\"settings.yaml:6 (global)\""},
		{name: "plan mode refuses an allowed write",
			rules: func(string) []permission.Spec { return []permission.Spec{rule(t, permission.ModeAllow, "write", 4)} },
			call: func(string) (string, map[string]string) {
				return "write", map[string]string{"path": "a.txt", "content": "x"}
			},
			plan: true, allow: true, reply: "Plan mode is read-only, so write did not run.",
			check: func(t *testing.T, cwd string) { noFile(t, filepath.Join(cwd, "a.txt")) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			writeFile(t, filepath.Join(cwd, "f.txt"), "hello")
			writeFile(t, filepath.Join(cwd, "app", ".env"), "KEY=1")
			ss := SessionSettings{Home: home}
			if tc.rules != nil {
				ss.Rules = tc.rules(cwd)
			}
			name, args := tc.call(cwd)
			client := &acptest.Client{Allow: tc.allow}
			model := &echoResult{call: llm.ToolCallDone{ID: "c1", Name: name, Args: toolArgs(t, args)}}
			c, logs := rulesAgent(t, model, client, ss, nil)
			id := openSession(t, c, cwd)
			if tc.plan {
				if _, err := c.Client.SetSessionMode(t.Context(), acp.SetSessionModeRequest{SessionId: id, ModeId: modePlan}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := prompt(t, c, id, "go"); err != nil {
				t.Fatal(err)
			}
			perms := client.Permissions()
			if len(perms) != tc.asked {
				t.Fatalf("%d questions %+v, want %d", len(perms), perms, tc.asked)
			}
			if tc.asked > 0 {
				marks, _ := perms[0].Meta[gobble].(map[string]any)
				_, hasLimit := marks[keyLimit]
				if hasLimit != tc.limit {
					t.Errorf("question _meta %v, want a limit mark %v", perms[0].Meta, tc.limit)
				}
			}
			if reply := agentText(client.Updates()); !strings.Contains(reply, tc.reply) {
				t.Errorf("reply %q, want %q in it", reply, tc.reply)
			}
			if got := logs.String(); !strings.Contains(got, "permission: decided") || !strings.Contains(got, tc.log) {
				t.Errorf("log %q, want a decision with %q", got, tc.log)
			}
			if tc.check != nil {
				tc.check(t, cwd)
			}
		})
	}
}

func hasFile(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Errorf("%s: %v, want it written", p, err)
	}
}

func noFile(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err == nil {
		t.Errorf("%s exists, want it not written", p)
	}
}

// A session's limits are placed by its settings' home and config
// directories, or by the user's home and PromptOptions.ConfigDir
// (0005-PLAN F6-2 decision 5).
func TestSessionLimitsPlaced(t *testing.T) {
	a := New(Options{Prompt: PromptOptions{ConfigDir: "/c"}})
	if ss := a.sessionSettings("/x"); ss.ConfigDir != "/c" {
		t.Errorf("ConfigDir %q, want PromptOptions'", ss.ConfigDir)
	}
	a = New(Options{Prompt: PromptOptions{ConfigDir: "/c"}, Settings: func(string) SessionSettings { return SessionSettings{Home: "/h", ConfigDir: "/d"} }})
	if ss := a.sessionSettings("/x"); ss.ConfigDir != "/d" || ss.Home != "/h" {
		t.Errorf("settings' own: %q, %q", ss.Home, ss.ConfigDir)
	}
}

// deferredStub is a deferred, read-only tool named name.
func deferredStub(name string) tool.Tool {
	return tool.New(name, "A tool for "+name+".", func(context.Context, struct{}, tool.Env) (string, error) { return name, nil },
		tool.WithExposure(tool.ExposureDeferred), tool.WithFamily("stubs"), tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true}))
}

// A tool whose last matching rule is a deny with no pattern, or "*", is
// neither sent nor found by tool_search, nor named in the system prompt; a
// deny with a pattern leaves it (0005-PLAN F6-2 decision 6).
func TestHiddenTools(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("s1", builtin.ToolSearchName, `{"query":"secret_tool"}`), llmtest.Text("ok"))
	ss := SessionSettings{Rules: []permission.Spec{
		rule(t, permission.ModeDeny, "delete", 3), rule(t, permission.ModeDeny, "secret_*(*)", 4), rule(t, permission.ModeDeny, "edit(*.go)", 5),
	}}
	tools := append(builtin.Tools(), deferredStub("secret_tool"), deferredStub("other_tool"))
	c, _ := rulesAgent(t, s, &acptest.Client{}, ss, tools)
	id := openSession(t, c, t.TempDir())
	if _, err := prompt(t, c, id, "go"); err != nil {
		t.Fatal(err)
	}
	reqs := s.Requests()
	sent := requestTools(t, reqs[0])
	if slices.Contains(sent, "delete") || !slices.Contains(sent, "edit") || !slices.Contains(sent, builtin.ToolSearchName) {
		t.Fatalf("tools sent %q: want delete hidden, edit and tool_search kept", sent)
	}
	var sys string
	for _, m := range reqs[0].Messages {
		if m.Role == llm.RoleSystem {
			sys = m.Content[0].Text
		}
	}
	if strings.Contains(sys, "A delete always asks") || !strings.Contains(sys, "Read a file before you edit it.") {
		t.Errorf("system prompt %q: want delete's line gone, edit's kept", sys)
	}
	last := reqs[1].Messages[len(reqs[1].Messages)-1].Content[0]
	if strings.Contains(last.Text, "- secret_tool:") || !strings.Contains(last.Text, "- other_tool:") {
		t.Fatalf("tool_search found %q; want secret_tool hidden, other_tool found", last.Text)
	}
}
