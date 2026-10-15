package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// piDir points PI_CODING_AGENT_DIR at a new directory holding auth.json
// with text, and returns the directory.
func piDir(t *testing.T, text string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PI_CODING_AGENT_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// ttyEnv is a runEnv whose stdin and stderr are terminals, reading stdin.
func ttyEnv(t *testing.T, stdin string) (e *runEnv, stdout, stderr *bytes.Buffer) {
	t.Helper()
	stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}
	o := &Output{out: stdout, err: stderr, caps: term.Caps{In: true, Err: term.Stream{TTY: true}}}
	e = &runEnv{ctx: t.Context(), out: o, stdin: stdinFile(t, stdin), logs: &lazyLog{out: o, stderr: stderr}}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	return e, stdout, stderr
}

// On a terminal, each entry gobble can use is asked about, and an entry
// auth.json has is replaced only when that is asked and answered yes: a key
// is copied as written, xAI's login with the SDK's issuer and client id;
// every other entry is skipped with its reason (0005-PLAN F4-3c decision 7).
func TestImportFromPi(t *testing.T) {
	c := newCfgEnv(t)
	piDir(t, `{
  "openai-codex": {"type": "oauth", "access": "a", "refresh": "r", "expires": 1},
  "anthropic": {"type": "oauth", "access": "a", "refresh": "r", "expires": 1},
  "xai": {"type": "oauth", "access": "xa", "refresh": "xr", "expires": 1781000000000},
  "openai": {"type": "api_key", "key": "sk-pi-openai"},
  "google": {"type": "api_key", "key": "$GEMINI_FROM_PI"},
  "together": {"type": "api_key", "key": "pi-tg"},
  "opencode-go": {"type": "api_key", "key": "og"},
  "huggingface": {"type": "api_key"},
  "nope": {"type": "api_key", "key": "k"}
}`)
	c.write(c.authFile(), `{"openai": {"type": "api_key", "key": "mine"}, "together": {"type": "api_key", "key": "mine-tg"}}`)
	answers := map[string]bool{
		"Import Pi's xai as grok?": true, "Import Pi's openai as openai?": true, "auth.json already has openai; replace it?": true,
		"Import Pi's google as gemini?": true, "Import Pi's together as together?": true,
		"auth.json already has together; replace it?": false, "Import Pi's opencode-go as opencode-go?": false,
	}
	var asked []string
	answer := confirm
	replace(t, &confirm, func(_ *runEnv, q string) (bool, error) {
		asked = append(asked, q)
		return answers[q], nil
	})
	e, stdout, _ := ttyEnv(t, "")
	if err := (&AuthImportCmd{FromPi: true}).Run(e); err != nil {
		t.Fatal(err)
	}
	want := `Skipped openai-codex: gobble has no provider openai-codex
Skipped anthropic: an OAuth login gobble does not use
Imported grok from Pi's xai
Imported openai from Pi's openai
Imported gemini from Pi's google
Skipped together: auth.json already has together
Skipped opencode-go: not chosen
Skipped huggingface: no key
Skipped nope: gobble has no provider nope
`
	if stdout.String() != want {
		t.Errorf("report:\n%s\nwant:\n%s", stdout, want)
	}
	if wantAsked := []string{"Import Pi's xai as grok?", "Import Pi's openai as openai?", "auth.json already has openai; replace it?",
		"Import Pi's google as gemini?", "Import Pi's together as together?", "auth.json already has together; replace it?",
		"Import Pi's opencode-go as opencode-go?"}; !slices.Equal(asked, wantAsked) {
		t.Errorf("asked %q, want %q", asked, wantAsked)
	}
	got := c.read(c.authFile())
	sdk, _ := provider.PiLogin("grok", "", "", time.Time{})
	if sdk == nil || sdk.Issuer == "" || sdk.ClientID == "" {
		t.Fatalf("PiLogin(grok) = %v", sdk)
	}
	for _, s := range []string{`"key": "sk-pi-openai"`, `"key": "$GEMINI_FROM_PI"`, `"key": "mine-tg"`, `"access": "xa"`, `"refresh": "xr"`,
		`"expires": 1781000000000`, `"issuer": "` + sdk.Issuer + `"`, `"clientId": "` + sdk.ClientID + `"`} {
		if !strings.Contains(got, s) {
			t.Errorf("auth.json has no %s:\n%s", s, got)
		}
	}
	for _, s := range []string{`"mine"`, `"pi-tg"`, `"opencode-go"`, `"claude"`, `"huggingface"`} {
		if strings.Contains(got, s) {
			t.Errorf("auth.json has %s:\n%s", s, got)
		}
	}

	// confirm asks on stderr and reads one line per question from stdin.
	e, _, stderr := ttyEnv(t, "y\nno\nYES\n")
	var said []bool
	for range 4 {
		ok, err := answer(e, "Q?")
		if err != nil {
			t.Fatal(err)
		}
		said = append(said, ok)
	}
	if !slices.Equal(said, []bool{true, false, true, false}) || stderr.String() != strings.Repeat("Q? [y/N] ", 4) {
		t.Errorf("confirm said %v, stderr %q", said, stderr)
	}
}

