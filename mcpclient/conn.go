package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// State is a server's connection state. The values are Pi's.
type State string

// The states of a server.
const (
	StateConnecting State = "connecting"
	StateConnected  State = "connected"
	StateNeedsAuth  State = "needs-auth"
	StateFailed     State = "failed"
	StateDisabled   State = "disabled"
	StateClosed     State = "closed"
)

const (
	stderrKeep = 64 << 10 // bytes of a stdio server's stderr kept
	stderrShow = 2000     // characters of it an error shows, as in Pi
)

// Options configure the connections of Start.
type Options struct {
	// Cwd is the session's directory: a stdio server's relative cwd
	// resolves against it, and it is the one root offered.
	Cwd string
	// ClientName and ClientVersion are MCP clientInfo (0003-MADR).
	ClientName, ClientVersion string
	// UserAgent is the User-Agent of HTTP requests (0003-MADR).
	UserAgent string
	// Getenv resolves $NAME values; nil is os.Getenv.
	Getenv func(string) string
	// Home is the directory ~ names; "" is os.UserHomeDir.
	Home string
}

// conn is one server's connection.
type conn struct {
	server Server
	opts   Options
	done   chan struct{}
	cancel context.CancelFunc
	stderr *tail
	// unauthorized records that the server answered 401.
	unauthorized atomic.Bool

	mu    sync.Mutex
	state State
	err   string
	cs    *mcp.ClientSession
	tools []*mcp.Tool
}

func newConn(s Server, o Options) *conn {
	return &conn{server: s, opts: o, done: make(chan struct{}), state: StateConnecting, stderr: &tail{}}
}

// connect opens the connection and records the outcome.
func (c *conn) connect(ctx context.Context) {
	defer close(c.done)
	cs, tools, err := c.open(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == StateClosed {
		if cs != nil {
			_ = cs.Close() //nolint:errcheck // closed while connecting; nothing reads the error
		}
		return
	}
	if err != nil {
		c.state, c.err = c.failure(err)
		return
	}
	c.state, c.cs, c.tools = StateConnected, cs, tools
}

// failure is the state and error text of a connection that did not open.
func (c *conn) failure(err error) (State, string) {
	if c.unauthorized.Load() && c.server.IsHTTP() && !hasAuthorization(c.server.Headers) {
		return StateNeedsAuth, fmt.Sprintf("MCP server %q requires sign-in, which arrives with 0005-PLAN F8 (gobble mcp login)", c.server.Name)
	}
	msg := err.Error()
	if t := c.stderr.String(); t != "" {
		if r := []rune(t); len(r) > stderrShow {
			t = string(r[len(r)-stderrShow:])
		}
		msg += "\n" + t
	}
	return StateFailed, msg
}

func hasAuthorization(headers []Pair) bool {
	for _, h := range headers {
		if strings.EqualFold(h.Name, "Authorization") {
			return true
		}
	}
	return false
}

// open connects, runs initialize and lists the tools, all within the
// server's timeout, as Pi's request timeout covers initialize.
func (c *conn) open(ctx context.Context) (*mcp.ClientSession, []*mcp.Tool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.server.Timeout)
	defer cancel()
	t, err := c.transport()
	if err != nil {
		return nil, nil, err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: c.opts.ClientName, Version: c.opts.ClientVersion}, nil)
	if c.opts.Cwd != "" {
		// Pi offers the session's directory as the one root (runtime.ts).
		//lint:ignore SA1019 roots are deprecated from MCP 2026-07-28 but still served, and servers on earlier protocols read them (0002-PLAN Phase 5, deviation 3)
		client.AddRoots(&mcp.Root{URI: fileURL(c.opts.Cwd), Name: filepath.Base(c.opts.Cwd)}) //nolint:staticcheck // as the lint:ignore above
	}
	cs, err := client.Connect(ctx, t, nil)
	if err != nil {
		return nil, nil, err
	}
	var tools []*mcp.Tool
	if caps := cs.InitializeResult().Capabilities; caps != nil && caps.Tools != nil {
		for tl, err := range cs.Tools(ctx, nil) {
			if err != nil {
				return nil, nil, errors.Join(err, cs.Close())
			}
			tools = append(tools, tl)
		}
	}
	return cs, tools, nil
}

