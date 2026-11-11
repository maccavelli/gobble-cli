package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/glyph"

	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/personality"
	"github.com/maccavelli/gobble-cli/session"
)

// untrustedLine is decision 9's line (0005-PLAN F6-1).
const untrustedLine = "this project is not trusted: its .gobble files and skills are not loaded; gobble trust trusts it"

// trustMenu is the trust question about proj as a test expects it: its
// title, its answers on one row, and cancel. It is written out here, not
// built by permissionQuestion, so a test of the drawing is not a test of
// the drawing against itself (0053-MADR D3).
func trustMenu(proj string, answers ...ask.Answer) ask.Question {
	return ask.Question{Title: "Trust " + proj + "?", Rows: []ask.Row{{Answers: answers}},
		Cancel: ask.Answer{Key: "c", Label: "cancel"}}
}

// numbered are labels as answers keyed 1, 2, and on.
func numbered(labels ...string) []ask.Answer {
	out := make([]ask.Answer, len(labels))
	for i, l := range labels {
		out[i] = ask.Answer{Key: strconv.Itoa(i + 1), Label: l}
	}
	return out
}

// drawnRows is how many times screen shows q as go-tui-lib's ask lays it
// out at width, every row of it but the prompt's, as one run of rows: a
// path too long for a row is checked as it wraps, so no host's temp root
// changes the result (0053-MADR D3).
func drawnRows(screen string, q ask.Question, width int) int {
	want := ask.Layout(q, width, ansi.WcWidth, glyph.For(false))
	want = want[:len(want)-1]
	rows := strings.Split(screen, "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	n := 0
	for i := 0; i+len(want) <= len(rows); i++ {
		if slices.Equal(rows[i:i+len(want)], want) {
			n++
		}
	}
	return n
}

// projectSystem is trustProject's SYSTEM.md.
const projectSystem = "The project's own system prompt."

// trustProject is a repository with a SYSTEM.md of its own, by its real
// path, as trust.json keys it.
func trustProject(t *testing.T) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(p, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeFile(t, filepath.Join(p, ".gobble", "SYSTEM.md"), projectSystem)
	return p
}

// setHome makes dir the home directory, as os.UserHomeDir reads it: HOME,
// and USERPROFILE on Windows.
func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

// saved is trust.json's decisions under home.
func saved(t *testing.T, home string) map[string]bool {
	t.Helper()
	m, err := permission.TrustStore{Path: filepath.Join(home, permission.TrustFile)}.Load()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// gobble trust and untrust save the project root's decision, by any
// directory in it; --forget removes it, --list lists them; home is refused
// (0005-PLAN F6-1 decisions 3 and 7).
func TestTrustCommands(t *testing.T) {
	home, proj := t.TempDir(), trustProject(t)
	t.Chdir(t.TempDir())
	if r := mustRun(t, home, "trust", proj); r.stdout != "Saved: "+proj+" is trusted.\n"+proj+": trusted, saved for "+proj+"\n" {
		t.Fatalf("trust: %q", r.stdout)
	}
	sub := filepath.Join(proj, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	// sub's own .gobble has nothing, so the root's decision decides nothing
	// there (decision 1).
	if r := mustRun(t, home, "untrust", sub); r.stdout != "Saved: "+proj+" is not trusted.\n"+sub+": trusted: the project has no files that need trust\n" {
		t.Fatalf("untrust: %q", r.stdout)
	}
	if m := saved(t, home); len(m) != 1 || m[proj] {
		t.Fatalf("trust.json %v", m)
	}
	if r := mustRun(t, home, "trust", "--list"); r.stdout != proj+"\tnot trusted\n" {
		t.Fatalf("--list: %q", r.stdout)
	}
	if r := mustRun(t, home, "untrust", "--forget", proj); !strings.HasPrefix(r.stdout, "Forgot the decision saved for "+proj+".\n"+proj+": not trusted: nothing is saved") {
		t.Fatalf("--forget: %q", r.stdout)
	}
	if r := mustRun(t, home, "untrust", "--forget", proj); !strings.HasPrefix(r.stdout, "No decision is saved for "+proj+".\n") {
		t.Fatalf("--forget again: %q", r.stdout)
	}
	if r := mustRun(t, home, "trust", "--list"); r.stdout != "No decisions are saved.\n" {
		t.Fatalf("--list, empty: %q", r.stdout)
	}
	if r := runIn(t, home, "trust", "--list", proj); r.code != ExitUsage {
		t.Fatalf("--list with a directory: exit %d", r.code)
	}
	fake, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	setHome(t, fake)
	if r := runIn(t, home, "trust", fake); r.code != ExitFailure || !strings.Contains(r.stderr, "trust "+fake+": trusting it would trust every project under it") {
		t.Fatalf("home: exit %d, stderr %q", r.code, r.stderr)
	}
	if m := saved(t, home); len(m) != 0 {
		t.Fatalf("home was saved: %v", m)
	}
}

// --approve, --no-approve, a saved decision and defaultProjectTrust decide
// whether print mode loads the project's files, in decision 2's order;
// with nothing decided print mode cannot ask, and says so once on stderr;
// a project with nothing to trust is told nothing.
func TestPrintModeTrust(t *testing.T) {
	proj := trustProject(t)
	t.Chdir(proj)
	run := func(home string, args ...string) (loaded bool, lines int) {
		t.Helper()
		model := llmtest.NewScript(llmtest.Text("ok"))
		withModel(t, model)
		r := mustRun(t, home, append(args, "-p", "hi")...)
		if strings.Contains(r.stdout, untrustedLine) {
			t.Errorf("%v: the reply has the line: %q", args, r.stdout)
		}
		return strings.Contains(model.Requests()[0].Messages[0].Content[0].Text, projectSystem), strings.Count(r.stderr, untrustedLine)
	}
	home := t.TempDir()
	for _, c := range []struct {
		name   string
		setup  []string
		args   []string
		loaded bool
	}{
		{"nothing decided", nil, nil, false},
		{"--approve", nil, []string{"--approve"}, true},
		{"saved", []string{"trust", proj}, nil, true},
		{"--no-approve over saved", nil, []string{"--no-approve"}, false},
		{"saved no", []string{"untrust", proj}, nil, false},
		{"--approve over saved no", nil, []string{"--approve"}, true},
	} {
		if c.setup != nil {
			mustRun(t, home, c.setup...)
		}
		if loaded, lines := run(home, c.args...); loaded != c.loaded || (lines == 1) == c.loaded || lines > 1 {
			t.Errorf("%s: loaded %v, %d lines", c.name, loaded, lines)
		}
	}
	mustRun(t, home, "untrust", "--forget", proj)
	writeFile(t, filepath.Join(home, "settings.yaml"), "defaultProjectTrust: always\n")
	if loaded, lines := run(home); !loaded || lines != 0 {
		t.Errorf("always: loaded %v, %d lines", loaded, lines)
	}
	writeFile(t, filepath.Join(home, "settings.yaml"), "defaultProjectTrust: never\n")
	if loaded, lines := run(home, "--approve"); !loaded || lines != 0 {
		t.Errorf("never, --approve: loaded %v, %d lines", loaded, lines)
	}
	t.Chdir(t.TempDir())
	if _, lines := run(t.TempDir(), "--no-approve"); lines != 0 {
		t.Errorf("nothing to trust, and told %d times", lines)
	}
}

// An untrusted project's settings.yaml has no effect, and a trusted one's
// does (0005-PLAN F6's accept).
func TestProjectSettingsNeedTrust(t *testing.T) {
	home, proj := t.TempDir(), trustProject(t)
	writeFile(t, filepath.Join(proj, ".gobble", "settings.yaml"), "steeringMode: all\n")
	t.Chdir(proj)
	if r := mustRun(t, home, "config", "get", "steeringMode"); strings.TrimSpace(r.stdout) != "one-at-a-time" {
		t.Fatalf("untrusted: %q", r.stdout)
	}
	mustRun(t, home, "trust")
	if r := mustRun(t, home, "config", "get", "steeringMode"); strings.TrimSpace(r.stdout) != "all" {
		t.Fatalf("trusted: %q", r.stdout)
	}
}

// A trusted project's personality messages and prompts reach its session's
// settings, checked as the user's are, a bad file left out; an untrusted
// project's do not (decision 8).
func TestProjectOverridesReachSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	proj := trustProject(t)
	dir := filepath.Join(proj, ".gobble", "personalities")
	writeFile(t, filepath.Join(dir, "dry", "messages.md"), "## mode.current\n\n- Project mode: {mode}\n")
	writeFile(t, filepath.Join(dir, "prompts", "tool.read.md"), readDescription)
	writeFile(t, filepath.Join(dir, "prompts", "rules.md"), "Project rules.\n")
	writeFile(t, filepath.Join(dir, "prompts", "bogus.md"), "x\n")
	var errw bytes.Buffer
	out := &Output{out: &bytes.Buffer{}, err: &errw}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, store: session.NewMemoryStore()}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	if ss := e.agentOptions().Settings(proj); ss.Trust != (permission.TrustDecision{Source: permission.SourceAsk}) || ss.Project != nil || ss.ProjectRoot != proj {
		t.Fatalf("untrusted: %+v %+v", ss.Trust, ss.Project)
	}
	if _, err := (permission.TrustStore{Path: filepath.Join(home, permission.TrustFile)}).Save(t.Context(), proj, true); err != nil {
		t.Fatal(err)
	}
	ss := e.agentOptions().Settings(proj)
	if ss.Trust != (permission.TrustDecision{Trusted: true, Source: permission.SourceSaved, At: proj}) || ss.Project == nil {
		t.Fatalf("trusted: %+v %+v", ss.Trust, ss.Project)
	}
	p := ss.Project
	if p.RulesSection != "Project rules." || !strings.HasPrefix(p.Descriptions["read"], "Read a text file's lines, a page at a time.") ||
		strings.Contains(p.Descriptions["read"], "{{") || len(p.Descriptions) != 1 {
		t.Fatalf("prompts: rules %q, descriptions %v", p.RulesSection, p.Descriptions)
	}
	v, err := p.Personalities.New(personality.Options{Name: personality.Dry})
	if err != nil || v.Text(personality.ModeCurrent, personality.Args{"mode": "plan"}) != "Project mode: plan" {
		t.Fatalf("dry: %v", err)
	}
}

// gobble personality check checks a trusted project's personality files,
// and notes an untrusted project's as held back (decision 8).
func TestPersonalityCheckProject(t *testing.T) {
	home, proj := t.TempDir(), trustProject(t)
	writeFile(t, filepath.Join(proj, ".gobble", "personalities", "dry", "personality.md"), "---\nname: dry\nlevel: 3\n---\nThe project's dry.\n")
	t.Chdir(proj)
	if r := runIn(t, home, "personality", "check"); r.code != ExitOK || !strings.Contains(r.stdout, "the project is not trusted") {
		t.Fatalf("untrusted: exit %d, %q", r.code, r.stdout)
	}
	mustRun(t, home, "trust")
	if r := runIn(t, home, "personality", "check"); r.code != ExitFailure || !strings.Contains(r.stdout, "a project may not change a personality's brief") {
		t.Fatalf("trusted: exit %d, %q", r.code, r.stdout)
	}
}

// fakeRaw makes the question's raw mode a no-op, for a stdin that is a
// file.
func fakeRaw(t *testing.T) {
	t.Helper()
	old := trustRaw
	trustRaw = func(*runEnv) (func() error, error) { return func() error { return nil }, nil }
	t.Cleanup(func() { trustRaw = old })
}

// The line view's question, before the session starts: each answer, a
// saved one in trust.json, a run's for the run alone, a cancel none; it is
// asked once a run (decision 5).
func TestTrustQuestion(t *testing.T) {
	fakeRaw(t)
	for _, c := range []struct {
		key     string
		trusted bool
		source  permission.TrustSource
		parent  bool // saved for the parent folder
		saved   bool
	}{
		{"1", true, permission.SourceSaved, false, true},
		{"2", true, permission.SourceSaved, true, true},
		{"3", true, permission.SourceAnswer, false, false},
		{"4", false, permission.SourceSaved, false, true},
		{"5", false, permission.SourceAnswer, false, false},
		{"zc", false, permission.SourceAsk, false, false},
	} {
		home := t.TempDir()
		t.Setenv("GOBBLE_HOME", home)
		proj := trustProject(t)
		parent := filepath.Dir(proj)
		var errw bytes.Buffer
		out := &Output{out: &bytes.Buffer{}, err: &errw}
		e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, stdin: stdinFile(t, c.key)}
		e.decideTrustHere(proj)
		e.decideTrustHere(proj)
		_, d := e.projectTrust(proj)
		if d.Trusted != c.trusted || d.Source != c.source {
			t.Errorf("%q: %+v", c.key, d)
		}
		// The question, drawn by ask as 0029-MADR D2 lays it out, once.
		s := shownOn(t, errw.String(), editor.MaxWidth)
		menu := trustMenu(proj, numbered("Trust", "Trust the parent folder ("+parent+")", "Trust for this run",
			"Do not trust", "Do not trust for this run")...)
		if !strings.Contains(errw.String(), "Trusting "+proj+" lets gobble load the project's .gobble settings, SYSTEM.md, skills, prompts and personalities.") ||
			drawnRows(s, menu, editor.MaxWidth) != 1 {
			t.Errorf("%q: the question is not drawn once as ask lays it out at %d columns:\n%s", c.key, editor.MaxWidth, s)
		}
		m := saved(t, home)
		at := proj
		if c.parent {
			at = parent
		}
		if v, ok := m[at]; ok != c.saved || len(m) > 1 || (ok && v != c.trusted) {
			t.Errorf("%q: trust.json %v", c.key, m)
		}
		next := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}}
		if _, d := next.projectTrust(proj); !c.saved && d != (permission.TrustDecision{Source: permission.SourceAsk}) {
			t.Errorf("%q: the next run's decision %+v", c.key, d)
		}
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	}
}

