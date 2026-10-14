package auth

import "github.com/maccavelli/gobble-cli/internal/appdirs"

// securePrivate gives path the user's DACL alone, before its first byte is
// written; a Windows file's mode says nothing of who may read it.
func securePrivate(path string) error { return appdirs.SecurePrivateFile(path) }
