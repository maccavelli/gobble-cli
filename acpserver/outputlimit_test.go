package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// The tests of 0071-MADR's interim, on go-llmprovider-sdk v1.32.0: each
// turn's output limit, its fit to the window over the one estimate the
// automatic trigger shares, the refusal and credit lines, the switch, the
// concurrent sessions, /context and the prompt response's _meta.

// lateScript is a provider that plays whichever script is set when a call
// is made, so a test can script turns from what it measured of a session.
type lateScript struct {
	mu sync.Mutex
	s  *llmtest.Script
}

func (l *lateScript) set(s *llmtest.Script) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.s = s
}

func (l *lateScript) script() *llmtest.Script {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s
}

func (*lateScript) ID() string                       { return "llmtest" }
func (l *lateScript) Capabilities() llm.Capabilities { return l.script().Capabilities() }
func (l *lateScript) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	return l.script().Stream(ctx, req)
}

// limitHarness is a connected agent whose providers fake (m1, m2) and other
// (x9) play scripts, whose catalog is info, read at each call, and whose
// sessions take settings.
type limitHarness struct {
	t      *testing.T
	c      *acptest.Conn
	client *acptest.Client
	ag     *Agent
	fake   *lateScript
	other  *llmtest.Script

	mu   sync.Mutex
	info map[string]llm.Model // by provider/model
}

func newLimitHarness(t *testing.T, settings SessionSettings, info map[string]llm.Model, turns ...llmtest.Turn) *limitHarness {
	t.Helper()
	return newLimitHarnessWith(t, settings, info, nil, turns...)
}

// newLimitHarnessWith is newLimitHarness with mod, when not nil, changing
// the agent's Options before it is built.
func newLimitHarnessWith(t *testing.T, settings SessionSettings, info map[string]llm.Model, mod func(*Options),
	turns ...llmtest.Turn) *limitHarness {
	t.Helper()
	h := &limitHarness{t: t, client: &acptest.Client{Allow: true}, fake: &lateScript{s: llmtest.NewScript(turns...)},
		other: llmtest.NewScript(llmtest.Text("from other")), info: info}
	if settings.Compaction.KeepRecentTokens == 0 {
		settings.Compaction.KeepRecentTokens = 50
	}
	choice := ModelChoice{Provider: "fake", Models: []string{"m1", "m2"}, Default: "m1", Providers: []ProviderChoice{
		{ID: "fake", Label: "Fake", State: StateLive, Models: []string{"m1", "m2"}},
		{ID: "other", Label: "Other", State: StateLive, Models: []string{"x9"}}}}
	opts := Options{Version: "1.2.3", Store: jsonl.New(t.TempDir(), jsonl.Options{}),
		Provider: func(*provider.ModelData, string, llm.Retry) (llm.Provider, error) { return h.fake, nil },
		ProviderFor: func(_ *provider.ModelData, id, _ string, _ llm.Retry) (llm.Provider, error) {
			if id == "other" {
				return h.other, nil
			}
			return nil, errors.New("no provider " + id)
		},
		Models:    func() ModelChoice { return choice },
		ModelInfo: h.modelInfo,
		Settings:  func(string) SessionSettings { return settings }}
	if mod != nil {
		mod(&opts)
	}
	h.ag = New(opts)
	h.c = acptest.Connect(t, h.ag, h.client)
	h.ag.SetConnection(h.c.Agent)
	t.Cleanup(func() {
		if err := h.ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, h.c.Client)
	return h
}

func (h *limitHarness) modelInfo(_ *provider.ModelData, provider, model string) llm.Model {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.info[provider+"/"+model]
	m.Provider, m.ID = provider, model
	return m
}

// setInfo changes the catalog's model ref.
func (h *limitHarness) setInfo(ref string, m llm.Model) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.info[ref] = m
}

func (h *limitHarness) session() acp.SessionId {
	return newSession(h.t, h.c, h.t.TempDir(), nil).SessionId
}

func (h *limitHarness) mustPrompt(id acp.SessionId, text string) acp.PromptResponse {
	h.t.Helper()
	resp, err := prompt(h.t, h.c, id, text)
	if err != nil {
		h.t.Fatalf("prompt %.20q: %v", text, err)
	}
	return resp
}

// context is the session's /context text.
func (h *limitHarness) context(id acp.SessionId) string {
	h.t.Helper()
	text, _ := slash(h.t, h.c, h.client, id, "/context")
	return text
}

// live is the session's state.
func (h *limitHarness) live(id acp.SessionId) *liveSession {
	h.ag.mu.Lock()
	defer h.ag.mu.Unlock()
	return h.ag.sessions[id]
}

// calls are fake's requests of the session's turns, without the summaries.
func (h *limitHarness) calls() []*llm.Request {
	var out []*llm.Request
	for _, r := range h.fake.script().Requests() {
		if !strings.HasPrefix(systemOf(r), "You are a context summarization assistant.") {
			out = append(out, r)
		}
	}
	return out
}

// summaries are fake's summary requests.
func (h *limitHarness) summaries() []*llm.Request {
	var out []*llm.Request
	for _, r := range h.fake.script().Requests() {
		if strings.HasPrefix(systemOf(r), "You are a context summarization assistant.") {
			out = append(out, r)
		}
	}
	return out
}

// lastCap is the last turn call's MaxTokens.
func (h *limitHarness) lastCap() int {
	h.t.Helper()
	calls := h.calls()
	if len(calls) == 0 {
		h.t.Fatal("no call made")
	}
	return calls[len(calls)-1].MaxTokens
}

// metaLimit is a prompt response's _meta.gobble.outputLimit, as decoded,
// or nil.
func metaLimit(resp acp.PromptResponse) map[string]any {
	g, ok := jsonx.Member[map[string]any](resp.Meta, gobble)
	if !ok {
		return nil
	}
	l, _ := g["outputLimit"].(map[string]any)
	return l
}

