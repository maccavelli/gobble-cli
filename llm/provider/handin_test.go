package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

	"github.com/maccavelli/gobble-cli/llm"
)

// The tests of 0072-PLAN's Phase 2b in llm/provider: the session's document
// handed to the lookups, the limits and New by catalog.WithMetadata
// (0072-MADR D17, M54, M55, M62, M73, M75, M76; "Builders take the document
// (Recommended)").

// handDoc is a models.dev document stating output tokens for savedModel,
// stamped from source.
func handDoc(t *testing.T, source string, output int) *ModelData {
	t.Helper()
	d := modelData(t, fmt.Sprintf(`{"anthropic":{"models":{"%s":{"name":"Saved One","limit":{"context":200000,"output":%d}}}}}`,
		savedModel, output))
	return d.Stamped(ModelDataStamp{Source: source, Fetched: time.Now()})
}

// metadataValues are the catalog.Metadata values opts hand the SDK by
// catalog.WithMetadata; a document handed as a layer is not one.
func metadataValues(t *testing.T, opts []llmprovider.Option) int {
	t.Helper()
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, opts)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, v := range st.Values() {
		if _, ok := v.(catalog.Metadata); ok {
			n++
		}
	}
	return n
}

// M55, M75: a lookup, the limits and discovery read the session's document,
// handed in by catalog.WithMetadata, not as a layer; nil reads the current
// one. (M75: 0066's TestLimits_BuiltInCopyBelowListing_0066 is retired into
// this test: the built-in copy's layer is gone.)
func TestLookupGetsSessionModelData_0072(t *testing.T) {
	session, current := handDoc(t, "https://session.invalid/api.json", 24000), handDoc(t, "https://current.invalid/api.json", 30000)
	SetCurrentModelData(current)
	t.Cleanup(func() { SetCurrentModelData(nil) })
	if m := ModelInfoWith(Options{ModelData: session}, "claude", savedModel, nil); m.MaxTokens != 24000 || m.Name != "Saved One" {
		t.Errorf("a lookup with the session's document: MaxTokens %d, name %q; want its 24,000", m.MaxTokens, m.Name)
	}
	if m := ModelInfoWith(Options{}, "claude", savedModel, nil); m.MaxTokens != 30000 {
		t.Errorf("a lookup with none: MaxTokens %d; want the current document's 30,000", m.MaxTokens)
	}
	if l := CachedLimits(Options{ModelData: session}, "claude", savedModel, nil); l.Output.Tokens != 24000 {
		t.Errorf("the limits with the session's document: output %+v; want its 24,000", l.Output)
	}
	var handed int
	old := lookupLimits
	lookupLimits = func(ctx context.Context, p llmprovider.ProviderID, _ llmprovider.TokenSource, model string, opts ...llmprovider.Option) (catalog.Limits, error) {
		handed = metadataValues(t, opts)
		return catalog.CachedLimits(ctx, p, model, opts...), nil
	}
	t.Cleanup(func() { lookupLimits = old })
	if err := Discover(t.Context(), Options{ModelData: session}, "claude", savedModel, nil); err != nil {
		t.Fatal(err)
	}
	if handed != 1 {
		t.Errorf("discovery handed %d documents by catalog.WithMetadata; want the session's, handed in, not as a layer", handed)
	}
}

