package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// These tests use only names the tree had before 0072-PLAN's Phase 3, so
// M33's is red on its assertion there, where the built-in copy answers.

// noModelData leaves no document current for t: what a run has with no
// copy, now that gobble carries no built-in one (0072-MADR D13, M19).
func noModelData(t *testing.T) {
	t.Helper()
	keepModelData(t)
	provider.SetCurrentModelData(nil)
}

// M33 (internal/cli's half): a first run that is never online, with no
// copy and its download failing, knows no model: no window, no cap of the
// model's, no prices, thinking levels off to xhigh, the curated ids alone,
// "-" for every fact in gobble models list, the data line "none", and D13's
// warning after the failed attempt (0072-MADR D13; Q3 "None; remove in 0072
// (Recommended)").
func TestNeverOnlineFirstRun_0072(t *testing.T) {
	noModelData(t)
	t.Run("facts", func(t *testing.T) {
		e, _ := modelEnv(t)
		o := e.agentOptions()
		m := o.ModelInfo(nil, "claude", "claude-opus-4-6")
		if m.ContextWindow != 0 || m.MaxTokens != 0 || m.Priced {
			t.Errorf("claude-opus-4-6: window %d, cap %d, priced %v; want each unknown", m.ContextWindow, m.MaxTokens, m.Priced)
		}
		if got, want := catalog.Levels(m), []string{"off", "minimal", "low", "medium", "high", "xhigh"}; !slices.Equal(got, want) {
			t.Errorf("thinking levels %v; want %v", got, want)
		}
		if got := provider.KnownModels("claude"); !slices.Equal(got, provider.Models("claude")) {
			t.Errorf("known models %v; want the curated ones alone, %v", got, provider.Models("claude"))
		}
		if got, want := o.ModelDataLine(nil), "Model data: none; models are unknown until gobble models refresh downloads them"; got != want {
			t.Errorf("data line %q; want %q", got, want)
		}
	})
	t.Run("models_list", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("GOBBLE_HOME", home)
		t.Chdir(t.TempDir())
		_, fetches := withCopy(t, nil)
		realTrigger(t)
		r := runIn(t, home, "models", "list", "--provider", "claude")
		rows := rowsOf(r.stdout)
		if r.code != ExitOK || len(rows) < 2 {
			t.Fatalf("exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
		}
		for _, row := range rows[1:] {
			if len(row) == 0 { // the table ends; the no-credential note follows
				break
			}
			if len(row) != 6 || !slices.Equal(row[2:], []string{"-", "-", "-", "-"}) {
				t.Errorf("row %q; want every fact unknown, \"-\"", row)
			}
		}
		const warning = "no model data: models are unknown; gobble models refresh downloads them"
		if fetches.Load() != 1 || !strings.Contains(r.stderr, warning) || !strings.Contains(r.stderr, "offline") {
			t.Errorf("%d fetches, stderr %q; want one failed download, then D13's warning naming it offline", fetches.Load(), r.stderr)
		}
	})
}

// M88: with no model data, the agent's start logs no "model catalog"
// warning, on stderr or in the log: SnapshotErr went with the built-in copy,
// and no ModelDataErr replaces it; D3's and D13's are the model data's
// warnings (0072-MADR M88; "Remove it (Recommended)").
func TestNoModelCatalogWarning_0072(t *testing.T) {
	noModelData(t)
	e, warnings := modelEnv(t)
	e.agentOptions()
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	logged, err := os.ReadFile(filepath.Join(dirs.State, "logs", "gobble.log"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(warnings.String(), "model catalog") || strings.Contains(string(logged), "model catalog") {
		t.Errorf("stderr %q, log %q; want no \"model catalog\" warning", warnings.String(), logged)
	}
}
