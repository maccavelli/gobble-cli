package acpserver

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/personality"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// appendPrompt is a client's system prompt in append mode, one text block
// per text.
func appendPrompt(texts ...string) *acp.SystemPrompt {
	p := &acp.SystemPrompt{Mode: acp.SystemPromptModeAppend, Content: []acp.ContentBlock{}}
	for _, s := range texts {
		p.Content = append(p.Content, acp.TextBlock(s))
	}
	return p
}

// clientSection is the client_prompt section as the system message holds
// it: the texts joined by a blank line, tagged.
func clientSection(texts ...string) string {
	return "<client_prompt>\n" + strings.Join(texts, "\n\n") + "\n</client_prompt>"
}

// withClient is the system message without, with the client_prompt section
// in its place: after the addendum, before project_context (0054-MADR D7).
func withClient(t *testing.T, without, section string) string {
	t.Helper()
	at := strings.Index(without, "<project_context>")
	if at < 0 {
		t.Fatalf("the system message has no project_context:\n%s", without)
	}
	return without[:at] + section + "\n\n" + without[at:]
}

// sessionWith opens a session in cwd whose session/new carries sp.
func sessionWith(t *testing.T, c *acptest.Conn, cwd string, sp *acp.SystemPrompt) acp.SessionId {
	t.Helper()
	resp, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: cwd, McpServers: []acp.McpServer{}, SystemPrompt: sp})
	if err != nil {
		t.Fatal(err)
	}
	return resp.SessionId
}

func closeSession(t *testing.T, c *acptest.Conn, id acp.SessionId) {
	t.Helper()
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
}

// systemPrompts counts entries' gobble.system_prompt entries.
func systemPrompts(entries []session.Entry) int {
	n := 0
	for _, e := range entries {
		if e.Type == session.TypeCustom && e.CustomType == session.CustomSystemPrompt {
			n++
		}
	}
	return n
}

// listed are session/list's ids, sorted.
func listed(t *testing.T, c *acptest.Conn) []string {
	t.Helper()
	resp, err := c.Client.ListSessions(t.Context(), acp.ListSessionsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, s := range resp.Sessions {
		out = append(out, string(s.SessionId))
	}
	slices.Sort(out)
	return out
}

// oneTurnSession writes a session of one turn, with no system prompt, to
// store through its own agent, and closes it.
func oneTurnSession(t *testing.T, store session.Store, cwd string) acp.SessionId {
	t.Helper()
	var logs bytes.Buffer
	c := promptAgent(t, "", llmtest.NewScript(llmtest.Text("stored")), PromptOptions{}, &logs, func(o *Options) { o.Store = store })
	id := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, id, "store me"); err != nil {
		t.Fatal(err)
	}
	closeSession(t, c, id)
	return id
}

// sessionRequest sends a session/new, session/load or session/resume
// carrying sp as a raw frame, its params JSON the client neither validates
// nor gates; new's id is _meta.gobble.sessionId.
func sessionRequest(t *testing.T, c *acptest.Conn, method string, id acp.SessionId, cwd string, sp *acp.SystemPrompt) error {
	t.Helper()
	var req any
	switch method {
	case acp.AgentMethodSessionNew:
		req = acp.NewSessionRequest{Cwd: cwd, McpServers: []acp.McpServer{}, SystemPrompt: sp,
			Meta: map[string]any{gobble: map[string]any{"sessionId": string(id)}}}
	case acp.AgentMethodSessionLoad:
		req = acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}, SystemPrompt: sp}
	case acp.AgentMethodSessionResume:
		req = acp.ResumeSessionRequest{SessionId: id, Cwd: cwd, SystemPrompt: sp}
	default:
		t.Fatalf("no session request %s", method)
	}
	params, err := jsonx.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	_, err = acp.SendRequest[jsontext.Value](c.Client.Connection(), t.Context(), method, jsontext.Value(params))
	return err
}

var sessionMethods = []string{acp.AgentMethodSessionNew, acp.AgentMethodSessionLoad, acp.AgentMethodSessionResume}

