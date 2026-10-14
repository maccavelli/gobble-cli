package catalog

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"maps"
	"math"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// sections are the nine models.dev sections go-llmprovider-sdk reads.
var sections = []string{"anthropic", "google", "huggingface", "kilo", "openai", "opencode", "opencode-go", "togetherai", "xai"}

// The snapshot holds the nine sections and no other, and source.json
// describes it: its source, its time, and each section's model count.
func TestSnapshot(t *testing.T) {
	var doc map[string]struct {
		Models map[string]jsontext.Value `json:"models"`
	}
	if err := json.Unmarshal(Snapshot(), &doc); err != nil {
		t.Fatal(err)
	}
	if got := slices.Sorted(maps.Keys(doc)); !slices.Equal(got, sections) {
		t.Fatalf("sections %q, want %q", got, sections)
	}
	var src struct {
		Providers map[string]int `json:"providers"`
		SHA256    string         `json:"sha256"`
	}
	if err := json.Unmarshal(source, &src); err != nil {
		t.Fatal(err)
	}
	for _, key := range sections {
		if n := len(doc[key].Models); n == 0 || src.Providers[key] != n {
			t.Errorf("%s: %d models, source.json says %d", key, n, src.Providers[key])
		}
	}
	if url, fetched := Source(); url == "" || fetched.IsZero() || len(src.SHA256) != 64 {
		t.Errorf("source %q fetched %v sha256 %q", url, fetched, src.SHA256)
	}
	b := Snapshot()
	b[0] = 'x'
	if Snapshot()[0] == 'x' {
		t.Error("Snapshot returned the embedded bytes, not a copy")
	}
}

// Levels follows Pi's rule, and a model of unknown reasoning keeps the
// levels gobble offered before the catalog.
func TestLevels(t *testing.T) {
	for _, c := range []struct {
		name string
		m    llm.Model
		want []string
	}{
		{"unknown", llm.Model{}, []string{"off", "minimal", "low", "medium", "high", "xhigh"}},
		{"no reasoning", llm.Model{ReasoningKnown: true}, []string{"off"}},
		{"no efforts", llm.Model{ReasoningKnown: true, Reasoning: true}, []string{"off", "minimal", "low", "medium", "high"}},
		{"none listed", llm.Model{ReasoningKnown: true, Reasoning: true, Efforts: []string{"none", "low", "high"}},
			[]string{"off", "low", "high"}},
		{"none not listed", llm.Model{ReasoningKnown: true, Reasoning: true, Efforts: []string{"low", "medium", "high", "xhigh", "max"}},
			[]string{"low", "medium", "high", "xhigh", "max"}},
		{"no level Pi names", llm.Model{ReasoningKnown: true, Reasoning: true, Efforts: []string{"default"}},
			[]string{"off", "minimal", "low", "medium", "high"}},
	} {
		if got := Levels(c.m); !slices.Equal(got, c.want) {
			t.Errorf("%s: Levels = %q, want %q", c.name, got, c.want)
		}
	}
}

// Clamp keeps an offered level, goes up first, then down, and maps a level
// Pi does not name to the first offered.
func TestClamp(t *testing.T) {
	levels := []string{"low", "medium", "high"}
	for _, c := range []struct{ level, want string }{
		{"medium", "medium"}, {"off", "low"}, {"minimal", "low"}, {"xhigh", "high"}, {"max", "high"}, {"turbo", "low"},
	} {
		if got := Clamp(levels, c.level); got != c.want {
			t.Errorf("Clamp(%q) = %q, want %q", c.level, got, c.want)
		}
	}
	if got := Clamp([]string{"low", "high"}, "medium"); got != "high" {
		t.Errorf("Clamp over a gap = %q, want high: up before down", got)
	}
	if got := Clamp(nil, "high"); got != "off" {
		t.Errorf("Clamp(nil) = %q, want off", got)
	}
}

// Price charges each category at its rate per million tokens, takes a tier
// once the input with its cache reads and writes passes it, and is zero for
// a model with no price.
func TestPrice(t *testing.T) {
	m := llm.Model{Priced: true, Cost: llm.ModelCost{Input: 3, Output: 15, CacheRead: 0.3, CacheWrite: 3.75,
		Tiers: []llm.CostTier{{Above: 200000, Input: 6, Output: 22.5, CacheRead: 0.6, CacheWrite: 7.5}}}}
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	got := Price(m, 1000, 500, 2000, 400)
	want := Charge{Input: 0.003, Output: 0.0075, CacheRead: 0.0006, CacheWrite: 0.0015, Total: 0.0126}
	if !near(got.Input, want.Input) || !near(got.Output, want.Output) || !near(got.CacheRead, want.CacheRead) ||
		!near(got.CacheWrite, want.CacheWrite) || !near(got.Total, want.Total) {
		t.Errorf("base: Price = %+v, want %+v", got, want)
	}
	if got := Price(m, 150000, 1000, 50000, 1); !near(got.Input, 0.9) || !near(got.Output, 0.0225) {
		t.Errorf("past the tier: Price = %+v, want input 0.9 and output 0.0225", got)
	}
	if got := Price(m, 150000, 1000, 50000, 0); !near(got.Input, 0.45) {
		t.Errorf("at the tier: Price = %+v, want input 0.45", got)
	}
	if got := Price(llm.Model{Cost: m.Cost}, 1000, 500, 0, 0); got != (Charge{}) {
		t.Errorf("unpriced: Price = %+v, want zero", got)
	}
}
