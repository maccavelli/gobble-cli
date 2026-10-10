package skill

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func names(skills []Skill) []string {
	var out []string
	for _, s := range skills {
		out = append(out, s.Name)
	}
	return out
}

func messages(diags []Diagnostic) []string {
	var out []string
	for _, d := range diags {
		out = append(out, filepath.Base(filepath.Dir(d.Path))+": "+d.Message)
	}
	return out
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A skills directory with a name collision warns and loads the first skill
// found (0005-PLAN F3's accept line). A skill with no description warns and
// is not loaded; a loose .md file at the top of a gobble-layout directory
// is a skill, named, with no name of its own, after its directory, as Pi
// names it (skills.ts:310-321); a .fdignore hides a directory.
func TestLoadCollision(t *testing.T) {
	first, second := filepath.Join("testdata", "first"), filepath.Join("testdata", "second")
	skills, diags := Load([]Source{{Dir: first}, {Dir: second}})
	if got := sortedNames(skills); !slices.Equal(got, []string{"deploy", "first", "review"}) {
		t.Fatalf("skills %v", got)
	}
	for _, s := range skills {
		if s.Name == "review" && !strings.HasPrefix(s.FilePath, first) {
			t.Fatalf("review loaded from %s, want the first directory's", s.FilePath)
		}
		// The frontmatter ends before the closing ---, as Pi slices it, so
		// the folded scalar has no last line break to keep.
		if s.Name == "first" && s.Description != "Keep notes as you work." {
			t.Fatalf("a folded block scalar read as %q", s.Description)
		}
	}
	got := messages(diags)
	want := []string{"broken: description is required", `review: name "review" collision`}
	if !slices.Equal(got, want) {
		t.Fatalf("diagnostics %q\nwant %q", got, want)
	}
	if last := diags[len(diags)-1]; !strings.HasPrefix(last.Winner, first) {
		t.Fatalf("the collision's winner %q", last.Winner)
	}
}

func sortedNames(skills []Skill) []string {
	out := names(skills)
	slices.Sort(out)
	return out
}

// Pi's name and description warnings, word for word (skills.ts:92-127);
// a skill with them still loads, and the name falls back to its directory.
func TestValidation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Bad--Name-", "SKILL.md"), "---\ndescription: d\n---\n")
	writeFile(t, filepath.Join(dir, "long", "SKILL.md"), "---\nname: "+strings.Repeat("a", 65)+"\ndescription: "+strings.Repeat("é", 1025)+"\n---\n")
	skills, diags := Load([]Source{{Dir: dir}})
	if len(skills) != 2 {
		t.Fatalf("skills %v", names(skills))
	}
	got := strings.Join(messages(diags), "\n")
	for _, want := range []string{
		"Bad--Name-: name contains invalid characters (must be lowercase a-z, 0-9, hyphens only)",
		"Bad--Name-: name must not start or end with a hyphen",
		"Bad--Name-: name must not contain consecutive hyphens",
		"long: description exceeds 1024 characters (1025)",
		"long: name exceeds 64 characters (65)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// The scan: a SKILL.md directory is not searched further; dotfiles and
// node_modules are skipped; a loose .md file without a description is
// skipped silently; in the .agents layout a loose file counts only below
// the top; one file reached twice through a link loads once.
func TestScan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a", "SKILL.md"), "---\ndescription: a\n---\n")
	writeFile(t, filepath.Join(dir, "a", "inner", "SKILL.md"), "---\nname: inner\ndescription: inner\n---\n")
	writeFile(t, filepath.Join(dir, ".dot", "SKILL.md"), "---\nname: dot\ndescription: dot\n---\n")
	writeFile(t, filepath.Join(dir, "node_modules", "m", "SKILL.md"), "---\nname: m\ndescription: m\n---\n")
	writeFile(t, filepath.Join(dir, "top.md"), "---\nname: top\ndescription: top\n---\n")
	writeFile(t, filepath.Join(dir, "plain.md"), "no frontmatter\n")
	writeFile(t, filepath.Join(dir, "group", "deep.md"), "---\ndescription: deep\n---\n")
	gobble, diags := Load([]Source{{Dir: dir}})
	if got := sortedNames(gobble); !slices.Equal(got, []string{"a", "top"}) || len(diags) != 0 {
		t.Fatalf("gobble layout: %v, %v", got, messages(diags))
	}
	agents, _ := Load([]Source{{Dir: dir, Agents: true}})
	if got := sortedNames(agents); !slices.Equal(got, []string{"a", "group"}) {
		t.Fatalf(".agents layout: %v", got)
	}
	if err := os.Symlink(filepath.Join(dir, "a"), filepath.Join(dir, "alias")); err != nil {
		t.Logf("no link test here: %v", err)
		return
	}
	linked, diags := Load([]Source{{Dir: dir}})
	if got := sortedNames(linked); !slices.Equal(got, []string{"a", "top"}) || len(diags) != 0 {
		t.Fatalf("with a link: %v, %v", got, messages(diags))
	}
	// A second source that reaches the same files through a link adds
	// nothing, and no collision: Pi skips a real path already loaded.
	other := filepath.Join(t.TempDir(), "other")
	if err := os.Symlink(dir, other); err != nil {
		t.Fatal(err)
	}
	twice, diags := Load([]Source{{Dir: dir}, {Dir: other}})
	if got := sortedNames(twice); !slices.Equal(got, []string{"a", "top"}) || len(diags) != 0 {
		t.Fatalf("two sources, one place: %v, %v", got, messages(diags))
	}
}

// Format is Pi's text, with hidden skills left out and XML escaped.
func TestFormat(t *testing.T) {
	skills := []Skill{
		{Name: "review", Description: `Check "it" & <that>`, FilePath: "/s/review/SKILL.md"},
		{Name: "deploy", Description: "d", FilePath: "/s/deploy/SKILL.md", DisableModelInvocation: true},
	}
	want := "The following skills provide specialized instructions for specific tasks.\n" +
		"Use the read tool to load a skill's file when the task matches its description.\n" +
		"When a skill file references a relative path, resolve it against the skill directory (parent of SKILL.md / dirname of the path) and use that absolute path in tool commands.\n" +
		"\n<available_skills>\n  <skill>\n    <name>review</name>\n    <description>Check &quot;it&quot; &amp; &lt;that&gt;</description>\n" +
		"    <location>/s/review/SKILL.md</location>\n  </skill>\n</available_skills>"
	if got := Format(skills); got != want {
		t.Fatalf("format:\n%s\nwant:\n%s", got, want)
	}
	if Format(skills[1:]) != "" {
		t.Fatal("only hidden skills gave text")
	}
}

// Expand is Pi's /skill:<name> [args]: the body in a <skill> block, the
// arguments after it; a hidden skill expands; an unknown one is left.
func TestExpand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deploy", "SKILL.md")
	writeFile(t, path, "---\nname: deploy\ndescription: d\n---\n\nShip it.\n\n")
	skills := []Skill{{Name: "deploy", FilePath: path, BaseDir: filepath.Dir(path), DisableModelInvocation: true}}
	block := `<skill name="deploy" location="` + path + `">` + "\nReferences are relative to " + filepath.Dir(path) + ".\n\nShip it.\n</skill>"
	if got, ok := Expand("/skill:deploy", skills); !ok || got != block {
		t.Fatalf("no args: %q, %v", got, ok)
	}
	if got, ok := Expand("/skill:deploy  to prod ", skills); !ok || got != block+"\n\nto prod" {
		t.Fatalf("args: %q, %v", got, ok)
	}
	for _, text := range []string{"/skill:nope x", "skill:deploy", "/deploy"} {
		if got, ok := Expand(text, skills); ok || got != text {
			t.Errorf("%q expanded to %q", text, got)
		}
	}
}
