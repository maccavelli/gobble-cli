package acpserver

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// skillHome is a config directory with a review skill, a home with a
// deploy skill in .agents/skills, and a cwd that is a repository root, so
// the walk for project skill directories stops there.
func skillHome(t *testing.T) (cfg, home, cwd string) {
	t.Helper()
	cfg, home, cwd = t.TempDir(), t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(cfg, "skills", "review", "SKILL.md"), "---\nname: review\ndescription: Review a change.\n---\nCheck the diff.\n")
	writeFile(t, filepath.Join(home, ".agents", "skills", "deploy", "SKILL.md"), "---\nname: deploy\ndescription: Deploy it.\n---\nShip.\n")
	if err := os.Mkdir(filepath.Join(cwd, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	return cfg, home, cwd
}

// The skills section lists the config directory's and ~/.agents/skills'
// skills, in Pi's format; with NoSkills there is none (0005-PLAN F3-2).
func TestSkillsSection(t *testing.T) {
	cfg, home, cwd := skillHome(t)
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"))
	var logs bytes.Buffer
	c := promptAgent(t, t.TempDir(), model, PromptOptions{ConfigDir: cfg, Home: home}, &logs)
	if _, err := prompt(t, c, newSession(t, c, cwd, nil).SessionId, "hi"); err != nil {
		t.Fatal(err)
	}
	sys := systemOf(model.Requests()[0])
	for _, want := range []string{"<skills>\nThe following skills provide specialized instructions", "<name>review</name>", "<name>deploy</name>",
		"<location>" + filepath.Join(cfg, "skills", "review", "SKILL.md") + "</location>"} {
		if !strings.Contains(sys, want) {
			t.Errorf("the system prompt lacks %q:\n%s", want, sys)
		}
	}
	c2 := promptAgent(t, t.TempDir(), model, PromptOptions{ConfigDir: cfg, Home: home, NoSkills: true}, &logs)
	if _, err := prompt(t, c2, newSession(t, c2, cwd, nil).SessionId, "hi"); err != nil {
		t.Fatal(err)
	}
	if sys := systemOf(model.Requests()[1]); strings.Contains(sys, "<skills>") {
		t.Fatalf("NoSkills, and a skills section:\n%s", sys)
	}
}

// /skill:<name> args sends Pi's <skill> block with the arguments after it,
// as the user's message; an unknown skill is sent as typed.
func TestSkillCommand(t *testing.T) {
	cfg, home, cwd := skillHome(t)
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"))
	var logs bytes.Buffer
	c := promptAgent(t, t.TempDir(), model, PromptOptions{ConfigDir: cfg, Home: home}, &logs)
	id := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, id, "/skill:review focus on tests"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg, "skills", "review", "SKILL.md")
	want := `<skill name="review" location="` + path + `">` + "\nReferences are relative to " + filepath.Dir(path) +
		".\n\nCheck the diff.\n</skill>\n\nfocus on tests"
	if got := lastUser(model.Requests()[0]); got != want {
		t.Fatalf("the user's message %q\nwant %q", got, want)
	}
	if _, err := prompt(t, c, id, "/skill:nope x"); err != nil {
		t.Fatal(err)
	}
	if got := lastUser(model.Requests()[1]); got != "/skill:nope x" {
		t.Fatalf("an unknown skill sent %q", got)
	}
}

func lastUser(req *llm.Request) string {
	users := userTextsOf(req)
	if len(users) == 0 {
		return ""
	}
	return users[len(users)-1]
}

// A project's own .agents/skills and .gobble/skills are not loaded before
// F6: each is noted once, and its skill is not in the prompt.
func TestProjectSkillsWait(t *testing.T) {
	cfg, home, cwd := skillHome(t)
	writeFile(t, filepath.Join(cwd, ".agents", "skills", "local", "SKILL.md"), "---\nname: local\ndescription: Local.\n---\n")
	writeFile(t, filepath.Join(cwd, ".gobble", "skills", "mine", "SKILL.md"), "---\nname: mine\ndescription: Mine.\n---\n")
	model := llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"))
	var logs bytes.Buffer
	c := promptAgent(t, t.TempDir(), model, PromptOptions{ConfigDir: cfg, Home: home}, &logs)
	id := newSession(t, c, cwd, nil).SessionId
	for _, p := range []string{"one", "two"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	if sys := systemOf(model.Requests()[0]); strings.Contains(sys, "<name>local</name>") || strings.Contains(sys, "<name>mine</name>") {
		t.Fatalf("a project skill loaded:\n%s", sys)
	}
	for _, dir := range []string{filepath.Join(cwd, ".agents", "skills"), filepath.Join(cwd, ".gobble", "skills")} {
		if n := strings.Count(logs.String(), "file="+dir+" "); n != 1 {
			t.Errorf("%s noted %d times:\n%s", dir, n, logs.String())
		}
	}
}
