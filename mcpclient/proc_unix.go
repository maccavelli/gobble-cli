//go:build unix

package mcpclient

import (
	"os"
	"os/exec"
	"syscall"
)

// prepare starts the server in its own process group, so a signal to the
// group reaches what a wrapper such as npx or uvx starts.
func prepare(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// tree is the server's process group.
type tree struct {
	proc *os.Process
}

func attach(p *os.Process) (*tree, error) { return &tree{proc: p}, nil }

func (t *tree) terminate() { t.signal(syscall.SIGTERM) }
func (t *tree) kill()      { t.signal(syscall.SIGKILL) }

// exited is Pi's SIGTERM to the group once the server has exited, for
// children that outlive it.
func (t *tree) exited() { t.signal(syscall.SIGTERM) }

func (*tree) release() {}

// signal signals the group, or, when there is no group, the server itself,
// as Pi falls back to the direct child (pkg-stdio.ts killProcessTree).
func (t *tree) signal(sig syscall.Signal) {
	if syscall.Kill(-t.proc.Pid, sig) != nil {
		_ = t.proc.Signal(sig) //nolint:errcheck // a server already gone is the goal
	}
}
