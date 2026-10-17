package acpserver

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// A stored session whose cwd is not absolute on this host is left out of
// session/list, logged once however often the list is asked for, and the
// other sessions still list (0019-MADR D5).
func TestListSkipsNonAbsoluteCwd(t *testing.T) {
	store := jsonl.New(t.TempDir(), jsonl.Options{})
	abs := t.TempDir()
	for i, cwd := range []string{abs, "relative/dir"} {
		id := session.ID([]string{"abs-0001", "rel-0002"}[i])
		h := session.Header{Type: session.TypeHeader, Version: session.Version, ID: id, Timestamp: "2026-10-13T10:00:00.000Z", Cwd: cwd}
		log, err := store.Create(t.Context(), h)
		if err != nil {
			t.Fatal(err)
		}
		var e session.Entry
		line := `{"type":"message","id":"m0000001","parentId":null,"timestamp":"2026-10-13T10:00:01.000Z","message":{"role":"user","content":"hello","timestamp":1}}`
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		if err := log.Append(t.Context(), e); err != nil {
			t.Fatal(err)
		}
		if err := log.Close(); err != nil {
			t.Fatal(err)
		}
	}
	var logs bytes.Buffer
	ag := New(Options{Version: "1.2.3", Provider: func(string, llm.Retry) (llm.Provider, error) { return llmtest.NewScript(), nil },
		Models: func() ModelChoice { return testModels }, Store: store, Logger: slog.New(slog.NewTextHandler(&logs, nil))})
	c := acptest.Connect(t, ag, &acptest.Client{})
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	for range 2 {
		resp, err := c.Client.ListSessions(t.Context(), acp.ListSessionsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Sessions) != 1 || resp.Sessions[0].SessionId != "abs-0001" || resp.Sessions[0].Cwd != abs {
			t.Fatalf("sessions %+v, want only abs-0001", resp.Sessions)
		}
	}
	if n := strings.Count(logs.String(), "session: not listed"); n != 1 || !strings.Contains(logs.String(), "session=rel-0002") ||
		!strings.Contains(logs.String(), "cwd=relative/dir") {
		t.Fatalf("%d not-listed lines in the log, want 1 naming rel-0002:\n%s", n, logs.String())
	}
}
