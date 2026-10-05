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
		{name: "rule 2 root pin", rule: 2, pkgs: []modPkg{{
			ImportPath: modulePath,
			Imports:    []string{"github.com/coder/acp-go-sdk"},
		}}},
		{name: "rule 2 cli acp", rule: 2, want: "acp-go-sdk", pkgs: []modPkg{{
			ImportPath: modulePath + "/internal/cli",
			Imports:    []string{"github.com/coder/acp-go-sdk"},
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

func joinViolations(vs []Violation) string {
	var b strings.Builder
	for _, v := range vs {
		b.WriteString(v.String())
		b.WriteByte('\n')
	}
	return b.String()
}
