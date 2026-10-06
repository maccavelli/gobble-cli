package cli

import (
	"bufio"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// sessionChoice is what the session flags ask of a run (0008-MADR D12):
// continue the newest session, resume a named one, or start one, chosen,
// forked or named.
type sessionChoice struct {
	continueLast bool
	resume       string // --session: a path, an id or a prefix
	fork         string // --fork: a path or an id
	opts         acpclient.NewSessionOptions
}

func (f *SharedFlags) sessionChoice() sessionChoice {
	return sessionChoice{continueLast: f.Continue, resume: f.Session, fork: f.Fork,
		opts: acpclient.NewSessionOptions{ID: f.SessionID, Name: f.Name}}
}

// resumes reports whether the run continues a stored session.
func (ch sessionChoice) resumes() bool { return ch.continueLast || ch.resume != "" }

// configureStore picks the agent's store: memory for --no-session, a flat
// directory for --session-dir (as Pi keeps a custom one), else the data
// directory's sessions/.
func (f *SharedFlags) configureStore(e *runEnv, base string) error {
	switch {
	case f.NoSession:
		e.store = session.NewMemoryStore()
	case f.SessionDir != "":
		dir, err := resolvePath(base, f.SessionDir)
		if err != nil {
			return usageErrorf("--session-dir %s: %v", f.SessionDir, err)
		}
		e.store = jsonl.New(dir, jsonl.Options{Flat: true, Logger: e.log()})
	}
	return nil
}

// openSession resumes or starts the run's session. resumed is nil for a new
// session. A session open in another process exits 1 with the agent's
// message (0008-MADR D19 item 8); a bad id, prefix or path exits 2.
func openSession(ctx context.Context, conn *acpclient.Conn, ch sessionChoice, cwd, base string, on func(acpclient.Update)) (*acpclient.Session, *acpclient.Resumed, error) {
	id, err := ch.resumeID(ctx, conn, cwd, base)
	if err != nil {
		return nil, nil, err
	}
	if id != "" {
		s, r, err := conn.ResumeSession(ctx, id, cwd, on)
		switch {
		case errors.Is(err, acpclient.ErrSessionLocked):
			return nil, nil, failf("%v", err)
		case err != nil:
			return nil, nil, failf("resume %s: %v", id, err)
		}
		return s, &r, nil
	}
	opts := ch.opts
	if ch.fork != "" {
		if opts.ForkFrom, err = resolveSession(ctx, conn, ch.fork, base, "--fork"); err != nil {
			return nil, nil, err
		}
	}
	s, err := conn.NewSessionWith(ctx, cwd, opts, on)
	switch {
	case errors.Is(err, acpclient.ErrInvalidSession):
		return nil, nil, usageErrorf("%v", err)
	case err != nil:
		return nil, nil, failf("%v", err)
	}
	return s, nil, nil
}

// resumeID is the session to resume: the newest with a file in cwd for -c
// (none starts a new one, as Pi's continueRecent does), the one --session
// names, or "" for a new session.
func (ch sessionChoice) resumeID(ctx context.Context, conn *acpclient.Conn, cwd, base string) (string, error) {
	switch {
	case ch.resume != "":
		return resolveSession(ctx, conn, ch.resume, base, "--session")
	case ch.continueLast:
		list, err := conn.ListSessions(ctx, cwd)
		if err != nil {
			return "", failf("%v", err)
		}
		for _, s := range list {
			if !s.UpdatedAt.IsZero() {
				return s.ID, nil
			}
		}
	}
	return "", nil
}

// resolveSession is the id a --session or --fork value names: a session
// file's header id (the file must be in the store), an id, or a unique
// prefix of an id or a title.
func resolveSession(ctx context.Context, conn *acpclient.Conn, value, base, flag string) (string, error) {
	list, err := conn.ListSessions(ctx, "")
	if err != nil {
		return "", failf("%v", err)
	}
	inStore := func(id string) bool {
		for _, s := range list {
			if s.ID == id {
				return true
			}
		}
		return false
	}
	if strings.HasSuffix(value, ".jsonl") || strings.ContainsAny(value, `/\`) {
		path, err := resolvePath(base, value)
		if err != nil {
			return "", usageErrorf("%s %s: %v", flag, value, err)
		}
		id, err := headerID(path)
		if err != nil {
			return "", usageErrorf("%s %s: %v", flag, value, err)
		}
		if !inStore(id) {
			return "", usageErrorf("%s %s: session %s is not in the session store", flag, value, id)
		}
		return id, nil
	}
	if inStore(value) {
		return value, nil
	}
	var matches []acpclient.SessionInfo
	lower := strings.ToLower(value)
	for _, s := range list {
		if strings.HasPrefix(s.ID, value) || strings.HasPrefix(strings.ToLower(s.Title), lower) {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return "", usageErrorf("%s %s: no session matches", flag, value)
	case 1:
		return matches[0].ID, nil
	}
	ids := make([]string, len(matches))
	for i, m := range matches {
		ids[i] = m.ID
	}
	return "", usageErrorf("%s %s matches %d sessions: %s", flag, value, len(matches), strings.Join(ids, ", "))
}

// headerID is the id in a session file's header line.
func headerID(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // G304: the user names the session file
	if err != nil {
		return "", fmt.Errorf("%w", unwrapPathError(err))
	}
	defer f.Close() //nolint:errcheck // read only
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var h session.Header
		if err := json.Unmarshal([]byte(line), &h); err != nil || h.Type != session.TypeHeader || h.ID == "" {
			return "", errors.New("not a session file")
		}
		return string(h.ID), nil
	}
	return "", errors.New("not a session file")
}

func unwrapPathError(err error) error {
	if pe, ok := errors.AsType[*os.PathError](err); ok {
		return pe.Err
	}
	return err
}

// resumedLine is the line the line session prints when it resumes a
// session (0002-PLAN Phase 4, owner's decision of 2026-10-06).
func resumedLine(r acpclient.Resumed) string {
	title := r.Title
	if title == "" {
		title = "(no messages)"
	}
	return fmt.Sprintf("resumed %s (%d messages)", title, r.Messages)
}
