package appdirs

import (
	"fmt"
	"path/filepath"
)

// absClean validates and cleans a path argument shared by the platform
// security helpers.
//
// Provenance: magic-cli-remote internal/appdirs/pathcheck.go at 9778cbc1
// (Apache-2.0), unchanged.
func absClean(path, what string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("appdirs: empty %s", what)
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("appdirs: %s must be absolute: %q", what, path)
	}
	return filepath.Clean(path), nil
}
