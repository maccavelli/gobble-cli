package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/personality"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// captureTools records the options each built-in tool set is made with.
func captureTools(t *testing.T) func() []builtin.Options {
	t.Helper()
	var mu sync.Mutex
	var got []builtin.Options
	old := toolsWith
	toolsWith = func(o builtin.Options) []tool.Tool {
		mu.Lock()
		got = append(got, o)
		mu.Unlock()
		return old(o)
	}
	t.Cleanup(func() { toolsWith = old })
	return func() []builtin.Options {
		mu.Lock()
		defer mu.Unlock()
		return append([]builtin.Options(nil), got...)
	}
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// The bash keys of the global file reach builtin.Options, ~ expanded, for
// chat and for acp; acp keeps builtin's own output directory (0018-PLAN P5
// deviation 16). With no file they are today's defaults.
func TestProcessSettingsReachTools(t *testing.T) {
	withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("ok")}}}})
	got := captureTools(t)
	home := t.TempDir()
	t.Chdir(t.TempDir())
	writeFile(t, filepath.Join(home, "settings.yaml"),
		"shellPath: ~/bin/bash\nshellCommandPrefix: set -e\ntools:\n  bash:\n    useClientTerminal: true\n    maxTimeoutMs: 1234\n")
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := builtin.Options{
		ShellPath: filepath.Join(userHome, "bin", "bash"), ShellCommandPrefix: "set -e",
		MaxTimeout: 1234 * time.Millisecond, UseClientTerminal: true,
	}
	if r := runIn(t, home, "-p", "hi"); r.code != ExitOK {
		t.Fatalf("chat: %+v", r)
	}
	if r := runIn(t, home, "acp"); r.code != ExitOK {
		t.Fatalf("acp: %+v", r)
	}
	opts := got()
	if len(opts) != 2 {
		t.Fatalf("%d tool sets built, want 2", len(opts))
	}
	chat, acp := opts[0], opts[1]
	if chat.OutputDir != filepath.Join(home, "output") {
		t.Errorf("chat's OutputDir %q", chat.OutputDir)
	}
	chat.OutputDir = ""
	// No file under personalities/prompts: no description of the user's
	// (0022-PLAN P5).
	if len(chat.Descriptions) != 0 || len(acp.Descriptions) != 0 {
		t.Fatalf("descriptions %v, %v", chat.Descriptions, acp.Descriptions)
	}
	chat.Descriptions, acp.Descriptions = nil, nil
	if !reflect.DeepEqual(chat, want) || !reflect.DeepEqual(acp, want) {
		t.Fatalf("chat %+v, acp %+v; want %+v", chat, acp, want)
	}
	if r := runIn(t, t.TempDir(), "acp"); r.code != ExitOK {
		t.Fatalf("acp: %+v", r)
	}
	last := got()[2]
	last.Descriptions = nil
	if !reflect.DeepEqual(last, builtinOptions(config.Defaults())) || last.MaxTimeout != 10*time.Minute || last.ShellPath != "" {
		t.Fatalf("no file gave %+v", last)
	}
}

// sessionDir keeps sessions flat in that directory, for chat and for acp's
// store (0018-PLAN P5 deviation 21).
func TestSessionDirSetting(t *testing.T) {
	withModel(t, llmtest.NewScript(llmtest.Text("ok")))
	home, dir := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeFile(t, filepath.Join(home, "settings.yaml"), "sessionDir: '"+dir+"'\n")
	mustRun(t, home, "-p", "x")
	if flat, err := filepath.Glob(filepath.Join(dir, "*.jsonl")); err != nil || len(flat) != 1 {
		t.Fatalf("flat files in sessionDir: %v, %v", flat, err)
	}
	if files := storeFiles(t, filepath.Join(home, "sessions")); len(files) != 0 {
		t.Fatalf("the data directory's store was used: %v", files)
	}
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	e.processSettings("") // as acp does
	if st, ok := e.sessionStore().(*jsonl.Store); !ok || st.Dir() != dir {
		t.Fatalf("acp's store %T at %v, want %s", e.sessionStore(), st, dir)
	}
}

