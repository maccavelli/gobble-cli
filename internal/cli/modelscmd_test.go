package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
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

// gobble models info resolves as --model does and prints the model's facts:
// each limit with its source, and whether the wire requires an output limit
// (0066-MADR D5, S15, S16, S18; 0071-MADR P45, the owner's "Expect the new
// output (Recommended)").
func TestModelsInfo(t *testing.T) {
	c := listEnv(t)
	r := c.run("models", "info", "custom-1")
	want := `provider  openai
model     custom-1
name      Custom One
context   200000 (models.json)
max-out   8192 (models.json)
input     text, image
thinking  off, minimal, low, medium, high
limit     optional on its wire
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

// A search finds, of each provider's models, what the SDK's search finds,
// as the ACP picker does; an empty one keeps every row (0031-MADR D8).
func TestModelsSearchIsTheSDKs(t *testing.T) {
	rows := []llm.Model{{Provider: "openai", ID: "gpt-6-luna"}, {Provider: "openai", ID: "gpt-6-astra"},
		{Provider: "claude", ID: "claude-haiku-5-5"}, {Provider: "claude", ID: "claude-opus-5-5"}}
	for _, q := range []string{"luna", "LUNA", "6gpt", "gpt6", "openai luna", "haiku", "opus-5", "zzz"} {
		var want []string
		for _, p := range []string{"openai", "claude"} {
			var ids []string
			for _, m := range rows {
				if m.Provider == p {
					ids = append(ids, m.ID)
				}
			}
			for _, id := range provider.Search(p, ids, q) {
				want = append(want, p+"/"+id)
			}
		}
		var got []string
		for _, m := range searchModels(rows, q, nil) {
			got = append(got, m.Provider+"/"+m.ID)
		}
		slices.Sort(want)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("search %q: %q, want the SDK's %q", q, got, want)
		}
	}
	if got := searchModels(rows, "  ", nil); len(got) != len(rows) {
		t.Errorf("an empty search kept %d rows, want all %d", len(got), len(rows))
	}
}

// factMetaDoc is the models.dev document factWire served by URL before
// go-llmprovider-sdk v1.33.0, which fetches none: factList hands the same
// bytes in as provider.Options.ModelData (0071-MADR P4).
const factMetaDoc = `{"openai":{"id":"openai","models":{
"gpt-6-luna":{"name":"Luna","limit":{"context":128000},"modalities":{"input":["text","image"],"output":["text"]},"tool_call":true},
"gpt-6-astra":{"name":"Astra","limit":{"context":400000},"modalities":{"input":["text"],"output":["text"]},"tool_call":true}}}}`

// factWire answers one OpenAI listing and its models.dev document, and
// nothing else, so the test never dials (0036-PLAN P5).
type factWire struct {
	meta string
}

func (w factWire) RoundTrip(r *http.Request) (*http.Response, error) {
	body, status := "{}", http.StatusNotFound
	switch r.URL.Host + r.URL.Path {
	case "api.openai.com/v1/models":
		body, status = `{"object":"list","data":[{"id":"gpt-6-luna","object":"model"},{"id":"gpt-6-astra","object":"model"}]}`, http.StatusOK
	case w.meta:
		body, status = factMetaDoc, http.StatusOK
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

// factList lists openai through factWire. Any other id is not live.
func factList(meta string) func(context.Context, string, string, provider.Options) (provider.Listing, error) {
	return func(ctx context.Context, id, key string, _ provider.Options) (provider.Listing, error) {
		if id != "openai" {
			return provider.Listing{Models: provider.Models(id)}, nil
		}
		data, err := provider.ParseModelData(strings.NewReader(factMetaDoc))
		if err != nil {
			return provider.Listing{}, err
		}
		return provider.List(ctx, id, key, provider.Options{
			HTTPClient: &http.Client{Transport: factWire{meta: meta}}, ModelData: data,
		})
	}
}

// gobble models list vision, and the picker's search, return the live
// listing's fact match first, then an id match the listing does not hold
// (0036-MADR D6). aaa-vision sorts ahead of gpt-6-luna, so a sorted table
// would show it first.
func TestModelsSearchListsFactsFirst(t *testing.T) {
	meta := fmt.Sprintf("metadata.test/%s-%d.json", t.Name(), os.Getpid())
	withLister(t, factList(meta))
	t.Setenv("OPENAI_API_KEY", fmt.Sprintf("sk-facts-%d", os.Getpid()))
	c := newCfgEnv(t)
	c.write(filepath.Join(c.home, "models.json"), `{"providers":{"openai":{"models":[{"id":"aaa-vision"}]}}}`)
	r := c.run("models", "list", "vision")
	var ids []string
	for _, row := range rowsOf(r.stdout)[1:] {
		if len(row) > 1 {
			ids = append(ids, row[1])
		}
	}
	if r.code != ExitOK || !slices.Equal(ids, []string{"gpt-6-luna", "aaa-vision"}) {
		t.Fatalf("models list vision: exit %d, ids %q, stdout %q, stderr %q", r.code, ids, r.stdout, r.stderr)
	}
	e, _ := modelEnv(t)
	got := e.agentOptions().SearchModels("openai", []string{"aaa-vision", "gpt-6-luna", "gpt-6-astra"}, "vision")
	if !slices.Equal(got, []string{"gpt-6-luna", "aaa-vision"}) {
		t.Fatalf("picker search %q, want the listing's match then the id match", got)
	}
}

// An id-only query for a model the live listing holds finds it, once, in
// the table and in the picker's search: the listing's own search ranks ids
// as well as facts (0036-PLAN, gap G6). gpt-6-astra is held by the listing.
func TestModelsSearchFindsHeldModelByID(t *testing.T) {
	meta := fmt.Sprintf("metadata.test/%s-%d.json", t.Name(), os.Getpid())
	withLister(t, factList(meta))
	t.Setenv("OPENAI_API_KEY", fmt.Sprintf("sk-facts-%d", os.Getpid()))
	c := newCfgEnv(t)
	r := c.run("models", "list", "gpt-6-astra")
	n := 0
	for _, row := range rowsOf(r.stdout)[1:] {
		if len(row) > 1 && row[0] == "openai" && row[1] == "gpt-6-astra" {
			n++
		}
	}
	if r.code != ExitOK || n != 1 {
		t.Fatalf("models list gpt-6-astra: exit %d, %d rows of it, want 1; stdout %q, stderr %q", r.code, n, r.stdout, r.stderr)
	}
	e, _ := modelEnv(t)
	got := e.agentOptions().SearchModels("openai", []string{"gpt-6-luna", "gpt-6-astra"}, "gpt-6-astra")
	if n := slices.Index(got, "gpt-6-astra"); n < 0 || slices.Index(got[n+1:], "gpt-6-astra") >= 0 {
		t.Fatalf("picker search %q, want gpt-6-astra once", got)
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
