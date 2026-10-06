package cli

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"iter"
	"os"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/mcpclient"
)

// MCPCmd configures and checks the MCP servers of <config>/mcp.json. These
// are configuration commands, not ACP methods (0002-MADR); a running
// session sees a change from its next session/new. They follow `pi mcp`
// (0002-PLAN Phase 5); login, logout and get arrive with 0005-PLAN F8.
type MCPCmd struct {
	Add    MCPAddCmd    `cmd:"" help:"Add or replace a server in mcp.json."`
	Remove MCPRemoveCmd `cmd:"" help:"Remove a server from mcp.json."`
	List   MCPListCmd   `cmd:"" help:"Connect to every enabled server and print its state, tools and errors; exit 1 while anything is wrong."`
}

// MCPAddCmd writes one server entry, validated as Pi validates it.
type MCPAddCmd struct {
	Server            string   `arg:"" complete:"none" help:"Server name: letters, digits, _ and -."`
	Command           []string `arg:"" optional:"" passthrough:"" complete:"none" help:"The stdio server's command and its arguments, after --."`
	URL               string   `name:"url" complete:"none" placeholder:"URL" help:"Streamable HTTP server URL, instead of a command."`
	Env               []string `name:"env" sep:"none" complete:"none" placeholder:"KEY=VALUE" help:"Environment variable for a stdio server; repeatable. A value may name a variable as $NAME."`
	Cwd               string   `name:"cwd" complete:"dir" placeholder:"DIR" help:"Working directory for a stdio server, relative to the session's."`
	Header            []string `name:"header" sep:"none" complete:"none" placeholder:"KEY=VALUE" help:"HTTP header; repeatable. A value may name a variable as $NAME."`
	BearerTokenEnvVar string   `name:"bearer-token-env-var" complete:"none" placeholder:"NAME" help:"Send an Authorization header of Bearer and the variable NAME's value, read when the server connects."`
	Exposure          string   `enum:",codemode,codemode-deferred,deferred,direct,hidden" default:"" help:"Exposure, kept in the file and applied from 0005-PLAN F8: codemode, codemode-deferred, deferred, direct or hidden."`
}

// Run validates the entry, then writes it.
func (c *MCPAddCmd) Run(e *runEnv) error {
	const usage = "usage: gobble mcp add <server> [flags] (--url <url> | -- <command> [args...])"
	if (c.URL == "") == (len(c.Command) == 0) {
		return usageErrorf("%s", usage)
	}
	for _, m := range []struct {
		set, http bool
		name      string
	}{
		{len(c.Header) > 0, true, "header"}, {c.BearerTokenEnvVar != "", true, "bearer-token-env-var"},
		{len(c.Env) > 0, false, "env"}, {c.Cwd != "", false, "cwd"},
	} {
		if m.set && m.http != (c.URL != "") {
			where := "stdio servers"
			if m.http {
				where = "HTTP servers (--url)"
			}
			return usageErrorf("--%s only applies to %s", m.name, where)
		}
	}
	entry, err := c.entry()
	if err != nil {
		return err
	}
	if _, err := mcpclient.Validate(c.Server, entry); err != nil {
		return usageErrorf("%v", err)
	}
	path := mcpConfigPath()
	if path == "" {
		return failf("the config directory cannot be resolved")
	}
	replaced, err := mcpclient.Add(path, c.Server, entry)
	if err != nil {
		return failf("could not update %s: %v", path, err)
	}
	verb := "Added"
	if replaced {
		verb = "Replaced"
	}
	return e.out.Result(fmt.Appendf(nil, "%s MCP server %q in %s.\nCheck it with: gobble mcp list\n", verb, c.Server, path))
}

// member is one name and value of an entry.
type member struct {
	name  string
	value any
}

// entry is the server's JSON in Pi's member order.
func (c *MCPAddCmd) entry() (jsontext.Value, error) {
	var ms []member
	add := func(name string, v any) { ms = append(ms, member{name, v}) }
	if c.URL != "" {
		add("url", c.URL)
		headers, err := pairs("header", c.Header)
		if err != nil {
			return nil, err
		}
		if c.BearerTokenEnvVar != "" {
			headers = setPair(headers, "Authorization", "Bearer ${"+c.BearerTokenEnvVar+"}")
		}
		if len(headers) > 0 {
			add("headers", headers)
		}
	} else {
		add("command", c.Command[0])
		if len(c.Command) > 1 {
			add("args", c.Command[1:])
		}
		env, err := pairs("env", c.Env)
		if err != nil {
			return nil, err
		}
		if len(env) > 0 {
			add("env", env)
		}
		if c.Cwd != "" {
			add("cwd", c.Cwd)
		}
	}
	if c.Exposure != "" {
		add("exposure", c.Exposure)
	}
	return encodeObject(ms)
}

// pairs reads repeated KEY=VALUE flags, in order.
func pairs(flag string, values []string) ([]mcpclient.Pair, error) {
	out := make([]mcpclient.Pair, 0, len(values))
	for _, v := range values {
		k, val, ok := strings.Cut(v, "=")
		if !ok || k == "" {
			return nil, usageErrorf("--%s expects KEY=VALUE, got %q", flag, v)
		}
		out = setPair(out, k, val)
	}
	return out, nil
}

// setPair sets name in place, or adds it last, as a JavaScript object does.
func setPair(ps []mcpclient.Pair, name, value string) []mcpclient.Pair {
	for i := range ps {
		if ps[i].Name == name {
			ps[i].Value = value
			return ps
		}
	}
	return append(ps, mcpclient.Pair{Name: name, Value: value})
}