// The agent's Settings function maps the global file's session keys to
// acpserver's (0018-MADR D17), compaction complete with Pi's reserve, and
// with no file gives today's defaults.
func TestSessionSettingsReachAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	writeFile(t, filepath.Join(home, "settings.yaml"),
		"defaultThinkingLevel: low\ncacheRetention: long\nsteeringMode: all\nfollowUpMode: all\ncompaction:\n  keepRecentTokens: 100\n"+
			"mcp:\n  startupWaitMs: 7\n  stopGraceMs: 8\n  stopTimeoutMs: 9\nshellCommandPrefix: set -e\n"+
			"permissions:\n  allowBypass: true\n  rules:\n    - deny: read(.env)\n")
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, store: session.NewMemoryStore()}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	cwd := t.TempDir()
	got := e.agentOptions().Settings(cwd)
	want := acpserver.SessionSettings{
		SteeringMode: "all", FollowUpMode: "all", Thinking: "low", CacheRetention: "long",
		Compaction:     compaction.Settings{ReserveTokens: compaction.DefaultSettings.ReserveTokens, KeepRecentTokens: 100},
		MCPStartupWait: 7 * time.Millisecond, MCPStopGrace: 8 * time.Millisecond, MCPStopTimeout: 9 * time.Millisecond,
		Retry: llm.DefaultRetry, Personality: personality.Default,
		// A project with no files that need trust is trusted (0005-PLAN F6-1).
		ProjectRoot: cwd, Trust: permission.TrustDecision{Trusted: true, Source: permission.SourceNothing},
		// The rules, each with its file and line (0005-PLAN F6-2).
		Rules: []permission.Spec{{Effect: permission.ModeDeny, Tool: "read", Pattern: ".env",
			Source: permission.RuleSource{Layer: permission.LayerGlobal, File: filepath.Join(home, "settings.yaml"), Line: 15}}},
		// bypass offered (0005-PLAN F6-4 decision 2).
		AllowBypass: true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Settings = %+v, want %+v", got, want)
	}
	if e.sessionSettings("").Settings.ShellCommandPrefix != "" {
		t.Fatal("a process key reached the session settings")
	}
	today := acpserver.SessionSettings{
		SteeringMode: "one-at-a-time", FollowUpMode: "one-at-a-time", Thinking: "off", CacheRetention: "short", Compaction: compaction.DefaultSettings,
		MCPStartupWait: 10 * time.Second, MCPStopGrace: 500 * time.Millisecond, MCPStopTimeout: 2 * time.Second,
		Retry: llm.DefaultRetry, Personality: personality.Default,
	}
	if got := toSession(config.Defaults()); !reflect.DeepEqual(got, today) {
		t.Fatalf("toSession(Defaults()) = %+v, want %+v", got, today)
	}
}

// --set beats the file.
func TestSetBeatsFile(t *testing.T) {
	withModel(t, llmtest.NewScript(llmtest.Text("ok")))
	home, fromFile, fromSet := t.TempDir(), t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	writeFile(t, filepath.Join(home, "settings.yaml"), "sessionDir: '"+fromFile+"'\n")
	mustRun(t, home, "--set", "sessionDir="+fromSet, "-p", "x")
	a, _ := filepath.Glob(filepath.Join(fromSet, "*.jsonl"))
	b, _ := filepath.Glob(filepath.Join(fromFile, "*.jsonl"))
	if len(a) != 1 || len(b) != 0 {
		t.Fatalf("--set's dir %v, the file's %v", a, b)
	}
}

// --session-dir beats --set sessionDir=….
func TestFlagBeatsSet(t *testing.T) {
	withModel(t, llmtest.NewScript(llmtest.Text("ok")))
	home, fromFlag, fromSet := t.TempDir(), t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	mustRun(t, home, "--set", "sessionDir="+fromSet, "--session-dir", fromFlag, "-p", "x")
	a, _ := filepath.Glob(filepath.Join(fromFlag, "*.jsonl"))
	b, _ := filepath.Glob(filepath.Join(fromSet, "*.jsonl"))
	if len(a) != 1 || len(b) != 0 {
		t.Fatalf("--session-dir's dir %v, --set's %v", a, b)
	}
}

