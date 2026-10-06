// Package acpclient connects gobble's own clients to an ACP agent.
// Stability: beta
//
// Start runs an agent in this process over a pair of pipes and returns a
// Conn; Conn.NewSession and Session.Prompt drive it. Updates arrive as
// gobble's Update type, so the CLI and the TUI import no SDK type
// (0004-MADR rule 2). The connection drops the newest notification when
// its queue is full, and reports each drop (0004-MADR, amendment of
// 2026-10-01).
package acpclient