// notOpened asserts that a refused request on session id left nothing:
// store holds no session new would have made, and a stored one is closed,
// unchanged, and opens again.
func notOpened(t *testing.T, c *acptest.Conn, store session.Store, method string, id acp.SessionId, before []session.Entry) {
	t.Helper()
	if method == acp.AgentMethodSessionNew {
		if _, _, _, err := store.Read(t.Context(), session.ID(id)); !errors.Is(err, session.ErrNotFound) {
			t.Errorf("%s: the store holds session %s (%v), want none", method, id, err)
		}
		return
	}
	_, err := prompt(t, c, id, "still there?")
	if code, reason := errText(t, err); code != -32602 || reason != "unknown session" {
		t.Errorf("%s: a prompt to %s: %d %q, want the session not open", method, id, code, reason)
	}
	if _, entries, _, err := store.Read(t.Context(), session.ID(id)); err != nil || len(entries) != len(before) {
		t.Errorf("%s: %s has %d entries (%v), want %d", method, id, len(entries), err, len(before))
	}
}

// A session/new whose append prompt has two text blocks gives the first
// turn's system message a client_prompt section of both texts, joined by a
// blank line, after the addendum and before project_context, and every
// section a session without it has (0054-MADR D7).
func TestSystemPromptReachesTheModel(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(cwd, "AGENTS.md"), "Project rule.")
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"))
	var logs bytes.Buffer
	c := promptAgent(t, dir, model, PromptOptions{Append: "Appended."}, &logs)
	with := sessionWith(t, c, cwd, appendPrompt("Review first.", "Then this."))
	without := newSession(t, c, cwd, nil).SessionId
	for _, id := range []acp.SessionId{with, without} {
		if _, err := prompt(t, c, id, "go"); err != nil {
			t.Fatal(err)
		}
	}
	reqs := model.Requests()
	section := clientSection("Review first.", "Then this.")
	llmtest.AssertSystem(t, reqs[0], section, "<addendum>\nAppended.\n</addendum>", "Project rule.")
	if got, want := systemOf(reqs[0]), withClient(t, systemOf(reqs[1]), section); got != want {
		t.Fatalf("system message:\n%s\n--- want ---\n%s", got, want)
	}
}

// A client's append keeps gobble's own: a SYSTEM.md, an APPEND_SYSTEM.md, a
// context file and a personality stay, and without a SYSTEM.md the tools
// and rules sections are as they are without the client's prompt
// (0054-MADR D7).
func TestSystemPromptKeepsGobblesOwn(t *testing.T) {
	dry, err := personality.New(personality.Options{Name: personality.Dry})
	if err != nil {
		t.Fatal(err)
	}
	for _, system := range []string{"From SYSTEM.md.", ""} {
		dir, cwd, cfg := t.TempDir(), t.TempDir(), t.TempDir()
		if system != "" {
			writeFile(t, filepath.Join(cfg, "SYSTEM.md"), system)
		}
		writeFile(t, filepath.Join(cfg, "APPEND_SYSTEM.md"), "From APPEND_SYSTEM.md.")
		writeFile(t, filepath.Join(cwd, "AGENTS.md"), "Project rule.")
		model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"))
		var logs bytes.Buffer
		c := promptAgent(t, dir, model, PromptOptions{ConfigDir: cfg}, &logs, func(o *Options) {
			o.Settings = func(string) SessionSettings { return SessionSettings{Personality: personality.Dry} }
		})
		with := sessionWith(t, c, cwd, appendPrompt("Client rule."))
		without := newSession(t, c, cwd, nil).SessionId
		for _, id := range []acp.SessionId{with, without} {
			if _, err := prompt(t, c, id, "go"); err != nil {
				t.Fatal(err)
			}
		}
		reqs := model.Requests()
		own := []string{"<addendum>\nFrom APPEND_SYSTEM.md.\n</addendum>", "Project rule.",
			"<personality>\n" + dry.Brief() + "\n</personality>", "<personality_rules>\n" + dry.Rules() + "\n</personality_rules>"}
		if system != "" {
			if sys := systemOf(reqs[0]); !strings.HasPrefix(sys, system+"\n\n") {
				t.Errorf("SYSTEM.md is not the preamble:\n%s", sys)
			}
		} else {
			own = append(own, "<tools>\n", "<rules>\n")
		}
		llmtest.AssertSystem(t, reqs[0], append(own, clientSection("Client rule."))...)
		if got, want := systemOf(reqs[0]), withClient(t, systemOf(reqs[1]), clientSection("Client rule.")); got != want {
			t.Errorf("SYSTEM.md %q: system message:\n%s\n--- want ---\n%s", system, got, want)
		}
	}
}

