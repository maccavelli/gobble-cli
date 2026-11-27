package acpserver

import (
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// M33 (acpserver's half): a session with no model data, its facts read by
// the real lookup with no document current (no copy, and no built-in copy
// since 0072-PLAN's Phase 3), has no window, so /context counts against an
// unknown one and nothing compacts on its own, and its turn sends 0071's
// assumed 16,384 (0072-MADR D13; 0071-MADR L4, L5). It uses only names the
// tree had before Phase 3, where the built-in copy answers, so it is red on
// its assertion there.
func TestNeverOnlineFirstRun_0072(t *testing.T) {
	found := provider.CurrentModelData()
	provider.SetCurrentModelData(nil)
	t.Cleanup(func() { provider.SetCurrentModelData(found) })
	h := newLimitHarnessWith(t, SessionSettings{}, map[string]llm.Model{}, func(o *Options) {
		o.ModelData = func() *provider.ModelData { return nil }
		o.ModelInfo = func(d *provider.ModelData, id, model string) llm.Model {
			m := provider.ModelInfoWith(provider.Options{ModelData: d}, "claude", "claude-opus-4-6", nil)
			m.Provider, m.ID = id, model
			return m
		}
	}, llmtest.Text("hello back"))
	id := h.session()
	h.mustPrompt(id, "hello")
	calls := h.calls()
	if len(calls) != 1 {
		t.Fatalf("%d calls; want one", len(calls))
	}
	if calls[0].MaxTokens != llm.AssumedMaxTokens {
		t.Errorf("the turn asked for %d; want the assumed %d", calls[0].MaxTokens, llm.AssumedMaxTokens)
	}
	if got := h.context(id); !strings.Contains(got, "of an unknown window") {
		t.Errorf("/context %q; want the window unknown", got)
	}
}
