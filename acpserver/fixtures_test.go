package acpserver

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// piFixtures are sessions written by Pi's own code, with Pi's context of
// each (0005-PLAN "F2 made executable"). Their cwd is /work/project.
const (
	piFixtures = "../session/testdata/pi"
	piCwd      = "/work/project"
)

// piModels choose the provider and model the fixtures were written with, so
// a load restores them.
var piModels = ModelChoice{Provider: "anthropic", Models: []string{"claude-sonnet-4-5", "other"}, Default: "other"}

// piAgent is an agent over a store holding the fixture name, on model p.
// It returns the connection, the client, the store's directory, the
// session's id and its file.
func piAgent(t *testing.T, name string, p llm.Provider) (*acptest.Conn, *acptest.Client, string, acp.SessionId, string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(piFixtures, name+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var h session.Header
	if err := json.Unmarshal(bytes.SplitN(body, []byte("\n"), 2)[0], &h); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, jsonl.EncodeCwd(h.Cwd), jsonl.FileName(h))
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, body, 0o600); err != nil {
		t.Fatal(err)
	}
	ag := New(Options{Version: "1.2.3", Provider: func() (llm.Provider, error) { return p, nil },
		Models: func() ModelChoice { return piModels }, Store: jsonl.New(dir, jsonl.Options{})})
	client := &acptest.Client{}
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c, client, dir, acp.SessionId(h.ID), file
}

// piContext is a fixture's context as Pi builds it.
type piContext struct {
	Messages []jsontext.Value `json:"messages"`
	LLM      []struct {
		Role    string         `json:"role"`
		Content jsontext.Value `json:"content"`
	} `json:"llm"`
}

func readPiContext(t *testing.T, name string) piContext {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(piFixtures, name+".context.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pc piContext
	if err := json.Unmarshal(b, &pc); err != nil {
		t.Fatal(err)
	}
	return pc
}

// llmRole is Pi's role of a message the model receives, as gobble's.
var llmRole = map[string]llm.Role{"user": llm.RoleUser, "assistant": llm.RoleAssistant, "toolResult": llm.RoleTool}

// turn is one message of a request, as role and text, the shape both
// sides share.
func turn(role llm.Role, text string) string { return string(role) + ": " + text }

// Loading Pi's v3 file gives the model Pi's context, restores the model and
// thinking level, answers get_messages with Pi's messages, and replays
// the shown messages only.
func TestPiFixtureContext(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("next"))
	c, client, _, id, _ := piAgent(t, "v3", model)
	resp, err := c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: piCwd, McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{string(resp.ConfigOptions[0].Select.CurrentValue), string(resp.ConfigOptions[1].Select.CurrentValue)}; got[0] != "claude-sonnet-4-5" || got[1] != "medium" {
		t.Errorf("restored model and thinking %v", got)
	}
	var shown []string
	for _, n := range client.Updates() {
		switch u := n.Update; {
		case u.UserMessageChunk != nil:
			shown = append(shown, "user: "+u.UserMessageChunk.Content.Text.Text)
		case u.AgentMessageChunk != nil:
			shown = append(shown, "agent: "+u.AgentMessageChunk.Content.Text.Text)
		}
	}
	replay := strings.Join(shown, "\n")
	for _, want := range []string{"user: A note from an extension.", "user: A custom message.", "user: Ran `ls`\n```\nREADME.md\n```",
		"agent: Branch summary:\n\nThe abandoned branch tried another approach.", "agent: There is one file, README.md."} {
		if !strings.Contains(replay, want) {
			t.Errorf("the replay lacks %q:\n%s", want, replay)
		}
	}
	for _, hidden := range []string{"Hidden context.", "Ran `pwd`", "There is one file.\n", "Trying another way.", "A fresh start."} {
		if strings.Contains(replay+"\n", hidden) {
			t.Errorf("the replay shows %q:\n%s", hidden, replay)
		}
	}

	pc := readPiContext(t, "v3")
	got := mustExt(t, c, "_gobble/get_messages", map[string]any{"sessionId": id})["messages"]
	var want any
	if err := json.Unmarshal(must(json.Marshal(pc.Messages)), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("get_messages:\n got %s\nwant %s", must(json.Marshal(got)), must(json.Marshal(want)))
	}

	if _, err := prompt(t, c, id, "and now?"); err != nil {
		t.Fatal(err)
	}
	var sent []string
	for _, m := range model.Requests()[0].Messages {
		if m.Role == llm.RoleSystem {
			continue
		}
		var text []string
		for _, ct := range m.Content {
			if ct.Type == llm.ContentText || ct.Type == llm.ContentToolResult {
				text = append(text, ct.Text)
			}
		}
		sent = append(sent, turn(m.Role, strings.Join(text, "")))
	}
	var pi []string
	for _, m := range pc.LLM {
		mm := session.Message{Role: m.Role, Content: m.Content}
		blocks, err := mm.Blocks()
		if err != nil {
			t.Fatal(err)
		}
		var text []string
		for _, b := range blocks {
			if b.Type == session.BlockText {
				text = append(text, b.Text)
			}
		}
		pi = append(pi, turn(llmRole[m.Role], strings.Join(text, "")))
	}
	pi = append(pi, turn(llm.RoleUser, "and now?"))
	if strings.Join(sent, "\n") != strings.Join(pi, "\n") {
		t.Errorf("the model was sent\n%s\nPi sends\n%s", strings.Join(sent, "\n"), strings.Join(pi, "\n"))
	}
}

