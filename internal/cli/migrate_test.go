package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lib "github.com/maccavelli/go-selfupdate-lib/buildinfo"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/internal/buildinfo"
	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// dormantAuth is an auth.json holding an oauth entry with a dormant key, as
// gobble before 0052-MADR F25 could leave one; built in a scratch home, never
// read from the owner's (0052-PLAN C10).
const dormantAuth = "{\n  \"anthropic\": {\n    \"type\": \"oauth\",\n    \"access\": \"a-secret\",\n" +
	"    \"refresh\": \"r-secret\",\n    \"expires\": 1781000000000,\n    \"key\": \"sk-dormant-secret\"\n  }\n}\n"

// backupsIn are the backups of auth.json in dir.
func backupsIn(t *testing.T, dir string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(dir, "auth.json.bak-*"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// kindlessPlan is a ChatGPT plan session as a build of the archived work
// stored one: no kind member, no answer (0052-MADR D9, Q6).
func kindlessPlan() *provider.Login {
	l := planLogin()
	l.Kind = ""
	return l
}

// failingMigration is M48's fixture: a plan entry holding both ids and a
// clients.json that does not parse, so the migration fails at step 4 while
// the state directory and the log file work.
func failingMigration(t *testing.T, c *cfgEnv) {
	t.Helper()
	saveLogin(t, c, "openai", planLogin())
	c.write(filepath.Join(c.home, "clients.json"), "not json\n")
}

// migrationRecords are the log file's Warn records of a failed migration
// whose message holds naming, clients.json when not given.
func migrationRecords(t *testing.T, home string, naming ...string) int {
	t.Helper()
	if len(naming) == 0 {
		naming = []string{"clients.json"}
	}
	b, err := os.ReadFile(filepath.Join(home, "logs", "gobble.log"))
	if err != nil {
		t.Fatalf("the log file: %v", err)
	}
	n := 0
	for line := range strings.Lines(string(b)) {
		var r map[string]any
		if err := jsonx.UnmarshalStrict([]byte(line), &r); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		if r["msg"] == "auth.json migration" && r["level"] == "WARN" && strings.Contains(r["message"].(string), naming[0]) {
			n++
		}
	}
	return n
}

// Every command run dispatches migrates but auth status and auth check,
// with or without --json, which leave auth.json, its backups and the state
// files as they are (rows M45, M56).
func TestMigrateRunsExceptStatusAndCheck(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), dormantAuth)
	for _, args := range [][]string{{"auth", "status"}, {"auth", "status", "--json"}, {"auth", "check", "--provider", "openai", "--json"}} {
		c.run(args...)
		if c.read(c.authFile()) != dormantAuth {
			t.Fatalf("%v changed auth.json", args)
		}
		if b := backupsIn(t, c.home); len(b) != 0 {
			t.Fatalf("%v made a backup: %v", args, b)
		}
		missing(t, filepath.Join(c.home, "host-id"))
		missing(t, filepath.Join(c.home, "clients.json"))
	}
	if r := c.run("version"); r.code != ExitOK {
		t.Fatalf("version: exit %d, stderr %q", r.code, r.stderr)
	}
	if strings.Contains(c.read(c.authFile()), "sk-dormant") {
		t.Error("gobble version did not migrate")
	}
}

// Completion mode never migrates: Main returns before run (row M44).
func TestCompletionModeDoesNotMigrate(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), dormantAuth)
	t.Setenv(complete.EnvShell, "fish")
	t.Setenv(complete.EnvProtocol, complete.Protocol)
	if r := c.run("--", "--log-level", ""); r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	if c.read(c.authFile()) != dormantAuth || len(backupsIn(t, c.home)) != 0 {
		t.Error("completion mode migrated")
	}
}

// gobble acp migrates once before it serves, with nothing of it on stdout
// (row M46).
func TestACPStartMigrates(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), dormantAuth)
	r := c.runStdin(stdinFile(t, ""), "acp")
	if r.code != ExitOK || r.stdout != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	if strings.Contains(c.read(c.authFile()), "sk-dormant") || len(backupsIn(t, c.home)) != 1 {
		t.Errorf("acp: key dropped %v, backups %v; want the key dropped and one backup",
			!strings.Contains(c.read(c.authFile()), "sk-dormant"), backupsIn(t, c.home))
	}
}

