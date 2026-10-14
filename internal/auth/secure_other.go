//go:build !windows

package auth

// securePrivate needs nothing here: os.CreateTemp opened the file 0600.
func securePrivate(string) error { return nil }
