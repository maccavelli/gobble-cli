package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// writeModelsJSON writes doc as the config directory's models.json.
func writeModelsJSON(t *testing.T, doc string) {
	t.Helper()
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirs.Config, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirs.Config, "models.json"), []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
}

// models.json is read once per process: no file is none, quietly; a bad
// file is none, warned once; each provider it skips is warned (0005-PLAN
// F4-2b).
func TestModelsJSONLoad(t *testing.T) {
	t.Run("no file", func(t *testing.T) {
		e, warnings := modelEnv(t)
		if mj := e.modelsJSON(); mj != nil || strings.Contains(warnings.String(), "models.json") {
			t.Errorf("models.json %v, warnings %q; want none of either", mj, warnings)
		}
	})
	t.Run("bad file", func(t *testing.T) {
		e, warnings := modelEnv(t)
		writeModelsJSON(t, `{"providers": `)
		first, second := e.modelsJSON(), e.modelsJSON()
		if first != nil || second != nil || strings.Count(warnings.String(), "the file is ignored") != 1 {
			t.Errorf("models.json %v and %v, warnings %q; want none, warned once", first, second, warnings)
		}
	})
	t.Run("a skipped provider", func(t *testing.T) {
		e, warnings := modelEnv(t)
		writeModelsJSON(t, `{"providers": {"mystery": {"baseUrl": "https://x.example"}, "openai": {"baseUrl": "https://y.example/v1"}}}`)
		mj := e.modelsJSON()
		if mj == nil || mj.BaseURL("openai") != "https://y.example/v1" || !strings.Contains(warnings.String(), `provider \"mystery\"`) {
			t.Errorf("models.json %v, warnings %q", mj, warnings)
		}
	})
}

// A command runs through the bash tool's shell: its output, or an error for
// a non-zero exit.
func TestRunCommand(t *testing.T) {
	e, _ := modelEnv(t)
	if _, err := builtin.BashPath(builtinOptions(e.processSettings("").Settings)); err != nil {
		t.Skipf("no bash to run commands with: %v", err)
	}
	if out, err := e.runCommand(t.Context(), "printf ' %s ' token"); err != nil || out != " token " {
		t.Errorf("printf: %q, %v", out, err)
	}
	if _, err := e.runCommand(t.Context(), "exit 3"); err == nil {
		t.Error("exit 3 gave no error")
	}
}

// A provider's headers come from models.json, resolved for the request's
// model from the environment.
func TestHeadersFor(t *testing.T) {
	e, _ := modelEnv(t)
	writeModelsJSON(t, `{"providers": {"openai": {"headers": {"X-Team": "$GOBBLE_TEST_TEAM"},
  "modelOverrides": {"m": {"headers": {"X-Model": "m"}}}}}}`)
	t.Setenv("GOBBLE_TEST_TEAM", "core")
	h := e.headersFor(e.modelsJSON(), "openai")
	if h == nil {
		t.Fatal("no Headers for a provider with headers")
	}
	got, err := h(t.Context(), "m")
	if err != nil || got["X-Team"] != "core" || got["X-Model"] != "m" {
		t.Errorf("headers %v, %v", got, err)
	}
	t.Setenv("GOBBLE_TEST_TEAM", "")
	if _, err := h(t.Context(), "m"); err == nil || !strings.Contains(err.Error(), `header "X-Team"`) {
		t.Errorf("an unset variable: %v, want an error naming the header", err)
	}
	if e.headersFor(e.modelsJSON(), "claude") != nil {
		t.Error("Headers for a provider with none")
	}
}

// The model choice is the listing's ids, then models.json's models, each
// once.
func TestModelChoiceModelsJSON(t *testing.T) {
	e, _ := modelEnv(t)
	writeModelsJSON(t, `{"providers": {"openai": {"models": [{"id": "custom-1"}, {"id": "gpt-6-luna"}]}}}`)
	t.Setenv("OPENAI_API_KEY", "test-only")
	l := &liveLister{ids: []string{"gpt-6-astra", "gpt-6-luna"}}
	withLister(t, l.list)
	if c := e.modelChoice(); !slices.Equal(c.Models, []string{"gpt-6-astra", "gpt-6-luna", "custom-1"}) || c.Default != "gpt-6-astra" {
		t.Errorf("model choice %+v", c)
	}
}
