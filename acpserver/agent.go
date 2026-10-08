package acpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"uuid"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/command"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// gobble is the agent's name, and the _meta key of gobble's own members.
const gobble = "gobble"

// keyReason, keyError and keySessionID are keys of an error's JSON-RPC
// data.
const (
	keyReason    = "reason"
	keyError     = "error"
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
	// Models is the model choice offered as the model config option. Nil
	// offers none.
	Models func() ModelChoice
	// Tools are the agent's tools. Nil is builtin.Tools(); an empty, non-nil
	// slice is none.
	Tools []tool.Tool
	// Store keeps sessions. Nil keeps them in memory.
	Store session.Store
	// Now is the clock entries are stamped with. Nil is time.Now.
	Now func() time.Time
	// MCP configures the MCP servers sessions connect to.
	MCP MCPOptions
	// Compaction is /compact's settings. Zero is Pi's defaults
	// (compaction.DefaultSettings).
	Compaction compaction.Settings
	// Logger receives diagnostics, such as MCP servers that need attention.
	// Nil discards them; Serve passes its own.
	Logger *slog.Logger
}

// Agent is gobble's acp.Agent.
type Agent struct {
	version  string
	newID    func() string
	provider func() (llm.Provider, error)
	models   func() ModelChoice
	tools    []tool.Tool
	byName   map[string]tool.Tool
	store    session.Store
	now      func() time.Time
	mcp      MCPOptions
	logger   *slog.Logger
	// compaction is /compact's settings.
	compaction compaction.Settings

	mu       sync.Mutex
	conn     *acp.AgentSideConnection
	sessions map[acp.SessionId]*liveSession
	model    llm.Provider // built on the first prompt
}

// liveSession is one open session: its log, the context rebuilt from it, the
// model and thinking chosen, and the cancel of its running turn.
type liveSession struct {
	id  acp.SessionId
	cwd string
	// dirs are the client's additional directories: workspace roots after
	// cwd (0005-MADR "Confinement").
	dirs    []string
	log     session.Log
	history []llm.Message
	leaf    string
	ids     map[string]bool
	model   string
	think   string
	cancel  context.CancelFunc
	mcp     *liveMCP
	mode    string
	used    int // the context's tokens, as the last usage_update said
	queue   *queue
	// streaming and compacting say what the running turn is: a model turn,
	// or a compaction (_gobble/get_state).
	streaming, compacting bool
	// appendMu serialises append: session/set_config_option writes while a
	// turn writes (0002-PLAN Phase 7, deviation 4).
	appendMu sync.Mutex
}

var (
	_ acp.Agent       = (*Agent)(nil)
	_ acp.AgentLoader = (*Agent)(nil)
)

// New returns an Agent. Bind it to its connection with SetConnection before
// the first request, so it can send session updates.
func New(opts Options) *Agent {
	a := &Agent{version: opts.Version, newID: opts.NewID, provider: opts.Provider, models: opts.Models,
		tools: opts.Tools, store: opts.Store, now: opts.Now, mcp: opts.MCP, logger: opts.Logger, sessions: map[acp.SessionId]*liveSession{}}
	if a.logger == nil {
		a.logger = slog.New(slog.DiscardHandler)
	}
	a.compaction = opts.Compaction
	if a.compaction == (compaction.Settings{}) {
		a.compaction = compaction.DefaultSettings
	}
	if a.newID == nil {
		a.newID = func() string { return uuid.NewV7().String() }
	}
	if a.tools == nil {
		a.tools = builtin.Tools()
	}
	if a.store == nil {
		a.store = session.NewMemoryStore()
	}
	if a.now == nil {
		a.now = time.Now
	}
	if a.models == nil {
		a.models = func() ModelChoice { return ModelChoice{} }
	}
	a.byName = map[string]tool.Tool{}
	for _, t := range a.tools {
		a.byName[t.Spec().Name] = t
	}
	return a
}

// Close closes every open session's log, releasing its writer lock. Serve
// calls it when the connection ends.
func (a *Agent) Close() error {
	a.mu.Lock()
	sessions := a.sessions
	a.sessions = map[acp.SessionId]*liveSession{}
	a.mu.Unlock()
	var errs []error
	for _, s := range sessions {
		a.closeMCP(s.mcp)
		errs = append(errs, s.log.Close())
	}
	return errors.Join(errs...)
}