// wantMeta checks resp's _meta.gobble.outputLimit.
func wantMeta(t *testing.T, resp acp.PromptResponse, tokens int, basis string) {
	t.Helper()
	if l := metaLimit(resp); l == nil || l["tokens"] != float64(tokens) || l["basis"] != basis {
		t.Errorf("_meta.gobble.outputLimit %v, want {tokens %d, basis %s}", l, tokens, basis)
	}
}

// wide is a catalog model with a window of a million tokens and cap c.
func wide(c int) llm.Model { return llm.Model{ContextWindow: 1000000, MaxTokens: c} }

// Each turn asks for the catalog's cap of its model: under 64,000 as
// stated; above it, the six GLM-5.3 models of 0071-MADR's table among them,
// 64,000; a cap no source states, the assumed 16,384; a wrong low listed cap
// as listed (0071-MADR D1, D2, L1, L5, L33, L36).
func TestTurnSendsCap_0071(t *testing.T) {
	for _, c := range []struct {
		name  string
		model llm.Model
		cap   int
		basis string
	}{
		{"catalog_32000", llm.Model{ContextWindow: 200000, MaxTokens: 32000}, 32000, "stated"},
		{"kilo_glm-5.3", llm.Model{ContextWindow: 1048576, MaxTokens: 943718}, 64000, "ceiling"},
		{"kilo_glm-5.3-flash", llm.Model{ContextWindow: 1048576, MaxTokens: 943718}, 64000, "ceiling"},
		{"opencode-go_glm-5.3", llm.Model{ContextWindow: 1000000, MaxTokens: 131072}, 64000, "ceiling"},
		{"opencode-go_glm-5.3-flash", llm.Model{ContextWindow: 1000000, MaxTokens: 131072}, 64000, "ceiling"},
		{"together_GLM-5.3", llm.Model{ContextWindow: 1048576, MaxTokens: 262144}, 64000, "ceiling"},
		{"together_GLM-5.3-Flash", llm.Model{ContextWindow: 1048575, MaxTokens: 400000}, 64000, "ceiling"},
		{"assumed", llm.Model{ContextWindow: 200000}, 16384, "assumed"},
		{"no_window_assumed", llm.Model{}, 16384, "assumed"},
		{"wrong_low_cap", wide(4096), 4096, "stated"},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": c.model}, llmtest.Text("ok"))
			id := h.session()
			resp := h.mustPrompt(id, "hello")
			if got := h.lastCap(); got != c.cap {
				t.Errorf("the turn asked for %d, want %d", got, c.cap)
			}
			wantMeta(t, resp, c.cap, c.basis)
		})
	}
	// 0071 new names: begin
	t.Run("user_above_ceiling", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": {ContextWindow: 1000000, MaxTokens: 100000,
			MaxTokensFrom: llm.CapUser}}, llmtest.Text("ok"))
		resp := h.mustPrompt(h.session(), "hello")
		if got := h.lastCap(); got != 100000 {
			t.Errorf("the turn asked for %d, want models.json's 100,000 above the ceiling", got)
		}
		wantMeta(t, resp, 100000, "user")
	})
	// 0071 new names: end
}

// A long anchored session is held to the room the window leaves: the last
// usage plus what follows it, less 4,096 (0071-MADR D2, D3, L2).
func TestTurnCapHeldToWindow_0071(t *testing.T) {
	const window, used = 1048576, 1000000
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": {ContextWindow: window, MaxTokens: 943718}},
		full("r1", used), llmtest.Text("ok"))
	id := h.session()
	h.mustPrompt(id, p1)
	resp := h.mustPrompt(id, p2)
	want := window - used - compaction.EstimateText(p2) - 4096
	if got := h.lastCap(); got != want || len(h.summaries()) != 0 {
		t.Fatalf("the turn asked for %d with %d summaries, want the room %d and no compaction", got, len(h.summaries()), want)
	}
	wantMeta(t, resp, want, "window")
	// 0071 P2b new names: begin
	// V3 over P17's facts: the cap is fitted to the SDK's limits' window,
	// which the turn prefers to the merged ContextWindow (windowOf). The
	// merged window is set apart from it, above the session, so a cap
	// fitted to it asks for more than the room; the compaction trigger,
	// which reads the merged window, has no cause to compact, and would at
	// half of it (0071-MADR P17, V3; 0071-PLAN Phase 2b's delta check).
	t.Run("limits", func(t *testing.T) {
		const merged = 1500000
		m := llm.Model{ContextWindow: merged, Limits: llm.Limits{Output: fact(943718, "listing:kilo"), StreamedOutput: fact(943718, "listing:kilo"),
			Window: fact(window, "listing:kilo")}}
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": m}, full("r1", used), llmtest.Text("ok"))
		id := h.session()
		h.mustPrompt(id, p1)
		resp := h.mustPrompt(id, p2)
		if got := h.lastCap(); got != want || len(h.summaries()) != 0 {
			t.Fatalf("the turn asked for %d with %d summaries, want the room %d and no compaction", got, len(h.summaries()), want)
		}
		wantMeta(t, resp, want, "window")
	})
	// 0071 P2b new names: end
}

