package fsx

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// goosWindows is runtime.GOOS on Windows.
const goosWindows = "windows"

// unicodeSpaces are the space variants a model or a terminal paste puts in
// a path where a plain space belongs (Pi utils/paths.ts UNICODE_SPACES).
var unicodeSpaces = regexp.MustCompile(`[\x{00A0}\x{2000}-\x{200A}\x{202F}\x{205F}\x{3000}]`)

// windowsShellPath matches Git Bash, MSYS, Cygwin and WSL drive paths:
// /c/x, /mnt/c/x and /cygdrive/c/x.
var windowsShellPath = regexp.MustCompile(`(?i)^/(?:mnt/|cygdrive/)?([a-z])(?:/(.*))?$`)

// Resolve makes a path a model gave absolute, in Pi's order (utils/paths.ts
// resolvePath with normalizeUnicodeSpaces and stripAtPrefix): Unicode spaces
// become a space, a leading @ goes, a Windows shell drive path becomes a
// native one, ~ is the home directory, a file:// URL is its path, and a
// relative path is joined to cwd. The result is cleaned.
func Resolve(cwd, path string) string {
	return resolveFor(runtime.GOOS, cwd, path)
}

// resolveFor is Resolve with the rules of goos.
func resolveFor(goos, cwd, path string) string {
	p := normalize(goos, path, true)
	base := normalize(goos, cwd, false)
	if isAbs(goos, p) {
		return clean(goos, p, base)
	}
	return clean(goos, join(goos, base, p), base)
}

// normalize is Pi's normalizePath. forTool turns on the two steps Pi's
// tools add for a model's path: Unicode spaces and the @ prefix.
func normalize(goos, path string, forTool bool) string {
	if forTool {
		path = unicodeSpaces.ReplaceAllString(path, " ")
		path = strings.TrimPrefix(path, "@")
	}
	if goos == goosWindows {
		path = windowsShell(path)
	}
	if home, err := os.UserHomeDir(); err == nil {
		switch {
		case path == "~":
			return home
		case strings.HasPrefix(path, "~/"), goos == goosWindows && strings.HasPrefix(path, `~\`):
			return join(goos, home, path[2:])
		}
	}
	if strings.HasPrefix(path, "file://") {
		if p, ok := fromFileURL(goos, path); ok {
			return p
		}
	}
	return path
}

// windowsShell is Pi's normalizeWindowsShellPath: /c/x is C:\x.
func windowsShell(path string) string {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.Contains(path, `\`) {
		return path
	}
	m := windowsShellPath.FindStringSubmatch(path)
	if m == nil {
		return path
	}
	return strings.ToUpper(m[1]) + `:\` + strings.ReplaceAll(m[2], "/", `\`)
}

// fromFileURL is a file:// URL's path, as Node's fileURLToPath gives it.
func fromFileURL(goos, raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "file" {
		return "", false
	}
	p := u.Path
	if goos == goosWindows {
		if u.Host != "" && u.Host != "localhost" {
			return `\\` + u.Host + strings.ReplaceAll(p, "/", `\`), true
		}
		p = strings.TrimPrefix(p, "/")
		return strings.ReplaceAll(p, "/", `\`), true
	}
	return p, true
}

// The path functions below take goos so the Windows rules can be tested on
// any host. On the host's own OS they are filepath's.

func isAbs(goos, p string) bool {
	if goos != goosWindows {
		return strings.HasPrefix(p, "/")
	}
	if goos == runtime.GOOS {
		// A rooted path without a drive, \x, is absolute to Node too: it is
		// on the base directory's drive, which clean supplies.
		return filepath.IsAbs(p) || strings.HasPrefix(p, `\`) || strings.HasPrefix(p, "/")
	}
	return len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/') ||
		strings.HasPrefix(p, `\`) || strings.HasPrefix(p, "/")
}

func join(goos, base, p string) string {
	if goos == runtime.GOOS {
		return filepath.Join(base, p)
	}
	sep := "/"
	if goos == goosWindows {
		sep = `\`
	}
	return strings.TrimSuffix(base, sep) + sep + p
}

func clean(goos, p, base string) string {
	if goos != runtime.GOOS {
		return p
	}
	if goos == goosWindows && filepath.VolumeName(p) == "" && filepath.VolumeName(base) != "" {
		p = filepath.VolumeName(base) + p
	}
	return filepath.Clean(p)
}

// ReadVariants are the other spellings Pi tries, in order, when a path to
// read does not exist (path-utils.ts resolveReadPath): the narrow no-break
// space macOS puts before AM and PM in screenshot names, NFD, U+2019 for an
// apostrophe, and NFD with U+2019.
func ReadVariants(path string) []string {
	amPM := amPMSpace.ReplaceAllString(path, "\u202F$1.")
	nfd := norm.NFD.String(path)
	curly := strings.ReplaceAll(path, "'", "\u2019")
	nfdCurly := strings.ReplaceAll(nfd, "'", "\u2019")
	var out []string
	for _, v := range []string{amPM, nfd, curly, nfdCurly} {
		if v != path {
			out = append(out, v)
		}
	}
	return out
}

var amPMSpace = regexp.MustCompile(`(?i) (AM|PM)\.`)
