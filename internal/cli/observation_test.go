package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The tests of 0071-PLAN's Phase 2c in internal/cli: limit-observations.json,
// the caps services stated, kept for a day (0066-MADR D7, S12 to S14;
// 0071-MADR P24 to P27); gobble models and gobble models info (0066-MADR D5,
// S15 to S18); the unknown limit's notice (S10).

// writeObservations writes limit-observations.json with body, the
// observations' JSON array.
func writeObservations(t *testing.T, body string) {
	t.Helper()
	path, err := stateFile(observationFileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"observations":`+body+`}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

// observationJSON is one saved cap's JSON, seen ago.
func observationJSON(id, model, scope string, tokens int, ago time.Duration) string {
	return fmt.Sprintf(`{"provider":%q,"model":%q,"scope":%q,"tokens":%d,"seen":%q}`, id, model, scope, tokens,
		time.Now().Add(-ago).UTC().Format(time.RFC3339Nano))
}

// savedFile is limit-observations.json's caps, as provider/model scope tokens.
func savedFile(t *testing.T) []string {
	t.Helper()
	path, err := stateFile(observationFileName)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f observationFile
	if err := jsonx.UnmarshalStrict(b, &f); err != nil {
		t.Fatalf("%s: %v\n%s", observationFileName, err, b)
	}
	var out []string
	for _, o := range f.Observations {
		out = append(out, fmt.Sprintf("%s/%s %s %d", o.Provider, o.Model, o.Scope, o.Tokens))
	}
	return out
}

// An unreadable limit-observations.json is one warning, and the provider is
// built with none handed back; a readable one with bad entries (no provider,
// no model, a cap below 1, no time) drops them with one warning and hands the
// good ones back (0066-MADR S13; 0071-MADR P27).
func TestAgent_UnreadableObservations_0066(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		e, _ := modelEnv(t)
		t.Setenv("OPENAI_API_KEY", "test-key")
		path, err := stateFile(observationFileName)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, session := range []string{"s1", "s2"} {
			if p, err := modelProvider(e)(session, llm.Retry{Off: true}); err != nil || p == nil {
				t.Fatalf("provider %s: %v, %v; want it built", session, p, err)
			}
		}
		stderr := e.out.err.(*bytes.Buffer)
		if n := strings.Count(stderr.String(), "Warning: "+observationFileName); n != 1 {
			t.Errorf("%d warnings, want 1; stderr %q", n, stderr.String())
		}
		if got := e.providerObservations("openai"); len(got) != 0 {
			t.Errorf("handed %+v, want none", got)
		}
	})
	t.Run("bad_entry", func(t *testing.T) {
		e, _ := modelEnv(t)
		good := observationJSON("kilo", "z-ai/glm-5.3", "", 16000, time.Hour)
		writeObservations(t, "["+strings.Join([]string{
			observationJSON("", "z-ai/glm-5.3", "", 16000, time.Hour),
			observationJSON("kilo", "", "", 16000, time.Hour),
			observationJSON("kilo", "z-ai/glm-5.3", "", 0, time.Hour),
			`{"provider":"kilo","model":"z-ai/glm-5.3","tokens":16000,"seen":"0001-01-01T00:00:00Z"}`,
			good,
		}, ",")+"]")
		got := e.providerObservations("kilo")
		if len(got) != 1 || got[0].Model != "z-ai/glm-5.3" || got[0].Tokens != 16000 || got[0].Seen.IsZero() {
			t.Errorf("handed %+v, want the one good entry", got)
		}
		e.providerObservations("kilo")
		stderr := e.out.err.(*bytes.Buffer)
		if n := strings.Count(stderr.String(), "Warning: "+observationFileName); n != 1 || !strings.Contains(stderr.String(), ": 4 saved output caps lack") {
			t.Errorf("%d warnings, want one for the four bad entries; stderr %q", n, stderr.String())
		}
	})
}

