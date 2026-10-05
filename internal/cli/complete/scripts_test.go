package complete

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden")

func TestScriptsGolden(t *testing.T) {
	for _, sh := range Shells {
		t.Run(sh, func(t *testing.T) {
			got, err := Script(sh, "gobble")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", sh+".golden")
			if *update {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run with -update to create it)", err)
			}
			if string(want) != got {
				t.Fatalf("%s script differs from %s; run with -update after reviewing the change", sh, path)
			}
		})
	}
}

// No script evaluates text: no eval, no Invoke-Expression, outside the
// comment that shows the user how to install it (0008-MADR D17).
func TestScriptsDoNotEvaluate(t *testing.T) {
	for _, sh := range Shells {
		s, err := Script(sh, "gobble")
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(s, "\n") {
			code := strings.TrimSpace(line)
			if strings.HasPrefix(code, "#") {
				continue
			}
			for _, bad := range []string{"eval ", "Invoke-Expression", "iex "} {
				if strings.Contains(code, bad) {
					t.Errorf("%s line %d evaluates text: %q", sh, i+1, line)
				}
			}
		}
		if !strings.Contains(s, "GOBBLE_COMPLETE_PROTOCOL") || !strings.Contains(s, Protocol) {
			t.Errorf("%s script does not pass the protocol", sh)
		}
	}
}

func TestScriptUnknownShell(t *testing.T) {
	if _, err := Script("tcsh", "gobble"); err == nil {
		t.Fatal("tcsh rendered a script")
	}
}