// With home the parent folder, it is not offered, and the line keeps
// 0008-MADR's keys, no kind repeating (decisions 3 and 6).
func TestTrustQuestionNoParent(t *testing.T) {
	fakeRaw(t)
	t.Setenv("GOBBLE_HOME", t.TempDir())
	proj := trustProject(t)
	setHome(t, filepath.Dir(proj))
	var errw bytes.Buffer
	out := &Output{out: &bytes.Buffer{}, err: &errw}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, stdin: stdinFile(t, "y")}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	e.decideTrustHere(proj)
	menu := trustMenu(proj, ask.Answer{Key: "y", Label: "allow once"}, ask.Answer{Key: "a", Label: "always"},
		ask.Answer{Key: "n", Label: "deny"}, ask.Answer{Key: "N", Label: "never"})
	if s := shownOn(t, errw.String(), editor.MaxWidth); drawnRows(s, menu, editor.MaxWidth) != 1 || !strings.Contains(s, "> allow once") {
		t.Fatalf("stderr\n%s", s)
	}
	if _, d := e.projectTrust(proj); !d.Trusted || d.Source != permission.SourceAnswer {
		t.Fatalf("%+v", d)
	}
}

// The line view says once, before its welcome, that the project is not
// trusted, unless it just asked (decision 9); /trust's question is the
// numbered line, and its answer is saved (decisions 6 and 7).
func TestLineViewTrust(t *testing.T) {
	proj := trustProject(t)
	h := newModelHarness(t, llmtest.NewScript(), proj)
	h.start(Prompt{})
	h.press("/trust\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "[3] Do not trust") })
	h.press("1")
	// The confirmation is a reply, wrapped at the harness's 80 columns
	// (0028-PLAN P2, deviation 3).
	waitUntil(t, func() bool { return shownWrapped(h.stdout.String(), "Saved: "+proj+" is trusted.", 80) })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(h.stderr.String(), untrustedLine); n != 1 {
		t.Fatalf("told %d times:\n%s", n, h.stderr.String())
	}
	shown := shownOn(t, h.stderr.String(), editor.MaxWidth)
	menu := trustMenu(proj, numbered("Trust", "Trust the parent folder ("+filepath.Dir(proj)+")", "Do not trust")...)
	if drawnRows(shown, menu, editor.MaxWidth) != 1 || !strings.Contains(shown, "> Trust") {
		t.Fatalf("/trust's question is not drawn once as ask lays it out at %d columns, then answered:\n%s", editor.MaxWidth, shown)
	}
	if m := saved(t, os.Getenv("GOBBLE_HOME")); !m[proj] || len(m) != 1 {
		t.Fatalf("trust.json %v", m)
	}
	h2 := newModelHarness(t, llmtest.NewScript(), proj)
	h2.e.trust.asked = map[string]bool{proj: true}
	h2.start(Prompt{})
	h2.press("\x04")
	if err := h2.wait(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h2.stderr.String(), untrustedLine) {
		t.Fatal("told after it asked")
	}
}
