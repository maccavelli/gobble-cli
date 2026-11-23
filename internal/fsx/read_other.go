//go:build !windows

package fsx

// sharingViolation is false: off Windows an open is never refused because
// another process holds the file (0067-MADR D1).
func sharingViolation(error) bool { return false }
