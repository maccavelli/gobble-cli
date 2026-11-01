package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestImportRules(t *testing.T) {
	real := loadReal(t)

	cases := []struct {
		name string
		rule int
		pkgs []modPkg
		want string
	}{
		{name: "rule 1 module", rule: 1, want: ""},
		{name: "rule 1 agent lipgloss", rule: 1, want: "charm.land/lipgloss/v2", pkgs: []modPkg{{
			ImportPath: modulePath + "/agent",
			Imports:    []string{"charm.land/lipgloss/v2"},
		}}},
		{name: "rule 1 provider sdk allowed only in llm/provider", rule: 1, pkgs: []modPkg{{
			ImportPath: modulePath + "/llm/provider",
			Imports:    []string{"github.com/maccavelli/go-llmprovider-sdk/llmprovider"},
		}}},
		{name: "rule 1 agent provider sdk", rule: 1, want: "go-llmprovider-sdk", pkgs: []modPkg{{
			ImportPath: modulePath + "/agent",
			Imports:    []string{"github.com/maccavelli/go-llmprovider-sdk/llmprovider"},
		}}},
		{name: "rule 2 module", rule: 2},
		{name: "rule 2 root", rule: 2, want: "acp-go-sdk", pkgs: []modPkg{{
			ImportPath: modulePath,
			Imports:    []string{"github.com/maccavelli/acp-go-sdk2"},
		}}},
		{name: "rule 2 cli acp", rule: 2, want: "acp-go-sdk", pkgs: []modPkg{{
			ImportPath: modulePath + "/internal/cli",
			Imports:    []string{"github.com/maccavelli/acp-go-sdk2"},
		}}},
		{name: "rule 3 module", rule: 3},
		{name: "rule 3 agent mcp", rule: 3, want: "go-sdk", pkgs: []modPkg{{
			ImportPath: modulePath + "/agent",
			Imports:    []string{"github.com/modelcontextprotocol/go-sdk"},
		}}},
		{name: "rule 4 module", rule: 4},
		{name: "rule 4 provider sdk outside provider", rule: 4, want: "go-llmprovider-sdk", pkgs: []modPkg{{
			ImportPath: modulePath + "/llm",
			Imports:    []string{"github.com/maccavelli/go-llmprovider-sdk/llmprovider"},
		}}},
		{name: "rule 4 vendor sdk", rule: 4, want: "anthropic-sdk-go", pkgs: []modPkg{{
			ImportPath: modulePath + "/llm/provider",
			Imports:    []string{"github.com/anthropics/anthropic-sdk-go"},
		}}},
		{name: "rule 4 provider may import sdk", rule: 4, pkgs: []modPkg{{
			ImportPath: modulePath + "/llm/provider",
			Imports:    []string{"github.com/maccavelli/go-llmprovider-sdk/llmprovider"},
		}}},
		{name: "rule 5 module", rule: 5},
		{name: "rule 5 agent lipgloss", rule: 5, want: "charm.land/lipgloss/v2", pkgs: []modPkg{{
			ImportPath: modulePath + "/agent",
			Imports:    []string{"charm.land/lipgloss/v2"},
		}}},
		{name: "rule 5 tui may import charm", rule: 5, pkgs: []modPkg{{
			ImportPath: modulePath + "/internal/tui",
			Imports:    []string{"charm.land/lipgloss/v2"},
		}}},
		// 0032-MADR D1: internal/cli may import a go-tui-lib package whose
		// whole closure holds no Charm, and never internal/tui.
		{name: "rule 5 cli may import a charm-free go-tui-lib package", rule: 5, pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli", Imports: []string{tuiLib + "/glyph"}},
			{ImportPath: tuiLib + "/glyph", Deps: []string{"strings", "unicode/utf8", tuiLib + "/internal/enum"}},
		}},
		{name: "rule 5 cli go-tui-lib with charm in its closure", rule: 5, want: tuiLib + "/termsvc", pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli/render", Imports: []string{tuiLib + "/termsvc"}},
			{ImportPath: tuiLib + "/termsvc", Deps: []string{"strings", tuiLib + "/termcap", "charm.land/bubbletea/v2"}},
		}},
		{name: "rule 5 cli go-tui-lib with charm further down", rule: 5, want: "github.com/charmbracelet/colorprofile", pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli", Imports: []string{tuiLib + "/a"}},
			{ImportPath: tuiLib + "/a", Deps: []string{tuiLib + "/b", "github.com/charmbracelet/colorprofile"}},
		}},
		// 0035-MADR D13: go-tui-lib is its /v2 module; the v1 path is refused
		// everywhere, and tuitest's exception is the /v2 path's. The paths are
		// written out, not built from tuiLib, so a constant that drifts fails.
		{name: "rule 5 v1 path refused", rule: 5, want: "go-tui-lib's v1 path", pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli", Imports: []string{"github.com/maccavelli/go-tui-lib/glyph"}},
			{ImportPath: "github.com/maccavelli/go-tui-lib/glyph", Deps: []string{"strings"}},
		}},
		{name: "rule 5 v1 path refused in internal/tui", rule: 5, want: "go-tui-lib's v1 path", pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/tui", Imports: []string{"github.com/maccavelli/go-tui-lib/theme"}},
		}},
		{name: "rule 5 tuitest v2 in a test", rule: 5, pkgs: []modPkg{
			{ImportPath: modulePath + "/agent", TestImports: []string{"github.com/maccavelli/go-tui-lib/v2/tuitest"}},
		}},
		{name: "rule 5 tuitest v2 outside a test", rule: 5, want: "github.com/maccavelli/go-tui-lib/v2/tuitest", pkgs: []modPkg{
			{ImportPath: modulePath + "/agent", Imports: []string{"github.com/maccavelli/go-tui-lib/v2/tuitest"}},
		}},
		// 0032-MADR amendment A1: x/ansi, and it alone of Charm, may reach
		// internal/cli.
		{name: "rule 5 cli may import x/ansi", rule: 5, pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli/render", Imports: []string{xansi, xansi + "/parser"}},
		}},
		{name: "rule 5 cli go-tui-lib whose charm is x/ansi alone", rule: 5, pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli", Imports: []string{tuiLib + "/fuzzy"}},
			{ImportPath: tuiLib + "/fuzzy", Deps: []string{xansi, xansi + "/parser", "github.com/clipperhouse/uax29/v2", "unicode/utf8"}},
		}},
		{name: "rule 5 cli go-tui-lib with bubbletea beyond x/ansi", rule: 5, want: "charm.land/bubbletea/v2", pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli", Imports: []string{tuiLib + "/termsvc"}},
			{ImportPath: tuiLib + "/termsvc", Deps: []string{xansi, "charm.land/bubbletea/v2"}},
		}},
		{name: "rule 5 cli go-tui-lib with ultraviolet beyond x/ansi", rule: 5, want: "github.com/charmbracelet/ultraviolet", pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli", Imports: []string{tuiLib + "/termcap"}},
			{ImportPath: tuiLib + "/termcap", Deps: []string{xansi, "github.com/charmbracelet/ultraviolet"}},
		}},
		{name: "rule 5 cli may not import another charm x module", rule: 5, want: "github.com/charmbracelet/x/term", pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli", Imports: []string{"github.com/charmbracelet/x/term"}},
		}},
		{name: "rule 5 a core package may not import x/ansi", rule: 5, want: xansi, pkgs: []modPkg{
			{ImportPath: modulePath + "/agent", Imports: []string{xansi}},
		}},
		{name: "rule 1 a core package may not import x/ansi", rule: 1, want: xansi, pkgs: []modPkg{
			{ImportPath: modulePath + "/permission", Imports: []string{xansi}},
		}},
		{name: "rule 5 cli go-tui-lib whose closure is unknown", rule: 5, want: tuiLib + "/unseen", pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli", Imports: []string{tuiLib + "/unseen"}},
		}},
		{name: "rule 5 cli may not import internal/tui", rule: 5, want: modulePath + "/internal/tui", pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/cli/editor", Imports: []string{modulePath + "/internal/tui/widgets"}},
		}},
		{name: "rule 5 a core package may not import a charm-free go-tui-lib package", rule: 5, want: tuiLib + "/glyph", pkgs: []modPkg{
			{ImportPath: modulePath + "/agent", Imports: []string{tuiLib + "/glyph"}},
			{ImportPath: tuiLib + "/glyph", Deps: []string{"strings"}},
		}},
		{name: "rule 5 tui may import any of go-tui-lib", rule: 5, pkgs: []modPkg{
			{ImportPath: modulePath + "/internal/tui", Imports: []string{tuiLib + "/workspace"}},
			{ImportPath: tuiLib + "/workspace", Deps: []string{"charm.land/bubbletea/v2"}},
		}},
		{name: "rule 5 a test may import tuitest", rule: 5, pkgs: []modPkg{
			{ImportPath: modulePath + "/agent", TestImports: []string{tuiLib + "/tuitest"}},
		}},
		{name: "rule 6 module", rule: 6},
		{name: "rule 6 cli agent", rule: 6, want: modulePath + "/agent", pkgs: []modPkg{{
			ImportPath: modulePath + "/internal/cli",
			Imports:    []string{modulePath + "/agent"},
		}}},
		{name: "rule 6 tui agent", rule: 6, want: modulePath + "/agent", pkgs: []modPkg{{
			ImportPath: modulePath + "/internal/tui",
			Imports:    []string{modulePath + "/agent"},
		}}},
		{name: "rule 7 module", rule: 7},
		{name: "rule 7 stable imports exp", rule: 7, want: modulePath + "/exp/sandbox", pkgs: []modPkg{{
			ImportPath: modulePath + "/gobble",
			Imports:    []string{modulePath + "/exp/sandbox"},
		}}},
		{name: "rule 7 cmd may import exp", rule: 7, pkgs: []modPkg{{
			ImportPath: modulePath + "/cmd/gobble",
			Imports:    []string{modulePath + "/exp/sandbox"},
		}}},
		{name: "rule 8 module", rule: 8},
		{name: "rule 8 non-test selfupdatetest", rule: 8, want: "selfupdatetest", pkgs: []modPkg{{
			ImportPath: modulePath + "/agent",
			Imports:    []string{"github.com/maccavelli/go-selfupdate-lib/selfupdate/selfupdatetest"},
		}}},
		{name: "rule 8 test selfupdatetest", rule: 8, pkgs: []modPkg{{
			ImportPath:  modulePath + "/agent",
			TestImports: []string{"github.com/maccavelli/go-selfupdate-lib/selfupdate/selfupdatetest"},
		}}},
		{name: "rule 8 cli may import selfupdate", rule: 8, pkgs: []modPkg{{
			ImportPath: modulePath + "/internal/cli",
			Imports:    []string{"github.com/maccavelli/go-selfupdate-lib/selfupdate"},
		}}},
		{name: "rule 8 old module path", rule: 8, want: "go-core-lib", pkgs: []modPkg{{
			ImportPath: modulePath + "/agent",
			Imports:    []string{"github.com/maccavelli/go-core-lib/selfupdate"},
		}}},
		{name: "rule 8 buildinfo may import the lib's buildinfo", rule: 8, pkgs: []modPkg{{
			ImportPath: modulePath + "/internal/buildinfo",
			Imports:    []string{libBuildinfo},
		}}},
		{name: "rule 8 buildinfo may not import selfupdate", rule: 8, want: "go-selfupdate-lib/selfupdate", pkgs: []modPkg{{
			ImportPath: modulePath + "/internal/buildinfo",
			Imports:    []string{"github.com/maccavelli/go-selfupdate-lib/selfupdate"},
		}}},
		{name: "rule 8 agent may not import the lib's buildinfo", rule: 8, want: "go-selfupdate-lib/buildinfo", pkgs: []modPkg{{
			ImportPath: modulePath + "/agent",
			Imports:    []string{libBuildinfo},
		}}},
		{name: "rule 9 module", rule: 9},
		{name: "rule 9 agent kong", rule: 9, want: "alecthomas/kong", pkgs: []modPkg{{
			ImportPath: modulePath + "/agent",
			Imports:    []string{"github.com/alecthomas/kong"},
		}}},
		{name: "rule 9 cli may import kong", rule: 9, pkgs: []modPkg{{
			ImportPath: modulePath + "/internal/cli/complete",
			Imports:    []string{"github.com/alecthomas/kong"},
		}}},
		{name: "rule 10 module", rule: 10},
		{name: "rule 10 jsonx session", rule: 10, want: "jsonx imports " + modulePath + "/session", pkgs: []modPkg{{
			ImportPath: modulePath + "/jsonx",
			Imports:    []string{"bytes", modulePath + "/session"},
		}}},
		{name: "rule 10 jsonx's tests may import the module", rule: 10, pkgs: []modPkg{{
			ImportPath:   modulePath + "/jsonx",
			TestImports:  []string{modulePath + "/session"},
			XTestImports: []string{modulePath + "/jsonx"},
		}}},
		{name: "rule 10 session may import jsonx", rule: 10, pkgs: []modPkg{{
			ImportPath: modulePath + "/session",
			Imports:    []string{modulePath + "/jsonx"},
		}}},
		{name: "rule 11 module", rule: 11},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkgs := tc.pkgs
			if pkgs == nil {
				pkgs = real
			}
			got := checkRule(tc.rule, pkgs)
			joined := joinViolations(got)
			if tc.want == "" && len(got) != 0 {
				t.Fatalf("unexpected edge:\n%s", joined)
			}
			if tc.want != "" && !strings.Contains(joined, tc.want) {
				t.Fatalf("missing %q in:\n%s", tc.want, joined)
			}
		})
	}
}

