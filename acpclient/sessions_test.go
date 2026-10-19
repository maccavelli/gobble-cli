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

	acp "github.com/maccavelli/acp-go-sdk2"

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
	s, err := c.NewSession(t.Context(), acptest.Abs("/w"), func(Update) {})
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

// dirsAgent is a script agent that records the additional directories of
// session/new and session/resume, and resumes any session.
type dirsAgent struct {
	*acptest.ScriptAgent
	mu  sync.Mutex
	got [][]string
}

func (a *dirsAgent) NewSession(ctx context.Context, p acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	a.mu.Lock()
	a.got = append(a.got, append([]string{"new"}, p.AdditionalDirectories...))
	a.mu.Unlock()
	return a.ScriptAgent.NewSession(ctx, p)
}

func (a *dirsAgent) ResumeSession(_ context.Context, p acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	a.mu.Lock()
	a.got = append(a.got, append([]string{"resume"}, p.AdditionalDirectories...))
	a.mu.Unlock()
	return acp.ResumeSessionResponse{}, nil
}

func (a *dirsAgent) serve(ctx context.Context, in io.Reader, out io.Writer) error {
	conn := acp.NewAgentSideConnection(a, out, in, acp.WithLogger(slog.New(slog.DiscardHandler)))
	select {
	case <-conn.Done():
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// --add-dir's directories reach an agent that advertises them on
// session/new and on session/resume; ResumeSession sends none (0018-MADR
// D14). acptest's recorder taps only acptest.Connect's pair, so the agent
// records them.
func TestAdditionalDirectoriesSent(t *testing.T) {
	a := &dirsAgent{ScriptAgent: &acptest.ScriptAgent{Capabilities: &acp.AgentCapabilities{SessionCapabilities: acp.SessionCapabilities{
		AdditionalDirectories: &acp.SessionAdditionalDirectoriesCapabilities{}, Resume: &acp.SessionResumeCapabilities{},
		Close: &acp.SessionCloseCapabilities{},
	}}}}
	c, err := Start(t.Context(), a.serve, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	w, a1, b1 := acptest.Abs("/w"), acptest.Abs("/w/a"), acptest.Abs("/w/b")
	dirs := []string{a1, b1}
	if _, err := c.NewSessionWith(t.Context(), w, NewSessionOptions{AdditionalDirectories: dirs}, func(Update) {}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.ResumeSessionWith(t.Context(), "s9", w, ResumeSessionOptions{AdditionalDirectories: dirs}, func(Update) {}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.ResumeSession(t.Context(), "s8", w, func(Update) {}); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	want := [][]string{{"new", a1, b1}, {"resume", a1, b1}, {"resume"}}
	if !slices.EqualFunc(a.got, want, slices.Equal) {
		t.Fatalf("got %q, want %q", a.got, want)
	}
}

// Toward an agent that does not advertise additional directories, the SDK
// refuses to send them, and the agent sees no request (0019-MADR D8).
func TestAdditionalDirectoriesNotAdvertised(t *testing.T) {
	a := &dirsAgent{ScriptAgent: &acptest.ScriptAgent{}}
	c, err := Start(t.Context(), a.serve, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	_, err = c.NewSessionWith(t.Context(), acptest.Abs("/w"), NewSessionOptions{AdditionalDirectories: []string{acptest.Abs("/w/a")}}, func(Update) {})
	want := acp.CapabilityError{Method: "session/new", Capability: "sessionCapabilities.additionalDirectories", Field: "additionalDirectories"}
	if ce, ok := errors.AsType[*acp.CapabilityError](err); !ok || *ce != want {
		t.Fatalf("err = %v, want the SDK's %+v", err, want)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.got) != 0 {
		t.Fatalf("the agent saw %q, want no request", a.got)
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
	if asked.Limit != "" || asked.Repeated != 0 {
		t.Fatalf("no _meta: asked %+v, want no marks", asked)
	}
	// gobble's marks, as JSON decodes them (gobble-cli 0005-PLAN F6-2a's
	// deviations 1 and 2); a mistyped one is none.
	req.Meta = map[string]any{"gobble": map[string]any{"limit": "a delete always asks", "repeated": float64(3)}}
	if selected(pick("n")); asked.Limit != "a delete always asks" || asked.Repeated != 3 {
		t.Fatalf("marks: asked %+v", asked)
	}
	req.Meta = map[string]any{"gobble": map[string]any{"limit": 7, "repeated": "3"}}
	if selected(pick("n")); asked.Limit != "" || asked.Repeated != 0 {
		t.Fatalf("mistyped marks: asked %+v", asked)
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
