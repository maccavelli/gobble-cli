package acpserver

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// globalFile stands in for the global settings file's rules.
type globalFile struct {
	mu    sync.Mutex
	rules []permission.Spec
	fail  error
}

func (g *globalFile) add(r permission.Spec) ([]permission.Spec, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.fail != nil {
		return nil, g.fail
	}
	r.Source = permission.RuleSource{Layer: permission.LayerGlobal, File: "settings.yaml", Line: 10 + len(g.rules)}
	g.rules = append(g.rules, r)
	return slices.Clone(g.rules), nil
}

func (g *globalFile) remove(r permission.Spec) ([]permission.Spec, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	i := slices.IndexFunc(g.rules, func(o permission.Spec) bool { return o == r })
	if i < 0 {
		return nil, errors.New("not in the file")
	}
	g.rules = slices.Delete(g.rules, i, i+1)
	return slices.Clone(g.rules), nil
}

func (g *globalFile) texts() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []string
	for _, r := range g.rules {
		out = append(out, r.String())
	}
	return out
}

// answersAgent joins an agent on p, whose sessions take ss and save to
// perms, with tools (nil is the built-ins), to client.
func answersAgent(t *testing.T, p llm.Provider, client *acptest.Client, ss SessionSettings, perms PermissionOptions, tools []tool.Tool) *acptest.Conn {
	t.Helper()
	a := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Tools: tools, Settings: func(string) SessionSettings { return ss }, Permissions: perms, Logger: slog.New(slog.DiscardHandler)})
	c := acptest.Connect(t, a, client)
	a.SetConnection(c.Agent)
	return c
}

// choose answers every question with id, and feedback in _meta when set.
func choose(id acp.PermissionOptionId, feedback string) func(acp.RequestPermissionRequest) acp.RequestPermissionResponse {
	return func(acp.RequestPermissionRequest) acp.RequestPermissionResponse {
		resp := acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(id)}
		if feedback != "" {
			resp.Meta = map[string]any{gobble: map[string]any{keyFeedback: feedback}}
		}
		return resp
	}
}

// lastResult is the text of the tool result request i sent last.
func lastResult(t *testing.T, s *llmtest.Script, i int) string {
	t.Helper()
	reqs := s.Requests()
	if len(reqs) <= i {
		t.Fatalf("%d requests, want %d", len(reqs), i+1)
	}
	m := reqs[i].Messages[len(reqs[i].Messages)-1]
	return m.Content[0].Text
}

func writeA(id, content string) llmtest.Turn {
	return llmtest.ToolCall(id, "write", `{"path":"a.txt","content":"`+content+`"}`)
}

