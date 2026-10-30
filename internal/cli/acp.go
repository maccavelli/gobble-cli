package cli

import (
	"bufio"
	"errors"
	"fmt"
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
	Mode   string `complete:"mode" placeholder:"MODE" help:"Mode the sessions start in: default, accept-edits, plan, or bypass (with permissions.allowBypass)."`
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
	if err := e.chooseMode(c.Mode); err != nil {
		return err
	}
	// The process keys come from the global file and --set only: a
	// project's do not apply under acp (0018-MADR D16). builtin's own
	// output directory stays, as it is today (0018-PLAN P5 deviation 16).
	o := builtinOptions(e.processSettings("").Settings)
	o.Descriptions = e.prompts().descriptions // the user's (0022-PLAN P5)
	e.tools = toolsWith(o)
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
	workspace, err := e.readWorkspace(id)
	if err != nil {
		return err
	}
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	if err := auth.SetKey(e.ctx, path, id, key, workspace); err != nil {
		return failf("%v", err)
	}
	return e.out.Resultf("Saved API key for %s\n", id)
}

// readWorkspace asks a terminal for Claude's workspace after its key, one
// line, empty for none; any other provider, or a stdin that is not a
// terminal, has none (0035-MADR D5, 0035-PLAN PQ1).
func (e *runEnv) readWorkspace(id string) (string, error) {
	if id != "claude" || e.stdin == nil || !e.out.caps.In {
		return "", nil
	}
	if _, err := fmt.Fprint(e.out.err, "Workspace ID (empty for none): "); err != nil {
		return "", fmt.Errorf("write stderr: %w", err)
	}
	line, err := bufio.NewReader(io.LimitReader(e.stdin, 4<<10)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", failf("workspace: %v", err)
	}
	workspace := strings.TrimSpace(line)
	return workspace, checkWorkspace(id, workspace)
}
