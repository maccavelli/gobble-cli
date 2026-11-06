package acpclient

import (
	"context"
	"encoding/base64"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"sync"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/acp-go-sdk2/agentproc"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/jsonx"
)

// closeTimeout bounds Close as a whole; each of the runner's steps waits
// half of it (0035-MADR D12).
var closeTimeout = 5 * time.Second

// ErrImagesUnsupported is returned by Prompt when the prompt carries images
// and the agent did not advertise promptCapabilities.image. Nothing is sent:
// an image is never dropped silently (0008-MADR D19 item 1).
var ErrImagesUnsupported = errors.New("the agent does not accept images")

// ErrAuthRequired matches a prompt the agent refused with ACP's
// auth_required error (-32000): it has no usable credential.
var ErrAuthRequired = errors.New("the agent needs a model credential")

// ServeFunc runs an agent over in and out until in reaches end of file.
// acpserver.Serve, wrapped in a closure, is one.
type ServeFunc func(ctx context.Context, in io.Reader, out io.Writer) error

// Options configure Start.
type Options struct {
	// Name and Version are sent as clientInfo.
	Name, Version string
	// Logger receives the connection's diagnostics. Nil discards them.
	Logger *slog.Logger
	// MaxQueued is how many inbound notifications wait before the
	// connection stops reading the agent until a handler returns: the agent
	// is paused, and nothing is dropped (0040-MADR D1, D3). Zero keeps the
	// SDK's default.
	MaxQueued int
	// Permission answers the agent's session/request_permission. Nil
	// declines every request, as a client with no one to ask must.
	Permission func(ctx context.Context, r PermissionRequest) PermissionAnswer
	// Params encodes Update.Params, for a client that passes the stream
	// through. Without it a session/update's Params is nil, and no update
	// is encoded for nothing (0039-MADR D16).
	Params bool
}

// metaGobble is gobble's key in an ACP _meta.
const metaGobble = "gobble"

// PermissionRequest is the agent asking to run a tool call.
//
// FullTitle is the call's whole description, the request's title when
// the agent sends none; Saves are the patterns an "always" saves, and
// Hides says a "never" hides the tool (gobble-cli 0029-MADR D4).
//
// Limit, Repeated and OnceOnly are gobble's marks, from the request's
// _meta.gobble: the limit that asked, which no "always" and no --yes may
// approve; a repeated call's run, whose question asks whether the turn may
// go on; and why a question offers "once" alone (gobble-cli 0005-PLAN F6-2
// decisions 5 and 8, F6-2a's deviations 1 and 2, F6-2b decision A).
// Another agent sends none.
type PermissionRequest struct {
	SessionID string
	Tool      ToolCall
	Options   []PermissionOption
	Limit     string
	Repeated  int
	OnceOnly  string
	FullTitle string
	Saves     []string
	Hides     bool
}

// PermissionOption is one answer the agent offers. Kind is ACP's:
// allow_once, allow_always, reject_once or reject_always.
type PermissionOption struct {
	ID, Name, Kind string
}

// PermissionAnswer is the option chosen; an empty OptionID cancels.
// Feedback, with a reject, is the person's reason, which the answer's
// _meta.gobble.feedback carries to the model (gobble-cli 0005-PLAN F6-2
// decision 11).
type PermissionAnswer struct {
	OptionID string
	Feedback string
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
	proc    *agentproc.Process
	handler *handler
	newMu   sync.Mutex // serialises NewSession
}