// An error from the migration is one warning line on stderr, and the command
// runs with its own exit status (row M48).
func TestMigrateErrorIsAWarning(t *testing.T) {
	c := newCfgEnv(t)
	failingMigration(t, c)
	r := c.run("version")
	if r.code != ExitOK || !strings.HasPrefix(r.stdout, "gobble ") {
		t.Fatalf("exit %d, stdout %q; want version's own", r.code, r.stdout)
	}
	lines := strings.Split(strings.TrimSuffix(r.stderr, "\n"), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "clients.json") {
		t.Errorf("stderr %q, want one warning line naming the error", r.stderr)
	}
}

// An auth.json that cannot be parsed is the command's to report, not the
// migration's: gobble version, which never reads the file, prints nothing of
// it on stderr, and the log file keeps one Warn record (row M66).
// TestAuthConsentCommand holds the other half: auth consent's one error line.
func TestMigrateUnreadableAuthJSONIsTheCommands(t *testing.T) {
	c := newCfgEnv(t)
	const broken = `{"openai": `
	c.write(c.authFile(), broken)
	r := c.run("version")
	if r.code != ExitOK || r.stderr != "" || !strings.HasPrefix(r.stdout, "gobble ") {
		t.Errorf("exit %d, stdout %q, stderr %q; want version's own output and nothing on stderr", r.code, r.stdout, r.stderr)
	}
	if c.read(c.authFile()) != broken {
		t.Error("auth.json was written")
	}
	if n := migrationRecords(t, c.home, "auth.json"); n != 1 {
		t.Errorf("%d Warn records of the migration in the log, want one", n)
	}
}

// A run whose context a signal cancelled stops after the migration: the
// command does not run, and the exit status is 128 plus the signal (row M51).
func TestMigrateSignalSkipsTheCommand(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), dormantAuth)
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(signalCause{sig: os.Interrupt})
	var out, errw bytes.Buffer
	code := Main(ctx, []string{"version"}, nil, &out, &errw)
	if want := exitForSignal(os.Interrupt); code != want || out.Len() != 0 {
		t.Errorf("exit %d, stdout %q; want %d and no output: the command must not run", code, out.String(), want)
	}
	if c.read(c.authFile()) != dormantAuth {
		t.Error("a cancelled migration wrote auth.json")
	}
}

// With no directories the migration is skipped with one warning on stderr,
// and none under auth login --json (row M52).
func TestMigrateSkippedWithoutDirs(t *testing.T) {
	t.Setenv("GOBBLE_HOME", "relative")
	var out, errw bytes.Buffer
	Main(t.Context(), []string{"version"}, nil, &out, &errw)
	if n := strings.Count(errw.String(), "auth.json not migrated"); n != 1 {
		t.Errorf("version: %d skip lines in %q, want one", n, errw.String())
	}
	out.Reset()
	errw.Reset()
	Main(t.Context(), []string{"auth", "login", "--json", "nosuchprovider"}, nil, &out, &errw)
	if n := strings.Count(errw.String(), "auth.json not migrated"); n != 0 {
		t.Errorf("auth login --json: %d skip lines in %q, want none", n, errw.String())
	}
}

// A build that is not a release migrates as one would: nothing checks the
// version (row M54).
func TestMigrateRunsOnAnyBuild(t *testing.T) {
	old := identity
	t.Cleanup(func() { identity = old })
	identity = func() buildinfo.Info {
		return buildinfo.Info{Version: "(devel)", Lib: lib.Info{Version: "(devel)", Kind: lib.KindLocal}}
	}
	c := newCfgEnv(t)
	c.write(c.authFile(), dormantAuth)
	if r := c.run("version"); r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	if strings.Contains(c.read(c.authFile()), "sk-dormant") {
		t.Error("a development build did not migrate")
	}
}

// Nothing of the migration reaches stderr under auth login --json and gobble
// acp, a log file that cannot be opened included; a failure is one Warn
// record in the log file (row M55).
func TestMigrateSilentUnderJSONAndACP(t *testing.T) {
	commands := map[string]func(c *cfgEnv) result{
		"auth login --json": func(c *cfgEnv) result { return c.run("auth", "login", "--json", "nosuchprovider") },
		"acp":               func(c *cfgEnv) result { return c.runStdin(stdinFile(t, ""), "acp") },
	}
	for name, run := range commands {
		t.Run(name, func(t *testing.T) {
			c := newCfgEnv(t)
			failingMigration(t, c)
			if r := run(c); r.stderr != "" {
				t.Errorf("stderr %q, want none", r.stderr)
			}
			if n := migrationRecords(t, c.home); n != 1 {
				t.Errorf("%d Warn records of the migration in the log, want one", n)
			}
		})
		t.Run(name+" with no log file", func(t *testing.T) {
			c := newCfgEnv(t)
			failingMigration(t, c)
			c.write(filepath.Join(c.home, "logs"), "a file, not a directory\n")
			if r := run(c); r.stderr != "" {
				t.Errorf("stderr %q, want none", r.stderr)
			}
		})
	}
}

