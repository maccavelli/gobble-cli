//go:build linux

package term

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// terminalPath is the path of the terminal open on fd: its link under
// /proc/self/fd (0063-MADR D22, Q25).
func terminalPath(fd int) (string, error) {
	p, err := os.Readlink("/proc/self/fd/" + strconv.Itoa(fd))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(p, "/dev/") {
		return "", fmt.Errorf("descriptor %d is %s, not a terminal's path", fd, p)
	}
	return p, nil
}
