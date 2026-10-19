package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/personality"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// trustState is the run's project trust (0005-PLAN F6-1): --approve's or
// --no-approve's flag, the question while gobble's terminal may ask it,
// and the answers given this run, by project root.
type trustState struct {
	mu      sync.Mutex
	flag    *bool
	ask     func(root string) trustAnswer
	answers map[string]bool
	asked   map[string]bool
}

// trustAnswer is the question's answer: whether to trust, the directory
// the answer is for, whether to save it, and whether there was one.
type trustAnswer struct {
	trusted, save, ok bool
	dir               string
}

// trustStore is trust.json in gobble's config directory (decision 3).
func trustStore() (permission.TrustStore, error) {
	d, _, err := systemDirs()
	if err != nil {
		return permission.TrustStore{}, err
	}
	return permission.TrustStore{Path: filepath.Join(d.Config, permission.TrustFile)}, nil
}

// projectRoot is cwd's project root (0018-MADR D2): its git top-level, or
// cwd; "" for none.
func projectRoot(cwd string) string {
	f, ok := config.ProjectFile(cwd)
	if !ok {
		return ""
	}
	return filepath.Dir(filepath.Dir(f))
}

// needsTrust reports whether cwd's project has a file only a trusted
// project loads (decision 1).
func needsTrust(cwd string) bool {
	if f, ok := config.ProjectFile(cwd); ok {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}
	home, _ := os.UserHomeDir() //nolint:errcheck // with no home, none is excluded
	return len(sysprompt.TrustFiles(cwd, home)) > 0
}

// projectTrust is cwd's project root and whether it is trusted, decided in
// decision 2's order each time it is asked: a decision saved meanwhile, by
// /trust or gobble trust, applies from the next session. A trust.json that
// cannot be read is logged, and trusts nothing. An empty cwd has no project.
func (e *runEnv) projectTrust(cwd string) (string, permission.TrustDecision) {
	root := projectRoot(cwd)
	if root == "" {
		return "", permission.TrustDecision{}
	}
	e.trust.mu.Lock()
	flag := e.trust.flag
	e.trust.mu.Unlock()
	in := permission.TrustInput{Root: root, Flag: flag, Needs: needsTrust(cwd), Default: e.defaultTrust(), Ask: e.trustAnswer}
	if store, err := trustStore(); err == nil {
		in.Saved = store.Lookup
	}
	d, err := permission.DecideTrust(in)
	if err != nil {
		e.log().Warn("trust", slog.String("error", err.Error()))
	}
	return root, d
}

// defaultTrust is defaultProjectTrust, from the global file and --set
// alone: it decides whether the project's file is read (decision 4).
func (e *runEnv) defaultTrust() string {
	d, _, err := systemDirs()
	if err != nil {
		return permission.DefaultAsk
	}
	r, _ := config.Resolve(config.Options{ConfigDir: d.Config, Overrides: e.overrides, Read: config.ReadProcess})
	return r.Settings.DefaultProjectTrust
}

// trustAnswer is the run's answer for root: one given this run, or the
// question's, while gobble's terminal may ask it. A saved answer is saved
// here; one that cannot be is warned of, and holds for the run.
func (e *runEnv) trustAnswer(root string) (bool, bool) {
	e.trust.mu.Lock()
	defer e.trust.mu.Unlock()
	if v, ok := e.trust.answers[root]; ok {
		return v, true
	}
	if e.trust.ask == nil || e.trust.asked[root] {
		return false, false
	}
	if e.trust.asked == nil {
		e.trust.asked, e.trust.answers = map[string]bool{}, map[string]bool{}
	}
	e.trust.asked[root] = true
	a := e.trust.ask(root)
	if !a.ok {
		return false, false
	}
	if a.save {
		store, err := trustStore()
		if err == nil {
			_, err = store.Save(e.ctx, a.dir, a.trusted)
		}
		if err != nil {
			e.out.Warnf("trust %s: %v; the answer holds for this run", a.dir, err)
		}
	}
	e.trust.answers[root] = a.trusted
	return a.trusted, true
}

// The question's answers (decision 5).
const (
	answerTrust       = "trust"
	answerParent      = "trust-parent"
	answerTrustRun    = "trust-run"
	answerUntrust     = "untrust"
	answerUntrustRun  = "untrust-run"
	trustExplanation  = "Trusting %s lets gobble load the project's .gobble settings, SYSTEM.md, skills, prompts and personalities."
	trustQuestionText = "Trust %s"
)

