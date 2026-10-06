package acpserver

import (
	"os"
	"testing"

	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
)

// TestMain lets this test binary serve as the stdio MCP fixture.
func TestMain(m *testing.M) {
	mcptest.ServeStdioIfAsked()
	os.Exit(m.Run())
}
