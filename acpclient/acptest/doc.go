// Package acptest pairs an ACP agent with a client over in-memory pipes and
// records every JSON-RPC frame in both directions, for tests of agents and
// clients alike.
// Stability: stable
//
// Pair connects an agent to the default Client, which records session
// updates and refuses what a test client cannot do. Connect takes a client
// of the test's own. Both record the client's side of the wire with
// acp-go-sdk2's acptest.Recorder, so a test compares the records with a
// golden file through acp-go-sdk2's acptest.Golden and checks them against
// the protocol with its acptest.ValidateGolden. HostIndependent is the one
// step gobble adds before the comparison: host paths in slash form, so a
// golden matches on every host (0016-MADR D6; 0004-MADR's package map, note
// of 2026-11-08).
package acptest
