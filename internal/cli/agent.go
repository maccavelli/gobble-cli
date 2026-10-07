package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// appName is the client name gobble gives providers and agents (0003-MADR).
const appName = "gobble"

// modelProvider is the model the agent builds on its first prompt: the first
// provider whose standard credential variable is set (0002-PLAN Phase 3,
// owner's decision of 2026-10-05). Tests replace it.
var modelProvider = func(e *runEnv) func() (llm.Provider, error) {
	return func() (llm.Provider, error) {
		p, _, err := provider.Ambient(os.Getenv, provider.Options{ClientName: appName, ClientVersion: identity().Version, Logger: e.log()})
		return p, err
	}
}

// agentOptions are gobble's agent's options, for `gobble acp` and the
// in-process agent alike: the ambient provider and its model choice, and
// the session store in the data directory.
func (e *runEnv) agentOptions() acpserver.Options {
	return acpserver.Options{
		Version:  identity().Version,
		Provider: modelProvider(e),
		Models:   modelChoice,
		Tools:    e.tools,
		Store:    e.sessionStore(),
		MCP: acpserver.MCPOptions{
			Config: mcpConfigPath(), Off: e.mcpOff, Allow: e.mcpAllow, UserAgent: identity().UserAgent(),
		},
	}
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

// modelChoice is the ambient provider's curated models, its first the
// default; none without a credential.
func modelChoice() acpserver.ModelChoice {
	sel, err := provider.Choose(os.Getenv)
	if err != nil {
		return acpserver.ModelChoice{}
	}
	return acpserver.ModelChoice{Provider: sel.ID, Models: provider.Models(sel.ID), Default: sel.Model}
}

// sessionStore is the store sessions are kept in: <data>/sessions
// (0003-MADR), laid out as Pi lays out its own. Without a data directory
// sessions are kept in memory, with a warning.
func (e *runEnv) sessionStore() session.Store {
	if e.store != nil {
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
		return acpserver.Serve(ctx, in, out, acpserver.ServeOptions{Agent: e.agentOptions(), Logger: e.log()})
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
}

// toolSet is the in-process agent's tools from -t/--tools, --exclude-tools
// and --no-tools, with Pi's rules (0002-PLAN Phase 5): -t is an allowlist
// over every tool, MCP tools included; --exclude-tools removes names; and
// --no-tools turns everything off. An MCP name is accepted unchecked, since
// MCP tools exist only once their server connects; any other unknown name
// is a usage error. When the flags leave no MCP name selectable, no server
// connects. The contradiction of --no-tools with the others is checked in
// validate.
func (f *SharedFlags) toolSet() (toolChoice, error) {
	known := builtin.Names()
	for _, n := range append(slices.Clone(f.Tools), f.ExcludeTools...) {
		if !slices.Contains(known, n) && !strings.HasPrefix(n, mcpPrefix) {
			return toolChoice{}, usageErrorf("unknown tool %q (the tools are %s)", n, strings.Join(known, ", "))
		}
	}
	switch {
	case f.NoTools:
		return toolChoice{builtins: []tool.Tool{}, mcpOff: true}, nil
	case len(f.Tools) == 0 && len(f.ExcludeTools) == 0:
		return toolChoice{}, nil
	}
	var out []tool.Tool
	for _, t := range builtin.Tools() {
		name := t.Spec().Name
		if (len(f.Tools) == 0 || slices.Contains(f.Tools, name)) && !slices.Contains(f.ExcludeTools, name) {
			out = append(out, t)
		}
	}
	if out == nil {
		out = []tool.Tool{}
	}
	listed := slices.DeleteFunc(slices.Clone(f.Tools), func(n string) bool { return !strings.HasPrefix(n, mcpPrefix) })
	excluded := slices.Clone(f.ExcludeTools)
	allowlist := len(f.Tools) > 0
	return toolChoice{
		builtins: out,
		mcpOff:   allowlist && len(listed) == 0,
		mcpAllow: func(name string) bool {
			return (!allowlist || slices.Contains(listed, name)) && !slices.Contains(excluded, name)
		},
	}, nil
}

// noCredentialMessage names the variables the ambient provider reads.
func noCredentialMessage() string {
	return "no model credential: set one of " + strings.Join(provider.CredentialVars(), ", ")
}
