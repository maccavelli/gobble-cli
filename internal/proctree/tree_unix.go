//go:build unix

package proctree

import (
	"os"
	"os/exec"
	"syscall"
)

// Prepare makes cmd start in its own process group, so a signal to the
// group reaches what it starts, such as the children of npx or of a shell.
// Call it before Start.
func Prepare(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// Tree is a started command's process group.
type Tree struct {
	proc *os.Process
}

// Attach is the tree of a process started after Prepare.
func Attach(p *os.Process) (*Tree, error) { return &Tree{proc: p}, nil }

// Terminate asks the tree to stop (SIGTERM).
func (t *Tree) Terminate() { t.signal(syscall.SIGTERM) }

// Kill stops the tree at once (SIGKILL).
func (t *Tree) Kill() { t.signal(syscall.SIGKILL) }

// Exited is SIGTERM to the group once its leader has exited, for children
// that outlive it, as Pi's MCP client does.
func (t *Tree) Exited() { t.signal(syscall.SIGTERM) }

// Release frees what Attach holds. A process group holds nothing.
func (*Tree) Release() {}

// signal signals the group, or, when there is no group, the process itself,
// as Pi falls back to the direct child (pkg-stdio.ts killProcessTree).
func (t *Tree) signal(sig syscall.Signal) {
	if syscall.Kill(-t.proc.Pid, sig) != nil {
		_ = t.proc.Signal(sig) //nolint:errcheck // a process already gone is the goal
	}
}