// Start runs serve in this process through agentproc.InProcess, connects
// to it and sends initialize. The agent runs until Close, whatever happens
// to ctx; ctx bounds only the handshake. Start fails if the agent speaks a
// protocol other than 1.
func Start(ctx context.Context, serve ServeFunc, opts Options) (*Conn, error) {
	c := &Conn{handler: &handler{sessions: map[acp.SessionId]func(Update){},
		commands: map[acp.SessionId][]Command{}, permission: opts.Permission, params: opts.Params}}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	// The logger goes in at construction: the connection starts reading
	// inside the constructor, so a later SetLogger would race with it.
	connOpts := []acp.ConnectionOption{
		acp.WithLogger(logger),
		// A full queue pauses the agent; nothing is dropped. The SDK's
		// default, named so that a test pins it (0040-MADR D1).
		acp.WithNotificationOverflowPolicy(acp.OverflowBlock),
	}
	if opts.MaxQueued > 0 {
		connOpts = append(connOpts, acp.WithMaxQueuedNotifications(opts.MaxQueued))
	}
	// serve runs on the runner's own context, which no caller's cancels.
	p, err := (&agentproc.InProcess{Serve: serve, Grace: closeTimeout / 2, ConnectionOptions: connOpts}).Start(c.handler)
	if err != nil {
		return nil, fmt.Errorf("start agent: %w", err)
	}
	c.proc, c.conn = p, p.Conn()
	resp, err := c.conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientInfo:      &acp.Implementation{Name: opts.Name, Version: opts.Version},
	})
	// The SDK negotiates the version; its refusal keeps gobble's words
	// (0019-MADR D7).
	if pve, ok := errors.AsType[*acp.ProtocolVersionError](err); ok {
		err = fmt.Errorf("agent speaks ACP protocol %d, want %d", pve.Answered, pve.Requested)
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("initialize: %w", err), c.closeFrom(ctx))
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

// Close stops the agent by the runner's steps: it ends the agent's input,
// then cancels its context, then abandons it. It then waits for the
// connection's handlers, so no update is handled after it returns
// (0016-MADR A1; 0031-MADR D5). closeTimeout bounds it, and the runner's
// kill half as long again. Its error wraps the agent's, and
// agentproc.ErrAbandoned, agentproc.ErrKilled and acp.ErrHandlersRunning
// read through it with errors.Is (0035-MADR D12).
func (c *Conn) Close() error { return c.closeFrom(context.Background()) }

// closeFrom is Close, its bound keeping base's values and never its
// cancellation: closeTimeout alone bounds it, whatever base says.
func (c *Conn) closeFrom(base context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(base), closeTimeout)
	defer cancel()
	err := c.proc.CloseContext(ctx)
	if err == nil {
		return nil
	}
	err = fmt.Errorf("agent: %w", err)
	if errors.Is(err, acp.ErrHandlersRunning) {
		return errors.Join(fmt.Errorf("the connection's handlers did not stop within %v of its close", closeTimeout), err)
	}
	return err
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
	return c.NewSessionWith(ctx, cwd, NewSessionOptions{}, on)
}

// ID is the session id.
func (s *Session) ID() string { return string(s.id) }

// Prompt is one user turn: text, and images as raw bytes.
type Prompt struct {
	// Context is text sent before Text, each its own text block marked
	// _meta.gobble.mentions false, which gobble's agent does not scan for
	// @path: input such as piped stdin and the files the CLI attached
	// (0005-PLAN F3-4 deviation 1).
	Context []string
	Text    string
	Images  []Image
}

// unscanned is a text block marked for gobble's agent not to scan for @path.
func unscanned(text string) (acp.ContentBlock, error) {
	b := acp.TextBlock(text)
	meta, err := jsonx.SetMember(nil, metaGobble, mentionsMeta{Mentions: false})
	if err != nil {
		return acp.ContentBlock{}, err
	}
	b.Text.Meta = meta
	return b, nil
}

// mentionsMeta is a text block's _meta.gobble (0039-MADR D15).
type mentionsMeta struct {
	Mentions bool `json:"mentions"`
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
	// SwitchTo names a session the agent wrote for this one to continue
	// in, from the response's _meta.gobble.switchTo: gobble's /fork and
	// /clone (0002-PLAN Phase 6).
	SwitchTo string
}

// toInt is an ACP count as an int: a count past an int's range is its
// maximum rather than wrapped (0014-PLAN C4).
func toInt(n uint64) int {
	if n > math.MaxInt {
		return math.MaxInt
	}
	return int(n)
}

// toInt64 is toInt for an int64 (0014-PLAN C4).
func toInt64(n uint64) int64 {
	if n > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(n)
}

