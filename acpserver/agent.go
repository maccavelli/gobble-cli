package acpserver

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/command"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/personality"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// gobble is the agent's name, and the _meta key of gobble's own members.
const gobble = acpext.MetaKey

// closeWait is how long session/close waits for a cancelled turn to
// release before it closes the session's log (0005-PLAN F4-3c-1).
var closeWait = 5 * time.Second

// keyReason is the reason member of an error's data, acpext.ErrorData's
// Reason, by which the tests read a refusal.
const keyReason = "reason"

// errData is d as an error's JSON-RPC data: its members sorted, each
// invalid UTF-8 byte written as U+FFFD, as jsonx.Strings wrote the maps it
// replaces (0039-MADR D14; 0042-MADR A3). A struct of strings cannot fail
// to encode under jsonx's options; if it did, the data says why.
func errData(d acpext.ErrorData) jsontext.Value {
	b, err := jsonx.Value(d)
	if err != nil {
		return jsonx.Strings(map[string]string{keyReason: err.Error()})
	}
	return b
}

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
	// prompt with auth_required. It is called once per session, with the
	// session's ACP id, so each session has its own provider and session id
	// (0021-MADR D4), and with the session's retry settings (0005-PLAN F5-1).
	Provider func(sessionID string, retry llm.Retry) (llm.Provider, error)
	// Models is the model choice offered as the model config option. Nil
	// offers none.
	Models func() ModelChoice
	// ProviderFor builds provider id's model for a session switched to it,
	// with the session's id and retry settings (0025-MADR D1). Nil offers no
	// switch to another provider.
	ProviderFor func(id, sessionID string, retry llm.Retry) (llm.Provider, error)
	// Remember is told each model the user chooses, as a provider/model ref,
	// by /model or the model option, for the recent models (0025-MADR D3).
	// Nil remembers nothing.
	Remember func(ref string)
	// SaveRefusal records a learned reasoning refusal for provider id
	// (0036-MADR D4). Nil records none.
	SaveRefusal func(providerID string, refusal llm.ReasoningRefusal)
	// CredentialGeneration is provider id's credential generation, which
	// advances when its credential changes, by a sign-in or a sign-out; a
	// mark a refused turn set holds only while the generation is the one
	// it was set under (0025-MADR D4, D6). Nil keeps a mark for the
	// agent's life.
	CredentialGeneration func(id string) uint64
	// PlanConsent is provider id's background-use consent, read now from the
	// stored session: whether it is a session that needs the consent, and
	// whether the user granted it. It is asked at each title and each
	// automatic compaction, so a withdrawal applies from the next request in
	// every process; an error is no grant, and nothing is made (0052-MADR
	// D2). Nil gates nothing.
	PlanConsent func(id string) (ConsentState, error)
	// SearchModels is the SDK's matcher over a provider's models, for /model
	// with words (0025-MADR D5). Nil matches words in each ref and name.
	SearchModels func(provider string, models []string, query string) []string
	// ModelInfo is what the catalog knows of a provider's model: its levels,
	// window and price (0005-PLAN F4-2a). Nil knows nothing, so every model
	// offers off to xhigh, with no window and no price.
	ModelInfo func(provider, model string) llm.Model
	// Discover reads every source of a provider's model's limits, so that
	// ModelInfo then reads what the service states. It is started in the
	// background, with a context the session's close cancels, when a
	// session's model is set (new, loaded or resumed, its title model too)
	// and on a model switch; each call waits for its own model's: a turn,
	// /compact and _gobble/compact for the session's model, the title in
	// the background for the title model's; a cancel ends that wait with
	// the call. Its context ends after 10 seconds, and a call waits no
	// longer whether it returns or not; a close waits for it to return, up
	// to the wait a close gives a title. It may be called concurrently, for
	// several sessions and models. Its error is logged at debug. Nil
	// discovers nothing (0066-MADR D1, D2, S8, S40; 0071-MADR P37, P40).
	Discover func(ctx context.Context, provider, model string) error
	// SaveObservation records a cap a service stated for providerID's model
	// when it refused a call's output limit, which the call (a turn's, a
	// summary's or a title's) was sent again at, for later runs; CallOutput a
	// completed call's output tokens, as the service reported them, which drop
	// a saved cap of its model they exceed; model is the one a cap learned for
	// the call's model was saved under, else the call's. Nil records nothing
	// (0066-MADR D7; 0071-MADR P19, P25, P34, P43, P49, P52).
	SaveObservation func(providerID string, o llm.LimitObservation)
	CallOutput      func(providerID, model string, outputTokens int)
	// CachedLimits are a provider's model's limits as they are cached now:
	// it must start no request. The model option's descriptions read it.
	// Nil describes nothing (0066-MADR D6, S19; 0071-MADR P46).
	CachedLimits func(provider, model string) llm.Limits
	// Override is the run's own model and thinking level, which win over
	// every session's, new, forked, loaded or resumed (0005-PLAN F4-2c).
	// Zero leaves each session its own.
	Override Override
	// Tools are the agent's tools. Nil is builtin.Tools(); an empty, non-nil
	// slice is none.
	Tools []tool.Tool
	// NoRequestPermissions leaves request_permissions out of every session,
	// as the tool flags may (0005-PLAN F6-5 deviation 1).
	NoRequestPermissions bool
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
	// Prompt configures the system prompt (0005-PLAN F3-1).
	Prompt PromptOptions
	// Settings gives a session's settings when it starts, by its working
	// directory (0018-MADR D17). Nil is today's defaults.
	Settings func(cwd string) SessionSettings
	// Auth is what ACP authentication offers (0016-MADR A2). Zero offers
	// nothing.
	Auth AuthOptions
	// SaveSetting writes a key's value to the global settings, as gobble
	// config set does, for _gobble/set_auto_retry (0005-PLAN F5-1 decision
	// 6). Nil writes nothing.
	SaveSetting func(path, value string) error
	// Personalities are the personalities a session may take, the
	// built-ins and the user's (0022-PLAN P3). Nil is the built-ins.
	Personalities *personality.Set
	// CompactionPrompts are the user's compaction prompts; empty ones are
	// gobble's (0005-PLAN F5-4 decision 8).
	CompactionPrompts compaction.Prompts
	// Title is the small model that names a new session from its first
	// prompt; zero names none (0005-PLAN F5-5).
	Title TitleOptions
	// Trust is /trust's store; zero saves nothing (0005-PLAN F6-1).
	Trust TrustOptions
	// Permissions are where rules are saved (0005-PLAN F6-2b).
	Permissions PermissionOptions
}

