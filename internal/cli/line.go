package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	xterm "golang.org/x/term"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/appdirs"
	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/internal/cli/render"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/internal/fsx"
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
	// resize notices the terminal's resizes; nil off a terminal
	// (0030-MADR D1).
	resize <-chan struct{}
	// termWidth, when set, is the terminal's width in place of the one
	// read from stdout: a test's.
	termWidth func() int
	// tr is what the session has shown, for a reprint; nil off a terminal
	// (0030-MADR D4, D6).
	tr *transcript
	// editorResize is the editor's resize notice: the terminal's, routed to
	// the editor at the prompt and to the view while a turn runs.
	editorResize chan struct{}

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
	// pasting is whether the editor turned bracketed paste on, which a
	// question turns off while it reads (0025-PLAN deviation 7).
	pasting bool
}

// runLine opens the line session on the process's terminal.
func runLine(e *runEnv, c *ChatCmd, first Prompt, cwd, base string) error {
	return newLineSession(e, c, cwd, base).run(first)
}

// newLineSession is the line session for a session in cwd, speaking the
// voice it starts with (0022-PLAN P3).
func newLineSession(e *runEnv, c *ChatCmd, cwd, base string) *lineSession {
	stdout, _ := e.out.out.(*os.File) //nolint:errcheck // not a file means no terminal to watch
	return &lineSession{
		e: e, cwd: cwd, base: base, opts: displayFor(c), voice: e.runVoice(cwd), in: e.stdin, choice: c.sessionChoice(),
		raw:    func() (func() error, error) { return editor.Raw(e.stdin) },
		resize: term.WatchResize(e.ctx, stdout),
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
	e.inLine = true // /login signs in here (0025-MADR D6)
	if ls.amb == nil {
		ls.amb = newAmbient(e.out.caps.Err.TTY)
	}
	if ls.resize != nil { // on a terminal: keep what is shown, and route resizes (0030-MADR D3, D4, D6)
		if e.out.rec == nil { // the chat command keeps it from its start (D4, as amended)
			e.out.rec = &transcript{}
		}
		ls.tr = e.out.rec
		defer func() { e.out.rec = nil }()
		ls.editorResize = make(chan struct{}, 1)
		ctx, cancel := context.WithCancel(e.ctx)
		defer cancel()
		go ls.route(ctx)
	}
	conn, err := e.startAgentWith(e.ctx, ls.askPermission)
	if err != nil {
		return err
	}
	ls.conn = conn
	defer ls.close()
	// The session opens without a credential; a turn then fails with the
	// authentication error (0008-MADR D1). On a terminal with no credential
	// at all, the sign-in is offered before the first prompt (0025-MADR
	// D7); otherwise the warning names the variables.
	setup := e.out.caps.In && !slices.ContainsFunc(provider.KeyProviders(), e.hasCredential)
	if _, err := provider.ChooseWith(e.hasCredential, e.chosenProvider()); err != nil && !setup {
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

	e.noteUntrusted(ls.cwd)
	if note := e.startNote(); note != "" { // which of several providers the run took (0025-MADR D3)
		e.out.Notef("%s", note)
	}
	if line, ok := ls.amb.welcome(ls.spoken(), e.now()); ok {
		e.out.Notef("%s", line)
	}

	var r editor.Reader
	r.Start(e.ctx, ls.in)
	ls.mu.Lock()
	ls.keys = &r
	ls.mu.Unlock()
	if setup {
		if err := ls.firstRun(); err != nil {
			return err
		}
	}
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
	opts := editor.Options{History: ls.history(), Complete: ls.complete,
		Header: func() string { return workspaceLine(ls.cwd) }, Width: ls.width, Height: ls.height, Resize: ls.editorResize}
	if ls.tr != nil {
		opts.Shown = ls.tr.drawn
		opts.Reprint = func(width int) string { return reprintText(ls.tr, width, e.out.caps.Out.Color) }
	}
	if !e.out.caps.Unicode {
		opts.Continue = "... "
	}
	ed := editor.New(&r, e.out.out, e.out.err, opts)
	edRestored := false
	defer func() {
		if edRestored {
			return
		}
		if rerr := ed.Restore(); rerr != nil {
			e.out.Warnf("restore the terminal: %v", rerr)
		}
	}()
	ls.pasting = true

	// outOfRaw runs f with the terminal out of raw mode, so output keeps its
	// line ends and Ctrl+C is a signal f takes. The terminal is raw only
	// while the editor reads: a submission, local command or prompt, is
	// handled out of it (0024-MADR D1).
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
			// /exit, /quit, Ctrl+D and a second Ctrl+C all end the read;
			// the deferred restores find nothing to undo.
			edRestored = true
			ls.goodbye(ed, restore)
			restore = nil
			return nil
		case e.ctx.Err() != nil:
			return context.Cause(e.ctx)
		case err != nil:
			return failf("%v", err)
		}
		err = outOfRaw(func() error {
			handled, err := ls.slash(text, ls.turn)
			if err != nil || handled {
				return err
			}
			return ls.turn(Prompt{Text: text})
		})
		if err != nil {
			return err
		}
	}
}

