package acpclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	acp "github.com/coder/acp-go-sdk"
)

// ErrSessionLocked matches a session/resume or session/load the agent
// refused because another process writes the session. The error's text is
// the agent's: "session <id> is open in another gobble process (pid N)".
var ErrSessionLocked = errors.New("acpclient: the session is open in another process")

// ErrInvalidSession matches a request the agent refused with invalid
// params: an unknown, invalid or duplicate session id, or an unreadable
// session.
var ErrInvalidSession = errors.New("acpclient: invalid session")

// agentError is the agent's refusal with its reason as the text.
type agentError struct {
	reason string
	kind   error
}

func (e *agentError) Error() string        { return e.reason }
func (e *agentError) Is(target error) bool { return target == e.kind }

// sessionError maps a session request's JSON-RPC error: invalid params to
// ErrInvalidSession, and an invalid request naming another gobble process to
// ErrSessionLocked, each with the agent's reason as its text.
func sessionError(method string, err error) error {
	re, ok := errors.AsType[*acp.RequestError](err)
	if !ok {
		return fmt.Errorf("%s: %w", method, err)
	}
	reason := re.Message
	if d, ok := re.Data.(map[string]any); ok {
		if r, ok := d["reason"].(string); ok && r != "" {
			reason = r
		}
	}
	switch {
	case re.Code == -32600 && strings.Contains(reason, "is open in another gobble process"):
		return &agentError{reason: reason, kind: ErrSessionLocked}
	case re.Code == -32602:
		return &agentError{reason: reason, kind: ErrInvalidSession}
	}
	return fmt.Errorf("%s: %w", method, err)
}

// NewSessionOptions are what session/new's _meta.gobble asks of gobble's
// agent: a chosen id, a session to fork, and a name (0002-PLAN Phase 4).
type NewSessionOptions struct {
	ID, ForkFrom, Name string
}

func (o NewSessionOptions) meta() map[string]any {
	g := map[string]any{}
	for k, v := range map[string]string{"sessionId": o.ID, "forkFrom": o.ForkFrom, "name": o.Name} {
		if v != "" {
			g[k] = v
		}
	}
	if len(g) == 0 {
		return nil
	}
	return map[string]any{"gobble": g}
}

// NewSessionWith is NewSession with options.
func (c *Conn) NewSessionWith(ctx context.Context, cwd string, opts NewSessionOptions, on func(Update)) (*Session, error) {
	c.newMu.Lock()
	defer c.newMu.Unlock()
	c.handler.setPending(on)
	defer c.handler.setPending(nil)
	resp, err := c.conn.NewSession(ctx, acp.NewSessionRequest{Cwd: cwd, McpServers: []acp.McpServer{}, Meta: opts.meta()})
	if err != nil {
		return nil, sessionError("session/new", err)
	}
	c.handler.add(resp.SessionId, on)
	return &Session{id: resp.SessionId, c: c}, nil
}

// Commands are the slash commands the agent last advertised for the
// session, or nil before any.
func (s *Session) Commands() []Command { return s.c.handler.commandsOf(s.id) }

// Resumed is what the agent says of a session it resumed: its title and
// the number of user and assistant messages on its path, from
// _meta.gobble.
type Resumed struct {
	Title    string
	Messages int
}

// ResumeSession continues a stored session, with no replay. on receives its
// updates from then on.
func (c *Conn) ResumeSession(ctx context.Context, id, cwd string, on func(Update)) (*Session, Resumed, error) {
	sid := acp.SessionId(id)
	c.handler.add(sid, on)
	resp, err := c.conn.ResumeSession(ctx, acp.ResumeSessionRequest{SessionId: sid, Cwd: cwd})
	if err != nil {
		c.handler.remove(sid)
		return nil, Resumed{}, sessionError("session/resume", err)
	}
	var r Resumed
	if g, ok := resp.Meta["gobble"].(map[string]any); ok {
		r.Title, _ = g["title"].(string) //nolint:errcheck // absent is none
		if n, ok := g["messages"].(float64); ok {
			r.Messages = int(n)
		}
	}
	return &Session{id: sid, c: c}, r, nil
}

// SessionInfo is one row of ListSessions.
type SessionInfo struct {
	ID, Cwd, Title string
	// UpdatedAt is the last activity, or zero for a live session with no
	// file yet.
	UpdatedAt time.Time
}

// ListSessions lists the agent's sessions, newest first; cwd "" lists
// every directory's.
func (c *Conn) ListSessions(ctx context.Context, cwd string) ([]SessionInfo, error) {
	req := acp.ListSessionsRequest{}
	if cwd != "" {
		req.Cwd = &cwd
	}
	resp, err := c.conn.ListSessions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("session/list: %w", err)
	}
	out := make([]SessionInfo, 0, len(resp.Sessions))
	for _, s := range resp.Sessions {
		info := SessionInfo{ID: string(s.SessionId), Cwd: s.Cwd}
		if s.Title != nil {
			info.Title = *s.Title
		}
		if s.UpdatedAt != nil {
			if t, err := time.Parse(time.RFC3339, *s.UpdatedAt); err == nil {
				info.UpdatedAt = t
			}
		}
		out = append(out, info)
	}
	return out, nil
}
