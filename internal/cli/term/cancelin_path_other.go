//go:build unix && !linux && !darwin

package term

import "errors"

// terminalPath has no way here to read a terminal's path from its
// descriptor; OpenGated then tries /dev/tty (0063-MADR D22, row R144).
func terminalPath(int) (string, error) {
	return "", errors.New("this system gives no terminal's path from its descriptor")
}
