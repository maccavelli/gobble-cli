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
package mcpclient
