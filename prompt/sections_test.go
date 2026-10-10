package prompt

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// copyTree copies the fixture project into a new directory, so the walk to
// the filesystem root meets no file of this repository, and returns it.
func copyTree(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o700)
		}
		b, err := os.ReadFile(p) //nolint:gosec // G304: the test's own fixture
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

var builtins = []string{"read", "write", "edit", "grep", "find", "tree", "move", "delete", "copy", "mkdir", "todo", "bash"}

// The system prompt of a fixture project matches system.golden (0005-PLAN
// F3's accept line): the config directory's AGENTS.md, then the repository
// root's AGENTS.override.md, which beats its AGENTS.md, then a
// sub-directory's CLAUDE.md, for a cwd below it with none of its own.
// PROMPT_UPDATE=1 rewrites the golden file.
func TestGoldenSystemPrompt(t *testing.T) {
	root := copyTree(t, filepath.Join("testdata", "project"))
	cwd := filepath.Join(root, "repo", "sub", "deep")
	files, diags := ContextFiles(filepath.Join(root, "config"), cwd)
	if len(diags) != 0 {
		t.Fatalf("diagnostics %+v", diags)
	}
	for _, f := range files {
		if !strings.HasPrefix(f.Path, root) {
			t.Fatalf("a context file outside the fixture: %s", f.Path)
		}
	}
	got := Text(Build(Options{Tools: builtins, ContextFiles: files, Cwd: cwd, GOOS: "linux"}))
	// The fixture's root is <root>, and Windows separators are /, so the
	// golden file holds on every system.
	got = filepath.ToSlash(strings.ReplaceAll(strings.ReplaceAll(got, root, "<root>"), filepath.ToSlash(root), "<root>"))
	golden := filepath.Join("testdata", "system.golden")
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
		t.Fatalf("system prompt:\n%s\n--- want ---\n%s", got, want)
	}
}

// SYSTEM.md replaces the preamble and leaves out tools and rules;
// APPEND_SYSTEM.md is the addendum; every section but the preamble is
// tag-wrapped (Pi's system-prompt.ts:143-178).
func TestBuildCustomAndAppend(t *testing.T) {
	names := func(ss []Section) string {
		var out []string
		for _, s := range ss {
			out = append(out, s.Name)
		}
		return strings.Join(out, ",")
	}
	def := Build(Options{Tools: builtins, Cwd: `C:\w`, GOOS: "windows"})
	if names(def) != "preamble,tools,rules,cwd" || def[0].Text != Preamble || def[len(def)-1].Text != "<cwd>\nC:/w\n</cwd>" {
		t.Fatalf("default sections %s: %+v", names(def), def)
	}
	custom := Build(Options{Custom: "Mine.", Append: "More.", Tools: builtins, Cwd: "/w", ContextFiles: []File{{Path: "/w/AGENTS.md", Content: "x"}}})
	if names(custom) != "preamble,addendum,project_context,cwd" || custom[0].Text != "Mine." || custom[1].Text != "<addendum>\nMore.\n</addendum>" {
		t.Fatalf("custom sections %s: %+v", names(custom), custom)
	}
	// The shell rule, as Pi words it, only with no search tool.
	if r := Build(Options{Tools: []string{"bash", "powershell"}, GOOS: "windows"})[2].Text; !strings.Contains(r, "- Use bash or PowerShell for file operations like listing, searching, and finding files") {
		t.Fatalf("shell rule missing: %s", r)
	}
	if r := def[2].Text; strings.Contains(r, "for file operations like") {
		t.Fatalf("shell rule with search tools: %s", r)
	}
	if tools := Build(Options{Tools: []string{"read"}, Deferred: []string{"code", "mcp-resources"}})[1].Text; !strings.Contains(tools, "- tool_search can find more tools for: code, mcp-resources.") || strings.Contains(tools, "edit it") {
		t.Fatalf("tools section %s", tools)
	}
	if got := Build(Options{}); names(got) != "preamble,rules,cwd" {
		t.Fatalf("no tools: %s", names(got))
	}
}

// Diff and Apply: a replaced section keeps its place, a new one goes last,
// a removed one is nil, and Apply of a Diff is the new sections (Pi's
// diffSystemPromptSections and its replay).
func TestDiffApply(t *testing.T) {
	prev := []Section{{"preamble", "p"}, {"rules", "r"}, {"cwd", "c"}}
	cur := []Section{{"preamble", "p"}, {"rules", "R"}, {"skills", "s"}}
	patch := Diff(prev, cur)
	var got []string
	for _, p := range patch {
		if p.Text == nil {
			got = append(got, p.Name+"=nil")
		} else {
			got = append(got, p.Name+"="+*p.Text)
		}
	}
	if strings.Join(got, ",") != "rules=R,skills=s,cwd=nil" {
		t.Fatalf("patch %v", got)
	}
	if after := Apply(prev, patch); !reflect.DeepEqual(after, cur) {
		t.Fatalf("applied %+v, want %+v", after, cur)
	}
	if prev[1].Text != "r" {
		t.Fatal("Apply changed its base")
	}
	if Diff(cur, cur) != nil {
		t.Fatal("no change gave a patch")
	}
	moved := Apply([]Section{{"a", "1"}, {"b", "2"}}, []Patch{{Name: "a", Text: new("3")}})
	if moved[0].Name != "a" || moved[0].Text != "3" {
		t.Fatalf("a replaced section moved: %+v", moved)
	}
}

// A baseline's and a patch's entry data read back as written, in order,
// with a removal's null kept.
func TestEntryData(t *testing.T) {
	base := []Section{{"preamble", "p"}, {"cwd", "<cwd>\n/w\n</cwd>"}}
	got, patch, err := ParseData(BaselineData(base))
	if err != nil || patch != nil || !reflect.DeepEqual(got, base) {
		t.Fatalf("baseline %+v, %+v, %v", got, patch, err)
	}
	want := []Patch{{Name: "rules", Text: new("r")}, {Name: "cwd"}}
	data := PatchData(want)
	if !strings.Contains(string(data), `{"name":"cwd","text":null}`) {
		t.Fatalf("patch data %s keeps no null", data)
	}
	b, p, err := ParseData(data)
	if err != nil || b != nil || !reflect.DeepEqual(p, want) {
		t.Fatalf("patch %+v, %+v, %v", b, p, err)
	}
	if _, _, err := ParseData([]byte(`{"other":1}`)); err == nil {
		t.Fatal("data with neither was read")
	}
}

// Update frames each change by name, as Pi does (ai/src/utils/text.ts:28-40).
func TestUpdate(t *testing.T) {
	got := Update([]Patch{{Name: "rules", Text: new("<rules>\nR\n</rules>")}, {Name: "cwd"}})
	want := "Updated system prompt section \"rules\":\n\n<rules>\nR\n</rules>\n\nRemoved system prompt section \"cwd\"."
	if got != want {
		t.Fatalf("update %q\nwant %q", got, want)
	}
}
