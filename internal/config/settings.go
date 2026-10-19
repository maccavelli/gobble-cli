package config

import (
	"fmt"
	"strconv"
	"time"
)

// Settings are the effective settings. The yaml and json names are the
// keys' paths; every field is optional in a file.
type Settings struct {
	DefaultProvider      string      `yaml:"defaultProvider" json:"defaultProvider,omitempty"`
	DefaultModel         string      `yaml:"defaultModel" json:"defaultModel,omitempty"`
	SmallModel           string      `yaml:"smallModel" json:"smallModel,omitempty"`
	CacheRetention       string      `yaml:"cacheRetention" json:"cacheRetention,omitempty"`
	DefaultThinkingLevel string      `yaml:"defaultThinkingLevel" json:"defaultThinkingLevel,omitempty"`
	SteeringMode         string      `yaml:"steeringMode" json:"steeringMode,omitempty"`
	FollowUpMode         string      `yaml:"followUpMode" json:"followUpMode,omitempty"`
	SessionDir           string      `yaml:"sessionDir" json:"sessionDir,omitempty"`
	DefaultProjectTrust  string      `yaml:"defaultProjectTrust" json:"defaultProjectTrust,omitempty"`
	Permissions          Permissions `yaml:"permissions" json:"permissions,omitzero"`
	Personality          string      `yaml:"personality" json:"personality,omitempty"`
	BombasticAccepted    bool        `yaml:"bombasticAccepted" json:"bombasticAccepted,omitempty"`
	Compaction           Compaction  `yaml:"compaction" json:"compaction,omitzero"`
	Retry                Retry       `yaml:"retry" json:"retry,omitzero"`
	ShellPath            string      `yaml:"shellPath" json:"shellPath,omitempty"`
	ShellCommandPrefix   string      `yaml:"shellCommandPrefix" json:"shellCommandPrefix,omitempty"`
	Tools                Tools       `yaml:"tools" json:"tools,omitzero"`
	MCP                  MCP         `yaml:"mcp" json:"mcp,omitzero"`
	ExternalEditor       string      `yaml:"externalEditor" json:"externalEditor,omitempty"`
}

// Permissions is the permission settings (0005-PLAN F6-2).
type Permissions struct {
	// Rules are every layer's, joined: the global file's, then a trusted
	// project's, then --set's.
	Rules []PermissionRule `yaml:"rules" json:"rules,omitempty"`
	// AllowBypass offers the bypass mode; only the global file sets it
	// (0005-PLAN F6-4 decision 2).
	AllowBypass bool `yaml:"allowBypass" json:"allowBypass,omitempty"`
}

// Compaction is the compaction settings (0005-PLAN F5-2).
type Compaction struct {
	Enabled          bool  `yaml:"enabled" json:"enabled,omitempty"`
	ReserveTokens    int64 `yaml:"reserveTokens" json:"reserveTokens,omitempty"`
	KeepRecentTokens int64 `yaml:"keepRecentTokens" json:"keepRecentTokens,omitempty"`
	ThresholdPercent int64 `yaml:"thresholdPercent" json:"thresholdPercent,omitempty"`
	ThresholdTokens  int64 `yaml:"thresholdTokens" json:"thresholdTokens,omitempty"`
}

// Retry is the retry settings (0005-PLAN F5-1).
type Retry struct {
	Enabled       bool  `yaml:"enabled" json:"enabled,omitempty"`
	MaxRetries    int64 `yaml:"maxRetries" json:"maxRetries,omitempty"`
	BaseDelayMs   int64 `yaml:"baseDelayMs" json:"baseDelayMs,omitempty"`
	MaxDelayMs    int64 `yaml:"maxDelayMs" json:"maxDelayMs,omitempty"`
	OfflineWaitMs int64 `yaml:"offlineWaitMs" json:"offlineWaitMs,omitempty"`
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
		return k.Default == boolTrue
	case KindRules:
		return []PermissionRule(nil)
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
	case "defaultProjectTrust":
		return &s.DefaultProjectTrust
	case pathPermissionRules:
		return &s.Permissions.Rules
	case "permissions.allowBypass":
		return &s.Permissions.AllowBypass
	case "personality":
		return &s.Personality
	case "bombasticAccepted":
		return &s.BombasticAccepted
	case "compaction.enabled":
		return &s.Compaction.Enabled
	case "compaction.reserveTokens":
		return &s.Compaction.ReserveTokens
	case "compaction.keepRecentTokens":
		return &s.Compaction.KeepRecentTokens
	case "compaction.thresholdPercent":
		return &s.Compaction.ThresholdPercent
	case "compaction.thresholdTokens":
		return &s.Compaction.ThresholdTokens
	case "retry.enabled":
		return &s.Retry.Enabled
	case "retry.maxRetries":
		return &s.Retry.MaxRetries
	case "retry.baseDelayMs":
		return &s.Retry.BaseDelayMs
	case "retry.maxDelayMs":
		return &s.Retry.MaxDelayMs
	case "retry.offlineWaitMs":
		return &s.Retry.OfflineWaitMs
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
	case "defaultModel":
		return &s.DefaultModel
	case pathSmallModel:
		return &s.SmallModel
	case "cacheRetention":
		return &s.CacheRetention
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
	case *[]PermissionRule:
		*p, ok = v.([]PermissionRule)
	case nil:
		return fmt.Errorf("config: no setting %s", path)
	}
	if !ok {
		return fmt.Errorf("config: setting %s cannot hold %T", path, v)
	}
	return nil
}

// Value is the value at path, of the key's Go type: string, bool, int64 or
// []PermissionRule.
func (s Settings) Value(path string) (any, bool) {
	switch p := s.field(path).(type) {
	case *string:
		return *p, true
	case *bool:
		return *p, true
	case *int64:
		return *p, true
	case *[]PermissionRule:
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
