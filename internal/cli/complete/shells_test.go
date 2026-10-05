//go:build shells

package complete

// The real-shell suite (0008-PLAN P5, run by `make completion-shells`).
// Each host must have the shells the plan assigns it, and a missing one
// fails the suite, never skips it (0008-PLAN C5):
//
//	windows: Git Bash, pwsh, powershell.exe (5.1)
//	linux:   bash, fish
//	darwin:  zsh, bash (Homebrew or PATH) and /bin/bash (3.2)

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

type shellEnvT struct {
	bin, work string // the directory holding gobble, and the cwd with path fixtures
	scripts   map[string]string
}

func setupShells(t *testing.T) shellEnvT {
	t.Helper()
	e := shellEnvT{bin: filepath.Dir(gobbleBinary(t)), work: t.TempDir(), scripts: map[string]string{}}
	if err := os.WriteFile(filepath.Join(e.work, "dés.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(e.work, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, sh := range Shells {
		s, err := Script(sh, "gobble")
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(dir, "gobble."+sh)
		if sh == shellPowerShell {
			name += ".ps1"
			s = "\ufeff" + s // a BOM, so Windows PowerShell 5.1 reads UTF-8
		}
		if err := os.WriteFile(name, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
		e.scripts[sh] = name
	}
	return e
}

// run starts a shell with gobble first on PATH, in the fixture directory.
func (e shellEnvT) run(t *testing.T, shell string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), shell, args...)
	cmd.Dir = e.work
	cmd.Env = append(os.Environ(), "PATH="+e.bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	switch runtime.GOOS {
	case "linux":
		cmd.Env = append(cmd.Env, "LC_ALL=C.UTF-8") // always present on Ubuntu
	case "darwin":
		cmd.Env = append(cmd.Env, "LC_ALL=en_US.UTF-8")
	}
	var out, errw bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errw
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %q: %v\nstdout:\n%s\nstderr:\n%s", shell, args, err, out.String(), errw.String())
	}
	return strings.ReplaceAll(out.String(), "\r\n", "\n")
}

func need(t *testing.T, name string, candidates ...string) string {
	t.Helper()
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	t.Fatalf("%s is required on %s and was not found (0008-PLAN C5: a missing shell fails)", name, runtime.GOOS)
	return ""
}

func TestShells(t *testing.T) {
	e := setupShells(t)
	switch runtime.GOOS {
	case "windows":
		t.Run("bash", func(t *testing.T) { testBash(t, e, gitBash(t)) })
		t.Run("pwsh", func(t *testing.T) { testPowerShell(t, e, need(t, "pwsh", "pwsh")) })
		t.Run("powershell.exe", func(t *testing.T) { testPowerShell(t, e, need(t, "powershell.exe", "powershell.exe")) })
	case "linux":
		t.Run("bash", func(t *testing.T) { testBash(t, e, need(t, "bash", "bash")) })
		t.Run("fish", func(t *testing.T) { testFish(t, e, need(t, "fish", "fish")) })
	case "darwin":
		t.Run("zsh", func(t *testing.T) { testZsh(t, e, need(t, "zsh", "zsh")) })
		t.Run("bash", func(t *testing.T) { testBash(t, e, need(t, "bash", "bash")) })
		t.Run("bash 3.2", func(t *testing.T) { testBash(t, e, need(t, "/bin/bash", "/bin/bash")) })
	default:
		t.Fatalf("no shells are assigned to %s", runtime.GOOS)
	}
}

