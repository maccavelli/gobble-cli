package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
)

// TestMain lets this test binary serve as the stdio MCP fixture (0002-PLAN
// Phase 5), then clears every model credential variable before any test
// runs, so no test reaches a real provider, however the host is set up
// (0002-PLAN Phase 3).
func TestMain(m *testing.M) {
	mcptest.ServeStdioIfAsked()
	for _, v := range provider.CredentialVars() {
		if err := os.Unsetenv(v); err != nil {
			fmt.Fprintf(os.Stderr, "unset %s: %v\n", v, err) //nolint:forbidigo // the test binary's own stderr
			os.Exit(2)                                       //nolint:forbidigo // TestMain's exit
		}
	}
	os.Exit(m.Run()) //nolint:forbidigo // TestMain's exit
}
