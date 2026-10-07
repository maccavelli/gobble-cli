package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/appdirs"
	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/internal/cli/render"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// cancelWait bounds the session/cancel a Ctrl+C sends.
const cancelWait = 2 * time.Second

// lineSession is the interactive session (0008-MADR D5): prompts from the
// line editor, each sent as one turn to the in-process agent over ACP.
type lineSession struct {
	e      *runEnv
	cwd    string
	base   string
	opts   display
	choice sessionChoice
	// in is the terminal's input, owned by one editor.Reader for the life
	// of the process (0008-MADR D6).
	in io.Reader
	// raw puts the terminal in raw mode and returns the restore.
	raw func() (restore func() error, err error)

	conn *acpclient.Conn
	sess *acpclient.Session

	mu   sync.Mutex
	view *view
	// keys is the editor's reader, which the permission prompt reads a
	// key from while a turn runs.
	keys *editor.Reader
}

// runLine opens the line session on the process's terminal.
func runLine(e *runEnv, c *ChatCmd, first Prompt, cwd, base string) error {
	ls := &lineSession{
		e: e, cwd: cwd, base: base, opts: displayFor(c), in: e.stdin, choice: c.sessionChoice(),
		raw: func() (func() error, error) { return editor.Raw(e.stdin) },
	}
	return ls.run(first)
}

func displayFor(c *ChatCmd) display {
	home, _ := os.UserHomeDir() //nolint:errcheck // no home only means paths are not shortened
	return display{showThinking: c.ShowThinking, verbose: c.Verbose, stats: c.Stats, home: home}
}

func (ls *lineSession) run(first Prompt) error {
	e := ls.e
	conn, err := e.startAgentWith(e.ctx, ls.askPermission)
	if err != nil {
		return err
	}
	ls.conn = conn
	defer ls.close()
	// The session opens without a credential; a turn then fails with the
	// authentication error (0008-MADR D1). gobble configure arrives with
	// 0005-PLAN F4, so the warning names the variables.
	if _, err := provider.Choose(os.Getenv); err != nil {
		e.out.Warnf("%s", noCredentialMessage())
	}
	// A resumed session opens at once, from its file, and says so; a new one
	// is created with the first prompt (D5).
	if ls.choice.resumes() {
		s, r, err := openSession(e.ctx, conn, ls.choice, ls.cwd, ls.base, ls.onUpdate)
		if err != nil {
			return err
		}
		ls.sess = s
		if err := e.out.Resultf("%s\n", dimmed(resumedLine(*r), e.out.caps.Out.Color)); err != nil {
			return err
		}
	}

	var r editor.Reader
	r.Start(e.ctx, ls.in)
	ls.mu.Lock()
	ls.keys = &r
	ls.mu.Unlock()
	restore, err := ls.raw()
	if err != nil {
		return failf("%v", err)
	}
	defer func() {
		if restore != nil {
			if rerr := restore(); rerr != nil {
				e.out.Warnf("restore the terminal: %v", rerr)
			}
		}
	}()
	opts := editor.Options{History: ls.history(), Complete: ls.complete}
	if !e.out.caps.Unicode {
		opts.Continue = "... "
	}
	ed := editor.New(&r, e.out.out, e.out.err, opts)
	defer ed.Restore()

	// outOfRaw runs f with the terminal out of raw mode: a turn, so output
	// keeps its line ends and Ctrl+C is a signal the turn takes, or the
	// external editor.
	outOfRaw := func(f func() error) error {
		if rerr := restore(); rerr != nil {
			return failf("restore the terminal: %v", rerr)
		}
		restore = nil
		ferr := f()
		next, rerr := ls.raw()
		if rerr != nil {
			return errors.Join(ferr, failf("%v", rerr))
		}
		restore = next
		return ferr
	}
	turn := func(p Prompt) error { return outOfRaw(func() error { return ls.turn(p) }) }
	if first.Text != "" || len(first.Images) > 0 {
		if err := turn(first); err != nil {
			return err
		}
	}
	for {
		text, err := ed.ReadPrompt(e.ctx)
		switch {
		case errors.Is(err, io.EOF), errors.Is(err, editor.ErrExit):
			return nil
		case e.ctx.Err() != nil:
			return context.Cause(e.ctx)
		case err != nil:
			return failf("%v", err)
		}
		handled, err := ls.slash(text, turn, outOfRaw)
		if err != nil {
			return err
		}
		if handled {
			continue
		}
		if err := turn(Prompt{Text: text}); err != nil {
			return err
		}
	}
}