// encodeObject encodes members as one JSON object, in order; a
// []mcpclient.Pair value is an object of strings.
func encodeObject(ms []member) (jsontext.Value, error) {
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(&buf)
	write := func(tok jsontext.Token) error { return enc.WriteToken(tok) }
	if err := write(jsontext.BeginObject); err != nil {
		return nil, err
	}
	for _, m := range ms {
		if err := write(jsontext.String(m.name)); err != nil {
			return nil, err
		}
		if ps, ok := m.value.([]mcpclient.Pair); ok {
			if err := write(jsontext.BeginObject); err != nil {
				return nil, err
			}
			for _, p := range ps {
				if err := write(jsontext.String(p.Name)); err != nil {
					return nil, err
				}
				if err := write(jsontext.String(p.Value)); err != nil {
					return nil, err
				}
			}
			if err := write(jsontext.EndObject); err != nil {
				return nil, err
			}
			continue
		}
		if err := json.MarshalEncode(enc, m.value); err != nil {
			return nil, err
		}
	}
	if err := write(jsontext.EndObject); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

// MCPRemoveCmd removes one server entry.
type MCPRemoveCmd struct {
	Server string `arg:"" complete:"mcpserver" help:"Server name."`
}

// Run removes the entry, or fails when the file does not define it.
func (c *MCPRemoveCmd) Run(e *runEnv) error {
	path := mcpConfigPath()
	if path == "" {
		return failf("the config directory cannot be resolved")
	}
	removed, err := mcpclient.Remove(path, c.Server)
	if err != nil {
		return failf("could not update %s: %v", path, err)
	}
	if !removed {
		return failf("no MCP server named %q in %s", c.Server, path)
	}
	return e.out.Result(fmt.Appendf(nil, "Removed MCP server %q from %s.\n", c.Server, path))
}

// MCPListCmd connects to every enabled server, as Pi's `pi mcp list` does.
type MCPListCmd struct {
	JSON bool `name:"json" help:"Print JSON."`
}

type listReport struct {
	Servers []mcpclient.Status `json:"servers"`
	Errors  []string           `json:"errors"`
}

// Run prints each server's state, transport, tools and error, then the
// file's config errors. It exits 1 when a config error exists or an
// enabled server is not connected.
func (c *MCPListCmd) Run(e *runEnv) error {
	path := mcpConfigPath()
	cfg := mcpclient.Load(path)
	cwd, err := os.Getwd()
	if err != nil {
		return failf("%v", err)
	}
	m := mcpclient.Start(e.ctx, cfg.Servers, mcpclient.Options{
		Cwd: cwd, ClientName: appName, ClientVersion: identity().Version, UserAgent: identity().UserAgent(),
	})
	waitErr := m.Wait(e.ctx)
	r := listReport{Servers: m.Statuses(), Errors: cfg.Errors}
	if r.Errors == nil {
		r.Errors = []string{}
	}
	_ = m.Close() //nolint:errcheck // a server's exit status once its state is read; list reports states, not exits
	if waitErr != nil {
		return waitErr
	}
	failed := len(cfg.Errors) > 0
	for _, s := range r.Servers {
		failed = failed || s.Enabled && s.State != mcpclient.StateConnected
	}
	var out []byte
	if c.JSON {
		if out, err = json.Marshal(r, jsontext.WithIndent("  ")); err != nil {
			return failf("mcp list: %v", err)
		}
		out = append(out, '\n')
	} else {
		out = listText(r, path)
	}
	if err := e.out.Result(out); err != nil {
		return err
	}
	if failed {
		return &exitError{code: ExitFailure}
	}
	return nil
}

// listText is Pi's text report, without the scope, with the exposure in
// force.
func listText(r listReport, path string) []byte {
	var b strings.Builder
	if len(r.Servers) == 0 && len(r.Errors) == 0 {
		fmt.Fprintf(&b, "No MCP servers configured. Add them to %s.\n", path)
	}
	for _, s := range r.Servers {
		state := string(s.State)
		switch s.State {
		case mcpclient.StateConnected:
			state = fmt.Sprintf("connected, %d tool", len(s.Tools))
			if len(s.Tools) != 1 {
				state += "s"
			}
		case mcpclient.StateNeedsAuth:
			state = "needs sign-in"
		}
		fmt.Fprintf(&b, "%s: %s (%s)\n  %s\n", s.Name, state, s.Exposure, s.Transport)
		if len(s.Tools) > 0 {
			fmt.Fprintf(&b, "  tools: %s\n", strings.Join(s.Tools, ", "))
		}
		if s.Error != "" && s.State != mcpclient.StateConnected {
			fmt.Fprintf(&b, "  %s\n", strings.ReplaceAll(s.Error, "\n", "\n  "))
		}
	}
	for _, e := range r.Errors {
		fmt.Fprintf(&b, "config error: %s\n", e)
	}
	return []byte(b.String())
}

// mcpServerCandidates are the names of mcp.json's valid servers, in file
// order. The file is only read: completion creates nothing.
func mcpServerCandidates(context.Context) iter.Seq[complete.Candidate] {
	return func(yield func(complete.Candidate) bool) {
		path := mcpConfigPath()
		if path == "" {
			return
		}
		for _, s := range mcpclient.Load(path).Servers {
			if !yield(complete.Candidate{Value: s.Name, Description: s.Transport()}) {
				return
			}
		}
	}
}
