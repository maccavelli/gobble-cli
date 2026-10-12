package config

import "strings"

// Kind is a key's value type.
type Kind int

// The kinds. Enum, Path and Command are strings with a rule.
const (
	KindString Kind = iota
	KindBool
	KindInt
	KindEnum
	KindPath    // ~ is the home directory; sessionDir must be absolute
	KindCommand // a command line, split as the editor chain splits one
)

// Scope says which files may set a key.
type Scope int

// The scopes (0018-MADR D6).
const (
	ScopeAny        Scope = iota
	ScopeGlobalOnly       // never from a project's settings.yaml
)

// Read says when a key is resolved (0018-MADR D16). Zero, in
// Options.Read, is both.
type Read int

// The reads.
const (
	ReadSession Read = 1 + iota // when a session starts, with its project
	ReadProcess                 // once, at process start
)

// Key is one setting: its dotted path, kind, default as the template shows
// it, allowed values, least value, scope, read, and the schema's
// description.
type Key struct {
	Path    string
	Kind    Kind
	Default string
	Enum    []string
	Min     int64
	Scope   Scope
	Read    Read
	Doc     string
}

// pathSessionDir is the one path key that must be absolute.
const pathSessionDir = "sessionDir"

// queueModes are steeringMode's and followUpMode's values, the default
// first.
var queueModes = []string{"one-at-a-time", "all"}

// Keys are 0018-MADR D14's thirteen, in the template's order. Every other
// part of the package reads them; a key added here without its template
// block, schema or Settings field fails the package's tests.
var Keys = []Key{
	{Path: "defaultThinkingLevel", Kind: KindEnum, Default: "off", Enum: []string{"off", "minimal", "low", "medium", "high", "xhigh"},
		Read: ReadSession, Doc: "The thinking level a new session starts with."},
	{Path: "steeringMode", Kind: KindEnum, Default: queueModes[0], Enum: queueModes,
		Read: ReadSession, Doc: "How steering messages queued while a turn runs reach the model: one before each model call, or all at once."},
	{Path: "followUpMode", Kind: KindEnum, Default: queueModes[0], Enum: queueModes,
		Read: ReadSession, Doc: "How follow-up messages queued while a turn runs reach the model: one at a time, or all at once."},
	{Path: pathSessionDir, Kind: KindPath, Scope: ScopeGlobalOnly, Read: ReadProcess,
		Doc: "The directory session files are kept in, flat, as --session-dir keeps them. Empty keeps them in gobble's data directory, by working directory."},
	{Path: "compaction.keepRecentTokens", Kind: KindInt, Default: "20000", Min: 1, Read: ReadSession,
		Doc: "The recent conversation, in tokens, that /compact keeps word for word."},
	{Path: "shellPath", Kind: KindPath, Scope: ScopeGlobalOnly, Read: ReadProcess,
		Doc: "The bash the bash tool runs. Empty finds one: Git's bash on Windows, bash on PATH elsewhere."},
	{Path: "shellCommandPrefix", Kind: KindString, Scope: ScopeGlobalOnly, Read: ReadProcess,
		Doc: "A line run before every bash command, such as \"set -euo pipefail\"."},
	{Path: "tools.bash.useClientTerminal", Kind: KindBool, Default: "false", Read: ReadProcess,
		Doc: "Whether bash runs in the ACP client's terminal when the client offers one."},
	{Path: "tools.bash.maxTimeoutMs", Kind: KindInt, Default: "600000", Min: 1, Read: ReadProcess,
		Doc: "The longest timeout, in milliseconds, a bash command may ask for."},
	{Path: "mcp.startupWaitMs", Kind: KindInt, Default: "10000", Min: 0, Read: ReadSession,
		Doc: "How long, in milliseconds, the first prompt waits for MCP servers to connect."},
	{Path: "mcp.stopGraceMs", Kind: KindInt, Default: "500", Min: 0, Read: ReadSession,
		Doc: "How long, in milliseconds, a stdio MCP server has to exit once its input closes."},
	{Path: "mcp.stopTimeoutMs", Kind: KindInt, Default: "2000", Min: 0, Read: ReadSession,
		Doc: "How long, in milliseconds, after the grace before a stdio MCP server is killed."},
	{Path: "externalEditor", Kind: KindCommand, Read: ReadProcess,
		Doc: "The editor /edit and gobble config edit open. GOBBLE_EDITOR wins over it; it wins over VISUAL and EDITOR."},
}

// KeyOf is the key at path.
func KeyOf(path string) (Key, bool) {
	for _, k := range Keys {
		if k.Path == path {
			return k, true
		}
	}
	return Key{}, false
}

// isGroup reports whether path is a mapping that holds keys, such as
// "tools" or "tools.bash".
func isGroup(path string) bool {
	for _, k := range Keys {
		if strings.HasPrefix(k.Path, path+".") {
			return true
		}
	}
	return false
}
