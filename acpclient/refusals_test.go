package acpclient

import (
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// The client's handler answers each method it does not offer "method not
// found", through the SDK's ClientRefusals, and ignores an elicitation's
// completion (0014-MADR D3; 0035-MADR D8).
func TestClientRefusals(t *testing.T) {
	var h acp.Client = &handler{}
	ctx := t.Context()
	for method, call := range map[string]func() error{
		acp.ClientMethodFsReadTextFile:      func() error { _, err := h.ReadTextFile(ctx, acp.ReadTextFileRequest{}); return err },
		acp.ClientMethodFsWriteTextFile:     func() error { _, err := h.WriteTextFile(ctx, acp.WriteTextFileRequest{}); return err },
		acp.ClientMethodTerminalCreate:      func() error { _, err := h.CreateTerminal(ctx, acp.CreateTerminalRequest{}); return err },
		acp.ClientMethodTerminalKill:        func() error { _, err := h.KillTerminal(ctx, acp.KillTerminalRequest{}); return err },
		acp.ClientMethodTerminalOutput:      func() error { _, err := h.TerminalOutput(ctx, acp.TerminalOutputRequest{}); return err },
		acp.ClientMethodTerminalRelease:     func() error { _, err := h.ReleaseTerminal(ctx, acp.ReleaseTerminalRequest{}); return err },
		acp.ClientMethodTerminalWaitForExit: func() error { _, err := h.WaitForTerminalExit(ctx, acp.WaitForTerminalExitRequest{}); return err },
		acp.ClientMethodElicitationCreate:   func() error { _, err := h.CreateElicitation(ctx, acp.CreateElicitationRequest{}); return err },
	} {
		if err := call(); !acp.IsMethodNotFound(err) {
			t.Errorf("%s: %v, want method not found", method, err)
		}
	}
	if err := h.CompleteElicitation(ctx, acp.CompleteElicitationNotification{}); err != nil {
		t.Errorf("CompleteElicitation: %v, want nil", err)
	}
}
