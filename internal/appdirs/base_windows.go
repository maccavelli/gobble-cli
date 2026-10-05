//go:build windows

package appdirs

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// systemBase reads the RoamingAppData and LocalAppData Known Folders:
// config roams with the user; data, state and cache stay on this machine.
// Provenance: magic-cli-remote internal/appdirs/roots_windows.go at 9778cbc1
// (Apache-2.0), reduced to the two folders gobble's table uses.
func systemBase() (Base, error) {
	roaming, err := windows.KnownFolderPath(windows.FOLDERID_RoamingAppData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return Base{}, fmt.Errorf("appdirs: known folder RoamingAppData: %w", err)
	}
	local, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return Base{}, fmt.Errorf("appdirs: known folder LocalAppData: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Base{}, fmt.Errorf("appdirs: home directory: %w", err)
	}
	return Base{Home: home, AppData: roaming, LocalAppData: local}, nil
}
