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
	}
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

// startAgent starts the agent and sends initialize. A dropped notification
// is reported once per drop on stderr; the session goes on.
func (e *runEnv) startAgent(ctx context.Context) (*acpclient.Conn, error) {
	conn, err := acpclient.Start(ctx, agentServe(e), acpclient.Options{
		Name:    appName,
		Version: identity().Version,
		Logger:  e.log(),
		OnDrop: func(method string, total uint64) {
			e.out.Warnf("dropped %d ACP notifications (%s)", total, method)
		},
	})
	if err != nil {
		return nil, failf("start the agent: %v", err)
	}
	return conn, nil
}

// toolSet is the in-process agent's tools from -t/--tools, --exclude-tools
// and --no-tools: nil (the built-ins) when none is given. An unknown name is
// a usage error; the contradiction of --no-tools with the others is checked
// in validate.
func (f *SharedFlags) toolSet() ([]tool.Tool, error) {
	known := builtin.Names()
	for _, n := range append(slices.Clone(f.Tools), f.ExcludeTools...) {
		if !slices.Contains(known, n) {
			return nil, usageErrorf("unknown tool %q (the tools are %s)", n, strings.Join(known, ", "))
		}
	}
	switch {
	case f.NoTools:
		return []tool.Tool{}, nil
	case len(f.Tools) == 0 && len(f.ExcludeTools) == 0:
		return nil, nil
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
	return out, nil
}

// noCredentialMessage names the variables the ambient provider reads.
func noCredentialMessage() string {
	return "no model credential: set one of " + strings.Join(provider.CredentialVars(), ", ")
}
