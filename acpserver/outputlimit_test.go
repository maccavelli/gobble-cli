package acpserver

import (
	"context"
	"errors"
	"fmt"
	"iter"
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
	h := &limitHarness{t: t, client: &acptest.Client{Allow: true}, fake: &lateScript{s: llmtest.NewScript(turns...)},
		other: llmtest.NewScript(llmtest.Text("from other")), info: info}
	if settings.Compaction.KeepRecentTokens == 0 {
		settings.Compaction.KeepRecentTokens = 50
	}
	choice := ModelChoice{Provider: "fake", Models: []string{"m1", "m2"}, Default: "m1", Providers: []ProviderChoice{
		{ID: "fake", Label: "Fake", State: StateLive, Models: []string{"m1", "m2"}},
		{ID: "other", Label: "Other", State: StateLive, Models: []string{"x9"}}}}
	h.ag = New(Options{Version: "1.2.3", Store: jsonl.New(t.TempDir(), jsonl.Options{}),
		Provider: func(string, llm.Retry) (llm.Provider, error) { return h.fake, nil },
		ProviderFor: func(id, _ string, _ llm.Retry) (llm.Provider, error) {
			if id == "other" {
				return h.other, nil
			}
			return nil, errors.New("no provider " + id)
		},
		Models:    func() ModelChoice { return choice },
		ModelInfo: h.modelInfo,
		Settings:  func(string) SessionSettings { return settings }})
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

func (h *limitHarness) modelInfo(provider, model string) llm.Model {
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
}

// Thinking does not change the limit: the same cap at every level, an
// effort only (0071-MADR D5, L8).
func TestTurnCapSameAtEveryLevel_0071(t *testing.T) {
	m := llm.Model{ContextWindow: 200000, MaxTokens: 32000, Reasoning: true, ReasoningKnown: true,
		Efforts: []string{"none", "low", "medium", "high", "xhigh", "max"}}
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": m})
	levels := h.ag.levelsOf("fake", "m1")
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