// SetConnection binds the agent to the connection it answers on.
func (a *Agent) SetConnection(c *acp.AgentSideConnection) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conn = c
}

// Capabilities is what is implemented, and nothing more: loading, listing,
// resuming and closing sessions, and MCP servers over stdio and streamable
// HTTP. Images, audio and embedded context are turned on by the phases
// that implement them.
func Capabilities() acp.AgentCapabilities {
	return acp.AgentCapabilities{
		LoadSession:     true,
		McpCapabilities: acp.McpCapabilities{Http: true},
		SessionCapabilities: acp.SessionCapabilities{
			AdditionalDirectories: &acp.SessionAdditionalDirectoriesCapabilities{},
			Close:                 &acp.SessionCloseCapabilities{},
			List:                  &acp.SessionListCapabilities{},
			Resume:                &acp.SessionResumeCapabilities{},
		},
		// gobble's extension methods, as ACP's extensibility page has custom
		// capabilities advertised (0002-PLAN Phase 7).
		Meta: map[string]any{gobble: map[string]any{"extensions": extensionNames()}},
	}
}

// Initialize answers the handshake. It makes no network call (0008-MADR
// D19 item 6).
func (a *Agent) Initialize(context.Context, acp.InitializeRequest) (acp.InitializeResponse, error) {
	title := gobble
	return acp.InitializeResponse{
		ProtocolVersion:   acp.ProtocolVersionNumber,
		AgentInfo:         &acp.Implementation{Name: gobble, Title: &title, Version: a.version},
		AgentCapabilities: Capabilities(),
		AuthMethods:       []acp.AuthMethod{},
	}, nil
}

func invalidParams(id acp.SessionId, reason string) error {
	return acp.NewInvalidParams(map[string]any{keySessionID: id, keyReason: reason})
}