// history opens <state>/history. Without it the session still runs.
func (ls *lineSession) history() editor.History {
	dirs, _, err := systemDirs()
	if err == nil {
		err = appdirs.EnsurePrivateDir(dirs.State)
	}
	var h *editor.FileHistory
	if err == nil {
		h, err = editor.OpenHistory(filepath.Join(dirs.State, "history"))
	}
	if err != nil {
		ls.e.out.Warnf("no history: %v", err)
		return nil
	}
	return h
}

// turn sends one prompt. session/new goes with the first one: nothing
// reaches the agent before the user asks (D5; D19 item 6). A turn's error
// is printed and the session goes on; only the process's own cancellation
// ends it.
func (ls *lineSession) turn(p Prompt) error {
	e := ls.e
	v := newView(e.out, ls.opts, e.now)
	sp := render.StartSpinner(e.ctx, e.out.err, e.out.caps.Err, e.out.caps.Unicode, "working")
	v.beforeWrite = func() {
		if err := sp.Stop(); err != nil {
			e.log().WarnContext(e.ctx, "spinner", slog.String("error", err.Error()))
		}
	}
	defer v.quiet()
	ls.mu.Lock()
	ls.view = v
	ls.mu.Unlock()
	defer func() {
		ls.mu.Lock()
		ls.view = nil
		ls.mu.Unlock()
	}()
	if err := ls.ensureSession(); err != nil {
		v.quiet()
		e.out.Errorf("%v", err)
		return e.ctx.Err()
	}
	release := e.intr.during(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(e.ctx), cancelWait)
		defer cancel()
		if err := ls.sess.Cancel(ctx); err != nil {
			e.out.Warnf("%v", err)
		}
	})
	res, perr := ls.sess.Prompt(e.ctx, clientPrompt(p))
	release()
	if e.ctx.Err() != nil {
		return context.Cause(e.ctx)
	}
	switch {
	case errors.Is(perr, acpclient.ErrImagesUnsupported):
		v.quiet()
		e.out.Errorf("the agent does not accept images (%s)", imageNames(p))
		return nil
	case errors.Is(perr, acpclient.ErrAuthRequired):
		v.quiet()
		e.out.Errorf("%s", noCredentialMessage())
		return nil
	case perr != nil:
		if err := v.finish(res); err != nil {
			return err
		}
		e.out.Errorf("%v", perr)
		return nil
	}
	if err := v.finish(res); err != nil {
		return err
	}
	if res.SwitchTo != "" {
		return ls.switchTo(res.SwitchTo)
	}
	return nil
}

// onUpdate hands the session's updates to the running turn's view.
func (ls *lineSession) onUpdate(u acpclient.Update) {
	ls.mu.Lock()
	v := ls.view
	ls.mu.Unlock()
	if v != nil {
		v.on(u)
	}
}

// close ends the session and the agent. Failures are warnings: the user
// has already left.
func (ls *lineSession) close() {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ls.e.ctx), closeWait)
	defer cancel()
	var errs []error
	if ls.sess != nil {
		if err := ls.sess.Close(ctx); err != nil && ls.e.ctx.Err() == nil {
			errs = append(errs, err)
		}
	}
	if err := ls.conn.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		ls.e.out.Warnf("%v", errors.Join(errs...))
	}
}
