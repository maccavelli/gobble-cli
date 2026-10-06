package acpclient

import (
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