// Agent is gobble's acp.Agent.
type Agent struct {
	// AgentRefusals answers the gated methods gobble does not offer,
	// session/delete and logout, "method not found"; its own CloseSession,
	// ListSessions and ResumeSession take precedence (0014-MADR D3,
	// 0016-MADR A2; 0035-MADR D8).
	acp.AgentRefusals

	version  string
	newID    func() string
	provider func(sessionID string, retry llm.Retry) (llm.Provider, error)
	// providerFor builds another provider for a switched session.
	providerFor   func(id, sessionID string, retry llm.Retry) (llm.Provider, error)
	models        func() ModelChoice
	rememberModel func(ref string)
	saveRefusal   func(providerID string, refusal llm.ReasoningRefusal)
	// saveObservation and callOutput are Options.SaveObservation and
	// Options.CallOutput; cachedLimits Options.CachedLimits (0071-MADR P43,
	// P46).
	saveObservation func(providerID string, o llm.LimitObservation)
	callOutput      func(providerID, model string, outputTokens int)
	cachedLimits    func(provider, model string) llm.Limits
	// savedAs is, by provider/model asked, the model a cap learned for
	// such a request was saved under, when that is another (0071-PLAN's
	// review deviation of 2026-11-25, R6); guarded by mu.
	savedAs map[string]string
	search  func(provider string, models []string, query string) []string
	// marksMu guards marks, the providers a turn saw refused (0025-MADR D4).
	marksMu sync.Mutex
	marks   map[string]mark
	// credGen is Options.CredentialGeneration; planConsent
	// Options.PlanConsent.
	credGen     func(id string) uint64
	planConsent func(id string) (ConsentState, error)
	info        func(provider, model string) llm.Model
	discover    func(ctx context.Context, provider, model string) error
	override    Override
	tools       []tool.Tool
	noGrants    bool
	byName      map[string]tool.Tool
	store       session.Store
	now         func() time.Time
	mcp         MCPOptions
	logger      *slog.Logger
	prompt      PromptOptions
	// compaction is /compact's settings, the fallback of each session's.
	compaction compaction.Settings
	settings   func(cwd string) SessionSettings
	auth       AuthOptions
	save       func(path, value string) error
	// personalities are the personalities sessions may take.
	personalities *personality.Set
	// compactionPrompts are the user's compaction prompts.
	compactionPrompts compaction.Prompts
	// titles is the small model that names sessions.
	titles TitleOptions
	// trust is /trust's store.
	trust TrustOptions
	perms PermissionOptions

	mu       sync.Mutex
	conn     *acp.AgentSideConnection
	sessions map[acp.SessionId]*liveSession
	// client is what the client offers, from initialize: its files and
	// terminals (0005-PLAN F1d-3).
	client acp.ClientCapabilities
	// unlisted are the stored sessions session/list leaves out, each
	// logged once (0019-MADR D5).
	unlisted map[acp.SessionId]bool
	// signing is the method of each provider's running sign-in, one at a
	// time (0051-MADR D4).
	signing map[string]string
	// insultsAccepted says the user agreed to a level-5 personality in this
	// process (0022-MADR D10).
	insultsAccepted bool
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
	// tree is the session's entries and its leaf, which the next entry
	// follows (0005-PLAN F2-2).
	tree  *session.Tree
	model string
	// providerID is the provider of model, the session's own: the run's
	// until the user switches (0025-MADR D1); "" is the run's.
	providerID string
	// provider is the session's own model provider, built on its first
	// use with the session's id; cache is the hint its turns send
	// (0021-MADR D3, D4).
	provider llm.Provider
	cache    llm.CacheRetention
	think    string
	cancel   context.CancelFunc
	// idle is closed when the running turn releases; nil with no turn
	// (0005-PLAN F4-3c-1).
	idle chan struct{}
	mcp  *liveMCP
	mode string
	used int // the context's tokens, as the last usage_update said
	// outLimit is the output limit the session's last model call asked for,
	// under the agent's mu; zero before its first (0071-MADR D4, D10). Once
	// the call's done event reports what it carried, it is that: the wire's
	// tokens with gobble's basis, "none" for a call that carried none, or
	// the SDK's own basis raw as "sdk:<basis>"; raisedFrom is gobble's value
	// when a thinking budget raised it; outFailed says the call failed after
	// asking it (0071-MADR P22, P31, P32).
	outLimit   llm.OutputLimit
	raisedFrom int
	outFailed  bool
	// said are the notes kept once per session, by kind and provider/model,
	// and notes those not yet said, which the next turn says before its
	// reply, under the agent's mu (0066-MADR D10; 0071-MADR P19, V7). sdkBasis
	// are the provider/model refs whose raw SDK basis was logged (P31).
	said     map[string]bool
	notes    []string
	sdkBasis map[string]bool
	// userOver are the provider/model refs whose models.json maxTokens a
	// service's stated cap held, already said, under the agent's mu
	// (0071-MADR P28).
	userOver map[string]bool
	// disc is the session's discovery of its models' limits (0071-MADR
	// P37).
	disc  discovery
	queue *queue
	// compaction is /compact's settings, from the session's settings.
	compaction compaction.Settings
	// retry is the session's retry settings; retryAbort, while a turn
	// runs, ends its waits when closed, and is nil once closed or with no
	// turn; waiting says a wait is under way (0005-PLAN F5-1).
	retry      llm.Retry
	retryAbort chan struct{}
	// warned is the warnings this session has already logged, shared by
	// the Agent each prompt builds (0036-MADR D3).
	warned  *agent.WarningLog
	waiting bool
	// autoCompactOff turns the automatic compaction off; pause, while it is
	// not pauseNone, pauses its trigger after a failure, which pauseErr
	// gives (0005-PLAN F5-2 decisions 10 and 11). The pause is not saved.
	autoCompactOff bool
	// noAutoResize is images.autoResize off. Zero resizes (0036-MADR D8).
	noAutoResize bool
	pause        acpext.CompactionPause
	pauseErr     string
	// streaming and compacting say what the running turn is: a model turn,
	// or a compaction (_gobble/get_state).
	streaming, compacting bool
	// appendMu serialises append: session/set_config_option writes while a
	// turn writes (0002-PLAN Phase 7, deviation 4).
	appendMu sync.Mutex
	// noted are the prompt diagnostics logged for the session, once each.
	noted map[string]bool
	// notices are the diagnostics kept to send as notices with the
	// session's next start frames or prompt, under the agent's mu
	// (0016-MADR D8).
	notices []acp.SessionUpdate
	// templates are the prompt templates found when the session started
	// (startFrames; 0005-PLAN F3-3).
	templates []sysprompt.Template
	// voice speaks the session's messages; nil is boring (0022-PLAN P1). A
	// switch changes it under Agent.mu (0022-PLAN P3).
	voice *personality.Voice
	// insultsAccepted is the bombasticAccepted setting the session started
	// with (0022-MADR D10).
	insultsAccepted bool
	// client is the client's system prompt, from the request that opened
	// the session or its path's last gobble.system_prompt entry, fixed
	// before the session is live; zero is none (0054-MADR D7).
	client systemPromptData
	// titleTried says the session's title was tried; titleCancel and
	// titleDone stop a title still running and tell when it stopped
	// (0005-PLAN F5-5). nameMu makes a name's check and its write one step,
	// for the title and setName.
	titleTried  bool
	titleCancel context.CancelFunc
	titleDone   chan struct{}
	nameMu      sync.Mutex
	// root is the session's project root, trust whether it is trusted, and
	// project a trusted project's overrides, all fixed when the session
	// starts; trustNoted says an untrusted project's notice was sent
	// (0005-PLAN F6-1).
	root       string
	trust      permission.TrustDecision
	project    ProjectOverrides
	trustNoted bool
	// allowBypass offers the bypass mode, from the settings at the start
	// (0005-PLAN F6-4 decision 2).
	allowBypass bool
	// grants are the user's request_permissions grants, the turn's and the
	// session's, under mu, replaced rather than changed in place (F6-5).
	grants []grant
	// rules is the session's rule engine, built when it starts; its
	// layers change as answers and /permissions add and remove rules,
	// under mu (0005-PLAN F6-2b decision E).
	rules permission.Engine
}

var (
	_ acp.Agent       = (*Agent)(nil)
	_ acp.AgentLoader = (*Agent)(nil)
)

// savedRefusal is the turn's saver for provider id, or nil when the run
// records none (0036-MADR D4).
func savedRefusal(a *Agent, id string) func(llm.ReasoningRefusal) {
	if a.saveRefusal == nil {
		return nil
	}
	return func(r llm.ReasoningRefusal) { a.saveRefusal(id, r) }
}

