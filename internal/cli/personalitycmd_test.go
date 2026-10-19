package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/llmtest"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// readDescription is a description template for read that 0012-MADR D5's
// checks pass, with a limit filled from the options.
const readDescription = `Read a text file's lines, a page at a time.

## Use when
- You need a file's contents before you change it.
- Not for searching many files: use grep.

## Returns
- At most {{.MaxLines}} numbered lines a call.
`

// The user's prompt files: the tools and rules sections and a tool's
// description, each checked; one with an id gobble does not have, or one
// that fails its check, is refused, naming the file and the rule; a
// compaction prompt is not among the ids until 0005-PLAN F5-4 (0022-MADR
// D15; 0022-PLAN P5's deviation 1).
func TestLoadPrompts(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"tools.md":              "Use the tools well.\n",
		"rules.md":              "Be careful.\n",
		"tool.read.md":          readDescription,
		"tool.grep.md":          "Search.\n",
		"compaction.summary.md": "Summarise.\n",
		"notes.txt":             "x",
	}
	for name, text := range files {
		writeFile(t, filepath.Join(dir, name), text)
	}
	got, errs := loadPrompts(dir, builtin.Options{})
	if got.sections[promptTools] != "Use the tools well." || got.sections[promptRules] != "Be careful." || got.descriptions["read"] != readDescription {
		t.Fatalf("overrides %+v", got)
	}
	if len(errs) != 3 {
		t.Fatalf("errors %v, want grep's, the compaction prompt's and notes.txt's", errs)
	}
	for _, want := range []string{
		"prompt " + filepath.Join(dir, "tool.grep.md") + ": 0012-MADR D5: ",
		"prompt " + filepath.Join(dir, "compaction.summary.md") + ": no prompt has this name",
		"prompt " + filepath.Join(dir, "notes.txt") + ": no prompt has this name",
	} {
		if !strings.Contains(joinErrs(errs), want) {
			t.Errorf("no error %q in %v", want, errs)
		}
	}
	for _, c := range []struct{ name, text, why string }{
		{"rules.md", "  \n", "it is empty"},
		{"tools.md", "Fine.\n</tools>\nNot fine.\n", "closes the section's tag"},
	} {
		d := t.TempDir()
		writeFile(t, filepath.Join(d, c.name), c.text)
		if _, errs := loadPrompts(d, builtin.Options{}); len(errs) != 1 || !strings.Contains(errs[0].Error(), c.why) {
			t.Errorf("%s: %v, want %q", c.name, errs, c.why)
		}
	}
}

func joinErrs(errs []error) string {
	var b strings.Builder
	for _, err := range errs {
		b.WriteString(err.Error() + "\n")
	}
	return b.String()
}

// The user's files reach the model: the tools and rules sections, and
// read's description rendered with its limits; a project's personalities
// wait for trust, and change nothing (0022-MADR D15).
func TestPromptOverridesReachTheModel(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Chdir(project)
	prompts := filepath.Join(home, "personalities", "prompts")
	writeFile(t, filepath.Join(prompts, "tools.md"), "Use the tools well.\n")
	writeFile(t, filepath.Join(prompts, "rules.md"), "Be careful.\n")
	writeFile(t, filepath.Join(prompts, "tool.read.md"), readDescription)
	writeFile(t, filepath.Join(project, ".gobble", "personalities", "dry", "personality.md"), "---\nname: dry\nlevel: 3\n---\nThe project's dry.\n")
	model := llmtest.NewScript(llmtest.Text("fine"))
	withModel(t, model)
	r := mustRun(t, home, "-p", "hi")
	req := model.Requests()[0]
	sys := req.Messages[0].Content[0].Text
	if !strings.Contains(sys, "<tools>\nUse the tools well.\n</tools>") || !strings.Contains(sys, "<rules>\nBe careful.\n</rules>") {
		t.Fatalf("the sections are not the user's:\n%s", sys)
	}
	if strings.Contains(sys, "The project's dry.") {
		t.Fatal("a project's personality reached the prompt")
	}
	var tools []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(req.Tools, &tools); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tl := range tools {
		if tl.Name == "read" {
			found = true
			if !strings.HasPrefix(tl.Description, "Read a text file's lines, a page at a time.") || strings.Contains(tl.Description, "{{") ||
				!strings.Contains(tl.Description, "- At most 2000 numbered lines a call.") {
				t.Fatalf("read's description %q", tl.Description)
			}
		}
	}
	if !found {
		t.Fatal("no read tool")
	}
	if !strings.Contains(r.stderr, filepath.Join(project, ".gobble", "personalities")) || !strings.Contains(r.stderr, sysprompt.TrustNote) {
		t.Fatalf("stderr %q does not say the project's personalities wait", r.stderr)
	}
}

