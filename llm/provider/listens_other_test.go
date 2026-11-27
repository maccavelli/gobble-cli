//go:build !linux && !darwin && !windows

package provider

import (
	"fmt"
	"runtime"
)

// ownListeningPorts has no lookup on this operating system: the tests that
// ask it fail rather than pass unseen.
func ownListeningPorts() (map[int]bool, error) {
	return nil, fmt.Errorf("no listener lookup on %s", runtime.GOOS)
}
