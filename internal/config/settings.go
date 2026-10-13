package config

import (
	"fmt"
	"strconv"
	"time"
)

// Settings are the effective settings. The yaml and json names are the
// keys' paths; every field is optional in a file.
type Settings struct {
	DefaultProvider      string     `yaml:"defaultProvider" json:"defaultProvider,omitempty"`
	DefaultThinkingLevel string     `yaml:"defaultThinkingLevel" json:"defaultThinkingLevel,omitempty"`
	SteeringMode         string     `yaml:"steeringMode" json:"steeringMode,omitempty"`
	FollowUpMode         string     `yaml:"followUpMode" json:"followUpMode,omitempty"`
	SessionDir           string     `yaml:"sessionDir" json:"sessionDir,omitempty"`
	Compaction           Compaction `yaml:"compaction" json:"compaction,omitzero"`
	ShellPath            string     `yaml:"shellPath" json:"shellPath,omitempty"`
	ShellCommandPrefix   string     `yaml:"shellCommandPrefix" json:"shellCommandPrefix,omitempty"`
	Tools                Tools      `yaml:"tools" json:"tools,omitzero"`
	MCP                  MCP        `yaml:"mcp" json:"mcp,omitzero"`
	ExternalEditor       string     `yaml:"externalEditor" json:"externalEditor,omitempty"`
}

// Compaction is the compaction settings.
type Compaction struct {
	KeepRecentTokens int64 `yaml:"keepRecentTokens" json:"keepRecentTokens,omitempty"`
}

// Tools is the tool settings.
type Tools struct {
	Bash Bash `yaml:"bash" json:"bash,omitzero"`
}

// Bash is the bash tool's settings.
type Bash struct {
	UseClientTerminal bool  `yaml:"useClientTerminal" json:"useClientTerminal,omitempty"`
	MaxTimeoutMs      int64 `yaml:"maxTimeoutMs" json:"maxTimeoutMs,omitempty"`
}

// MCP is the MCP servers' timings, in milliseconds.
type MCP struct {
	StartupWaitMs int64 `yaml:"startupWaitMs" json:"startupWaitMs,omitempty"`
	StopGraceMs   int64 `yaml:"stopGraceMs" json:"stopGraceMs,omitempty"`
	StopTimeoutMs int64 `yaml:"stopTimeoutMs" json:"stopTimeoutMs,omitempty"`
}

// Defaults are the settings with no file: today's code defaults (0018-MADR
// D14), each the default its key's template line shows.
func Defaults() Settings {
	var s Settings
	for _, k := range Keys {
		if err := s.set(k.Path, defaultValue(k)); err != nil {
			panic(err) // a key table and Settings out of step is a programming error
		}
	}
	return s
}

// defaultValue is a key's Default as its Go value.
func defaultValue(k Key) any {
	switch k.Kind {
	case KindBool:
		return k.Default == "true"
	case KindInt:
		n, err := strconv.ParseInt(k.Default, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("config: key %s: default %q: %v", k.Path, k.Default, err))
		}
		return n
	default:
		return k.Default
	}
}

// field is the field at path: a *string, *bool or *int64, or nil.
func (s *Settings) field(path string) any {
	switch path {
	case "defaultThinkingLevel":
		return &s.DefaultThinkingLevel
	case "steeringMode":
		return &s.SteeringMode
	case "followUpMode":
		return &s.FollowUpMode
	case pathSessionDir:
		return &s.SessionDir
	case "compaction.keepRecentTokens":
		return &s.Compaction.KeepRecentTokens
	case "shellPath":
		return &s.ShellPath
	case "shellCommandPrefix":
		return &s.ShellCommandPrefix
	case "tools.bash.useClientTerminal":
		return &s.Tools.Bash.UseClientTerminal
	case "tools.bash.maxTimeoutMs":
		return &s.Tools.Bash.MaxTimeoutMs
	case "mcp.startupWaitMs":
		return &s.MCP.StartupWaitMs
	case "mcp.stopGraceMs":
		return &s.MCP.StopGraceMs
	case "mcp.stopTimeoutMs":
		return &s.MCP.StopTimeoutMs
	case "defaultProvider":
		return &s.DefaultProvider
	case "externalEditor":
		return &s.ExternalEditor
	}
	return nil
}

// set stores v, of the key's Go type, at path.
func (s *Settings) set(path string, v any) error {
	ok := false
	switch p := s.field(path).(type) {
	case *string:
		*p, ok = v.(string)
	case *bool:
		*p, ok = v.(bool)
	case *int64:
		*p, ok = v.(int64)
	case nil:
		return fmt.Errorf("config: no setting %s", path)
	}
	if !ok {
		return fmt.Errorf("config: setting %s cannot hold %T", path, v)
	}
	return nil
}

// Value is the value at path, of the key's Go type: string, bool or int64.
func (s Settings) Value(path string) (any, bool) {
	switch p := s.field(path).(type) {
	case *string:
		return *p, true
	case *bool:
		return *p, true
	case *int64:
		return *p, true
	}
	return nil, false
}

// MaxTimeout is tools.bash.maxTimeoutMs as a duration.
func (s Settings) MaxTimeout() time.Duration { return ms(s.Tools.Bash.MaxTimeoutMs) }

// MCPStartupWait is mcp.startupWaitMs as a duration.
func (s Settings) MCPStartupWait() time.Duration { return ms(s.MCP.StartupWaitMs) }

// MCPStopGrace is mcp.stopGraceMs as a duration.
func (s Settings) MCPStopGrace() time.Duration { return ms(s.MCP.StopGraceMs) }

// MCPStopTimeout is mcp.stopTimeoutMs as a duration.
func (s Settings) MCPStopTimeout() time.Duration { return ms(s.MCP.StopTimeoutMs) }

func ms(n int64) time.Duration { return time.Duration(n) * time.Millisecond }
