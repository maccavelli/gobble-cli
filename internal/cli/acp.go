package cli

import (
	"io"
	"strings"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// ACPCmd runs gobble's ACP agent on stdio (0002-PLAN Phase 1). stdout is
// the protocol: nothing else is written to it, and the logger has no stderr
// handler (0004-PLAN Phase 2 step 4). It exits 0 at stdin end of file.
// With --login it serves nothing: it signs in, or stores a key, in the
// terminal an ACP client's terminal auth method opened (0016-MADR A2).
type ACPCmd struct {
	Login  string `name:"login" complete:"provider" placeholder:"PROVIDER" help:"Sign in to a provider, or store its API key, in this terminal, then exit: what an ACP client's terminal auth method runs."`
	Device bool   `name:"device" help:"With --login, sign in with a device code."`
}

// Run serves the agent until stdin closes or a signal arrives.
func (c *ACPCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	switch {
	case c.Login != "":
		return e.acpLogin(c.Login, c.Device)
	case c.Device:
		return usageErrorf("--device is for --login")
	}
	// The process keys come from the global file and --set only: a
	// project's do not apply under acp (0018-MADR D16). builtin's own
	// output directory stays, as it is today (0018-PLAN P5 deviation 16).
	e.tools = toolsWith(builtinOptions(e.processSettings("").Settings))
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

// acpLogin is gobble acp --login: what gobble auth login does, in the
// terminal the client opened. A provider with a login signs in; any other
// provider's key is asked for without echo and stored (0005-PLAN F4-3c
// decision 5).
func (e *runEnv) acpLogin(name string, device bool) error {
	id, err := keyProvider(name)
	if err != nil {
		return err
	}
	if provider.HasLogin(id) || provider.HasKeyLogin(id) {
		return e.login(id, device)
	}
	if device {
		return usageErrorf("%s has no login; --device is for a login", id)
	}
	key, err := e.readAPIKey(id)
	if err != nil {
		return err
	}
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	if err := auth.Set(e.ctx, path, id, auth.Entry{Type: auth.TypeAPIKey, Key: key}); err != nil {
		return failf("%v", err)
	}
	return e.out.Resultf("Saved API key for %s\n", id)
}