// Thinking does not change the limit: the same cap at every level, an
// effort only (0071-MADR D5, L8).
func TestTurnCapSameAtEveryLevel_0071(t *testing.T) {
	m := llm.Model{ContextWindow: 200000, MaxTokens: 32000, Reasoning: true, ReasoningKnown: true,
		Efforts: []string{"none", "low", "medium", "high", "xhigh", "max"}}
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": m})
	levels := h.ag.levelsOf(nil, "fake", "m1")
	if !slices.Contains(levels, "off") || !slices.Contains(levels, "max") {
		t.Fatalf("levels %v, want off to max", levels)
	}
	turns := make([]llmtest.Turn, len(levels))
	for i := range turns {
		turns[i] = llmtest.Text("ok")
	}
	h.fake.set(llmtest.NewScript(turns...))
	id := h.session()
	for _, level := range levels {
		if _, err := setOption(t, h.c, id, configThinking, level); err != nil {
			t.Fatal(err)
		}
		h.mustPrompt(id, "at "+level)
		if r := h.calls()[len(h.calls())-1]; r.MaxTokens != 32000 || (level != "off" && r.Thinking != level) {
			t.Errorf("%s: asked for %d at %q, want 32,000 at the level", level, r.MaxTokens, r.Thinking)
		}
	}
}

// The trigger and the limit read one estimate (0071-MADR D3, L9, L41): an
// anchored count does not add the system prompt the usage already measured,
// so a session the old count compacted goes ahead, its limit the room over
// that count; an unanchored count adds the request's tools, so a session the
// old count let through compacts.
func TestOneEstimate_0071(t *testing.T) {
	const window = 200000
	threshold := window - compaction.DefaultSettings.ReserveTokens
	t.Run("anchored_no_system_twice", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": {ContextWindow: window, MaxTokens: 100000}})
		id := h.session()
		sys := compaction.EstimateText(h.ag.systemText(h.live(id)))
		if sys < 200 {
			t.Fatalf("the system prompt is %d tokens, too few to tell the counts apart", sys)
		}
		used := threshold - compaction.EstimateText(p2) - sys/2 // at or under the threshold alone, over it with the system prompt
		h.fake.set(llmtest.NewScript(full("r1", int64(used)), llmtest.Text("ok"), llmtest.Text("spare"), llmtest.Text("spare")))
		h.mustPrompt(id, p1)
		h.mustPrompt(id, p2)
		want := window - (used + compaction.EstimateText(p2)) - 4096
		if n := len(h.summaries()); n != 0 {
			t.Fatalf("%d summaries: the trigger counted the system prompt twice", n)
		}
		if got := h.lastCap(); got != want {
			t.Errorf("the turn asked for %d, want the room %d over the trigger's count", got, want)
		}
	})
	t.Run("unanchored_counts_tools", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(100000)})
		bare := llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: strings.Repeat("r", 4000)}, llm.Done{StopReason: llm.StopEndTurn}}}
		h.fake.set(llmtest.NewScript(bare, llmtest.Text("SUMMARY"), llmtest.Text("ok"), llmtest.Text("spare")))
		id := h.session()
		h.mustPrompt(id, p1)
		s := h.live(id)
		msgs := compaction.EstimateEntries(s.tree.Path()) // no usage: every message's estimate
		tools := compaction.EstimateText(string(h.calls()[0].Tools))
		sys := compaction.EstimateText(h.ag.systemText(s))
		if tools < 200 {
			t.Fatalf("the request's tools are %d tokens, too few to tell the counts apart", tools)
		}
		// The window puts the threshold halfway through the tools' count.
		h.setInfo("fake/m1", llm.Model{ContextWindow: msgs + compaction.EstimateText(p2) + sys + tools/2 + compaction.DefaultSettings.ReserveTokens,
			MaxTokens: 100000})
		h.mustPrompt(id, p2)
		if n := len(h.summaries()); n != 1 {
			t.Fatalf("%d summaries, want 1: the trigger did not count the tools", n)
		}
	})
}

// apiError is a scripted failure as gobble's adapter maps one.
func apiError(status int, provider, msg string, wrap error) llmtest.Turn {
	err := fmt.Errorf("invalid request: %s", msg)
	if wrap != nil {
		err = fmt.Errorf("%w: %s", wrap, msg)
	}
	return llmtest.Turn{Err: &llm.APIError{Status: status, Provider: provider, Message: msg, Err: err}}
}

// A refusal of the limit fails the turn with the line that names it: an
// invalid request naming an output-limit field, the SDK's "fills the
// context window" reason, and a refusal still refused after its one resend;
// an invalid request naming no such field gains nothing (0071-MADR D8 (c),
// L12, L13, L14, L39).
func TestLimitRefusalLine_0071(t *testing.T) {
	line := func(n, basis string) string {
		return "gobble asked for at most " + n + " output tokens (" + basis + "); to send less, set maxTokens for m1 under modelOverrides in models.json"
	}
	for _, c := range []struct {
		name  string
		turns []llmtest.Turn
		want  string // "" for no line
	}{
		{"names_field", []llmtest.Turn{apiError(400, "fake", "max_tokens: must be at most 8192 for this model", nil)}, line("32,000", "stated")},
		{"other_invalid", []llmtest.Turn{apiError(400, "fake", "temperature: must be at most 2", nil)}, ""},
		{"fills_context", []llmtest.Turn{{Err: &llm.APIError{Status: 400, Provider: "fake", Message: "the completion alone is over the window",
			Err: errors.New("max_tokens alone fills the context window: lower it")}}}, line("32,000", "stated")},
		{"refused_after_resend", []llmtest.Turn{
			apiError(400, "fake", "max_tokens: 32000 > 16000, which is the maximum allowed number of output tokens for m1", nil),
			apiError(400, "fake", "max_tokens: 16000 > 8000, which is the maximum allowed number of output tokens for m1", nil)},
			line("16,000", "refusal")},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": {ContextWindow: 200000, MaxTokens: 32000}}, c.turns...)
			_, err := prompt(t, h.c, h.session(), "hello")
			if err == nil {
				t.Fatal("the turn did not fail")
			}
			if has := strings.Contains(err.Error(), "gobble asked for at most"); c.want == "" && has || c.want != "" && !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q, want the line %q", err, c.want)
			}
		})
	}
	// 0071-PLAN Phase 2c: a refusal whose cap only Observed states (P34).
	t.Run("refused_again_observed", refusedAgainObserved)
}

