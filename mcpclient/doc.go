// Package mcpclient connects gobble to MCP servers over stdio and
// streamable HTTP, through the official go-sdk, and offers their tools as
// tool.Tool values.
// Stability: beta
//
// Load and Validate read Pi's mcp.json schema with Pi's checks and
// messages, and Add and Remove edit the file keeping the rest of it. Start
// connects a session's servers in the background; Manager.Tools names
// their tools mcp__<server>__<tool>. The behaviour follows Pi's MCP
// extension at maccavelli/pi 312184edb (0002-PLAN Phase 5). OAuth,
// exposure modes, list_changed and reconnects arrive with 0005-PLAN F8.
//
// A tool's schema, as the model is sent it, is made on a copy of the
// go-sdk's: "type": "object" and "properties": {} are added when missing,
// and a root-level anyOf or oneOf of an object root has each branch typed
// "object" and given the root's properties, patternProperties and
// additionalProperties when it has none of its own, which keeps the
// arguments the schema accepts and is the form Grok accepts of a root union
// (0060-MADR D2). A root union the rule does not cover, with a branch that is
// not typed object, is sent as the server gave it; Manager.FlaggedTools
// reports it, and Manager.RewrittenTools reports each rewritten one.
// docs/guides/mcp-servers.md says what a user sees and can do.
package mcpclient