// Each "always" answer saves a rule that matches the call alone, where it
// was chosen, and the turn's next call obeys it without asking; a project
// answer applies to the project's next session (0005-PLAN F6-2b decisions
// C and E).
func TestAlwaysAnswers(t *testing.T) {
	for _, tc := range []struct {
		option acp.PermissionOptionId
		allow  bool
		from   string // the listed source, a suffix
	}{
		{allowSession, true, "(session)"},
		{allowProject, true, "(project store))"},
		{allowAlways, true, "(settings.yaml:10 (global))"},
		{rejectSession, false, "(session)"},
		{rejectProject, false, "(project store))"},
		{rejectAlways, false, "(settings.yaml:10 (global))"},
	} {
		t.Run(string(tc.option), func(t *testing.T) {
			cwd := t.TempDir()
			global := &globalFile{}
			perms := PermissionOptions{Store: permission.RuleStore{Path: filepath.Join(t.TempDir(), permission.PermissionsFile)},
				AddGlobal: global.add, RemoveGlobal: global.remove}
			s := llmtest.NewScript(writeA("w1", "x"), writeA("w2", "y"), llmtest.Text("done"))
			client := &acptest.Client{Answer: choose(tc.option, "")}
			c := answersAgent(t, s, client, SessionSettings{}, perms, nil)
			id := openSession(t, c, cwd)
			if _, err := prompt(t, c, id, "go"); err != nil {
				t.Fatal(err)
			}
			if n := len(client.Permissions()); n != 1 {
				t.Fatalf("%d questions, want the first call's alone", n)
			}
			data, err := os.ReadFile(filepath.Join(cwd, "a.txt"))
			if tc.allow != (err == nil && string(data) == "y") {
				t.Fatalf("a.txt %q, %v; want both writes run %v", data, err, tc.allow)
			}
			effect := "deny"
			if tc.allow {
				effect = "allow"
			} else if got := lastResult(t, s, 2); !strings.Contains(got, `the permission rule "deny: write(/a.txt)"`) {
				t.Fatalf("the second call: %q, want the saved rule's refusal", got)
			}
			list, _ := slash(t, c, client, id, "/permissions")
			if !strings.Contains(list, "3. "+effect+": write(/a.txt) ") || !strings.Contains(list, tc.from) {
				t.Fatalf("/permissions:\n%s\nwant the rule from %s", list, tc.from)
			}
			switch tc.option {
			case allowProject, rejectProject:
				real, _ := filepath.EvalSymlinks(cwd)
				m, _ := perms.Store.Load()
				if len(m[real]) != 1 || m[real][0] != permission.StoredOf(permission.Spec{Effect: permission.Mode(effect), Tool: "write", Pattern: "/a.txt"}) {
					t.Fatalf("store %v", m)
				}
				// The project's next session takes it, untrusted though it
				// is (decision B).
				s2 := llmtest.NewScript(writeA("w3", "z"), llmtest.Text("done"))
				c2 := answersAgent(t, s2, &acptest.Client{}, SessionSettings{ProjectRoot: cwd}, perms, nil)
				if _, err := prompt(t, c2, openSession(t, c2, cwd), "go"); err != nil {
					t.Fatal(err)
				}
				if data, _ := os.ReadFile(filepath.Join(cwd, "a.txt")); tc.allow != (string(data) == "z") {
					t.Fatalf("the next session: a.txt %q", data)
				}
			case allowAlways, rejectAlways:
				if got := global.texts(); len(got) != 1 || got[0] != effect+": write(/a.txt)" {
					t.Fatalf("global file %q", got)
				}
			}
		})
	}
}

// pingStub is a tool that changes something and names no subject.
func pingStub() tool.Tool {
	return tool.New("ping", "Ping.", func(context.Context, struct{}, tool.Env) (string, error) { return "pong", nil })
}