// warningLog is the session's warning log, created on the first prompt.
func (s *liveSession) warningLog() *agent.WarningLog {
	if s.warned == nil {
		s.warned = &agent.WarningLog{}
	}
	return s.warned
}

// New returns an Agent. Bind it to its connection with SetConnection before
// the first request, so it can send session updates.
func New(opts Options) *Agent {
	a := &Agent{version: opts.Version, newID: opts.NewID, provider: opts.Provider, providerFor: opts.ProviderFor, models: opts.Models, info: opts.ModelInfo,
		discover: opts.Discover, saveObservation: opts.SaveObservation, callOutput: opts.CallOutput, cachedLimits: opts.CachedLimits,
		rememberModel: opts.Remember, saveRefusal: opts.SaveRefusal, search: opts.SearchModels, credGen: opts.CredentialGeneration,
		planConsent: opts.PlanConsent,
		override:    opts.Override, tools: opts.Tools, noGrants: opts.NoRequestPermissions, store: opts.Store, now: opts.Now, mcp: opts.MCP, logger: opts.Logger, prompt: opts.Prompt,
		settings: opts.Settings, auth: opts.Auth, save: opts.SaveSetting, personalities: opts.Personalities, compactionPrompts: opts.CompactionPrompts, titles: opts.Title,
		trust: opts.Trust, perms: opts.Permissions, sessions: map[acp.SessionId]*liveSession{}}
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
	if a.info == nil {
		a.info = func(provider, model string) llm.Model { return llm.Model{Provider: provider, ID: model} }
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
		a.endDiscovery(s)
		a.closeMCP(s.mcp)
		a.closeTitle(s)
		errs = append(errs, s.log.Close())
	}
	return errors.Join(errs...)
}

// SetConnection binds the agent to the connection it answers on, for a
// connection built without SetAgentConnection's call.
func (a *Agent) SetConnection(c *acp.AgentSideConnection) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conn = c
}

// SetAgentConnection is SetConnection, which the SDK's constructor calls
// before it starts reading, so no request finds the agent unbound
// (0016-MADR A1).
func (a *Agent) SetAgentConnection(c *acp.AgentSideConnection) { a.SetConnection(c) }

var _ acp.AgentConnAware = (*Agent)(nil)

// Capabilities is what is implemented, and nothing more: loading, listing,
// resuming and closing sessions, a client's system prompt in append mode
// alone (0054-MADR D7, Q2), and MCP servers over stdio and streamable HTTP.
// Image input is set by Initialize from the default model (0036-MADR D8).
// Audio stays off. Embedded context is on.
func Capabilities() (acp.AgentCapabilities, error) {
	meta, err := jsonx.SetMember(nil, gobble, acpext.CapabilitiesMeta{Extensions: extensionNames()})
	if err != nil {
		return acp.AgentCapabilities{}, err
	}
	return acp.AgentCapabilities{
		LoadSession:     true,
		McpCapabilities: acp.McpCapabilities{Http: true},
		// Resource blocks are read (0005-PLAN F3-4). Initialize sets image
		// from the default model (0036-MADR D8).
		PromptCapabilities: acp.PromptCapabilities{EmbeddedContext: true},
		SessionCapabilities: acp.SessionCapabilities{
			AdditionalDirectories: &acp.SessionAdditionalDirectoriesCapabilities{},
			Close:                 &acp.SessionCloseCapabilities{},
			List:                  &acp.SessionListCapabilities{},
			Resume:                &acp.SessionResumeCapabilities{},
			// No Mode: append alone, so the SDK refuses override.
			SystemPrompt: &acp.SessionSystemPromptCapabilities{},
		},
		// gobble's extension methods, as ACP's extensibility page has custom
		// capabilities advertised (0002-PLAN Phase 7).
		Meta: meta,
	}, nil
}

// Initialize answers the handshake. It makes no network call (0008-MADR
// D19 item 6).
func (a *Agent) Initialize(_ context.Context, p acp.InitializeRequest) (acp.InitializeResponse, error) {
	caps, err := Capabilities()
	if err != nil {
		return acp.InitializeResponse{}, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	caps.PromptCapabilities.Image = a.promptImage()
	a.mu.Lock()
	a.client = p.ClientCapabilities
	a.mu.Unlock()
	title := gobble
	return acp.InitializeResponse{
		ProtocolVersion:   acp.ProtocolVersionNumber,
		AgentInfo:         &acp.Implementation{Name: gobble, Title: &title, Version: a.version},
		AgentCapabilities: caps,
		AuthMethods:       a.authMethods(p.ClientCapabilities.Auth.Terminal),
	}, nil
}

func invalidParams(id acp.SessionId, reason string) error {
	return acp.NewInvalidParams(errData(acpext.ErrorData{Reason: reason, SessionID: string(id)}))
}

// NewSession opens a session in the client's working directory. Its file is
// written with its first message. _meta.gobble may choose the id, fork an
// existing session, and name it (0002-PLAN Phase 4); a system prompt
// replaces a fork's (0054-MADR D7).
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
	// Checked before the store is touched, so a refusal writes nothing.
	client, err := clientPrompt(acp.SessionId(id), p.SystemPrompt)
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
		return acp.NewSessionResponse{}, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	ss := a.sessionSettings(p.Cwd)
	// The run's override starts a new session, as defaultThinkingLevel does;
	// a fork's history is set to it after its restore (0005-PLAN F4-2c).
	s := &liveSession{id: acp.SessionId(id), cwd: p.Cwd, dirs: dirs, log: log, tree: session.NewTree(nil),
		think: cmp.Or(a.override.Thinking, ss.Thinking), model: cmp.Or(a.override.Model, a.models().Default), mode: a.startMode(ss.AllowBypass),
		allowBypass: ss.AllowBypass,
		queue:       sessionQueue(ss), compaction: ss.Compaction, cache: cacheOf(ss.CacheRetention), retry: ss.Retry,
		autoCompactOff: ss.AutoCompactionOff, noAutoResize: ss.NoAutoResize, voice: a.voiceOf(ss), insultsAccepted: ss.BombasticAccepted,
		root: ss.ProjectRoot, trust: ss.Trust, project: projectOf(ss.Project),
		rules: engineOf(ss, p.Cwd, a.storedRules(cmp.Or(ss.ProjectRoot, p.Cwd)))}
	if len(forked) > 0 {
		for _, e := range forked {
			if err := log.Append(ctx, e); err != nil {
				return acp.NewSessionResponse{}, errors.Join(acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()})), log.Close())
			}
		}
		a.restore(s, forked)
		if err := a.applyOverride(ctx, s); err != nil {
			return acp.NewSessionResponse{}, errors.Join(err, log.Close())
		}
	}
	if err := a.takeSystemPrompt(ctx, s, client); err != nil {
		return acp.NewSessionResponse{}, errors.Join(acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()})), log.Close())
	}
	if meta.Name != "" {
		if err := s.append(ctx, a.now(), session.Entry{Type: session.TypeSessionInfo, Name: new(oneLine(meta.Name))}); err != nil {
			return acp.NewSessionResponse{}, errors.Join(acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()})), log.Close())
		}
	}
	s.mcp = a.startMCP(ctx, s.cwd, servers, notes, ss)
	// The model and provider discovery starts for are read before the
	// session is published, while no other request can reach it
	// (0071-PLAN Phase 2b's review, R5).
	startModel, startProvider := s.model, a.provOf(s)
	a.mu.Lock()
	a.sessions[s.id] = s
	a.mu.Unlock()
	a.discoverModels(ctx, s, startProvider, startModel, true)
	a.mu.Lock()
	conn := a.conn
	a.mu.Unlock()
	if err := a.startFrames(ctx, conn, s); err != nil {
		return acp.NewSessionResponse{}, err
	}
	return acp.NewSessionResponse{SessionId: s.id, ConfigOptions: a.configOptions(s), Modes: modeState(s.mode, s.allowBypass)}, nil
}