// replyWidth is the width a turn's replies wrap to: the terminal's, at
// most 120 columns, or 0, no wrapping, when stdout is not a terminal
// (0028-MADR D1, D2; Q1).
func (ls *lineSession) replyWidth() int {
	if !ls.e.out.caps.Out.TTY {
		return 0
	}
	return min(ls.width(), editor.MaxWidth)
}

// goodbye turns bracketed paste off and leaves raw mode (restore, when not
// nil), then writes the goodbye, so it ends with a real line end
// (0027-MADR D3, as amended for P5).
func (ls *lineSession) goodbye(ed *editor.Editor, restore func() error) {
	e := ls.e
	if err := ed.Restore(); err != nil {
		e.out.Warnf("restore the terminal: %v", err)
	}
	if restore != nil {
		if err := restore(); err != nil {
			e.out.Warnf("restore the terminal: %v", err)
		}
	}
	if line, ok := ls.amb.goodbye(ls.spoken()); ok {
		e.out.Notef("%s", line)
	}
}

// height is the terminal's height, for a prompt taller than the screen
// (0027-MADR D3), or 0 when standard output is not a terminal.
func (ls *lineSession) height() int {
	if f, ok := ls.e.out.out.(*os.File); ok && ls.e.out.caps.Out.TTY {
		if _, h, err := xterm.GetSize(int(f.Fd())); err == nil && h > 0 { //nolint:gosec // G115: a terminal's descriptor fits an int
			return h
		}
	}
	return 0
}

// workspaceLine is the line above each new prompt: the name of the
// repository cwd is in, else cwd's own, and the branch checked out, read
// from HEAD (0024-MADR D8).
func workspaceLine(cwd string) string {
	name := filepath.Base(cwd)
	if repo, _, ok := fsx.GitRepo(cwd); ok {
		name = filepath.Base(repo)
	}
	if b := fsx.GitBranch(cwd); b != "" {
		return name + " (" + b + ")"
	}
	return name
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
	opts.width = ls.replyWidth
	v := newView(e.out, opts, e.now)
	v.reprintWidth = ls.width
	v.afterTurn = func() { notifyOnce(ls.editorResize) }
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
		v.cutOff()
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

// route takes the terminal's resize notices until ctx is done: to the view
// while a turn runs, which reprints at the reply's next line end, else to
// the editor, which redraws its line and reprints (0030-MADR D2, D3).
func (ls *lineSession) route(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ls.resize:
		}
		ls.mu.Lock()
		v := ls.view
		ls.mu.Unlock()
		if v != nil {
			v.resized()
		} else {
			notifyOnce(ls.editorResize)
		}
	}
}

// notifyOnce puts a notice on ch unless one already waits there.
func notifyOnce(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
