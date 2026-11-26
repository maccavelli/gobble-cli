package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/internal/logging"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/personality"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/tool"
)

const description = "gobble is a coding agent for the terminal and for ACP clients."

// Main runs gobble with args and returns the exit code (0008-MADR D10). It
// writes only through stdout and stderr, and reads stdin only when it is
// not a terminal. It never calls os.Exit; cmd/gobble does, with this code.
func Main(ctx context.Context, args []string, stdin *os.File, stdout, stderr io.Writer) int {
	var root Root
	k, err := newParser(&root, stdout, stderr)
	if err != nil {
		(&Output{out: stdout, err: stderr}).Errorf("%v", err)
		return ExitFailure
	}
	// Completion mode runs straight after the model is built, before
	// parsing, terminal detection, signals, logging and any file but the
	// ones it completes (0008-MADR D17).
	if completionMode(os.Getenv) {
		return complete.Run(ctx, k.Model, completionRegistry(), os.Getenv, args, stdout)
	}
	caps := term.Detect(os.Getenv, stdin, fileOf(stdout), fileOf(stderr))
	out := &Output{out: stdout, err: stderr, caps: caps}
	if err := clearCompletionEnv(); err != nil {
		out.Warnf("clear completion environment: %v", err)
	}
	intr := &interrupts{}
	ctx, stop := watchSignals(ctx, term.ShutdownSignals(), signal.Notify, intr)
	defer stop()
	code := run(ctx, k, &root, args, stdin, out, intr)
	if c, ok := signalExit(ctx); ok {
		return c
	}
	return code
}

// newParser builds gobble's Kong parser over root. Kong's Exit panics with
// exitPanic, which parse recovers, so Kong never ends the process itself.
func newParser(root *Root, stdout, stderr io.Writer) (*kong.Kong, error) {
	return kong.New(root,
		kong.Name("gobble"),
		kong.Description(description),
		kong.Writers(stdout, stderr),
		kong.Exit(func(code int) { panic(exitPanic{code: code}) }),
	)
}

func run(ctx context.Context, k *kong.Kong, root *Root, args []string, stdin *os.File, out *Output, intr *interrupts) int {
	kctx, code, done := parse(k, args, out)
	if done {
		return code
	}
	if err := misplacedFlag(k, kctx); err != nil {
		return exitCode(ctx, out, err)
	}
	overrides, err := parseOverrides(root.Set)
	if err != nil {
		return exitCode(ctx, out, err)
	}
	logs := &lazyLog{level: levelOf(root.LogLevel), out: out, stderr: out.err}
	if strings.HasPrefix(kctx.Command(), "acp") {
		logs.stderr = nil // stdio is the protocol (0004-PLAN Phase 2 step 4)
	}
	e := &runEnv{ctx: ctx, out: out, stdin: stdin, logs: logs, intr: intr, now: clock, overrides: overrides}
	e.ownListings()
	if e.migrateAuth(kctx.Command(), loginJSON(kctx)) {
		err = kctx.Run(e)
	} else {
		err = ctx.Err() // a signal ended the run during the migration (0052-MADR D9, row M51)
	}
	e.tellWaitWarn() // a warning a fallback queued, when no prompt's text came after it (0063-MADR D23, row R156)
	code = exitCode(ctx, out, err)
	e.endListings(ctx) // before the log closes: a listing may still log (0046-MADR D1)
	if err := logs.close(); err != nil {
		out.Warnf("close log file: %v", err)
	}
	return code
}

// misplacedFlag refuses a flag of chat's given before another command.
// Kong takes the default command's flags ahead of any command, so
// `gobble --model x acp` parses, and acp would never read the flag; a flag
// is refused, never ignored (0008-MADR D12; 0049-MADR D12).
func misplacedFlag(k *kong.Kong, kctx *kong.Context) error {
	var chat *kong.Node
	for _, n := range k.Model.Children {
		if n.Name == "chat" {
			chat = n
		}
	}
	selected := kctx.Selected()
	if chat == nil || selected == nil || selected == chat {
		return nil
	}
	for _, p := range kctx.Path {
		if p.Flag == nil || p.Resolved || !slices.Contains(chat.Flags, p.Flag) {
			continue
		}
		var words []string
		for _, q := range kctx.Path {
			if q.Command != nil {
				words = append(words, q.Command.Name)
			}
		}
		command := "gobble " + strings.Join(words, " ")
		for _, f := range selected.Flags {
			if f.Name == p.Flag.Name {
				return usageErrorf("--%s is a flag of gobble's chat command; put it after `%s`", p.Flag.Name, command)
			}
		}
		return usageErrorf("--%s is a flag of gobble's chat command, not of `%s`", p.Flag.Name, command)
	}
	return nil
}