// A credit or rate failure of a capped call fails the turn with its error
// text byte-identical, and D8 (d)'s line in the error data's outputLimitNote:
// a 402, a 429 after the SDK's retries and Kilo's 503 for an upstream's 402
// carry it; another provider's 503 and a turn whose calls carried no limit do
// not (0071-MADR D8 (d) as amended 2026-11-24, "Move the line into a new
// field"; L15, L43).
func TestCreditRefusalLine_0071(t *testing.T) {
	note := "gobble asked for at most 32,000 output tokens (stated); some services hold credit or rate against that number; " +
		"set maxTokens for m1 under modelOverrides in models.json to ask for less"
	for _, c := range []struct {
		name string
		turn llmtest.Turn
		note string // "" for no member
	}{
		{"quota_402", apiError(402, "fake", "insufficient credits", llm.ErrQuota), note},
		{"rate_429", llmtest.RateLimited(0), note},
		{"kilo_503", apiError(503, "kilo", "upstream unavailable", llm.ErrUnavailable), note},
		{"other_503", apiError(503, "fake", "unavailable", llm.ErrUnavailable), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": {ContextWindow: 200000, MaxTokens: 32000}}, c.turn)
			_, err := prompt(t, h.c, h.session(), "hello")
			if err == nil {
				t.Fatal("the turn did not fail")
			}
			_, data := wireData(t, err)
			var d struct {
				Error           string `json:"error"`
				OutputLimitNote string `json:"outputLimitNote"`
			}
			if jerr := jsonx.Unmarshal([]byte(data), &d); jerr != nil || d.Error != c.turn.Err.Error() {
				t.Errorf("error data's text %q (%v), want the failure's own %q", d.Error, jerr, c.turn.Err.Error())
			}
			if d.OutputLimitNote != c.note {
				t.Errorf("outputLimitNote %q, want %q", d.OutputLimitNote, c.note)
			}
			if has := strings.Contains(data, "outputLimitNote"); has != (c.note != "") {
				t.Errorf("the member is present: %v, want %v, in %s", has, c.note != "", data)
			}
		})
	}
	// 0071 new names: begin
	t.Run("uncapped_call", func(t *testing.T) {
		quota := &llm.APIError{Status: 402, Err: llm.ErrQuota}
		if got := limitLine(quota, llm.OutputLimit{}, "m1"); !errors.Is(got, quota) || got.Error() != quota.Error() {
			t.Errorf("a turn whose calls carried no limit: %v, want the error as it was", got)
		}
		if got := creditNote(quota, llm.OutputLimit{}, "m1"); got != "" {
			t.Errorf("a turn whose calls carried no limit: note %q, want none", got)
		}
	})
	// 0071 new names: end
}

// A model switch, of model or of provider, takes effect at the next
// prompt: its calls ask for the new model's cap (0071-MADR D1, L21, L22).
func TestCapAfterModelSwitch_0071(t *testing.T) {
	info := map[string]llm.Model{"fake/m1": wide(32000), "fake/m2": wide(8000), "other/x9": wide(4000)}
	for _, c := range []struct {
		name, to string
		cap      int
	}{
		{"same_provider", "m2", 8000},
		{"other_provider", "other/x9", 4000},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newLimitHarness(t, SessionSettings{}, info, llmtest.Text("one"), llmtest.Text("two"))
			id := h.session()
			h.mustPrompt(id, "first")
			if _, err := setOption(t, h.c, id, configModel, c.to); err != nil {
				t.Fatal(err)
			}
			resp := h.mustPrompt(id, "second")
			reqs := append(h.fake.script().Requests(), h.other.Requests()...)
			if len(reqs) != 2 || reqs[0].MaxTokens != 32000 || reqs[1].MaxTokens != c.cap {
				t.Errorf("caps %v, want 32,000 then %d", caps(reqs), c.cap)
			}
			wantMeta(t, resp, c.cap, "stated")
		})
	}
}

// Two sessions at once, on models capped at 64,000 and 4,096: each request
// and each /context shows its own (0071-MADR D4, L23).
func TestCapConcurrentSessions_0071(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(100000), "fake/m2": wide(4096)},
		llmtest.Text("a"), llmtest.Text("b"))
	one, two := h.session(), h.session()
	if _, err := setOption(t, h.c, two, configModel, "m2"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, id := range []acp.SessionId{one, two} {
		wg.Go(func() { _, errs[i] = prompt(t, h.c, id, "at once") })
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		t.Fatal(err)
	}
	for _, r := range h.calls() {
		if want := map[string]int{"m1": 64000, "m2": 4096}[r.Model]; r.MaxTokens != want {
			t.Errorf("%s asked for %d, want %d", r.Model, r.MaxTokens, want)
		}
	}
	for id, want := range map[acp.SessionId]string{
		one: "Output limit: 64,000 tokens on the last call (gobble's 64,000 ceiling; set maxTokens in models.json to raise it)\n",
		two: "Output limit: 4,096 tokens on the last call (the model's stated limit)\n",
	} {
		if got := h.context(id); !strings.Contains(got, want) {
			t.Errorf("/context %q lacks %q", got, want)
		}
	}
}