// oneLine is a name on one line, trimmed, as Pi stores one
// (session-manager.ts appendSessionInfo).
func oneLine(s string) string {
	return strings.TrimSpace(strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(s))
}

// restore rebuilds a session's tree, context and choices from its entries,
// its leaf on the last of them. The saved model is kept only when its
// provider is the current one; the saved personality, when one has its
// name (0022-MADR D6); the saved client's system prompt always (0054-MADR
// D7).
func (a *Agent) restore(s *liveSession, entries []session.Entry) {
	s.tree = session.NewTree(entries)
	st := replayPath(entries, s.tree.Path())
	s.history, s.think, s.client = st.history, st.thinking, st.systemPrompt
	// The session goes on with the provider it last used, the run's or one
	// it switched to, while that one is usable (0025-MADR D1).
	choice := a.choice()
	prov := canonicalProvider(st.provider)
	switch {
	case st.model == "":
	case choice.Provider == "" || prov == "" || prov == choice.Provider:
		s.model = st.model
	case slices.ContainsFunc(choice.providerList(), func(p ProviderChoice) bool { return p.ID == prov && p.State.Usable() }):
		s.providerID, s.model = prov, st.model
	default:
		a.logger.Warn("model: the session's provider is not usable; the session goes on with the run's",
			slog.String("provider", prov), slog.String("model", st.model))
	}
	if st.personality != "" {
		v, err := a.newVoice(s, st.personality)
		if err != nil {
			a.logger.Warn("personality: the session's is not kept", slog.String("error", err.Error()))
			return
		}
		s.voice = v
	}
}

// append stamps an entry as a child of the tree's leaf, writes it, and only
// then adds it to the tree as the leaf, one append at a time.
func (s *liveSession) append(ctx context.Context, now time.Time, e session.Entry) error {
	s.appendMu.Lock()
	defer s.appendMu.Unlock()
	e = s.tree.Next(e)
	e.Timestamp = session.Timestamp(now)
	if err := s.log.Append(ctx, e); err != nil {
		return err
	}
	s.tree.Add(e)
	return nil
}

// state is the session's replay state along its tree's path.
func (s *liveSession) state() state { return replayPath(s.tree.Entries(), s.tree.Path()) }

// replayFile sets the session's history to the file's: what a load of the
// file would give the model. A prompt that fails with entries written that
// the live history lacks calls it on its way out, so the session's next
// request is the request a reload would send (0037-MADR D15). It runs only
// on a failure.
func (a *Agent) replayFile(s *liveSession) {
	st := s.state()
	a.mu.Lock()
	s.history = st.history
	a.mu.Unlock()
}

