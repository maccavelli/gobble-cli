package mcpclient

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/maccavelli/gobble-cli/tool"
)

// Manager is one session's servers: each connects in the background from
// Start, and the tools of those connected are offered from then on.
type Manager struct {
	conns []*conn

	mu sync.Mutex
	// owners maps each tool name given out to "<server>\x00<tool>", so a
	// name stays its tool's for the session, as Pi's toolOwners does.
	owners map[string]string
	tools  map[string]*mcpTool
}

// Start connects every enabled server in its own goroutine and returns at
// once. Disabled servers are listed, and never connected. The connections
// keep ctx's values but not its cancellation: they live until Close.
func Start(ctx context.Context, servers []Server, o Options) *Manager {
	base := context.WithoutCancel(ctx)
	m := &Manager{owners: map[string]string{}, tools: map[string]*mcpTool{}}
	for _, s := range servers {
		c := newConn(s, o)
		m.conns = append(m.conns, c)
		if !s.Enabled {
			c.state = StateDisabled
			close(c.done)
			continue
		}
		cctx, cancel := context.WithCancel(base)
		c.cancel = cancel
		go c.connect(cctx)
	}
	return m
}

// Wait waits until every connection has opened or failed, or ctx is done.
func (m *Manager) Wait(ctx context.Context) error {
	for _, c := range m.conns {
		select {
		case <-c.done:
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}
	return nil
}

// Tools are the connected servers' tools, in server order and then in the
// order each server lists them, named by ToolName across the whole set.
// allow, when not nil, keeps only the names it accepts.
func (m *Manager) Tools(allow func(name string) bool) []tool.Tool {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []tool.Tool
	current := map[string]bool{}
	for _, c := range m.conns {
		state, _, tools := c.status()
		if state != StateConnected {
			continue
		}
		for _, t := range tools {
			owner := c.server.Name + "\x00" + t.Name
			name := ToolName(c.server.Name, t.Name, func(candidate string) bool {
				o, ok := m.owners[candidate]
				return ok && o != owner || current[candidate]
			})
			m.owners[name] = owner
			current[name] = true
			mt, ok := m.tools[name]
			if !ok || mt.tool != t {
				mt = newTool(c, t, name)
				m.tools[name] = mt
			}
			if allow == nil || allow(name) {
				out = append(out, mt)
			}
		}
	}
	return out
}

// Status is a server's state, as `gobble mcp list` prints it. Exposure is
// the one in force: direct, until 0005-PLAN F8 applies the configured one.
type Status struct {
	Name      string   `json:"name"`
	Enabled   bool     `json:"enabled"`
	Exposure  string   `json:"exposure"`
	Transport string   `json:"transport"`
	State     State    `json:"state"`
	Tools     []string `json:"tools"`
	Error     string   `json:"error,omitzero"`
	// Exposed reports that the entry sets exposure or toolExposure, which
	// are not applied yet.
	Exposed bool `json:"-"`
}

// Statuses are the servers' states, in server order. A tool is listed by
// the name its server gives it.
func (m *Manager) Statuses() []Status {
	out := make([]Status, 0, len(m.conns))
	for _, c := range m.conns {
		state, msg, tools := c.status()
		st := Status{
			Name: c.server.Name, Enabled: c.server.Enabled, Exposure: "direct", Transport: c.server.Transport(),
			State: state, Tools: []string{}, Error: msg,
			Exposed: c.server.Exposure != "" || len(c.server.ToolExposure) > 0,
		}
		for _, t := range tools {
			st.Tools = append(st.Tools, t.Name)
		}
		out = append(out, st)
	}
	return out
}

// FirstLine is the first line of an error text, as Pi's start-up report
// shows it.
func FirstLine(s string) string {
	first, _, _ := strings.Cut(s, "\n")
	return first
}

// Close stops connections in progress and closes the open ones.
func (m *Manager) Close() error {
	errs := make([]error, len(m.conns))
	var wg sync.WaitGroup
	for i, c := range m.conns {
		wg.Go(func() { errs[i] = c.close() })
	}
	wg.Wait()
	return errors.Join(errs...)
}