// /context and /session name the last call's limit and its basis, in words,
// after the context's line; before the session's first call, the model's
// limit and its source, or that it is unknown (0071-MADR D10, L25, L26).
func TestContextShowsOutputLimit_0071(t *testing.T) {
	t.Run("before_call", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": {ContextWindow: 200000, MaxTokens: 32000}, "fake/m2": {ContextWindow: 200000}})
		id := h.session()
		got := h.context(id)
		if want := "\nOutput limit: none sent yet; the model's limit is 32,000 (the catalog)\n"; !strings.Contains(got, want) ||
			strings.Index(got, "Context: ") > strings.Index(got, "Output limit: ") {
			t.Errorf("/context %q, want %q after the context's line", got, want)
		}
		if _, err := setOption(t, h.c, id, configModel, "m2"); err != nil {
			t.Fatal(err)
		}
		if got, want := h.context(id), "Output limit: none sent yet; the model's limit is unknown\n"; !strings.Contains(got, want) {
			t.Errorf("/context %q, want %q", got, want)
		}
		if got, _ := slash(t, h.c, h.client, id, "/session"); !strings.Contains(got, "Output limit: none sent yet") {
			t.Errorf("/session %q lacks the line", got)
		}
	})
	words := func(n, why string) string { return "Output limit: " + n + " tokens on the last call (" + why + ")\n" }
	for _, c := range []struct {
		name     string
		model    llm.Model
		settings SessionSettings
		turns    []llmtest.Turn
		want     string
	}{
		{"stated", wide(32000), SessionSettings{}, []llmtest.Turn{llmtest.Text("a"), llmtest.Text("b")}, words("32,000", "the model's stated limit")},
		{"ceiling", wide(943718), SessionSettings{}, []llmtest.Turn{llmtest.Text("a"), llmtest.Text("b")},
			words("64,000", "gobble's 64,000 ceiling; set maxTokens in models.json to raise it")},
		{"assumed", llm.Model{ContextWindow: 1000000}, SessionSettings{}, []llmtest.Turn{llmtest.Text("a"), llmtest.Text("b")},
			words("16,384", "assumed: no source states this model's limit")},
		{"window", llm.Model{ContextWindow: 1048576, MaxTokens: 943718}, SessionSettings{}, []llmtest.Turn{full("r1", 1000000), llmtest.Text("b")},
			words(thousands(int64(1048576-1000000-compaction.EstimateText(p2)-4096)), "held to the room the window leaves")},
		{"full", llm.Model{ContextWindow: 200000, MaxTokens: 32000}, SessionSettings{Compaction: compaction.Settings{ReserveTokens: 16384, KeepRecentTokens: 10000000}},
			[]llmtest.Turn{full("r1", 199000), llmtest.Text("b")}, words("32,000", "the window is full by gobble's estimate")},
		{"refusal", wide(32000), SessionSettings{}, []llmtest.Turn{llmtest.Text("a"),
			apiError(400, "fake", "max_tokens: 32000 > 16000, which is the maximum allowed number of output tokens for m1", nil), llmtest.Text("b")},
			words("16,000", "the cap the service stated when it refused the first")},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newLimitHarness(t, c.settings, map[string]llm.Model{"fake/m1": c.model}, c.turns...)
			id := h.session()
			h.mustPrompt(id, p1)
			h.mustPrompt(id, p2)
			if got := h.context(id); !strings.Contains(got, c.want) {
				t.Errorf("/context %q lacks %q", got, c.want)
			}
		})
	}
	// 0071 new names: begin
	t.Run("user", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": {ContextWindow: 1000000, MaxTokens: 100000, MaxTokensFrom: llm.CapUser}},
			llmtest.Text("a"))
		id := h.session()
		if got, want := h.context(id), "Output limit: none sent yet; the model's limit is 100,000 (your models.json)\n"; !strings.Contains(got, want) {
			t.Errorf("/context %q lacks %q", got, want)
		}
		h.mustPrompt(id, p1)
		if got, want := h.context(id), words("100,000", "your models.json maxTokens"); !strings.Contains(got, want) {
			t.Errorf("/context %q lacks %q", got, want)
		}
	})
	// 0071 new names: end
}

// The prompt response carries the turn's last call's limit as
// _meta.gobble.outputLimit {tokens, basis}; a tool round's last call is the
// one carried (0071-MADR D10, L27).
func TestPromptResponseOutputLimit_0071(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(32000)},
		llmtest.ToolCall("c1", "read", `{"path":"nothing.txt"}`), llmtest.Text("done"))
	resp := h.mustPrompt(h.session(), "read it")
	if n := len(h.calls()); n != 2 {
		t.Fatalf("%d calls, want a tool round's 2", n)
	}
	wantMeta(t, resp, 32000, "stated")
	// 0071 new names: begin
	s := &liveSession{queue: newQueue()}
	s.queue.steer("s")
	meta, err := turnMeta(s, true, llm.OutputLimit{Tokens: 32000, Basis: llm.BasisStated})
	if err != nil {
		t.Fatal(err)
	}
	b, err := jsonx.Marshal(meta)
	if got, want := string(b), `{"gobble":{"cleared":{"followUp":[],"steering":["s"]},"outputLimit":{"tokens":32000,"basis":"stated"}}}`; err != nil || got != want {
		t.Errorf("a cancelled turn's _meta %s (%v), want %s", got, err, want)
	}
	// 0071 new names: end
	// A call that carried no limit is {0, "none"}, not dropped (0071-MADR
	// P35); "none" is llm.BasisNone.
	t.Run("none", func(t *testing.T) {
		meta, err := turnMeta(&liveSession{queue: newQueue()}, false, llm.OutputLimit{Tokens: 0, Basis: "none"})
		if err != nil {
			t.Fatal(err)
		}
		b, err := jsonx.Marshal(meta)
		if got, want := string(b), `{"gobble":{"outputLimit":{"tokens":0,"basis":"none"}}}`; err != nil || got != want {
			t.Errorf("_meta %s (%v), want %s", got, err, want)
		}
		// 0071 P2a new names: begin
		named, err := turnMeta(&liveSession{queue: newQueue()}, false, llm.OutputLimit{Basis: llm.BasisNone})
		b, err2 := jsonx.Marshal(named)
		if want := `{"gobble":{"outputLimit":{"tokens":0,"basis":"none"}}}`; err != nil || err2 != nil || string(b) != want {
			t.Errorf("llm.BasisNone's _meta %s (%v, %v), want %s", b, err, err2, want)
		}
		// 0071 P2a new names: end
	})
	// 0071-PLAN Phase 2c: what the last call carried (P22, P32).
	t.Run("sent", promptResponseSentLimit)
}

