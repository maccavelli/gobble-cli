package cli

// CompletionCmd will print a shell completion script (0008-MADR D17). Until
// 0008-PLAN P5 it parses its argument and exits 2, so it is never silently
// ignored (0008-PLAN P4 deviation 3).
type CompletionCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish,powershell" help:"Shell: ${enum}."`
}

// Run reports that completion is not available yet.
func (*CompletionCmd) Run() error {
	return usageErrorf("gobble completion is not yet available (0008-PLAN P5)")
}
