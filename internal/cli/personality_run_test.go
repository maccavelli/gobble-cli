package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/personality"
)

// briefIn reports whether the request's system prompt carries the named
// personality's brief, or, for boring, no personality section.
func briefIn(t *testing.T, req *llm.Request, name string) bool {
	t.Helper()
	var sys string
	if len(req.Messages) > 0 && req.Messages[0].Role == llm.RoleSystem {
		sys = req.Messages[0].Content[0].Text
	}
	v, err := personality.New(personality.Options{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	if name == personality.Boring {
		return !strings.Contains(sys, "<personality>")
	}
	return strings.Contains(sys, "<personality>\n"+v.Brief()+"\n</personality>")
}

// The setting, --personality and a user's own personality each give a
// session its voice; --personality takes boring's other names; a name no
// personality has is a usage error from the flag, and from the setting a
// warning and dry; a directory under personalities/ that is no personality
// is left out with a warning (0022-PLAN P3).
func TestPersonalityChosen(t *testing.T) {
	home := t.TempDir()
	t.Chdir(t.TempDir())
	writeFile(t, filepath.Join(home, "personalities", "pirate", "personality.md"), "---\nname: pirate\nlevel: 2\nsummary: Arr.\n---\nYou talk like a pirate.\n")
	writeFile(t, filepath.Join(home, "personalities", "broken", "personality.md"), "---\nname: other\nlevel: 2\n---\nx\n")
	for _, c := range []struct {
		name, setting string
		args          []string
		want          string
		warn          string
	}{
		{"default", "", nil, personality.Dry, ""},
		{"setting", "personality: snarky\n", nil, personality.Snarky, ""},
		{"flag", "personality: snarky\n", []string{"--personality", "dry"}, personality.Dry, ""},
		{"plain", "", []string{"--personality", "plain"}, personality.Boring, ""},
		{"unknown setting", "personality: zany\n", nil, personality.Dry, `personality: no personality is named \"zany\"; sessions use dry`},
	} {
		t.Run(c.name, func(t *testing.T) {
			writeFile(t, filepath.Join(home, "settings.yaml"), c.setting)
			model := llmtest.NewScript(llmtest.Text("fine"))
			withModel(t, model)
			r := mustRun(t, home, append(c.args, "-p", "hi")...)
			if !briefIn(t, model.Requests()[0], c.want) {
				t.Errorf("the session is not %s", c.want)
			}
			if c.warn != "" && !strings.Contains(r.stderr, c.warn) {
				t.Errorf("stderr %q lacks %q", r.stderr, c.warn)
			}
			if !strings.Contains(r.stderr, "personality broken (") {
				t.Errorf("stderr %q does not warn of personalities/broken", r.stderr)
			}
		})
	}
	writeFile(t, filepath.Join(home, "settings.yaml"), "")
	model := llmtest.NewScript(llmtest.Text("arr"))
	withModel(t, model)
	mustRun(t, home, "--personality", "pirate", "-p", "hi")
	if sys := model.Requests()[0].Messages[0].Content[0].Text; !strings.Contains(sys, "<personality>\nYou talk like a pirate.\n</personality>") {
		t.Fatalf("pirate's brief is not in:\n%s", sys)
	}
	r := runIn(t, home, "--personality", "zany", "-p", "hi")
	if r.code != ExitUsage || !strings.Contains(r.stderr, "--personality zany: no personality has that name; the personalities are boring, dry, snarky, bombastic, pirate") {
		t.Fatalf("--personality zany: %+v", r)
	}
}

// --personality completes the personalities, the user's too, each with its
// level and summary (0008-MADR D17; 0022-PLAN P3).
func TestPersonalityCandidates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	writeFile(t, filepath.Join(home, "personalities", "pirate", "personality.md"), "---\nname: pirate\nlevel: 2\nsummary: Arr.\n---\nYou talk like a pirate.\n")
	var got []string
	candidates, _ := completionRegistry()["personality"].Complete(t.Context(), "")
	for c := range candidates {
		got = append(got, c.Value+" "+c.Description)
	}
	if len(got) != 5 || got[0] != "boring 1: "+personality.BoringSummary || !strings.HasPrefix(got[3], "bombastic 5: ") || got[4] != "pirate 2: Arr." {
		t.Fatalf("candidates %q", got)
	}
}

// Print mode cannot ask the level-5 question: it answers no, says so on
// stderr, and the run is dry, its reply alone on stdout; the settings'
// yes asks nothing (0022-MADR D10).
func TestPrintPersonalityFallback(t *testing.T) {
	home := t.TempDir()
	t.Chdir(t.TempDir())
	for _, c := range []struct {
		setting, want string
		warns         bool
	}{
		{"personality: bombastic\n", personality.Dry, true},
		{"personality: bombastic\nbombasticAccepted: true\n", personality.Bombastic, false},
	} {
		file := filepath.Join(home, "settings.yaml")
		writeFile(t, file, c.setting)
		model := llmtest.NewScript(llmtest.Text("fine"))
		withModel(t, model)
		r := mustRun(t, home, "-p", "hi")
		const warning = "Warning: a level-5 personality asks first, and print mode cannot ask; this run uses dry. Choose it once in a session to agree.\n"
		if r.stdout != "fine\n" || strings.Contains(r.stderr, warning) != c.warns || !briefIn(t, model.Requests()[0], c.want) {
			t.Fatalf("%q: stdout %q, stderr %q", c.setting, r.stdout, r.stderr)
		}
		if b, err := os.ReadFile(file); err != nil || string(b) != c.setting {
			t.Fatalf("the settings changed: %q, %v", b, err)
		}
	}
}

// The line view asks the level-5 question as a permission prompt, once;
// its yes is saved to the global settings (0022-MADR D10).
func TestLinePersonalityQuestion(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("first"), llmtest.Text("second"))
	h := newModelHarness(t, model, t.TempDir())
	file := filepath.Join(os.Getenv("GOBBLE_HOME"), "settings.yaml")
	writeFile(t, file, "personality: bombastic\n")
	h.start(Prompt{})
	h.press("hello\r")
	const question = "Bombastic insults you and your decisions. It does what you choose, and it keeps gobble's house rules. Switch to it?" +
		"  [a] always  [n] deny  [c] cancel "
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), question) })
	h.press("a")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "first") })
	h.press("again\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "second") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(h.stderr.String(), question); n != 1 {
		t.Fatalf("asked %d times", n)
	}
	b, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(b), "bombasticAccepted: true") {
		t.Fatalf("settings %q, %v", b, err)
	}
	if !briefIn(t, model.Requests()[0], personality.Bombastic) {
		t.Fatal("the session is not bombastic")
	}
}

// The CLI's own lines follow the session's _gobble/personality, in the
// line view and in print mode (0022-PLAN P3 step 7).
func TestCLIVoiceFollows(t *testing.T) {
	t.Setenv("GOBBLE_HOME", t.TempDir())
	out := &Output{}
	e := &runEnv{out: out, logs: &lazyLog{out: out}}
	ls := &lineSession{e: e}
	sink := newPrintSink(e, &ChatCmd{}, t.TempDir())
	for _, name := range []string{personality.Snarky, personality.Boring, personality.Bombastic} {
		u := acpclient.Update{Kind: acpclient.KindPersonality, Personality: acpclient.Personality{Name: name}}
		ls.onUpdate(u)
		sink.on(u)
		if ls.spoken().Name() != name || sink.voice.Name() != name {
			t.Fatalf("%s: the line view speaks %s, print mode %s", name, ls.spoken().Name(), sink.voice.Name())
		}
	}
}
