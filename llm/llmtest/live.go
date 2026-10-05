package llmtest

import (
	"os"
	"testing"
)

// LiveCredential returns the credential in the environment variable env,
// such as ANTHROPIC_API_KEY, for a live_<provider> test. When the variable
// is unset the test is skipped: a live test needs a real credential, and
// default CI provides none.
func LiveCredential(t testing.TB, env string) string {
	t.Helper()
	v := os.Getenv(env)
	if v == "" {
		t.Skipf("live test: set %s to run it", env)
	}
	return v
}