// The question offers decision D's eight options, or the two "once" ones
// for a limit, a repeated call and a call no rule can match alone, with
// the reason in _meta.gobble.onceOnly; each carries its deciding rule in
// _meta.gobble.rule; a "never" for a tool with no subject says it hides
// the tool (decisions A, C, D and G).
func TestQuestionOptions(t *testing.T) {
	ask := func(t *testing.T, call llmtest.Turn, tools []tool.Tool) acp.RequestPermissionRequest {
		t.Helper()
		client := &acptest.Client{}
		c := answersAgent(t, llmtest.NewScript(call, llmtest.Text("done")), client, SessionSettings{}, PermissionOptions{}, tools)
		cwd := t.TempDir()
		writeFile(t, filepath.Join(cwd, "f.txt"), "x")
		if _, err := prompt(t, c, openSession(t, c, cwd), "go"); err != nil {
			t.Fatal(err)
		}
		perms := client.Permissions()
		if len(perms) != 1 {
			t.Fatalf("%d questions", len(perms))
		}
		return perms[0]
	}
	ids := func(r acp.RequestPermissionRequest) (out []string) {
		for _, o := range r.Options {
			out = append(out, string(o.OptionId)+"="+o.Name)
		}
		return out
	}
	marks := func(r acp.RequestPermissionRequest) map[string]any {
		m, _ := r.Meta[gobble].(map[string]any)
		return m
	}
	q := ask(t, writeA("w1", "x"), nil)
	want := []string{"allow_once=Allow once", "allow_session=Allow for this session", "allow_project=Allow for this project",
		"allow_always=Allow everywhere", "reject_once=Reject", "reject_session=Reject for this session",
		"reject_project=Reject for this project", "reject_always=Reject everywhere"}
	if got := ids(q); !slices.Equal(got, want) {
		t.Errorf("a write's options %q", got)
	}
	if r, _ := marks(q)[keyRule].(map[string]any); r[keyRule] != "ask: write" || r[keyFrom] != "default" || len(marks(q)) != 1 {
		t.Errorf("a write's _meta %v", q.Meta)
	}
	once := []string{"allow_once=Allow once", "reject_once=Reject"}
	q = ask(t, llmtest.ToolCall("d1", "delete", `{"path":"f.txt"}`), nil)
	if got := ids(q); !slices.Equal(got, once) || marks(q)[keyLimit] == nil || marks(q)[keyRule] == nil {
		t.Errorf("a delete: %q, _meta %v", got, q.Meta)
	}
	q = ask(t, llmtest.ToolCall("b1", "bash", `{"command":"./run-*.sh"}`), nil)
	if got := ids(q); !slices.Equal(got, once) || marks(q)[keyOnceOnly] != textOnceOnly || marks(q)[keyLimit] != nil {
		t.Errorf("a glob command: %q, _meta %v", got, q.Meta)
	}
	q = ask(t, llmtest.ToolCall("p1", "ping", `{}`), append(builtin.Tools(), pingStub()))
	if got := ids(q); len(got) != 8 || got[7] != "reject_always=Reject everywhere, hiding the tool" || got[5] != "reject_session=Reject for this session, hiding the tool" {
		t.Errorf("a tool with no subject: %q", got)
	}
}

// A "never" for a tool with no subject hides it from the next turn's
// request (decisions C and 6).
func TestRejectAlwaysHides(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("p1", "ping", `{}`), llmtest.Text("done"), llmtest.Text("again"))
	c := answersAgent(t, s, &acptest.Client{Answer: choose(rejectSession, "")}, SessionSettings{}, PermissionOptions{}, append(builtin.Tools(), pingStub()))
	id := openSession(t, c, t.TempDir())
	for _, text := range []string{"ping it", "and again"} {
		if _, err := prompt(t, c, id, text); err != nil {
			t.Fatal(err)
		}
	}
	reqs := s.Requests()
	if !slices.Contains(requestTools(t, reqs[0]), "ping") || slices.Contains(requestTools(t, reqs[2]), "ping") {
		t.Fatalf("tools %q, then %q; want ping, then not", requestTools(t, reqs[0]), requestTools(t, reqs[2]))
	}
}

// An "always" for an outside read saves the read's rule and an
// external_directory one, and the turn's next read of it is not asked
// (decision C).
func TestAlwaysOutside(t *testing.T) {
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	writeFile(t, secret, "42")
	read := func(id string) llmtest.Turn { return llmtest.ToolCall(id, "read", `{"path":`+jsonQuote(t, secret)+`}`) }
	s := llmtest.NewScript(read("r1"), read("r2"), llmtest.Text("done"))
	client := &acptest.Client{Answer: choose(allowSession, "")}
	c := answersAgent(t, s, client, SessionSettings{}, PermissionOptions{}, nil)
	id := openSession(t, c, t.TempDir())
	if _, err := prompt(t, c, id, "go"); err != nil {
		t.Fatal(err)
	}
	if n := len(client.Permissions()); n != 1 || !strings.Contains(lastResult(t, s, 2), "42") {
		t.Fatalf("%d questions, the second read %q", n, lastResult(t, s, 2))
	}
	list, _ := slash(t, c, client, id, "/permissions")
	p := filepath.ToSlash(secret)
	if !strings.Contains(list, "3. allow: read("+p+") (session)") || !strings.Contains(list, "4. allow: external_directory("+p+") (session)") {
		t.Fatalf("/permissions:\n%s", list)
	}
}