func TestRule8ScratchFile(t *testing.T) {
	dir := t.TempDir()
	nonTest := filepath.Join(dir, "scratch.go")
	if err := os.WriteFile(nonTest, []byte("package agent\nimport _ \"github.com/maccavelli/go-selfupdate-lib/selfupdate/selfupdatetest\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := checkRule(8, []modPkg{pkgFromFile(t, nonTest, modulePath+"/agent", false)})
	if len(got) == 0 {
		t.Fatal("rule 8 did not fail on a non-test scratch file")
	}
	for _, v := range got {
		t.Log(v.String())
	}

	testFile := filepath.Join(dir, "scratch_test.go")
	if err := os.WriteFile(testFile, []byte("package agent\nimport _ \"github.com/maccavelli/go-selfupdate-lib/selfupdate/selfupdatetest\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = checkRule(8, []modPkg{pkgFromFile(t, testFile, modulePath+"/agent", true)})
	if len(got) != 0 {
		t.Fatalf("test file should be allowed, got %s", joinViolations(got))
	}
}

// Rule 11 refuses a []byte field with a json tag outside its list, in a
// struct type or one inside a function, and allows an untagged one, a
// tag with no json key, one json leaves out, and llm.Content.Data
// (0039-MADR D24).
func TestRule11ScratchFile(t *testing.T) {
	dir := t.TempDir()
	src := "package p\n\n" +
		"type Entry struct {\n\tRaw []byte `json:\"raw\"`\n\tPlain []byte\n\tYAML []byte `yaml:\"y\"`\n\tA, B []uint8 `json:\"ab\"`\n\tSkip []byte `json:\"-\"`\n}\n\n" +
		"type Content struct {\n\tData []byte `json:\"data,omitzero\"`\n}\n\n" +
		"func f() {\n\t_ = struct {\n\t\tInner []byte `json:\"inner\"`\n\t}{}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "scratch.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := func(rel string) []modPkg {
		return []modPkg{{ImportPath: modulePath + "/" + rel, Dir: dir, GoFiles: []string{"scratch.go"}}}
	}
	got := joinViolations(checkRule(11, pkg("session")))
	for _, want := range []string{"session has a []byte field with a json tag, Entry.Raw", "Entry.A", "Entry.B", "f.Inner", "Content.Data"} {
		if !strings.Contains(got, want) {
			t.Errorf("rule 11 in session lacks %s:\n%s", want, got)
		}
	}
	for _, not := range []string{"Plain", "YAML", "Skip"} {
		if strings.Contains(got, not) {
			t.Errorf("rule 11 refused %s, which has no json tag:\n%s", not, got)
		}
	}
	if got := joinViolations(checkRule(11, pkg("llm"))); strings.Contains(got, "Content.Data") {
		t.Errorf("rule 11 refused llm.Content.Data, which is listed:\n%s", got)
	}
}

func pkgFromFile(t *testing.T, path, importPath string, testFile bool) modPkg {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	var imports []string
	for _, spec := range file.Imports {
		imp, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		imports = append(imports, imp)
	}
	pkg := modPkg{ImportPath: importPath}
	if testFile {
		pkg.TestImports = imports
	} else {
		pkg.Imports = imports
	}
	return pkg
}

func loadReal(t *testing.T) []modPkg {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -m: %v", err)
	}
	pkgs, err := loadGraph(t.Context(), strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		if pkg.Err != "" {
			t.Errorf("go list %s: %s", pkg.ImportPath, pkg.Err)
		}
	}
	if t.Failed() {
		t.FailNow()
	}
	return pkgs
}

// The real graph holds the closure of a go-tui-lib package only a test
// imports, inline/inlinetest, and none of a test build's variants or test
// mains (0035-PLAN P14, deviation 5).
func TestLoadGraphTestClosures(t *testing.T) {
	var closure bool
	for _, pkg := range loadReal(t) {
		if strings.Contains(pkg.ImportPath, " [") || strings.HasSuffix(pkg.ImportPath, ".test") {
			t.Errorf("a test build's package listed: %s", pkg.ImportPath)
		}
		if pkg.ImportPath == tuiLib+"/inline/inlinetest" && len(pkg.Deps) > 0 {
			closure = true
		}
	}
	if !closure {
		t.Fatal("no closure for " + tuiLib + "/inline/inlinetest, which only tests import")
	}
}

func joinViolations(vs []Violation) string {
	var b strings.Builder
	for _, v := range vs {
		b.WriteString(v.String())
		b.WriteByte('\n')
	}
	return b.String()
}