// NewSession opens a session in the client's working directory. Its file is
// written with its first message. _meta.gobble may choose the id, fork an
// existing session, and name it (0002-PLAN Phase 4).
func (a *Agent) NewSession(ctx context.Context, p acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	meta := readNewSessionMeta(p.Meta)
	id := session.ID(a.newID())
	if meta.SessionID != "" {
		id = session.ID(meta.SessionID)
		if !session.ValidID(id) {
			return acp.NewSessionResponse{}, invalidParams(acp.SessionId(id), "a session id is letters, digits, '.', '_' and '-', starting and ending with a letter or digit")
		}
	}
	dirs, err := additionalDirs(acp.SessionId(id), p.AdditionalDirectories)
	if err != nil {
		return acp.NewSessionResponse{}, err
	}
	servers, notes, err := a.mcpServers(acp.SessionId(id), p.McpServers)
	if err != nil {
		return acp.NewSessionResponse{}, err
	}
	var forked []session.Entry
	h := session.Header{Type: session.TypeHeader, Version: session.Version, ID: id, Timestamp: session.Timestamp(a.now()), Cwd: p.Cwd}
	if meta.ForkFrom != "" {
		src, entries, _, err := a.store.Read(ctx, session.ID(meta.ForkFrom))
		if err != nil {
			return acp.NewSessionResponse{}, invalidParams(acp.SessionId(meta.ForkFrom), fmt.Sprintf("cannot fork: %v", err))
		}
		h.ParentSession = string(src.ID)
		if pf, ok := a.store.(interface{ PathOf(session.ID) string }); ok {
			h.ParentSession = pf.PathOf(src.ID)
		}
		forked = entries
	}
	a.mu.Lock()
	_, live := a.sessions[acp.SessionId(id)]
	a.mu.Unlock()
	if live {
		return acp.NewSessionResponse{}, invalidParams(acp.SessionId(id), "a session with this id is open")
	}
	log, err := a.store.Create(ctx, h)
	if errors.Is(err, session.ErrExists) {
		return acp.NewSessionResponse{}, invalidParams(acp.SessionId(id), "a session with this id exists")
	}
	if err != nil {
		return acp.NewSessionResponse{}, acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	s := &liveSession{id: acp.SessionId(id), cwd: p.Cwd, dirs: dirs, log: log, ids: map[string]bool{}, think: thinkingOff, model: a.models().Default, mode: modeDefault, queue: newQueue()}
	if len(forked) > 0 {
		for _, e := range forked {
			if err := log.Append(ctx, e); err != nil {
				return acp.NewSessionResponse{}, errors.Join(acp.NewInternalError(map[string]any{keyReason: err.Error()}), log.Close())
			}
		}
		a.restore(s, forked)
	}
	if meta.Name != "" {
		if err := s.append(ctx, a.now(), session.Entry{Type: session.TypeSessionInfo, Name: new(oneLine(meta.Name))}); err != nil {
			return acp.NewSessionResponse{}, errors.Join(acp.NewInternalError(map[string]any{keyReason: err.Error()}), log.Close())
		}
	}
	s.mcp = a.startMCP(ctx, s.cwd, servers, notes)
	a.mu.Lock()
	a.sessions[s.id] = s
	a.mu.Unlock()
	a.mu.Lock()
	conn := a.conn
	a.mu.Unlock()
	if err := a.startFrames(ctx, conn, s); err != nil {
		return acp.NewSessionResponse{}, err
	}
	model, think := a.choices(s)
	return acp.NewSessionResponse{SessionId: s.id, ConfigOptions: configOptions(a.models(), model, think), Modes: modeState(s.mode)}, nil
}

// oneLine is a name on one line, trimmed, as Pi stores one
// (session-manager.ts appendSessionInfo).
func oneLine(s string) string {
	return strings.TrimSpace(strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(s))
}

// restore rebuilds a session's context and choices from its entries. The
// saved model is kept only when its provider is the current one.
func (a *Agent) restore(s *liveSession, entries []session.Entry) {
	st := replayState(entries)
	s.history, s.leaf, s.ids, s.think = st.history, st.leaf, st.ids, st.thinking
	choice := a.models()
	if st.model != "" && (choice.Provider == "" || st.provider == choice.Provider) {
		s.model = st.model
	}
}

// append stamps an entry as a child of the leaf, records it, and makes it
// the leaf, one append at a time.
func (s *liveSession) append(ctx context.Context, now time.Time, e session.Entry) error {
	s.appendMu.Lock()
	defer s.appendMu.Unlock()
	e.ID = session.NewEntryID(func(id string) bool { return s.ids[id] })
	if s.leaf != "" {
		e.ParentID = new(s.leaf)
	}
	e.Timestamp = session.Timestamp(now)
	if err := s.log.Append(ctx, e); err != nil {
		return err
	}
	s.ids[e.ID] = true
	s.leaf = e.ID
	return nil
}

// Prompt runs one turn of the session: the model, and the tools it asks
// for, streamed as session updates. One prompt runs at a time per session.
// Each message is written before the update that reports it is sent
// (0008-MADR D19 item 6).
func (a *Agent) Prompt(ctx context.Context, p acp.PromptRequest) (acp.PromptResponse, error) {
	a.mu.Lock()
	s, ok := a.sessions[p.SessionId]
	conn := a.conn
	a.mu.Unlock()
	if !ok {
		return acp.PromptResponse{}, invalidParams(p.SessionId, "unknown session")
	}
	turnCtx, release, err := a.claimTurn(ctx, s)
	if err != nil {
		return acp.PromptResponse{}, err
	}
	defer release()
	if name, args, isCommand := command.Parse(promptText(p.Prompt)); isCommand {
		return a.runCommand(turnCtx, s, conn, name, args)
	}
	model, err := a.modelProvider()
	if err != nil {
		return acp.PromptResponse{}, promptError(err)
	}
	a.mu.Lock()
	s.streaming = true
	history := s.history
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		s.streaming = false
		a.mu.Unlock()
	}()

	prompt := userMessage(p.Prompt)
	chosen, think := a.choices(s)
	w := writer{a: a, s: s, ctx: context.WithoutCancel(ctx), provider: a.providerID(model), model: chosen, written: map[string]bool{}}
	w.add(session.Entry{Type: session.TypeMessage, Message: userEntryMessage(prompt, session.Timestamp(a.now()))})
	if w.err != nil {
		return acp.PromptResponse{}, acp.NewInternalError(map[string]any{keyReason: w.err.Error()})
	}
	tools := append(slices.Clip(a.tools), a.mcpTools(turnCtx, s.mcp)...)
	if a.modeOf(s) == modePlan {
		tools = append(tools, exitPlan{a: a, conn: conn, s: s})
	}
	ag := agent.New(agent.Config{
		Provider: model,
		Tools:    tools,
		System:   systemPrompt(s.cwd, tools),
		Model:    chosen,
		Thinking: think,
		Policy:   planPolicy{a: a, s: s, inner: &acpPolicy{conn: conn, session: p.SessionId}},
		Queue:    s.queue,
	})
	out := newUpdates(ctx, conn, p.SessionId)
	var last llm.Usage
	for ev, err := range ag.Run(turnCtx, s.env(), history, prompt) {
		if err != nil {
			out.flush()
			a.mu.Lock()
			s.history = append(s.history, prompt) // the file keeps the prompt too
			a.mu.Unlock()
			return acp.PromptResponse{}, promptError(err)
		}
		w.event(ev)
		if w.err != nil {
			out.flush()
			return acp.PromptResponse{}, acp.NewInternalError(map[string]any{keyReason: w.err.Error()})
		}
		out.event(ev, &last)
		if end, ok := ev.(agent.End); ok {
			a.mu.Lock()
			s.history = append(s.history, end.Messages...)
			a.mu.Unlock()
			a.setUsed(s, int(last.InputTokens+last.OutputTokens))
			out.send(usageUpdate(last))
			if out.err != nil {
				return acp.PromptResponse{}, out.err
			}
			resp := acp.PromptResponse{StopReason: acp.StopReason(end.StopReason), Usage: acpUsage(end.Usage)}
			if end.StopReason == agent.StopCancelled {
				resp.Meta = clearedMeta(s)
			}
			return resp, nil
		}
	}
	return acp.PromptResponse{StopReason: acp.StopReasonCancelled, Meta: clearedMeta(s)}, nil
}

