package cli

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// customDoc is a models.json defining one openai model with every fact
// gobble prints.
const customDoc = `{"providers": {"openai": {"models": [{"id": "custom-1", "name": "Custom One",
  "contextWindow": 200000, "maxTokens": 8192, "input": ["text", "image"], "reasoning": true,
  "cost": {"input": 1, "output": 2, "cacheRead": 0.5, "cacheWrite": 0,
    "tiers": [{"inputTokensAbove": 100000, "input": 3, "output": 4, "cacheRead": 1, "cacheWrite": 0}]}}]}}}`

// listEnv is a config directory holding customDoc, with OPENAI_API_KEY set
// and a listing that answers live with gpt-6-astra and live-only-1.
func listEnv(t *testing.T) *cfgEnv {
	t.Helper()
	c := newCfgEnv(t)
	c.write(filepath.Join(c.home, "models.json"), customDoc)
	t.Setenv("OPENAI_API_KEY", "test-only")
	withLister(t, (&liveLister{ids: []string{"gpt-6-astra", "live-only-1"}}).list)
	return c
}

// rowsOf are a table's lines as fields, its header first.
func rowsOf(out string) [][]string {
	var rows [][]string
	for line := range strings.Lines(out) {
		rows = append(rows, strings.Fields(line))
	}
	return rows
}

// gobble models list prints Pi's columns for the providers with a
// credential: the offline set and the live listing, each model once, sorted;
// a fact the catalog lacks is "-" (0005-PLAN F4-2c).
func TestModelsList(t *testing.T) {
	t.Run("no credential", func(t *testing.T) {
		c := newCfgEnv(t)
		r := c.run("models", "list")
		if r.code != ExitOK || r.stdout != "No models available. Set one of "+strings.Join(provider.CredentialVars(), ", ")+".\n" {
			t.Errorf("exit %d, stdout %q", r.code, r.stdout)
		}
	})
	c := listEnv(t)
	r := c.run("models", "list")
	rows := rowsOf(r.stdout)
	if r.code != ExitOK || len(rows) < 3 || !slices.Equal(rows[0], []string{"provider", "model", "context", "max-out", "thinking", "images"}) {
		t.Fatalf("exit %d, stdout %q", r.code, r.stdout)
	}
	var refs []string
	for _, row := range rows[1:] {
		refs = append(refs, row[0]+"/"+row[1])
		switch row[1] {
		case "custom-1":
			if !slices.Equal(row, []string{"openai", "custom-1", "200K", "8.2K", "yes", "yes"}) {
				t.Errorf("custom-1's row %q", row)
			}
		case "live-only-1":
			if !slices.Equal(row, []string{"openai", "live-only-1", "-", "-", "-", "-"}) {
				t.Errorf("live-only-1's row %q, want its unknown facts as -", row)
			}
		}
	}
	for _, ref := range []string{"openai/custom-1", "openai/live-only-1", "openai/gpt-6-astra"} {
		if n := slices.Index(refs, ref); n < 0 || slices.Index(refs[n+1:], ref) >= 0 {
			t.Errorf("%s is not listed once", ref)
		}
	}
	if !slices.IsSortedFunc(refs, catalog.CompareIDs) {
		t.Error("the rows are not sorted by provider, then id")
	}
	if strings.Contains(r.stdout, " \n") {
		t.Error("a line ends in spaces")
	}
	if r := c.run("models", "list", "live", "only"); !slices.EqualFunc(rowsOf(r.stdout)[1:], [][]string{{"openai", "live-only-1", "-", "-", "-", "-"}}, slices.Equal) {
		t.Errorf("search: %q", r.stdout)
	}
	if r := c.run("models", "list", "qqqqqqqqq"); r.code != ExitOK || r.stdout != "No models matching \"qqqqqqqqq\"\n" {
		t.Errorf("no match: exit %d, %q", r.code, r.stdout)
	}
}

