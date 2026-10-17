package acpserver

import (
	"slices"
	"sync"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// cacheAgent is an agent on model p whose Provider records the session id
// of each provider it builds, with settings, connected and initialized.
func cacheAgent(t *testing.T, p llm.Provider, settings func(string) SessionSettings) (*acptest.Conn, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var built []string
	a := New(Options{Version: "1.2.3", Models: func() ModelChoice { return testModels }, Store: jsonl.New(t.TempDir(), jsonl.Options{}),
		Settings: settings, Compaction: compaction.Settings{ReserveTokens: 16384, KeepRecentTokens: 150},
		Provider: func(id string, _ llm.Retry) (llm.Provider, error) {
			mu.Lock()
			defer mu.Unlock()
			built = append(built, id)
			return p, nil
		}})
	c := acptest.Connect(t, a, &acptest.Client{})
	a.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(built)
	}
}

// Each session builds its own provider, once, with its ACP id; a session
// loaded again builds one more, with the same id (0021-MADR D4).
func TestSessionProvidersHaveTheirIDs(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"), llmtest.Text("c"), llmtest.Text("d"))
	c, built := cacheAgent(t, model, nil)
	cwd := t.TempDir()
	s1, s2 := newSession(t, c, cwd, nil).SessionId, newSession(t, c, cwd, nil).SessionId
	for _, id := range []acp.SessionId{s1, s2, s1} {
		if _, err := prompt(t, c, id, "hi"); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := built(), []string{string(s1), string(s2)}; !slices.Equal(got, want) {
		t.Fatalf("providers built for %q, want %q: one per session, kept", got, want)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: s1}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: s1, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := prompt(t, c, s1, "again"); err != nil {
		t.Fatal(err)
	}
	if got, want := built(), []string{string(s1), string(s2), string(s1)}; !slices.Equal(got, want) {
		t.Fatalf("after a reload, providers built for %q, want %q", got, want)
	}
}

// Every model call of a turn carries the session's cacheRetention: short
// by default, long, or nothing for none (0021-MADR D2, D3).
func TestTurnAsksForCache(t *testing.T) {
	for _, c := range []struct {
		name     string
		settings func(string) SessionSettings
		want     llm.CacheRetention
	}{
		{"default", nil, llm.CacheShort},
		{"long", func(string) SessionSettings { return SessionSettings{CacheRetention: "long"} }, llm.CacheLong},
		{"none", func(string) SessionSettings { return SessionSettings{CacheRetention: "none"} }, llm.CacheNone},
	} {
		t.Run(c.name, func(t *testing.T) {
			model := llmtest.NewScript(llmtest.ToolCall("c1", "read", `{"path":"missing.txt"}`), llmtest.Text("done"))
			conn, _ := cacheAgent(t, model, c.settings)
			if _, err := prompt(t, conn, newSession(t, conn, t.TempDir(), nil).SessionId, "read it"); err != nil {
				t.Fatal(err)
			}
			reqs := model.Requests()
			if len(reqs) != 2 || reqs[0].Cache != c.want || reqs[1].Cache != c.want {
				var got []llm.CacheRetention
				for _, r := range reqs {
					got = append(got, r.Cache)
				}
				t.Fatalf("requests ask %q, want two asking %q", got, c.want)
			}
		})
	}
}

// A compaction's summary request asks for no cache, while the turns around
// it do (0021-MADR D2, F7; 0021-PLAN C3).
func TestCompactionAsksNoCache(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text(big(100)), llmtest.Text(big(100)), llmtest.Text("## Goal\nSUMMARY"), llmtest.Text("after"))
	c, _ := cacheAgent(t, model, nil)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	for _, p := range []string{"one " + big(100), "two " + big(100), "/compact", "three"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	reqs := model.Requests()
	if len(reqs) != 4 {
		t.Fatalf("%d requests, want 4", len(reqs))
	}
	for i, want := range []llm.CacheRetention{llm.CacheShort, llm.CacheShort, llm.CacheNone, llm.CacheShort} {
		if reqs[i].Cache != want {
			t.Errorf("request %d asks %q, want %q", i, reqs[i].Cache, want)
		}
	}
}
