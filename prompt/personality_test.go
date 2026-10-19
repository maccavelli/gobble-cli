package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/personality"
)

func sectionNames(ss []Section) string {
	var out []string
	for _, s := range ss {
		out = append(out, s.Name)
	}
	return strings.Join(out, ",")
}

// The personality and its house rules follow the preamble, a custom one
// included, tag-wrapped; with neither, the prompt is as it was
// (0022-MADR D3, D4).
func TestBuildPersonality(t *testing.T) {
	o := Options{Tools: builtins, Cwd: "/w", GOOS: "linux", Personality: "BRIEF", PersonalityRules: "RULES"}
	got := Build(o)
	if sectionNames(got) != "preamble,personality,personality_rules,tools,rules,cwd" ||
		got[1].Text != "<personality>\nBRIEF\n</personality>" || got[2].Text != "<personality_rules>\nRULES\n</personality_rules>" {
		t.Fatalf("sections %s: %+v", sectionNames(got), got)
	}
	o.Custom = "Mine."
	if got := sectionNames(Build(o)); got != "preamble,personality,personality_rules,cwd" {
		t.Fatalf("with SYSTEM.md, sections %s", got)
	}
	if got := sectionNames(Build(Options{Tools: builtins, Cwd: "/w", GOOS: "linux"})); got != "preamble,tools,rules,cwd" {
		t.Fatalf("boring's sections %s", got)
	}
}

// The fixture project's system prompt under dry matches
// system_dry.golden: system.golden's, with dry's brief and the house
// rules after the preamble. PROMPT_UPDATE=1 rewrites it.
func TestGoldenSystemPromptDry(t *testing.T) {
	dry, err := personality.New(personality.Options{Name: personality.Dry})
	if err != nil {
		t.Fatal(err)
	}
	root := copyTree(t, filepath.Join("testdata", "project"))
	cwd := filepath.Join(root, "repo", "sub", "deep")
	files, _ := ContextFiles(filepath.Join(root, "config"), cwd)
	got := Text(Build(Options{Tools: builtins, ContextFiles: files, Cwd: cwd, GOOS: "linux", Personality: dry.Brief(), PersonalityRules: dry.Rules()}))
	got = filepath.ToSlash(strings.ReplaceAll(strings.ReplaceAll(got, root, "<root>"), filepath.ToSlash(root), "<root>"))
	golden := filepath.Join("testdata", "system_dry.golden")
	if os.Getenv("PROMPT_UPDATE") == "1" {
		if err := os.WriteFile(golden, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("system prompt under dry:\n%s\n--- want ---\n%s", got, want)
	}
	plain, err := os.ReadFile(filepath.Join("testdata", "system.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, Preamble+"\n\n<personality>\n# Dry\n") || !strings.HasSuffix(got, strings.TrimPrefix(string(plain), Preamble)) {
		t.Fatal("the dry prompt is not boring's with the two sections after the preamble")
	}
}
