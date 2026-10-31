package acpclient

import (
	"bytes"
	"encoding/json/jsontext"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// syncBuffer is a log the connection writes while the test reads it.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// notifyPair connects an agent-side connection to a client-side one serving
// h, through two pipes, and sends initialize. send sends each notification
// from the agent, then session/new from the client: the SDK returns a
// response to its caller only after the notifications received before it
// are handled, so send returns once h has handled them and the connection
// has logged any failure. logs is the client connection's log. A
// notification reaches h only through a connection, which ExtensionMux
// needs to tell it from a request (0039-PLAN deviation 6).
func notifyPair(t *testing.T, h *handler) (send func(method string, params ...jsontext.Value), logs *syncBuffer) {
	t.Helper()
	logs = &syncBuffer{}
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	t.Cleanup(func() {
		toAgentW.Close()  //nolint:errcheck,gosec // test cleanup
		toClientW.Close() //nolint:errcheck,gosec // test cleanup
	})
	agent := acp.NewAgentSideConnection(&acptest.ScriptAgent{}, toClientW, toAgentR, acp.WithLogger(slog.New(slog.DiscardHandler)))
	client := acp.NewClientSideConnection(h, toAgentW, toClientR, acp.WithLogger(slog.New(slog.NewTextHandler(logs, nil))))
	if _, err := client.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber}); err != nil {
		t.Fatal(err)
	}
	send = func(method string, params ...jsontext.Value) {
		t.Helper()
		for _, p := range params {
			if err := agent.NotifyExtension(t.Context(), method, p); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: acptest.Abs("/w"), McpServers: []acp.McpServer{}}); err != nil {
			t.Fatal(err)
		}
	}
	return send, logs
}

// _gobble/retry reaches its session's callback as KindRetry, decoded, with
// its params kept; a session the client does not have gets nothing; another
// extension notification is ignored; and params that do not decode are
// logged and reach no callback (0005-PLAN F5-1 decision 5; 0039-MADR D12).
func TestRetryNotification(t *testing.T) {
	var got []Update
	h := &handler{sessions: map[acp.SessionId]func(Update){"s1": func(u Update) { got = append(got, u) }}}
	send, logs := notifyPair(t, h)
	params := jsontext.Value(`{"sessionId":"s1","phase":"start","cause":"empty_reply","attempt":1,"maxAttempts":2,"delayMs":2000}`)
	send("_gobble/retry", params, jsontext.Value(`{"sessionId":"other","phase":"end","cause":"offline"}`))
	want := Retry{Phase: "start", Cause: "empty_reply", Attempt: 1, MaxAttempts: 2, Delay: 2 * time.Second}
	if len(got) != 1 || got[0].Kind != KindRetry || got[0].Retry != want || string(got[0].Params) != string(params) {
		t.Fatalf("updates %+v, want one KindRetry %+v with its params", got, want)
	}
	send("_gobble/other", jsontext.Value(`{"sessionId":"s1"}`))
	if len(got) != 1 || strings.Contains(logs.String(), "_gobble/other") {
		t.Fatalf("another method: updates %+v, log %q; want it ignored", got, logs)
	}
	send("_gobble/retry", jsontext.Value(`{"sessionId":"s1","phase":"start","attempt":"x"}`))
	if len(got) != 1 || !strings.Contains(logs.String(), "failed to handle notification") || !strings.Contains(logs.String(), "_gobble/retry") {
		t.Fatalf("bad params: updates %+v, log %q; want it logged and not delivered", got, logs)
	}
}

// The client decodes a _gobble/* notification with encoding/json/v2 under
// the data policy: a cause ending in a lone surrogate escape reaches the
// callback with U+FFFD, and one whose attempt is a string is logged and
// ignored (0039-MADR D12, Confirmation 18).
func TestNotificationDecodedWithV2(t *testing.T) {
	var got []Update
	h := &handler{sessions: map[acp.SessionId]func(Update){"s1": func(u Update) { got = append(got, u) }}}
	send, logs := notifyPair(t, h)
	surrogate := string(rune(92)) + "ud800"
	send("_gobble/retry", jsontext.Value(`{"sessionId":"s1","phase":"start","cause":"offline`+surrogate+`"}`))
	if len(got) != 1 || got[0].Retry.Cause != "offline�" {
		t.Fatalf("lone surrogate: updates %+v, want one with cause %q", got, "offline�")
	}
	send("_gobble/retry", jsontext.Value(`{"sessionId":"s1","phase":"start","attempt":"1"}`))
	if len(got) != 1 || !strings.Contains(logs.String(), "failed to handle notification") {
		t.Fatalf("a string attempt: updates %+v, log %q; want it logged and not delivered", got, logs)
	}
}