// claimTurn makes the caller's work the session's one running turn, or
// refuses it while another runs. The check and the claim are one section
// of a.mu (0002-PLAN Phase 7, deviation 2). session/cancel cancels the
// returned context; release frees the turn.
func (a *Agent) claimTurn(ctx context.Context, s *liveSession) (context.Context, func(), error) {
	turnCtx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	defer a.mu.Unlock()
	if s.cancel != nil {
		cancel()
		return nil, nil, acp.NewInvalidRequest(map[string]any{keySessionID: s.id, keyReason: "a prompt is already running"})
	}
	s.cancel = cancel
	return turnCtx, func() {
		a.mu.Lock()
		s.cancel = nil
		a.mu.Unlock()
		cancel()
	}, nil
}

// clearedMeta empties a cancelled turn's queues into the response's
// _meta.gobble.cleared, so a client can restore the text, as Pi's terminal
// does (owner's decision of 2026-10-07). It is nil when both were empty.
func clearedMeta(s *liveSession) map[string]any {
	steering, followUp := s.queue.clear()
	if len(steering) == 0 && len(followUp) == 0 {
		return nil
	}
	return map[string]any{gobble: map[string]any{"cleared": map[string]any{"steering": steering, "followUp": followUp}}}
}

// providerID names the provider assistant messages record: the model
// choice's, else the provider's own id.
func (a *Agent) providerID(p llm.Provider) string {
	if c := a.models(); c.Provider != "" {
		return c.Provider
	}
	return p.ID()
}

// writer records one turn's messages as they complete.
type writer struct {
	a        *Agent
	s        *liveSession
	ctx      context.Context
	provider string
	model    string
	written  map[string]bool // tool calls whose result is written
	err      error
}

func (w *writer) add(e session.Entry) {
	if w.err == nil {
		w.err = w.s.append(w.ctx, w.a.now(), e)
	}
}