// Override is not offered (0054-MADR Q2 (a)): an override prompt on
// session/new, session/load and session/resume is refused -32602, naming
// the capability, and nothing is created, loaded or resumed.
func TestSystemPromptOverrideRefused(t *testing.T) {
	store, cwd := session.NewMemoryStore(), t.TempDir()
	stored := oneTurnSession(t, store, cwd)
	_, before, _, err := store.Read(t.Context(), session.ID(stored))
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	c := promptAgent(t, "", llmtest.NewScript(), PromptOptions{}, &logs, func(o *Options) { o.Store = store })
	list := listed(t, c)
	override := &acp.SystemPrompt{Mode: acp.SystemPromptModeOverride, Content: []acp.ContentBlock{acp.TextBlock("Only this.")}}
	for _, method := range sessionMethods {
		id := stored
		if method == acp.AgentMethodSessionNew {
			id = "overridden"
		}
		err := sessionRequest(t, c, method, id, cwd, override)
		re, ok := errors.AsType[*acp.RequestError](err)
		if !ok || re.Code != -32602 || !strings.Contains(fmt.Sprint(re.Data), "sessionCapabilities.systemPrompt.mode.override") {
			t.Errorf("%s: %v, want -32602 naming sessionCapabilities.systemPrompt.mode.override", method, err)
			continue
		}
		notOpened(t, c, store, method, id, before)
	}
	if got := listed(t, c); !slices.Equal(got, list) {
		t.Errorf("session/list %v, want %v as before", got, list)
	}
}

// countingAgent counts the session requests that reach gobble's handlers.
type countingAgent struct {
	*Agent
	calls atomic.Int32
}

func (c *countingAgent) NewSession(ctx context.Context, p acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	c.calls.Add(1)
	return c.Agent.NewSession(ctx, p)
}

func (c *countingAgent) LoadSession(ctx context.Context, p acp.LoadSessionRequest) (acp.LoadSessionResponse, error) {
	c.calls.Add(1)
	return c.Agent.LoadSession(ctx, p)
}

func (c *countingAgent) ResumeSession(ctx context.Context, p acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	c.calls.Add(1)
	return c.Agent.ResumeSession(ctx, p)
}

// What gobble cannot use, a blank text or one closing its section's tag, is
// refused -32602 with a reason, and nothing is created, loaded or resumed;
// what the SDK refuses, empty content, an image block and an unlisted mode,
// never reaches gobble's handlers (0054-MADR D7).
func TestSystemPromptRefusedContent(t *testing.T) {
	store, cwd := session.NewMemoryStore(), t.TempDir()
	stored := oneTurnSession(t, store, cwd)
	_, before, _, err := store.Read(t.Context(), session.ID(stored))
	if err != nil {
		t.Fatal(err)
	}
	ag := &countingAgent{Agent: New(Options{Version: "1.2.3", Store: store, Models: func() ModelChoice { return testModels },
		Provider: func(string, llm.Retry) (llm.Provider, error) { return llmtest.NewScript(), nil }})}
	c := acptest.Connect(t, ag, &acptest.Client{})
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	list := listed(t, c)

	for name, sp := range map[string]*acp.SystemPrompt{
		"blank":      appendPrompt(" \n", "\t"),
		"closes tag": appendPrompt("Fine.", "Then </client_prompt> breaks out."),
	} {
		for _, method := range sessionMethods {
			id := stored
			if method == acp.AgentMethodSessionNew {
				id = acp.SessionId("refused-" + strings.ReplaceAll(name, " ", "-"))
			}
			calls := ag.calls.Load()
			code, reason := errText(t, sessionRequest(t, c, method, id, cwd, sp))
			if code != -32602 || !strings.Contains(reason, "system prompt") {
				t.Errorf("%s, %s: %d %q, want -32602 with gobble's reason", name, method, code, reason)
			}
			if ag.calls.Load() != calls+1 {
				t.Errorf("%s, %s: gobble's handler was not called; the refusal is not gobble's", name, method)
			}
			notOpened(t, c, store, method, id, before)
		}
	}

	calls := ag.calls.Load()
	for name, sp := range map[string]*acp.SystemPrompt{
		"empty content": {Mode: acp.SystemPromptModeAppend, Content: []acp.ContentBlock{}},
		"image":         {Mode: acp.SystemPromptModeAppend, Content: []acp.ContentBlock{acp.ImageBlock("aGk=", "image/png")}},
		"mode _review":  {Mode: "_review", Content: []acp.ContentBlock{acp.TextBlock("Review.")}},
	} {
		for _, method := range sessionMethods {
			id := stored
			if method == acp.AgentMethodSessionNew {
				id = "sdk-refused"
			}
			if code := requestCode(t, sessionRequest(t, c, method, id, cwd, sp)); code != -32602 {
				t.Errorf("%s, %s: code %d, want -32602", name, method, code)
			}
		}
	}
	if n := ag.calls.Load() - calls; n != 0 {
		t.Errorf("gobble's handlers were called %d times for what the SDK refuses, want 0", n)
	}
	if got := listed(t, c); !slices.Equal(got, list) {
		t.Errorf("session/list %v, want %v as before", got, list)
	}
}