// jsonLoginParseFailure writes a command line Kong refused as the failed step
// gobble auth login --json prints for a usage error, and nothing on stderr,
// and returns the exit code, when the parser's own context shows that the
// selected command is auth login and that --json was read as true before
// Kong stopped (0051-MADR D5; 0051-PLAN P5 step 2; the owner's ruling,
// relayed by the coordinator, "Emit the failed step from the parser's own
// context"). A --json after the argument Kong rejects is never read, so such
// a command line gets Kong's text on stderr. The step is written by
// jsonSteps.fail, as every other failed step is. The provider the person
// named is not known to the context at a parse error, so the line's provider
// is empty.
func jsonLoginParseFailure(err error, out *Output) (code int, ok bool) {
	pe, isParse := errors.AsType[*kong.ParseError](err)
	if !isParse || pe.Context == nil {
		return 0, false
	}
	sel := pe.Context.Selected()
	if sel == nil || sel.FullPath() != "gobble auth login" {
		return 0, false
	}
	for _, f := range sel.Flags {
		if f.Name != "json" {
			continue
		}
		if set, isBool := pe.Context.FlagValue(f).(bool); !isBool || !set {
			return 0, false
		}
		end := (&jsonSteps{out: out}).fail(usageErrorf("%v", err))
		if ee, isExit := errors.AsType[*exitError](end); isExit {
			return ee.code, true
		}
		return ExitFailure, true
	}
	return 0, false
}

// parse runs Kong's parser. Kong's Exit panics with exitPanic, which stops
// parsing at once (after --help, say); parse recovers its code. done
// reports that Main should return code without running a command.
func parse(k *kong.Kong, args []string, out *Output) (kctx *kong.Context, code int, done bool) {
	defer func() {
		if r := recover(); r != nil {
			p, ok := r.(exitPanic)
			if !ok {
				panic(r)
			}
			kctx, code, done = nil, p.code, true
		}
	}()
	kctx, err := k.Parse(args)
	if err != nil {
		if code, ok := jsonLoginParseFailure(err, out); ok {
			return nil, code, true
		}
		out.Errorf("%v", err)
		out.Errorf("run 'gobble --help' for usage")
		return nil, ExitUsage, true
	}
	return kctx, ExitOK, false
}

// exitCode maps a command's error to the D10 table, printing its message.
func exitCode(ctx context.Context, out *Output, err error) int {
	switch {
	case err == nil, errors.Is(err, errBrokenPipe):
		return ExitOK
	}
	if _, ok := signalExit(ctx); ok {
		return ExitFailure // Main replaces it with 128 plus the signal
	}
	if e, ok := errors.AsType[*exitError](err); ok {
		if e.msg != "" {
			out.Errorf("%s", e.msg)
		}
		return e.code
	}
	out.Errorf("%v", err)
	return ExitFailure
}

// clock is the time source; tests replace it.
var clock = time.Now