// Without a terminal, only --provider's entries are copied, named by
// gobble's or Pi's name, and an entry auth.json has is replaced only with
// --replace (0005-PLAN F4-3c decision 7).
func TestImportNoTerminal(t *testing.T) {
	c := newCfgEnv(t)
	piDir(t, `{"openai": {"type": "api_key", "key": "sk-pi"}, "xai": {"type": "oauth", "access": "xa", "refresh": "xr", "expires": 1},
  "anthropic": {"type": "api_key", "key": "sk-ant"}}`)
	c.write(c.authFile(), `{"openai": {"type": "api_key", "key": "mine"}}`)
	c.want(c.run("auth", "import", "--from-pi", "--provider", "grok", "--provider", "openai"), ExitOK,
		"Skipped openai: auth.json already has openai; --replace replaces it\nImported grok from Pi's xai\nSkipped anthropic: not named by --provider\n", "")
	c.want(c.run("auth", "import", "--from-pi", "--provider", "anthropic"), ExitOK,
		"Skipped openai: not named by --provider\nSkipped xai: not named by --provider\nImported claude from Pi's anthropic\n", "")
	if got := c.read(c.authFile()); !strings.Contains(got, `"key": "mine"`) || !strings.Contains(got, `"key": "sk-ant"`) {
		t.Errorf("auth.json:\n%s", got)
	}
	c.want(c.run("auth", "import", "--from-pi", "--provider", "openai", "--replace"), ExitOK, "Imported openai from Pi's openai\n", "")
	if got := c.read(c.authFile()); !strings.Contains(got, `"key": "sk-pi"`) || strings.Contains(got, `"mine"`) {
		t.Errorf("auth.json after --replace:\n%s", got)
	}
	r := c.run("auth", "import", "--from-pi")
	c.want(r, ExitOK, "Skipped openai: not named by --provider\nSkipped xai: not named by --provider\nSkipped anthropic: not named by --provider\n", "")
	c.want(c.run("auth", "import", "--from-pi", "--provider", "nope"), ExitUsage, "", `unknown provider "nope"`)
	c.want(c.run("auth", "import"), ExitUsage, "", "--from-pi")
}

// The import reads Pi's auth.json and never writes it: its bytes and mode
// are kept, and no lock file appears beside it (0003-MADR; 0005-PLAN F4-3c
// decision 7).
func TestImportPiUntouched(t *testing.T) {
	c := newCfgEnv(t)
	text := "{\n  \"openai\": {\"type\": \"api_key\", \"key\": \"sk-pi\", \"env\": \"OPENAI_API_KEY\"},\n" +
		"  \"xai\": {\"type\": \"oauth\", \"access\": \"xa\", \"refresh\": \"xr\", \"expires\": 1}\n}\n"
	dir := piDir(t, text)
	file := filepath.Join(dir, "auth.json")
	before, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	c.want(c.run("auth", "import", "--from-pi", "--provider", "openai", "--provider", "xai"), ExitOK,
		"Imported openai from Pi's openai\nImported grok from Pi's xai\n", "")
	after, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.read(file); got != text || after.Mode() != before.Mode() || !after.ModTime().Equal(before.ModTime()) {
		t.Errorf("Pi's auth.json changed: mode %v to %v, text %q", before.Mode(), after.Mode(), got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range entries {
		names = append(names, d.Name())
	}
	if !slices.Equal(names, []string{"auth.json"}) {
		t.Errorf("Pi's directory holds %q, want auth.json alone", names)
	}
}

// Pi's auth.json is in PI_CODING_AGENT_DIR, a leading ~ expanded, or in
// ~/.pi/agent; a missing file says where it looked, and exits 1 (0005-PLAN
// F4-3c decision 7).
func TestImportLocation(t *testing.T) {
	c := newCfgEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for dir, want := range map[string]string{
		filepath.Join(c.dir, "pi"): filepath.Join(c.dir, "pi", "auth.json"),
		"~/pi-elsewhere":           filepath.Join(home, "pi-elsewhere", "auth.json"),
		"":                         filepath.Join(home, ".pi", "agent", "auth.json"),
	} {
		t.Setenv("PI_CODING_AGENT_DIR", dir)
		if got, err := piAuthPath(); err != nil || got != want {
			t.Errorf("PI_CODING_AGENT_DIR %q: %q, %v; want %q", dir, got, err, want)
		}
	}
	t.Setenv("PI_CODING_AGENT_DIR", "")
	c.want(c.run("auth", "import", "--from-pi"), ExitFailure, "",
		"Pi has no auth.json at "+filepath.Join(home, ".pi", "agent", "auth.json"))
}
