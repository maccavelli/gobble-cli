package builtin

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/tool"
)

const (
	// maxLines and maxBytes bound what a read or a command's output gives
	// the model (0005-MADR, Tools).
	maxLines = 2000
	maxBytes = 50 * 1024
	// maxTitle is a title's length in runes, and maxSummary a summary's in
	// bytes (0008-MADR D19 item 3).
	maxTitle   = 60
	maxSummary = 400
)

// Options configure the built-in tools. Until gobble has a settings file
// they are set in Go by an embedder or by gobble's CLI (0005-PLAN F1,
// owner's decision 2 of 2026-10-07).
type Options struct {
	// ShellPath is the bash to run. Empty finds one: on Windows Git's bash,
	// elsewhere bash on PATH.
	ShellPath string
	// ShellCommandPrefix runs before every bash command, on a line of its
	// own, such as "set -euo pipefail" or a source of an environment file.
	ShellCommandPrefix string
	// MaxTimeout caps a command's timeout. Zero is 10 minutes.
	MaxTimeout time.Duration
	// OutputDir holds the whole output of commands too long to show, as
	// gobble-bash-<id>.log files kept 7 days. read may open files there
	// without asking. Empty is gobble-output in the OS temp directory.
	OutputDir string
}

// goosWindows is runtime.GOOS on Windows.
const goosWindows = "windows"

// defaultMaxTimeout is MaxTimeout's zero value.
const defaultMaxTimeout = 10 * time.Minute

func (o Options) withDefaults() Options {
	if o.MaxTimeout <= 0 {
		o.MaxTimeout = defaultMaxTimeout
	}
	if o.OutputDir == "" {
		o.OutputDir = filepath.Join(os.TempDir(), "gobble-output")
	}
	return o
}

// ToolsWith returns the built-in tools configured by o, in 0012-MADR D7's
// order: read, write, edit, grep, find, tree and bash, and on Windows
// powershell.
func ToolsWith(o Options) []tool.Tool {
	o = o.withDefaults()
	ts := []tool.Tool{readWith(o), Write(), Edit(), Grep(), Find(), Tree(), bashWith(o)}
	if runtime.GOOS == goosWindows {
		ts = append(ts, powershellWith(o))
	}
	return ts
}

// Tools is ToolsWith the default options.
func Tools() []tool.Tool {
	return ToolsWith(Options{})
}

// Names are the built-in tools' names, in Tools' order.
func Names() []string {
	ts := Tools()
	names := make([]string, len(ts))
	for i, t := range ts {
		names[i] = t.Spec().Name
	}
	return names
}

// title is s on one line, cut to maxTitle runes.
func title(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= maxTitle {
		return s
	}
	r := []rune(s)
	return string(r[:maxTitle-1]) + "…"
}

// summary is s cut to maxSummary bytes on a rune boundary.
func summary(s string) string {
	if len(s) <= maxSummary {
		return s
	}
	cut := maxSummary - len("…")
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}

// count is n with its noun: "1 line", "3 lines".
func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimSuffix(s, "\n"), "\n") + 1
}
