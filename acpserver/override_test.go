package acpserver

import (
	"slices"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// overrideAgent is an agent over the store in dir with override o, whose
// catalog is catalogModels. close ends it, so another agent may open its
// sessions.
func overrideAgent(t *testing.T, dir string, o Override, p llm.Provider) (c *acptest.Conn, closeAgent func()) {
	t.Helper()
	ag := New(Options{
		Version:  "1.2.3",
		Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Models:   func() ModelChoice { return testModels },
		ModelInfo: func(provider, model string) llm.Model {
			m := catalogModels[model]
			m.Provider, m.ID = provider, model
			return m
		},
		Override: o,
		Store:    jsonl.New(dir, jsonl.Options{}),
	})
	c = acptest.Connect(t, ag, &acptest.Client{})
	ag.SetConnection(c.Agent)
	initialize(t, c.Client)
	closed := false
	closeAgent = func() {
		if !closed {
			closed = true
			if err := ag.Close(); err != nil {
				t.Error(err)
			}
		}
	}
	t.Cleanup(closeAgent)
	return c, closeAgent
}

// currentModel is the model option's current value in opts.
func currentModel(t *testing.T, opts []acp.SessionConfigOption) string {
	t.Helper()
	for _, o := range opts {
		if o.Select != nil && string(o.Select.Id) == configModel {
			return string(o.Select.CurrentValue)
		}
	}
	t.Fatal("no model option")
	return ""
}

// storedSession is a session written by an agent with no override: one
// turn on m1, at level think, then that agent closed.
func storedSession(t *testing.T, dir, cwd, think string) acp.SessionId {
	t.Helper()
	c, closeAgent := overrideAgent(t, dir, Override{}, llmtest.NewScript(llmtest.Text("hi")))
	id := newSession(t, c, cwd, nil).SessionId
	if think != thinkingOff {
		if _, err := setOption(t, c, id, configThinking, think); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := prompt(t, c, id, "hello"); err != nil {
		t.Fatal(err)
	}
	closeAgent()
	return id
}

func resume(t *testing.T, c *acptest.Conn, id acp.SessionId, cwd string) []acp.SessionConfigOption {
	t.Helper()
	resp, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd})
	if err != nil {
		t.Fatal(err)
	}
	return resp.ConfigOptions
}

// A new session starts with the override's model and level, and records
// nothing for them, as a session starts with its defaults (0005-PLAN F4-2c).
func TestOverrideNew(t *testing.T) {
	c, _ := overrideAgent(t, t.TempDir(), Override{Provider: "fake", Model: "m2"}, llmtest.NewScript())
	resp := newSession(t, c, t.TempDir(), nil)
	if got := currentModel(t, resp.ConfigOptions); got != "fake/m2" {
		t.Errorf("model %q, want the override's m2", got)
	}
	if got := entryTypes(t, c, resp.SessionId); slices.Contains(got, "model_change") || slices.Contains(got, "thinking_level_change") {
		t.Errorf("entries %q, want no change recorded", got)
	}
	c, _ = overrideAgent(t, t.TempDir(), Override{Thinking: "low"}, llmtest.NewScript())
	resp = newSession(t, c, t.TempDir(), nil)
	if _, current := thoughtLevel(t, resp.ConfigOptions); current != "low" {
		t.Errorf("thought_level %q, want the override's low", current)
	}
	if got := entryTypes(t, c, resp.SessionId); slices.Contains(got, "thinking_level_change") {
		t.Errorf("entries %q, want no change recorded", got)
	}
}

// A resumed session is set to the override after its history: the model
// change, and the level it clamps, each recorded once. A second resume with
// the same override records nothing more.
func TestOverrideResume(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	id := storedSession(t, dir, cwd, "low")
	o := Override{Provider: "fake", Model: "m2"}
	c, closeAgent := overrideAgent(t, dir, o, llmtest.NewScript())
	opts := resume(t, c, id, cwd)
	if got := currentModel(t, opts); got != "fake/m2" {
		t.Errorf("model %q, want the override's m2", got)
	}
	if _, current := thoughtLevel(t, opts); current != thinkingOff {
		t.Errorf("thought_level %q, want low clamped to m2's off", current)
	}
	// The stored session has /thinking low's entry and no model_change.
	after := entryTypes(t, c, id)
	if tail := after[max(0, len(after)-2):]; !slices.Equal(tail, []string{"model_change", "thinking_level_change"}) ||
		countOf(after, "model_change") != 1 || countOf(after, "thinking_level_change") != 2 {
		t.Errorf("entries %q, want one model_change and the clamp's thinking_level_change after the history", after)
	}
	if got := recordedLevel(t, c, id); got != thinkingOff {
		t.Errorf("recorded level %q, want low clamped to m2's off", got)
	}
	closeAgent()
	c, _ = overrideAgent(t, dir, o, llmtest.NewScript())
	resume(t, c, id, cwd)
	if again := entryTypes(t, c, id); len(again) != len(after) {
		t.Errorf("a second resume wrote %q, want nothing", again[len(after):])
	}
}

// --fork's copy is set to the override after the copied history.
func TestOverrideFork(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	src := storedSession(t, dir, cwd, thinkingOff)
	c, _ := overrideAgent(t, dir, Override{Provider: "fake", Model: "m2"}, llmtest.NewScript())
	resp := newSession(t, c, cwd, map[string]any{"gobble": map[string]any{"forkFrom": string(src)}})
	if got := currentModel(t, resp.ConfigOptions); got != "fake/m2" {
		t.Errorf("model %q, want the override's m2", got)
	}
	if got := entryTypes(t, c, resp.SessionId); len(got) == 0 || got[len(got)-1] != "model_change" {
		t.Errorf("entries %q, want the copied history then a model_change", got)
	}
}

// The override's level is clamped to the model's: max, on m1, which offers
// off, low and high, is high.
func TestOverrideClamp(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	id := storedSession(t, dir, cwd, thinkingOff)
	c, _ := overrideAgent(t, dir, Override{Thinking: "max"}, llmtest.NewScript())
	opts := resume(t, c, id, cwd)
	if _, current := thoughtLevel(t, opts); current != "high" {
		t.Errorf("thought_level %q, want max clamped to m1's high", current)
	}
	// The stored session, at off, has no thinking_level_change of its own.
	if got := entryTypes(t, c, id); len(got) == 0 || got[len(got)-1] != "thinking_level_change" ||
		countOf(got, "thinking_level_change") != 1 || countOf(got, "model_change") != 0 {
		t.Errorf("entries %q, want one thinking_level_change after the history, and no model_change", got)
	}
	if got := recordedLevel(t, c, id); got != "high" {
		t.Errorf("recorded level %q, want max clamped to m1's high", got)
	}
}

// recordedLevel is the level the session's last thinking_level_change
// records: what a later resume restores, where thought_level shows a level
// clamped for display.
func recordedLevel(t *testing.T, c *acptest.Conn, id acp.SessionId) string {
	t.Helper()
	level := ""
	for _, e := range mustExt(t, c, "_gobble/get_entries", map[string]any{"sessionId": id})["entries"].([]any) {
		if m := e.(map[string]any); m["type"] == "thinking_level_change" {
			level, _ = m["thinkingLevel"].(string)
		}
	}
	return level
}

func countOf(types []string, typ string) int {
	n := 0
	for _, t := range types {
		if t == typ {
			n++
		}
	}
	return n
}
