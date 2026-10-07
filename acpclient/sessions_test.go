package acpclient

import (
	"context"
	"errors"
	"testing"

	acp "github.com/coder/acp-go-sdk"
)

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