// appendFailed is a failed append as an ACP error: a session whose file
// gobble does not rewrite, an older version's, is invalid request with
// session.ErrOldVersion's text (0005-PLAN F2); any other failure is
// internal.
func appendFailed(err error) error {
	if old, ok := errors.AsType[*session.ErrOldVersion](err); ok {
		return acp.NewInvalidRequest(errData(acpext.ErrorData{Reason: old.Error()}))
	}
	return acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
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
	a.noteTrust(turnCtx, conn, s)
	// A skill, or a template, is expanded into the user's message, as Pi
	// expands them; any other command runs, a built-in or unknown. No
	// template has a built-in's name (Agent.templates), so a built-in
	// wins (0005-PLAN F3-2, F3-3).
	text := promptText(p.Prompt)
	name, args, isCommand := command.Parse(text)
	if isCommand {
		// The text as Parse reads it, from its "/".
		text = strings.TrimLeftFunc(text, unicode.IsSpace)
	}
	expanded, isExpanded := a.expand(s, text)
	// A session at level 5 asks first, unless the prompt chooses the voice
	// (0022-MADR D10).
	if !isCommand || isExpanded || (name != cmdPersonality && name != cmdHumor) {
		if err := a.settleVoice(turnCtx, conn, s); err != nil {
			if turnCtx.Err() != nil {
				return cancelled(s)
			}
			return acp.PromptResponse{}, err
		}
	}
	if isCommand && !isExpanded {
		return a.runCommand(turnCtx, s, conn, name, args)
	}
	// A new session's first prompt names it, alongside the turn
	// (0005-PLAN F5-5 decision 4).
	if a.titleWanted(s) {
		a.startTitle(turnCtx, s, text)
	}
	model, err := a.sessionProvider(s)
	if err != nil {
		return acp.PromptResponse{}, promptError(a.provOf(s), err)
	}
	abort := make(chan struct{})
	a.mu.Lock()
	s.streaming = true
	history := s.history
	retry := s.retry
	s.retryAbort, s.waiting = abort, false
	if s.pause == acpext.PauseTurn { // a turn's pause ends with it (0005-PLAN F5-2 decision 11)
		s.pause, s.pauseErr = pauseNone, ""
	}
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		s.streaming = false
		s.retryAbort, s.waiting = nil, false
		a.mu.Unlock()
		a.endTurnGrants(s) // a turn's grants end with it (F6-5 decision 4)
	}()

	prompt := userMessage(p.Prompt)
	if isExpanded {
		prompt = llm.Message{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: expanded}}}
	}
	// The files come first, then the text, as Pi's CLI orders them. An image
	// @path is image content when the model takes images (0036-MADR D8). The
	// prompt's text is then one block, joined as the user entry joins it, so
	// the live session sends the bytes a resumed one sends from the entry
	// (0005-PLAN F3-4; 0037-MADR D42).
	chosen, think := a.choices(s)
	files, pics := promptAttachments(p.Prompt, s.cwd, fsx.Workspace{Roots: s.env().Roots}, takesImages(a.info(a.providerID(s, model), chosen), model), !s.noAutoResize)
	if files != "" {
		prompt.Content = append([]llm.Content{{Type: llm.ContentText, Text: files}}, prompt.Content...)
	}
	prompt.Content = append(prompt.Content, pics...)
	prompt = asEntry(prompt)
	think = catalog.Clamp(a.levelsOf(a.provOf(s), chosen), think) // as thought_level shows it
	w := writer{a: a, s: s, ctx: context.WithoutCancel(ctx), provider: a.providerID(s, model), model: chosen, written: map[string]bool{}}
	// The system prompt's baseline or patch comes before the user's message,
	// as Pi puts a system patch before it (0005-PLAN F3-1).
	system, update := a.writePrompt(&w, s)
	if w.err != nil {
		// No replay: the one append before this exit failed whole, and the
		// history takes the patch's update only after it, so the file holds
		// nothing the history lacks (0037-PLAN P2a).
		return acp.PromptResponse{}, appendFailed(w.err)
	}
	if update != nil {
		history = append(slices.Clip(history), *update)
		a.mu.Lock()
		s.history = append(s.history, *update)
		a.mu.Unlock()
	}
	w.add(session.Entry{Type: session.TypeMessage, Message: userEntryMessage(prompt, session.Timestamp(a.now()))})
	if w.err != nil {
		// No replay: the user entry failed whole, and the history holds the
		// patch's update exactly as the file's patch entry does, so the file
		// holds nothing the history lacks (0037-PLAN P2a). The exit after
		// the gobble.tools entry replays, where the file holds the prompt.
		return acp.PromptResponse{}, appendFailed(w.err)
	}
	mcpTools := a.mcpTools(turnCtx, s)
	tools := append(a.toolsOf(s), mcpTools...)
	if a.modeOf(s) == modePlan {
		tools = append(tools, exitPlan{a: a, conn: conn, s: s})
	}
	// request_permissions, deferred until a declined call or tool_search
	// loads it (0005-PLAN F6-5; 0012-MADR D9).
	if !a.noGrants {
		tools = append(tools, requestPermissions{a: a, conn: conn, s: s})
	}
	// The rules' hidden tools are neither sent nor found by tool_search
	// (0005-PLAN F6-2 decision 6).
	tools = visible(withDefaults(a.rulesOf(s), mcpTools, a.modeOf(s)), tools)
	loaded := loadedTools(s.tree.Path())
	if fresh := startLoads(tools, loaded); len(fresh) > 0 {
		w.addEntry(toolsEntry(fresh, ""))
		if w.err != nil {
			a.replayFile(s) // the file holds the prompt
			return acp.PromptResponse{}, appendFailed(w.err)
		}
		loaded = append(loaded, fresh...)
	}
	auto := &autoCompaction{a: a, s: s, provider: model, providerID: w.provider}
	limits := &turnLimits{a: a, s: s, provider: w.provider, model: chosen}
	// The text the live error shows, as promptErrorNoted builds it: the
	// limit's line, and a workspace's advice in place of the service's words
	// (0070-MADR D2, Q10).
	w.errText = func(err error) string { return errorData(a.provOf(s), limitLine(err, limits.last, chosen)).Error }
	called, cuts := a.turnProvider(model, w.provider, limits)
	cfg := agent.Config{
		Provider: called,
		Tools:    tools,
		Model:    chosen,
		Thinking: think,
		Policy:   planPolicy{a: a, s: s, mcp: mcpTools, inner: &acpPolicy{conn: conn, session: p.SessionId}},
		Queue:    s.queue,
		Search:   searchFor(tools),
		Loaded:   loaded,
		System:   system,
		Cache:    s.cache,
		// The session's retry settings, as the turn starts (0005-PLAN F5-1).
		Retry:       &retry,
		RetryAbort:  abort,
		Compact:     auto.compact,
		Logger:      a.logger,
		Warned:      s.warningLog(),
		SaveRefusal: savedRefusal(a, w.provider),
		OutputLimit: limits.limit,
		// A cap a service stated, which the call was sent again at, is
		// learned (0071-MADR P19).
		SaveObservation: func(o llm.LimitObservation) { a.learnedCap(s, w.provider, chosen, o) },
	}
	ag := agent.New(cfg)
	out := newUpdates(ctx, conn, p.SessionId)
	auto.out = out
	a.sendNotices(out, s) // the start-up and MCP diagnostics, as notices (0016-MADR D8)
	// The first call waits for its model's discovery (0066-MADR D1;
	// 0071-MADR P40), before the turn reads its model's facts, image input
	// included; cancelled meanwhile, the turn ends before any call.
	a.waitTurnDiscovery(turnCtx, s, w.provider, chosen)
	if turnCtx.Err() != nil {
		return a.cancelledWaiting(s, w.provider, w.model, prompt)
	}
	var last llm.Usage
	env := s.env()
	env.Images = takesImages(a.info(w.provider, chosen), model)
	env.Environ = markers(string(s.id), a.fileOf(s), w.provider, chosen, think)
	env.Instructions = a.instructions(s)
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if conn != nil && client.Fs.ReadTextFile && client.Fs.WriteTextFile {
		env.Files = clientFiles{conn: conn, session: p.SessionId}
	}
	if conn != nil && client.Terminal {
		env.Terminal = clientTerminal{conn: conn, session: p.SessionId, created: out.terminal}
	}
	out.locate(tools, env)
	// sayCut tells the user that the limit cut a tool call after the reply's
	// text, and the call was dropped: an agent message after the text, live
	// only (0071-MADR P29; 0065-MADR D7, D8).
	sayCut := func() {
		out.send(acp.UpdateAgentMessageText("\n\n" + a.voice(s).Text(personality.TurnStopMaxTokensTool,
			personality.Args{argModel: chosen})))
	}
	for ev, err := range ag.Run(turnCtx, env, history, prompt) {
		if err != nil {
			out.flush()
			// The file holds the prompt, and every message and tool result
			// the turn wrote before its failure, a compaction's rebuilt
			// history included: the live session takes what a reload would
			// (0005-PLAN F5-2 decision 3; 0037-MADR D15).
			a.replayFile(s)
			a.markFailure(a.provOf(s), err)
			return acp.PromptResponse{}, promptErrorNoted(a.provOf(s), limitLine(err, limits.last, chosen), creditNote(err, limits.last, chosen))
		}
		w.event(ev)
		if w.err != nil {
			out.flush()
			a.replayFile(s) // the file holds the entries written before the failed one
			return acp.PromptResponse{}, appendFailed(w.err)
		}
		limits.sayNote(out)
		a.sayNotes(out, s)
		if r, ok := ev.(agent.Retry); ok {
			a.mu.Lock()
			s.waiting = r.Phase == agent.RetryStart
			a.mu.Unlock()
		}
		if _, ok := ev.(agent.Queued); ok && cuts.take() {
			// Queued messages carry the turn on after a call the limit cut
			// while it wrote a tool call: the notice comes first, after the
			// reply's text (0071-MADR P36, "Show the notice anyway
			// (Recommended)").
			out.flush()
			sayCut()
		}
		out.event(ev, &last)
		if end, ok := ev.(agent.End); ok {
			if end.StopReason == agent.StopMaxTokens && cuts.take() {
				sayCut()
			}
			msgs := stamped(end.Messages, llm.Origin{Provider: w.provider, Model: w.model}, end.StopReason == agent.StopCancelled)
			a.mu.Lock()
			s.history = append(s.history, msgs...)
			a.mu.Unlock()
			used := int(last.InputTokens + last.OutputTokens)
			a.setUsed(s, used)
			out.send(a.contextUpdate(s, used))
			if out.err != nil {
				return acp.PromptResponse{}, out.err
			}
			//lint:ignore SA1019 acp-go-sdk2 keeps this unstable-schema name under v1 (0035-MADR D6)
			resp := acp.PromptResponse{StopReason: acp.StopReason(end.StopReason), Usage: acpUsage(end.Usage)} //nolint:staticcheck // as the lint:ignore above
			return withTurnMetaAsked(s, resp, end.StopReason == agent.StopCancelled, limits.shown, limits.failed)
		}
	}
	return withTurnMetaAsked(s, acp.PromptResponse{StopReason: acp.StopReasonCancelled}, true, limits.shown, limits.failed)
}

// cancelledWaiting ends a turn cancelled while its first call waited for
// discovery: the prompt, already written, joins the live history as a
// cancelled turn's does, and no limit was computed, recorded or noted
// (0066-MADR S40; 0071-PLAN Phase 2b's review).
func (a *Agent) cancelledWaiting(s *liveSession, provider, model string, prompt llm.Message) (acp.PromptResponse, error) {
	msgs := stamped([]llm.Message{prompt}, llm.Origin{Provider: provider, Model: model}, true)
	a.mu.Lock()
	s.history = append(s.history, msgs...)
	a.mu.Unlock()
	return withTurnMeta(s, acp.PromptResponse{StopReason: acp.StopReasonCancelled}, true, llm.OutputLimit{})
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
		return nil, nil, acp.NewInvalidRequest(errData(acpext.ErrorData{SessionID: string(s.id), Reason: "a prompt is already running"}))
	}
	idle := make(chan struct{})
	s.cancel, s.idle = cancel, idle
	return turnCtx, func() {
		a.mu.Lock()
		s.cancel, s.idle = nil, nil
		a.mu.Unlock()
		cancel()
		close(idle)
	}, nil
}