// Under acp no project file is read for the process keys (0018-MADR D16):
// chat in the same directory reports the project file, acp's log does not.
func TestACPIgnoresProjectProcessKeys(t *testing.T) {
	withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("ok")}}}})
	got := captureTools(t)
	home, project := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(project, ".gobble", "settings.yaml"), "tools:\n  bash:\n    maxTimeoutMs: 5\n")
	t.Chdir(project)
	r := runIn(t, home, "-p", "hi")
	if r.code != ExitOK || !strings.Contains(r.stderr, "project settings apply once the project is trusted") {
		t.Fatalf("chat did not read the project file: %+v", r)
	}
	if r := runIn(t, home, "acp"); r.code != ExitOK {
		t.Fatalf("acp: %+v", r)
	}
	b, err := os.ReadFile(filepath.Join(home, "logs", "gobble.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), ".gobble") {
		t.Fatalf("acp read the project file:\n%s", b)
	}
	if acp := got()[1]; acp.MaxTimeout != 10*time.Minute {
		t.Fatalf("acp's MaxTimeout %v", acp.MaxTimeout)
	}
}

// --add-dir is made absolute against --cwd, or the working directory, and
// must be a directory.
func TestAddDirAbsoluteAndChecked(t *testing.T) {
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	got, err := resolveAddDirs(base, []string{"sub", base})
	if err != nil || len(got) != 2 || got[0] != filepath.Join(base, "sub") || got[1] != base {
		t.Fatalf("resolveAddDirs = %q, %v", got, err)
	}
	t.Chdir(base)
	if got, err := resolveAddDirs("", []string{"sub"}); err != nil || got[0] != filepath.Join(base, "sub") {
		t.Fatalf("against the working directory: %q, %v", got, err)
	}
	writeFile(t, filepath.Join(base, "file"), "x")
	for _, bad := range []string{"missing", "file"} {
		if _, err := resolveAddDirs(base, []string{bad}); err == nil || !strings.Contains(err.Error(), "--add-dir "+bad+": not a directory") {
			t.Errorf("%s: %v", bad, err)
		}
	}
	withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("ok")}}}})
	if r := runMain(t, nil, "--add-dir", filepath.Join(base, "missing"), "-p", "hi"); r.code != ExitUsage || !strings.Contains(r.stderr, "not a directory") {
		t.Fatalf("a missing --add-dir: %+v", r)
	}
}

// defaultProvider's values are the SDK's providers that take a key, in
// order; the setting reaches the agent's provider and model list as gobble
// acp reads it. A setting whose provider has no credential, while another
// has one, starts the run there with a warning (0025-MADR D3; it amends
// 0019-MADR D15's error, which stays for --model and for a run with no
// credential at all: TestStartNoCredential).
func TestDefaultProviderSetting(t *testing.T) {
	k, ok := config.KeyOf("defaultProvider")
	if want := append([]string{""}, provider.KeyProviders()...); !ok || !slices.Equal(k.Enum, want) {
		t.Fatalf("defaultProvider's enum %q, want %q", k.Enum, want)
	}
	home := t.TempDir()
	t.Chdir(t.TempDir())
	t.Setenv("GOBBLE_HOME", home)
	t.Setenv("OPENCODE_API_KEY", "test-only")
	writeFile(t, filepath.Join(home, "settings.yaml"), "defaultProvider: opencode-go\n")
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	e.processSettings("") // as acp does
	if c := e.modelChoice(); c.Provider != "opencode-go" || c.Default != "claude-haiku-5-5" || !slices.Equal(c.Models, provider.Models("opencode-go")) {
		t.Fatalf("model choice %+v, want opencode-go's", c)
	}
	if p, err := modelProvider(e)("s1", llm.DefaultRetry); err != nil || p.ID() != "opencode-go" {
		t.Fatalf("provider %v, %v; want opencode-go", p, err)
	}
	writeFile(t, filepath.Join(home, "settings.yaml"), "defaultProvider: openai\n")
	stderr := &syncBuffer{} // background listings warn into it
	out2 := &Output{out: &bytes.Buffer{}, err: stderr}
	e2 := &runEnv{ctx: t.Context(), out: out2, logs: &lazyLog{out: out2, stderr: stderr}}
	t.Cleanup(func() {
		if err := e2.logs.close(); err != nil {
			t.Error(err)
		}
	})
	e2.processSettings("")
	if c := e2.modelChoice(); c.Provider != "opencode-zen" || !strings.Contains(stderr.String(), "Warning: defaultProvider openai has no credential; the run starts on another provider") {
		t.Fatalf("openai without its key: started on %q, warnings %q; want opencode-zen, the first with a key, and the warning", c.Provider, stderr.String())
	}
}
