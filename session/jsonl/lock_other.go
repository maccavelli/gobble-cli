//go:build !unix && !windows

package jsonl

import "os"

// tryLock always succeeds where the platform has no advisory lock.
func tryLock(*os.File) (bool, error) { return true, nil }

func unlock(*os.File) error { return nil }
