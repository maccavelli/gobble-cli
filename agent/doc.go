// Package agent is the turn loop.
// Stability: stable
//
// An Agent sends the history and a prompt to its llm.Provider, runs the tool
// calls the model asks for, and calls the model again until it ends its
// reply. Calls to tools that are not read-only, and calls naming paths
// outside the workspace, are asked of a permission.Policy first. Run yields
// the turn as events and ends with End, whose messages the caller appends to
// its history. Tool calls run one at a time. A Queue gives the turn steering
// messages after each model reply and follow-ups when it would end. Retry,
// Compact and the loop's limits (identical calls, refused arguments) are
// 0005-PLAN F5's.
package agent
