package cli

import (
	"slices"
	"strings"
)

// SharedFlags is the default command's flag set (0008-MADR D12). Every
// value-taking flag carries an enum or a complete:"<source>" tag for
// completion (0008-MADR D17). Long forms only, apart from -p, -c, -n, -t.
type SharedFlags struct {
	// Prompt and output.
	Print        bool   `short:"p" help:"Print the response and exit, without a session."`
	OutputFormat string `name:"output-format" enum:"text,json,stream-json" default:"text" help:"Print-mode output: ${enum}."`
	File         string `complete:"path" placeholder:"PATH" help:"Read the prompt from a file. Not with piped input or @path."`
	ShowThinking bool   `name:"show-thinking" help:"Show the model's thinking."`
	Stats        bool   `help:"Print token and cost statistics after the turn."`
	Verbose      bool   `help:"Show full tool output."`

	// Sessions.
	Continue   bool   `short:"c" help:"Continue the most recent session in this directory."`
	Session    string `complete:"session" placeholder:"PATH|ID|PREFIX" help:"Resume a session."`
	SessionID  string `name:"session-id" complete:"none" placeholder:"ID" help:"Start a session with this id."`
	Fork       string `complete:"session" placeholder:"PATH|ID" help:"Start a new session from a copy of another."`
	NoSession  bool   `name:"no-session" help:"Do not save the session."`
	SessionDir string `name:"session-dir" complete:"dir" placeholder:"DIR" help:"Directory for session files."`
	Name       string `short:"n" complete:"none" help:"Name the session."`

	// Models.
	Provider string `complete:"provider" help:"Provider; requires --model."`
	Model    string `complete:"model" placeholder:"PROVIDER/ID[:THINKING]" help:"Model to use."`
	Thinking string `enum:",off,minimal,low,medium,high,xhigh" default:"" help:"Thinking level: off, minimal, low, medium, high or xhigh."`
	APIKey   string `name:"api-key" complete:"none" placeholder:"KEY" help:"API key for this run."`

	// Tools.
	Tools        []string `short:"t" sep:"," complete:"tools" placeholder:"NAME,…" help:"Enable only these tools."`
	ExcludeTools []string `name:"exclude-tools" sep:"," complete:"tools" placeholder:"NAME,…" help:"Disable these tools."`
	NoTools      bool     `name:"no-tools" help:"Disable all tools."`

	// Prompts and context.
	SystemPrompt       string `name:"system-prompt" complete:"path" placeholder:"TEXT|PATH" help:"Replace the system prompt."`
	AppendSystemPrompt string `name:"append-system-prompt" complete:"path" placeholder:"TEXT|PATH" help:"Append to the system prompt."`
	NoContextFiles     bool   `name:"no-context-files" help:"Do not load AGENTS.md and other context files."`

	// Trust, approval and environment.
	Approve bool   `negatable:"" help:"Approve tool calls without asking (--no-approve: always ask)."`
	Yes     bool   `help:"Answer yes to confirmations."`
	Offline bool   `help:"Make no network calls except to the model."`
	Cwd     string `complete:"dir" placeholder:"DIR" help:"Run as if started in DIR."`
}

// The plans that deliver the default command's remaining flags.
const (
	plan0005F3 = "0005-PLAN F3"
	plan0005F4 = "0005-PLAN F4"
	plan0005F6 = "0005-PLAN F6"
)

// laterFlags maps each flag whose behaviour a later plan delivers to that
// plan. Such a flag is parsed and rejected with exit 2, never ignored
// (0008-MADR D12).
var laterFlags = map[string]string{
	"system-prompt": plan0005F3, "append-system-prompt": plan0005F3, "no-context-files": plan0005F3,
	"provider": plan0005F4, "model": plan0005F4, "thinking": plan0005F4, "api-key": plan0005F4, "offline": plan0005F4,
	"approve": plan0005F6, "yes": plan0005F6,
}

// given is the set of flags named on the command line, by long name.
type given map[string]bool

// validate applies the D12 conflict checks, then rejects any given flag a
// later plan delivers. Conflicts come first, so a contradiction is reported
// as one. stdinPiped and atPaths describe the other inputs.
func (f *SharedFlags) validate(g given, stdinPiped, atPaths bool) error {
	switch {
	case g["provider"] && !g["model"]:
		return usageErrorf("--provider requires --model")
	case g["fork"] && (g["session"] || g["continue"] || g["no-session"]):
		return usageErrorf("--fork cannot be combined with %s", firstGiven(g, "session", "continue", "no-session"))
	case g["session-id"] && (g["session"] || g["continue"]):
		return usageErrorf("--session-id cannot be combined with %s", firstGiven(g, "session", "continue"))
	case g["file"] && stdinPiped:
		return usageErrorf("--file cannot be combined with piped input")
	case g["file"] && atPaths:
		return usageErrorf("--file cannot be combined with an @path argument")
	case g["name"] && (g["continue"] || g["session"]):
		return usageErrorf("--name with --continue or --session is not yet available (0002-PLAN Phase 7)")
	case g["no-tools"] && (g["tools"] || g["exclude-tools"]):
		return usageErrorf("--no-tools cannot be combined with %s", firstGiven(g, "tools", "exclude-tools"))
	}
	names := make([]string, 0, len(g))
	for name := range g {
		if _, later := laterFlags[name]; later {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	slices.Sort(names)
	return usageErrorf("--%s is not yet available (%s)", names[0], laterFlags[names[0]])
}

func firstGiven(g given, names ...string) string {
	for _, n := range names {
		if g[n] {
			return "--" + n
		}
	}
	return ""
}

// hasAtPath reports whether any word names a file with @.
func hasAtPath(words []string) bool {
	return slices.ContainsFunc(words, isAtPath)
}

func isAtPath(w string) bool { return len(w) > 1 && strings.HasPrefix(w, "@") }
