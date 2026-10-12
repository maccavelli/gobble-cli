//go:build !unix && !windows

package filelock

import "os"

// lockFile always succeeds where the platform has no advisory lock.
func lockFile(*os.File) (bool, error) { return true, nil }

func unlockFile(*os.File) error { return nil }
