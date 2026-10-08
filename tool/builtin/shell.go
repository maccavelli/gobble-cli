package builtin

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/internal/proctree"
	"github.com/maccavelli/gobble-cli/tool"
)

const (
	// defaultTimeout is a command's timeout when the model gives none
	// (0005-PLAN F1b, owner's decision 2 of 2026-10-07).
	defaultTimeout = 2 * time.Minute
	// killGrace is how long a stopped command has between SIGTERM and
	// SIGKILL (owner's decision 3).
	killGrace = 3 * time.Second
	// waitDelay is how long a stopped command's output pipes stay open, so
	// a grandchild holding them cannot hold the turn.
	waitDelay = 2 * time.Second
)

// shellIn is what bash and powershell take.
type shellIn struct {
	Command string `json:"command" jsonschema:"the command to run, in this shell's own syntax"`
	Timeout int    `json:"timeout,omitzero" jsonschema:"seconds before the command, and everything it started, is stopped"`
	Workdir string `json:"workdir,omitzero" jsonschema:"the directory to run in, relative to the working directory or absolute; use this instead of cd"`
}

// markerKeys are the 0003-MADR session markers. A command's environment
// has them for its own session, never ones inherited from a gobble that
// started this one.
var markerKeys = []string{"AI_AGENT", "GOBBLE_SESSION_ID", "GOBBLE_SESSION_FILE", "GOBBLE_PROVIDER", "GOBBLE_MODEL", "GOBBLE_THINKING_LEVEL"}

// shell is one way to run a command: the program and its arguments.
type shell struct {
	name string                        // the tool's name, for errors
	find func() (string, error)        // the program
	args func(command string) []string // its arguments
}

// shellSchema publishes a non-empty command and the timeout's bounds and
// default, the maximum from o (0012-MADR D2 rules 6 and 8).
func shellSchema(o Options) tool.Option {
	return tool.WithSchema(func(s *jsonschema.Schema) {
		nonEmpty(s, "command")
		bound(s, "timeout", 1, int(o.MaxTimeout/time.Second), int(defaultTimeout/time.Second))
	})
}