// runEnv is what Kong binds into every command's Run.
type runEnv struct {
	ctx   context.Context
	out   *Output
	stdin *os.File
	logs  *lazyLog
	// intr lets a line-session turn take the first Ctrl+C (0008-MADR D5).
	intr *interrupts
	// termIn is the process's one terminal input, which every prompt reads
	// through, made on first use by terminalIn; termInMu guards it
	// (0063-MADR D1). flushWarn gives a failed flush's warning once (D17).
	termInMu  sync.Mutex
	termIn    *terminalInput
	flushWarn sync.Once
	// waitWarn gives Q24's warning once: the terminal is read as built
	// because gobble cannot wait on it without reading it (0063-MADR D22).
	waitWarn sync.Once
	// waitQueued is that warning's error when a fallback on inline.Reader's
	// goroutine queued it, until tellWaitWarn writes it on the main flow;
	// waitMu guards it. posixTerm says standard input is a POSIX terminal,
	// gated or fallen back (0063-MADR D23, rows R156, R157).
	waitMu     sync.Mutex
	waitQueued error
	posixTerm  bool
	// now is the clock, for stream-json's session timestamp.
	now func() time.Time
	// listingsMu guards listings, the process's model listings by
	// provider (0020-MADR D3).
	listingsMu sync.Mutex
	listings   map[string]*modelListing
	// lists owns the listings' goroutines: run ends and waits for them
	// (0046-MADR D1).
	lists listingOwner
	// cache is the run's SDK cache, which every provider.Options the run
	// builds carries and endListings closes (0046-MADR D7).
	cache provider.Cache
	// sessions are the run's OAuth sessions, one per provider, which every
	// provider.Options the run builds carries, refreshCredentials drops one
	// of, and endListings waits for (0046-MADR D8).
	sessions provider.Sessions
	// provSet is the run's provider set, built once beside models.json
	// (0036-MADR D2). provOnce guards it.
	provOnce sync.Once
	provSet  *provider.Set
	// chatOnce registers models.json's Chat Completions servers on provSet
	// once (0036-MADR D5).
	chatOnce sync.Once
	// recent and recentLoaded are recent-models.json's recent models, read once,
	// under recentMu (0025-MADR D3).
	recent       []string
	recentLoaded bool
	// refusals and refusalsLoaded are reasoning-refusals.json, read once,
	// under refusalMu (0036-MADR D4).
	refusals       refusalLists
	refusalsLoaded bool
	// observations and observationsLoaded are limit-observations.json, read
	// once, under observationMu (0066-MADR D7; 0071-MADR P24 to P27).
	observations       []savedObservation
	observationsLoaded bool
	// modelDataWarned are the model data warnings this run has logged, by
	// what they warn of, under modelDataMu: each is logged once (0072-MADR
	// D3, D18, M69).
	modelDataMu     sync.Mutex
	modelDataWarned map[string]bool
	// tools are the in-process agent's tools; nil is the built-ins.
	tools []tool.Tool
	// mcpOff and mcpAllow are the MCP tools the tool flags select, and
	// noGrants leaves request_permissions out.
	mcpOff   bool
	mcpAllow func(name string) bool
	noGrants bool
	// store is the agent's session store, opened on first use.
	store session.Store
	// prompt is what the prompt flags set: the system prompt's replacement
	// and addendum texts, and whether to read context files (0005-PLAN
	// F3-1). agentOptions adds the config directory.
	prompt acpserver.PromptOptions
	// overrides are --set's, the settings' session layer (0018-MADR D10).
	overrides []config.Override
	// process are the process keys' settings, resolved once.
	process processOnce
	// pick is what --provider, --model and --thinking chose; defaultModelOnce
	// gives defaultModel's warning once (0005-PLAN F4-2c).
	pick             modelPick
	defaultModelOnce sync.Once
	// inLine is whether the run is the line session, where /login signs in
	// (0025-MADR D6).
	inLine bool
	// startAt is where a run starts when --model names no provider, chosen
	// once, and again after a sign-in or a sign-out; startMu guards it
	// (0025-MADR D3, D6).
	startMu sync.Mutex
	started bool
	startAt startChoice
	// creds is the run's credential resolver, built when first needed and
	// again after a sign-in or a sign-out; credsBuild guards it. credsMu
	// guards credWarned, the providers whose resolution error was warned,
	// knownKeys, the credentials resolved once for lookups, their keys and
	// workspaces (0005-PLAN F4-3a; 0035-MADR D5), and
	// credGen, each provider's credential generation (0025-MADR D4).
	credsBuild sync.Mutex
	creds      *auth.Resolver
	// authErr is why creds has no auth.json although the file exists: it
	// could not be read or parsed (0045-MADR D5). credsBuild guards it.
	authErr    error
	credsMu    sync.Mutex
	credWarned map[string]bool
	knownKeys  map[string]auth.Credential
	credGen    map[string]uint64
	// personality is --personality's, by its canonical name, which the
	// run's sessions start with; "" leaves the setting's (0022-PLAN P3).
	// personalitySet are the personalities, loaded once.
	personality       string
	personalitiesOnce sync.Once
	personalitySet    *personality.Set
	// promptSet are the user's prompt overrides, read once (0022-PLAN P5).
	promptsOnce sync.Once
	promptSet   promptOverrides
	// trust is the run's project trust (0005-PLAN F6-1).
	trust trustState
	// yes is --yes: print mode allows once what a rule or the mode asks
	// (0005-PLAN F6-2 decision 8).
	yes bool
	// mode is --mode, the mode the run's sessions start in (0005-PLAN F6-4
	// decision 7).
	mode string
}

func (e *runEnv) log() *slog.Logger { return e.logs.get() }

// migrateAuth runs auth.json's one-time migration before the command
// (0052-MADR D9, as the owner's answers of 2026-11-21 amend it): for every
// command run dispatches but auth status and auth check, with or without
// --json, which read the file as it is; completion mode never reaches run
// (Q1). Under auth login --json it first silences the command as its Run
// does, so nothing of the migration reaches stderr there or under gobble
// acp (row M55). Directories that cannot be resolved skip it with one
// warning (M52). A failure is a Warn record in the log file alone, through
// the log's file-only view, and, but under those two commands, one warning
// line on stderr; the command runs on with the file as it is (M48). An
// auth.json that cannot be read or parsed is the command's to report, so
// that failure goes to the log file alone (M66). It reports false when the
// run's context ended, so the command is not run (M51).
func (e *runEnv) migrateAuth(command string, loginJSON bool) bool {
	if strings.HasPrefix(command, "auth status") || strings.HasPrefix(command, "auth check") {
		return true
	}
	if loginJSON {
		e.out.err = io.Discard
		e.logs.stderr = nil
	}
	silent := e.logs.stderr == nil
	dirs, _, err := systemDirs()
	if err != nil {
		if !silent {
			e.out.Warnf("auth.json not migrated: %v", err)
		}
		return e.ctx.Err() == nil
	}
	path := filepath.Join(dirs.Config, "auth.json")
	_, err = auth.Migrate(e.ctx, auth.MigrateOptions{
		AuthPath:     path,
		StateDir:     dirs.State,
		Settings:     globalSetting(dirs.Config),
		PlanSession:  planSessionHook(path),
		CarryConsent: carryConsentHook(path),
		Log:          e.logs.fileOnly,
	})
	if e.ctx.Err() != nil {
		return false
	}
	if err != nil {
		e.logs.fileOnly().WarnContext(e.ctx, "auth.json migration", slog.String("message", err.Error()))
		if !silent && !errors.Is(err, auth.ErrUnreadable) {
			e.out.Warnf("auth.json migration: %v", err)
		}
	}
	return true
}

