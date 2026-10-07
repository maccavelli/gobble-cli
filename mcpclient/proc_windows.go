//go:build windows

package mcpclient

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// prepare needs nothing on Windows: the tree is a Job object, joined after
// Start.
func prepare(*exec.Cmd) {}

// tree is the Job object the server runs in. It kills on close, so the tree
// also dies with gobble.
type tree struct {
	job  windows.Handle
	once sync.Once
}

// attach puts the server in a new kill-on-close Job object. A child the
// server starts before this runs escapes the job (0002-PLAN Phase 8,
// owner's decision 2).
func attach(p *os.Process) (*tree, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("mcpclient: create a job object: %w", err)
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
		return nil, fmt.Errorf("mcpclient: set the job object's limits: %w", err)
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid)) //nolint:gosec // G115: a pid fits in uint32
	if err != nil {
		_ = windows.CloseHandle(job) //nolint:errcheck // as above
		return nil, fmt.Errorf("mcpclient: open the server process: %w", err)
	}
	defer windows.CloseHandle(h) //nolint:errcheck // the job holds the process now
	if err := windows.AssignProcessToJobObject(job, h); err != nil {
		_ = windows.CloseHandle(job) //nolint:errcheck // as above
		return nil, fmt.Errorf("mcpclient: assign the server to its job object: %w", err)
	}
	return &tree{job: job}, nil
}

// terminate ends the whole tree: Windows has no graceful signal.
func (t *tree) terminate() {
	_ = windows.TerminateJobObject(t.job, 1) //nolint:errcheck // a job already empty is the goal
}

func (t *tree) kill() { t.terminate() }

// exited closes the job, which kills what the server left running.
func (t *tree) exited() { t.release() }

func (t *tree) release() {
	t.once.Do(func() {
		_ = windows.CloseHandle(t.job) //nolint:errcheck // nothing to do on failure
	})
}