// runShell runs in.Command with sh as a process tree, in the working
// directory or in.Workdir, with the call's markers in its environment. It
// stops the tree on cancel and on the timeout: SIGTERM, then SIGKILL after
// killGrace. A cancel is ctx's error, which the agent reports as such.
func runShell(ctx context.Context, sh shell, o Options, in shellIn, env tool.Env) (tool.Result, error) {
	if strings.TrimSpace(in.Command) == "" {
		return tool.Result{}, fmt.Errorf("%s: the command is empty; send the command to run", sh.name)
	}
	program, err := sh.find()
	if err != nil {
		return tool.Result{}, err
	}
	dir, err := workdir(env, sh.name, in.Workdir)
	if err != nil {
		return tool.Result{}, err
	}
	timeout := commandTimeout(in.Timeout, o.MaxTimeout)

	out := &spill{dir: o.OutputDir}
	// Not CommandContext: on cancel that kills only the leader, and the
	// runner stops the whole tree itself (stop).
	cmd := exec.Command(program, sh.args(in.Command)...) //nolint:gosec,noctx // G204: the model's command, behind the session's approval; noctx: as above
	cmd.Dir = dir
	cmd.Env = childEnv(os.Environ(), env.Environ)
	cmd.Stdout, cmd.Stderr = out, out
	cmd.WaitDelay = waitDelay
	proctree.Prepare(cmd)
	if err := cmd.Start(); err != nil {
		return tool.Result{}, fmt.Errorf("%s: %w", sh.name, err)
	}
	tree, err := proctree.Attach(cmd.Process)
	if err != nil {
		_ = cmd.Process.Kill() //nolint:errcheck // the call fails with err
		_ = cmd.Wait()         //nolint:errcheck // as above
		return tool.Result{}, fmt.Errorf("%s: %w", sh.name, err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var runErr error
	stopped := false
	select {
	case runErr = <-done:
	case <-ctx.Done():
		runErr, stopped = stop(tree, done), true
	case <-timer.C:
		runErr, stopped = stop(tree, done), true
	}
	// What the command left running ends with it: background commands are
	// 0005-PLAN X2's.
	tree.Exited()
	tree.Release()
	out.close()
	if stopped && ctx.Err() != nil {
		return tool.Result{}, context.Cause(ctx)
	}
	return shellResult(sh.name, out, runErr, stopped, timeout, o.MaxTimeout)
}

// commandTimeout is the timeout the model asked for, in seconds, or
// defaultTimeout when it asked for none, and never more than maxTimeout.
func commandTimeout(seconds int, maxTimeout time.Duration) time.Duration {
	t := defaultTimeout
	if seconds > 0 {
		t = time.Duration(seconds) * time.Second
	}
	return min(t, maxTimeout)
}

// stop ends a running tree: SIGTERM, then SIGKILL if it has not exited
// within killGrace. It returns Wait's error.
func stop(tree *proctree.Tree, done <-chan error) error {
	tree.Terminate()
	grace := time.NewTimer(killGrace)
	defer grace.Stop()
	select {
	case err := <-done:
		return err
	case <-grace.C:
		tree.Kill()
		return <-done
	}
}

// shellResult is what the model reads: the output's tail, a notice naming
// the whole output's file when it was cut, and a status line.
func shellResult(name string, out *spill, runErr error, stopped bool, timeout, maxTimeout time.Duration) (tool.Result, error) {
	status := 0
	if exit, ok := errors.AsType[*exec.ExitError](runErr); ok {
		status = exit.ExitCode()
	} else if runErr != nil && !stopped && !errors.Is(runErr, exec.ErrWaitDelay) {
		return tool.Result{}, fmt.Errorf("%s: %w", name, runErr)
	}
	// Windows programs end lines with CRLF; the model reads them as LF. The
	// saved file keeps the bytes as written.
	t := truncateTail(strings.ReplaceAll(string(out.mem), "\r\n", "\n"), maxLines, maxBytes)
	var b strings.Builder
	if t.truncated() || out.file != nil {
		whole := "the whole output could not be saved: " + errText(out.err)
		if out.err == nil && out.path != "" {
			whole = "the whole output is in " + out.path
		} else if out.err == nil {
			whole = "the whole output was not saved"
		}
		fmt.Fprintf(&b, "(output cut: the last %s of %d are shown; %s)\n", count(t.outputLines, "line", "lines"), out.lines(), whole)
	}
	b.WriteString(strings.TrimSuffix(t.content, "\n"))
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	if stopped {
		fmt.Fprintf(&b, "[stopped after %d s; give a larger timeout, at most %d]", int(timeout.Seconds()), int(maxTimeout.Seconds()))
		r := tool.ErrorResult(b.String())
		r.Summary = summary(fmt.Sprintf("stopped after %d s", int(timeout.Seconds())))
		return r, nil
	}
	fmt.Fprintf(&b, "[exit status %d]", status)
	r := tool.TextResult(b.String())
	r.IsError = status != 0
	first, _, _ := strings.Cut(strings.TrimSpace(t.content), "\n")
	r.Summary = summary(strings.TrimSpace(fmt.Sprintf("exit status %d: %s", status, first)))
	return r, nil
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// workdir is the directory a command runs in: the working directory, or the
// model's workdir resolved against it, which must be a directory.
func workdir(env tool.Env, name, wd string) (string, error) {
	if wd == "" {
		return env.Cwd, nil
	}
	abs := resolve(env, wd)
	info, err := workspace(env).Stat(abs)
	if err != nil {
		return "", fmt.Errorf("%s: workdir %s: %w", name, shown(env, wd), unwrapPath(err))
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: workdir %s is not a directory", name, shown(env, wd))
	}
	return abs, nil
}

// childEnv is base without inherited session markers, then extra. On
// Windows names compare without regard to case, as the environment does.
func childEnv(base, extra []string) []string {
	out := slices.DeleteFunc(slices.Clone(base), func(kv string) bool {
		k, _, _ := strings.Cut(kv, "=")
		return slices.ContainsFunc(markerKeys, func(m string) bool { return sameKey(k, m) })
	})
	return append(out, extra...)
}

func sameKey(a, b string) bool {
	if runtime.GOOS == goosWindows {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// shellDescribe titles a call "run <command>".
func shellDescribe(c tool.Call, _ tool.Env) string {
	var a shellIn
	if err := json.Unmarshal(c.Args, &a); err != nil || a.Command == "" {
		return "run ?"
	}
	return title("run " + a.Command)
}

// shellOutside is the Confined answer of a shell tool: its workdir, when it
// is outside the roots.
func shellOutside(c tool.Call, env tool.Env) []string {
	var a shellIn
	if err := json.Unmarshal(c.Args, &a); err != nil {
		return nil
	}
	return outsideOf(workspace(env), env, a.Workdir)
}
