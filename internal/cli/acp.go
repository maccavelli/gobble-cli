package cli

import (
	"io"
	"strings"

	"github.com/maccavelli/gobble-cli/acpserver"
)

// ACPCmd runs gobble's ACP agent on stdio (0002-PLAN Phase 1). stdout is
// the protocol: nothing else is written to it, and the logger has no stderr
// handler (0004-PLAN Phase 2 step 4). It exits 0 at stdin end of file.
type ACPCmd struct{}

// Run serves the agent until stdin closes or a signal arrives.
func (*ACPCmd) Run(e *runEnv) error {
	var in io.Reader = strings.NewReader("") // no stdin: end of file at once
	if r := e.stdinReader(); r != nil {
		in = r
	}
	err := acpserver.Serve(e.ctx, in, e.out.out, acpserver.ServeOptions{Agent: e.agentOptions(), Logger: e.log()})
	if err != nil {
		return failf("acp: %v", err)
	}
	return nil
}