// A cap a service stated is saved with its provider, model, scope, cap and
// time, newest first; ten kept per provider, base URL and scope; another
// scope's kept apart. At the next start each is handed back to its provider
// and lookups, where the scope the run's credential is in picks its own; one
// saved 25 hours earlier is dropped (0066-MADR D7, S12, S14; 0071-MADR P24).
func TestAgent_SavesLearnedCap_0066(t *testing.T) {
	e, _ := modelEnv(t)
	seen := time.Now()
	for i := range 11 {
		e.rememberObservation("kilo", llm.LimitObservation{Model: fmt.Sprintf("m%d", i), Scope: "org-a", Tokens: 1000 + i, Seen: seen})
	}
	e.rememberObservation("kilo", llm.LimitObservation{Model: "m0", Scope: "org-b", Tokens: 64000, Seen: seen})
	e.rememberObservation("kilo", llm.LimitObservation{Model: "m10", Scope: "org-a", Tokens: 900, Seen: seen})
	got := savedFile(t)
	var a []string
	for _, s := range got {
		if strings.Contains(s, " org-a ") {
			a = append(a, s)
		}
	}
	if len(a) != 10 || a[0] != "kilo/m10 org-a 900" || slices.Contains(a, "kilo/m0 org-a 1000") || !slices.Contains(got, "kilo/m0 org-b 64000") {
		t.Errorf("saved %q; want org-a's ten newest, m10 replaced at 900 and m0 dropped, org-b's kept apart", got)
	}
	// The next start: a run that reads the file afresh, a Claude key's
	// workspace its scope.
	next, _ := modelEnv(t)
	auth := filepath.Join(os.Getenv("GOBBLE_HOME"), "auth.json")
	if err := os.WriteFile(auth, []byte(`{"claude": {"type": "api_key", "key": "sk-ant-test-only", "workspace": "wrkspc_a"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeObservations(t, "["+strings.Join([]string{
		observationJSON("claude", "claude-saved-1", "wrkspc_a", 16000, time.Hour),
		observationJSON("claude", "claude-saved-1", "wrkspc_b", 12000, time.Hour),
		observationJSON("claude", "claude-saved-2", "wrkspc_a", 8000, 25*time.Hour),
	}, ",")+"]")
	o := next.lookupOptions("claude")
	o.APIKey = "" // the lookup reads no listing here; its scope is still the workspace
	if m := provider.ModelInfoWith(o, "claude", "claude-saved-1", nil); m.MaxTokens != 16000 || m.Limits.Output.Source != "service-stated" {
		t.Errorf("the lookup's output %+v, want workspace a's 16,000, service-stated", m.Limits.Output)
	}
	if got := next.providerObservations("claude"); len(got) != 2 {
		t.Errorf("handed %+v, want the two under a day old", got)
	}
	// An llm.APIError whose Observed the SDK set, in a run: saved with its
	// provider, model, scope, cap and time (S12; 0071-MADR P43).
	t.Run("apierror", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("OPENAI_API_KEY", "sk-test-only")
		withModel(t, llmtest.NewScript(lunaRefusal(12000), llmtest.Text("ok")))
		before := time.Now().Add(-time.Minute)
		mustRun(t, home, "-p", "hello")
		if got := savedFile(t); !slices.Equal(got, []string{"openai/gpt-6-luna  12000"}) {
			t.Fatalf("the file holds %q, want openai's 12,000 for gpt-6-luna", got)
		}
		if o := envAt(t).providerObservations("openai"); len(o) != 1 || o[0].Seen.Before(before) {
			t.Errorf("saved %+v, want it seen now", o)
		}
	})
}

// Each saved cap is handed only to its own provider: an opencode-zen cap is
// not handed to claude (0071-MADR P26).
func TestSavedCapPerProvider_0071(t *testing.T) {
	e, _ := modelEnv(t)
	writeObservations(t, "["+observationJSON("opencode-zen", "claude-saved-1", "", 16000, time.Hour)+"]")
	if got := e.providerOptions("claude", provider.Options{}).Observations; len(got) != 0 {
		t.Errorf("claude's provider is handed %+v, want none", got)
	}
	if got := e.lookupOptions("claude").Observations; len(got) != 0 {
		t.Errorf("claude's lookups are handed %+v, want none", got)
	}
	if got := e.providerOptions("opencode-zen", provider.Options{}).Observations; len(got) != 1 || got[0].Tokens != 16000 {
		t.Errorf("opencode-zen's provider is handed %+v, want its own 16,000", got)
	}
	if got := e.lookupOptions("opencode-zen").Observations; len(got) != 1 {
		t.Errorf("opencode-zen's lookups are handed %+v, want its own", got)
	}
}

// A completed call whose output exceeds a saved cap of its provider, model,
// base URL and scope drops the cap; one at or below it keeps it (0071-MADR
// P25; go-llmprovider-sdk 0089-MADR DL39).
func TestSavedCapDroppedWhenExceeded_0071(t *testing.T) {
	e, _ := modelEnv(t)
	writeObservations(t, "["+strings.Join([]string{
		observationJSON("kilo", "z-ai/glm-5.3", "", 16000, time.Hour),
		observationJSON("together", "z-ai/glm-5.3", "", 16000, time.Hour),
	}, ",")+"]")
	e.dropExceeded("kilo", "z-ai/glm-5.3", 15000)
	if got := e.providerObservations("kilo"); len(got) != 1 {
		t.Fatalf("15,000 output tokens dropped the 16,000 cap: %+v", got)
	}
	e.dropExceeded("kilo", "Z-AI/GLM-5.3", 20000)
	if got := e.providerObservations("kilo"); len(got) != 0 {
		t.Errorf("handed %+v after 20,000 output tokens, want the cap dropped", got)
	}
	if got := savedFile(t); !slices.Equal(got, []string{"together/z-ai/glm-5.3  16000"}) {
		t.Errorf("the file holds %q, want together's cap alone", got)
	}
	t.Run("run", savedCapDroppedRun)
}

// gobble models reads cached limits only: it discovers nothing, so a stalled
// listing never holds it, and an unknown limit is "-" (0066-MADR D5, S17).
func TestModelsCmd_TableCachedOnly_0066(t *testing.T) {
	c := listEnv(t)
	var discovered []string
	old := discoverLimits
	discoverLimits = func(_ context.Context, _ provider.Options, id, model string, _ *catalog.ModelsJSON) error {
		discovered = append(discovered, id+"/"+model)
		return nil
	}
	t.Cleanup(func() { discoverLimits = old })
	r := c.run("models", "list")
	if r.code != ExitOK || len(discovered) != 0 {
		t.Fatalf("exit %d, discoveries %q; want none", r.code, discovered)
	}
	for _, row := range rowsOf(r.stdout) {
		if len(row) > 3 && row[1] == "live-only-1" && (row[2] != "-" || row[3] != "-") {
			t.Errorf("live-only-1's row %q, want its unknown limits as -", row)
		}
	}
}

// gobble models info discovers the model's limits, then prints each with
// its source, each other source on a line of its own, "(stale)" past its
// source's freshness, and whether its wire requires an output limit
// (0066-MADR D5, S15, S16, S18).
func TestModelsCmd_InfoShowsEvidence_0066(t *testing.T) {
	c := newCfgEnv(t)
	var order []string
	oldD, oldF := discoverLimits, modelFacts
	discoverLimits = func(_ context.Context, _ provider.Options, id, model string, _ *catalog.ModelsJSON) error {
		order = append(order, "discover "+id+"/"+model)
		return nil
	}
	modelFacts = func(_ provider.Options, id, model string, _ *catalog.ModelsJSON) llm.Model {
		order = append(order, "facts")
		return llm.Model{Provider: id, ID: model, ContextWindow: 200000, MaxTokens: 20000, Limits: llm.Limits{
			Output: llm.LimitFact{Tokens: 20000, Known: true, Conflicting: true, Source: "models.json",
				Others: []llm.LimitFact{{Tokens: 128000, Known: true, Source: "listing:claude"}}},
			Window:   llm.LimitFact{Tokens: 200000, Known: true, Source: "listing:claude", Stale: true},
			Required: true}}
	}
	t.Cleanup(func() { discoverLimits, modelFacts = oldD, oldF })
	r := c.run("models", "info", "claude/claude-test-1")
	for _, want := range []string{"context   200000 (listing:claude) (stale)\n", "max-out   20000 (models.json)\n          listing:claude: 128000\n",
		"limit     required on its wire\n"} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("stdout lacks %q:\n%s", want, r.stdout)
		}
	}
	if !slices.Equal(order, []string{"discover claude/claude-test-1", "facts"}) {
		t.Errorf("order %q, want the discovery, then the facts it found", order)
	}
}

// When discovery ends with the model's output limit unknown, print mode
// says model.limit.unknown before the reply, in its text and its JSON
// (0066-MADR D10 as amended, S10; 0071-MADR V7).
func TestNotice_LimitUnknown_0066(t *testing.T) {
	// The terminal's default personality is dry.
	line := "Nothing states gpt-6-luna's output limit, so gobble assumes 16,384 tokens, fitted to the window."
	unknown := func(t *testing.T) {
		t.Helper()
		t.Setenv("OPENAI_API_KEY", "sk-test-only")
		oldD, oldF := discoverLimits, modelFacts
		discoverLimits = func(context.Context, provider.Options, string, string, *catalog.ModelsJSON) error { return nil }
		modelFacts = func(_ provider.Options, id, model string, _ *catalog.ModelsJSON) llm.Model {
			return llm.Model{Provider: id, ID: model}
		}
		t.Cleanup(func() { discoverLimits, modelFacts = oldD, oldF })
	}
	t.Run("headless_text", func(t *testing.T) {
		unknown(t)
		withModel(t, llmtest.NewScript(llmtest.Text("ok")))
		r := runMain(t, nil, "-p", "hello")
		if r.code != ExitOK || !strings.Contains(r.stdout, line) || strings.Index(r.stdout, line) > strings.Index(r.stdout, "ok") {
			t.Errorf("exit %d, stdout %q; want the line before the reply", r.code, r.stdout)
		}
	})
	t.Run("headless_json", func(t *testing.T) {
		unknown(t)
		withModel(t, llmtest.NewScript(llmtest.Text("ok")))
		r := runMain(t, nil, "-p", "--output-format", "json", "hello")
		var res printResult
		if err := jsonx.Unmarshal([]byte(strings.TrimSpace(r.stdout)), &res); err != nil {
			t.Fatalf("stdout %q: %v", r.stdout, err)
		}
		if !strings.HasPrefix(res.Text, line) {
			t.Errorf("result text %q, want the line first", res.Text)
		}
	})
}

// envAt is modelEnv's runEnv over the GOBBLE_HOME already set: a later run
// in the same state directory.
func envAt(t *testing.T) *runEnv {
	t.Helper()
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out, stderr: &syncBuffer{}}}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	ownListings(t, e)
	e.processSettings("")
	return e
}

// lunaRefusal is a refusal of gpt-6-luna's limit in Kilo's range form, which
// only the SDK reads, with the cap it validated as Observed.
func lunaRefusal(stated int) llmtest.Turn {
	msg := fmt.Sprintf("The max_tokens parameter is illegal.：限制数值范围[1,%d]", stated)
	return llmtest.Turn{Err: &llm.APIError{Status: 400, Message: msg, Err: fmt.Errorf("invalid request: %s", msg),
		Observed: llm.LimitObservation{Model: "gpt-6-luna", Tokens: stated, Seen: time.Now()}}}
}

// A run whose call is refused with a cap the SDK observed sends it again at
// that cap, saves it to limit-observations.json through the agent's
// SaveObservation hook, and says so; the next run's provider, lookups and
// limits are given it back (0071-MADR P19, P43, the owner's "Two Options
// hooks (Recommended)"; 0066-MADR S12, S14).
func TestRefusedCapLearned_0071(t *testing.T) {
	home := t.TempDir()
	t.Setenv("OPENAI_API_KEY", "sk-test-only")
	s := llmtest.NewScript(lunaRefusal(16000), llmtest.Text("ok"))
	withModel(t, s)
	r := mustRun(t, home, "-p", "hello")
	if caps := capsOf(s.Requests()); len(caps) != 2 || caps[1] != 16000 {
		t.Fatalf("calls asking %v, want the stated 16,000 second", caps)
	}
	if want := "The service says gpt-6-luna stops at 16,000 output tokens."; !strings.Contains(r.stdout, want) {
		t.Errorf("stdout %q lacks %q", r.stdout, want)
	}
	if got := savedFile(t); !slices.Equal(got, []string{"openai/gpt-6-luna  16000"}) {
		t.Fatalf("the file holds %q, want openai's 16,000 for gpt-6-luna", got)
	}
	next := envAt(t)
	for name, o := range map[string]provider.Options{"provider": next.providerOptions("openai", provider.Options{}),
		"lookups": next.lookupOptions("openai")} {
		if got := o.Observations; len(got) != 1 || got[0].Tokens != 16000 || got[0].Model != "gpt-6-luna" {
			t.Errorf("the next run's %s are given %+v, want the saved 16,000", name, got)
		}
	}
	o := next.lookupOptions("openai")
	o.APIKey = "" // the layer alone: no listing is read
	if l := provider.CachedLimits(o, "openai", "gpt-6-luna", nil); l.StreamedOutput.Tokens != 16000 || l.StreamedOutput.Source != "service-stated" {
		t.Errorf("the next run's limits %+v, want the saved 16,000, service-stated", l.StreamedOutput)
	}
}

// A refusal with no Observed, a window form, is sent again at its room, and
// nothing is saved: the run writes no limit-observations.json (0071-MADR
// P20, P43).
func TestRefusalNoStatedCap_0071(t *testing.T) {
	home := t.TempDir()
	t.Setenv("OPENAI_API_KEY", "sk-test-only")
	msg := "This endpoint's maximum context length is 131072 tokens. However, you requested about 160002 tokens " +
		"(100000 of text input, 60002 in the output)."
	s := llmtest.NewScript(llmtest.Turn{Err: &llm.APIError{Status: 400, Message: msg, Err: fmt.Errorf("%w: %s", llm.ErrContextOverflow, msg)}},
		llmtest.Text("ok"))
	withModel(t, s)
	mustRun(t, home, "-p", "hello")
	if n := len(s.Requests()); n != 2 {
		t.Fatalf("%d calls, want the call sent again at its room", n)
	}
	path, err := stateFile(observationFileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s: %v; want no file: no cap was observed", observationFileName, err)
	}
}

// A cap saved at models.json's base URL is stated to the lookups (and so to
// discovery) and to the agent's CachedLimits hook, which read with that base
// URL; a cap saved at the default base is not handed to it (0071-MADR P24;
// 0071-PLAN's review deviation of 2026-11-25, R1).
func TestSavedCapAtModelsJSONBase_0071(t *testing.T) {
	e, _ := modelEnv(t)
	writeModelsJSON(t, `{"providers":{"claude":{"baseUrl":"https://gw.example/v1/"}}}`)
	seen := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	writeObservations(t, fmt.Sprintf(`[{"provider":"claude","model":"claude-saved-1","baseURL":"https://gw.example/v1","tokens":16000,"seen":%q},`+
		`{"provider":"claude","model":"claude-saved-2","tokens":12000,"seen":%q}]`, seen, seen))
	o := e.lookupOptions("claude")
	hook := e.agentOptions().CachedLimits
	for model, want := range map[string]int{"claude-saved-1": 16000, "claude-saved-2": 0} {
		for name, f := range map[string]llm.LimitFact{"lookup": provider.ModelInfoWith(o, "claude", model, nil).Limits.Output,
			"CachedLimits hook": hook("claude", model).Output} {
			if stated := f.Source == "service-stated"; stated != (want > 0) || (stated && f.Tokens != want) {
				t.Errorf("%s of %s: output %+v; want service-stated %v (%d)", name, model, f, want > 0, want)
			}
		}
	}
}

// A lookup lists at models.json's base URL with models.json's headers, as
// the run's listing does: a proxy that refuses a request without its header
// serves discovery's listing, and the listed cap is stated (0041-MADR D1,
// D2; 0071-MADR P47; 0071-PLAN's review deviation of 2026-11-25, R8).
func TestLookupSendsModelsJSONHeaders_0071(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Proxy-Key")
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path+" key="+got)
		mu.Unlock()
		if got != "proxy-secret" {
			http.Error(w, `{"type":"error","error":{"type":"authentication_error","message":"no proxy key"}}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"id":"claude-proxied-1","type":"model","display_name":"claude-proxied-1",`+
			`"created_at":"2026-10-01T00:00:00Z","max_tokens":24000,"max_input_tokens":200000}],"has_more":false,`+
			`"first_id":"claude-proxied-1","last_id":"claude-proxied-1"}`)
	}))
	t.Cleanup(srv.Close)
	e, _ := modelEnv(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-only")
	writeModelsJSON(t, `{"providers":{"claude":{"baseUrl":"`+srv.URL+`/v1","headers":{"X-Proxy-Key":"proxy-secret"}}}}`)
	o := e.lookupOptions("claude")
	if err := provider.Discover(t.Context(), o, "claude", "claude-proxied-1", e.modelsJSON()); err != nil {
		t.Errorf("discovery: %v; want the proxy's listing read", err)
	}
	if f := provider.ModelInfoWith(o, "claude", "claude-proxied-1", e.modelsJSON()).Limits.Output; f.Tokens != 24000 || f.Source != "listing:claude" {
		t.Errorf("the lookup's output %+v; want the proxy's listed 24,000", f)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) == 0 || slices.ContainsFunc(seen, func(s string) bool { return !strings.HasSuffix(s, " key=proxy-secret") }) {
		t.Errorf("requests %q; want each with models.json's header", seen)
	}
}

// A saved cap seen more than five minutes after now is not fresh: the store
// drops it on load, as an expired one; one within the skew is handed
// (0066-MADR D7, S14; 0071-PLAN's review deviation of 2026-11-25, R2).
func TestSavedCapSeenAhead_0071(t *testing.T) {
	e, _ := modelEnv(t)
	writeObservations(t, "["+strings.Join([]string{
		observationJSON("kilo", "ahead-an-hour", "", 16000, -time.Hour),
		observationJSON("kilo", "ahead-a-minute", "", 12000, -time.Minute),
	}, ",")+"]")
	if got := e.providerObservations("kilo"); len(got) != 1 || got[0].Model != "ahead-a-minute" {
		t.Errorf("handed %+v, want the one a minute ahead alone", got)
	}
}

// Two runs writing limit-observations.json each keep what the other wrote:
// a write re-reads the file and applies its one change, a save by its key
// (the newest Seen wins), a drop from what the file holds (0066-MADR D7;
// 0071-MADR P25; 0071-PLAN's review deviation of 2026-11-25, R5).
func TestSavedCapsTwoWriters_0071(t *testing.T) {
	both := func(t *testing.T) (*runEnv, *runEnv) {
		t.Helper()
		a, _ := modelEnv(t)
		b := envAt(t)
		a.providerObservations("kilo") // each run reads the file before the other writes
		b.providerObservations("kilo")
		return a, b
	}
	now := time.Now()
	t.Run("two_saves", func(t *testing.T) {
		a, b := both(t)
		a.rememberObservation("kilo", llm.LimitObservation{Model: "m1", Tokens: 16000, Seen: now})
		b.rememberObservation("kilo", llm.LimitObservation{Model: "m2", Tokens: 12000, Seen: now})
		if got := savedFile(t); !slices.Equal(got, []string{"kilo/m2  12000", "kilo/m1  16000"}) {
			t.Errorf("the file holds %q, want both runs' caps", got)
		}
	})
	t.Run("drop_kept", func(t *testing.T) {
		a, _ := modelEnv(t)
		writeObservations(t, "["+observationJSON("kilo", "m1", "", 16000, time.Hour)+"]")
		b := envAt(t)
		a.providerObservations("kilo")
		b.providerObservations("kilo")
		a.dropExceeded("kilo", "m1", 20000)
		b.rememberObservation("kilo", llm.LimitObservation{Model: "m2", Tokens: 12000, Seen: now})
		if got := savedFile(t); !slices.Equal(got, []string{"kilo/m2  12000"}) {
			t.Errorf("the file holds %q, want m1's drop kept and m2 saved", got)
		}
	})
	t.Run("newest_wins", func(t *testing.T) {
		a, b := both(t)
		a.rememberObservation("kilo", llm.LimitObservation{Model: "m1", Tokens: 16000, Seen: now})
		b.rememberObservation("kilo", llm.LimitObservation{Model: "m1", Tokens: 12000, Seen: now.Add(-time.Minute)})
		if got := savedFile(t); !slices.Equal(got, []string{"kilo/m1  16000"}) {
			t.Errorf("the file holds %q, want the newer 16,000", got)
		}
	})
}

// savedCapDroppedRun is TestSavedCapDroppedWhenExceeded_0071's run: a
// completed call whose output exceeds a saved cap drops it from the file,
// through the agent's CallOutput hook (0071-MADR P25, P43).
func savedCapDroppedRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	t.Setenv("OPENAI_API_KEY", "sk-test-only")
	writeObservations(t, "["+strings.Join([]string{
		observationJSON("openai", "gpt-6-luna", "", 16000, time.Hour),
		observationJSON("openai", "gpt-6-terra", "", 16000, time.Hour),
	}, ",")+"]")
	withModel(t, llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "long"},
		llm.Usage{InputTokens: 10, OutputTokens: 20000}, llm.Done{StopReason: llm.StopEndTurn}}}))
	mustRun(t, home, "-p", "hello")
	if got := savedFile(t); !slices.Equal(got, []string{"openai/gpt-6-terra  16000"}) {
		t.Errorf("the file holds %q, want gpt-6-luna's cap dropped", got)
	}
}