// A session created with a prompt and one turn, closed, then loaded with no
// systemPrompt, sends it at its next turn, and writes no new
// gobble.system_prompt entry and no gobble.prompt patch; so does a resume,
// and a resume with the same prompt (0054-MADR D7).
func TestSystemPromptPersists(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"), llmtest.Text("c"), llmtest.Text("d"))
	var logs bytes.Buffer
	c := promptAgent(t, dir, model, PromptOptions{}, &logs)
	sp := appendPrompt("Keep this.")
	id := sessionWith(t, c, cwd, sp)
	if _, err := prompt(t, c, id, "one"); err != nil {
		t.Fatal(err)
	}
	closeSession(t, c, id)
	for i, reopen := range []func(*acptest.Conn) error{
		func(c *acptest.Conn) error {
			_, err := c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}})
			return err
		},
		func(c *acptest.Conn) error {
			_, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd})
			return err
		},
		func(c *acptest.Conn) error {
			_, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd, SystemPrompt: sp})
			return err
		},
	} {
		c := promptAgent(t, dir, model, PromptOptions{}, &logs)
		if err := reopen(c); err != nil {
			t.Fatal(err)
		}
		if _, err := prompt(t, c, id, "again"); err != nil {
			t.Fatal(err)
		}
		llmtest.AssertSystem(t, model.Requests()[i+1], clientSection("Keep this."))
		if got := strings.Join(promptEntries(t, dir, string(id)), ","); got != "baseline" {
			t.Errorf("reopening %d: prompt entries %s, want the first turn's baseline alone", i, got)
		}
		if n := systemPrompts(savedEntries(t, dir, string(id))); n != 1 {
			t.Errorf("reopening %d: %d gobble.system_prompt entries, want session/new's alone", i, n)
		}
		closeSession(t, c, id)
	}
}

// A prompt given at session/new is the session's before any turn: closed
// before its first turn and loaded with none, the session's first baseline
// holds it. The store is the in-memory one, which keeps a session with no
// message; a JSONL file is written with the first (0054-MADR D7).
func TestSystemPromptBeforeFirstTurn(t *testing.T) {
	store, cwd := session.NewMemoryStore(), t.TempDir()
	model := llmtest.NewScript(llmtest.Text("a"))
	var logs bytes.Buffer
	c := promptAgent(t, "", model, PromptOptions{}, &logs, func(o *Options) { o.Store = store })
	id := sessionWith(t, c, cwd, appendPrompt("From the start."))
	closeSession(t, c, id)
	c2 := promptAgent(t, "", model, PromptOptions{}, &logs, func(o *Options) { o.Store = store })
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c2, id, "first"); err != nil {
		t.Fatal(err)
	}
	llmtest.AssertSystem(t, model.Requests()[0], clientSection("From the start."))
	_, entries, _, err := store.Read(t.Context(), session.ID(id))
	if err != nil {
		t.Fatal(err)
	}
	base, _, ok := promptState(entries)
	if !ok || !slices.ContainsFunc(base, func(s sysprompt.Section) bool { return s.Text == clientSection("From the start.") }) {
		t.Fatalf("the first baseline %+v lacks the client's prompt", base)
	}
}

