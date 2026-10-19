package acpserver

import (
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
)

// modeAgent joins an agent on s whose sessions take ss, starting in
// override, to client; it opens a session in cwd, sets mode when it is not
// "", and gives the session and the agent's log.
func modeAgent(t *testing.T, s llm.Provider, client *acptest.Client, ss SessionSettings, override, mode, cwd string) (*acptest.Conn, acp.SessionId, *syncBuffer) {
	t.Helper()
	logs := &syncBuffer{}
	a := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return s, nil },
		Settings: func(string) SessionSettings { return ss }, Override: Override{Mode: override}, Logger: slog.New(slog.NewTextHandler(logs, nil))})
	c := acptest.Connect(t, a, client)
	a.SetConnection(c.Agent)
	id := openSession(t, c, cwd)
	if mode != "" {
		if _, err := c.Client.SetSessionMode(t.Context(), acp.SetSessionModeRequest{SessionId: id, ModeId: acp.SessionModeId(mode)}); err != nil {
			t.Fatal(err)
		}
	}
	return c, id, logs
}

// The modes offered are default, accept-edits and plan, and bypass with
// allowBypass; set_mode and /mode refuse bypass without it (0005-PLAN F6-4
// decision 1).
func TestModesOffered(t *testing.T) {
	for _, allow := range []bool{false, true} {
		client := &acptest.Client{}
		a := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return llmtest.NewScript(), nil },
			Settings: func(string) SessionSettings { return SessionSettings{AllowBypass: allow} }})
		c := acptest.Connect(t, a, client)
		a.SetConnection(c.Agent)
		initialize(t, c.Client)
		s := newSessionIn(t, c, t.TempDir())
		var ids []string
		for _, m := range s.Modes.AvailableModes {
			ids = append(ids, string(m.Id))
		}
		want := []string{"default", "accept-edits", "plan"}
		if allow {
			want = append(want, "bypass")
		}
		if !slices.Equal(ids, want) {
			t.Fatalf("allow %v: modes %q", allow, ids)
		}
		_, err := c.Client.SetSessionMode(t.Context(), acp.SetSessionModeRequest{SessionId: s.SessionId, ModeId: modeBypass})
		if allow != (err == nil) {
			t.Fatalf("allow %v: set_mode bypass: %v", allow, err)
		}
		text, _ := slash(t, c, client, s.SessionId, "/mode")
		list := "Modes: default, accept-edits, plan\n"
		if allow {
			list = "Modes: default, accept-edits, plan, bypass\n"
		}
		if !strings.Contains(text+"\n", list) {
			t.Fatalf("allow %v: /mode %q, want %q", allow, text, list)
		}
		if !allow {
			if text, _ := slash(t, c, client, s.SessionId, "/mode bypass"); !strings.Contains(text, `Unknown mode "bypass". The modes are default, accept-edits, plan.`) {
				t.Fatalf("/mode bypass: %q", text)
			}
		}
	}
}

func newSessionIn(t *testing.T, c *acptest.Conn, cwd string) acp.NewSessionResponse {
	t.Helper()
	s, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: cwd, McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// modeCase is one call in a mode: whether it is asked, and whether it runs
// when the client says no.
type modeCase struct {
	name  string
	call  llmtest.Turn
	rules []permission.Spec
	asked bool
	limit bool
	from  string // the log's deciding source, when set
}

func runModeCases(t *testing.T, mode string, ss SessionSettings, cases []modeCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cwd := t.TempDir()
			writeFile(t, filepath.Join(cwd, "a.txt"), "x")
			writeFile(t, filepath.Join(cwd, "b.txt"), "y")
			s := llmtest.NewScript(c.call, llmtest.Text("done"))
			client := &acptest.Client{}
			ss := ss
			ss.Rules = c.rules
			conn, id, logs := modeAgent(t, s, client, ss, "", mode, cwd)
			if _, err := prompt(t, conn, id, "go"); err != nil {
				t.Fatal(err)
			}
			perms := client.Permissions()
			if (len(perms) == 1) != c.asked || len(perms) > 1 {
				t.Fatalf("%d questions, want asked %v; result %q", len(perms), c.asked, lastResult(t, s, 1))
			}
			if c.asked {
				marks, _ := perms[0].Meta[gobble].(map[string]any)
				if _, has := marks[keyLimit]; has != c.limit {
					t.Errorf("limit mark %v, want %v", has, c.limit)
				}
			}
			if !strings.Contains(logs.String(), "mode="+mode) || c.from != "" && !strings.Contains(logs.String(), `from="`+c.from+`"`) {
				t.Errorf("log %q, want the mode and from %q", logs.String(), c.from)
			}
		})
	}
}

