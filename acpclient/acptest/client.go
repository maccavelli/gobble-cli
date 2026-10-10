package acptest

import (
	"context"
	"slices"
	"sync"

	acp "github.com/coder/acp-go-sdk"
)

// Client is the default test client. It records session/update
// notifications and permission requests, answers a permission request with
// the cancelled outcome (or its first allow option when Allow is set), and
// answers file-system and terminal requests with "method not found", unless
// Files or Terminals is set.
type Client struct {
	// Allow selects the first allow_once or allow_always option of every
	// permission request.
	Allow bool
	// Files, when not nil, serves fs/read_text_file and fs/write_text_file:
	// a read answers a path's text, a write sets it. FileCalls records
	// each. A test that advertises fs sets it.
	Files map[string]string
	// Terminals, when not nil, serves the terminal methods.
	Terminals *Terminals

	mu          sync.Mutex
	updates     []acp.SessionNotification
	permissions []acp.RequestPermissionRequest
	fileCalls   []string
}

var _ acp.Client = (*Client)(nil)

// Updates returns the session/update notifications received, in order.
func (c *Client) Updates() []acp.SessionNotification {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.updates)
}

// SessionUpdate records the notification.
func (c *Client) SessionUpdate(_ context.Context, n acp.SessionNotification) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updates = append(c.updates, n)
	return nil
}

// Permissions returns the permission requests received, in order.
func (c *Client) Permissions() []acp.RequestPermissionRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.permissions)
}

// RequestPermission records the request and answers with the cancelled
// outcome, or with its first allow option when Allow is set.
func (c *Client) RequestPermission(_ context.Context, r acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	c.mu.Lock()
	c.permissions = append(c.permissions, r)
	c.mu.Unlock()
	if c.Allow {
		for _, o := range r.Options {
			if o.Kind == acp.PermissionOptionKindAllowOnce || o.Kind == acp.PermissionOptionKindAllowAlways {
				return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(o.OptionId)}, nil
			}
		}
	}
	return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}, nil
}

// FileCalls are the file requests served, in order, each "read <path>" or
// "write <path>".
func (c *Client) FileCalls() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.fileCalls)
}

// ReadTextFile answers a path's text from Files, or "method not found"
// without Files.
func (c *Client) ReadTextFile(_ context.Context, r acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Files == nil {
		return acp.ReadTextFileResponse{}, acp.NewMethodNotFound(acp.ClientMethodFsReadTextFile)
	}
	c.fileCalls = append(c.fileCalls, "read "+r.Path)
	text, ok := c.Files[r.Path]
	if !ok {
		return acp.ReadTextFileResponse{}, acp.NewInvalidParams(map[string]any{"path": r.Path, "reason": "no such file"})
	}
	return acp.ReadTextFileResponse{Content: text}, nil
}

// WriteTextFile sets a path's text in Files, or answers "method not found"
// without Files.
func (c *Client) WriteTextFile(_ context.Context, r acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Files == nil {
		return acp.WriteTextFileResponse{}, acp.NewMethodNotFound(acp.ClientMethodFsWriteTextFile)
	}
	c.fileCalls = append(c.fileCalls, "write "+r.Path)
	c.Files[r.Path] = r.Content
	return acp.WriteTextFileResponse{}, nil
}

// Terminals is a scripted client terminal: each command it is asked to run
// exits at once with ExitCode and Output, unless Hang is set, when it runs
// until terminal/kill. Calls records each request.
type Terminals struct {
	Output   string
	ExitCode int
	Hang     bool

	mu       sync.Mutex
	calls    []string
	requests []acp.CreateTerminalRequest
	killed   chan struct{}
}

// Calls are the terminal methods called, in order, by their ACP names
// without the "terminal/" prefix.
func (t *Terminals) Calls() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.calls)
}

// Requests are the terminal/create requests, in order.
func (t *Terminals) Requests() []acp.CreateTerminalRequest {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.requests)
}

func (t *Terminals) record(call string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls = append(t.calls, call)
}

func (t *Terminals) exit() *int {
	if t.Hang {
		return nil
	}
	return &t.ExitCode
}

// CreateTerminal starts a scripted command, or answers "method not found"
// without Terminals.
func (c *Client) CreateTerminal(_ context.Context, r acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	t := c.Terminals
	if t == nil {
		return acp.CreateTerminalResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalCreate)
	}
	t.mu.Lock()
	t.calls = append(t.calls, "create")
	t.requests = append(t.requests, r)
	t.killed = make(chan struct{})
	t.mu.Unlock()
	return acp.CreateTerminalResponse{TerminalId: "term-1"}, nil
}

// KillTerminal stops a hanging command.
func (c *Client) KillTerminal(context.Context, acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	t := c.Terminals
	if t == nil {
		return acp.KillTerminalResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalKill)
	}
	t.mu.Lock()
	t.calls = append(t.calls, "kill")
	if t.killed != nil {
		close(t.killed)
		t.killed = nil
	}
	t.mu.Unlock()
	return acp.KillTerminalResponse{}, nil
}

// TerminalOutput answers Output.
func (c *Client) TerminalOutput(context.Context, acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	t := c.Terminals
	if t == nil {
		return acp.TerminalOutputResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalOutput)
	}
	t.record("output")
	return acp.TerminalOutputResponse{Output: t.Output, ExitStatus: &acp.TerminalExitStatus{ExitCode: t.exit()}}, nil
}

// ReleaseTerminal frees the terminal.
func (c *Client) ReleaseTerminal(context.Context, acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	t := c.Terminals
	if t == nil {
		return acp.ReleaseTerminalResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalRelease)
	}
	t.record("release")
	return acp.ReleaseTerminalResponse{}, nil
}

// WaitForTerminalExit answers ExitCode at once, or, with Hang, when the
// command is killed or ctx is done.
func (c *Client) WaitForTerminalExit(ctx context.Context, _ acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	t := c.Terminals
	if t == nil {
		return acp.WaitForTerminalExitResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalWaitForExit)
	}
	t.mu.Lock()
	t.calls = append(t.calls, "wait_for_exit")
	killed := t.killed
	t.mu.Unlock()
	if t.Hang && killed != nil {
		select {
		case <-killed:
			return acp.WaitForTerminalExitResponse{Signal: new("SIGKILL")}, nil
		case <-ctx.Done():
			return acp.WaitForTerminalExitResponse{}, context.Cause(ctx)
		}
	}
	return acp.WaitForTerminalExitResponse{ExitCode: &t.ExitCode}, nil
}
