package cli

// Root is gobble's Kong grammar. chat is the default command: bare
// `gobble`, `gobble -p "…"` and `gobble fix the test` all run it
// (0008-MADR D1).
type Root struct {
	Chat       ChatCmd       `cmd:"" default:"withargs" help:"Start a session, or run one prompt with -p or piped input."`
	Version    VersionCmd    `cmd:"" help:"Print version information."`
	Completion CompletionCmd `cmd:"" help:"Print a shell completion script."`
	Config     ConfigCmd     `cmd:"" help:"Inspect configuration."`
	ACP        ACPCmd        `cmd:"" name:"acp" help:"Run the ACP agent on stdio."`
	MCP        MCPCmd        `cmd:"" name:"mcp" help:"Add, remove and check MCP servers."`
	LogLevel   string        `name:"log-level" enum:"debug,info,warn,error" default:"info" help:"Log level: ${enum}."`
}