// A reject's reason reaches the model after the decline, cut to 2 KiB; an
// allow's is left (decision F).
func TestFeedback(t *testing.T) {
	for _, tc := range []struct {
		option   acp.PermissionOptionId
		feedback string
		want     string
	}{
		{rejectOnce, "use git mv", "The user declined this call, so it did not run. The user said: use git mv"},
		{rejectSession, "not here", "The user declined this call, so it did not run. The user said: not here"},
		{rejectOnce, strings.Repeat("é", 2000), "The user said: " + strings.Repeat("é", 1022) + "…"},
		{allowOnce, "fine", "wrote a.txt"},
	} {
		s := llmtest.NewScript(writeA("w1", "x"), llmtest.Text("done"))
		c := answersAgent(t, s, &acptest.Client{Answer: choose(tc.option, tc.feedback)}, SessionSettings{}, PermissionOptions{}, nil)
		if _, err := prompt(t, c, openSession(t, c, t.TempDir()), "go"); err != nil {
			t.Fatal(err)
		}
		got := lastResult(t, s, 1)
		if !strings.Contains(got, tc.want) || tc.option == allowOnce && strings.Contains(got, "said") {
			t.Errorf("%s %.20q: result %.200q, want %.200q", tc.option, tc.feedback, got, tc.want)
		}
	}
}

// An "always" that cannot be saved where it was chosen is told, and holds
// for the session (decision E).
func TestAlwaysSaveFailed(t *testing.T) {
	global := &globalFile{fail: errors.New("disk full")}
	s := llmtest.NewScript(writeA("w1", "x"), writeA("w2", "y"), llmtest.Text("done"))
	client := &acptest.Client{Answer: choose(allowAlways, "")}
	c := answersAgent(t, s, client, SessionSettings{}, PermissionOptions{AddGlobal: global.add}, nil)
	id := openSession(t, c, t.TempDir())
	if _, err := prompt(t, c, id, "go"); err != nil {
		t.Fatal(err)
	}
	if n := len(client.Permissions()); n != 1 {
		t.Fatalf("%d questions", n)
	}
	if text := agentText(client.Updates()); !strings.Contains(text, "allow: write(/a.txt) was not saved where you chose: disk full. It applies to this session.") {
		t.Fatalf("reply %q", text)
	}
	for _, perms := range []PermissionOptions{{}, {Store: permission.RuleStore{}}} {
		s := llmtest.NewScript(writeA("w1", "x"), llmtest.Text("done"))
		client := &acptest.Client{Answer: choose(allowProject, "")}
		c := answersAgent(t, s, client, SessionSettings{}, perms, nil)
		if _, err := prompt(t, c, openSession(t, c, t.TempDir()), "go"); err != nil {
			t.Fatal(err)
		}
		if text := agentText(client.Updates()); !strings.Contains(text, errNoStore.Error()) {
			t.Fatalf("no store: reply %q", text)
		}
	}
}