// A turn with no model call carries no output limit, and /context keeps
// the last call's (0071-MADR D10, L30).
func TestNoCallNoOutputLimit_0071(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(32000)}, llmtest.Text("a"))
	id := h.session()
	h.mustPrompt(id, "hello")
	resp := h.mustPrompt(id, "/todos")
	if l := metaLimit(resp); l != nil {
		t.Errorf("a turn with no call carries %v", l)
	}
	if got, want := h.context(id), "Output limit: 32,000 tokens on the last call (the model's stated limit)\n"; !strings.Contains(got, want) {
		t.Errorf("/context %q lacks %q", got, want)
	}
	// 0071 new names: begin
	if meta, err := turnMeta(&liveSession{queue: newQueue()}, false, llm.OutputLimit{}); meta != nil || err != nil {
		t.Errorf("no call, not cancelled: _meta %v, %v; want none", meta, err)
	}
	// 0071 new names: end
}

// Over the threshold, a session Prepare refuses goes ahead, and a call with
// no room by the estimate sends the cap, basis full, for the service's
// refusal to start the overflow path (0071-MADR D2, L35, L37).
func TestCapWhenPrepareRefuses_0071(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{Compaction: compaction.Settings{ReserveTokens: 16384, KeepRecentTokens: 10000000}},
		map[string]llm.Model{"fake/m1": {ContextWindow: 200000, MaxTokens: 32000}}, full("r1", 199000), llmtest.Text("ok"))
	id := h.session()
	h.mustPrompt(id, p1)
	resp := h.mustPrompt(id, p2)
	if got := h.lastCap(); got != 32000 || len(h.summaries()) != 0 {
		t.Errorf("asked for %d with %d summaries, want the cap 32,000 and no compaction", got, len(h.summaries()))
	}
	wantMeta(t, resp, 32000, "full")
}

// A reserve below 5,120 lets through a call with less than 1,024 of room:
// it asks for the floor, 1,024 (0071-MADR D2, L3, L38).
func TestCapSmallReserve_0071(t *testing.T) {
	const window = 200000
	used := window - 4096 - 500 - compaction.EstimateText(p2) // a room of 500 at the next call
	h := newLimitHarness(t, SessionSettings{Compaction: compaction.Settings{ReserveTokens: 2000}},
		map[string]llm.Model{"fake/m1": {ContextWindow: window, MaxTokens: 32000}}, full("r1", int64(used)), llmtest.Text("ok"))
	id := h.session()
	h.mustPrompt(id, p1)
	resp := h.mustPrompt(id, p2)
	if got := h.lastCap(); got != 1024 || len(h.summaries()) != 0 {
		t.Errorf("asked for %d with %d summaries, want the floor 1,024 and no compaction", got, len(h.summaries()))
	}
	wantMeta(t, resp, 1024, "window")
}

// Every model call gobble makes names a limit of at least 1, the turn's, a
// compaction summary's and the title's alike, the title's with its model's
// cap unknown included, so go-llmprovider-sdk v1.33.0 refuses none before
// sending on a Messages route where nothing states the model's limit
// (0071-MADR P7, F21).
func TestEveryCallNamesLimit_0071(t *testing.T) {
	long := func(s string) string { return s + " " + strings.Repeat("x", 400) }
	t.Run("turn", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{}, llmtest.Text("ok"))
		h.mustPrompt(h.session(), "hello")
		if got := h.lastCap(); got < 1 {
			t.Errorf("the turn asked for %d, want a limit", got)
		}
	})
	t.Run("summary", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{}, llmtest.Text("r1"), llmtest.Text("r2"), llmtest.Text("SUMMARY"))
		id := h.session()
		h.mustPrompt(id, long("one"))
		h.mustPrompt(id, long("two"))
		h.mustPrompt(id, "/compact")
		sums := h.summaries()
		if len(sums) == 0 {
			t.Fatal("no summary was asked")
		}
		for i, r := range sums {
			if r.MaxTokens < 1 {
				t.Errorf("summary %d asked for %d, want a limit", i+1, r.MaxTokens)
			}
		}
	})
	t.Run("title", func(t *testing.T) {
		small := llmtest.NewScript(llmtest.Text("A title"))
		h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), small, func(o *Options) {
			o.ModelInfo = func(_ *provider.ModelData, provider, model string) llm.Model {
				return llm.Model{Provider: provider, ID: model}
			}
		})
		id := newSession(t, h.c, t.TempDir(), nil).SessionId
		if _, err := prompt(t, h.c, id, "Name this session"); err != nil {
			t.Fatal(err)
		}
		waitFor(t, func() bool { return h.titled("A title") })
		if got := small.Requests()[0].MaxTokens; got < 1 {
			t.Errorf("the title asked for %d with its model's cap unknown, want a limit", got)
		}
	})
}

// agentChunks are the agent message chunks of c's updates, in order.
func agentChunks(c *acptest.Client) []string {
	var out []string
	for _, n := range c.Updates() {
		if u := n.Update.AgentMessageChunk; u != nil && u.Content.Text != nil {
			out = append(out, u.Content.Text.Text)
		}
	}
	return out
}

// toolCutLine is boring's turn.stop.max_tokens_tool for model m1.
const toolCutLine = "The reply hit the output limit while writing a tool call, so the call was dropped. Ask again, or set " +
	"maxTokens for m1 in models.json."