// accept-edits allows write, edit, move, copy, mkdir and a shell mv
// unasked; a delete, bash, an outside path and a configuration path still
// ask; a rule the user wrote still decides (decision 3a).
func TestAcceptEdits(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "x.txt")
	runModeCases(t, modeAcceptEdits, SessionSettings{}, []modeCase{
		{name: "write", call: llmtest.ToolCall("w", "write", `{"path":"c.txt","content":"z"}`), from: sourceAcceptEdits},
		{name: "edit", call: llmtest.ToolCall("e", "edit", `{"path":"a.txt","edits":[{"oldText":"x","newText":"z"}]}`)},
		{name: "move", call: llmtest.ToolCall("m", "move", `{"from":"a.txt","to":"c.txt"}`)},
		{name: "copy", call: llmtest.ToolCall("c", "copy", `{"from":"a.txt","to":"c.txt"}`)},
		{name: "mkdir", call: llmtest.ToolCall("k", "mkdir", `{"path":"d"}`)},
		{name: "a shell mv", call: bashCall(t, "s", "mv a.txt c.txt")},
		{name: "a delete", call: llmtest.ToolCall("d", "delete", `{"path":"a.txt"}`), asked: true, limit: true},
		{name: "bash", call: bashCall(t, "b", "ls"), asked: true},
		{name: "an outside write", call: llmtest.ToolCall("o", "write", `{"path":`+jsonQuote(t, outside)+`,"content":"z"}`), asked: true},
		{name: "AGENTS.md", call: llmtest.ToolCall("g", "write", `{"path":"AGENTS.md","content":"z"}`), asked: true, limit: true},
		{name: "a rule that asks", call: llmtest.ToolCall("r", "write", `{"path":"c.txt","content":"z"}`),
			rules: []permission.Spec{rule(t, permission.ModeAsk, "write(/c.txt)", 3)}, asked: true},
	})
}

// bypass allows what a rule, a default or external_directory asks; a deny
// still refuses; gobble's limits still ask (decision 4a).
func TestBypass(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "x.txt")
	writeFile(t, outside, "42")
	runModeCases(t, modeBypass, SessionSettings{AllowBypass: true}, []modeCase{
		{name: "a default's ask", call: llmtest.ToolCall("w", "write", `{"path":"c.txt","content":"z"}`), from: sourceBypass},
		{name: "a rule's ask", call: llmtest.ToolCall("r", "read", `{"path":"a.txt"}`), rules: []permission.Spec{rule(t, permission.ModeAsk, "read", 3)}},
		{name: "an outside read", call: llmtest.ToolCall("o", "read", `{"path":`+jsonQuote(t, outside)+`}`)},
		{name: "a deny", call: llmtest.ToolCall("x", "write", `{"path":"c.txt","content":"z"}`),
			rules: []permission.Spec{rule(t, permission.ModeDeny, "write(/c.txt)", 3)}},
		{name: "a delete", call: llmtest.ToolCall("d", "delete", `{"path":"a.txt"}`), asked: true, limit: true},
		{name: "AGENTS.md", call: llmtest.ToolCall("g", "write", `{"path":"AGENTS.md","content":"z"}`), asked: true, limit: true},
	})
	// The deny refused, and nothing was asked.
	cwd := t.TempDir()
	s := llmtest.NewScript(llmtest.ToolCall("x", "write", `{"path":"c.txt","content":"z"}`), llmtest.Text("done"))
	ss := SessionSettings{AllowBypass: true, Rules: []permission.Spec{rule(t, permission.ModeDeny, "write(/c.txt)", 3), rule(t, permission.ModeDeny, "delete", 4)}}
	conn, id, logs := modeAgent(t, s, &acptest.Client{}, ss, "", modeBypass, cwd)
	if _, err := prompt(t, conn, id, "go"); err != nil {
		t.Fatal(err)
	}
	if got := lastResult(t, s, 1); !strings.Contains(got, `the permission rule "deny: write(/c.txt)"`) {
		t.Errorf("the deny: %q", got)
	}
	if slices.Contains(requestTools(t, s.Requests()[0]), "delete") {
		t.Error("a hidden tool was sent in bypass")
	}
	if _, err := os.Stat(filepath.Join(cwd, "c.txt")); err == nil {
		t.Error("the denied write ran")
	}
	if !strings.Contains(logs.String(), `effect=deny rule="deny: write(/c.txt)"`) {
		t.Errorf("log %q", logs.String())
	}
	// The repeated-call question is still asked.
	cwd = t.TempDir()
	writeFile(t, filepath.Join(cwd, "f.txt"), "x")
	s = llmtest.NewScript(readCall("a"), readCall("b"), readCall("c"), llmtest.Text("done"))
	client := &acptest.Client{Allow: true}
	conn, id, _ = modeAgent(t, s, client, SessionSettings{AllowBypass: true}, "", modeBypass, cwd)
	if _, err := prompt(t, conn, id, "go"); err != nil {
		t.Fatal(err)
	}
	if perms := client.Permissions(); len(perms) != 1 || perms[0].ToolCall.ToolCallId != "c" {
		t.Errorf("repeated call: %d questions", len(perms))
	}
}

