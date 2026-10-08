//go:build windows

package proctree

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Prepare needs nothing on Windows: the tree is a Job object, joined after
// Start.
func Prepare(*exec.Cmd) {}

// Tree is the Job object a command runs in. It kills on close, so the tree
// also dies with gobble.
type Tree struct {
	job  windows.Handle
	once sync.Once
}

// Attach puts a started process in a new kill-on-close Job object. A child
// the process starts before this runs escapes the job (0002-PLAN Phase 8,
// owner's decision 2).
func Attach(p *os.Process) (*Tree, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("proctree: create a job object: %w", err)
	}
	// On the heap, and kept alive across the call: SetInformationJobObject
	// is a Go wrapper taking a uintptr, so a stack copy made before it
	// reaches the syscall would leave the address stale.
	info := &windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE},
	}
	_, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(info)), uint32(unsafe.Sizeof(*info))) //nolint:gosec // G103: the Win32 call takes the struct's address
	runtime.KeepAlive(info)
	if err != nil {
		_ = windows.CloseHandle(job) //nolint:errcheck // the attach fails with err
		return nil, fmt.Errorf("proctree: set the job object's limits: %w", err)
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid)) //nolint:gosec // G115: a pid fits in uint32
	if err != nil {
		_ = windows.CloseHandle(job) //nolint:errcheck // as above
		return nil, fmt.Errorf("proctree: open the process: %w", err)
	}
	defer windows.CloseHandle(h) //nolint:errcheck // the job holds the process now
	if err := windows.AssignProcessToJobObject(job, h); err != nil {
		_ = windows.CloseHandle(job) //nolint:errcheck // as above
		return nil, fmt.Errorf("proctree: assign the process to its job object: %w", err)
	}
	return &Tree{job: job}, nil
}

// Terminate ends the whole tree: Windows has no graceful signal for one.
func (t *Tree) Terminate() {
	_ = windows.TerminateJobObject(t.job, 1) //nolint:errcheck // a job already empty is the goal
}

// Kill ends the whole tree.
func (t *Tree) Kill() { t.Terminate() }

// Exited closes the job, which kills what the leader left running.
func (t *Tree) Exited() { t.Release() }

// Release closes the job.
func (t *Tree) Release() {
	t.once.Do(func() {
		_ = windows.CloseHandle(t.job) //nolint:errcheck // nothing to do on failure
	})
}
