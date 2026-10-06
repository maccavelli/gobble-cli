package mcpclient

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// maxToolName is the providers' limit on a tool name's length.
const maxToolName = 64

// ToolName is Pi's createMcpToolName (tools.ts): mcp__<server>__<tool>
// with every character outside [A-Za-z0-9_-] made "_". A name over 64
// characters, or one taken reports as used by another tool, is cut and
// given "_" and the first 8 hex digits of sha256("<server>\x00<tool>").
// Pi counts UTF-16 code units, so a character outside the Basic
// Multilingual Plane becomes two underscores here too.
func ToolName(server, tool string, taken func(string) bool) string {
	var b strings.Builder
	for _, r := range "mcp__" + server + "__" + tool {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		case r > 0xFFFF:
			b.WriteString("__")
		default:
			b.WriteByte('_')
		}
	}
	name := b.String()
	if len(name) <= maxToolName && (taken == nil || !taken(name)) {
		return name
	}
	sum := sha256.Sum256([]byte(server + "\x00" + tool))
	hash := hex.EncodeToString(sum[:])[:8]
	return name[:min(len(name), maxToolName-len(hash)-1)] + "_" + hash
}