// --list-models is gobble models list, the prompt words its search.
func TestListModelsFlag(t *testing.T) {
	c := listEnv(t)
	want := c.run("models", "list", "custom")
	got := c.run("--list-models", "custom")
	if got.code != ExitOK || got.stdout != want.stdout || !strings.Contains(got.stdout, "custom-1") {
		t.Errorf("--list-models: exit %d, %q; want models list's %q", got.code, got.stdout, want.stdout)
	}
}

// gobble models info resolves as --model does and prints the model's facts.
func TestModelsInfo(t *testing.T) {
	c := listEnv(t)
	r := c.run("models", "info", "custom-1")
	want := `provider  openai
model     custom-1
name      Custom One
context   200000
max-out   8192
input     text, image
thinking  off, minimal, low, medium, high
cost      input $1, output $2, cache read $0.5, cache write $0, per million tokens
tier      above 100000: input $3, output $4, cache read $1, cache write $0
`
	if r.code != ExitOK || r.stdout != want {
		t.Errorf("exit %d, stdout:\n%s\nwant:\n%s", r.code, r.stdout, want)
	}
	if r := c.run("models", "info", "openai/unseen-9"); r.code != ExitOK ||
		!strings.Contains(r.stdout, "name      unknown\ncontext   unknown\n") || !strings.Contains(r.stderr, "Using custom model id.") {
		t.Errorf("a custom id: exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	if r := c.run("models", "info", "zzz-none"); r.code != ExitUsage ||
		!strings.Contains(r.stderr, `Model "zzz-none" not found. Use --list-models to see available models.`) {
		t.Errorf("unknown: exit %d, stderr %q", r.code, r.stderr)
	}
}

// A search word matches as a subsequence, without case, or with its letters
// and digits swapped, as Pi's fuzzyMatch decides.
func TestFuzzyMatch(t *testing.T) {
	for _, c := range []struct {
		query, text string
		want        bool
	}{
		{"luna", "openai gpt-6-luna", true},
		{"LUNA", "openai gpt-6-luna", true},
		{"gpt6", "openai gpt-6-luna", true},
		{"6gpt", "openai gpt-6-luna", true},
		{"gpt7", "openai gpt-6-luna", false},
		{"anul", "openai gpt-6-luna", false},
		{"", "openai gpt-6-luna", true},
	} {
		if got := fuzzyMatch(c.query, c.text); got != c.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", c.query, c.text, got, c.want)
		}
	}
}

// Counts print as Pi's formatTokenCount prints them; unknown is "-".
func TestTokenCount(t *testing.T) {
	for n, want := range map[int]string{0: "-", 999: "999", 1000: "1K", 1500: "1.5K", 8192: "8.2K", 200000: "200K", 1000000: "1M", 1050000: "1.1M"} {
		if got := tokenCount(n); got != want {
			t.Errorf("tokenCount(%d) = %q, want %q", n, got, want)
		}
	}
}

// --model completes the offline set, models.json's models included, each
// described by its name.
func TestModelCandidates(t *testing.T) {
	c := newCfgEnv(t)
	c.write(filepath.Join(c.home, "models.json"), customDoc)
	got := map[string]string{}
	for cand := range modelCandidates(t.Context()) {
		got[cand.Value] = cand.Description
	}
	if got["openai/custom-1"] != "Custom One" {
		t.Errorf("openai/custom-1: %q, want its name", got["openai/custom-1"])
	}
	for _, p := range provider.KeyProviders() {
		if _, ok := got[p+"/"+provider.Models(p)[0]]; !ok {
			t.Errorf("no candidate for %s's first curated model", p)
		}
	}
}

// --provider completes the key providers, each described by its variable.
func TestProviderCandidates(t *testing.T) {
	var values []string
	for cand := range providerCandidates(t.Context()) {
		values = append(values, cand.Value)
		if cand.Description != provider.CredentialVar(cand.Value) {
			t.Errorf("%s: description %q", cand.Value, cand.Description)
		}
	}
	if !slices.Equal(values, provider.KeyProviders()) {
		t.Errorf("providers %q, want %q", values, provider.KeyProviders())
	}
}
