//go:build !windows

package appdirs

import (
	"fmt"
	"os"
)

func systemBase() (Base, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Base{}, fmt.Errorf("appdirs: home directory: %w", err)
	}
	return Base{Home: home}, nil
}
