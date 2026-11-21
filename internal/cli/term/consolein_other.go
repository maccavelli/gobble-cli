//go:build !windows

package term

import "os"

// isConsoleInput is false outside Windows: D19's rule is a Windows
// console's, and standard input is read unchanged (0063-MADR D19, row R118).
func isConsoleInput(*os.File) bool { return false }
