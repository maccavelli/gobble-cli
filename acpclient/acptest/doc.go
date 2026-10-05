// Package acptest pairs an ACP agent with a client over in-memory pipes and
// records every JSON-RPC frame in both directions, for tests of agents and
// clients alike.
// Stability: stable
//
// Pair connects an agent to the default Client, which records session
// updates and refuses what a test client cannot do. Connect takes a client
// of the test's own. A Recorder's Transcript is canonical JSON, one frame per
// line, and Golden compares it with a file: on a mismatch the actual
// transcript is written to the test's artifact directory, and
// ACPTEST_UPDATE=1 rewrites the golden file instead.
package acptest