func toIntPtr(n *uint64) *int {
	if n == nil {
		return nil
	}
	return new(toInt(*n))
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
	for _, c := range p.Context {
		if c != "" {
			b, err := unscanned(c)
			if err != nil {
				return Result{}, fmt.Errorf("session/prompt: %w", err)
			}
			blocks = append(blocks, b)
		}
	}
	if p.Text != "" {
		blocks = append(blocks, acp.TextBlock(p.Text))
	}
	for _, img := range p.Images {
		blocks = append(blocks, acp.ImageBlock(base64.StdEncoding.EncodeToString(img.Data), img.MIME))
	}
	resp, err := s.c.conn.Prompt(ctx, acp.PromptRequest{SessionId: s.id, Prompt: blocks})
	if re, ok := errors.AsType[*acp.RequestError](err); ok && re.Code == int(acp.ErrorCodeAuthenticationRequired) {
		return Result{}, fmt.Errorf("session/prompt: %w: %w", ErrAuthRequired, err)
	}
	if err != nil {
		return Result{}, fmt.Errorf("session/prompt: %w", err)
	}
	r := Result{StopReason: string(resp.StopReason)}
	if g, ok := jsonx.Member[map[string]any](resp.Meta, metaGobble); ok {
		r.SwitchTo, _ = jsonx.Member[string](g, "switchTo") //nolint:errcheck // absent is none
	}
	//lint:ignore SA1019 acp-go-sdk2 keeps this unstable-schema name under v1 (0035-MADR D6)
	if u := resp.Usage; u != nil { //nolint:staticcheck // as the lint:ignore above
		r.Usage = &Usage{
			InputTokens: toInt(u.InputTokens), OutputTokens: toInt(u.OutputTokens), TotalTokens: toInt(u.TotalTokens),
			ThoughtTokens: toIntPtr(u.ThoughtTokens), CachedReadTokens: toIntPtr(u.CachedReadTokens), CachedWriteTokens: toIntPtr(u.CachedWriteTokens),
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
	// ClientRefusals answers the file-system, terminal and elicitation
	// methods "method not found": the client advertises none of them
	// (0014-MADR D3; 0035-MADR D8).
	acp.ClientRefusals

	mu       sync.Mutex
	sessions map[acp.SessionId]func(Update)
	// pending receives updates for an unknown session while a session/new
	// is in flight.
	pending func(Update)
	// commands are each session's latest available_commands_update.
	commands   map[acp.SessionId][]Command
	permission func(context.Context, PermissionRequest) PermissionAnswer
	// params is Options.Params.
	params bool

	// ext routes gobble's _gobble/* notifications, registered on first use
	// (0039-MADR D12).
	extOnce sync.Once
	ext     acp.ExtensionMux
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
	u, err := translate(n, h.params)
	if err != nil {
		return err
	}
	if u.Kind == KindCommands {
		h.mu.Lock()
		h.commands[n.SessionId] = u.Commands
		h.mu.Unlock()
	}
	on(u)
	return nil
}

var _ acp.ExtensionMethodHandler = (*handler)(nil)

// HandleExtensionMethod serves gobble's _gobble/retry, _gobble/compaction
// and _gobble/personality notifications through an acp.ExtensionMux. Each
// is decoded by gobble with encoding/json/v2 and delivered to its
// session's callback, as KindRetry, KindCompaction and KindPersonality; a
// decode failure is the connection's to log, and reaches no callback. Any
// other extension notification is ignored, and a request is answered
// -32601 (0039-MADR D12).
func (h *handler) HandleExtensionMethod(ctx context.Context, method string, params jsontext.Value) (any, error) {
	h.extOnce.Do(h.registerExtensions)
	return h.ext.HandleExtensionMethod(ctx, method, params)
}

// registerExtensions registers the three notifications on h.ext.
func (h *handler) registerExtensions() {
	acp.Notify(&h.ext, acpext.MethodRetry, func(_ context.Context, params jsontext.Value) error {
		id, r, err := retryOf(params)
		if err != nil {
			return err
		}
		h.deliver(id, Update{Kind: KindRetry, Retry: r, Params: params})
		return nil
	})
	acp.Notify(&h.ext, acpext.MethodCompaction, func(_ context.Context, params jsontext.Value) error {
		id, c, err := compactionOf(params)
		if err != nil {
			return err
		}
		h.deliver(id, Update{Kind: KindCompaction, Compaction: c, Params: params})
		return nil
	})
	acp.Notify(&h.ext, acpext.MethodPersonality, func(_ context.Context, params jsontext.Value) error {
		id, p, err := personalityOf(params)
		if err != nil {
			return err
		}
		h.deliver(id, Update{Kind: KindPersonality, Personality: p, Params: params})
		return nil
	})
}

// deliver gives u to the session's callback. A notification for an unknown
// session goes to the session being created, as an update does:
// _gobble/personality comes before session/new's response (0022-MADR D6).
func (h *handler) deliver(id acp.SessionId, u Update) {
	h.mu.Lock()
	on := h.sessions[id]
	if on == nil && h.pending != nil {
		on = h.pending
		h.sessions[id] = on
	}
	h.mu.Unlock()
	if on != nil {
		on(u)
	}
}

func (h *handler) commandsOf(id acp.SessionId) []Command {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.commands[id]
}

// RequestPermission asks Options.Permission when it is set (the line
// session's prompt, 0008-MADR D16). Otherwise it declines: it selects the
// first reject_once option, else reject_always, else answers cancelled.
func (h *handler) RequestPermission(ctx context.Context, r acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	if h.permission != nil {
		req := PermissionRequest{SessionID: string(r.SessionId), Tool: ToolCall{ID: string(r.ToolCall.ToolCallId), Content: toolContent(r.ToolCall.Content)}}
		gobbleMarks(&req, r.Meta)
		if r.ToolCall.Title != nil {
			req.Tool.Title = *r.ToolCall.Title
		}
		if req.FullTitle == "" {
			req.FullTitle = req.Tool.Title
		}
		for _, o := range r.Options {
			req.Options = append(req.Options, PermissionOption{ID: string(o.OptionId), Name: o.Name, Kind: string(o.Kind)})
		}
		a := h.permission(ctx, req)
		for _, o := range r.Options {
			if a.OptionID != "" && string(o.OptionId) == a.OptionID {
				resp := acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(o.OptionId)}
				if a.Feedback != "" {
					meta, err := jsonx.SetMember(nil, metaGobble, feedbackMeta{Feedback: a.Feedback})
					if err != nil {
						return acp.RequestPermissionResponse{}, err
					}
					resp.Meta = meta
				}
				return resp, nil
			}
		}
		return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}, nil
	}
	if o, ok := acp.PickPermissionOption(r.Options, acp.PermissionOptionKindRejectOnce, acp.PermissionOptionKindRejectAlways); ok {
		return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(o.OptionId)}, nil
	}
	return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}, nil
}

// gobbleMarks sets req's marks from a permission request's _meta.gobble;
// a missing or mistyped one is none.
func gobbleMarks(req *PermissionRequest, meta map[string]any) {
	g, ok := jsonx.Member[map[string]any](meta, metaGobble)
	if !ok {
		return
	}
	if s, ok := jsonx.Member[string](g, "limit"); ok {
		req.Limit = s
	}
	if s, ok := jsonx.Member[string](g, "onceOnly"); ok {
		req.OnceOnly = s
	}
	if s, ok := jsonx.Member[string](g, "title"); ok {
		req.FullTitle = s
	}
	if l, ok := jsonx.Member[[]string](g, "saves"); ok {
		req.Saves = l
	}
	if b, ok := jsonx.Member[bool](g, "hides"); ok {
		req.Hides = b
	}
	if n, ok := jsonx.Member[int](g, "repeated"); ok {
		req.Repeated = n
	}
}

// feedbackMeta is a permission answer's _meta.gobble (0039-MADR D15).
type feedbackMeta struct {
	Feedback string `json:"feedback"`
}
