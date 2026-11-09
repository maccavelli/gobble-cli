package cli

import (
	"context"
	_ "embed" // the release spec
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-selfupdate-lib/selfupdate"
	sucli "github.com/maccavelli/go-selfupdate-lib/selfupdate/cli"
	"github.com/maccavelli/go-selfupdate-lib/selfupdate/releasespec"
)

// releaseSpecJSON is the release spec go-selfupdate-lib's build workflow
// reads. The updater reads the same file, so the two cannot name different
// assets (0048-MADR D2).
//
//go:embed selfupdate-release.json
var releaseSpecJSON []byte

// releaseSpec parses releaseSpecJSON once. An error fails only the update
// command.
var releaseSpec = sync.OnceValues(func() (releasespec.Spec, error) {
	return releasespec.Parse(releaseSpecJSON)
})

// The release gobble updates from.
const (
	updateProduct = "gobble"
	updateOwner   = "maccavelli"
	updateRepo    = "gobble-cli"
	// probeTimeout bounds one run of a staged or installed binary.
	probeTimeout = 10 * time.Second
)

// agentRefusal is the one line update prints when it runs inside gobble's
// own tool and was asked for more than --check (0048-MADR D5).
const agentRefusal = "gobble update: refusing to install from inside gobble; run it in your own terminal"

// UpdateCmd checks for, verifies and installs gobble's GitHub releases
// through go-selfupdate-lib (0048-MADR D1). Its flags are its own, so the
// root's --version and the chat's --yes never reach the request.
type UpdateCmd struct {
	Check   bool   `name:"check" help:"Report whether an update is available; install nothing."`
	Yes     bool   `name:"yes" short:"y" help:"Install without asking."`
	Force   bool   `name:"force" help:"Replace a local build, or reinstall the same version."`
	DryRun  bool   `name:"dry-run" help:"Download and verify the release; install nothing."`
	JSON    bool   `name:"json" help:"Write JSON Lines to stdout; everything else to stderr."`
	Release string `name:"release" complete:"none" placeholder:"vX.Y.Z" help:"Install exactly this release."`
}

// Help is the command's detailed help: the exit status, where gobble can
// update itself, and the agent guard.
func (*UpdateCmd) Help() string {
	return `Exit status: 0 up to date, declined or installed; 10 an update is
available (with --check); 1 any error; 2 contradictory flags.

gobble replaces only a regular file under your home directory, such as
~/.local/bin/gobble. GH_TOKEN or GITHUB_TOKEN, when set, authenticates the
requests to GitHub. Inside gobble's own tools (AI_AGENT=gobble), only --check
runs.`
}

// updateDeps are the parts of the updater a test replaces. Production
// leaves them zero: GitHub's API, DefaultHTTPClient, os.Executable, and
// probes that run the binary.
type updateDeps struct {
	apiBase  *url.URL
	client   *http.Client
	target   selfupdate.TargetPolicy
	probeRun func(ctx context.Context, path string, args []string, stdout io.Writer) error
}

// updaterDeps is updateDeps for this process; tests set it.
var updaterDeps updateDeps

// probeWant is the text `gobble version --json` prints for tag: its
// "version" field, which has no "v" (0048-MADR D2, Q5).
func probeWant(tag string) string {
	return `"version":"` + strings.TrimPrefix(tag, "v") + `"`
}

