package acpclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// nameAgent is a script agent that answers _gobble/set_session_name,
// refusing an empty name as gobble's agent does.
type nameAgent struct {
	*acptest.ScriptAgent
	mu    sync.Mutex
	calls []string
}

func (a *nameAgent) HandleExtensionMethod(_ context.Context, method string, params json.RawMessage) (any, error) {
	a.mu.Lock()
	a.calls = append(a.calls, method+" "+string(params))
	a.mu.Unlock()
	if method != "_gobble/set_session_name" {
		return nil, acp.NewMethodNotFound(method)
	}
	var p setNameParams
	if err := json.Unmarshal(params, &p); err != nil || p.Name == "" {
		return nil, acp.NewInvalidParams(map[string]any{"reason": "Session name cannot be empty"})
	}
	return map[string]any{}, nil
}

func (a *nameAgent) serve(ctx context.Context, in io.Reader, out io.Writer) error {
	conn := acp.NewAgentSideConnection(a, out, in, acp.WithLogger(slog.New(slog.DiscardHandler)))
	select {
	case <-conn.Done():
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// SetName calls _gobble/set_session_name with the session and the name; a
// refusal is ErrInvalidSession with the agent's reason.
func TestSetName(t *testing.T) {
	a := &nameAgent{ScriptAgent: &acptest.ScriptAgent{}}
	c, err := Start(t.Context(), a.serve, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	s, err := c.NewSession(t.Context(), "/w", func(Update) {})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetName(t.Context(), "x"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetName(t.Context(), ""); !errors.Is(err, ErrInvalidSession) || err.Error() != "Session name cannot be empty" {
		t.Fatalf("empty name: %v", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	want := []string{`_gobble/set_session_name {"sessionId":"s1","name":"x"}`, `_gobble/set_session_name {"sessionId":"s1","name":""}`}
	if !slices.Equal(a.calls, want) {
		t.Fatalf("calls %q, want %q", a.calls, want)
	}
}

// NewSessionOptions travel as _meta.gobble, holding only what is set.
func TestNewSessionMeta(t *testing.T) {
	if (NewSessionOptions{}).meta() != nil {
		t.Fatal("empty options sent _meta")
	}
	m := NewSessionOptions{ID: "i", Name: "n"}.meta()
	g, _ := m["gobble"].(map[string]any) //nolint:errcheck // checked below
	if len(g) != 2 || g["sessionId"] != "i" || g["name"] != "n" {
		t.Fatalf("meta %v", m)
	}
}

// Options.Permission answers permission requests with an offered option;
// an answer naming none cancels; with no Permission the client declines.
func TestRequestPermission(t *testing.T) {
	req := acp.RequestPermissionRequest{SessionId: "s", ToolCall: acp.ToolCallUpdate{ToolCallId: "c1", Title: new("write a.txt")},
		Options: []acp.PermissionOption{{OptionId: "y", Name: "Allow", Kind: acp.PermissionOptionKindAllowOnce}, {OptionId: "n", Name: "Reject", Kind: acp.PermissionOptionKindRejectOnce}}}
	var asked PermissionRequest
	pick := func(id string) *handler {
		return &handler{permission: func(_ context.Context, r PermissionRequest) PermissionAnswer {
			asked = r
			return PermissionAnswer{OptionID: id}
		}}
	}
	selected := func(h *handler) string {
		resp, err := h.RequestPermission(t.Context(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Outcome.Selected == nil {
			return ""
		}
		return string(resp.Outcome.Selected.OptionId)
	}
	if got := selected(pick("y")); got != "y" || asked.Tool.Title != "write a.txt" || asked.Tool.ID != "c1" || len(asked.Options) != 2 || asked.Options[1].Kind != "reject_once" {
		t.Fatalf("allow: %q, asked %+v", got, asked)
	}
	if got := selected(pick("")); got != "" {
		t.Fatalf("an empty answer selected %q, want cancelled", got)
	}
	if got := selected(pick("other")); got != "" {
		t.Fatalf("an answer naming no option selected %q", got)
	}
	if got := selected(&handler{}); got != "n" {
		t.Fatalf("no Permission selected %q, want reject_once", got)
	}
}

// The agent's refusals become ErrSessionLocked and ErrInvalidSession, with
// its reason as the text; anything else stays wrapped.
func TestSessionError(t *testing.T) {
	locked := acp.NewInvalidRequest(map[string]any{"reason": "session s is open in another gobble process (pid 7)"})
	if err := sessionError("session/resume", locked); !errors.Is(err, ErrSessionLocked) || err.Error() != "session s is open in another gobble process (pid 7)" {
		t.Fatalf("locked: %v", err)
	}
	busy := acp.NewInvalidRequest(map[string]any{"reason": "a prompt is already running"})
	if err := sessionError("session/resume", busy); errors.Is(err, ErrSessionLocked) || errors.Is(err, ErrInvalidSession) {
		t.Fatalf("another invalid request mapped: %v", err)
	}
	bad := acp.NewInvalidParams(map[string]any{"reason": "a session with this id exists"})
	if err := sessionError("session/new", bad); !errors.Is(err, ErrInvalidSession) || err.Error() != "a session with this id exists" {
		t.Fatalf("invalid: %v", err)
	}
	other := errors.New("pipe closed")
	if err := sessionError("session/new", other); !errors.Is(err, other) {
		t.Fatalf("other: %v", err)
	}
}
