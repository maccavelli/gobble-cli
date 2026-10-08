//go:build !windows

package fsx

// transient is false: a Unix rename replaces a file whoever holds it open.
func transient(error) bool { return false }
