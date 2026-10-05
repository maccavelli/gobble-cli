package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alecthomas/kong"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/internal/logging"
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
	ctx, stop := watchSignals(ctx, term.ShutdownSignals(), signal.Notify)
	defer stop()
	code := run(ctx, k, &root, args, stdin, out)
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

func run(ctx context.Context, k *kong.Kong, root *Root, args []string, stdin *os.File, out *Output) int {
	kctx, code, done := parse(k, args, out)
	if done {
		return code
	}
	logs := &lazyLog{level: levelOf(root.LogLevel), out: out, stderr: out.err}
	if strings.HasPrefix(kctx.Command(), "acp") {
		logs.stderr = nil // stdio is the protocol (0004-PLAN Phase 2 step 4)
	}
	e := &runEnv{ctx: ctx, out: out, stdin: stdin, logs: logs}
	code = exitCode(ctx, out, kctx.Run(e))
	if err := logs.close(); err != nil {
		out.Warnf("close log file: %v", err)
	}
	return code
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

// runEnv is what Kong binds into every command's Run.
type runEnv struct {
	ctx   context.Context
	out   *Output
	stdin *os.File
	logs  *lazyLog
}

func (e *runEnv) log() *slog.Logger { return e.logs.get() }

// stdinReader is stdin as a reader, or nil when the process has none.
func (e *runEnv) stdinReader() io.Reader {
	if e.stdin == nil {
		return nil
	}
	return e.stdin
}

// lazyLog opens the logger on its first use, so commands that never log
// (version, config path, help) create no state directory or log file.
type lazyLog struct {
	once   sync.Once
	level  slog.Level
	out    *Output
	stderr io.Writer // nil under `gobble acp`
	logger *slog.Logger
	closer io.Closer
}

func (l *lazyLog) get() *slog.Logger {
	l.once.Do(func() {
		opts := logging.Options{Level: l.level, Stderr: l.stderr, Warn: l.stderr}
		if d, _, err := systemDirs(); err != nil {
			l.out.Warnf("no log file: %v", err)
		} else {
			opts.File = filepath.Join(d.State, "logs", "gobble.log")
		}
		logger, closer, err := logging.Open(opts)
		if err != nil {
			// The default path failing is a warning, not a stop
			// (magic-cli-remote 0157-MADR D8).
			l.out.Warnf("no log file: %v", err)
			opts.File = ""
			if logger, closer, err = logging.Open(opts); err != nil {
				logger, closer = slog.New(slog.DiscardHandler), nil
			}
		}
		l.logger, l.closer = logger, closer
	})
	return l.logger
}

func (l *lazyLog) close() error {
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