// /permissions lists the rules in force in order with their sources,
// between the defaults and the limits; removes a session, store or global
// rule and refuses a project's and --set's; adds to each scope; and says
// how to use it (decision H).
func TestPermissionsCommand(t *testing.T) {
	cwd := t.TempDir()
	global := &globalFile{}
	if _, err := global.add(rule(t, permission.ModeAllow, "read", 0)); err != nil {
		t.Fatal(err)
	}
	perms := PermissionOptions{Store: permission.RuleStore{Path: filepath.Join(t.TempDir(), permission.PermissionsFile)},
		AddGlobal: global.add, RemoveGlobal: global.remove}
	if _, err := perms.Store.Add(t.Context(), cwd, rule(t, permission.ModeAsk, "grep", 0)); err != nil {
		t.Fatal(err)
	}
	project := rule(t, permission.ModeDeny, "read(.env)", 2)
	project.Source = permission.RuleSource{Layer: permission.LayerProject, File: "proj.yaml", Line: 2}
	set := rule(t, permission.ModeAsk, "bash(*)", 1)
	set.Source = permission.RuleSource{Layer: layerSetName}
	ss := SessionSettings{Rules: slices.Concat(global.rules, []permission.Spec{project, set})}
	client := &acptest.Client{}
	c := answersAgent(t, llmtest.NewScript(), client, ss, perms, nil)
	id := openSession(t, c, cwd)
	run := func(text string) string {
		t.Helper()
		got, _ := slash(t, c, client, id, text)
		return got
	}
	list := run("/permissions")
	// The default rules come first (0005-PLAN F6-3 decision 10).
	for _, want := range []string{"With no rule, a read-only built-in tool runs", "1. ask: bash(gobble update*) (default)",
		"2. ask: powershell(gobble update*) (default)", "3. allow: read (settings.yaml:10 (global))",
		"4. deny: read(.env) (proj.yaml:2 (project))", "5. ask: bash(*) (--set)", "6. ask: grep (" + perms.Store.Path + " (project store))",
		"Whatever the rules say"} {
		if !strings.Contains(list, want) {
			t.Errorf("list lacks %q:\n%s", want, list)
		}
	}
	for _, c := range []struct{ cmd, want string }{
		{"/permissions allow todo", "Added allow: todo (session)."},
		{"/permissions deny edit(/x.go) project", "Added deny: edit(/x.go) (" + perms.Store.Path + " (project store))."},
		{"/permissions ask bash(make test) global", "Added ask: bash(make test) (settings.yaml:11 (global))."},
		{"/permissions allow read([a)", "That is not a rule:"},
		{"/permissions frob", "/permissions lists the rules;"},
		{"/permissions remove 99", "There is no rule 99;"},
		{"/permissions remove 1", "Rule 1, ask: bash(gobble update*), is set in default; change it there."},
		{"/permissions remove 5", "Rule 5, deny: read(.env), is set in proj.yaml:2 (project); change it there."},
		{"/permissions remove 6", "Rule 6, ask: bash(*), is set in --set; change it there."},
	} {
		if got := run(c.cmd); !strings.Contains(got, c.want) {
			t.Errorf("%s: %q, want %q", c.cmd, got, c.want)
		}
	}
	// In force now: 1 and 2 the defaults, 3 read (global), 4 bash(make test)
	// (global), 5 read(.env), 6 bash(*), 7 grep (store), 8 edit(/x.go)
	// (store), 9 todo (session).
	for _, c := range []struct{ cmd, want string }{
		{"/permissions remove 9", "Removed allow: todo (session)."},
		{"/permissions remove 8", "Removed deny: edit(/x.go)"},
		{"/permissions remove 4", "Removed ask: bash(make test) (settings.yaml:11 (global))."},
	} {
		if got := run(c.cmd); !strings.Contains(got, c.want) {
			t.Errorf("%s: %q, want %q", c.cmd, got, c.want)
		}
	}
	if got := global.texts(); !slices.Equal(got, []string{"allow: read"}) {
		t.Errorf("global file %q", got)
	}
	if got, _ := perms.Store.Rules(cwd); len(got) != 1 || got[0].String() != "ask: grep" {
		t.Errorf("store %v", got)
	}
	list = run("/permissions")
	if strings.Contains(list, "todo") || strings.Contains(list, "make test") || strings.Contains(list, "x.go") || !strings.Contains(list, "6. ask: grep") || !strings.Contains(list, "1. ask: bash(gobble update*) (default)") {
		t.Errorf("after the removals:\n%s", list)
	}
}

func jsonQuote(t *testing.T, s string) string {
	t.Helper()
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}
