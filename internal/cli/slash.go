package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/command"
	"github.com/maccavelli/gobble-cli/internal/cli/editor"
)

// localCommands are the line session's own commands, united with the
// agent's (0008-MADR D13). /exit and /quit end the editor's read itself.
var localCommands = []acpclient.Command{
	{Name: "help", Description: "List the commands"},
	{Name: "exit", Description: "Leave the session"},
	{Name: "quit", Description: "Leave the session"},
	{Name: "edit", Description: "Write the prompt in your editor", Hint: "[text]"},
	{Name: "new", Description: "Start a new session"},
}

// compose is editor.Compose; tests replace it.
var compose = editor.Compose

// registry is the slash names the line session completes and accepts:
// the local commands and, once a session exists, the agent's, which by
// 0002-MADR's rule are the commands it executes (D19 item 9).
func (ls *lineSession) registry() []string {
	names := make([]string, 0, len(localCommands))
	for _, c := range localCommands {
		names = append(names, c.Name)
	}
	if ls.sess != nil {
		for _, c := range ls.sess.Commands() {
			if !slices.Contains(names, c.Name) {
				names = append(names, c.Name)
			}
		}
	}
	slices.Sort(names)
	return names
}

// ensureSession opens the session when there is none yet. session/new
// makes no model call, so a slash command may open it (D5).
func (ls *lineSession) ensureSession() error {
	if ls.sess != nil {
		return nil
	}
	s, _, err := openSession(ls.e.ctx, ls.conn, ls.choice, ls.cwd, ls.base, ls.onUpdate)
	if err != nil {
		return err
	}
	ls.sess = s
	return nil
}

// slash handles a submission that is a command the line session owns, or
// one no one owns, which is an error printed here and never sent (D13).
// handled is false for prompt text and for the agent's commands, which go
// to the agent as prompts. outOfRaw runs f with the terminal out of raw
// mode, for the external editor.
func (ls *lineSession) slash(text string, turn func(Prompt) error, outOfRaw func(func() error) error) (handled bool, err error) {
	name, args, ok := command.Parse(text)
	if !ok {
		return false, nil
	}
	e := ls.e
	switch name {
	case "help":
		if err := ls.ensureSession(); err != nil {
			e.out.Errorf("%v", err)
		}
		return true, e.out.Result([]byte(ls.helpText()))
	case "new":
		ls.closeSession()
		ls.choice = sessionChoice{}
		return true, e.out.Resultf("%s\n", dimmed("new session", e.out.caps.Out.Color))
	case "edit":
		var composed string
		err := outOfRaw(func() error {
			var cerr error
			composed, cerr = compose(e.ctx, os.Getenv, ls.stateDir(), args, ls.runEditor)
			return cerr
		})
		if err != nil {
			e.out.Errorf("%v", err)
			return true, nil
		}
		if strings.TrimSpace(composed) == "" {
			return true, nil
		}
		return true, turn(Prompt{Text: composed})
	}
	if err := ls.ensureSession(); err != nil {
		e.out.Errorf("%v", err)
		return true, nil
	}
	if !slices.Contains(ls.registry(), name) {
		e.out.Errorf("unknown command /%s; /help lists the commands", name)
		return true, nil
	}
	return false, nil
}

// helpText lists the local commands, then the agent's.
func (ls *lineSession) helpText() string {
	var b strings.Builder
	line := func(c acpclient.Command) {
		name := "/" + c.Name
		if c.Hint != "" {
			name += " " + c.Hint
		}
		fmt.Fprintf(&b, "  %s — %s\n", name, c.Description)
	}
	b.WriteString("Commands:\n")
	for _, c := range localCommands {
		line(c)
	}
	if ls.sess != nil {
		var agent []acpclient.Command
		for _, c := range ls.sess.Commands() {
			if !slices.ContainsFunc(localCommands, func(l acpclient.Command) bool { return l.Name == c.Name }) {
				agent = append(agent, c)
			}
		}
		if len(agent) > 0 {
			b.WriteString("Agent commands:\n")
			for _, c := range agent {
				line(c)
			}
		}
	}
	return b.String()
}

// closeSession ends the open session, if any.
func (ls *lineSession) closeSession() {
	if ls.sess == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ls.e.ctx), closeWait)
	defer cancel()
	if err := ls.sess.Close(ctx); err != nil {
		ls.e.out.Warnf("%v", err)
	}
	ls.sess = nil
}

// switchTo continues in the session the agent wrote, as Pi's terminal
// does after /fork and /clone (0002-PLAN Phase 6, owner's decision of
// 2026-10-06).
func (ls *lineSession) switchTo(id string) error {
	ls.closeSession()
	s, r, err := ls.conn.ResumeSession(ls.e.ctx, id, ls.cwd, ls.onUpdate)
	if err != nil {
		ls.e.out.Errorf("resume %s: %v", id, err)
		return nil
	}
	ls.sess = s
	ls.choice = sessionChoice{}
	return ls.e.out.Resultf("%s\n", dimmed(resumedLine(r), ls.e.out.caps.Out.Color))
}

// stateDir is where /edit's temp file goes: the state directory, else the
// system's temp directory.
func (*lineSession) stateDir() string {
	if dirs, _, err := systemDirs(); err == nil {
		return dirs.State
	}
	return os.TempDir()
}

// runEditor runs the external editor on the process's terminal.
func (ls *lineSession) runEditor(c *exec.Cmd) error {
	c.Stdin, c.Stdout, c.Stderr = ls.e.stdin, ls.e.out.out, ls.e.out.err
	return c.Run()
}

// complete is Tab in the line editor (D6): a slash name from the
// registry, or an @path from the directory it names.
func (ls *lineSession) complete(line string, pos int) (string, int, bool) {
	head, tail := line[:pos], line[pos:]
	if strings.HasPrefix(head, "/") && !strings.ContainsAny(head, " \t") {
		var names []string
		for _, n := range ls.registry() {
			if strings.HasPrefix(n, head[1:]) {
				names = append(names, n)
			}
		}
		return extend(head, "/", names, tail)
	}
	start := strings.LastIndexAny(head, " \t") + 1
	word := head[start:]
	if !strings.HasPrefix(word, "@") {
		return line, pos, false
	}
	partial := word[1:]
	dir, prefix := filepath.Split(filepath.FromSlash(partial))
	root := dir
	if !filepath.IsAbs(root) {
		root = filepath.Join(ls.base, dir)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return line, pos, false
	}
	var names []string
	for _, en := range entries {
		if strings.HasPrefix(en.Name(), prefix) {
			n := en.Name()
			if en.IsDir() {
				n += "/"
			}
			names = append(names, n)
		}
	}
	return extend(head[:start]+"@"+filepath.ToSlash(dir), "", names, tail)
}

// extend completes head with the names' common prefix; a single name that
// is not a directory gets a trailing space.
func extend(head, sep string, names []string, tail string) (string, int, bool) {
	if len(names) == 0 {
		return head + tail, len(head), false
	}
	common := names[0]
	for _, n := range names[1:] {
		for !strings.HasPrefix(n, common) {
			common = common[:len(common)-1]
		}
	}
	if len(names) == 1 && !strings.HasSuffix(common, "/") {
		common += " "
	}
	if sep == "/" {
		head = "/"
	}
	out := head + common
	return out + tail, len(out), true
}