// decideTrustHere asks gobble's terminal about cwd's project, before the
// session starts, when decision 2 reaches the question (decision 5); no
// later step asks.
func (e *runEnv) decideTrustHere(cwd string) {
	e.trust.mu.Lock()
	e.trust.ask = e.askTrust
	e.trust.mu.Unlock()
	e.projectTrust(cwd)
	e.trust.mu.Lock()
	e.trust.ask = nil
	e.trust.mu.Unlock()
}

// askTrust is decision 5's question on stderr, its answer one key read in
// raw mode; c, Ctrl+C and the end of input cancel.
func (e *runEnv) askTrust(root string) trustAnswer {
	parent := filepath.Dir(root)
	opts := []acpclient.PermissionOption{{ID: answerTrust, Name: "Trust", Kind: kindAllowAlways}}
	if resolved, err := filepath.EvalSymlinks(parent); err == nil && !permission.TooBroad(resolved, "") {
		opts = append(opts, acpclient.PermissionOption{ID: answerParent, Name: "Trust the parent folder (" + parent + ")", Kind: kindAllowAlways})
	}
	opts = append(opts,
		acpclient.PermissionOption{ID: answerTrustRun, Name: "Trust for this run", Kind: kindAllowOnce},
		acpclient.PermissionOption{ID: answerUntrust, Name: "Do not trust", Kind: kindRejectAlways},
		acpclient.PermissionOption{ID: answerUntrustRun, Name: "Do not trust for this run", Kind: kindRejectOnce})
	e.out.Notef(trustExplanation, root)
	r := acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: fmt.Sprintf(trustQuestionText, root)}, Options: opts}
	line, keys, labels := permissionLine(r)
	errw := e.out.err
	if _, err := fmt.Fprint(errw, line); err != nil {
		return trustAnswer{}
	}
	restore, err := trustRaw(e)
	if err != nil {
		return trustAnswer{}
	}
	id := chooseKey(keys, e.stdin.Read)
	rerr := restore()
	if _, err := fmt.Fprintf(errw, "%s\n", labelOf(labels, id)); err != nil || rerr != nil {
		e.log().Warn("trust prompt", slog.String("write_error", fmt.Sprint(err)), slog.String("restore_error", fmt.Sprint(rerr)))
	}
	switch id {
	case answerTrust:
		return trustAnswer{trusted: true, save: true, ok: true, dir: root}
	case answerParent:
		return trustAnswer{trusted: true, save: true, ok: true, dir: parent}
	case answerTrustRun:
		return trustAnswer{trusted: true, ok: true}
	case answerUntrust:
		return trustAnswer{save: true, ok: true, dir: root}
	case answerUntrustRun:
		return trustAnswer{ok: true}
	}
	return trustAnswer{}
}

// trustRaw puts the terminal in raw mode for the question; tests replace
// it.
var trustRaw = func(e *runEnv) (func() error, error) { return editor.Raw(e.stdin) }

// noteUntrusted is decision 9's line on stderr, for the line view and print
// mode: once, when cwd's project has files that are held back because it
// is not trusted, and was not just asked about.
func (e *runEnv) noteUntrusted(cwd string) {
	root, d := e.projectTrust(cwd)
	e.trust.mu.Lock()
	asked := e.trust.asked[root]
	e.trust.mu.Unlock()
	if root == "" || d.Trusted || asked || !needsTrust(cwd) {
		return
	}
	e.out.Notef("%s", (*personality.Voice)(nil).Text(personality.TrustUntrusted, nil))
}

// trustFlag is --approve's (true) or --no-approve's (false) for the run,
// or nil when neither was given.
func trustFlag(g given, approve bool) *bool {
	if !g["approve"] {
		return nil
	}
	return &approve
}