// The Settings closure reads the global settings.yaml alone, by its own
// parse: only the YAML boolean true is true; a file it cannot parse, a
// duplicated key included, is (false, false) (rows M30, M53, M65).
func TestMigrateSettingsIsGlobalOnly(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.project(), "chatgptPlanBackground: true\n")
	check := func(what string, want [2]bool) {
		t.Helper()
		got, ok := globalSetting(c.home)()
		if [2]bool{got, ok} != want {
			t.Errorf("%s: (%v, %v), want %v", what, got, ok, want)
		}
	}
	check("true in the project file only", [2]bool{false, true})
	c.write(c.global(), "theme: dark\nchatgptPlanBackground: true\n")
	check("true in the global file", [2]bool{true, true})
	for _, v := range []string{`"true"`, "yes", "1", "false"} {
		c.write(c.global(), "chatgptPlanBackground: "+v+"\n")
		check("the value "+v, [2]bool{false, true})
	}
	c.write(c.global(), "chatgptPlanBackground: false\nchatgptPlanBackground: true\n")
	check("a duplicated key", [2]bool{false, false})
	c.write(c.global(), "chatgptPlanBackground: [true\n")
	check("a file that does not parse", [2]bool{false, false})
	if err := os.Remove(c.global()); err != nil {
		t.Fatal(err)
	}
	check("no global file", [2]bool{false, true})
}

// The PlanSession closure tells a plan session by the SDK's kind, inferred
// for an entry with no kind member, and its answer by the consent's id that
// llm/provider gives (row M12).
func TestPlanSessionHook(t *testing.T) {
	c := newCfgEnv(t)
	hook := planSessionHook(c.authFile())
	check := func(what string, wantPlan, wantAnswered bool) {
		t.Helper()
		plan, answered, err := hook(t.Context())
		if err != nil || plan != wantPlan || answered != wantAnswered {
			t.Errorf("%s: (%v, %v, %v), want (%v, %v, nil)", what, plan, answered, err, wantPlan, wantAnswered)
		}
	}
	check("no entry", false, false)
	saveLogin(t, c, "openai", kindlessPlan())
	check("a plan entry with no kind member and no answer", true, false)
	granted := kindlessPlan()
	granted.Consents = planConsentID + "=granted"
	saveLogin(t, c, "openai", granted)
	check("granted", true, true)
	if _, err := auth.Delete(t.Context(), c.authFile(), "openai"); err != nil {
		t.Fatal(err)
	}
	saveLogin(t, c, "openai", foreignLogin())
	check("another application's ChatGPT login", false, false)
	c.write(c.authFile(), "not json\n")
	if _, _, err := hook(t.Context()); err == nil {
		t.Error("an unparseable auth.json gave no error")
	}
}

// The consent carried end to end: a kind-less plan session and the global
// setting true; after gobble version, auth status --json shows the consent
// granted (row M24).
func TestMigrateCarriesConsentEndToEnd(t *testing.T) {
	c := newCfgEnv(t)
	saveLogin(t, c, "openai", kindlessPlan())
	c.write(c.global(), "chatgptPlanBackground: true\n")
	if r := c.run("version"); r.code != ExitOK {
		t.Fatalf("version: exit %d, stderr %q", r.code, r.stderr)
	}
	r := c.run("auth", "status", "--json")
	var report struct {
		Providers []statusPlanRow `json:"providers"`
	}
	if err := jsonx.Unmarshal([]byte(r.stdout), &report); err != nil {
		t.Fatalf("auth status --json: %v\n%s", err, r.stdout)
	}
	for _, row := range report.Providers {
		if row.Provider == "openai" {
			if got := row.Consents[planConsentID]; got != "granted" {
				t.Errorf("openai's consent %q, want granted; row %+v", got, row)
			}
			return
		}
	}
	t.Errorf("no openai row in %s", r.stdout)
}
