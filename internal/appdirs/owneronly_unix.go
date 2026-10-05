//go:build unix

package appdirs

import "os"

// FileIsOwnerOnly reports whether path is readable only by its owner.
//
// On Unix that is the POSIX mode test Perm()&0o077 == 0. The Windows
// implementation asks the same question of the file's owner SID and DACL,
// because mode bits carry no access control there.
//
// Provenance: magic-cli-remote internal/appdirs/owneronly_unix.go at 9778cbc1
// (Apache-2.0), unchanged apart from this comment.
func FileIsOwnerOnly(path string) (bool, error) {
	path, err := absClean(path, "path")
	if err != nil {
		return false, err
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return fi.Mode().Perm()&0o077 == 0, nil
}
