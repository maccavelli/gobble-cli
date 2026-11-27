package cli

import (
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// A resolver built over a plan session still says openai is usable after
// auth.json's entry is replaced by another application's ChatGPT login (as
// another process's sign-in can do while a run is live); the provider build,
// which reads the stored session when it builds, refuses the foreign session
// in the resolver's words, for the run's provider and a switched session's
// (0052-PLAN P4, the owner's ruling of 2026-11-16, relayed by the coordinator).
func TestStaleResolverCannotBuildAForeignSession(t *testing.T) {
	c := newCfgEnv(t)
	saveLogin(t, c, "openai", planLogin())
	e := envHere(t)
	if !e.hasCredential("openai") {
		t.Fatal("the resolver does not accept the plan session: the test holds nothing")
	}

	saveLogin(t, c, "openai", foreignLogin())
	if !e.hasCredential("openai") {
		t.Fatal("the resolver was rebuilt: it is not the stale one the test needs")
	}
	const want = "provider openai has an OAuth login in auth.json, which gobble does not use (0005-PLAN F4-3b)"
	if _, err := modelProvider(e)(nil, "session-1", llm.Retry{}); err == nil || err.Error() != want {
		t.Errorf("the run's provider over a foreign session gave %v, want the refusal %q", err, want)
	}
	if _, err := providerFor(e)(nil, "openai", "session-2", llm.Retry{}); err == nil || err.Error() != want {
		t.Errorf("a session's provider over a foreign session gave %v, want the refusal %q", err, want)
	}
}
