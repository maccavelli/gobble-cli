//go:build windows

package appdirs

import (
	"fmt"
	"os"
)

// EnsurePrivateDir creates dir (and parents) as a user-owned private directory.
//
// Idempotent and converging. Creates the directory and any parents if absent;
// if present, verifies owner and access and repairs access to the private
// state; returns an error rather than repairing when the leaf is a symlink,
// reparse point, or not owned by the current principal. A second call on a
// converged directory performs no writes.
//
// "Private" on Windows is an explicit DACL granting the current user and
// SYSTEM full control with inheritance severed — the analogue of mode 0700.
// The 0700 passed to MkdirAll is ignored by the platform and is kept only so
// the two implementations read alike.
//
// Provenance: magic-cli-remote internal/appdirs/ensure_windows.go at 9778cbc1
// (Apache-2.0); checkPrivateDir always repairs here, because gobble has no
// read-only validator.
func EnsurePrivateDir(dir string) error {
	dir, err := absClean(dir, "directory")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return checkPrivateDir(dir)
}
