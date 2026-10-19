// Package acpserver is gobble's ACP agent: the acp.Agent that `gobble acp`
// serves on stdio, and that the CLI and TUI reach in process.
// Stability: beta
//
// It advertises only what it implements (0002-MADR; 0002-PLAN amendment of
// 2026-10-05). A prompt runs agent/'s turn loop over the session's history,
// held in memory until 0002-PLAN Phase 4 persists it. Text and thoughts are
// coalesced to one chunk per 50 ms or 4 KiB, tool calls carry
// phone-legible titles and summaries, and each call is decided by the
// session's permission rules and gobble's limits: a deny refuses it, an
// allow runs it, and an ask asks the client with
// session/request_permission (0008-MADR D19 items 2 and 3; 0005-PLAN
// F6-2).
package acpserver