// gitBash is Git's bash. A bare "bash" on Windows may be WSL's launcher.
func gitBash(t *testing.T) string {
	t.Helper()
	for _, p := range []string{os.Getenv("GIT_BASH"), `C:\Program Files\Git\bin\bash.exe`, `C:\Program Files\Git\usr\bin\bash.exe`} {
		if p != "" {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	t.Fatal("Git Bash is required on windows and was not found")
	return ""
}

var wantFormats = []string{"text", "json", "stream-json"}

type bashCase struct {
	line  string
	words []string
	want  []string
	// point, when set, is the text before the cursor; COMP_POINT is its
	// length in the running bash's unit (bytes before 4.4, measured).
	point string
}

func testBash(t *testing.T, e shellEnvT, bash string) {
	script := filepath.ToSlash(e.scripts["bash"])
	version := e.run(t, bash, "-c", `echo "$BASH_VERSION"`)
	t.Logf("%s: bash %s", bash, strings.TrimSpace(version))
	e.run(t, bash, "-n", script)
	cases := []bashCase{
		{"gobble --output-format ", []string{"gobble", "--output-format", ""}, wantFormats, ""},
		// bash breaks words at '=': the current word is "=".
		{"gobble --log-level=", []string{"gobble", "--log-level", "="}, []string{"=debug", "=info", "=warn", "=error"}, ""},
		{"gobble --file dé", []string{"gobble", "--file", "dé"}, []string{"dés.txt"}, ""},
		{"gobble comp", []string{"gobble", "comp"}, []string{"completion"}, ""},
		// The cursor after "dé", in the middle of the line: the cut must
		// count COMP_POINT in the running bash's own unit.
		{"gobble --file dé --verbose", []string{"gobble", "--file", "dé"}, []string{"dés.txt"}, "gobble --file dé"},
	}
	var prog strings.Builder
	fmt.Fprintf(&prog, "source %s\n", shq(script))
	for _, c := range cases {
		quoted := make([]string, len(c.words))
		for i, w := range c.words {
			quoted[i] = shq(w)
		}
		point := cmpOr(c.point, c.line)
		fmt.Fprintf(&prog, "COMP_LINE=%s; p=%s; if (( BASH_VERSINFO[0] < 4 )); then COMP_POINT=$(LC_ALL=C; echo ${#p}); else COMP_POINT=${#p}; fi; "+
			"COMP_WORDS=(%s); COMP_CWORD=%d; COMPREPLY=(); _gobble; printf '%%s\\n' \"${COMPREPLY[@]}\"; echo ---\n",
			shq(c.line), shq(point), strings.Join(quoted, " "), len(c.words)-1)
	}
	got := strings.Split(e.run(t, bash, "-c", prog.String()), "---\n")
	for i, c := range cases {
		if lines := splitLines(got[i]); !slices.Equal(lines, c.want) {
			t.Errorf("%q: COMPREPLY = %q, want %q", c.line, lines, c.want)
		}
	}
}

func testFish(t *testing.T, e shellEnvT, fish string) {
	t.Logf("%s", strings.TrimSpace(e.run(t, fish, "--version")))
	e.run(t, fish, "--no-execute", e.scripts["fish"])
	for line, want := range map[string][]string{
		"gobble --output-format ": wantFormats,
		"gobble --log-level=":     {"--log-level=debug", "--log-level=info", "--log-level=warn", "--log-level=error"},
		"gobble --file dé":        {"dés.txt"},
	} {
		out := e.run(t, fish, "-c", fmt.Sprintf("source %s; complete -C %s", fishq(e.scripts["fish"]), fishq(line)))
		var got []string
		for _, l := range splitLines(out) {
			got = append(got, strings.SplitN(l, "\t", 2)[0])
		}
		if !slices.Equal(got, want) {
			t.Errorf("complete -C %q = %q, want %q", line, got, want)
		}
	}
}

func testZsh(t *testing.T, e shellEnvT, zsh string) {
	t.Logf("%s", strings.TrimSpace(e.run(t, zsh, "--version")))
	e.run(t, zsh, "-n", e.scripts["zsh"])
	// A stub harness: compdef and _describe record what the script asks for.
	harness := `compdef() { : }
_describe() { print -r -- "opts:$*"; print -rl -- "${comps[@]}"; print -r -- --- }
source ` + shq(e.scripts["zsh"]) + `
words=(gobble --output-format ''); CURRENT=3; PREFIX=''; _gobble
words=(gobble --log-level=); CURRENT=2; PREFIX='--log-level='; _gobble
words=(gobble --model x:); CURRENT=3; PREFIX='x:'; _gobble
words=(gobble --file dé); CURRENT=3; PREFIX='dé'; _gobble
`
	got := strings.Split(e.run(t, zsh, "-c", harness), "---\n")
	want := [][]string{
		wantFormats,
		{"--log-level=debug", "--log-level=info", "--log-level=warn", "--log-level=error"},
		// ':' in a value is escaped for _describe.
		{`x\:off`, `x\:minimal`, `x\:low`, `x\:medium`, `x\:high`, `x\:xhigh`},
		{"dés.txt"},
	}
	for i, w := range want {
		lines := splitLines(got[i])
		if len(lines) == 0 || !strings.HasPrefix(lines[0], "opts:") {
			t.Fatalf("case %d: no _describe call: %q", i, got[i])
		}
		if !slices.Equal(lines[1:], w) {
			t.Errorf("case %d: _describe got %q, want %q", i, lines[1:], w)
		}
	}
	if !strings.Contains(got[0], "opts:-V ") {
		t.Errorf("keep-order directive did not ask _describe for -V: %q", got[0])
	}
}

func testPowerShell(t *testing.T, e shellEnvT, ps string) {
	dir := t.TempDir()
	harness, results := filepath.Join(dir, "harness.ps1"), filepath.Join(dir, "results.txt")
	// The harness writes its results to a UTF-8 file. The console's code page
	// is left as the host has it, which is the case the script's own UTF-8
	// switch exists for; printing results through it would garble them.
	body := "\ufeff" + `$ErrorActionPreference = 'Stop'
$lines = [System.Collections.Generic.List[string]]::new()
$s = Get-Content -Raw -Encoding UTF8 -LiteralPath '` + e.scripts[shellPowerShell] + `'
$errs = $null
[void][System.Management.Automation.Language.Parser]::ParseInput($s, [ref]$null, [ref]$errs)
$lines.Add("parse-errors:$($errs.Count)")
$lines.Add("version:$($PSVersionTable.PSVersion)")
. ([scriptblock]::Create($s))
foreach ($in in @('gobble --output-format ', 'gobble --outp', 'gobble --file dé')) {
    $r = TabExpansion2 -inputScript $in -cursorColumn $in.Length
    foreach ($m in $r.CompletionMatches) { $lines.Add("$in|$($m.CompletionText)|$($m.ResultType)") }
}
[System.IO.File]::WriteAllLines('` + results + `', $lines, [System.Text.UTF8Encoding]::new($false))
`
	if err := os.WriteFile(harness, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	e.run(t, ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", harness)
	b, err := os.ReadFile(results)
	if err != nil {
		t.Fatal(err)
	}
	out := strings.ReplaceAll(string(b), "\r\n", "\n")
	lines := splitLines(out)
	if !slices.Contains(lines, "parse-errors:0") {
		t.Fatalf("the PowerShell parser reported errors:\n%s", out)
	}
	for _, l := range lines {
		if v, ok := strings.CutPrefix(l, "version:"); ok {
			t.Logf("%s: PowerShell %s", ps, v)
		}
	}
	want := []string{
		"gobble --output-format |text|ParameterValue",
		"gobble --output-format |json|ParameterValue",
		"gobble --output-format |stream-json|ParameterValue",
		"gobble --outp|--output-format|ParameterName",
		"gobble --file dé|dés.txt|ProviderItem",
	}
	var got []string
	for _, l := range lines {
		if strings.Contains(l, "|") {
			got = append(got, l)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("TabExpansion2 results:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// shq quotes for POSIX shells and zsh.
func shq(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// fishq quotes for fish.
func fishq(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, "'", `\'`).Replace(s) + "'"
}