// newUpdater composes the updater from the embedded release spec
// (0048-MADR D2). It returns the installer too, for CleanupPending.
func newUpdater(d updateDeps, userAgent string, stdin io.Reader, stderr io.Writer, interactive bool) (*selfupdate.Updater, *selfupdate.StandaloneInstaller, error) {
	spec, err := releaseSpec()
	if err != nil {
		return nil, nil, err
	}
	if _, err := spec.Product(updateProduct); err != nil {
		return nil, nil, err
	}
	assets, err := spec.AssetSelector()
	if err != nil {
		return nil, nil, err
	}
	client := d.client
	if client == nil {
		client = selfupdate.DefaultHTTPClient()
	}
	src, err := selfupdate.NewGitHubSource(selfupdate.GitHubOptions{
		Repository: selfupdate.Repository{Owner: updateOwner, Name: updateRepo},
		Client:     client,
		APIBaseURL: d.apiBase,
		UserAgent:  userAgent,
		Limits:     selfupdate.DefaultLimits(),
	})
	if err != nil {
		return nil, nil, err
	}
	image, err := selfupdate.NewImageVerifier(selfupdate.Platform{})
	if err != nil {
		return nil, nil, err
	}
	probe, err := selfupdate.NewVersionProberWith(selfupdate.VersionProbeOptions{
		Args: []string{"version", "--json"}, Want: probeWant, Timeout: probeTimeout, Run: d.probeRun,
	})
	if err != nil {
		return nil, nil, err
	}
	inst, err := selfupdate.NewStandaloneInstaller(selfupdate.InstallOptions{TargetPolicy: d.target, PostInstall: probe})
	if err != nil {
		return nil, nil, err
	}
	u, err := selfupdate.New(selfupdate.Config{
		Source:    src,
		Versions:  selfupdate.NewStrictVersionPolicy(),
		Assets:    assets,
		Verifiers: []selfupdate.Verifier{image},
		Installer: inst,
		Reporter:  selfupdate.NewTextReporter(stderr),
		Confirmer: selfupdate.NewPromptConfirmer(stdin, stderr, interactive),
		Limits:    selfupdate.DefaultLimits(),
		Probes:    []selfupdate.Prober{probe},
	})
	if err != nil {
		return nil, nil, err
	}
	return u, inst, nil
}

// Run checks for, or installs, a release through selfupdate/cli's Request,
// Run and Exit (0048-MADR D3). Main keeps the process's exit and its
// signals: Run installs no signal handler of its own.
func (c *UpdateCmd) Run(e *runEnv) error {
	flags := sucli.Flags{Check: c.Check, Yes: c.Yes, Force: c.Force, DryRun: c.DryRun, JSON: c.JSON, Version: c.Release}
	id := identity()
	req, err := flags.Request(updateProduct, id.Lib)
	if err != nil {
		return usageErrorf("update: %v", err)
	}
	if os.Getenv("AI_AGENT") == appName && !req.CheckOnly {
		return failf("%s", agentRefusal)
	}
	var stdin io.Reader
	if e.stdin != nil {
		stdin = e.stdin
	}
	interactive := e.out.caps.In
	u, inst, err := newUpdater(updaterDeps, id.UserAgent(), stdin, e.out.err, interactive)
	if err != nil {
		return failf("update: %v", err)
	}
	var targetErr error
	if !req.CheckOnly {
		targetErr = prepareTarget(e.ctx, inst, e.out)
	}
	res, err := sucli.Run(e.ctx, u, req, sucli.Options{
		Stdout: e.out.out, Stderr: e.out.err, Stdin: stdin, Interactive: interactive,
		JSON: c.JSON, Signals: []os.Signal{},
	})
	code := sucli.Exit(e.out.err, res, err)
	if err != nil && targetErr != nil && !req.DryRun {
		e.out.diag("", "%s", targetHint(updaterDeps.target))
	}
	if code == ExitOK {
		return nil
	}
	return &exitError{code: code}
}

// prepareTarget resolves the binary an install would replace, and finishes
// what an earlier update left there (0048-MADR D4). It returns the
// resolution's error, which the run then reports in the library's words. A
// cleanup another update holds the lock for is benign; any other cleanup
// error is a warning.
func prepareTarget(ctx context.Context, inst *selfupdate.StandaloneInstaller, out *Output) error {
	if _, err := inst.ResolveTarget(ctx); err != nil {
		return err
	}
	if err := inst.CleanupPending(ctx); err != nil && !errors.Is(err, selfupdate.ErrConcurrentUpdate) {
		out.Warnf("update: finish an earlier update: %v", err)
	}
	return nil
}

// targetHint is the line after the library's error when it refused the
// binary as a target (0048-MADR D6).
func targetHint(policy selfupdate.TargetPolicy) string {
	p := policy.ExecutablePath
	if p == "" {
		exe, err := os.Executable()
		if err != nil {
			exe = "this gobble"
		}
		p = exe
	}
	return "hint: gobble replaces only a regular, non-symlinked file under your home directory, and " + p +
		" is not one. Install gobble there, such as in ~/.local/bin, and run that copy's gobble update."
}