// A custom message is replayed only when its display is true, and a bash
// execution only when the model sees it.
func TestReplayHidesHidden(t *testing.T) {
	yes, no := true, false
	for _, c := range []struct {
		m    *session.Message
		want int
	}{
		{&session.Message{Role: session.RoleCustom, Content: jsontext.Value(`"shown"`), Display: &yes}, 1},
		{&session.Message{Role: session.RoleCustom, Content: jsontext.Value(`"hidden"`), Display: &no}, 0},
		{&session.Message{Role: session.RoleCustom, Content: jsontext.Value(`"unset"`)}, 0},
		{&session.Message{Role: session.RoleBashExecution, Command: "ls"}, 1},
		{&session.Message{Role: session.RoleBashExecution, Command: "ls", ExcludeFromContext: &yes}, 0},
	} {
		if got := replayMessage(session.Entry{}, c.m, state{}, nil, ""); len(got) != c.want {
			t.Errorf("%s %s: %d updates, want %d", c.m.Role, c.m.Content, len(got), c.want)
		}
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// /session's totals count Pi's usage entries, and compactions' usage, as
// Pi's stats do: v3.jsonl's seven assistant messages have 1320 input and
// 100 output tokens, its compaction 300 and 60, its usage entry 50 and 0.
func TestPiFixtureTotals(t *testing.T) {
	c, _, _, id, _ := piAgent(t, "v3", llmtest.NewScript())
	if _, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: piCwd}); err != nil {
		t.Fatal(err)
	}
	tok, _ := mustExt(t, c, "_gobble/get_session_stats", map[string]any{"sessionId": id})["tokens"].(map[string]any) //nolint:errcheck // checked below
	if tok["input"] != float64(1670) || tok["output"] != float64(160) {
		t.Fatalf("tokens %v, want 1670 input and 160 output", tok)
	}
}

// An older file is read and never written: a prompt, /name and /compact
// are refused with ErrOldVersion's text before any model call, the file is
// unchanged, and /clone and forkFrom write v3 files of the migrated
// entries.
func TestPiFixtureOlder(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("never"))
	c, _, dir, id, file := piAgent(t, "v1", model)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: piCwd}); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"continue", "/name renamed", "/compact"} {
		_, err := prompt(t, c, id, text)
		re, ok := errors.AsType[*acp.RequestError](err)
		if !ok || re.Code != -32600 || !strings.Contains(err.Error(), "is a version 1 file, which gobble reads but does not rewrite; /clone it to continue in a new session") {
			t.Errorf("%s: %v, want invalid request with ErrOldVersion's text", text, err)
		}
	}
	if n := len(model.Requests()); n != 0 {
		t.Errorf("the model was asked %d times", n)
	}
	if after, err := os.ReadFile(file); err != nil || !bytes.Equal(after, before) {
		t.Fatalf("the v1 file changed (%v)", err)
	}

	store := jsonl.New(dir, jsonl.Options{})
	_, migrated, _, err := store.Read(t.Context(), session.ID(id))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := prompt(t, c, id, "/clone")
	if err != nil {
		t.Fatal(err)
	}
	fork := newSession(t, c, piCwd, map[string]any{"gobble": map[string]any{"forkFrom": string(id)}}).SessionId
	for what, copyID := range map[string]string{"clone": switchTo(t, resp), "forkFrom": string(fork)} {
		h, entries, _, err := store.Read(t.Context(), session.ID(copyID))
		if what == "forkFrom" && errors.Is(err, session.ErrNotFound) {
			continue // forkFrom writes with the fork's first message
		}
		if err != nil || h.Version != session.Version || !reflect.DeepEqual(entries, migrated) {
			t.Errorf("%s: version %d, %v; entries equal the migrated ones: %v", what, h.Version, err, reflect.DeepEqual(entries, migrated))
		}
	}
	if _, err := prompt(t, c, fork, "go on"); err != nil {
		t.Fatal(err)
	}
	if _, entries, _, err := store.Read(t.Context(), session.ID(fork)); err != nil || !reflect.DeepEqual(entries[:len(migrated)], migrated) {
		t.Errorf("forkFrom's file: %v; the migrated entries first: %v", err, err == nil && reflect.DeepEqual(entries[:len(migrated)], migrated))
	}
}
