package permission

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// TrustFile is the saved trust decisions' file in gobble's config
// directory (0005-PLAN F6-1 decision 3).
const TrustFile = "trust.json"

// The values of defaultProjectTrust (decision 4).
const (
	DefaultAsk    = "ask"
	DefaultAlways = "always"
	DefaultNever  = "never"
)

// TrustSource is the step of DecideTrust's order that decided.
type TrustSource int

// The steps, in DecideTrust's order. SourceHook is F7's, and decides
// nothing until then.
const (
	SourceFlag    TrustSource = iota + 1 // --approve or --no-approve
	SourceNothing                        // the project has no file that needs trust
	SourceHook                           // a hook's answer (0005-PLAN F7)
	SourceSaved                          // trust.json
	SourceDefault                        // defaultProjectTrust: always or never
	SourceAnswer                         // the question's answer
	SourceAsk                            // ask, with no one to ask
)

// TrustDecision is whether a project root is trusted, which step said so,
// and, for a saved decision, the path it was saved for.
type TrustDecision struct {
	Trusted bool
	Source  TrustSource
	At      string
}

// String says where the decision came from, for gobble trust and /trust.
func (d TrustDecision) String() string {
	word := "not trusted"
	if d.Trusted {
		word = "trusted"
	}
	switch d.Source {
	case SourceFlag:
		return word + ", for this run (--approve or --no-approve)"
	case SourceNothing:
		return word + ": the project has no files that need trust"
	case SourceSaved:
		return word + ", saved for " + d.At
	case SourceDefault:
		return word + ", by defaultProjectTrust"
	case SourceAnswer:
		return word + ", as answered for this run"
	default:
		return word + ": nothing is saved, and there was no one to ask (gobble trust saves a decision)"
	}
}

// TrustInput is what DecideTrust weighs, step by step.
type TrustInput struct {
	// Root is the project root (0018-MADR D2).
	Root string
	// Flag is --approve's (true) or --no-approve's (false) for this run;
	// nil is neither.
	Flag *bool
	// Needs reports whether the project has a file that needs trust.
	Needs bool
	// Saved is the nearest saved decision, from TrustStore.Lookup; ok is
	// false when there is none.
	Saved func(root string) (trusted bool, at string, ok bool, err error)
	// Default is defaultProjectTrust: ask, always or never.
	Default string
	// Ask asks the user, where someone can be asked; nil or a false ok is
	// no answer.
	Ask func(root string) (trusted bool, ok bool)
}

// DecideTrust is Pi's order (0005-PLAN F6-1 decision 2): the run's flag;
// trusted when nothing needs trust; the hook (F7's); the nearest saved
// decision; defaultProjectTrust; then the question, and with no answer,
// not trusted.
func DecideTrust(in TrustInput) (TrustDecision, error) {
	if in.Flag != nil {
		return TrustDecision{Trusted: *in.Flag, Source: SourceFlag}, nil
	}
	if !in.Needs {
		return TrustDecision{Trusted: true, Source: SourceNothing}, nil
	}
	if in.Saved != nil {
		trusted, at, ok, err := in.Saved(in.Root)
		if err != nil {
			return TrustDecision{Source: SourceSaved}, err
		}
		if ok {
			return TrustDecision{Trusted: trusted, Source: SourceSaved, At: at}, nil
		}
	}
	switch in.Default {
	case DefaultAlways:
		return TrustDecision{Trusted: true, Source: SourceDefault}, nil
	case DefaultNever:
		return TrustDecision{Source: SourceDefault}, nil
	}
	if in.Ask != nil {
		if trusted, ok := in.Ask(in.Root); ok {
			return TrustDecision{Trusted: trusted, Source: SourceAnswer}, nil
		}
	}
	return TrustDecision{Source: SourceAsk}, nil
}

// TrustStore is trust.json: each saved path, by its real path, true for
// trusted and false for not (decision 3).
type TrustStore struct {
	// Path is the file, TrustFile in gobble's config directory.
	Path string
	// Home is the home directory, never saved; "" is os.UserHomeDir's.
	Home string
}

// ErrTooBroad is Save's refusal of a home directory or a filesystem root.
var ErrTooBroad = errors.New("trusting it would trust every project under it")

// Load is every saved decision; a missing file is none.
func (s TrustStore) Load() (map[string]bool, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", TrustFile, err)
	}
	m := map[string]bool{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", TrustFile, err)
	}
	return m, nil
}

// Lookup is the decision saved for root's real path or the nearest of its
// parents, and the path it was saved for (Pi's trust-manager.ts:45-59).
func (s TrustStore) Lookup(root string) (trusted bool, at string, ok bool, err error) {
	m, err := s.Load()
	if err != nil {
		return false, "", false, err
	}
	p, err := realPath(root)
	if err != nil {
		return false, "", false, err
	}
	for d := p; ; {
		for k, v := range m {
			if samePath(k, d) {
				return v, k, true, nil
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			return false, "", false, nil
		}
		d = parent
	}
}

// Save records trusted for dir's real path, which it returns. A home
// directory or a filesystem root is refused with ErrTooBroad
// (grok-build's rule).
func (s TrustStore) Save(ctx context.Context, dir string, trusted bool) (string, error) {
	p, err := realPath(dir)
	if err != nil {
		return "", err
	}
	if TooBroad(p, s.Home) {
		return p, fmt.Errorf("%s: %w", p, ErrTooBroad)
	}
	return p, s.modify(ctx, func(m map[string]bool) bool {
		for k := range m {
			if samePath(k, p) {
				delete(m, k)
			}
		}
		m[p] = trusted
		return true
	})
}

// Forget removes the decision saved for dir's real path, which it returns
// with whether there was one.
func (s TrustStore) Forget(ctx context.Context, dir string) (string, bool, error) {
	p, err := realPath(dir)
	if err != nil {
		return "", false, err
	}
	removed := false
	err = s.modify(ctx, func(m map[string]bool) bool {
		for k := range m {
			if samePath(k, p) {
				delete(m, k)
				removed = true
			}
		}
		return removed
	})
	return p, removed, err
}

// modify changes the saved decisions under the file's lock, and writes
// them, sorted and indented two spaces, when change reports a change.
func (s TrustStore) modify(ctx context.Context, change func(map[string]bool) bool) (err error) {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("%s: %w", TrustFile, err)
	}
	release, err := filelock.Lock(ctx, s.Path+".lock", filelock.EditPolicy)
	if err != nil {
		return fmt.Errorf("%s: %w", TrustFile, err)
	}
	defer func() { err = errors.Join(err, release()) }()
	m, err := s.Load()
	if err != nil {
		return err
	}
	if !change(m) {
		return nil
	}
	data, err := json.Marshal(m, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("%s: %w", TrustFile, err)
	}
	return fsx.WriteFileMode(s.Path, append(data, '\n'), 0o600)
}

// TooBroad reports whether p, a real path, is home or a filesystem root,
// which a decision is never saved for (decision 3). home "" is
// os.UserHomeDir's.
func TooBroad(p, home string) bool {
	if filepath.Dir(p) == p {
		return true
	}
	if home == "" {
		home, _ = os.UserHomeDir() //nolint:errcheck // with no home, only roots are refused
	}
	if home == "" {
		return false
	}
	if h, err := filepath.EvalSymlinks(home); err == nil {
		home = h
	}
	return samePath(p, home)
}

// realPath is dir's absolute path with its links resolved.
func realPath(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// samePath compares two clean paths as the OS does: without regard to
// case on Windows.
func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
