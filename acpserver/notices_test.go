package acpserver

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// offersNotices are client capabilities with session.notices.
var offersNotices = acp.ClientCapabilities{Session: &acp.ClientSessionCapabilities{Notices: &acp.NoticeCapabilities{}}}

// noticeAgent is an agent over p with o's MCP settings and cfg as its
// config directory, storing sessions in dir, its client offering caps.
func noticeAgent(t *testing.T, dir, cfg string, p llm.Provider, mo MCPOptions, caps acp.ClientCapabilities) (*acptest.Conn, *acptest.Client, *syncBuffer) {
	t.Helper()
	logs := &syncBuffer{}
	ag := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Models: func() ModelChoice { return testModels }, Store: jsonl.New(dir, jsonl.Options{}), Tools: []tool.Tool{builtin.Read()},
		Prompt: PromptOptions{ConfigDir: cfg}, MCP: mo, Logger: slog.New(slog.NewTextHandler(logs, nil))})
	client := &acptest.Client{Allow: true}
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Logf("close: %v", err)
		}
	})
	if _, err := c.Client.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber, ClientCapabilities: caps,
		ClientInfo: &acp.Implementation{Name: "test", Version: "0.0.0"}}); err != nil {
		t.Fatal(err)
	}
	return c, client, logs
}

// noticeMCP is an mcp.json with each MCP diagnostic: a config error, a
// server that fails, and a slow server whose exposure is not applied yet
// and which is still connecting at the first prompt.
func noticeMCP(t *testing.T) string {
	t.Helper()
	cmd, env := mcptest.StdioServer(t)
	env[mcptest.EnvDelay] = "1500ms"
	doc := map[string]any{
		"autoEnableCodemode": "yes",
		"mcpServers": map[string]any{
			"down": map[string]any{"type": "http", "url": "http://127.0.0.1:1/mcp"},
			"slow": map[string]any{"command": cmd, "env": env, "exposure": "deferred"},
		},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// shadowConfig is a config directory whose help template a built-in
// shadows, so the session's start notes a file not used.
func shadowConfig(t *testing.T) string {
	t.Helper()
	cfg := t.TempDir()
	writeFile(t, filepath.Join(cfg, "prompts", "help.md"), "Not the help\n")
	return cfg
}

// notices are the notice updates the client has received, in order.
func notices(client *acptest.Client) []*acp.SessionUpdateNotice {
	var out []*acp.SessionUpdateNotice
	for _, u := range client.Updates() {
		if u.Update.Notice != nil {
			out = append(out, u.Update.Notice)
		}
	}
	return out
}

func descriptionOf(n *acp.SessionUpdateNotice) string {
	if n.Description == nil {
		return ""
	}
	return *n.Description
}

// Each of the five diagnostics reaches a client that offered notices as one
// notice, with D8's severity, the log message as its title and the detail
// as its description, and only once per session (0016-MADR D8).
func TestNoticesWhenOffered(t *testing.T) {
	t.Run("mcp", func(t *testing.T) {
		model := llmtest.NewScript(llmtest.Text("one"), llmtest.Text("two"))
		c, client, _ := noticeAgent(t, t.TempDir(), "", model, MCPOptions{Config: noticeMCP(t), StartupWait: 200 * time.Millisecond}, offersNotices)
		id := newSession(t, c, t.TempDir(), nil).SessionId
		if _, err := prompt(t, c, id, "one"); err != nil {
			t.Fatal(err)
		}
		want := []struct {
			severity    acp.NoticeSeverity
			title, desc string
		}{
			{acp.NoticeSeverityWarning, "mcp: config error", "autoEnableCodemode must be a boolean"},
			{acp.NoticeSeverityWarning, "mcp: exposure and toolExposure apply from 0005-PLAN F8; its tools are declared directly", "slow"},
			{acp.NoticeSeverityWarning, "mcp: server needs attention", "down"},
			{acp.NoticeSeverityInfo, "mcp: server still connecting; its tools join a later prompt", "slow"},
		}
		got := notices(client)
		if len(got) != len(want) {
			t.Fatalf("%d notices, want %d: %+v", len(got), len(want), got)
		}
		for _, w := range want {
			found := false
			for _, n := range got {
				if n.Severity == w.severity && n.Title == w.title && strings.Contains(descriptionOf(n), w.desc) {
					found = true
				}
			}
			if !found {
				t.Errorf("no %s notice %q with a description holding %q among %d", w.severity, w.title, w.desc, len(got))
			}
		}
		if _, err := prompt(t, c, id, "two"); err != nil {
			t.Fatal(err)
		}
		if n := len(notices(client)); n != len(want) {
			t.Errorf("%d notices after a second prompt, want still %d: each once per session", n, len(want))
		}
	})
	t.Run("prompt file", func(t *testing.T) {
		c, client, _ := noticeAgent(t, t.TempDir(), shadowConfig(t), llmtest.NewScript(), MCPOptions{}, offersNotices)
		_ = newSession(t, c, t.TempDir(), nil)
		got := notices(client)
		if len(got) != 1 || got[0].Severity != acp.NoticeSeverityWarning || got[0].Title != "prompt: a file was not used" ||
			!strings.Contains(descriptionOf(got[0]), "help.md") || !strings.Contains(descriptionOf(got[0]), `name "/help" collision`) {
			t.Fatalf("notices %+v, want one warning naming help.md and its collision", got)
		}
	})
}

// A client that did not offer notices receives none, and no new agent text;
// the diagnostics are logged as before (0016-MADR D8).
func TestNoNoticeWithoutTheCapability(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("one"))
	c, client, logs := noticeAgent(t, t.TempDir(), shadowConfig(t), model, MCPOptions{Config: noticeMCP(t), StartupWait: 200 * time.Millisecond},
		acp.ClientCapabilities{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "one"); err != nil {
		t.Fatal(err)
	}
	if got := notices(client); len(got) != 0 {
		t.Fatalf("notices %+v, want none", got)
	}
	var text bytes.Buffer
	for _, u := range client.Updates() {
		if m := u.Update.AgentMessageChunk; m != nil && m.Content.Text != nil {
			text.WriteString(m.Content.Text.Text)
		}
	}
	if text.String() != "one" {
		t.Errorf("agent text %q, want only the reply", text.String())
	}
	for _, w := range []string{"mcp: config error", "mcp: server needs attention", "still connecting", "exposure and toolExposure", "prompt: a file was not used"} {
		if !strings.Contains(logs.String(), w) {
			t.Errorf("the log lacks %q", w)
		}
	}
}

// A notice is live: a session/load replays none, and the session file holds
// none (0016-MADR D8; SDK 0039-MADR, notices are not session history).
func TestNoticesAreNotReplayed(t *testing.T) {
	dir := t.TempDir()
	model := llmtest.NewScript(llmtest.Text("one"))
	c, client, _ := noticeAgent(t, dir, "", model, MCPOptions{Config: noticeMCP(t), StartupWait: 200 * time.Millisecond}, offersNotices)
	cwd := t.TempDir()
	id := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, id, "one"); err != nil {
		t.Fatal(err)
	}
	if len(notices(client)) == 0 {
		t.Fatal("no notice was sent live; the test needs one to show it is not replayed")
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	c2, client2, _ := noticeAgent(t, dir, "", llmtest.NewScript(), MCPOptions{Config: noticeMCP(t), StartupWait: 200 * time.Millisecond}, offersNotices)
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	if got := notices(client2); len(got) != 0 {
		t.Errorf("session/load sent notices %+v, want none", got)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*", "*.jsonl"))
	if err != nil || len(files) == 0 {
		files, err = filepath.Glob(filepath.Join(dir, "*.jsonl"))
	}
	if err != nil || len(files) == 0 {
		t.Fatalf("no session file under %s (%v)", dir, err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f) //nolint:gosec // G304: the test's own session file.
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(b, []byte(`"notice"`)) {
			t.Errorf("%s holds a notice", f)
		}
	}
}
