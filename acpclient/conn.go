package acpclient

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	acp "github.com/coder/acp-go-sdk"
)

// closeTimeout bounds how long Close waits for the agent to stop.
const closeTimeout = 5 * time.Second

// ErrImagesUnsupported is returned by Prompt when the prompt carries images
// and the agent did not advertise promptCapabilities.image. Nothing is sent:
// an image is never dropped silently (0008-MADR D19 item 1).
var ErrImagesUnsupported = errors.New("the agent does not accept images")

// ServeFunc runs an agent over in and out until in reaches end of file.
// acpserver.Serve, wrapped in a closure, is one.
type ServeFunc func(ctx context.Context, in io.Reader, out io.Writer) error

// Options configure Start.
type Options struct {
	// Name and Version are sent as clientInfo.
	Name, Version string
	// Logger receives the connection's diagnostics. Nil discards them.
	Logger *slog.Logger
	// OnDrop is called when an inbound notification is dropped because the
	// queue is full. The connection keeps going (0004-MADR, amendment of
	// 2026-10-01).
	OnDrop func(method string, total uint64)
	// MaxQueued is the inbound notification queue's size. Zero keeps the
	// SDK's default.
	MaxQueued int
}

// AgentInfo is what the agent said about itself in initialize.
type AgentInfo struct {
	Name, Title, Version string
	// Image and LoadSession are the capabilities the client acts on.
	Image, LoadSession bool
}

// Conn is a client connection to an agent running in this process.
type Conn struct {
	conn    *acp.ClientSideConnection
	agent   AgentInfo
	toAgent *io.PipeWriter
	served  chan error
	handler *handler
	newMu   sync.Mutex // serialises NewSession
}

// Start runs serve over a pair of pipes, connects to it and sends
// initialize. The agent runs until Close, whatever happens to ctx; ctx
// bounds only the handshake. Start fails if the agent speaks a protocol
// other than 1.
func Start(ctx context.Context, serve ServeFunc, opts Options) (*Conn, error) {
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	c := &Conn{toAgent: toAgentW, served: make(chan error, 1), handler: &handler{sessions: map[acp.SessionId]func(Update){}}}
	go func() {
		err := serve(context.WithoutCancel(ctx), toAgentR, toClientW)
		toClientW.CloseWithError(cmp.Or(err, io.EOF)) //nolint:errcheck,gosec // CloseWithError always returns nil
		c.served <- err
	}()
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	// The logger goes in at construction: the connection starts reading
	// inside the constructor, so a later SetLogger would race with it.
	connOpts := []acp.ConnectionOption{
		acp.WithLogger(logger),
		acp.WithNotificationOverflowPolicy(acp.OverflowDropNewest),
	}
	if opts.OnDrop != nil {
		connOpts = append(connOpts, acp.WithNotificationDropHandler(opts.OnDrop))
	}
	if opts.MaxQueued > 0 {
		connOpts = append(connOpts, acp.WithMaxQueuedNotifications(opts.MaxQueued))
	}
	c.conn = acp.NewClientSideConnection(c.handler, toAgentW, toClientR, connOpts...)
	resp, err := c.conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientInfo:      &acp.Implementation{Name: opts.Name, Version: opts.Version},
	})
	if err == nil && resp.ProtocolVersion != acp.ProtocolVersionNumber {
		err = fmt.Errorf("agent speaks ACP protocol %d, want %d", resp.ProtocolVersion, acp.ProtocolVersionNumber)
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("initialize: %w", err), c.Close())
	}
	c.agent = AgentInfo{
		Image:       resp.AgentCapabilities.PromptCapabilities.Image,
		LoadSession: resp.AgentCapabilities.LoadSession,
	}
	if info := resp.AgentInfo; info != nil {
		c.agent.Name, c.agent.Version = info.Name, info.Version
		if info.Title != nil {
			c.agent.Title = *info.Title
		}
	}
	return c, nil
}

// Agent is the agent's initialize answer.
func (c *Conn) Agent() AgentInfo { return c.agent }

// Dropped is the number of inbound notifications dropped so far.
func (c *Conn) Dropped() uint64 { return c.conn.DroppedNotifications() }

// Close ends the agent's input and waits for the agent to stop, then for
// the connection. It returns the agent's error, if any.
func (c *Conn) Close() error {
	if err := c.toAgent.Close(); err != nil {
		return fmt.Errorf("close agent input: %w", err)
	}
	deadline := time.NewTimer(closeTimeout)
	defer deadline.Stop()
	var served error
	select {
	case served = <-c.served:
	case <-deadline.C:
		return fmt.Errorf("the agent did not stop within %v of its input closing", closeTimeout)
	}
	select {
	case <-c.conn.Done():
	case <-deadline.C:
		return fmt.Errorf("the connection did not stop within %v", closeTimeout)
	}
	if served != nil {
		return fmt.Errorf("agent: %w", served)
	}
	return nil
}

// Session is one ACP session.
type Session struct {
	id acp.SessionId
	c  *Conn
}

// NewSession opens a session in cwd. on receives the session's updates,
// in order, on the connection's notification goroutine; it must not block
// for long. Updates the agent sends before its session/new answer, such
// as available_commands_update (0002-PLAN Phase 6), reach on too. Calls
// are serialised, so such an update can only belong to this session.
func (c *Conn) NewSession(ctx context.Context, cwd string, on func(Update)) (*Session, error) {
	c.newMu.Lock()
	defer c.newMu.Unlock()
	c.handler.setPending(on)
	defer c.handler.setPending(nil)
	resp, err := c.conn.NewSession(ctx, acp.NewSessionRequest{Cwd: cwd, McpServers: []acp.McpServer{}})
	if err != nil {
		return nil, fmt.Errorf("session/new: %w", err)
	}
	c.handler.add(resp.SessionId, on)
	return &Session{id: resp.SessionId, c: c}, nil
}

