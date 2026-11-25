package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

	"github.com/maccavelli/gobble-cli/llm"
	gcatalog "github.com/maccavelli/gobble-cli/llm/catalog"
)

// The tests of 0071-PLAN's Phase 2b in llm/provider: discovery, and the
// limits a lookup reads with their sources (0071-MADR P17, P28, P33;
// 0066-MADR D1, D3, D4, D9 as amended 2026-11-25). Each serves its own
// listing; none asserts a value the embedded snapshot states (0066-PLAN's
// rule).

// limitWire is an Anthropic service in process: GET /v1/models answers
// listing after delay, POST /v1/messages streams a reply, or answers refused
// with 400 when it is set. It records each request in order, "GET models" or
// "POST messages", with its body.
type limitWire struct {
	listing string
	delay   time.Duration
	refused string

	mu     sync.Mutex
	order  []string
	bodies []string
}

func (w *limitWire) RoundTrip(r *http.Request) (*http.Response, error) {
	var body []byte
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = b
	}
	reply := func(status int, ct, b string) (*http.Response, error) {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {ct}}, Body: io.NopCloser(strings.NewReader(b)),
			Request: r}, nil
	}
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/v1/models"):
		w.record("GET models", "")
		select {
		case <-time.After(w.delay):
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
		return reply(http.StatusOK, "application/json", w.listing)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/messages"):
		w.record("POST messages", string(body))
		if w.refused != "" {
			return reply(http.StatusBadRequest, "application/json", w.refused)
		}
		s, _ := wireStreamFor(r.URL.Path, false)
		return reply(http.StatusOK, "text/event-stream", s)
	}
	w.record(r.Method+" "+r.URL.Path, "")
	return reply(http.StatusNotFound, "application/json", `{"type":"error","error":{"type":"not_found_error","message":"no"}}`)
}

func (w *limitWire) record(what, body string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.order = append(w.order, what)
	w.bodies = append(w.bodies, body)
}

func (w *limitWire) seen() ([]string, []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.order), slices.Clone(w.bodies)
}

// claudeListingOf is Anthropic's listing of model, its max_tokens and
// max_input_tokens each left out when 0.
func claudeListingOf(model string, output, input int) string {
	e := `{"id":"` + model + `","type":"model","display_name":"` + model + `","created_at":"2026-10-01T00:00:00Z"`
	if output > 0 {
		e += fmt.Sprintf(`,"max_tokens":%d`, output)
	}
	if input > 0 {
		e += fmt.Sprintf(`,"max_input_tokens":%d`, input)
	}
	return `{"data":[` + e + `}],"has_more":false,"first_id":"` + model + `","last_id":"` + model + `"}`
}

// mjOf is models.json doc, parsed.
func mjOf(t *testing.T, doc string) *gcatalog.ModelsJSON {
	t.Helper()
	mj, _, err := gcatalog.ParseModelsJSON([]byte(doc), SDKVersion(), (*Set)(nil).IDs())
	if err != nil {
		t.Fatal(err)
	}
	return mj
}

// claudeOpts are the options of a Claude run over w with key and its own
// cache, closed with the test.
func claudeOpts(t *testing.T, w *limitWire, key string) Options {
	t.Helper()
	c := &Cache{}
	t.Cleanup(func() {
		if err := c.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return Options{APIKey: key, HTTPClient: &http.Client{Transport: w}, Cache: c, Retry: &llm.Retry{Off: true}}
}

// closedWith is a cache closed when t ends.
func closedWith(t *testing.T) *Cache {
	t.Helper()
	c := &Cache{}
	t.Cleanup(func() {
		if err := c.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return c
}

// servedWire answers GET requests whose path ends in path with body, and
// every other request 404, and says whether it served one.
type servedWire struct {
	path, body string
	mu         sync.Mutex
	hits       int
}

func (w *servedWire) RoundTrip(r *http.Request) (*http.Response, error) {
	status, body := http.StatusNotFound, `{"error":"not found"}`
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, w.path) {
		w.mu.Lock()
		w.hits++
		w.mu.Unlock()
		status, body = http.StatusOK, w.body
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)),
		Request: r}, nil
}

func (w *servedWire) served() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.hits > 0
}