// Plan mode's ceiling holds over an allow rule with bypass offered
// (decision 5).
func TestPlanCeiling(t *testing.T) {
	cwd := t.TempDir()
	s := llmtest.NewScript(llmtest.ToolCall("w", "write", `{"path":"c.txt","content":"z"}`), llmtest.Text("done"))
	ss := SessionSettings{AllowBypass: true, Rules: []permission.Spec{rule(t, permission.ModeAllow, "*", 3)}}
	conn, id, _ := modeAgent(t, s, &acptest.Client{Allow: true}, ss, "", modePlan, cwd)
	if _, err := prompt(t, conn, id, "go"); err != nil {
		t.Fatal(err)
	}
	if got := lastResult(t, s, 1); !strings.Contains(got, "Plan mode is read-only, so write did not run.") {
		t.Errorf("result %q", got)
	}
}

// --mode starts each new session in its mode, announced as the session's
// current mode and by current_mode_update; a mode the session does not
// offer starts it in default; a loaded session starts in default or
// --mode's, its earlier mode not saved (decisions 6 and 7).
func TestStartMode(t *testing.T) {
	for _, c := range []struct {
		override string
		allow    bool
		want     string
	}{{modeAcceptEdits, false, modeAcceptEdits}, {modeBypass, true, modeBypass}, {modeBypass, false, modeDefault}, {"", false, modeDefault}} {
		client := &acptest.Client{}
		a := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return llmtest.NewScript(), nil },
			Settings: func(string) SessionSettings { return SessionSettings{AllowBypass: c.allow} }, Override: Override{Mode: c.override},
			Logger: slog.New(slog.DiscardHandler)})
		conn := acptest.Connect(t, a, client)
		a.SetConnection(conn.Agent)
		initialize(t, conn.Client)
		s := newSessionIn(t, conn, t.TempDir())
		if string(s.Modes.CurrentModeId) != c.want {
			t.Errorf("%s, allow %v: started in %s, want %s", c.override, c.allow, s.Modes.CurrentModeId, c.want)
		}
		slash(t, conn, client, s.SessionId, "/mode")
		if got := modeUpdates(client); len(got) == 0 || got[0] != c.want {
			t.Errorf("%s, allow %v: mode updates %q", c.override, c.allow, got)
		}
	}
	dir, cwd := t.TempDir(), t.TempDir()
	c1, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Text("hi")), &acptest.Client{})
	id := newSession(t, c1, cwd, nil).SessionId
	if _, err := c1.Client.SetSessionMode(t.Context(), acp.SetSessionModeRequest{SessionId: id, ModeId: modePlan}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c1, id, "hello"); err != nil {
		t.Fatal(err)
	}
	if _, err := c1.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	c2, _ := sessionAgent(t, dir, llmtest.NewScript(), &acptest.Client{})
	resp, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Modes.CurrentModeId != modeDefault {
		t.Errorf("loaded in %s, want default", resp.Modes.CurrentModeId)
	}
}