// gobble personality list, show, check and path (0022-MADR D15).
func TestPersonalityCommand(t *testing.T) {
	home := t.TempDir()
	t.Chdir(t.TempDir())
	dir := filepath.Join(home, "personalities")
	writeFile(t, filepath.Join(dir, "dry", "personality.md"), "---\nname: dry\nlevel: 3\nsummary: Drier.\n---\nYou are very dry.\n")
	writeFile(t, filepath.Join(dir, "dry", "messages.md"), "## retry.offline\n\n- The network has stepped out, {{politely}}.\n")
	writeFile(t, filepath.Join(dir, "pirate", "personality.md"), "---\nname: pirate\nlevel: 2\nsummary: Arr.\n---\nYou talk like a pirate.\n")

	list := mustRun(t, home, "personality", "list").stdout
	for _, want := range []string{"NAME", "boring", "built-in", "dry", "built-in, changed by " + filepath.Join(dir, "dry"), "Drier.", "pirate", filepath.Join(dir, "pirate"), "Arr."} {
		if !strings.Contains(list, want) {
			t.Errorf("list lacks %q:\n%s", want, list)
		}
	}
	if got := mustRun(t, home, "personality", "show", "dry").stdout; got != "dry, level 3: Drier.\nsource: built-in, changed by "+filepath.Join(dir, "dry")+"\n\nYou are very dry.\n" {
		t.Errorf("show dry %q", got)
	}
	if got := mustRun(t, home, "personality", "show", "plain").stdout; !strings.Contains(got, "boring has no brief") {
		t.Errorf("show plain %q", got)
	}
	if got := mustRun(t, home, "personality", "show", "dry", "retry.offline").stdout; got != "- The network has stepped out, {politely}.\n" {
		t.Errorf("show dry retry.offline %q", got)
	}
	if got := mustRun(t, home, "personality", "show", "pirate", "retry.attempt").stdout; !strings.Contains(got, "{attempt}") {
		t.Errorf("pirate's fallback to boring %q", got)
	}
	for _, args := range [][]string{{"show", "zany"}, {"show", "dry", "no.such.key"}} {
		if r := runIn(t, home, append([]string{"personality"}, args...)...); r.code != ExitUsage {
			t.Errorf("%v: %+v", args, r)
		}
	}
	if got := mustRun(t, home, "personality", "path").stdout; got != dir+"\n" {
		t.Errorf("path %q", got)
	}
	if r := runIn(t, home, "personality", "check"); r.code != ExitOK || !strings.HasPrefix(r.stdout, "0 problems; 2 personalities yours or changed, 0 prompts") {
		t.Errorf("check, clean: %+v", r)
	}
	writeFile(t, filepath.Join(dir, "snarky", "messages.md"), "## retry.empty_reply\n\n- Again.\n")
	writeFile(t, filepath.Join(dir, "prompts", "rules.md"), "\n")
	r := runIn(t, home, "personality", "check")
	if r.code != ExitFailure || !strings.Contains(r.stdout, "personality snarky ("+filepath.Join(dir, "snarky")+")") ||
		!strings.Contains(r.stdout, "prompt "+filepath.Join(dir, "prompts", "rules.md")+": it is empty") || !strings.Contains(r.stdout, "2 problems;") {
		t.Errorf("check, two problems: %+v", r)
	}
}

// gobble personality reset removes the named files, and only them, after a
// yes; a no removes nothing (0022-MADR D15).
func TestPersonalityReset(t *testing.T) {
	home := t.TempDir()
	t.Chdir(t.TempDir())
	dir := filepath.Join(home, "personalities")
	var answer bool
	var asked []string
	old := confirm
	confirm = func(_ *runEnv, q string) (bool, error) { asked = append(asked, q); return answer, nil }
	t.Cleanup(func() { confirm = old })
	keep := []string{filepath.Join(dir, "dry", "notes.txt"), filepath.Join(dir, "prompts", "rules.md"), filepath.Join(dir, "pirate", "personality.md")}
	gone := []string{filepath.Join(dir, "dry", "personality.md"), filepath.Join(dir, "dry", "messages.md")}
	for _, f := range append(append([]string{}, keep...), gone...) {
		writeFile(t, f, "x\n")
	}
	writeFile(t, filepath.Join(dir, "prompts", "tools.md"), "Use the tools.\n")
	exists := func(f string) bool { _, err := os.Stat(f); return err == nil }

	if r := mustRun(t, home, "personality", "reset", "dry"); r.stdout != "Nothing removed.\n" || !exists(gone[0]) || !exists(gone[1]) {
		t.Fatalf("a no removed something: %+v", r)
	}
	answer = true
	mustRun(t, home, "personality", "reset", "dry")
	if exists(gone[0]) || exists(gone[1]) || !exists(keep[0]) {
		t.Fatal("reset dry did not remove exactly its two files")
	}
	mustRun(t, home, "personality", "reset", "prompts/tools")
	if exists(filepath.Join(dir, "prompts", "tools.md")) || !exists(keep[1]) || !exists(keep[2]) {
		t.Fatal("reset prompts/tools did not remove exactly tools.md")
	}
	if want := "Remove " + gone[0] + " and " + gone[1] + "?"; len(asked) != 3 || asked[0] != want || asked[1] != want {
		t.Fatalf("asked %q", asked)
	}
	if r := runIn(t, home, "personality", "reset", "snarky"); r.code != ExitFailure || !strings.Contains(r.stderr, "nothing to reset") {
		t.Fatalf("reset snarky: %+v", r)
	}
	for _, target := range []string{"prompts/compaction.summary", "../x", "prompts"} {
		if r := runIn(t, home, "personality", "reset", target); r.code != ExitUsage {
			t.Errorf("reset %s: %+v", target, r)
		}
	}
	if len(asked) != 3 {
		t.Fatalf("a refused reset asked: %q", asked)
	}
}