// refuseStated has the service refuse a streamed call of model at asked,
// stating its cap, through New over o, so the SDK keeps the cap for the
// cache's process (0071-MADR P15, F24).
func refuseStated(t *testing.T, o Options, w *limitWire, model string, asked, stated int) {
	t.Helper()
	w.refused = fmt.Sprintf(`{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: %d > %d, `+
		`which is the maximum allowed number of output tokens for %s"}}`, asked, stated, model)
	defer func() { w.refused = "" }()
	o.Model = model
	p, err := New("claude", o)
	if err != nil {
		t.Fatal(err)
	}
	req := wireRequest()
	req.Model, req.MaxTokens = model, asked
	_, err = drain(t, p, req)
	if ae, ok := errors.AsType[*llm.APIError](err); !ok || ae.Observed.Tokens != stated {
		t.Fatalf("refused: %v, want an APIError observing %d", err, stated)
	}
}

// statedLayer is a caller's layer named "service-stated" that states
// model's output as tokens: the layer 0071-MADR P24 builds from the saved
// caps, as P28's hold reads it.
type statedLayer struct {
	model  string
	tokens int
}

func (statedLayer) Name() string { return sourceServiceStated }

func (l statedLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	if id != l.model {
		return catalog.ModelInfo{}, false
	}
	return catalog.ModelInfo{MaxOutputTokens: l.tokens}.Known(catalog.FieldMaxOutputTokens), true
}

// otherFrom is f's other statement by source, or a zero fact.
func otherFrom(f llm.LimitFact, source string) llm.LimitFact {
	for _, o := range f.Others {
		if o.Source == source {
			return o
		}
	}
	return llm.LimitFact{}
}