// ID is the session id.
func (s *Session) ID() string { return string(s.id) }

// Prompt is one user turn: text, and images as raw bytes.
type Prompt struct {
	Text   string
	Images []Image
}

// Image is an image content block's data.
type Image struct {
	MIME string
	Data []byte
}

// Result is how a turn ended.
type Result struct {
	StopReason string
	// Usage is the response's token usage, when the agent reports it.
	Usage *Usage
}

// Usage is ACP's per-turn token usage.
type Usage struct {
	InputTokens       int  `json:"inputTokens"`
	OutputTokens      int  `json:"outputTokens"`
	TotalTokens       int  `json:"totalTokens"`
	ThoughtTokens     *int `json:"thoughtTokens,omitempty"`
	CachedReadTokens  *int `json:"cachedReadTokens,omitempty"`
	CachedWriteTokens *int `json:"cachedWriteTokens,omitempty"`
}

// Prompt sends one turn and waits for it to end. Updates arrive through the
// session's callback before Prompt returns.
func (s *Session) Prompt(ctx context.Context, p Prompt) (Result, error) {
	if len(p.Images) > 0 && !s.c.agent.Image {
		return Result{}, ErrImagesUnsupported
	}
	var blocks []acp.ContentBlock
	if p.Text != "" {
		blocks = append(blocks, acp.TextBlock(p.Text))
	}
	for _, img := range p.Images {
		blocks = append(blocks, acp.ImageBlock(base64.StdEncoding.EncodeToString(img.Data), img.MIME))
	}
	resp, err := s.c.conn.Prompt(ctx, acp.PromptRequest{SessionId: s.id, Prompt: blocks})
	if err != nil {
		return Result{}, fmt.Errorf("session/prompt: %w", err)
	}
	r := Result{StopReason: string(resp.StopReason)}
	if u := resp.Usage; u != nil {
		r.Usage = &Usage{
			InputTokens: u.InputTokens, OutputTokens: u.OutputTokens, TotalTokens: u.TotalTokens,
			ThoughtTokens: u.ThoughtTokens, CachedReadTokens: u.CachedReadTokens, CachedWriteTokens: u.CachedWriteTokens,
		}
	}
	return r, nil
}

// Cancel sends session/cancel. The running Prompt then ends with the
// cancelled stop reason.
func (s *Session) Cancel(ctx context.Context) error {
	if err := s.c.conn.Cancel(ctx, acp.CancelNotification{SessionId: s.id}); err != nil {
		return fmt.Errorf("session/cancel: %w", err)
	}
	return nil
}

// Close sends session/close and stops delivering the session's updates.
func (s *Session) Close(ctx context.Context) error {
	defer s.c.handler.remove(s.id)
	if _, err := s.c.conn.CloseSession(ctx, acp.CloseSessionRequest{SessionId: s.id}); err != nil {
		return fmt.Errorf("session/close: %w", err)
	}
	return nil
}

// handler is the client side of the connection: it routes session updates
// to their session and declines what the client does not offer.
type handler struct {
	mu       sync.Mutex
	sessions map[acp.SessionId]func(Update)
	// pending receives updates for an unknown session while a session/new
	// is in flight.
	pending func(Update)
}

var _ acp.Client = (*handler)(nil)

func (h *handler) add(id acp.SessionId, on func(Update)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions[id] = on
}

func (h *handler) setPending(on func(Update)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending = on
}

func (h *handler) remove(id acp.SessionId) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, id)
}

// SessionUpdate delivers the update to its session's callback. An update
// for an unknown session goes to the session being created, if any, and
// is otherwise ignored.
func (h *handler) SessionUpdate(_ context.Context, n acp.SessionNotification) error {
	h.mu.Lock()
	on := h.sessions[n.SessionId]
	if on == nil && h.pending != nil {
		on = h.pending
		h.sessions[n.SessionId] = on
	}
	h.mu.Unlock()
	if on == nil {
		return nil
	}
	u, err := translate(n)
	if err != nil {
		return err
	}
	on(u)
	return nil
}

// RequestPermission answers cancelled: the CLI's permission prompt arrives
// in 0002-PLAN Phase 6.
func (*handler) RequestPermission(context.Context, acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}, nil
}

// The client advertises no file-system or terminal capability, so these
// answer "method not found".

// ReadTextFile is not offered.
func (*handler) ReadTextFile(context.Context, acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, acp.NewMethodNotFound(acp.ClientMethodFsReadTextFile)
}

// WriteTextFile is not offered.
func (*handler) WriteTextFile(context.Context, acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, acp.NewMethodNotFound(acp.ClientMethodFsWriteTextFile)
}

// CreateTerminal is not offered.
func (*handler) CreateTerminal(context.Context, acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalCreate)
}

// KillTerminal is not offered.
func (*handler) KillTerminal(context.Context, acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalKill)
}

// TerminalOutput is not offered.
func (*handler) TerminalOutput(context.Context, acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalOutput)
}

// ReleaseTerminal is not offered.
func (*handler) ReleaseTerminal(context.Context, acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalRelease)
}

// WaitForTerminalExit is not offered.
func (*handler) WaitForTerminalExit(context.Context, acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalWaitForExit)
}