// projectOverrides are a trusted project's personality messages and
// prompts, under its .gobble/personalities, laid over the user's, each
// checked as the user's are, each problem a warning (decision 8); nil when
// it has none.
func (e *runEnv) projectOverrides(cwd string) *acpserver.ProjectOverrides {
	dir := filepath.Join(cwd, ".gobble", "personalities")
	if !isDir(dir) {
		return nil
	}
	set, errs := e.personalities().WithProject(dir)
	o := builtinOptions(e.processSettings(cwd).Settings)
	prompts, perrs := loadPrompts(filepath.Join(dir, personality.PromptsDir), o)
	for _, err := range append(errs, perrs...) {
		e.log().Warn("personality", slog.String("error", err.Error()))
	}
	descriptions := map[string]string{}
	for name, text := range prompts.descriptions {
		if d, err := builtin.CheckDescription(name, text, o); err == nil {
			descriptions[name] = d
		}
	}
	return &acpserver.ProjectOverrides{Personalities: set, ToolsSection: prompts.sections[promptTools],
		RulesSection: prompts.sections[promptRules], CompactionPrompts: prompts.compaction, Descriptions: descriptions}
}

// trustOptions are /trust's store: trust.json, as gobble trust saves to it.
func trustOptions() acpserver.TrustOptions {
	return acpserver.TrustOptions{Save: func(ctx context.Context, dir string, trusted bool) (string, error) {
		store, err := trustStore()
		if err != nil {
			return "", err
		}
		return store.Save(ctx, dir, trusted)
	}}
}

// TrustCmd is gobble trust (decision 7).
type TrustCmd struct {
	Dir  string `arg:"" optional:"" complete:"dir" placeholder:"DIR" help:"A directory of the project; the working directory by default."`
	List bool   `help:"List the saved decisions."`
}

// UntrustCmd is gobble untrust (decision 7).
type UntrustCmd struct {
	Dir    string `arg:"" optional:"" complete:"dir" placeholder:"DIR" help:"A directory of the project; the working directory by default."`
	Forget bool   `help:"Remove the saved decision, so defaultProjectTrust decides again."`
}

// Run saves "trusted" for the project, or lists the saved decisions.
func (c *TrustCmd) Run(e *runEnv) error {
	if c.List {
		if c.Dir != "" {
			return usageErrorf("--list takes no directory")
		}
		return e.listTrust()
	}
	return e.saveTrust(c.Dir, true, false)
}

// Run saves "not trusted" for the project, or forgets its decision.
func (c *UntrustCmd) Run(e *runEnv) error {
	return e.saveTrust(c.Dir, false, c.Forget)
}

// saveTrust saves trusted for dir's project root, or forgets the decision
// saved for it, then prints the project's decision and its source.
func (e *runEnv) saveTrust(dir string, trusted, forget bool) error {
	cwd, _, err := resolveCwd(dir)
	if err != nil {
		return err
	}
	root := projectRoot(cwd)
	store, err := trustStore()
	if err != nil {
		return failf("%v", err)
	}
	var b strings.Builder
	switch {
	case forget:
		at, removed, err := store.Forget(e.ctx, root)
		if err != nil {
			return failf("untrust %s: %v", root, err)
		}
		if removed {
			fmt.Fprintf(&b, "Forgot the decision saved for %s.\n", at)
		} else {
			fmt.Fprintf(&b, "No decision is saved for %s.\n", at)
		}
	default:
		at, err := store.Save(e.ctx, root, trusted)
		if errors.Is(err, permission.ErrTooBroad) {
			return failf("trust %s: trusting it would trust every project under it; name a project, or set defaultProjectTrust: always to trust every one", at)
		}
		if err != nil {
			return failf("trust %s: %v", root, err)
		}
		fmt.Fprintf(&b, "Saved: %s is %s.\n", at, map[bool]string{true: "trusted", false: "not trusted"}[trusted])
	}
	_, d := e.projectTrust(cwd)
	fmt.Fprintf(&b, "%s: %s\n", cwd, d)
	return e.out.Result([]byte(b.String()))
}

// listTrust prints each saved decision, by path.
func (e *runEnv) listTrust() error {
	store, err := trustStore()
	if err != nil {
		return failf("%v", err)
	}
	m, err := store.Load()
	if err != nil {
		return failf("%v", err)
	}
	if len(m) == 0 {
		return e.out.Result([]byte("No decisions are saved.\n"))
	}
	paths := make([]string, 0, len(m))
	for p := range m {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	var b strings.Builder
	for _, p := range paths {
		fmt.Fprintf(&b, "%s\t%s\n", p, map[bool]string{true: "trusted", false: "not trusted"}[m[p]])
	}
	return e.out.Result([]byte(b.String()))
}
