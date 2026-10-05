package acptest

import (
	"context"
	"slices"
	"sync"

	acp "github.com/coder/acp-go-sdk"
)

// Client is the default test client. It records session/update
// notifications, answers permission requests with the cancelled outcome, and
// answers file-system and terminal requests with "method not found", since a
// test client has neither.
type Client struct {
	mu      sync.Mutex
	updates []acp.SessionNotification
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

// RequestPermission answers with the cancelled outcome.
func (*Client) RequestPermission(context.Context, acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
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