// Text, then a call the output limit cut, which the SDK dropped: the turn
// ends max_tokens with its text, and the agent says why after it; a reply
// cut while it wrote text, with no call dropped, says nothing more
// (0071-MADR P29, the owner's "End the turn, keep the text, notify
// (Recommended)"; 0065-MADR D7, D8).
func TestCutCallAfterText_0071(t *testing.T) {
	cut := func(warn bool) llmtest.Turn {
		evs := []llm.Event{llm.TextDelta{Text: "Let me read it."}}
		if warn {
			evs = append(evs, llm.Warning{Kind: "tool_call_cut", Model: "m1", Detail: "the output limit cut call 1 (read)"}) // llm.WarningToolCallCut
		}
		return llmtest.Turn{Events: append(evs, llm.Usage{InputTokens: 3, OutputTokens: 2}, llm.Done{StopReason: llm.StopMaxTokens})}
	}
	for _, c := range []struct {
		name string
		warn bool
		want []string
	}{
		{"cut_call", true, []string{"Let me read it.", "\n\n" + toolCutLine}},
		{"cut_text", false, []string{"Let me read it."}},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(32000)}, cut(c.warn))
			resp := h.mustPrompt(h.session(), "read it")
			if resp.StopReason != acp.StopReasonMaxTokens {
				t.Errorf("stop %q, want max_tokens", resp.StopReason)
			}
			if got := agentChunks(h.client); !slices.Equal(got, c.want) {
				t.Errorf("agent messages %q, want %q", got, c.want)
			}
		})
	}
}

// messagesWire answers each streamed Messages request with "hello" and
// records each request's max_tokens; it never dials.
type messagesWire struct {
	mu   sync.Mutex
	caps []float64
}

func (w *messagesWire) RoundTrip(r *http.Request) (*http.Response, error) {
	var body map[string]any
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal(b, &body) //nolint:errcheck // a body that is not JSON records no limit
	}
	w.mu.Lock()
	limit, _ := body["max_tokens"].(float64)
	w.caps = append(w.caps, limit)
	w.mu.Unlock()
	ev := func(name, data string) string { return "event: " + name + "\ndata: " + data + "\n\n" }
	stream := ev("message_start", `{"type":"message_start","message":{"id":"m","type":"message","role":"assistant","content":[],"usage":{"input_tokens":12,"output_tokens":0}}}`) +
		ev("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`) +
		ev("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`) +
		ev("content_block_stop", `{"type":"content_block_stop","index":0}`) +
		ev("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`) +
		ev("message_stop", `{"type":"message_stop"}`)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(stream)), Request: r}, nil
}

// A turn on Claude through provider.New, on a canned wire, for a model no
// source states a limit for: gobble names the limit, so go-llmprovider-sdk
// v1.33.0 sends the call where it would refuse one that named none (0071-MADR
// P7, F21; the review's case).
func TestEveryCallNamesLimitColdClaude_0071(t *testing.T) {
	w := &messagesWire{}
	p, err := provider.New("claude", provider.Options{APIKey: "k", Model: "m1", HTTPClient: &http.Client{Transport: w},
		Retry: &llm.Retry{Off: true}})
	if err != nil {
		t.Fatal(err)
	}
	client := &acptest.Client{Allow: true}
	choice := ModelChoice{Provider: "fake", Models: []string{"m1"}, Default: "m1", Providers: []ProviderChoice{
		{ID: "fake", Label: "Fake", State: StateLive, Models: []string{"m1"}}}}
	ag := New(Options{Version: "1.2.3", Store: jsonl.New(t.TempDir(), jsonl.Options{}),
		Provider: func(*provider.ModelData, string, llm.Retry) (llm.Provider, error) { return p, nil },
		Models:   func() ModelChoice { return choice },
		ModelInfo: func(_ *provider.ModelData, provider, model string) llm.Model {
			return llm.Model{Provider: provider, ID: model}
		}})
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "hello"); err != nil {
		t.Fatalf("the turn failed: %v", err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.caps) != 1 || w.caps[0] < 1 {
		t.Fatalf("requests' max_tokens %v, want one request that names a limit", w.caps)
	}
}

// Text, then a call the output limit cut, while a follow-up is queued: the
// turn goes on with the follow-up, and the notice is sent once, after the
// text and before the follow-up's echo (0071-MADR P36, the owner's "Show the
// notice anyway (Recommended)").
func TestCutCallNoticeBeforeQueued_0071(t *testing.T) {
	cut := llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "Let me read it."},
		llm.Warning{Kind: "tool_call_cut", Model: "m1", Detail: "the output limit cut call 1 (read)"}, // llm.WarningToolCallCut
		llm.Usage{InputTokens: 3, OutputTokens: 2}, llm.Done{StopReason: llm.StopMaxTokens}}}
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(32000)}, cut, llmtest.Text("on it"))
	id := h.session()
	h.live(id).queue.followUpWith("and then this")
	resp := h.mustPrompt(id, "read it")
	if resp.StopReason != acp.StopReasonEndTurn || len(h.calls()) != 2 {
		t.Fatalf("stop %q after %d calls, want the follow-up's call, then end_turn", resp.StopReason, len(h.calls()))
	}
	var order []string
	for _, n := range h.client.Updates() {
		switch u := n.Update; {
		case u.AgentMessageChunk != nil && u.AgentMessageChunk.Content.Text != nil:
			order = append(order, "agent "+u.AgentMessageChunk.Content.Text.Text)
		case u.UserMessageChunk != nil && u.UserMessageChunk.Content.Text != nil:
			order = append(order, "user "+u.UserMessageChunk.Content.Text.Text)
		}
	}
	want := []string{"agent Let me read it.", "agent \n\n" + toolCutLine, "user and then this", "agent on it"}
	if !slices.Equal(order, want) {
		t.Errorf("updates %q, want %q", order, want)
	}
}

