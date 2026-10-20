package cli

// Root is gobble's Kong grammar. chat is the default command: bare
// `gobble`, `gobble -p "…"` and `gobble fix the test` all run it
// (0008-MADR D1).
type Root struct {
	Chat        ChatCmd        `cmd:"" default:"withargs" help:"Start a session, or run one prompt with -p or piped input."`
	Version     VersionCmd     `cmd:"" help:"Print version information."`
	Completion  CompletionCmd  `cmd:"" help:"Print a shell completion script."`
	Config      ConfigCmd      `cmd:"" help:"Inspect configuration."`
	ACP         ACPCmd         `cmd:"" name:"acp" help:"Run the ACP agent on stdio."`
	MCP         MCPCmd         `cmd:"" name:"mcp" help:"Add, remove and check MCP servers."`
	Models      ModelsCmd      `cmd:"" help:"List the models gobble knows, or show one."`
	Providers   ProvidersCmd   `cmd:"" help:"List the providers, each with its credential's source and how to sign in."`
	Auth        AuthCmd        `cmd:"" help:"Store, remove and check model credentials."`
	Personality PersonalityCmd `cmd:"" help:"List and check the personalities, and the files that change gobble's text."`
	Trust       TrustCmd       `cmd:"" help:"Trust a project, so its own .gobble settings, SYSTEM.md, skills, prompts and personalities load."`
	Untrust     UntrustCmd     `cmd:"" help:"Stop trusting a project, or forget the decision saved for it."`
	LogLevel    string         `name:"log-level" enum:"debug,info,warn,error" default:"info" help:"Log level: ${enum}."`
	Set         []string       `name:"set" sep:"none" complete:"none" placeholder:"PATH=VALUE" help:"Override a setting for this run; repeatable."`
}
