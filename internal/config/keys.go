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
// it, allowed values, least and greatest value, scope, read, and the
// schema's description. A Max of 0 is no greatest value.
type Key struct {
	Path    string
	Kind    Kind
	Default string
	Enum    []string
	Min     int64
	Max     int64
	Scope   Scope
	Read    Read
	Doc     string
}

// boolTrue is a bool key's true, as the template writes it.
const boolTrue = "true"

// pathSessionDir is the one path key that must be absolute.
const pathSessionDir = "sessionDir"

// pathSmallModel is the small model, provider/id (0005-PLAN F5-5).
const pathSmallModel = "smallModel"

// queueModes are steeringMode's and followUpMode's values, the default
// first.
var queueModes = []string{"one-at-a-time", "all"}

// Keys are 0018-MADR D14's thirteen, 0019-MADR D15's defaultProvider,
// 0021-MADR D3's cacheRetention, 0005-PLAN F4-2c's defaultModel, F5-1's
// retry keys, F5-2's compaction keys, F5-5's smallModel and 0022-PLAN P3's
// personality keys, in the template's order. Every other part
// of the package reads them; a key added here without its template block,
// schema or Settings field fails the package's tests.
var Keys = []Key{
	{Path: "defaultProvider", Kind: KindEnum, Enum: []string{"", "gemini", "openai", "claude", "grok", "opencode-zen", "opencode-go", "huggingface", "kilo", "together"},
		Read: ReadProcess, Doc: "The provider gobble uses, with its first model; its credential variable must be set. Empty takes the first provider whose credential variable is set."},
	{Path: "defaultModel", Kind: KindString, Read: ReadProcess,
		Doc: "The model a new session starts with, of defaultProvider's, its id exactly. Empty takes the provider's first listed model; without defaultProvider it is ignored."},
	{Path: pathSmallModel, Kind: KindString, Scope: ScopeGlobalOnly, Read: ReadProcess,
		Doc: "The model, as provider/id, that names a new session from its first prompt; empty names none. It is sent that prompt."},
	{Path: "cacheRetention", Kind: KindEnum, Default: "short", Enum: []string{"none", "short", "long"},
		Read: ReadSession, Doc: "How long the model service keeps a session's prompt cached: none, short (five minutes on Anthropic's models) or long (an hour)."},
	{Path: "defaultThinkingLevel", Kind: KindEnum, Default: "off", Enum: []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"},
		Read: ReadSession, Doc: "The thinking level a new session starts with."},
	{Path: "steeringMode", Kind: KindEnum, Default: queueModes[0], Enum: queueModes,
		Read: ReadSession, Doc: "How steering messages queued while a turn runs reach the model: one before each model call, or all at once."},
	{Path: "followUpMode", Kind: KindEnum, Default: queueModes[0], Enum: queueModes,
		Read: ReadSession, Doc: "How follow-up messages queued while a turn runs reach the model: one at a time, or all at once."},
	{Path: pathSessionDir, Kind: KindPath, Scope: ScopeGlobalOnly, Read: ReadProcess,
		Doc: "The directory session files are kept in, flat, as --session-dir keeps them. Empty keeps them in gobble's data directory, by working directory."},
	{Path: "personality", Kind: KindString, Default: "dry", Scope: ScopeGlobalOnly, Read: ReadSession,
		Doc: "The personality gobble speaks in: boring (also plain or normal), dry, snarky, bombastic, or one of your own under personalities/ in this directory."},
	{Path: "bombasticAccepted", Kind: KindBool, Default: "false", Scope: ScopeGlobalOnly, Read: ReadSession,
		Doc: "Whether you have agreed to bombastic, which insults you; gobble asks once, and sets this when you agree."},
	{Path: "compaction.enabled", Kind: KindBool, Default: boolTrue, Read: ReadSession,
		Doc: "Whether gobble compacts the conversation on its own: before a model call once it nears the model's window, and once when the model says it is over it."},
	{Path: "compaction.reserveTokens", Kind: KindInt, Default: "16384", Min: 1, Read: ReadSession,
		Doc: "The room, in tokens, kept free in the model's window: the conversation is compacted once it is over the window less this. A summary's reply is capped at 80 percent of it."},
	{Path: "compaction.keepRecentTokens", Kind: KindInt, Default: "20000", Min: 1, Read: ReadSession,
		Doc: "The recent conversation, in tokens, that a compaction keeps word for word."},
	{Path: "compaction.thresholdPercent", Kind: KindInt, Default: "0", Min: 0, Max: 100, Read: ReadSession,
		Doc: "Compacts sooner, once the conversation reaches this percentage of the model's window; 0 is off."},
	{Path: "compaction.thresholdTokens", Kind: KindInt, Default: "0", Min: 0, Read: ReadSession,
		Doc: "Compacts sooner, once the conversation reaches this many tokens, whatever the model's window; 0 is off."},
	{Path: "retry.enabled", Kind: KindBool, Default: boolTrue, Read: ReadSession,
		Doc: "Whether a failed model call is retried: the provider's retries, the wait for the network, and an empty reply's two more runs."},
	{Path: "retry.maxRetries", Kind: KindInt, Default: "3", Min: 0, Read: ReadSession,
		Doc: "How many times the provider retries a failed call."},
	{Path: "retry.baseDelayMs", Kind: KindInt, Default: "2000", Min: 1, Read: ReadSession,
		Doc: "The provider's first wait before a retry, in milliseconds; each later wait doubles."},
	{Path: "retry.maxDelayMs", Kind: KindInt, Default: "60000", Min: 1, Read: ReadSession,
		Doc: "The longest wait before a retry, in milliseconds; a service asking for longer is not retried."},
	{Path: "retry.offlineWaitMs", Kind: KindInt, Default: "300000", Min: 0, Read: ReadSession,
		Doc: "How long, in milliseconds, a turn waits for the network to come back; 0 does not wait."},
	{Path: "shellPath", Kind: KindPath, Scope: ScopeGlobalOnly, Read: ReadProcess,
		Doc: "The bash the bash tool runs. Empty finds one: Git's bash on Windows, bash on PATH elsewhere."},
	{Path: "shellCommandPrefix", Kind: KindString, Scope: ScopeGlobalOnly, Read: ReadProcess,
		Doc: "A line run before every bash command, such as \"set -euo pipefail\"."},
	{Path: "tools.bash.useClientTerminal", Kind: KindBool, Default: "false", Read: ReadProcess,
		Doc: "Whether bash runs in the ACP client's terminal when the client offers one."},
	{Path: "tools.bash.maxTimeoutMs", Kind: KindInt, Default: "600000", Min: 1, Read: ReadProcess,
		Doc: "The longest timeout, in milliseconds, a bash command may ask for."},
	{Path: "mcp.startupWaitMs", Kind: KindInt, Default: "10000", Min: 1, Read: ReadSession,
		Doc: "How long, in milliseconds, the first prompt waits for MCP servers to connect."},
	{Path: "mcp.stopGraceMs", Kind: KindInt, Default: "500", Min: 1, Read: ReadSession,
		Doc: "How long, in milliseconds, a stdio MCP server has to exit once its input closes."},
	{Path: "mcp.stopTimeoutMs", Kind: KindInt, Default: "2000", Min: 1, Read: ReadSession,
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