// event writes what ev completes: a queued user message as it enters, the
// assistant message of a model call, each tool result before its terminal
// update, and at the end the results of calls a cancel cut short.
func (w *writer) event(ev agent.Event) {
	ts := func() string { return session.Timestamp(w.a.now()) }
	switch ev := ev.(type) {
	case agent.Queued:
		w.add(session.Entry{Type: session.TypeMessage, Message: userEntryMessage(ev.Message, ts())})
	case agent.Message:
		w.add(session.Entry{Type: session.TypeMessage, Message: assistantEntryMessage(ev, w.provider, w.model, ts())})
	case agent.ToolEnd:
		w.written[ev.Call.ID] = true
		w.add(session.Entry{Type: session.TypeMessage, Message: toolResultMessage(ev.Call.ID, ev.Call.Name, ev.Result.Text(), ev.Result.IsError, ts())})
	case agent.End:
		for _, m := range ev.Messages {
			for _, c := range m.Content {
				if c.Type == llm.ContentToolResult && !w.written[c.ToolCallID] {
					w.written[c.ToolCallID] = true
					w.add(session.Entry{Type: session.TypeMessage, Message: toolResultMessage(c.ToolCallID, c.ToolName, c.Text, c.IsError, ts())})
				}
			}
		}
	}
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

// promptText is the prompt's text blocks, as its user message joins them.
func promptText(blocks []acp.ContentBlock) string {
	var parts []string
	for _, b := range blocks {
		if b.Text != nil {
			parts = append(parts, b.Text.Text)
		}
	}
	return strings.Join(parts, "\n\n")
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

// open opens a stored session for writing, rebuilds it, and starts its MCP
// servers.
func (a *Agent) open(ctx context.Context, id acp.SessionId, cwd string, extra []string, req []acp.McpServer) (*liveSession, error) {
	dirs, err := additionalDirs(id, extra)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	_, live := a.sessions[id]
	a.mu.Unlock()
	if live {
		return nil, acp.NewInvalidRequest(map[string]any{keySessionID: id, keyReason: "the session is already open"})
	}
	servers, notes, err := a.mcpServers(id, req)
	if err != nil {
		return nil, err
	}
	log, err := a.store.Open(ctx, session.ID(id))
	if err != nil {
		if locked, ok := errors.AsType[*session.ErrLocked](err); ok {
			return nil, acp.NewInvalidRequest(map[string]any{keySessionID: id, keyReason: locked.Error()})
		}
		return nil, invalidParams(id, err.Error())
	}
	if cwd == "" {
		cwd = log.Header().Cwd
	}
	s := &liveSession{id: id, cwd: cwd, dirs: dirs, log: log, think: thinkingOff, model: a.models().Default, mode: modeDefault, queue: newQueue()}
	a.restore(s, log.Entries())
	s.mcp = a.startMCP(ctx, cwd, servers, notes)
	a.mu.Lock()
	a.sessions[id] = s
	a.mu.Unlock()
	return s, nil
}

// LoadSession opens a stored session and replays it compactly (0008-MADR
// D19 item 5), then sends the usage snapshot, before answering.
func (a *Agent) LoadSession(ctx context.Context, p acp.LoadSessionRequest) (acp.LoadSessionResponse, error) {
	s, err := a.open(ctx, p.SessionId, p.Cwd, p.AdditionalDirectories, p.McpServers)
	if err != nil {
		return acp.LoadSessionResponse{}, err
	}
	a.mu.Lock()
	conn := a.conn
	a.mu.Unlock()
	out := newUpdates(ctx, conn, s.id)
	st := replayState(s.log.Entries())
	for _, u := range replayUpdates(st, a.byName, s.cwd) {
		out.send(u)
	}
	if out.err != nil {
		return acp.LoadSessionResponse{}, out.err
	}
	if err := a.startFrames(ctx, conn, s); err != nil {
		return acp.LoadSessionResponse{}, err
	}
	model, think := a.choices(s)
	return acp.LoadSessionResponse{ConfigOptions: configOptions(a.models(), model, think), Modes: modeState(s.mode), Meta: resumedMeta(st)}, nil
}

// resumedMeta is a resumed session's _meta.gobble: its title (the name,
// else the first prompt) and the number of user and assistant messages on
// its path, which ACP's SessionInfo does not carry (0002-PLAN Phase 4,
// deviation of 2026-10-06).
func resumedMeta(st state) map[string]any {
	title, n := st.name, 0
	for _, e := range st.messages {
		switch e.Message.Role {
		case session.RoleUser:
			n++
			if title == "" {
				title = e.Message.Text()
			}
		case session.RoleAssistant:
			n++
		}
	}
	return map[string]any{gobble: map[string]any{"title": cutTitle(title), "messages": n}}
}

// ResumeSession opens a stored session with no replay.
func (a *Agent) ResumeSession(ctx context.Context, p acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	s, err := a.open(ctx, p.SessionId, p.Cwd, p.AdditionalDirectories, p.McpServers)
	if err != nil {
		return acp.ResumeSessionResponse{}, err
	}
	a.mu.Lock()
	conn := a.conn
	a.mu.Unlock()
	if err := a.startFrames(ctx, conn, s); err != nil {
		return acp.ResumeSessionResponse{}, err
	}
	model, think := a.choices(s)
	return acp.ResumeSessionResponse{ConfigOptions: configOptions(a.models(), model, think), Modes: modeState(s.mode),
		Meta: resumedMeta(replayState(s.log.Entries()))}, nil
}

// ListSessions lists the sessions with a file, and the live ones of this
// process without one yet, newest first.
func (a *Agent) ListSessions(ctx context.Context, p acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	f := session.Filter{}
	if p.Cwd != nil {
		f.Cwd = *p.Cwd
	}
	seen := map[acp.SessionId]bool{}
	out := []acp.SessionInfo{}
	for sum, err := range a.store.List(ctx, f) {
		if err != nil {
			return acp.ListSessionsResponse{}, acp.NewInternalError(map[string]any{keyReason: err.Error()})
		}
		seen[acp.SessionId(sum.ID)] = true
		out = append(out, sessionInfo(sum))
	}
	a.mu.Lock()
	for id, s := range a.sessions {
		if !seen[id] && (f.Cwd == "" || s.cwd == f.Cwd) {
			out = append(out, acp.SessionInfo{SessionId: id, Cwd: s.cwd, AdditionalDirectories: s.dirs})
		}
	}
	a.mu.Unlock()
	return acp.ListSessionsResponse{Sessions: out}, nil
}

// additionalDirs checks a client's additional directories: each must be
// absolute, as the ACP schema requires. They are kept cleaned.
func additionalDirs(id acp.SessionId, dirs []string) ([]string, error) {
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if !filepath.IsAbs(d) {
			return nil, invalidParams(id, fmt.Sprintf("additional directory %q is not an absolute path", d))
		}
		out = append(out, filepath.Clean(d))
	}
	return out, nil
}

// env is a call's context: the working directory, and the workspace roots,
// cwd first and then the additional directories.
func (s *liveSession) env() tool.Env {
	var roots []string
	if s.cwd != "" {
		roots = append(roots, s.cwd)
	}
	return tool.Env{Cwd: s.cwd, Roots: append(roots, s.dirs...)}
}

// sessionInfo is a list row: the name, else the first prompt cut to 60
// runes, and the last activity.
func sessionInfo(sum session.Summary) acp.SessionInfo {
	info := acp.SessionInfo{SessionId: acp.SessionId(sum.ID), Cwd: sum.Cwd}
	title := sum.Name
	if title == "" {
		title = sum.First
	}
	if title != "" {
		if utf8.RuneCountInString(title) > maxTitle {
			title = cutTitle(title)
		}
		info.Title = &title
	}
	if !sum.Modified.IsZero() {
		info.UpdatedAt = new(sum.Modified.UTC().Format(time.RFC3339))
	}
	return info
}

// CloseSession closes the session's log, releasing its writer lock.
func (a *Agent) CloseSession(_ context.Context, p acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	a.mu.Lock()
	s, ok := a.sessions[p.SessionId]
	delete(a.sessions, p.SessionId)
	a.mu.Unlock()
	if !ok {
		return acp.CloseSessionResponse{}, invalidParams(p.SessionId, "unknown session")
	}
	a.closeMCP(s.mcp)
	if err := s.log.Close(); err != nil {
		return acp.CloseSessionResponse{}, acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	return acp.CloseSessionResponse{}, nil
}

// SetSessionConfigOption sets the model or the thinking level for the
// session's next prompt, and records the change (model_change,
// thinking_level_change).
func (a *Agent) SetSessionConfigOption(ctx context.Context, p acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	v := p.ValueId
	if v == nil {
		return acp.SetSessionConfigOptionResponse{}, acp.NewInvalidParams(map[string]any{keyReason: "only select options are offered"})
	}
	a.mu.Lock()
	s, ok := a.sessions[v.SessionId]
	a.mu.Unlock()
	if !ok {
		return acp.SetSessionConfigOptionResponse{}, invalidParams(v.SessionId, "unknown session")
	}
	opts, err := a.setConfig(ctx, s, string(v.ConfigId), string(v.Value))
	if err != nil {
		return acp.SetSessionConfigOptionResponse{}, err
	}
	return acp.SetSessionConfigOptionResponse{ConfigOptions: opts}, nil
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
