// Package llmtest is a scripted llm.Provider and request assertions for
// tests of code above the llm facade: the agent loop, compaction, retry.
// Stability: stable
//
// It exercises the facade, not the SDK: it never imports go-llmprovider-sdk
// or its llmtest package, whose Run and Fake test the SDK contract.
//
// # Live tests
//
// A test that calls a real provider carries the build tag live_<provider>
// (live_anthropic, live_openai, …) and gets its credential with
// LiveCredential, which skips the test when the provider's usual variable is
// unset:
//
//	//go:build live_anthropic
//
//	key := llmtest.LiveCredential(t, "ANTHROPIC_API_KEY")
//
// Run one with go test -tags live_anthropic. Default CI passes no live_ tag,
// so live tests never run there.
package llmtest
