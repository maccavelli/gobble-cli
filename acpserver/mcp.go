package acpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/mcpclient"
	"github.com/maccavelli/gobble-cli/tool"
)

// DefaultStartupWait is how long a session's first prompt waits for MCP
// servers still connecting, as in Pi.
const DefaultStartupWait = 10 * time.Second

// MCPOptions configure the MCP servers sessions connect to (0002-PLAN
// Phase 5).
type MCPOptions struct {
	// Config is the mcp.json read at each session start; "" reads none.
	Config string
	// Off connects no server, from the file or the session: the CLI's
	// tool flags left no MCP tool selectable.
	Off bool
	// Allow reports whether an MCP tool, by its mcp__ name, is offered;
	// nil offers every one.
	Allow func(name string) bool
	// UserAgent is the User-Agent of HTTP requests (0003-MADR).
	UserAgent string
	// StartupWait bounds the first prompt's wait; zero is
	// DefaultStartupWait.
	StartupWait time.Duration
	// Getenv resolves the file's $NAME values; nil is os.Getenv.
	Getenv func(string) string
}

// liveMCP is a session's MCP servers and what has been reported of them.
type liveMCP struct {
	m        *mcpclient.Manager
	notes    []note // config errors and exposure notes, logged once
	waited   bool
	reported map[string]bool
}

// note is one start-up diagnostic: a config error, or a server whose
// exposure settings are not applied yet.
type note struct {
	exposure bool
	text     string // the error, or the server's name
}

// mcpServers are a session's servers: the file's, in file order, with a
// server the session names the same replacing it in place, then the
// session's others in request order. An entry the agent does not offer is
// invalid params.
func (a *Agent) mcpServers(id acp.SessionId, req []acp.McpServer) ([]mcpclient.Server, []note, error) {
	if a.mcp.Off {
		return nil, nil, nil
	}
	var (
		servers []mcpclient.Server
		notes   []note
	)
	if a.mcp.Config != "" {
		c := mcpclient.Load(a.mcp.Config)
		servers = c.Servers
		for _, e := range c.Errors {
			notes = append(notes, note{text: e})
		}
	}
	for _, r := range req {
		s, err := sessionServer(r)
		if err != nil {
			return nil, nil, invalidParams(id, err.Error())
		}
		if i := slices.IndexFunc(servers, func(x mcpclient.Server) bool { return x.Name == s.Name }); i >= 0 {
			servers[i] = s
			continue
		}
		servers = append(servers, s)
	}
	for _, s := range servers {
		if s.Enabled && (s.Exposure != "" || len(s.ToolExposure) > 0) {
			notes = append(notes, note{exposure: true, text: s.Name})
		}
	}
	return servers, notes, nil
}

// sessionServer converts one ACP server. SSE and ACP-transport servers
// are not offered, so a client must not send them.
func sessionServer(r acp.McpServer) (mcpclient.Server, error) {
	switch {
	case r.Stdio != nil:
		env := make([]mcpclient.Pair, len(r.Stdio.Env))
		for i, e := range r.Stdio.Env {
			env[i] = mcpclient.Pair{Name: e.Name, Value: e.Value}
		}
		return mcpclient.SessionStdio(r.Stdio.Name, r.Stdio.Command, r.Stdio.Args, env)
	case r.Http != nil:
		headers := make([]mcpclient.Pair, len(r.Http.Headers))
		for i, h := range r.Http.Headers {
			headers[i] = mcpclient.Pair{Name: h.Name, Value: h.Value}
		}
		return mcpclient.SessionHTTP(r.Http.Name, r.Http.Url, headers)
	case r.Sse != nil:
		return mcpclient.Server{}, fmt.Errorf("MCP server %q: the SSE transport is not offered (mcpCapabilities.sse is false)", r.Sse.Name)
	case r.Acp != nil:
		return mcpclient.Server{}, fmt.Errorf("MCP server %q: the ACP transport is not offered (mcpCapabilities.acp is false)", r.Acp.Name)
	}
	return mcpclient.Server{}, errors.New("an MCP server has no transport")
}

// startMCP starts a session's servers in the background.
func (a *Agent) startMCP(ctx context.Context, cwd string, servers []mcpclient.Server, notes []note) *liveMCP {
	if len(servers) == 0 && len(notes) == 0 {
		return nil
	}
	m := mcpclient.Start(ctx, servers, mcpclient.Options{
		Cwd: cwd, ClientName: gobble, ClientVersion: a.version, UserAgent: a.mcp.UserAgent, Getenv: a.mcp.Getenv,
	})
	return &liveMCP{m: m, notes: notes, reported: map[string]bool{}}
}

// tools are the MCP tools for a prompt, then the MCP resource tools while
// a connected server offers resources, each kept by the tool filter. The
// session's first prompt waits for servers still connecting, up to the
// start-up wait; then each problem is logged once, when first seen.
func (a *Agent) mcpTools(ctx context.Context, l *liveMCP) []tool.Tool {
	if l == nil {
		return nil
	}
	first := !l.waited
	if first {
		l.waited = true
		wait := a.mcp.StartupWait
		if wait <= 0 {
			wait = DefaultStartupWait
		}
		wctx, cancel := context.WithTimeout(ctx, wait)
		if err := l.m.Wait(wctx); err != nil {
			a.logger.DebugContext(ctx, "mcp: start-up wait ended", slog.String("reason", err.Error()))
		}
		cancel()
		for _, n := range l.notes {
			if n.exposure {
				a.logger.WarnContext(ctx, "mcp: exposure and toolExposure apply from 0005-PLAN F8; its tools are declared directly", slog.String("server", n.text))
			} else {
				a.logger.WarnContext(ctx, "mcp: config error", slog.String("error", n.text))
			}
		}
	}
	for _, st := range l.m.Statuses() {
		if l.reported[st.Name] {
			continue
		}
		switch st.State {
		case mcpclient.StateFailed, mcpclient.StateNeedsAuth:
			l.reported[st.Name] = true
			a.logger.WarnContext(ctx, "mcp: server needs attention", slog.String("server", st.Name),
				slog.String("state", string(st.State)), slog.String("error", mcpclient.FirstLine(st.Error)))
		case mcpclient.StateConnecting:
			if first {
				a.logger.WarnContext(ctx, "mcp: server still connecting; its tools join a later prompt", slog.String("server", st.Name))
			}
		}
	}
	tools := l.m.Tools(a.mcp.Allow)
	for _, t := range l.m.ResourceTools() {
		if a.mcp.Allow == nil || a.mcp.Allow(t.Spec().Name) {
			tools = append(tools, t)
		}
	}
	return tools
}

// closeMCP closes a session's servers. A server's exit status is logged,
// not returned: closing the session has succeeded.
func (a *Agent) closeMCP(l *liveMCP) {
	if l == nil {
		return
	}
	if err := l.m.Close(); err != nil {
		a.logger.Debug("mcp: close", slog.String("error", err.Error()))
	}
}
