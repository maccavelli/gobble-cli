package catalog

import (
	"encoding/json/v2"
	"os/exec"
	"testing"
)

// M47: llm/catalog embeds no file, as go list reports the package: the
// built-in models.dev copy (data/models.json, data/source.json) is gone, and
// gobble carries no model data (0072-MADR D14; Q3 "None; remove in 0072
// (Recommended)").
func TestNoEmbeddedModelData_0072(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "list", "-json", ".").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	var p struct {
		ImportPath    string
		EmbedPatterns []string
		EmbedFiles    []string
	}
	if err := json.Unmarshal(out, &p); err != nil {
		t.Fatal(err)
	}
	if p.ImportPath != "github.com/maccavelli/gobble-cli/llm/catalog" {
		t.Fatalf("go list read %q", p.ImportPath)
	}
	if len(p.EmbedFiles) > 0 || len(p.EmbedPatterns) > 0 {
		t.Errorf("llm/catalog embeds %q (patterns %q); want no file", p.EmbedFiles, p.EmbedPatterns)
	}
}
