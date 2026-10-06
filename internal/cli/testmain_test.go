package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// TestMain clears every model credential variable before any test runs, so
// no test reaches a real provider, however the host is set up (0002-PLAN
// Phase 3).
func TestMain(m *testing.M) {
	for _, v := range provider.CredentialVars() {
		if err := os.Unsetenv(v); err != nil {
			fmt.Fprintf(os.Stderr, "unset %s: %v\n", v, err) //nolint:forbidigo // the test binary's own stderr
			os.Exit(2)                                       //nolint:forbidigo // TestMain's exit
		}
	}
	os.Exit(m.Run()) //nolint:forbidigo // TestMain's exit
}
