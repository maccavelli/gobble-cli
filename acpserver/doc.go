// Package acpserver is gobble's ACP agent: the acp.Agent that `gobble acp`
// serves on stdio, and that the CLI and TUI reach in process.
// Stability: beta
//
// It advertises only what it implements (0002-MADR; 0002-PLAN amendment of
// 2026-10-05). Phase 1 answers initialize, opens and closes sessions, and
// echoes a prompt back as one agent message chunk. The turn loop over agent/
// arrives in 0002-PLAN Phase 3.
package acpserver
