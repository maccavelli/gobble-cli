package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Options configure Open.
type Options struct {
	// Level is the file's level. Stderr shows max(Level, LevelWarn).
	Level slog.Level
	// File is the active log file, normally <state>/logs/gobble.log. Empty
	// means no file.
	File string
	// Stderr receives warnings and errors as text. Nil means none, as under
	// `gobble acp`.
	Stderr io.Writer
	// TerminalWidth is Stderr's width when it is a terminal, where a record
	// is gobble's own diagnostic; 0, not a terminal, keeps slog's text
	// (0024-MADR D6).
	TerminalWidth int
	// Show, when set, is given each warning and error as gobble's own
	// diagnostic, unwrapped, in place of Stderr: its caller wraps it with
	// Wrap at the terminal's width when it is shown, and can draw it again
	// at another (0030-MADR D4, as amended).
	Show func(diagnostic string)
	// Home is the user's home directory, shown as ~ on a terminal; empty
	// shows every path whole.
	Home string
	// Warn receives the one report of a failed rotation. Nil means nowhere.
	Warn io.Writer
}

// Open returns the logger and the closer of its file. It fails only when
// File is set and cannot be opened; the caller decides whether that is
// fatal (an explicit path) or a warning (the default path).
func Open(opts Options) (*slog.Logger, io.Closer, error) {
	logger, _, closer, err := OpenFileView(opts)
	return logger, closer, err
}

// OpenFileView is Open, and also file, a view of the same logger that
// writes to the log file alone, through the one file handler Open builds:
// a record given to file never reaches Stderr or Show. With no File, file
// writes nothing. It opens one rolling writer, as Open does, so the view is
// not a second writer on the file (0052-PLAN P5, the owner's approval of
// 2026-11-21).
func OpenFileView(opts Options) (logger, file *slog.Logger, closer io.Closer, err error) {
	var handlers []slog.Handler
	closer = nopCloser{}
	file = slog.New(slog.DiscardHandler)
	if opts.File != "" {
		r, rerr := openRolling(opts.File, opts.Warn)
		if rerr != nil {
			return nil, nil, nil, fmt.Errorf("logging: %w", rerr)
		}
		fh := slog.NewJSONHandler(r, &slog.HandlerOptions{Level: opts.Level, ReplaceAttr: redact})
		handlers = append(handlers, fh)
		file = slog.New(fh)
		closer = r
	}
	switch {
	case opts.Show != nil:
		handlers = append(handlers, newTerminalHandler(nil, opts.Show, max(opts.Level, slog.LevelWarn), 0, opts.Home))
	case opts.Stderr != nil && opts.TerminalWidth > 0:
		handlers = append(handlers, newTerminalHandler(opts.Stderr, nil, max(opts.Level, slog.LevelWarn), opts.TerminalWidth, opts.Home))
	case opts.Stderr != nil:
		handlers = append(handlers, slog.NewTextHandler(opts.Stderr,
			&slog.HandlerOptions{Level: max(opts.Level, slog.LevelWarn), ReplaceAttr: redact}))
	}
	var h slog.Handler
	switch len(handlers) {
	case 0:
		h = slog.DiscardHandler
	case 1:
		h = handlers[0]
	default:
		h = slog.NewMultiHandler(handlers...)
	}
	return slog.New(h), file, closer, nil
}

// ParseLevel reads debug, info, warn or error, in any case.
func ParseLevel(s string) (slog.Level, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(strings.TrimSpace(s))); err != nil {
		return 0, fmt.Errorf("logging: level %q: %w", s, err)
	}
	return l, nil
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }
