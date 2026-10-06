// Package agent is the turn loop.
// Stability: stable
//
// An Agent sends the history and a prompt to its llm.Provider, runs the tool
// calls the model asks for, and calls the model again until it ends its
// reply. Calls to tools that are not read-only are asked of a
// permission.Policy first. Run yields the turn as events and ends with End,
// whose messages the caller appends to its history. Tool calls run one at a
// time; batches, steering and follow-up queues come later (0004-MADR).
package agent
