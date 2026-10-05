package cli

// ACPCmd will run the ACP agent on stdio (0002-PLAN Phase 1). Until then it
// writes nothing to stdout, because stdout is the protocol, and exits 2.
// This stub is not the product command.
type ACPCmd struct{}

// Run reports that the command is not implemented.
func (*ACPCmd) Run() error {
	return usageErrorf("gobble acp is not yet implemented (0002-PLAN Phase 1)")
}
