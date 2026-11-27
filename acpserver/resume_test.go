package acpserver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// bodyRecorder is an HTTP transport that keeps each request's path and
// body and answers 400, so a provider built over it renders a request and
// sends it nowhere (0037-MADR D42).
type bodyRecorder struct {
	mu     sync.Mutex
	paths  []string
	bodies [][]byte
}

func (r *bodyRecorder) RoundTrip(q *http.Request) (*http.Response, error) {
	b, err := io.ReadAll(q.Body)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.paths = append(r.paths, q.URL.Path)
	r.bodies = append(r.bodies, b)
	r.mu.Unlock()
	return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
		Body:    io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"invalid_request_error","message":"recorded, not sent"}}`)),
		Request: q}, nil
}

// wireCase is a provider and a model id whose request body has one of the
// shapes the providers' wires come in, and the end of the path that shape
// is sent to. The path is asserted, so a model id that a provider's route
// table later sends to another shape fails the test instead of covering a
// shape twice.
type wireCase struct{ shape, provider, model, pathEnd string }

// resumeWires is one wire per body shape of the 16 wires and routes
// measured by 0037-PLAN-resume-prefix-and-failed-turn-history.md P1. Each
// model id is the one least tied to a catalog or a route table:
//   - claude and gemini: a model the provider serves itself, with one wire
//     whatever the id;
//   - openai: the same, for the Responses wire;
//   - ollama: a local engine with no catalog, so the id is free text, and
//     the one chat-completions wire that no catalog, route table or prefix
//     rule reaches;
//   - opencode-zen: the only provider with the Google route, which its
//     embedded route table gives to a Gemini model id; the path check
//     above is what holds that table to this shape.
var resumeWires = []wireCase{
	{"Messages", "claude", "claude-haiku-5-5", "/v1/messages"},
	{"Responses", "openai", "gpt-6-luna", "/v1/responses"},
	{"chat-completions", "ollama", "llama3", "/v1/chat/completions"},
	{"Gemini interactions", "gemini", "gemini-3.8-flash", "/v1beta/interactions"},
	{"OpenCode Google route", "opencode-zen", "gemini-3.5-flash", ":streamGenerateContent"},
}

// wireBodies renders each request through w's real wire and with no retry,
// and returns the bodies it sent.
func wireBodies(t *testing.T, w wireCase, reqs []*llm.Request) [][]byte {
	t.Helper()
	tr := &bodyRecorder{}
	p, err := provider.New(w.provider, provider.Options{APIKey: "not-a-real-key", SessionID: "resume-prefix", HTTPClient: &http.Client{Transport: tr},
		Retry: &llm.Retry{Off: true}})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reqs {
		r := *r
		r.Model = w.model
		msgs := make([]llm.Message, len(r.Messages))
		for i, m := range r.Messages {
			if m.Origin.Provider != "" {
				m.Origin.Provider, m.Origin.Model = w.provider, w.model
			}
			msgs[i] = m
		}
		r.Messages = msgs
		var failure error
		for _, err := range p.Stream(t.Context(), &r) {
			failure = err
		}
		// The recorder's 400 is the only way a request ends here.
		if ae, ok := errors.AsType[*llm.APIError](failure); !ok || ae.Status != http.StatusBadRequest {
			t.Fatalf("%s: the rendered request ended with %v, want the recorder's 400", w.shape, failure)
		}
	}
	if len(tr.bodies) != len(reqs) {
		t.Fatalf("%s: %d requests rendered %d bodies", w.shape, len(reqs), len(tr.bodies))
	}
	for _, path := range tr.paths {
		if !strings.HasSuffix(path, w.pathEnd) {
			t.Fatalf("%s (%s, %s): a request went to %s, want a path ending %s", w.shape, w.provider, w.model, path, w.pathEnd)
		}
	}
	return tr.bodies
}

// withoutMarkers is v with every cache_control member removed: the moving
// marker is not part of the prefix a cache reads.
func withoutMarkers(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, y := range x {
			if k != "cache_control" {
				out[k] = withoutMarkers(y)
			}
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, y := range x {
			out[i] = withoutMarkers(y)
		}
		return out
	}
	return v
}

// sequenceKeys are the members that hold a wire body's message sequence: the
// Messages, chat-completions and Gemini interactions wires' "messages" or
// "input", the Responses wire's "input", and the Google route's "contents".
var sequenceKeys = []string{"messages", "input", "contents"}

// wireShape is a wire body split into its message items, each as the bytes
// it encodes to with its markers removed, and everything else in the body,
// the tools and the system prompt among it, likewise.
func wireShape(t *testing.T, body []byte) (items []string, head string) {
	t.Helper()
	var m map[string]any
	if err := jsonx.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	var list []any
	for _, k := range sequenceKeys {
		if l, ok := m[k].([]any); ok {
			list = l
			delete(m, k)
			break
		}
	}
	if list == nil {
		t.Fatalf("no message sequence in the body %.200s", body)
	}
	for _, it := range list {
		b, err := jsonx.Marshal(withoutMarkers(it))
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, string(b))
	}
	h, err := jsonx.Marshal(withoutMarkers(m))
	if err != nil {
		t.Fatal(err)
	}
	return items, string(h)
}

// A session's requests keep every earlier request of it as a prefix, live
// and after a close and a load, with the cache markers removed: the cache
// reads a request's prefix, and a resumed session that rewrites it pays
// full price once (0037-MADR D42). Four scripted sessions, each with a
// tool call, two replies, a close, a load into a second Agent over the same
// store and one more prompt, are rendered through one wire of each body
// shape (resumeWires):
//   - mention: the prompt's @path adds a files block, which the live
//     message once held as a block of its own and the file holds joined;
//   - two text blocks: the same join, with no mention;
//   - spaced arguments: the model's spacing in the call's arguments, which
//     the file drops;
//   - empty arguments: a call whose argument text is empty, which the file
//     gives back as "{}".
func TestResumeKeepsPrefix(t *testing.T) {
	cases := []struct {
		name   string
		prompt []acp.ContentBlock
		args   string
	}{
		{"mention", []acp.ContentBlock{acp.TextBlock("look at @notes.md please")}, `{"path":"notes.md"}`},
		{"two text blocks", []acp.ContentBlock{acp.TextBlock("first block"), acp.TextBlock("second block")}, `{"path":"notes.md"}`},
		{"spaced arguments", []acp.ContentBlock{acp.TextBlock("read notes.md")}, `{"path": "notes.md"}`},
		{"empty arguments", []acp.ContentBlock{acp.TextBlock("read notes.md")}, ``},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reqs := resumedRequests(t, tc.prompt, tc.args)
			for _, w := range resumeWires {
				checkPrefix(t, w.shape, wireBodies(t, w, reqs))
			}
		})
	}
}

// resumedRequests runs the scripted session of TestResumeKeepsPrefix with
// the given first prompt and the arguments of its one read call, and
// returns the model requests it sent: three before the close, one after the
// load.
func resumedRequests(t *testing.T, first []acp.ContentBlock, args string) []*llm.Request {
	t.Helper()
	dir, cwd := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "notes.md"), []byte("forty-two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	before := llmtest.NewScript(llmtest.ToolCall("toolu_1", "read", args), llmtest.Text("it says forty-two"), llmtest.Text("second answer"))
	c := promptAgent(t, dir, before, PromptOptions{}, &logs)
	sid := newSession(t, c, cwd, nil).SessionId
	if _, err := c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: sid, Prompt: first}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, sid, "and again"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: sid}); err != nil {
		t.Fatal(err)
	}
	after := llmtest.NewScript(llmtest.Text("third answer"))
	c2 := promptAgent(t, dir, after, PromptOptions{}, &logs)
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: sid, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c2, sid, "third"); err != nil {
		t.Fatal(err)
	}
	reqs := append(before.Requests(), after.Requests()...)
	if len(reqs) != 4 {
		t.Fatalf("%d model requests, want 4", len(reqs))
	}
	return reqs
}

// checkPrefix fails for each earlier request whose tools or system differ
// from the last request's, or whose message items are not a prefix of the
// last request's, naming the first item that is not.
func checkPrefix(t *testing.T, id string, bodies [][]byte) {
	t.Helper()
	last, lastHead := wireShape(t, bodies[len(bodies)-1])
	for i, b := range bodies[:len(bodies)-1] {
		items, head := wireShape(t, b)
		if head != lastHead {
			t.Errorf("%s: request %d's tools or system differ from the post-reload request's", id, i)
		}
		for j, it := range items {
			if j < len(last) && it == last[j] {
				continue
			}
			got := "<missing>"
			if j < len(last) {
				got = last[j]
			}
			t.Errorf("%s: request %d's item %d of %d is not the post-reload request's:\n  before: %.160s\n  after:  %.160s", id, i, j, len(items), it, got)
			break
		}
	}
}

// roleSeq is req's messages as "role:type", the type of each message's first
// content block, as a failure message reads them.
func roleSeq(req *llm.Request) []string {
	out := make([]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		kind := "empty"
		if len(m.Content) > 0 {
			kind = m.Content[0].Type
		}
		out = append(out, string(m.Role)+":"+kind)
	}
	return out
}

// sameRequest fails unless live, the request a session sent after a failed
// turn, is the request reloaded, which a session loaded from its file sent
// for the same prompt: the same roles and block types, and the same bytes on
// the claude and openai wires with the cache markers removed (0037-MADR D15).
func sameRequest(t *testing.T, live, reloaded *llm.Request) {
	t.Helper()
	if lr, rr := roleSeq(live), roleSeq(reloaded); !slices.Equal(lr, rr) {
		t.Errorf("after a failure, the live request is not the reloaded one:\n  live:     %v\n  reloaded: %v", lr, rr)
		return
	}
	for _, w := range resumeWires[:2] { // claude and openai
		lb, rb := wireBodies(t, w, []*llm.Request{live}), wireBodies(t, w, []*llm.Request{reloaded})
		// Only the messages: the two sessions are two sessions, so their
		// system prompts name their own directories and ids.
		litems, _ := wireShape(t, lb[0])
		ritems, _ := wireShape(t, rb[0])
		if !slices.Equal(litems, ritems) {
			t.Errorf("%s: after a failure, the live request's wire body is not the reloaded one's:\n  live:     %.300s\n  reloaded: %.300s", w.shape, strings.Join(litems, " "), strings.Join(ritems, " "))
		}
	}
}

// failedTurnRequests runs a turn that fails and then a second prompt, and
// returns the second prompt's request: in one process when reload is false,
// and with a close and a load into a second Agent over the same store before
// the second prompt when it is true. first is the first prompt's script, and
// change, when set, changes the first Agent's options, such as to wrap its
// store; wantErr says whether the first prompt's error is the failure the
// test scripted.
func failedTurnRequests(t *testing.T, reload bool, first *llmtest.Script, change func(*Options), wantErr func(error) bool) *llm.Request {
	t.Helper()
	dir, cwd := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "notes.md"), []byte("forty-two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	var changes []func(*Options)
	if change != nil {
		changes = append(changes, func(o *Options) { change(o) })
	}
	c := promptAgent(t, dir, first, PromptOptions{}, &logs, changes...)
	sid := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, sid, "read it"); err == nil || !wantErr(err) {
		t.Fatalf("the failing turn answered %v", err)
	}
	second := first
	if reload {
		if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: sid}); err != nil {
			t.Fatal(err)
		}
		second = llmtest.NewScript(llmtest.Text("after"))
		c = promptAgent(t, dir, second, PromptOptions{}, &logs)
		if _, err := c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: sid, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := prompt(t, c, sid, "again"); err != nil {
		t.Fatal(err)
	}
	reqs := second.Requests()
	return reqs[len(reqs)-1]
}

// After a failed model call, the live session's next request is the request
// a reload of its file sends: the failed turn's prompt, and the tool call
// and result that ran before the failure, are in it, as the file holds them
// (0037-MADR D15).
func TestFailedTurnHistoryIsTheFiles(t *testing.T) {
	script := func() *llmtest.Script {
		return llmtest.NewScript(
			llmtest.ToolCall("r1", "read", `{"path":"notes.md"}`),
			llmtest.Turn{Err: &llm.APIError{Status: http.StatusBadRequest, Err: errors.New("scripted, not retried")}},
			llmtest.Text("after"))
	}
	isAPI := func(err error) bool { return err != nil }
	live := failedTurnRequests(t, false, script(), nil, isAPI)
	reloaded := failedTurnRequests(t, true, script(), nil, isAPI)
	sameRequest(t, live, reloaded)
}

// failAt is a store that fails the append of the first entry that match
// accepts, and keeps every other append as the store it wraps does.
type failAt struct {
	session.Store
	match func(session.Entry) bool
	done  *atomic.Bool
}

type failAtLog struct {
	session.Log
	f failAt
}

func (f failAt) Create(ctx context.Context, h session.Header) (session.Log, error) {
	l, err := f.Store.Create(ctx, h)
	return failAtLog{Log: l, f: f}, err
}

func (f failAt) Open(ctx context.Context, id session.ID) (session.Log, error) {
	l, err := f.Store.Open(ctx, id)
	return failAtLog{Log: l, f: f}, err
}

func (l failAtLog) Append(ctx context.Context, e session.Entry) error {
	if l.f.match(e) && l.f.done.CompareAndSwap(false, true) {
		return errors.New("scripted append failure")
	}
	return l.Log.Append(ctx, e)
}

// After a failed append inside a turn, the live session's next request is
// the request a reload of its file sends: the assistant's tool call, which
// was written before its result failed to be, is in it (0037-MADR D15).
func TestFailedAppendHistoryIsTheFiles(t *testing.T) {
	run := func(reload bool) *llm.Request {
		var done atomic.Bool
		store := func(o *Options) {
			o.Store = failAt{Store: o.Store, done: &done, match: func(e session.Entry) bool {
				return e.Message != nil && e.Message.Role == session.RoleToolResult
			}}
		}
		first := llmtest.NewScript(llmtest.ToolCall("r1", "read", `{"path":"notes.md"}`), llmtest.Text("after"))
		return failedTurnRequests(t, reload, first, store, func(err error) bool {
			return strings.Contains(err.Error(), "scripted append failure")
		})
	}
	live, reloaded := run(false), run(true)
	if roles := roleSeq(live); !slices.Contains(roles, "assistant:tool_call") {
		t.Errorf("the live request %v lacks the assistant's tool call that the file holds", roles)
	}
	sameRequest(t, live, reloaded)
}

// preTurnAgent is an Agent as promptAgent builds one, with the read tool and
// a config directory of its own, and change applied to its options. It
// returns the Agent too, so a test can read the history it holds.
func preTurnAgent(t *testing.T, dir, cfg string, p llm.Provider, change func(*Options)) (*Agent, *acptest.Conn) {
	t.Helper()
	var logs bytes.Buffer
	opts := Options{Version: "1.2.3", Provider: func(*provider.ModelData, string, llm.Retry) (llm.Provider, error) { return p, nil },
		Models: func() ModelChoice { return testModels }, Store: jsonl.New(dir, jsonl.Options{}),
		Tools: []tool.Tool{builtin.Read()}, Prompt: PromptOptions{ConfigDir: cfg},
		Logger: slog.New(slog.NewTextHandler(&logs, nil))}
	if change != nil {
		change(&opts)
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
	return ag, c
}

// historyOf is the history session id holds in memory, one string for each
// message, with its role and each block's type, text, call id, tool and
// arguments: what the model is sent, without what the file adds to it (usage,
// cost, stamps).
func historyOf(a *Agent, id acp.SessionId) []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []string
	for _, m := range a.sessions[id].history {
		var parts []string
		for _, c := range m.Content {
			parts = append(parts, strings.Join([]string{c.Type, c.Text, c.ToolCallID, c.ToolName, c.ToolArgs}, "|"))
		}
		out = append(out, string(m.Role)+"{"+strings.Join(parts, ";")+"}")
	}
	return out
}

// preTurnFailure fails the first append that match accepts, once the session
// is armed, in a prompt that fails before its turn starts, and returns the
// history the live session holds after it and the history a fresh load of
// its file holds. A session whose file was never created holds none. When
// warm, a first prompt succeeds and AGENTS.md changes before the failing
// one, so that prompt writes a system-prompt patch; resources adds a
// deferred tool of a session-start family, so that the first prompt loads it.
func preTurnFailure(t *testing.T, warm, resources bool, match func(session.Entry) bool) (live, file []string) {
	t.Helper()
	dir, cwd, cfg := t.TempDir(), t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(cwd, "AGENTS.md"), "First rule.")
	var armed, fired atomic.Bool
	script := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"))
	ag, c := preTurnAgent(t, dir, cfg, script, func(o *Options) {
		o.Store = failAt{Store: o.Store, done: &fired, match: func(e session.Entry) bool { return armed.Load() && match(e) }}
		if resources {
			o.Tools = append(o.Tools, tool.New("resource_stub", "A deferred tool of a session-start family.",
				func(context.Context, struct{}, tool.Env) (string, error) { return "ran", nil },
				tool.WithExposure(tool.ExposureDeferred), tool.WithFamily(builtin.ResourceFamily),
				tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true})))
		}
	})
	sid := newSession(t, c, cwd, nil).SessionId
	if warm {
		if _, err := prompt(t, c, sid, "one"); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(cwd, "AGENTS.md"), "Second rule.")
	}
	armed.Store(true)
	if _, err := prompt(t, c, sid, "two"); err == nil || !strings.Contains(err.Error(), "scripted append failure") {
		t.Fatalf("the failing prompt answered %v, want the scripted append failure", err)
	}
	if !fired.Load() {
		t.Fatal("the append the test means to fail never ran")
	}
	live = historyOf(ag, sid)
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: sid}); err != nil {
		t.Fatal(err)
	}
	if files, err := filepath.Glob(filepath.Join(dir, "*", "*_"+string(sid)+".jsonl")); err != nil || len(files) == 0 {
		return live, nil // no file: nothing to load, so no history
	}
	ag2, c2 := preTurnAgent(t, dir, cfg, llmtest.NewScript(llmtest.Text("c")), nil)
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: sid, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	return live, historyOf(ag2, sid)
}

// After an append that fails before the turn starts, the live session holds
// the history a fresh load of its file gives (0037-MADR D15). Three appends
// come before the turn: the system prompt's baseline or patch, the user
// entry, and the gobble.tools entry. At the first two exits the history
// already equals the file's with no replay, because at most one append
// precedes each and the history takes an entry only after its write
// succeeds, so those cases check that equality: each fails when the history
// is let run ahead there. At the gobble.tools exit the file holds the prompt
// that the history lacks, and the replay there is what makes them equal, so
// that case fails without it.
func TestFailedPreTurnAppendHistoryIsTheFiles(t *testing.T) {
	isPrompt := func(e session.Entry) bool {
		return e.Type == session.TypeCustom && e.CustomType == session.CustomPrompt
	}
	isUser := func(e session.Entry) bool { return e.Message != nil && e.Message.Role == session.RoleUser }
	isTools := func(e session.Entry) bool { return e.Type == session.TypeCustom && e.CustomType == customTools }
	for _, tc := range []struct {
		name            string
		warm, resources bool
		match           func(session.Entry) bool
	}{
		{"system prompt baseline", false, false, isPrompt},
		{"system prompt patch", true, false, isPrompt},
		{"user entry", false, false, isUser},
		{"user entry after a patch", true, false, isUser},
		{"tools entry", false, true, isTools},
	} {
		t.Run(tc.name, func(t *testing.T) {
			live, file := preTurnFailure(t, tc.warm, tc.resources, tc.match)
			if !slices.Equal(live, file) {
				t.Errorf("after the failed append, the live history is not the file's:\n  live: %q\n  file: %q", live, file)
			}
		})
	}
}
