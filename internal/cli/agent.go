package cli

import (
	"cmp"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// appName is the client name gobble gives providers and agents (0003-MADR).
const appName = "gobble"

// modelProvider is the model the agent builds on its first prompt: the
// run's provider, --model's or the defaultProvider setting's, or else the
// first provider with a credential (0002-PLAN Phase 3, owner's decision of
// 2026-10-05; 0019-MADR D15; 0005-PLAN F4-2c), its key found in Pi's order
// and resolved now (0005-PLAN F4-3a), or its stored OAuth login (0005-PLAN
// F4-3b). Each session's provider names the session (0021-MADR D4), and
// retries as its settings say (0005-PLAN F5-1). Tests replace it.
var modelProvider = func(e *runEnv) func(sessionID string, retry llm.Retry) (llm.Provider, error) {
	return func(sessionID string, retry llm.Retry) (llm.Provider, error) {
		o := provider.Options{ClientName: appName, ClientVersion: identity().Version, Logger: e.log(), SessionID: sessionID,
			Logins: e.logins(), Retry: &retry}
		if sel, err := provider.ChooseWith(e.hasCredential, e.chosenProvider()); err == nil {
			o = e.providerOptions(sel.ID, o) // models.json's base URL and headers (0005-PLAN F4-2b)
			if src, ok := e.credentials().Has(sel.ID); ok {
				o.CredentialSource = string(src) // a rejected key names it (0005-PLAN F4-3a)
			}
		}
		p, _, err := ambientProvider(e.credentialKey, e.chosenProvider(), o)
		return p, err
	}
}

// ambientProvider builds the ambient provider; a test replaces it to see
// the options it is given.
var ambientProvider = provider.AmbientWith

// agentOptions are gobble's agent's options, for `gobble acp` and the
// in-process agent alike: the ambient provider and its model choice, and
// the session store in the data directory.
func (e *runEnv) agentOptions() acpserver.Options {
	e.startModelListing()
	if err := provider.SnapshotErr(); err != nil { // models are then unknown, never refused (0005-PLAN F4-2a deviation 1)
		e.log().Warn("model catalog", slog.Any("error", err))
	}
	return acpserver.Options{
		Version:  identity().Version,
		Provider: modelProvider(e),
		Models:   e.modelChoice,
		// The recent models, and the SDK's matcher for /model with words
		// (0025-MADR D3, D5).
		Remember:     e.rememberModel,
		SearchModels: provider.Search,
		// What the catalog knows of a model, with the provider's key or login,
		// found as modelProvider finds it, for its listing (0005-PLAN F4-2a,
		// F4-3a, F4-3b).
		ModelInfo: func(id, model string) llm.Model {
			var logins provider.LoginStore
			if e.knownLogin(id) {
				logins = e.logins()
			}
			return provider.ModelInfo(id, e.knownKey(id), logins, model, e.modelsJSON())
		},
		// --model and --thinking win over every session's own (0005-PLAN F4-2c).
		Override: acpserver.Override{Provider: e.pick.Provider, Model: e.pick.Model, Thinking: e.pick.Thinking, Mode: e.mode},
		Tools:    e.tools,
		// The tool flags select request_permissions too (0005-PLAN F6-5).
		NoRequestPermissions: e.noGrants,
		Store:                e.sessionStore(),
		MCP: acpserver.MCPOptions{
			Config: mcpConfigPath(), Off: e.mcpOff, Allow: e.mcpAllow, UserAgent: identity().UserAgent(),
		},
		Prompt: e.promptOptions(),
		// Each session's settings, when it starts (0018-MADR D17), its
		// personality resolved (0022-PLAN P3).
		Settings: func(cwd string) acpserver.SessionSettings {
			ss := toSession(e.sessionSettings(cwd).Settings)
			ss.Personality = e.personalityName(ss.Personality)
			// The project's trust, and a trusted project's own overrides
			// (0005-PLAN F6-1).
			ss.ProjectRoot, ss.Trust = e.projectTrust(cwd)
			if ss.Trust.Trusted {
				ss.Project = e.projectOverrides(cwd)
			}
			return ss
		},
		// An agent method per provider that takes a credential, answered from
		// gobble's credentials as gobble auth check finds them (0016-MADR A2).
		Auth: acpserver.AuthOptions{Providers: authProviders, Has: func(id string) bool {
			_, ok := e.credential(id)
			return ok
		}},
		SaveSetting:   saveGlobalSetting,
		Personalities: e.personalities(),
		// The user's compaction prompts (0005-PLAN F5-4 decision 8).
		CompactionPrompts: e.prompts().compaction,
		// The small model that names a new session (0005-PLAN F5-5).
		Title: e.titleOptions(),
		// /trust saves to trust.json, as gobble trust does (0005-PLAN F6-1).
		Trust: trustOptions(),
		// The "always" answers and /permissions save to permissions.json
		// and the global settings.yaml (0005-PLAN F6-2b decision E).
		Permissions: permissionOptions(),
	}
}

// saveGlobalSetting writes path's value to the global settings file, as
// gobble config set does, creating the file from the template if it is
// missing (0005-PLAN F5-1 decision 6).
func saveGlobalSetting(path, value string) error {
	file, layer, err := settingsFile(false)
	if err != nil {
		return err
	}
	if err := config.Check(path, value, layer); err != nil {
		return err
	}
	if err := prepare(file, layer, false); err != nil {
		return err
	}
	return config.Set(file, path, value)
}

// promptOptions are the prompt flags' options with the config directory,
// which holds the user's AGENTS.md, SYSTEM.md, APPEND_SYSTEM.md and skills/,
// and the home directory, which holds ~/.agents/skills; each none when it
// cannot be resolved (0005-PLAN F3-1, F3-2).
func (e *runEnv) promptOptions() acpserver.PromptOptions {
	o := e.prompt
	if dirs, _, err := systemDirs(); err == nil {
		o.ConfigDir = dirs.Config
	}
	// The user's tools and rules sections (0022-PLAN P5).
	o.ToolsSection, o.RulesSection = e.prompts().sections[promptTools], e.prompts().sections[promptRules]
	if home, err := os.UserHomeDir(); err == nil {
		o.Home = home
	}
	return o
}

// mcpConfigPath is <config>/mcp.json (0003-MADR), or "" when the config
// directory cannot be resolved.
func mcpConfigPath() string {
	dirs, _, err := systemDirs()
	if err != nil {
		return ""
	}
	return filepath.Join(dirs.Config, "mcp.json")
}

// modelChoice is the run's provider's listed models, then models.json's,
// the listing's first the default, or its static models and
// StaticModels[0] when the listing failed; none without a credential
// (0020-MADR D3, D4). The defaultModel setting may name the default, and
// --model's model and the default join the models when the listing lacks
// them (0005-PLAN F4-2c).
func (e *runEnv) modelChoice() acpserver.ModelChoice {
	sel, err := provider.ChooseWith(e.hasCredential, e.chosenProvider())
	if err != nil {
		return acpserver.ModelChoice{}
	}
	ids, def := models(sel, e.modelListing(sel).get())
	ids = withModelsJSON(ids, e.modelsJSON(), sel.ID)
	def = cmp.Or(e.defaultModel(sel.ID, ids), def)
	for _, m := range []string{e.pick.Model, def} {
		if m != "" && !slices.Contains(ids, m) {
			ids = append(ids, m)
		}
	}
	return acpserver.ModelChoice{Provider: sel.ID, Models: ids, Default: def,
		Providers: e.providerChoices(sel.ID, ids), Recent: e.recentModels()}
}

// sessionStore is the store sessions are kept in: <data>/sessions
// (0003-MADR), laid out as Pi lays out its own. Without a data directory
// sessions are kept in memory, with a warning.
func (e *runEnv) sessionStore() session.Store {
	if e.store != nil {
		return e.store
	}
	// sessionDir gives the flat store --session-dir gives; the flag, set in
	// configureStore, wins (0018-MADR D17). The process settings are
	// resolved by now, for the run's cwd.
	if dir := e.processSettings("").Settings.SessionDir; dir != "" {
		e.store = jsonl.New(expandHome(dir), jsonl.Options{Flat: true, Logger: e.log()})
		return e.store
	}
	dirs, _, err := systemDirs()
	if err != nil {
		e.out.Warnf("sessions are not saved: %v", err)
		e.store = session.NewMemoryStore()
		return e.store
	}
	e.store = jsonl.New(filepath.Join(dirs.Data, "sessions"), jsonl.Options{Logger: e.log()})
	return e.store
}

// agentServe is the agent gobble's own modes drive: gobble's agent, in this
// process, spoken to over ACP like any other client (0002-MADR rule 3).
// Tests replace it with a scripted agent.
var agentServe = func(e *runEnv) acpclient.ServeFunc {
	return func(ctx context.Context, in io.Reader, out io.Writer) error {
		o := e.agentOptions()
		// gobble's own modes tell the user of an untrusted project
		// themselves (0005-PLAN F6-1 decision 9).
		o.Trust.Quiet = true
		return acpserver.Serve(ctx, in, out, acpserver.ServeOptions{Agent: o, Logger: e.log()})
	}
}

// startAgent starts the agent and sends initialize, declining every
// permission request (print mode). A dropped notification is reported
// once per drop on stderr; the session goes on.
func (e *runEnv) startAgent(ctx context.Context) (*acpclient.Conn, error) {
	return e.startAgentWith(ctx, nil)
}

// startAgentWith is startAgent with permission requests answered by
// permission: the line session's prompt (0008-MADR D16).
func (e *runEnv) startAgentWith(ctx context.Context, permission func(context.Context, acpclient.PermissionRequest) acpclient.PermissionAnswer) (*acpclient.Conn, error) {
	conn, err := acpclient.Start(ctx, agentServe(e), acpclient.Options{
		Name:    appName,
		Version: identity().Version,
		Logger:  e.log(),
		OnDrop: func(method string, total uint64) {
			e.out.Warnf("dropped %d ACP notifications (%s)", total, method)
		},
		Permission: permission,
	})
	if err != nil {
		return nil, failf("start the agent: %v", err)
	}
	return conn, nil
}

// mcpPrefix starts every MCP tool's name.
const mcpPrefix = "mcp__"

// toolChoice is the tools the flags select: the built-ins (nil is all of
// them), and the MCP tools. mcpOff connects no MCP server; mcpAllow, when
// not nil, keeps only the MCP tools it accepts.
type toolChoice struct {
	builtins []tool.Tool
	mcpOff   bool
	mcpAllow func(name string) bool
	// noGrants leaves request_permissions out (0005-PLAN F6-5 deviation 1).
	noGrants bool
}

// toolSet is the in-process agent's tools from -t/--tools, --exclude-tools
// and --no-tools, with Pi's rules (0002-PLAN Phase 5): -t is an allowlist
// over every tool, MCP tools included; --exclude-tools removes names; and
// --no-tools turns everything off. An MCP name is accepted unchecked, since
// MCP tools exist only once their server connects; so is an MCP resource
// tool's name, which is filtered as an MCP tool is (0005-PLAN F1d-2). Any
// other unknown name is a usage error. When the flags leave no MCP name
// selectable, no server connects. The contradiction of --no-tools with the others is checked in
// validate. builtins are the built-in tools to choose from, with the
// process settings' options (0018-PLAN P5 deviation 15).
func (f *SharedFlags) toolSet(builtins []tool.Tool) (toolChoice, error) {
	known := append(builtin.Names(), acpserver.RequestPermissionsTool)
	for _, n := range append(slices.Clone(f.Tools), f.ExcludeTools...) {
		if !slices.Contains(known, n) && !isMCPName(n) {
			return toolChoice{}, usageErrorf("unknown tool %q (the tools are %s)", n, strings.Join(known, ", "))
		}
	}
	switch {
	case f.NoTools:
		return toolChoice{builtins: []tool.Tool{}, mcpOff: true, noGrants: true}, nil
	case len(f.Tools) == 0 && len(f.ExcludeTools) == 0:
		return toolChoice{builtins: builtins}, nil
	}
	var out []tool.Tool
	for _, t := range builtins {
		name := t.Spec().Name
		if (len(f.Tools) == 0 || slices.Contains(f.Tools, name)) && !slices.Contains(f.ExcludeTools, name) {
			out = append(out, t)
		}
	}
	if out == nil {
		out = []tool.Tool{}
	}
	listed := slices.DeleteFunc(slices.Clone(f.Tools), func(n string) bool { return !isMCPName(n) })
	excluded := slices.Clone(f.ExcludeTools)
	allowlist := len(f.Tools) > 0
	return toolChoice{
		builtins: out,
		mcpOff:   allowlist && len(listed) == 0,
		noGrants: allowlist && !slices.Contains(f.Tools, acpserver.RequestPermissionsTool) ||
			slices.Contains(f.ExcludeTools, acpserver.RequestPermissionsTool),
		mcpAllow: func(name string) bool {
			return (!allowlist || slices.Contains(listed, name)) && !slices.Contains(excluded, name)
		},
	}, nil
}

// isMCPName reports whether n names a tool an MCP server brings: an
// mcp__ tool, or one of the MCP resource tools.
func isMCPName(n string) bool {
	return strings.HasPrefix(n, mcpPrefix) || slices.Contains(builtin.ResourceNames(), n)
}

// toolsWith is builtin.ToolsWith; tests replace it to see the options.
var toolsWith = builtin.ToolsWith

// builtinTools are the built-in tools with gobble's options: the bash keys
// of the process settings (0018-MADR D17), and the whole output of long
// commands kept in <state>/output (0005-PLAN F1b), or in the OS temp
// directory when the state directory cannot be resolved.
func (e *runEnv) builtinTools(s config.Settings) []tool.Tool {
	o := builtinOptions(s)
	if dirs, _, err := systemDirs(); err == nil {
		o.OutputDir = filepath.Join(dirs.State, "output")
	}
	o.Descriptions = e.prompts().descriptions // the user's (0022-PLAN P5)
	return toolsWith(o)
}

// builtinOptions are the bash keys' options, ~ expanded in shellPath.
func builtinOptions(s config.Settings) builtin.Options {
	return builtin.Options{
		ShellPath: expandHome(s.ShellPath), ShellCommandPrefix: s.ShellCommandPrefix,
		MaxTimeout: s.MaxTimeout(), UseClientTerminal: s.Tools.Bash.UseClientTerminal,
	}
}

// defaultProvider is the defaultProvider setting, a process key
// (0019-MADR D15); the process settings are resolved by now, for the run's
// cwd, as sessionStore reads them.
func (e *runEnv) defaultProvider() string { return e.processSettings("").Settings.DefaultProvider }

// authProviders are ACP authentication's providers: each that takes a
// credential, in the SDK's order, with its label and the gobble command
// that stores one (0016-MADR A2).
func authProviders() []acpserver.AuthProvider {
	var out []acpserver.AuthProvider
	for _, id := range provider.KeyProviders() {
		login := provider.HasLogin(id) || provider.HasKeyLogin(id)
		command := "gobble auth login " + id + " --api-key -"
		if login {
			command = "gobble auth login " + id
		}
		out = append(out, acpserver.AuthProvider{ID: id, Name: provider.Label(id), Command: command, Login: login})
	}
	return out
}
