package acpserver

import (
	"context"
	"errors"
	"sync"
	"uuid"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// keyReason and keySessionID are keys of an error's JSON-RPC data.
const (
	keyReason    = "reason"
	keySessionID = "sessionId"
)

// ErrNoProvider is a prompt's error when Options.Provider is nil.
var ErrNoProvider = errors.New("acpserver: no model provider configured")

// Options configure an Agent.
type Options struct {
	// Version is agentInfo.version: SemVer on every build (0008-MADR D19
	// item 1), as internal/buildinfo reports it.
	Version string
	// NewID makes session ids. The default is a UUIDv7.
	NewID func() string
	// Provider builds the model provider. It is called on the first prompt,
	// so initialize and session/new need no credential and make no network
	// call (0008-MADR D19 item 6). An error matching llm.ErrAuth answers the
	// prompt with auth_required.
	Provider func() (llm.Provider, error)
	// Tools are the agent's tools. Nil is builtin.Tools(); an empty, non-nil
	// slice is none.
	Tools []tool.Tool
}

// Agent is gobble's acp.Agent.
type Agent struct {
	version  string
	newID    func() string
	provider func() (llm.Provider, error)
	tools    []tool.Tool

	mu       sync.Mutex
	conn     *acp.AgentSideConnection
	sessions map[acp.SessionId]*session
	model    llm.Provider // built on the first prompt
}

// session is one session's state: its directory, its history in memory
// (0002-PLAN Phase 4 persists it), and the cancel of its running turn.
type session struct {
	cwd     string
	history []llm.Message
	cancel  context.CancelFunc
}

var _ acp.Agent = (*Agent)(nil)

// New returns an Agent. Bind it to its connection with SetConnection before
// the first request, so it can send session updates.
func New(opts Options) *Agent {
	a := &Agent{version: opts.Version, newID: opts.NewID, provider: opts.Provider, tools: opts.Tools, sessions: map[acp.SessionId]*session{}}
	if a.newID == nil {
		a.newID = func() string { return uuid.NewV7().String() }
	}
	if a.tools == nil {
		a.tools = builtin.Tools()
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
	a.sessions[id] = &session{cwd: p.Cwd}
	return acp.NewSessionResponse{SessionId: id}, nil
}

// Prompt runs one turn of the session: the model, and the tools it asks
// for, streamed as session updates. One prompt runs at a time per session.
func (a *Agent) Prompt(ctx context.Context, p acp.PromptRequest) (acp.PromptResponse, error) {
	a.mu.Lock()
	s, ok := a.sessions[p.SessionId]
	conn := a.conn
	busy := ok && s.cancel != nil
	a.mu.Unlock()
	switch {
	case !ok:
		return acp.PromptResponse{}, acp.NewInvalidParams(map[string]any{keySessionID: p.SessionId, keyReason: "unknown session"})
	case busy:
		return acp.PromptResponse{}, acp.NewInvalidRequest(map[string]any{keySessionID: p.SessionId, keyReason: "a prompt is already running"})
	}
	model, err := a.modelProvider()
	if err != nil {
		return acp.PromptResponse{}, promptError(err)
	}
	turnCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	a.mu.Lock()
	s.cancel = cancel
	history := s.history
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		s.cancel = nil
		a.mu.Unlock()
	}()

	ag := agent.New(agent.Config{
		Provider: model,
		Tools:    a.tools,
		System:   systemPrompt(s.cwd, a.tools),
		Policy:   &acpPolicy{conn: conn, session: p.SessionId},
	})
	out := newUpdates(ctx, conn, p.SessionId)
	var last llm.Usage
	for ev, err := range ag.Run(turnCtx, tool.Env{Cwd: s.cwd}, history, userMessage(p.Prompt)) {
		if err != nil {
			out.flush()
			return acp.PromptResponse{}, promptError(err)
		}
		out.event(ev, &last)
		if end, ok := ev.(agent.End); ok {
			a.mu.Lock()
			s.history = append(s.history, end.Messages...)
			a.mu.Unlock()
			out.send(usageUpdate(last))
			if out.err != nil {
				return acp.PromptResponse{}, out.err
			}
			return acp.PromptResponse{StopReason: acp.StopReason(end.StopReason), Usage: acpUsage(end.Usage)}, nil
		}
	}
	return acp.PromptResponse{StopReason: acp.StopReasonCancelled}, nil
}

// modelProvider builds the provider on first use and keeps it.
func (a *Agent) modelProvider() (llm.Provider, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.model != nil {
		return a.model, nil
	}
	if a.provider == nil {
		return nil, ErrNoProvider
	}
	p, err := a.provider()
	if err != nil {
		return nil, err
	}
	a.model = p
	return p, nil
}

// userMessage is the prompt's text blocks as one user message. Images are
// not advertised, so none arrive.
func userMessage(blocks []acp.ContentBlock) llm.Message {
	m := llm.Message{Role: llm.RoleUser}
	for _, b := range blocks {
		if b.Text != nil {
			m.Content = append(m.Content, llm.Content{Type: llm.ContentText, Text: b.Text.Text})
		}
	}
	return m
}

// Cancel stops the session's running turn. The SDK also cancels the
// prompt's context; the turn then ends with the cancelled stop reason.
func (a *Agent) Cancel(_ context.Context, p acp.CancelNotification) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[p.SessionId]; ok && s.cancel != nil {
		s.cancel()
	}
	return nil
}

// CloseSession forgets the session.
func (a *Agent) CloseSession(_ context.Context, p acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.sessions[p.SessionId]; !ok {
		return acp.CloseSessionResponse{}, acp.NewInvalidParams(map[string]any{keySessionID: p.SessionId, keyReason: "unknown session"})
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