// jsonFlag is the name of a command's --json flag.
const jsonFlag = "json"

// loginJSON reports whether the parsed command is auth login with --json,
// read from the parsed command as jsonLoginParseFailure reads it from the
// parser's context.
func loginJSON(kctx *kong.Context) bool {
	sel := kctx.Selected()
	if sel == nil || sel.FullPath() != "gobble auth login" {
		return false
	}
	for _, f := range sel.Flags {
		if f.Name == jsonFlag {
			set, isBool := kctx.FlagValue(f).(bool)
			return isBool && set
		}
	}
	return false
}

// stdinReader is stdin as a reader, or nil when the process has none.
func (e *runEnv) stdinReader() io.Reader {
	if e.stdin == nil {
		return nil
	}
	return e.stdin
}

// lazyLog opens the logger on its first use, so commands that never log
// (version, config path, help) create no state directory or log file. After
// close it opens nothing and writes nothing: a record then is dropped
// (0046-MADR D5). mu guards opened, closed, logger, file and closer. file is
// the same log's file-only view (0052-PLAN P5).
type lazyLog struct {
	mu     sync.Mutex
	opened bool
	closed bool
	level  slog.Level
	out    *Output
	stderr io.Writer // nil under `gobble acp`
	logger *slog.Logger
	file   *slog.Logger
	closer io.Closer
}

// discardLogger is a closed log's: it writes nothing.
var discardLogger = slog.New(slog.DiscardHandler)

func (l *lazyLog) get() *slog.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return discardLogger
	}
	if !l.opened {
		l.opened = true
		l.open()
	}
	return l.logger
}

// fileOnly is a logger over the log file's handler alone, opening the log on
// its first use as get does: a record given to it never reaches stderr. With
// no log file it writes nothing (0052-MADR D9, row M55).
func (l *lazyLog) fileOnly() *slog.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return discardLogger
	}
	if !l.opened {
		l.opened = true
		l.open()
	}
	return l.file
}

// open opens the logger; l.mu is held. Its "no log file" warning is not
// written when the log has no stderr: under `gobble acp`, and auth status
// --json and auth login --json once they silence it (0052-MADR D9, row M55).
func (l *lazyLog) open() {
	opts := logging.Options{Level: l.level, Stderr: l.stderr, Warn: l.stderr}
	// On a terminal a record is a diagnostic, as D3's are (0024-MADR D6),
	// shown through Output at the width of the moment and kept for a
	// reprint (0030-MADR D4, as amended).
	if l.stderr != nil && l.out.caps.Err.TTY {
		opts.Show = l.out.logRecord
		opts.Home, _ = os.UserHomeDir() //nolint:errcheck // with no home, paths are shown whole
	}
	warn := func(err error) {
		if l.stderr != nil {
			l.out.Warnf("no log file: %v", err)
		}
	}
	if d, _, err := systemDirs(); err != nil {
		warn(err)
	} else {
		opts.File = filepath.Join(d.State, "logs", "gobble.log")
	}
	logger, file, closer, err := logging.OpenFileView(opts)
	if err != nil {
		// The default path failing is a warning, not a stop
		// (magic-cli-remote 0157-MADR D8).
		warn(err)
		opts.File = ""
		if logger, file, closer, err = logging.OpenFileView(opts); err != nil {
			logger, file, closer = slog.New(slog.DiscardHandler), discardLogger, nil
		}
	}
	l.logger, l.file, l.closer = logger, file, closer
}

// close closes the log file, once; after it the log opens and writes nothing.
func (l *lazyLog) close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	if l.closer == nil {
		return nil
	}
	return l.closer.Close()
}

func levelOf(s string) slog.Level {
	l, err := logging.ParseLevel(s)
	if err != nil {
		return slog.LevelInfo
	}
	return l
}

func fileOf(w io.Writer) *os.File {
	if f, ok := w.(*os.File); ok {
		return f
	}
	return nil
}