// M62, M76: New hands the provider the session's document, so a request
// that names no limit is sent at the document's output; none handed, the
// SDK refuses it before sending (0093-MADR D3); the saved caps' layer stays
// beside the document, below models.json.
func TestProviderGetsSessionModelData_0072(t *testing.T) {
	session := handDoc(t, "https://session.invalid/api.json", 24000)
	send := func(t *testing.T, o Options) (*cannedWire, error) {
		t.Helper()
		c := &cannedWire{}
		o.APIKey, o.Model, o.HTTPClient, o.Retry = "k", savedModel, &http.Client{Transport: c}, &llm.Retry{Off: true}
		p, err := New("claude", o)
		if err != nil {
			t.Fatal(err)
		}
		req := wireRequest()
		req.Model = savedModel // no limit named: the SDK chooses it from what the provider was given
		_, err = drain(t, p, req)
		return c, err
	}
	t.Run("document", func(t *testing.T) {
		c, err := send(t, Options{ModelData: session})
		if err != nil {
			t.Fatalf("the call failed: %v; want it sent at the session's document's output", err)
		}
		if got := sentMaxTokens(t, c); got != 24000 {
			t.Errorf("max_tokens %d, want the session's document's 24,000", got)
		}
	})
	t.Run("none", func(t *testing.T) {
		c, err := send(t, Options{})
		if paths, _ := c.requests(); err == nil || len(paths) != 0 {
			t.Errorf("with no document: %v, %d requests; want it refused before sending", err, len(paths))
		}
	})
	t.Run("saved_caps", func(t *testing.T) {
		fresh := []llm.LimitObservation{savedCap(savedModel, 16000, time.Hour, "", "")}
		c, err := send(t, Options{ModelData: session, Observations: fresh})
		if err != nil {
			t.Fatalf("the call failed: %v", err)
		}
		if got := sentMaxTokens(t, c); got != 16000 {
			t.Errorf("max_tokens %d, want the saved 16,000 over the session's document's 24,000", got)
		}
	})
}

// M73: an OpenCode model's requests go to the route the provider's document
// names (its provider.npm); with none, to the gateway's table's (0093-MADR
// Q4, D6).
func TestOpenCodeRoutesFromSessionModelData_0072(t *testing.T) {
	const model = "gpt-5.5" // the table routes it to responses
	doc := modelData(t, `{"opencode":{"models":{"`+model+`":{"name":"GPT 5.5","provider":{"npm":"@ai-sdk/openai-compatible"}}}}}`)
	for _, c := range []struct {
		name string
		data *ModelData
		path string
	}{{"document", doc, "/chat/completions"}, {"none", nil, "/responses"}} {
		t.Run(c.name, func(t *testing.T) {
			w := &cannedWire{}
			p, err := New("opencode-zen", Options{APIKey: "k", Model: model, HTTPClient: &http.Client{Transport: w}, Retry: &llm.Retry{Off: true},
				ModelData: c.data})
			if err != nil {
				t.Fatal(err)
			}
			req := wireRequest()
			req.Model, req.MaxTokens = model, 1000
			_, _ = drain(t, p, req) //nolint:errcheck // the route is in the request; the canned reply may not be this route's
			if paths, _ := w.requests(); len(paths) != 1 || !strings.HasSuffix(paths[0], c.path) {
				t.Errorf("requests %q; want one to %s", paths, c.path)
			}
		})
	}
}

// M62 (the listing): a listing reads the document Options.ModelData hands
// in and requests none: handed, its facts are the document's; none handed,
// they are not known.
func TestListingGetsModelData_0072(t *testing.T) {
	for _, c := range []struct {
		name   string
		data   *ModelData
		vision []string
	}{{"handed", modelData(t, factDoc), []string{"gpt-6-luna"}}, {"none", nil, nil}} {
		t.Run(c.name, func(t *testing.T) {
			run := metadataRuns.Add(1)
			w, o := cannedClient(map[string]route{openaiModels: {status: http.StatusOK, body: factModels}})
			o.ModelData = c.data
			l, err := List(t.Context(), "openai", fmt.Sprintf("sk-handin-%d-%d", os.Getpid(), run), o)
			if err != nil || !l.Live {
				t.Fatalf("listing %+v (%v), want live", l, err)
			}
			if got := SearchListing(l, "vision"); !slices.Equal(got, c.vision) {
				t.Errorf("vision %q, want %q", got, c.vision)
			}
			seen, _ := w.requests()
			if len(seen) != 1 || !strings.HasSuffix(strings.Fields(seen[0])[1], "/models") {
				t.Errorf("requests %q, want the listing's alone: no document is fetched", seen)
			}
		})
	}
}
