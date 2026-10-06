package acptest

import (
	"cmp"
	"context"
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"

	acp "github.com/coder/acp-go-sdk"
)

// Turn is a scripted answer to one session/prompt. Updates are ACP
// SessionUpdate objects as JSON, such as
// {"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"hi"}},
// so a test needs no SDK import to write one.
type Turn struct {
	// Updates are sent as session/update notifications, in order, before
	// the turn ends.
	Updates []string
	// StopReason ends the turn. Empty means end_turn.
	StopReason string
	// Usage is the response's usage object as JSON, or empty for none.
	Usage string
	// ErrCode, when not zero, answers the prompt with this JSON-RPC error
	// after the updates are sent.
	ErrCode    int
	ErrMessage string
	// WaitCancel holds the turn open, after its updates, until
	// session/cancel arrives; it then ends with "cancelled".
	WaitCancel bool
}

// ScriptAgent is an acp.Agent that answers each session/prompt with the
// next Turn and records what it was sent. Serve runs it on a stream pair,
// as acpserver.Serve runs gobble's agent.
type ScriptAgent struct {
	Turns []Turn
	// Image is advertised as promptCapabilities.image.
	Image bool
	// SessionUpdates are sent, as JSON SessionUpdate objects, before each
	// session/new answer, as an agent sends available_commands_update.
	SessionUpdates []string

	mu       sync.Mutex
	conn     *acp.AgentSideConnection
	next     int
	sessions int
	methods  []string
	prompts  []string
	cwds     []string
	cancels  int
	waiting  map[acp.SessionId]chan struct{}
}

var _ acp.Agent = (*ScriptAgent)(nil)

// Serve runs the agent over in and out until in reaches end of file, when
// it returns nil, or ctx is done, when it returns context.Cause(ctx).
func (a *ScriptAgent) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	conn := acp.NewAgentSideConnection(a, out, in, acp.WithLogger(slog.New(slog.DiscardHandler)))
	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()
	select {
	case <-conn.Done():
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// Methods are the agent methods received, in order.
func (a *ScriptAgent) Methods() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.methods)
}

// Prompts are the session/prompt params received, each canonical JSON.
func (a *ScriptAgent) Prompts() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.prompts)
}

// Cwds are the cwd of each session/new received.
func (a *ScriptAgent) Cwds() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.cwds)
}

// Cancels is the number of session/cancel notifications received.
func (a *ScriptAgent) Cancels() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cancels
}

func (a *ScriptAgent) record(method string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.methods = append(a.methods, method)
}

// Initialize advertises protocol 1, session close, and Image.
func (a *ScriptAgent) Initialize(context.Context, acp.InitializeRequest) (acp.InitializeResponse, error) {
	a.record(acp.AgentMethodInitialize)
	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersionNumber,
		AgentInfo:       &acp.Implementation{Name: "script", Version: "0.0.0"},
		AgentCapabilities: acp.AgentCapabilities{
			PromptCapabilities:  acp.PromptCapabilities{Image: a.Image},
			SessionCapabilities: acp.SessionCapabilities{Close: &acp.SessionCloseCapabilities{}},
		},
		AuthMethods: []acp.AuthMethod{},
	}, nil
}

// NewSession records the cwd, sends SessionUpdates, and returns s1, s2, ….
func (a *ScriptAgent) NewSession(ctx context.Context, p acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	a.mu.Lock()
	a.methods = append(a.methods, acp.AgentMethodSessionNew)
	a.cwds = append(a.cwds, p.Cwd)
	a.sessions++
	id := acp.SessionId(fmt.Sprintf("s%d", a.sessions))
	conn := a.conn
	a.mu.Unlock()
	if err := sendUpdates(ctx, conn, id, a.SessionUpdates); err != nil {
		return acp.NewSessionResponse{}, err
	}
	return acp.NewSessionResponse{SessionId: id}, nil
}

// sendUpdates sends each JSON SessionUpdate to the session.
func sendUpdates(ctx context.Context, conn *acp.AgentSideConnection, id acp.SessionId, updates []string) error {
	for i, u := range updates {
		var update acp.SessionUpdate
		if err := json.Unmarshal([]byte(u), &update); err != nil {
			return acp.NewInternalError(map[string]any{"update": i, "error": err.Error()})
		}
		if err := conn.SessionUpdate(ctx, acp.SessionNotification{SessionId: id, Update: update}); err != nil {
			return err
		}
	}
	return nil
}

