// Package command is the slash-command vocabulary shared by the agent and
// its clients: what counts as a command, and how one is described.
// Stability: beta
//
// A command is a prompt whose text, trimmed, begins with "/" and a name of
// [A-Za-z0-9][A-Za-z0-9_-]*, the names magic-cli-remote can list
// (0008-MADR D19 item 7). Anything else, such as a path or /skill:name, is
// prompt text. The agent's handlers and the CLI's registry are built on
// Parse (0002-PLAN Phase 6).
package command
