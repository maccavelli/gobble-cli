package acpserver

import (
	"log/slog"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// countingSettings is a Settings function that records each call's cwd.
type countingSettings struct {
	mu   sync.Mutex
	cwds []string
	s    SessionSettings
}

func (c *countingSettings) get(cwd string) SessionSettings {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cwds = append(c.cwds, cwd)
	return c.s
}

func (c *countingSettings) calls() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.cwds...)
}

// settingsAgent is an agent with settings, a store, an MCP config and a
// log, joined to an approving client.
func settingsAgent(t *testing.T, store session.Store, p llm.Provider, settings func(string) SessionSettings, mo MCPOptions) (*acptest.Conn, *acptest.Client, *syncBuffer) {
	t.Helper()
	logs := &syncBuffer{}
	a := New(Options{
		Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Store: store, Settings: settings, MCP: mo, Logger: slog.New(slog.NewTextHandler(logs, nil)),
	})
	client := &acptest.Client{Allow: true}
	c := acptest.Connect(t, a, client)
	a.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Logf("close: %v", err)
		}
	})
	initialize(t, c.Client)
	return c, client, logs
}

// A session takes its settings once, when it starts, from its cwd: the
// queue modes and thinking in get_state, /compact's keepRecentTokens, and
// the MCP start-up wait.
func TestSessionSettingsApplied(t *testing.T) {
	stub := &countingSettings{s: SessionSettings{
		SteeringMode: queueAll, FollowUpMode: queueAll, Thinking: "low",
		Compaction:     compaction.Settings{KeepRecentTokens: 150},
		MCPStartupWait: time.Millisecond,
	}}
	model := llmtest.NewScript(llmtest.Text(big(100)), llmtest.Text(big(100)), llmtest.Text("## Goal\nSUMMARY"))
	cfg := writeMCP(t, stdioEntry(t, "fx", map[string]string{mcptest.EnvDelay: "5s"}))
	c, client, logs := settingsAgent(t, session.NewMemoryStore(), model, stub.get, MCPOptions{Config: cfg})
	cwd := t.TempDir()
	id := newSession(t, c, cwd, nil).SessionId

	state := mustExt(t, c, "_gobble/get_state", map[string]any{"sessionId": id})
	if state["steeringMode"] != queueAll || state["followUpMode"] != queueAll || state["thinkingLevel"] != "low" {
		t.Fatalf("state %v", state)
	}
	begin := time.Now()
	for _, p := range []string{"one " + big(100), "two " + big(100)} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(begin); d > 3*time.Second {
		t.Fatalf("the first prompt waited %v for a server that answers in 5s", d)
	}
	if !strings.Contains(logs.String(), "still connecting") {
		t.Fatalf("the shortened wait was not logged:\n%s", logs)
	}
	n := len(client.Updates())
	if _, err := prompt(t, c, id, "/compact"); err != nil {
		t.Fatal(err)
	}
	if text := agentText(client.Updates()[n:]); !strings.Contains(text, "Compacted from ") {
		t.Fatalf("keepRecentTokens 150 did not let /compact cut: %q", text)
	}
	if got := stub.calls(); len(got) != 1 || got[0] != cwd {
		t.Fatalf("Settings called %d times, with %q; want once, at session start, with the cwd", len(got), got)
	}
}

// A nil Settings is today's defaults, and a partial value is completed from
// them, Options.Compaction and MCPOptions.StartupWait among them, and the
// user's home, which places ~/.ssh (0005-PLAN F6-2 decision 5).
func TestNilSettingsIsToday(t *testing.T) {
	home, _ := os.UserHomeDir()
	today := SessionSettings{
		SteeringMode: queueOneAtATime, FollowUpMode: queueOneAtATime, Thinking: thinkingOff, CacheRetention: "short",
		Compaction: compaction.DefaultSettings, MCPStartupWait: DefaultStartupWait, Retry: llm.DefaultRetry, Home: home,
	}
	if got := New(Options{}).sessionSettings("/x"); !reflect.DeepEqual(got, today) {
		t.Fatalf("nil Settings = %+v, want %+v", got, today)
	}
	custom := compaction.Settings{ReserveTokens: 1, KeepRecentTokens: 2}
	a := New(Options{Compaction: custom, MCP: MCPOptions{StartupWait: time.Second},
		Settings: func(string) SessionSettings {
			return SessionSettings{FollowUpMode: queueAll, Compaction: compaction.Settings{KeepRecentTokens: 7}}
		}})
	want := SessionSettings{
		SteeringMode: queueOneAtATime, FollowUpMode: queueAll, Thinking: thinkingOff, CacheRetention: "short",
		Compaction: compaction.Settings{ReserveTokens: 1, KeepRecentTokens: 7}, MCPStartupWait: time.Second,
		Retry: llm.DefaultRetry, Home: home,
	}
	if got := a.sessionSettings("/x"); !reflect.DeepEqual(got, want) {
		t.Fatalf("partial Settings = %+v, want %+v", got, want)
	}
	c, _ := sessionAgent(t, t.TempDir(), llmtest.NewScript(), &acptest.Client{})
	state := mustExt(t, c, "_gobble/get_state", map[string]any{"sessionId": newSession(t, c, t.TempDir(), nil).SessionId})
	if state["steeringMode"] != queueOneAtATime || state["followUpMode"] != queueOneAtATime || state["thinkingLevel"] != thinkingOff {
		t.Fatalf("state %v", state)
	}
}

// A loaded session keeps its history's thinking level, "off" when the
// history has none, whatever Settings says; its queue modes are the
// settings'.
func TestResumedKeepsHistoryThinking(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	settings := func(string) SessionSettings { return SessionSettings{Thinking: "minimal", SteeringMode: queueAll} }
	c, _, _ := settingsAgent(t, jsonl.New(dir, jsonl.Options{}), llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b")), settings, MCPOptions{})
	set := newSession(t, c, cwd, nil).SessionId
	if _, err := c.Client.SetSessionConfigOption(t.Context(), acp.SetSessionConfigOptionRequest{ValueId: &acp.SetSessionConfigOptionValueId{
		SessionId: set, ConfigId: acp.SessionConfigId(configThinking), Value: acp.SessionConfigValueId("high")}}); err != nil {
		t.Fatal(err)
	}
	plain := newSession(t, c, cwd, nil).SessionId
	for _, id := range []acp.SessionId{set, plain} {
		if _, err := prompt(t, c, id, "hi"); err != nil { // a session is written with its first turn
			t.Fatal(err)
		}
		if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
			t.Fatal(err)
		}
	}
	c2, _, _ := settingsAgent(t, jsonl.New(dir, jsonl.Options{}), llmtest.NewScript(), settings, MCPOptions{})
	for id, want := range map[acp.SessionId]string{set: "high", plain: thinkingOff} {
		if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
			t.Fatal(err)
		}
		state := mustExt(t, c2, "_gobble/get_state", map[string]any{"sessionId": id})
		if state["thinkingLevel"] != want || state["steeringMode"] != queueAll {
			t.Errorf("%s: state %v, want thinking %s and the settings' steering mode", id, state, want)
		}
	}
}