// Prompt plays the next Turn.
func (a *ScriptAgent) Prompt(ctx context.Context, p acp.PromptRequest) (acp.PromptResponse, error) {
	turn, conn, wait, err := a.startTurn(p)
	if err != nil {
		return acp.PromptResponse{}, err
	}
	if err := sendUpdates(ctx, conn, p.SessionId, turn.Updates); err != nil {
		return acp.PromptResponse{}, err
	}
	if turn.ErrCode != 0 {
		return acp.PromptResponse{}, &acp.RequestError{Code: turn.ErrCode, Message: turn.ErrMessage}
	}
	resp := acp.PromptResponse{StopReason: acp.StopReason(cmp.Or(turn.StopReason, string(acp.StopReasonEndTurn)))}
	if turn.Usage != "" {
		resp.Usage = &acp.Usage{}
		if err := json.Unmarshal([]byte(turn.Usage), resp.Usage); err != nil {
			return acp.PromptResponse{}, acp.NewInternalError(map[string]any{"usage": err.Error()})
		}
	}
	if wait != nil {
		// The SDK cancels the prompt's context when session/cancel arrives,
		// and an agent answers that with the cancelled stop reason.
		select {
		case <-wait:
		case <-ctx.Done():
		}
		resp.StopReason = acp.StopReasonCancelled
	}
	return resp, nil
}

// startTurn records the prompt and takes the next Turn.
func (a *ScriptAgent) startTurn(p acp.PromptRequest) (Turn, *acp.AgentSideConnection, chan struct{}, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return Turn{}, nil, nil, err
	}
	v := jsontext.Value(b)
	if err := v.Canonicalize(); err != nil {
		return Turn{}, nil, nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.methods = append(a.methods, acp.AgentMethodSessionPrompt)
	a.prompts = append(a.prompts, string(v))
	if a.next >= len(a.Turns) {
		return Turn{}, nil, nil, acp.NewInternalError(map[string]any{"script": fmt.Sprintf("acptest: script exhausted after %d turns", len(a.Turns))})
	}
	turn := a.Turns[a.next]
	a.next++
	var wait chan struct{}
	if turn.WaitCancel {
		wait = make(chan struct{})
		if a.waiting == nil {
			a.waiting = map[acp.SessionId]chan struct{}{}
		}
		a.waiting[p.SessionId] = wait
	}
	return turn, a.conn, wait, nil
}

// Cancel releases a WaitCancel turn of the session.
func (a *ScriptAgent) Cancel(_ context.Context, p acp.CancelNotification) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.methods = append(a.methods, acp.AgentMethodSessionCancel)
	a.cancels++
	if w, ok := a.waiting[p.SessionId]; ok {
		close(w)
		delete(a.waiting, p.SessionId)
	}
	return nil
}

// CloseSession succeeds.
func (a *ScriptAgent) CloseSession(context.Context, acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	a.record(acp.AgentMethodSessionClose)
	return acp.CloseSessionResponse{}, nil
}

// Authenticate is not offered.
func (*ScriptAgent) Authenticate(context.Context, acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, acp.NewMethodNotFound(acp.AgentMethodAuthenticate)
}

// Logout is not offered.
func (*ScriptAgent) Logout(context.Context, acp.LogoutRequest) (acp.LogoutResponse, error) {
	return acp.LogoutResponse{}, acp.NewMethodNotFound(acp.AgentMethodLogout)
}

// ListSessions is not offered.
func (*ScriptAgent) ListSessions(context.Context, acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionList)
}

// ResumeSession is not offered.
func (*ScriptAgent) ResumeSession(context.Context, acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionResume)
}

// SetSessionMode is not offered.
func (*ScriptAgent) SetSessionMode(context.Context, acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionSetMode)
}

// SetSessionConfigOption is not offered.
func (*ScriptAgent) SetSessionConfigOption(context.Context, acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionSetConfigOption)
}
