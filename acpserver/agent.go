package acpserver

import (
	"context"
	"strings"
	"sync"
	"uuid"

	acp "github.com/coder/acp-go-sdk"
)

// Options configure an Agent.
type Options struct {
	// Version is agentInfo.version: SemVer on every build (0008-MADR D19
	// item 1), as internal/buildinfo reports it.
	Version string
	// NewID makes session ids. The default is a UUIDv7.
	NewID func() string
}

// Agent is gobble's acp.Agent.
type Agent struct {
	version string
	newID   func() string

	mu       sync.Mutex
	conn     *acp.AgentSideConnection
	sessions map[acp.SessionId]session
}

type session struct {
	cwd string
}

var _ acp.Agent = (*Agent)(nil)

// New returns an Agent. Bind it to its connection with SetConnection before
// the first request, so it can send session updates.
func New(opts Options) *Agent {
	a := &Agent{version: opts.Version, newID: opts.NewID, sessions: map[acp.SessionId]session{}}
	if a.newID == nil {
		a.newID = func() string { return uuid.NewV7().String() }
	}
	return a
}

// SetConnection binds the agent to the connection it answers on.
func (a *Agent) SetConnection(c *acp.AgentSideConnection) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conn = c
}

// Capabilities is what this phase implements, and nothing more: closing a
// session. Loading, listing and resuming sessions, MCP, images, audio and
// embedded context are turned on by the phases that implement them.
func Capabilities() acp.AgentCapabilities {
	return acp.AgentCapabilities{
		SessionCapabilities: acp.SessionCapabilities{Close: &acp.SessionCloseCapabilities{}},
	}
}

// Initialize answers the handshake. It makes no network call (0008-MADR
// D19 item 6).
func (a *Agent) Initialize(context.Context, acp.InitializeRequest) (acp.InitializeResponse, error) {
	title := "gobble"
	return acp.InitializeResponse{
		ProtocolVersion:   acp.ProtocolVersionNumber,
		AgentInfo:         &acp.Implementation{Name: "gobble", Title: &title, Version: a.version},
		AgentCapabilities: Capabilities(),
		AuthMethods:       []acp.AuthMethod{},
	}, nil
}

// NewSession opens a session in the client's working directory.
func (a *Agent) NewSession(_ context.Context, p acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	id := acp.SessionId(a.newID())
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions[id] = session{cwd: p.Cwd}
	return acp.NewSessionResponse{SessionId: id}, nil
}

// Prompt echoes the prompt's text as one agent message chunk and ends the
// turn. The model turn arrives in 0002-PLAN Phase 3.
func (a *Agent) Prompt(ctx context.Context, p acp.PromptRequest) (acp.PromptResponse, error) {
	a.mu.Lock()
	_, ok := a.sessions[p.SessionId]
	conn := a.conn
	a.mu.Unlock()
	if !ok {
		return acp.PromptResponse{}, acp.NewInvalidParams(map[string]any{"sessionId": p.SessionId, "reason": "unknown session"})
	}
	var parts []string
	for _, b := range p.Prompt {
		if b.Text != nil {
			parts = append(parts, b.Text.Text)
		}
	}
	if conn != nil {
		if err := conn.SessionUpdate(ctx, acp.SessionNotification{
			SessionId: p.SessionId,
			Update:    acp.UpdateAgentMessageText(strings.Join(parts, "\n\n")),
		}); err != nil {
			return acp.PromptResponse{}, err
		}
	}
	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
}

// Cancel has nothing to stop: a prompt finishes before it returns.
func (*Agent) Cancel(context.Context, acp.CancelNotification) error { return nil }

// CloseSession forgets the session.
func (a *Agent) CloseSession(_ context.Context, p acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.sessions[p.SessionId]; !ok {
		return acp.CloseSessionResponse{}, acp.NewInvalidParams(map[string]any{"sessionId": p.SessionId, "reason": "unknown session"})
	}
	delete(a.sessions, p.SessionId)
	return acp.CloseSessionResponse{}, nil
}

// The methods below are not implemented in this phase and are not
// advertised; each answers "method not found".

// Authenticate is not offered: authMethods is empty.
func (*Agent) Authenticate(context.Context, acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, acp.NewMethodNotFound(acp.AgentMethodAuthenticate)
}

// Logout is not offered.
func (*Agent) Logout(context.Context, acp.LogoutRequest) (acp.LogoutResponse, error) {
	return acp.LogoutResponse{}, acp.NewMethodNotFound(acp.AgentMethodLogout)
}

// ListSessions arrives with sessions (0002-PLAN Phase 4).
func (*Agent) ListSessions(context.Context, acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionList)
}

// ResumeSession arrives with sessions (0002-PLAN Phase 4).
func (*Agent) ResumeSession(context.Context, acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionResume)
}

// SetSessionMode arrives with modes (0002-PLAN Phase 6).
func (*Agent) SetSessionMode(context.Context, acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionSetMode)
}

// SetSessionConfigOption arrives with model and thinking options (0002-PLAN
// Phase 6).
func (*Agent) SetSessionConfigOption(context.Context, acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionSetConfigOption)
}