// clearedMeta empties a cancelled turn's queues into the response's
// _meta.gobble.cleared, so a client can restore the text, as Pi's terminal
// does (owner's decision of 2026-10-07). It is nil when both were empty.
func clearedMeta(s *liveSession) (map[string]any, error) {
	return turnMeta(s, true, llm.OutputLimit{})
}

// turnMeta is a turn's response _meta: with cancelled, the session's
// queues emptied into _meta.gobble.cleared; and the output limit of the
// turn's last model call as _meta.gobble.outputLimit, which a turn with no
// call carries none of (0071-MADR D10); a call that carried no limit
// carries {0, "none"} (P35). It is nil when there is neither.
func turnMeta(s *liveSession, cancelled bool, lim llm.OutputLimit) (map[string]any, error) {
	return turnMetaAsked(s, cancelled, lim, false)
}

// turnMetaAsked is turnMeta with asked: the last call failed, and lim is
// what it asked for, marked asked: true (0071-MADR P32).
func turnMetaAsked(s *liveSession, cancelled bool, lim llm.OutputLimit, asked bool) (map[string]any, error) {
	var g turnGobble
	if cancelled {
		if steering, followUp := s.queue.clear(); len(steering) > 0 || len(followUp) > 0 {
			g.Cleared = &clearedQueues{FollowUp: followUp, Steering: steering}
		}
	}
	if lim.Basis != "" {
		g.OutputLimit = &limitMeta{Tokens: lim.Tokens, Basis: lim.Basis, Asked: asked}
	}
	if g.Cleared == nil && g.OutputLimit == nil {
		return nil, nil
	}
	return jsonx.SetMember(nil, gobble, g)
}

// withCleared is resp with the session's queues emptied into its
// _meta.gobble.cleared, for a cancelled turn.
func withCleared(s *liveSession, resp acp.PromptResponse) (acp.PromptResponse, error) {
	return withTurnMeta(s, resp, true, llm.OutputLimit{})
}

// withTurnMeta is resp with turnMeta's _meta.
func withTurnMeta(s *liveSession, resp acp.PromptResponse, cancelled bool, lim llm.OutputLimit) (acp.PromptResponse, error) {
	return withTurnMetaAsked(s, resp, cancelled, lim, false)
}