// A call's cap is the streamed output the SDK's limits choose, its source
// told apart, and its window the window else the input limit (0071-MADR
// P17, P28, P33): a cap a service stated for a streamed call bounds the
// streamed output alone; models.json's maxTokens over a listing is the
// user's; Pi's default for a model models.json defines is assumed; nothing
// known is unknown; Claude's input limit is the window; a model only the
// cached listing names is read with the key; models.json above a cap a
// service stated in this process, or on an earlier run, is held to it, the
// user's still.
func TestTurnCapFromLimits_0071(t *testing.T) {
	const model = "claude-test-1"
	t.Run("streamed", func(t *testing.T) {
		w := &limitWire{listing: claudeListingOf(model, 64000, 200000)}
		o := claudeOpts(t, w, "sk-streamed")
		if err := Discover(t.Context(), o, "claude", model, nil); err != nil {
			t.Fatal(err)
		}
		refuseStated(t, o, w, model, 64000, 16000)
		m := ModelInfoWith(o, "claude", model, nil)
		if s := m.Limits.StreamedOutput; s.Tokens != 16000 || s.Source != "observed" || m.Limits.Output.Tokens != 64000 ||
			m.MaxTokens != 64000 || m.MaxTokensFrom != llm.CapCatalog {
			t.Errorf("streamed %+v, output %+v, MaxTokens %d from %q; want the stream's stated 16,000 apart from the listing's 64,000",
				s, m.Limits.Output, m.MaxTokens, m.MaxTokensFrom)
		}
	})
	t.Run("user", func(t *testing.T) {
		w := &limitWire{listing: claudeListingOf(model, 128000, 200000)}
		o := claudeOpts(t, w, "sk-user")
		mj := mjOf(t, `{"providers":{"claude":{"modelOverrides":{"`+model+`":{"maxTokens":20000}}}}}`)
		if err := Discover(t.Context(), o, "claude", model, mj); err != nil {
			t.Fatal(err)
		}
		m := ModelInfoWith(o, "claude", model, mj)
		if s := m.Limits.StreamedOutput; s.Tokens != 20000 || s.Source != "models.json" || otherFrom(s, "listing:claude").Tokens != 128000 ||
			m.MaxTokensFrom != llm.CapUser {
			t.Errorf("%+v from %q, want models.json's 20,000 over the listing's 128,000, the user's", s, m.MaxTokensFrom)
		}
	})
	t.Run("assumed_default", func(t *testing.T) {
		mj := mjOf(t, `{"providers":{"claude":{"models":[{"id":"claude-mine"}]}}}`)
		m := ModelInfoWith(Options{}, "claude", "claude-mine", mj)
		if s := m.Limits.StreamedOutput; s.Tokens != 16384 || s.Source != "models.json" || m.MaxTokensFrom != llm.CapAssumed {
			t.Errorf("%+v from %q, want Pi's 16,384 from models.json, assumed", s, m.MaxTokensFrom)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		m := ModelInfoWith(Options{}, "claude", "claude-no-such-model", nil)
		if m.Limits.StreamedOutput.Known || m.MaxTokens != 0 || m.MaxTokensFrom != "" || m.ContextWindow != 0 {
			t.Errorf("%+v, MaxTokens %d from %q, window %d; want nothing known", m.Limits, m.MaxTokens, m.MaxTokensFrom, m.ContextWindow)
		}
	})
	t.Run("input_only", func(t *testing.T) {
		w := &limitWire{listing: claudeListingOf(model, 32000, 1000000)}
		o := claudeOpts(t, w, "sk-input")
		if err := Discover(t.Context(), o, "claude", model, nil); err != nil {
			t.Fatal(err)
		}
		m := ModelInfoWith(o, "claude", model, nil)
		if m.Limits.Window.Known || m.Limits.Input.Tokens != 1000000 || m.ContextWindow != 1000000 {
			t.Errorf("window %+v, input %+v, ContextWindow %d; want the input limit as the window", m.Limits.Window, m.Limits.Input,
				m.ContextWindow)
		}
	})
	t.Run("listing_only", func(t *testing.T) {
		w := &limitWire{listing: claudeListingOf(model, 32000, 200000)}
		o := claudeOpts(t, w, "sk-listing-only")
		if err := Discover(t.Context(), o, "claude", model, nil); err != nil {
			t.Fatal(err)
		}
		m := ModelInfoWith(o, "claude", model, nil)
		if s := m.Limits.StreamedOutput; s.Tokens != 32000 || s.Source != "listing:claude" || m.MaxTokens != 32000 {
			t.Errorf("%+v, MaxTokens %d; want the cached listing's 32,000, read with the key", s, m.MaxTokens)
		}
	})
	t.Run("user_over_observed", func(t *testing.T) {
		w := &limitWire{listing: claudeListingOf(model, 128000, 200000)}
		o := claudeOpts(t, w, "sk-over-observed")
		mj := mjOf(t, `{"providers":{"claude":{"modelOverrides":{"`+model+`":{"maxTokens":100000}}}}}`)
		if err := Discover(t.Context(), o, "claude", model, mj); err != nil {
			t.Fatal(err)
		}
		refuseStated(t, o, w, model, 100000, 16000)
		m := ModelInfoWith(o, "claude", model, mj)
		if s := m.Limits.StreamedOutput; s.Tokens != 16000 || s.Source != "observed" || otherFrom(s, "models.json").Tokens != 100000 ||
			m.MaxTokensFrom != llm.CapUser {
			t.Errorf("%+v from %q, want the service's 16,000 over models.json's 100,000, the user's held", s, m.MaxTokensFrom)
		}
	})
	t.Run("user_over_saved", func(t *testing.T) {
		mj := mjOf(t, `{"providers":{"claude":{"modelOverrides":{"`+model+`":{"maxTokens":100000}}}}}`)
		f, _ := mj.Facts("claude", model)
		m := capSourced(lookupModelWith(t.Context(), "claude", nil, model, "", "", "", nil, statedLayer{model, 16000}, modelsJSONLayer{mj}), f)
		if s := m.Limits.StreamedOutput; s.Tokens != 16000 || s.Source != sourceServiceStated || otherFrom(s, "models.json").Tokens != 100000 ||
			m.MaxTokens != 16000 || m.MaxTokensFrom != llm.CapUser {
			t.Errorf("%+v, MaxTokens %d from %q; want models.json's 100,000 held to the saved 16,000, the user's", s, m.MaxTokens, m.MaxTokensFrom)
		}
	})
}

// modelOf's limits carry each source; MaxTokens is the output's chosen value,
// ContextWindow the window's, and an unknown limit is 0, not known
// (0066-MADR D4, S35).
func TestModel_LimitsCarrySources_0066(t *testing.T) {
	l := catalog.Limits{
		Output: catalog.Fact{Tokens: 128000, Known: true, Conflicting: true, Evidence: []catalog.Evidence{
			{Source: "listing:claude", Tokens: 128000}, {Source: "models.dev", Tokens: 64000}}},
		Window: catalog.Fact{Tokens: 200000, Known: true, Evidence: []catalog.Evidence{{Source: "listing:claude", Tokens: 200000}}},
	}
	l.StreamedOutput = l.Output
	m := withLimits(llm.Model{Provider: "claude", ID: "x"}, l)
	out := m.Limits.Output
	if out.Tokens != 128000 || !out.Known || out.Source != "listing:claude" || len(out.Others) != 1 || out.Others[0].Source != "models.dev" ||
		out.Others[0].Tokens != 64000 || m.MaxTokens != 128000 || m.ContextWindow != 200000 {
		t.Errorf("output %+v, MaxTokens %d, window %d; want 128,000 from listing:claude, models.dev's 64,000 beside it", out, m.MaxTokens,
			m.ContextWindow)
	}
	if u := withLimits(llm.Model{}, catalog.Limits{}); u.MaxTokens != 0 || u.Limits.Output.Known || u.ContextWindow != 0 {
		t.Errorf("unknown: %+v, MaxTokens %d; want 0, not known", u.Limits.Output, u.MaxTokens)
	}
}

// models.json's maxTokens is the caller's statement, chosen over a listing,
// and the turn asks for it, basis user (0066-MADR S28, as amended).
func TestProvider_ModelsJSONIsCaller_0066(t *testing.T) {
	const model = "claude-test-1"
	w := &limitWire{listing: claudeListingOf(model, 128000, 200000)}
	o := claudeOpts(t, w, "sk-caller")
	mj := mjOf(t, `{"providers":{"claude":{"modelOverrides":{"`+model+`":{"maxTokens":20000}}}}}`)
	if err := Discover(t.Context(), o, "claude", model, mj); err != nil {
		t.Fatal(err)
	}
	m := ModelInfoWith(o, "claude", model, mj)
	if m.Limits.Output.Tokens != 20000 || m.Limits.Output.Source != "models.json" || m.MaxTokens != 20000 {
		t.Fatalf("output %+v, MaxTokens %d; want models.json's 20,000", m.Limits.Output, m.MaxTokens)
	}
	if lim := llm.FitOutput(m.Limits.StreamedOutput.Tokens, m.MaxTokensFrom, m.ContextWindow, 0); lim != (llm.OutputLimit{Tokens: 20000, Basis: llm.BasisUser}) {
		t.Errorf("the turn asks %+v, want 20,000, basis user", lim)
	}
}

// The built-in copy reaches a lookup's limits as the "models.dev" layer,
// registry evidence ranked below the provider's listing, which states the
// chosen value (0066-MADR S25 to S27, S42; 0071-MADR Q5 (1)). The copy's
// own value is not asserted.
func TestLimits_BuiltInCopyBelowListing_0066(t *testing.T) {
	const model = "claude-haiku-5-5" // a model the built-in copy states
	meta, err := snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if info, ok := meta.Model(llmprovider.ProviderClaude, model); !ok || !info.Has(catalog.FieldMaxOutputTokens) || info.MaxOutputTokens == 12345 {
		t.Fatalf("the built-in copy states no output for %s other than the listing's, so the test shows nothing", model)
	}
	w := &limitWire{listing: claudeListingOf(model, 12345, 200000)}
	o := claudeOpts(t, w, "sk-builtin")
	if err := Discover(t.Context(), o, "claude", model, nil); err != nil {
		t.Fatal(err)
	}
	m := ModelInfoWith(o, "claude", model, nil)
	if out := m.Limits.Output; out.Tokens != 12345 || out.Source != "listing:claude" || otherFrom(out, "models.dev").Tokens == 0 {
		t.Errorf("output %+v; want the listing's 12,345 chosen, the built-in copy beside it as models.dev", out)
	}
}

// Discovery and the provider's request read one listing: Claude in a
// workspace, discovery, then a turn: one listing request in all, and the
// turn carries the listing's value (0066-MADR S23).
func TestProvider_DiscoverySharesListingKey_0066(t *testing.T) {
	const model = "claude-test-1"
	w := &limitWire{listing: claudeListingOf(model, 32000, 200000)}
	o := claudeOpts(t, w, "sk-shared")
	o.Workspace = "wrkspc_1"
	if err := Discover(t.Context(), o, "claude", model, nil); err != nil {
		t.Fatal(err)
	}
	m := ModelInfoWith(o, "claude", model, nil)
	o.Model = model
	p, err := New("claude", o)
	if err != nil {
		t.Fatal(err)
	}
	req := wireRequest()
	req.Model, req.MaxTokens = model, llm.FitOutput(m.Limits.StreamedOutput.Tokens, m.MaxTokensFrom, m.ContextWindow, 0).Tokens
	if _, err := drain(t, p, req); err != nil {
		t.Fatal(err)
	}
	order, bodies := w.seen()
	if !slices.Equal(order, []string{"GET models", "POST messages"}) || !strings.Contains(bodies[1], `"max_tokens":32000`) {
		t.Errorf("requests %q, want one listing, then the turn carrying the listing's 32,000 (%.200s)", order, bodies[len(bodies)-1])
	}
}

// Concurrent discoveries, each with its own models.json, read their own
// value: nothing is shared but the cache (0066-MADR S24).
func TestProvider_ConcurrentSessionsDiscovery_0066(t *testing.T) {
	const model = "claude-test-1"
	w := &limitWire{listing: claudeListingOf(model, 128000, 200000)}
	o := claudeOpts(t, w, "sk-concurrent")
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := range 16 {
		want := []int{4096, 2048}[i%2]
		mj := mjOf(t, fmt.Sprintf(`{"providers":{"claude":{"modelOverrides":{"%s":{"maxTokens":%d}}}}}`, model, want))
		wg.Go(func() {
			if err := Discover(t.Context(), o, "claude", model, mj); err != nil {
				errs <- err
				return
			}
			if m := ModelInfoWith(o, "claude", model, mj); m.Limits.StreamedOutput.Tokens != want {
				errs <- fmt.Errorf("session %d: %d, want its own %d", i, m.Limits.StreamedOutput.Tokens, want)
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// The window compaction reads (0066-MADR D9, S22, S30, S31): Claude's input
// limit when only that is stated; the ChatGPT plan backend's context_window;
// the context Ollama serves a loaded model, read with no key; and nothing
// known, none.
func TestCompaction_WindowFromLimits_0066(t *testing.T) {
	t.Run("claude_input_only", func(t *testing.T) {
		w := &limitWire{listing: claudeListingOf("claude-test-1", 32000, 1000000)}
		o := claudeOpts(t, w, "sk-window")
		if err := Discover(t.Context(), o, "claude", "claude-test-1", nil); err != nil {
			t.Fatal(err)
		}
		if got := ModelInfoWith(o, "claude", "claude-test-1", nil).ContextWindow; got != 1000000 {
			t.Errorf("window %d, want the input limit 1,000,000", got)
		}
	})
	t.Run("chatgpt_plan", func(t *testing.T) {
		w := &servedWire{path: "/models", body: `{"models":[{"slug":"gpt-6-luna","context_window":272000}]}`}
		logins := &memLogins{logins: map[string]*Login{"openai": {Access: "a", Refresh: "r", Expires: time.Now().Add(time.Hour),
			Issuer: auth.DefaultOpenAIIssuer, ClientID: "c", Kind: string(auth.SessionChatGPTPlan)}}}
		o := Options{Logins: logins, HTTPClient: &http.Client{Transport: w}, Cache: closedWith(t)}
		if err := Discover(t.Context(), o, "openai", "gpt-6-luna", nil); err != nil {
			t.Fatal(err)
		}
		if got := ModelInfoWith(o, "openai", "gpt-6-luna", nil).ContextWindow; got != 272000 || !w.served() {
			t.Errorf("window %d (backend read %v), want the plan backend's 272,000", got, w.served())
		}
	})
	t.Run("ollama_served", func(t *testing.T) {
		w := &servedWire{path: "/api/ps", body: `{"models":[{"name":"llama3:latest","model":"llama3:latest","context_length":8192}]}`}
		o := Options{HTTPClient: &http.Client{Transport: w}, Cache: closedWith(t)}
		if err := Discover(t.Context(), o, "ollama", "llama3", nil); err != nil {
			t.Logf("discovery: %v (the listing is not served)", err)
		}
		if got := ModelInfoWith(o, "ollama", "llama3", nil).ContextWindow; got != 8192 || !w.served() {
			t.Errorf("window %d (served read %v), want the 8,192 the server serves", got, w.served())
		}
	})
	t.Run("nothing_known", func(t *testing.T) {
		if got := ModelInfoWith(Options{}, "claude", "claude-no-such-model", nil).ContextWindow; got != 0 {
			t.Errorf("window %d, want 0", got)
		}
	})
}

// headerWire records each request's organization header and answers 404.
type headerWire struct {
	mu   sync.Mutex
	orgs []string
}

func (w *headerWire) RoundTrip(r *http.Request) (*http.Response, error) {
	w.mu.Lock()
	w.orgs = append(w.orgs, r.Header.Get("X-Kilocode-Organizationid"))
	w.mu.Unlock()
	return &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"error":"not found"}`)), Request: r}, nil
}

// Discovery names no Kilo organization: gobble passes none, so a Kilo cap's
// scope is empty (0066-MADR S43; 0071-MADR F25).
func TestDiscover_KiloNoOrganization_0066(t *testing.T) {
	w := &headerWire{}
	o := Options{APIKey: "sk-kilo", HTTPClient: &http.Client{Transport: w}, Cache: closedWith(t)}
	_ = Discover(t.Context(), o, "kilo", "deepseek/deepseek-v4.1-flash", nil) //nolint:errcheck // only the requests matter
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.orgs) == 0 || slices.ContainsFunc(w.orgs, func(s string) bool { return s != "" }) {
		t.Errorf("organization headers %q, want requests naming none", w.orgs)
	}
}

// A discovery whose listing fails, or stalls past the SDK's 10-second bound,
// ends with an error, and the lookup after it gives what is cached, so the
// call goes on (0066-MADR S7, S8). The stall runs in a synctest bubble, so
// the bound passes at once.
func TestSession_DiscoveryFails_0066(t *testing.T) {
	const model = "claude-test-1"
	t.Run("fails", func(t *testing.T) {
		w := &servedWire{path: "/never", body: "{}"} // every request 404
		o := Options{APIKey: "sk-fails", HTTPClient: &http.Client{Transport: w}, Cache: closedWith(t)}
		if err := Discover(t.Context(), o, "claude", model, nil); err == nil {
			t.Error("discovery over a failing listing: no error")
		}
		if m := ModelInfoWith(o, "claude", model, nil); m.Limits.StreamedOutput.Known {
			t.Errorf("%+v, want nothing known, so the call asks what it would with nothing cached", m.Limits.StreamedOutput)
		}
	})
	t.Run("slow", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			c := &Cache{}
			o := Options{APIKey: "sk-slow", HTTPClient: &http.Client{Transport: newHangingListing()}, Cache: c}
			start := time.Now()
			err := Discover(context.Background(), o, "claude", model, nil)
			if waited := time.Since(start); err == nil || waited < 10*time.Second || waited > 11*time.Second {
				t.Errorf("discovery ended after %v with %v, want the SDK's 10-second bound and an error", waited, err)
			}
			if m := ModelInfoWith(o, "claude", model, nil); m.Limits.StreamedOutput.Known {
				t.Errorf("%+v, want nothing known", m.Limits.StreamedOutput)
			}
			if err := c.Close(context.Background()); err != nil {
				t.Error(err)
			}
		})
	})
}
