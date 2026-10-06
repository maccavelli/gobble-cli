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
// answers file-system and terminal requests with "method not found", since a
// test client has neither.
type Client struct {
	// Allow selects the first allow_once or allow_always option of every
	// permission request.
	Allow bool

	mu          sync.Mutex
	updates     []acp.SessionNotification
	permissions []acp.RequestPermissionRequest
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

// ReadTextFile is not offered.
func (*Client) ReadTextFile(context.Context, acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, acp.NewMethodNotFound(acp.ClientMethodFsReadTextFile)
}

// WriteTextFile is not offered.
func (*Client) WriteTextFile(context.Context, acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, acp.NewMethodNotFound(acp.ClientMethodFsWriteTextFile)
}

// CreateTerminal is not offered.
func (*Client) CreateTerminal(context.Context, acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalCreate)
}

// KillTerminal is not offered.
func (*Client) KillTerminal(context.Context, acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalKill)
}

// TerminalOutput is not offered.
func (*Client) TerminalOutput(context.Context, acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalOutput)
}

// ReleaseTerminal is not offered.
func (*Client) ReleaseTerminal(context.Context, acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalRelease)
}

// WaitForTerminalExit is not offered.
func (*Client) WaitForTerminalExit(context.Context, acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, acp.NewMethodNotFound(acp.ClientMethodTerminalWaitForExit)
}