// withTurnMetaAsked is resp with turnMetaAsked's _meta.
func withTurnMetaAsked(s *liveSession, resp acp.PromptResponse, cancelled bool, lim llm.OutputLimit, asked bool) (acp.PromptResponse, error) {
	meta, err := turnMetaAsked(s, cancelled, lim, asked)
	if err != nil {
		return acp.PromptResponse{}, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	resp.Meta = meta
	return resp, nil
}

// turnGobble is a turn's _meta.gobble (0039-MADR D15): a cancelled turn's
// cleared queues, an empty queue being [], as queue.clear gives it and v1
// wrote it; and the last call's output limit, {tokens, basis}
// (0071-MADR D10).
type turnGobble struct {
	Cleared     *clearedQueues `json:"cleared,omitzero"`
	OutputLimit *limitMeta     `json:"outputLimit,omitzero"`
}

// limitMeta is _meta.gobble.outputLimit: the last call's tokens and basis,
// and asked: true when that call failed, so the tokens are what it asked
// for (0071-MADR D10, P32).
type limitMeta struct {
	Tokens int            `json:"tokens"`
	Basis  llm.LimitBasis `json:"basis"`
	Asked  bool           `json:"asked,omitzero"`
}

// clearedQueues are the queues' texts.
type clearedQueues struct {
	FollowUp []string `json:"followUp"`
	Steering []string `json:"steering"`
}

// providerID names the provider assistant messages record: the model
// choice's, else the provider's own id.
func (a *Agent) providerID(s *liveSession, p llm.Provider) string {
	if id := a.provOf(s); id != "" {
		return id
	}
	return p.ID()
}

// customTools is the custom entry recording tools a load added: a
// tool_search call's, or the session's start. Its data is {loaded}, the
// names in load order (0005-PLAN F1d-2).
const customTools = "gobble.tools"

// sessionStartFamilies are the deferred families loaded at the first prompt
// that has them, rather than left behind tool_search (0012-MADR D9): the
// MCP resource tools, which exist only while a server offers resources.
var sessionStartFamilies = []string{builtin.ResourceFamily}

type toolsData struct {
	Loaded []string `json:"loaded"`
	// CallID is the call whose result made the load; "" for a load at
	// session start, and for every entry gobble wrote before 0012-MADR D18.
	CallID string `json:"callId,omitzero"`
}

// toolsEntry records a load, with the call whose result made it, or ""
// for none (0012-MADR D18).
func toolsEntry(names []string, callID string) (session.Entry, error) {
	data, err := jsonx.Value(toolsData{Loaded: names, CallID: callID})
	return session.Entry{Type: session.TypeCustom, CustomType: customTools, Data: data}, err
}

// loadedTools are the tools path's gobble.tools entries loaded, in load
// order, each once: the start of a prompt loads them again (0012-MADR D9).
func loadedTools(path []session.Entry) []string {
	var out []string
	for _, e := range path {
		if e.Type != session.TypeCustom || e.CustomType != customTools {
			continue
		}
		var d toolsData
		if jsonx.Unmarshal(e.Data, &d) != nil {
			continue
		}
		for _, n := range d.Loaded {
			if !slices.Contains(out, n) {
				out = append(out, n)
			}
		}
	}
	return out
}

// searchFor is tool_search when tools hold a deferred tool outside the
// session-start families, loaded or not, and nil otherwise: such a tool was
// behind the search at the first prompt or loaded by a call since, so the
// search stays, while one of those families loads before the first request
// and leaves the search nothing to find (0012-MADR D19).
func searchFor(tools []tool.Tool) tool.Tool {
	for _, t := range tools {
		if s := t.Spec(); s.Exposure == tool.ExposureDeferred && !slices.Contains(sessionStartFamilies, s.Family) {
			return builtin.ToolSearch()
		}
	}
	return nil
}

// startLoads are the deferred tools of a session-start family that are not
// loaded yet, in tools' order.
func startLoads(tools []tool.Tool, loaded []string) []string {
	var out []string
	for _, t := range tools {
		s := t.Spec()
		if s.Exposure == tool.ExposureDeferred && slices.Contains(sessionStartFamilies, s.Family) && !slices.Contains(loaded, s.Name) {
			out = append(out, s.Name)
		}
	}
	return out
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
	// errText is a failed call's error as the live error shows it, which its
	// entry keeps as errorMessage (0070-MADR D2, Q10); nil is the error's own
	// text.
	errText func(error) string
	// held is a failed call's append error: the turn's error that follows it
	// is answered, not this one, which ends the prompt only if the turn goes
	// on instead (0070-MADR D5).
	held error
}

func (w *writer) add(e session.Entry) {
	if w.err == nil {
		w.err = w.s.append(w.ctx, w.a.now(), e)
	}
}

// fail keeps err as the writer's error, as add keeps an append's: the first
// one ends the prompt (0039-MADR D7).
func (w *writer) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

// failedEntry writes m, a failed call's message, in Pi's shape: stopReason
// error, with the error the user saw (0070-MADR D2, D4, Q10). When its append
// fails, the error is logged at warn and held: the turn's error that follows
// is the one answered (D5).
func (w *writer) failedEntry(m *session.Message, failure error) {
	m.StopReason, m.ErrorMessage = piError, failure.Error()
	if w.errText != nil {
		m.ErrorMessage = w.errText(failure)
	}
	if w.err != nil {
		return
	}
	if err := w.s.append(w.ctx, w.a.now(), session.Entry{Type: session.TypeMessage, Message: m}); err != nil {
		w.a.logger.WarnContext(w.ctx, "session: a failed call's entry was not written", slog.String("session", string(w.s.id)),
			slog.String("error", err.Error()))
		w.held = err
	}
}

// addEntry adds e, or keeps err when building e failed.
func (w *writer) addEntry(e session.Entry, err error) {
	if err != nil {
		w.fail(err)
		return
	}
	w.add(e)
}

// event writes what ev completes: a queued user message as it enters, the
// assistant message of a model call, each tool result before its terminal
// update, followed by its plan as a gobble.todo entry when it has one, and
// at the end the results of calls a cancel cut short.
func (w *writer) event(ev agent.Event) {
	if w.held != nil {
		// The turn went on after a failed call's entry was lost: the append's
		// error ends the prompt now (0070-MADR D5).
		w.fail(w.held)
		return
	}
	ts := func() string { return session.Timestamp(w.a.now()) }
	switch ev := ev.(type) {
	case agent.Queued:
		w.add(session.Entry{Type: session.TypeMessage, Message: userEntryMessage(ev.Message, ts())})
	case agent.Message:
		m, err := assistantEntryMessage(ev, w.provider, w.model, w.a.modelOf(w.provider, w.model), ts())
		if err != nil || ev.Err == nil {
			w.addEntry(session.Entry{Type: session.TypeMessage, Message: m}, err)
			return
		}
		w.failedEntry(m, ev.Err)
	case agent.ToolEnd:
		w.written[ev.Call.ID] = true
		w.add(session.Entry{Type: session.TypeMessage, Message: toolResultWithImages(ev.Call.ID, ev.Call.Name, ev.Result, ts())})
		if ev.Result.Plan != nil {
			w.addEntry(todoEntry(ev.Result.Plan))
		}
		if len(ev.Result.Instructions) > 0 {
			w.addEntry(contextEntry(ev.Result.Instructions))
		}
	case agent.Loaded:
		w.addEntry(toolsEntry(ev.Names, ev.CallID))
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

// sessionProvider builds s's provider on first use, with s's id, and keeps
// it for the session (0021-MADR D4). A failed build is not kept.
func (a *Agent) sessionProvider(s *liveSession) (llm.Provider, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s.provider != nil {
		return s.provider, nil
	}
	build := a.provider
	if id := s.providerID; id != "" && id != a.models().Provider { // a switched session's (0025-MADR D1)
		if a.providerFor == nil {
			return nil, ErrNoProvider
		}
		build = func(sid string, retry llm.Retry) (llm.Provider, error) { return a.providerFor(id, sid, retry) }
	}
	if build == nil {
		return nil, ErrNoProvider
	}
	p, err := build(string(s.id), s.retry)
	if err != nil {
		return nil, err
	}
	s.provider = p
	return p, nil
}

// userMessage is the prompt's text, and its images, as one user message
// (0036-MADR D8). An image block's data is base64.
func userMessage(blocks []acp.ContentBlock) llm.Message {
	m := llm.Message{Role: llm.RoleUser}
	for _, b := range blocks {
		switch {
		case b.Text != nil:
			m.Content = append(m.Content, llm.Content{Type: llm.ContentText, Text: b.Text.Text})
		case b.Image != nil && b.Image.Data != "":
			data, err := base64.StdEncoding.DecodeString(b.Image.Data)
			if err != nil {
				continue
			}
			m.Content = append(m.Content, llm.Content{Type: llm.ContentImage, MediaType: b.Image.MimeType, Data: data})
		}
	}
	return m
}

// promptImage reports whether initialize advertises image input. The
// configured default model's facts decide: input holding image, or input
// known without it. Unknown facts follow the provider, and a provider that
// cannot be built advertises nothing. The call does not fail the handshake
// (0036-MADR D8; 0008-MADR D19).
func (a *Agent) promptImage() bool {
	choice := a.models()
	info := a.info(choice.Provider, cmp.Or(a.override.Model, choice.Default))
	if slices.Contains(info.Input, "image") {
		return true
	}
	if len(info.Input) > 0 {
		return false
	}
	if a.provider == nil {
		return false
	}
	p, err := a.provider("", llm.Retry{Off: true})
	if err != nil || p == nil {
		return false
	}
	return p.Capabilities().ImageInput
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

// open opens a stored session for writing, rebuilds it, takes the client's
// system prompt when it differs from the stored one (0054-MADR D7), and
// starts its MCP servers.
func (a *Agent) open(ctx context.Context, id acp.SessionId, cwd string, extra []string, req []acp.McpServer, sp *acp.SystemPrompt) (*liveSession, error) {
	dirs, err := additionalDirs(id, extra)
	if err != nil {
		return nil, err
	}
	client, err := clientPrompt(id, sp)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	_, live := a.sessions[id]
	a.mu.Unlock()
	if live {
		return nil, acp.NewInvalidRequest(errData(acpext.ErrorData{SessionID: string(id), Reason: "the session is already open"}))
	}
	servers, notes, err := a.mcpServers(id, req)
	if err != nil {
		return nil, err
	}
	log, err := a.store.Open(ctx, session.ID(id))
	if err != nil {
		if locked, ok := errors.AsType[*session.ErrLocked](err); ok {
			return nil, acp.NewInvalidRequest(errData(acpext.ErrorData{SessionID: string(id), Reason: locked.Error()}))
		}
		return nil, invalidParams(id, err.Error())
	}
	ss := a.sessionSettings(cwd)
	// The history's thinking level wins: restore sets it (0018-MADR D17).
	s := &liveSession{id: id, cwd: cwd, dirs: dirs, log: log, think: thinkingOff, model: a.models().Default, mode: a.startMode(ss.AllowBypass),
		allowBypass: ss.AllowBypass,
		queue:       sessionQueue(ss), compaction: ss.Compaction, cache: cacheOf(ss.CacheRetention), retry: ss.Retry,
		autoCompactOff: ss.AutoCompactionOff, noAutoResize: ss.NoAutoResize, voice: a.voiceOf(ss), insultsAccepted: ss.BombasticAccepted,
		root: ss.ProjectRoot, trust: ss.Trust, project: projectOf(ss.Project),
		rules: engineOf(ss, cwd, a.storedRules(cmp.Or(ss.ProjectRoot, cwd)))}
	a.restore(s, log.Entries())
	if err := a.takeSystemPrompt(ctx, s, client); err != nil {
		return nil, errors.Join(appendFailed(err), log.Close())
	}
	// The run's override wins over the history's choice (0005-PLAN F4-2c).
	if err := a.applyOverride(ctx, s); err != nil {
		return nil, errors.Join(err, log.Close())
	}
	s.mcp = a.startMCP(ctx, cwd, servers, notes, ss)
	a.mu.Lock()
	a.sessions[id] = s
	a.mu.Unlock()
	model, _ := a.choices(s)
	a.discoverModels(ctx, s, a.provOf(s), model, true)
	return s, nil
}

// LoadSession opens a stored session and replays it compactly (0008-MADR
// D19 item 5), then sends the usage snapshot, before answering.
func (a *Agent) LoadSession(ctx context.Context, p acp.LoadSessionRequest) (acp.LoadSessionResponse, error) {
	s, err := a.open(ctx, p.SessionId, p.Cwd, p.AdditionalDirectories, p.McpServers, p.SystemPrompt)
	if err != nil {
		return acp.LoadSessionResponse{}, err
	}
	a.mu.Lock()
	conn := a.conn
	a.mu.Unlock()
	out := newUpdates(ctx, conn, s.id)
	st := s.state()
	for _, u := range replayUpdates(st, a.byName, s.cwd, a.offersCompaction(s.id)) {
		out.send(u)
	}
	if out.err != nil {
		return acp.LoadSessionResponse{}, out.err
	}
	if err := a.startFrames(ctx, conn, s); err != nil {
		return acp.LoadSessionResponse{}, err
	}
	meta, err := resumedMeta(st)
	if err != nil {
		return acp.LoadSessionResponse{}, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	return acp.LoadSessionResponse{ConfigOptions: a.configOptions(s), Modes: modeState(s.mode, s.allowBypass), Meta: meta}, nil
}

// resumedMeta is a resumed session's _meta.gobble: its title (the name,
// else the first prompt) and the number of user and assistant messages on
// its path, which ACP's SessionInfo does not carry (0002-PLAN Phase 4,
// deviation of 2026-10-06).
func resumedMeta(st state) (map[string]any, error) {
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
	return jsonx.SetMember(nil, gobble, resumedGobble{Messages: n, Title: cutTitle(title)})
}

// resumedGobble is a resumed session's _meta.gobble (0039-MADR D15).
type resumedGobble struct {
	Messages int    `json:"messages"`
	Title    string `json:"title"`
}

// ResumeSession opens a stored session with no replay.
func (a *Agent) ResumeSession(ctx context.Context, p acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	s, err := a.open(ctx, p.SessionId, p.Cwd, p.AdditionalDirectories, p.McpServers, p.SystemPrompt)
	if err != nil {
		return acp.ResumeSessionResponse{}, err
	}
	a.mu.Lock()
	conn := a.conn
	a.mu.Unlock()
	if err := a.startFrames(ctx, conn, s); err != nil {
		return acp.ResumeSessionResponse{}, err
	}
	meta, err := resumedMeta(s.state())
	if err != nil {
		return acp.ResumeSessionResponse{}, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	return acp.ResumeSessionResponse{ConfigOptions: a.configOptions(s), Modes: modeState(s.mode, s.allowBypass),
		Meta: meta}, nil
}

// ListSessions lists the sessions with a file, and the live ones of this
// process without one yet, newest first. A stored session the SDK's rule
// refuses, a cwd not absolute on this host, is left out and logged once,
// so the rest still list (0019-MADR D5); a live session's cwd came through
// the SDK, so it is always absolute.
func (a *Agent) ListSessions(ctx context.Context, p acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	f := session.Filter{}
	if p.Cwd != nil {
		f.Cwd = *p.Cwd
	}
	seen := map[acp.SessionId]bool{}
	out := []acp.SessionInfo{}
	for sum, err := range a.store.List(ctx, f) {
		if err != nil {
			return acp.ListSessionsResponse{}, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
		}
		info := sessionInfo(sum)
		if err := info.Validate(); err != nil {
			a.notListed(ctx, info, err)
			continue
		}
		seen[info.SessionId] = true
		out = append(out, info)
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

// notListed logs, once per session, a stored session session/list leaves
// out.
func (a *Agent) notListed(ctx context.Context, info acp.SessionInfo, err error) {
	a.mu.Lock()
	logged := a.unlisted[info.SessionId]
	if a.unlisted == nil {
		a.unlisted = map[acp.SessionId]bool{}
	}
	a.unlisted[info.SessionId] = true
	a.mu.Unlock()
	if !logged {
		a.logger.WarnContext(ctx, "session: not listed: its cwd is not absolute on this host",
			slog.String("session", string(info.SessionId)), slog.String("cwd", info.Cwd), slog.String("reason", err.Error()))
	}
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
	return tool.Env{Cwd: s.cwd, Roots: append(roots, s.dirs...), NoAutoResize: s.noAutoResize}
}

// takesImages is whether a model accepts an image: its facts' input holds
// image, or the facts are unknown and the provider takes images
// (0036-MADR D8).
func takesImages(info llm.Model, p llm.Provider) bool {
	if slices.Contains(info.Input, "image") {
		return true
	}
	if len(info.Input) > 0 {
		return false
	}
	return p != nil && p.Capabilities().ImageInput
}

// markers are the 0003-MADR markers a session's commands see: AI_AGENT, and
// the session, its file, the provider, the model and the thinking level.
// An empty value is left out.
func markers(id, file, provider, model, think string) []string {
	out := []string{"AI_AGENT=gobble"}
	for _, kv := range [][2]string{
		{"GOBBLE_SESSION_ID", id}, {"GOBBLE_SESSION_FILE", file}, {"GOBBLE_PROVIDER", provider},
		{"GOBBLE_MODEL", model}, {"GOBBLE_THINKING_LEVEL", think},
	} {
		if kv[1] != "" {
			out = append(out, kv[0]+"="+kv[1])
		}
	}
	return out
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
func (a *Agent) CloseSession(ctx context.Context, p acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	a.mu.Lock()
	s, ok := a.sessions[p.SessionId]
	delete(a.sessions, p.SessionId)
	var idle chan struct{}
	if ok {
		idle = s.idle
	}
	a.mu.Unlock()
	if !ok {
		return acp.CloseSessionResponse{}, invalidParams(p.SessionId, "unknown session")
	}
	// The SDK has cancelled a running prompt; its last entries are written
	// before the log closes (0016-MADR A1).
	if idle != nil {
		t := time.NewTimer(closeWait)
		select {
		case <-idle:
		case <-t.C:
			a.logger.WarnContext(ctx, "session/close: the running turn did not release in time; closing its log",
				slog.String("session", string(p.SessionId)), slog.Duration("waited", closeWait))
		}
		t.Stop()
	}
	a.endDiscovery(s)
	a.closeMCP(s.mcp)
	a.closeTitle(s)
	if err := s.log.Close(); err != nil {
		return acp.CloseSessionResponse{}, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	return acp.CloseSessionResponse{}, nil
}

// SetSessionConfigOption sets the model or the thinking level for the
// session's next prompt, and records the change (model_change,
// thinking_level_change).
func (a *Agent) SetSessionConfigOption(ctx context.Context, p acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	v := p.ValueId
	if v == nil {
		return acp.SetSessionConfigOptionResponse{}, acp.NewInvalidParams(errData(acpext.ErrorData{Reason: "only select options are offered"}))
	}
	a.mu.Lock()
	s, ok := a.sessions[v.SessionId]
	a.mu.Unlock()
	if !ok {
		return acp.SetSessionConfigOptionResponse{}, invalidParams(v.SessionId, "unknown session")
	}
	opts, err := a.setConfig(ctx, s, string(v.ConfigId), string(v.Value))
	if se, ok := errors.AsType[*switchError](err); ok {
		return acp.SetSessionConfigOptionResponse{}, invalidParams(s.id, se.Error())
	}
	if err != nil {
		return acp.SetSessionConfigOptionResponse{}, err
	}
	return acp.SetSessionConfigOptionResponse{ConfigOptions: opts}, nil
}
