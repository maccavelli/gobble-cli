//go:build unix

package mcpclient

import "syscall"

// alive reports whether pid is a running process. A child reparented to
// init is reaped there, so a dead one is gone (ESRCH), not a zombie.
func alive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
