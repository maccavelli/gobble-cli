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
	"github.com/maccavelli/gobble-cli/personality"
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
	// voice is the session's, as _gobble/personality last said; the CLI's
	// own lines speak it (0022-PLAN P3).
	voice *personality.Voice
	// amb is the ambient text, made when the session runs (0022-PLAN P4).
	amb *ambient
	// keys is the editor's reader, which the permission prompt reads a
	// key from while a turn runs.
	keys *editor.Reader
}

// runLine opens the line session on the process's terminal.
func runLine(e *runEnv, c *ChatCmd, first Prompt, cwd, base string) error {
	return newLineSession(e, c, cwd, base).run(first)
}

// newLineSession is the line session for a session in cwd, speaking the
// voice it starts with (0022-PLAN P3).
func newLineSession(e *runEnv, c *ChatCmd, cwd, base string) *lineSession {
	return &lineSession{
		e: e, cwd: cwd, base: base, opts: displayFor(c), voice: e.runVoice(cwd), in: e.stdin, choice: c.sessionChoice(),
		raw: func() (func() error, error) { return editor.Raw(e.stdin) },
	}
}

func displayFor(c *ChatCmd) display {
	home, _ := os.UserHomeDir() //nolint:errcheck // no home only means paths are not shortened
	return display{showThinking: c.ShowThinking, verbose: c.Verbose, stats: c.Stats, home: home}
}

// spinnerLabel is the spinner's label for a turn that started at start: a
// working line, or the slow turn's (0022-MADR D12).
func (ls *lineSession) spinnerLabel(start time.Time) func() string {
	return func() string { return ls.amb.label(ls.spoken(), ls.e.now().Sub(start)) }
}

// spoken is the session's voice now.
func (ls *lineSession) spoken() *personality.Voice {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return ls.voice
}

func (ls *lineSession) run(first Prompt) error {
	e := ls.e
	if ls.amb == nil {
		ls.amb = newAmbient(e.out.caps.Err.TTY)
	}
	conn, err := e.startAgentWith(e.ctx, ls.askPermission)
	if err != nil {
		return err
	}
	ls.conn = conn
	defer ls.close()
	// The session opens without a credential; a turn then fails with the
	// authentication error (0008-MADR D1). gobble configure arrives with
	// 0005-PLAN F4, so the warning names the variables.
	if _, err := provider.ChooseWith(e.hasCredential, e.chosenProvider()); err != nil {
		e.out.Lead(ls.spoken(), personality.LeadNoCredential)
		e.out.Warnf("%s", e.noCredentialMessage())
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

	if line, ok := ls.amb.welcome(ls.spoken(), e.now()); ok {
		e.out.Notef("%s", line)
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
			// /exit, /quit, Ctrl+D and a second Ctrl+C all end the read.
			if line, ok := ls.amb.goodbye(ls.spoken()); ok {
				e.out.Notef("%s", line)
			}
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
	opts := ls.opts
	opts.voice = ls.spoken()
	v := newView(e.out, opts, e.now)
	ls.amb.startTurn()
	start := e.now()
	sp := render.StartSpinner(e.ctx, e.out.err, e.out.caps.Err, e.out.caps.Unicode, ls.spinnerLabel(start))
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
		e.out.Lead(ls.spoken(), personality.LeadSessionFailed)
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
		e.out.Lead(ls.spoken(), personality.LeadImages)
		e.out.Errorf("the agent does not accept images (%s)", imageNames(p))
		return nil
	case errors.Is(perr, acpclient.ErrAuthRequired):
		v.quiet()
		e.out.Lead(ls.spoken(), personality.LeadAuth)
		e.out.Errorf("%s", e.authMessage())
		return nil
	case perr != nil:
		if err := v.finish(res); err != nil {
			return err
		}
		e.out.Lead(ls.spoken(), personality.LeadTurnFailed)
		e.out.Errorf("%v", perr)
		return nil
	}
	if err := v.finish(res); err != nil {
		return err
	}
	if line, ok := ls.amb.endTurn(ls.spoken(), e.now().Sub(start)); ok {
		e.out.Notef("%s", line)
	}
	if res.SwitchTo != "" {
		return ls.switchTo(res.SwitchTo)
	}
	return nil
}

// onUpdate hands the session's updates to the running turn's view.
func (ls *lineSession) onUpdate(u acpclient.Update) {
	if ls.amb != nil {
		ls.amb.note(u)
	}
	if u.Kind == acpclient.KindPersonality {
		voice := ls.e.voiceNamed(u.Personality.Name)
		ls.mu.Lock()
		ls.voice = voice
		v := ls.view
		ls.mu.Unlock()
		if v != nil {
			v.setVoice(voice)
		}
		return
	}
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
