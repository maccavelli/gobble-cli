package acptest

import (
	"cmp"
	"os"
	"path/filepath"
	"runtime"
)

// Abs is a POSIX fixture path, such as "/work", as an absolute path on
// the host: itself off Windows, and on Windows under the system drive,
// "C:\work". acp-go-sdk2 checks a path it receives with the host's rule
// from v0.16.0 (0019-MADR D4).
func Abs(p string) string {
	if runtime.GOOS != "windows" {
		return p
	}
	return cmp.Or(os.Getenv("SystemDrive"), "C:") + filepath.FromSlash(p)
}