// fact is a known limit fact of tokens from source, the others beside it.
func fact(tokens int, source string, others ...llm.LimitFact) llm.LimitFact {
	return llm.LimitFact{Tokens: tokens, Known: true, Source: source, Others: others}
}

// userOverLine is boring's model.limit.user_over for m1 over models.json's
// 100,000 and a stated 16,000.
const userOverLine = "models.json asks for 100,000 output tokens for m1; the service stated 16,000, which gobble sends."

// Each call's cap is the SDK's streamed output, its source told apart, and
// its window the window else the input limit (0071-MADR P17, P28; 0066-MADR
// D4, D9): a kept stream cap below the output is asked; models.json's
// maxTokens over a listing is the user's; Pi's default is assumed; nothing
// known is the assumed 16,384; Claude's input limit holds the call to its
// room; a models.json maxTokens over a cap a service stated in this process
// (observed) or on an earlier run (the saved layer) sends the service's cap,
// basis service, said once per session and model.
func TestTurnCapFromLimits_0071(t *testing.T) {
	for _, c := range []struct {
		name  string
		model llm.Model
		cap   int
		basis string
	}{
		{"streamed", llm.Model{MaxTokens: 32000, MaxTokensFrom: llm.CapCatalog, ContextWindow: 1000000, Limits: llm.Limits{
			Output: fact(32000, "listing:claude"), StreamedOutput: fact(16000, "observed", fact(32000, "listing:claude")),
			Window: fact(1000000, "listing:claude")}}, 16000, "stated"},
		{"user", llm.Model{MaxTokens: 20000, MaxTokensFrom: llm.CapUser, ContextWindow: 1000000, Limits: llm.Limits{
			Output: fact(20000, "models.json", fact(128000, "listing:claude")), StreamedOutput: fact(20000, "models.json",
				fact(128000, "listing:claude")), Window: fact(1000000, "listing:claude")}}, 20000, "user"},
		{"assumed_default", llm.Model{MaxTokens: 16384, MaxTokensFrom: llm.CapAssumed, ContextWindow: 128000, Limits: llm.Limits{
			Output: fact(16384, "models.json"), StreamedOutput: fact(16384, "models.json"), Window: fact(128000, "models.json")}},
			16384, "assumed"},
		{"unknown", llm.Model{}, 16384, "assumed"},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": c.model}, llmtest.Text("ok"))
			resp := h.mustPrompt(h.session(), "hello")
			if got := h.lastCap(); got != c.cap {
				t.Errorf("the turn asked for %d, want %d", got, c.cap)
			}
			wantMeta(t, resp, c.cap, c.basis)
		})
	}
	t.Run("input_only", func(t *testing.T) {
		// Input 40,000 only, the merged window unknown: the room the input
		// limit leaves holds the 32,000 cap.
		m := llm.Model{MaxTokens: 32000, MaxTokensFrom: llm.CapCatalog, Limits: llm.Limits{Output: fact(32000, "listing:claude"),
			StreamedOutput: fact(32000, "listing:claude"), Input: fact(40000, "listing:claude")}}
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": m}, llmtest.Text("ok"))
		id := h.session()
		resp := h.mustPrompt(id, "hello")
		if got := h.lastCap(); got >= 32000 || got < llm.MinOutputTokens {
			t.Errorf("the turn asked for %d, want the room the 40,000 input limit leaves, below 32,000", got)
		}
		wantMeta(t, resp, h.lastCap(), "window")
	})
	userOver := func(source string) llm.Model {
		over := fact(16000, source, fact(100000, "models.json"), fact(128000, "listing:claude"))
		return llm.Model{MaxTokens: 16000, MaxTokensFrom: llm.CapUser, ContextWindow: 1000000, Limits: llm.Limits{Output: over,
			StreamedOutput: over, Window: fact(1000000, "listing:claude")}}
	}
	for _, c := range []struct{ name, source string }{{"user_over_observed", "observed"}, {"user_over_saved", "service-stated"}} {
		t.Run(c.name, func(t *testing.T) {
			h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": userOver(c.source)}, llmtest.Text("one"),
				llmtest.Text("two"))
			id := h.session()
			resp := h.mustPrompt(id, "first")
			if got := h.lastCap(); got != 16000 {
				t.Errorf("the turn asked for %d, want the service's 16,000 over models.json's 100,000", got)
			}
			wantMeta(t, resp, 16000, "service")
			h.mustPrompt(id, "second")
			if got := agentChunks(h.client); !slices.Equal(got, []string{userOverLine + "\n\n", "one", "two"}) {
				t.Errorf("agent messages %q, want the note once, before the first reply", got)
			}
			if ctx := h.context(id); !strings.Contains(ctx, "Output limit: 16,000 tokens on the last call (the cap the service stated, "+
				"below your models.json maxTokens)") {
				t.Errorf("/context %q, want the service's cap shown as such", ctx)
			}
		})
	}
}

// P28's note is marked as said when it is sent: a call that fails before
// any event leaves it unsent, and the session's next call keeps it to say
// (0071-PLAN Phase 2b's review). The agent yields a failed call's message
// before its error, so the case is the turn's limits dropped unsaid.
func TestUserOverNoteAfterFailure_0071(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{})
	s := h.live(h.session())
	first := &turnLimits{a: h.ag, s: s, provider: "fake", model: "m1"}
	first.userOver(100000, 16000) // kept; the call fails before its first event, so it is never said
	next := &turnLimits{a: h.ag, s: s, provider: "fake", model: "m1"}
	next.userOver(100000, 16000)
	next.mu.Lock()
	note := next.note
	next.mu.Unlock()
	if note != userOverLine {
		t.Errorf("the next call's note %q, want %q: the first was never said", note, userOverLine)
	}
}
