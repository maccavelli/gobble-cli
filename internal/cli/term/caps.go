package term

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	xterm "golang.org/x/term"
)

// defaultWidth is used when neither the terminal nor COLUMNS gives a width.
const defaultWidth = 80

// Stream is what one output stream supports.
type Stream struct {
	TTY   bool
	Color bool
	Width int
}

// Caps is what the process's standard streams support.
type Caps struct {
	Out, Err Stream
	In       bool
	Unicode  bool
}

// Detect reports the capabilities of in, out and err. env is the environment
// lookup (os.Getenv in production), injected so tests need no process
// environment. A nil file is not a terminal.
func Detect(env func(string) string, in, out, err *os.File) Caps {
	return Caps{
		Out:     detectStream(env, out),
		Err:     detectStream(env, err),
		In:      isTerminal(in),
		Unicode: unicodeOK(env, runtime.GOOS),
	}
}

func isTerminal(f *os.File) bool {
	return f != nil && xterm.IsTerminal(int(f.Fd())) //nolint:gosec // a console handle or fd fits in int
}

func detectStream(env func(string) string, f *os.File) Stream {
	tty := isTerminal(f)
	s := Stream{TTY: tty, Color: colorOK(env, tty), Width: defaultWidth}
	if tty {
		if w, _, err := xterm.GetSize(int(f.Fd())); err == nil && w > 0 { //nolint:gosec // as above
			s.Width = w
			return s
		}
	}
	if w, err := strconv.Atoi(env("COLUMNS")); err == nil && w > 0 {
		s.Width = w
	}
	return s
}

// colorOK follows NO_COLOR (no-color.org) and CLICOLOR_FORCE.
func colorOK(env func(string) string, tty bool) bool {
	if env("CLICOLOR_FORCE") == "1" {
		return true
	}
	return tty && env("NO_COLOR") == "" && env("TERM") != "dumb"
}

// unicodeOK reports whether glyphs outside ASCII may be drawn. Windows
// consoles render UTF-8 once the code page is set; elsewhere a locale
// variable must name UTF-8.
func unicodeOK(env func(string) string, goos string) bool {
	if env("TERM") == "dumb" {
		return false
	}
	if goos == "windows" {
		return true
	}
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := strings.ToLower(env(name))
		if strings.Contains(v, "utf-8") || strings.Contains(v, "utf8") {
			return true
		}
	}
	return false
}