// session/load with a different prompt writes one new entry; the next turn
// carries a gobble.prompt patch for client_prompt and its update message;
// after /compact the new baseline's system message holds the new text
// (0054-MADR D7).
func TestSystemPromptChangedOnLoad(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	model := llmtest.NewScript(llmtest.Text(strings.Repeat("x", 4000)), llmtest.Text("y"), llmtest.Text("## Goal\nS"+strings.Repeat(" detail", 80)), llmtest.Text("z"))
	var logs bytes.Buffer
	c := promptAgent(t, dir, model, PromptOptions{}, &logs)
	id := sessionWith(t, c, cwd, appendPrompt("Old way."))
	if _, err := prompt(t, c, id, strings.Repeat("a ", 2000)); err != nil {
		t.Fatal(err)
	}
	closeSession(t, c, id)
	c2 := promptAgent(t, dir, model, PromptOptions{}, &logs, func(o *Options) { o.Compaction = compaction.Settings{KeepRecentTokens: 50} })
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{},
		SystemPrompt: appendPrompt("New way.")}); err != nil {
		t.Fatal(err)
	}
	if n := systemPrompts(savedEntries(t, dir, string(id))); n != 2 {
		t.Fatalf("%d gobble.system_prompt entries after the load, want 2", n)
	}
	if _, err := prompt(t, c2, id, "second"); err != nil {
		t.Fatal(err)
	}
	req := model.Requests()[1]
	if sys := systemOf(req); !strings.Contains(sys, clientSection("Old way.")) {
		t.Fatalf("the system message changed before a new baseline:\n%s", sys)
	}
	users := userTextsOf(req)
	if n := len(users); n < 2 || users[n-2] != `Updated system prompt section "client_prompt":`+"\n\n"+clientSection("New way.") || users[n-1] != "second" {
		t.Fatalf("user messages %q; want the client_prompt update before the prompt", users)
	}
	if got := strings.Join(promptEntries(t, dir, string(id)), ","); got != "baseline,patch" {
		t.Fatalf("prompt entries %s, want baseline,patch", got)
	}
	if _, err := ext(t, c2, "_gobble/compact", map[string]any{"sessionId": id}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c2, id, "after"); err != nil {
		t.Fatal(err)
	}
	reqs := model.Requests()
	if sys := systemOf(reqs[len(reqs)-1]); !strings.Contains(sys, clientSection("New way.")) || strings.Contains(sys, "Old way.") {
		t.Fatalf("system message after the compaction:\n%s", sys)
	}
}

// noClientSystem is the system message a session with no client prompt
// sent before 0054-PLAN P2, for TestNoSystemPromptUnchanged's options.
const noClientSystem = "You are gobble, a coding agent. You help the user with software engineering tasks in their project.\n\n" +
	"<tools>\n- Read a file before you edit it.\n- tool_search can find more tools for: permissions.\n\n" +
	"Each request describes the tools you can call; you may also have tools from MCP servers and the project.\n</tools>\n\n" +
	"<rules>\n- Use bash for file operations like ls, rg, find\n" +
	"- A call to a tool that changes something may be declined by the user; then do not retry it, and say what you would have done.\n" +
	"- Commands run on linux.\n- Be concise in your responses\n- Show file paths clearly when working with files\n- Answer in Markdown.\n</rules>\n\n" +
	"<cwd>\n/work\n</cwd>"

// A session with no systemPrompt sends the system message it sent before
// client prompts, byte for byte (0054-MADR D7).
func TestNoSystemPromptUnchanged(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("hi"))
	var logs bytes.Buffer
	c := promptAgent(t, t.TempDir(), model, PromptOptions{GOOS: "linux", cwd: "/work"}, &logs, func(o *Options) {
		o.Tools = []tool.Tool{builtin.Read(), builtin.Write(), builtin.Edit(), builtin.Bash()}
	})
	id := newSession(t, c, acptest.Abs("/work"), nil).SessionId
	if _, err := prompt(t, c, id, "hello"); err != nil {
		t.Fatal(err)
	}
	if got := systemOf(model.Requests()[0]); got != noClientSystem {
		t.Fatalf("system message:\n%s\n--- want ---\n%s", got, noClientSystem)
	}
}
