package llm_test

import (
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// FitOutput is 0071-MADR D2: the cap under the ceiling unless the user set
// it, then fitted to the room the window leaves, with the floor, the
// unknown window and the full window (L1 to L5, L7, L33, L35).
func TestFitOutput(t *testing.T) {
	const w = 200000
	for _, c := range []struct {
		name             string
		capTokens        int
		from             llm.CapSource
		window, estimate int
		want             llm.OutputLimit
		room             int // window − estimate − 4,096, for the message
	}{
		// R ≥ C': the cap, by its basis.
		{"stated", 32000, llm.CapCatalog, w, 1000, llm.OutputLimit{Tokens: 32000, Basis: llm.BasisStated}, 0},
		{"stated_unsourced", 32000, "", w, 1000, llm.OutputLimit{Tokens: 32000, Basis: llm.BasisStated}, 0},
		{"ceiling", 943718, llm.CapCatalog, 1048576, 1000, llm.OutputLimit{Tokens: 64000, Basis: llm.BasisCeiling}, 0},
		{"user_above_ceiling", 100000, llm.CapUser, w, 1000, llm.OutputLimit{Tokens: 100000, Basis: llm.BasisUser}, 0},
		{"assumed_unknown", 0, "", w, 1000, llm.OutputLimit{Tokens: 16384, Basis: llm.BasisAssumed}, 0},
		{"assumed_models_json", 16384, llm.CapAssumed, w, 1000, llm.OutputLimit{Tokens: 16384, Basis: llm.BasisAssumed}, 0},
		// 1,024 ≤ R < C': the room, exactly C' − 1 at the edge.
		{"window", 64000, llm.CapCatalog, w, w - 4096 - 40000, llm.OutputLimit{Tokens: 40000, Basis: llm.BasisWindow}, 40000},
		{"window_at_floor", 64000, llm.CapCatalog, w, w - 4096 - 1024, llm.OutputLimit{Tokens: 1024, Basis: llm.BasisWindow}, 1024},
		{"room_equal_cap", 64000, llm.CapCatalog, w, w - 4096 - 64000, llm.OutputLimit{Tokens: 64000, Basis: llm.BasisStated}, 64000},
		// 0 < R < 1,024: the cap held to the floor.
		{"floor", 64000, llm.CapCatalog, w, w - 4096 - 500, llm.OutputLimit{Tokens: 1024, Basis: llm.BasisWindow}, 500},
		{"floor_small_cap", 800, llm.CapCatalog, w, w - 4096 - 500, llm.OutputLimit{Tokens: 800, Basis: llm.BasisWindow}, 500},
		{"floor_one", 64000, llm.CapCatalog, w, w - 4096 - 1, llm.OutputLimit{Tokens: 1024, Basis: llm.BasisWindow}, 1},
		// W unknown: C'.
		{"no_window", 943718, llm.CapCatalog, 0, 5000000, llm.OutputLimit{Tokens: 64000, Basis: llm.BasisCeiling}, 0},
		{"no_window_user", 100000, llm.CapUser, 0, 5000000, llm.OutputLimit{Tokens: 100000, Basis: llm.BasisUser}, 0},
		// R ≤ 0: C', basis full.
		{"full", 943718, llm.CapCatalog, w, w - 4096, llm.OutputLimit{Tokens: 64000, Basis: llm.BasisFull}, 0},
		{"full_over", 32000, llm.CapCatalog, w, w + 5000, llm.OutputLimit{Tokens: 32000, Basis: llm.BasisFull}, -9096},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := llm.FitOutput(c.capTokens, c.from, c.window, c.estimate); got != c.want {
				t.Errorf("FitOutput(%d, %q, %d, %d) = %+v, want %+v (room %d)", c.capTokens, c.from, c.window, c.estimate, got, c.want, c.room)
			}
		})
	}
}

// The numbers are the record's (0071-MADR "Every number").
func TestLimitNumbers(t *testing.T) {
	if llm.ContextSafetyTokens != 4096 || llm.OutputCeiling != 64000 || llm.MinOutputTokens != 1024 || llm.AssumedMaxTokens != 16384 {
		t.Errorf("numbers %d, %d, %d, %d; want 4096, 64000, 1024, 16384", llm.ContextSafetyTokens, llm.OutputCeiling,
			llm.MinOutputTokens, llm.AssumedMaxTokens)
	}
}