func (c *conn) getenv() func(string) string {
	if c.opts.Getenv != nil {
		return c.opts.Getenv
	}
	return os.Getenv
}

// resolve resolves a value unless the server's values are literal.
func (c *conn) resolve(value, what string) (string, error) {
	if c.server.Literal {
		return value, nil
	}
	return Resolve(value, what, c.getenv())
}

func (c *conn) transport() (mcp.Transport, error) {
	s := c.server
	if s.IsHTTP() {
		headers := make([]Pair, 0, len(s.Headers))
		for _, h := range s.Headers {
			v, err := c.resolve(h.Value, fmt.Sprintf("MCP server %q header %q", s.Name, h.Name))
			if err != nil {
				return nil, err
			}
			headers = append(headers, Pair{Name: h.Name, Value: v})
		}
		rt := &headerTransport{base: http.DefaultTransport, headers: headers, userAgent: c.opts.UserAgent, unauthorized: &c.unauthorized}
		return &mcp.StreamableClientTransport{Endpoint: s.URL, HTTPClient: &http.Client{Transport: rt}}, nil
	}
	env := os.Environ()
	for _, e := range s.Env {
		v, err := c.resolve(e.Value, fmt.Sprintf("MCP server %q env %q", s.Name, e.Name))
		if err != nil {
			return nil, err
		}
		env = append(env, e.Name+"="+v)
	}
	args := make([]string, len(s.Args))
	for i, a := range s.Args {
		args[i] = c.expandHome(a)
	}
	// No context: the go-sdk's CommandTransport.Close stops the process,
	// closing stdin first and terminating it only if it does not exit.
	cmd := exec.Command(c.expandHome(s.Command), args...) //nolint:gosec,noctx // the user's configured MCP server
	cmd.Dir = c.dir()
	cmd.Env = env
	cmd.Stderr = c.stderr
	return &mcp.CommandTransport{Command: cmd}, nil
}

// dir is a stdio server's working directory: its cwd, resolved against
// the session's directory, which is the default.
func (c *conn) dir() string {
	d := c.expandHome(c.server.Cwd)
	switch {
	case d == "":
		return c.opts.Cwd
	case filepath.IsAbs(d) || c.opts.Cwd == "":
		return d
	}
	return filepath.Join(c.opts.Cwd, d)
}

// expandHome is Pi's: ~ and ~/… (also ~\… on Windows) name the home
// directory.
func (c *conn) expandHome(v string) string {
	home := c.opts.Home
	if home == "" {
		home, _ = os.UserHomeDir() //nolint:errcheck // no home: ~ stays as written
	}
	switch {
	case home == "":
		return v
	case v == "~":
		return home
	case strings.HasPrefix(v, "~/"), runtime.GOOS == "windows" && strings.HasPrefix(v, `~\`):
		return filepath.Join(home, v[2:])
	}
	return v
}

// fileURL is a directory as a file: URL, as Node's pathToFileURL makes one.
func fileURL(dir string) string {
	p := filepath.ToSlash(dir)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

// session is the open session, or nil.
func (c *conn) session() *mcp.ClientSession {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cs
}

func (c *conn) status() (State, string, []*mcp.Tool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state, c.err, c.tools
}

// close stops a connection in progress, or closes the open session.
func (c *conn) close() error {
	c.mu.Lock()
	if c.state == StateDisabled {
		c.mu.Unlock()
		return nil
	}
	c.state = StateClosed
	cs := c.cs
	c.cs = nil
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	<-c.done
	if cs != nil {
		return cs.Close()
	}
	return nil
}

// headerTransport sends the configured headers and gobble's User-Agent on
// every request, and records a 401: without an OAuth handler the go-sdk
// reports one only as text.
type headerTransport struct {
	base         http.RoundTripper
	headers      []Pair
	userAgent    string
	unauthorized *atomic.Bool
}

func (t *headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	if t.userAgent != "" {
		r.Header.Set("User-Agent", t.userAgent)
	}
	for _, h := range t.headers {
		r.Header.Set(h.Name, h.Value)
	}
	resp, err := t.base.RoundTrip(r)
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		t.unauthorized.Store(true)
	}
	return resp, err
}

// tail keeps the last stderrKeep bytes written to it.
type tail struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - stderrKeep; over > 0 {
		t.buf = t.buf[over:]
	}
	return len(p), nil
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.TrimSpace(string(t.buf))
}
