package acpserver

import (
	"context"
	"os/exec"
	"path/filepath"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// resolveBareCommands is the agent connection's inbound rewrite: a client's
// stdio MCP server whose command is not absolute is resolved on gobble's
// PATH with exec.LookPath, before the SDK validates it, as acp-go-sdk2's
// v0.16.0 notes name. A command that does not resolve to an absolute path
// is left as it is, for the SDK to refuse with its own -32602 (0019-MADR
// D6). It never refuses a message itself.
func resolveBareCommands(_ context.Context, _ string, params any) error {
	var servers []acp.McpServer
	switch p := params.(type) {
	case *acp.NewSessionRequest:
		servers = p.McpServers
	case *acp.LoadSessionRequest:
		servers = p.McpServers
	case *acp.ResumeSessionRequest:
		servers = p.McpServers
	}
	for _, s := range servers {
		if s.Stdio == nil || filepath.IsAbs(s.Stdio.Command) {
			continue
		}
		if path, err := exec.LookPath(s.Stdio.Command); err == nil && filepath.IsAbs(path) {
			s.Stdio.Command = path
		}
	}
	return nil
}
