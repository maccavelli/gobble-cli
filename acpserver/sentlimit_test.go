package acpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// The tests of 0071-PLAN's Phase 2c in acpserver: what the last call
// carried, as /context, /session and _meta show it (0071-MADR P22, P31,
// P32); the caps a service stated, learned and said (P19, P25, P34;
// 0066-MADR D10, S10, S12); the title's cap (V8).

// doneWith is a reply whose done event reports lim, as go-llmprovider-sdk
// v1.33.0's Response.OutputLimit reaches llm.Done (0071-MADR P14).
func doneWith(text string, lim llm.SentLimit) llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: text}, llm.Usage{InputTokens: 10, OutputTokens: int64(len(text))},
		llm.Done{StopReason: llm.StopEndTurn, OutputLimit: lim}}}
}

// observedTurn is a refusal of the limit sent in Kilo's range form, which
// only the SDK reads, with the cap it validated as the error's Observed.
func observedTurn(model string, stated int) llmtest.Turn {
	return observedTurnSaying(model, stated, fmt.Sprintf("The max_tokens parameter is illegal.：限制数值范围[1,%d]", stated))
}

// observedTurnSaying is observedTurn whose refusal's text is msg.
func observedTurnSaying(model string, stated int, msg string) llmtest.Turn {
	return llmtest.Turn{Err: &llm.APIError{Status: 400, Provider: "fake", Message: msg, Err: fmt.Errorf("invalid request: %s", msg),
		Observed: llm.LimitObservation{Model: model, Tokens: stated, Seen: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)}}}
}

// savedCaps records what the agent hands its store of saved caps.
type savedCaps struct {
	mu      sync.Mutex
	saved   []string
	outputs []string
}

func (c *savedCaps) save(provider string, o llm.LimitObservation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.saved = append(c.saved, fmt.Sprintf("%s/%s %d", provider, o.Model, o.Tokens))
}

func (c *savedCaps) output(provider, model string, tokens int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.outputs = append(c.outputs, fmt.Sprintf("%s/%s %d", provider, model, tokens))
}

func (c *savedCaps) got() ([]string, []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.saved), slices.Clone(c.outputs)
}

// 0071 2c new names: begin

// store gives h's agent c as its store of saved caps, as the run wires it.
func (c *savedCaps) store(h *limitHarness) *savedCaps {
	h.ag.saveObservation, h.ag.callOutput = c.save, c.output
	return c
}

// 0071 2c new names: end

// /context shows what the last call carried, from its done event's report:
// the wire's tokens with gobble's basis; the raise a thinking budget made;
// "none sent" on a route that sends none; another SDK basis raw, as
// sdk:<basis>, logged once per session and model; a failed call's asked
// value, marked; and, for a loaded or forked session, D10's "none sent yet"
// until its first call in this process (0071-MADR P22, P31, P32, the
// owner's "gobble's basis, wire value (Recommended)" and "Show what was
// asked, marked (Recommended)").
func TestContextShowsSentLimit_0071(t *testing.T) {
	stated := map[string]llm.Model{"fake/m1": wide(32000)}
	for _, c := range []struct {
		name string
		turn llmtest.Turn
		want string
	}{
		{"request", doneWith("a", llm.SentLimit{Tokens: 32000, Sent: true, Basis: "request"}),
			"Output limit: 32,000 tokens on the last call (the model's stated limit)\n"},
		{"raised", doneWith("a", llm.SentLimit{Tokens: 36096, Sent: true, Basis: "request"}),
			"Output limit: 36,096 tokens on the last call (the model's stated limit); raised from 32,000 for the thinking budget\n"},
		// A wire value below gobble's is shown as it is, not as a raise
		// (0071-PLAN's review deviation of 2026-11-25, R4).
		{"lowered", doneWith("a", llm.SentLimit{Tokens: 30000, Sent: true, Basis: "request"}),
			"Output limit: 30,000 tokens on the last call (the model's stated limit)\n"},
		{"chatgpt_none", doneWith("a", llm.SentLimit{Basis: "none"}),
			"Output limit: none sent on the last call; the service's default applies\n"},
		{"failed_asked", apiError(500, "fake", "upstream exploded", nil),
			"Output limit: 32,000 tokens asked on the last call, which failed (the model's stated limit)\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newLimitHarness(t, SessionSettings{}, stated, c.turn)
			id := h.session()
			if _, err := prompt(t, h.c, id, "hello"); (err != nil) != (c.name == "failed_asked") {
				t.Fatalf("prompt: %v", err)
			}
			if got := h.context(id); !strings.Contains(got, c.want) {
				t.Errorf("/context %q lacks %q", got, c.want)
			}
		})
	}
	t.Run("sdk_basis_raw", func(t *testing.T) {
		logs := &syncBuffer{}
		fact := llm.SentLimit{Tokens: 16000, Sent: true, Basis: "fact", Source: "listing:fake"}
		h := newLimitHarnessWith(t, SessionSettings{}, stated, func(o *Options) { o.Logger = slog.New(slog.NewTextHandler(logs, nil)) },
			doneWith("a", fact), doneWith("b", fact))
		id := h.session()
		resp := h.mustPrompt(id, "one")
		h.mustPrompt(id, "two")
		if got, want := h.context(id), "Output limit: 16,000 tokens on the last call (sdk:fact)\n"; !strings.Contains(got, want) {
			t.Errorf("/context %q lacks %q", got, want)
		}
		wantMeta(t, resp, 16000, "sdk:fact")
		if n := strings.Count(logs.String(), "the SDK reports a basis gobble did not ask for"); n != 1 {
			t.Errorf("%d warnings over two turns, want one per session and model:\n%s", n, logs)
		}
	})
	t.Run("loaded_session", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, stated, doneWith("a", llm.SentLimit{Tokens: 32000, Sent: true, Basis: "request"}))
		cwd := t.TempDir()
		id := newSession(t, h.c, cwd, nil).SessionId
		h.mustPrompt(id, "hello")
		_, resp := slash(t, h.c, h.client, id, "/clone")
		clone := acp.SessionId(switchTo(t, resp))
		if _, err := h.c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: clone, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
			t.Fatal(err)
		}
		if got, want := h.context(clone), "Output limit: none sent yet; the model's limit is 32,000 (the catalog)\n"; !strings.Contains(got, want) {
			t.Errorf("a loaded session's /context %q lacks %q", got, want)
		}
	})
	t.Run("forked_session", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, stated, doneWith("a", llm.SentLimit{Tokens: 32000, Sent: true, Basis: "request"}),
			doneWith("b", llm.SentLimit{Tokens: 32000, Sent: true, Basis: "request"}))
		cwd := t.TempDir()
		id := newSession(t, h.c, cwd, nil).SessionId
		h.mustPrompt(id, "first")
		h.mustPrompt(id, "second")
		list, _ := slash(t, h.c, h.client, id, "/fork")
		var second string
		for line := range strings.SplitSeq(list, "\n") {
			if f := strings.Fields(line); len(f) == 2 && f[1] == "second" {
				second = f[0]
			}
		}
		_, resp := slash(t, h.c, h.client, id, "/fork "+second)
		forked := acp.SessionId(switchTo(t, resp))
		if _, err := h.c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: forked, Cwd: cwd}); err != nil {
			t.Fatal(err)
		}
		if got, want := h.context(forked), "Output limit: none sent yet; the model's limit is 32,000 (the catalog)\n"; !strings.Contains(got, want) {
			t.Errorf("a forked session's /context %q lacks %q", got, want)
		}
	})
}

// promptResponseSentLimit is TestPromptResponseOutputLimit_0071's Phase 2c
// cases: the prompt response carries what the last call carried, the wire's
// tokens; {0, "none"} for a call that carried none; asked: true for a call
// that failed (0071-MADR P22, P32, P35).
func promptResponseSentLimit(t *testing.T) {
	t.Run("none_sent", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(32000)}, doneWith("a", llm.SentLimit{Basis: "none"}))
		wantMeta(t, h.mustPrompt(h.session(), "hello"), 0, "none")
	})
	t.Run("raised", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(32000)},
			doneWith("a", llm.SentLimit{Tokens: 36096, Sent: true, Basis: "request"}))
		wantMeta(t, h.mustPrompt(h.session(), "hello"), 36096, "stated")
	})
	// 0071 2c new names: begin
	t.Run("asked", func(t *testing.T) {
		meta, err := turnMetaAsked(&liveSession{queue: newQueue()}, false, llm.OutputLimit{Tokens: 32000, Basis: llm.BasisStated}, true)
		if err != nil {
			t.Fatal(err)
		}
		b, err := jsonx.Marshal(meta)
		if got, want := string(b), `{"gobble":{"outputLimit":{"tokens":32000,"basis":"stated","asked":true}}}`; err != nil || got != want {
			t.Errorf("a failed call's _meta %s (%v), want %s", got, err, want)
		}
	})
	// 0071 2c new names: end
}

// A refusal whose Observed states a cap below the limit sent: the turn is
// sent again once at it, basis refusal; the cap is handed to the run's
// store with its provider, and model.limit.learned is said once per session
// and model, before the reply (0071-MADR P19; 0066-MADR D7, D10, S12).
func TestRefusedCapLearned_0071(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(100000)},
		observedTurn("m1", 16000), llmtest.Text("ok"), observedTurn("m1", 12000), llmtest.Text("ok again"))
	// 0071 2c new names: begin
	caps := (&savedCaps{}).store(h)
	// 0071 2c new names: end
	id := h.session()
	resp := h.mustPrompt(id, "one")
	if got := maxTokensOf(h.calls()); !slices.Equal(got, []int{64000, 16000}) {
		t.Fatalf("calls asking %v, want 64,000, then the stated 16,000", got)
	}
	wantMeta(t, resp, 16000, "refusal")
	h.mustPrompt(id, "two")
	// 0071 2c new names: begin
	saved, _ := caps.got()
	if !slices.Equal(saved, []string{"fake/m1 16000", "fake/m1 12000"}) {
		t.Errorf("saved %q, want each stated cap with its provider", saved)
	}
	// 0071 2c new names: end
	learned := "The service stated m1's output limit as 16,000 tokens; gobble sent the call again at that and keeps it for a day.\n\n"
	if got := agentChunks(h.client); slices.Index(got, learned) < 0 || slices.Index(got, learned) > slices.Index(got, "ok") {
		t.Errorf("agent messages %q, want the learned line before the reply", got)
	}
}

// model.limit.learned is said once per session and model, whatever the
// number of caps learned (0066-MADR D10, S12).
func TestNotice_LimitLearned_0066(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(100000)},
		observedTurn("m1", 16000), llmtest.Text("ok"), observedTurn("m1", 12000), llmtest.Text("ok again"))
	id := h.session()
	h.mustPrompt(id, "one")
	h.mustPrompt(id, "two")
	n := 0
	for _, c := range agentChunks(h.client) {
		if strings.HasPrefix(c, "The service stated m1's output limit") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d learned lines over two turns, want one: %q", n, agentChunks(h.client))
	}
}

// A completed call's output tokens reach the run's store of saved caps, with
// the call's provider and model, which drops a saved cap they exceed
// (0071-MADR P25).
func TestSavedCapDroppedWhenExceeded_0071(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(32000)},
		llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "long"}, llm.Usage{InputTokens: 10, OutputTokens: 20000},
			llm.Done{StopReason: llm.StopEndTurn}}})
	caps := (&savedCaps{}).store(h)
	h.mustPrompt(h.session(), "hello")
	if _, outputs := caps.got(); !slices.Equal(outputs, []string{"fake/m1 20000"}) {
		t.Errorf("outputs %q, want the call's 20,000 for fake/m1", outputs)
	}
	// An estimated usage, a byte count over four, drops nothing (0071-PLAN's
	// review deviation of 2026-11-25, R3).
	t.Run("estimated", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(32000)},
			llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "long"}, llm.Usage{InputTokens: 10, OutputTokens: 20000, Estimated: true},
				llm.Done{StopReason: llm.StopEndTurn}}})
		caps := (&savedCaps{}).store(h)
		h.mustPrompt(h.session(), "hello")
		if _, outputs := caps.got(); len(outputs) != 0 {
			t.Errorf("outputs %q, want none from an estimated usage", outputs)
		}
	})
	// A request that names no model: the SDK saves the cap under the
	// provider's own model, so the output is reported under that model
	// (0071-PLAN's review deviation of 2026-11-25, R6).
	t.Run("saved_under", func(t *testing.T) {
		h := newLimitHarnessWith(t, SessionSettings{}, map[string]llm.Model{}, func(o *Options) {
			o.Models = func() ModelChoice {
				return ModelChoice{Provider: "fake", Providers: []ProviderChoice{{ID: "fake", Label: "Fake", State: StateLive}}}
			}
		}, observedTurn("fake-own-1", 9000), llmtest.Text("ok"),
			llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "long"}, llm.Usage{InputTokens: 10, OutputTokens: 20000},
				llm.Done{StopReason: llm.StopEndTurn}}})
		caps := (&savedCaps{}).store(h)
		id := h.session()
		h.mustPrompt(id, "one")
		h.mustPrompt(id, "two")
		saved, outputs := caps.got()
		if !slices.Equal(saved, []string{"fake/fake-own-1 9000"}) || !slices.Equal(outputs, []string{"fake/fake-own-1 2", "fake/fake-own-1 20000"}) {
			t.Errorf("saved %q, outputs %q; want each output under the model the cap was saved under", saved, outputs)
		}
	})
}

// refusedAgainObserved is TestLimitRefusalLine_0071's Phase 2c case: a
// refusal still refused after its resend at the cap Observed stated fails
// the turn with D8 (c)'s line, though gobble's own reader does not read its
// form (0071-MADR P34).
func refusedAgainObserved(t *testing.T) {
	// The refused-again text names no output-limit field and is no form
	// gobble reads: only its Observed tells the refusal (P34).
	again := observedTurnSaying("m1", 16000, "参数非法：限制数值范围[1,16000]")
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(100000)},
		observedTurn("m1", 16000), again)
	_, err := prompt(t, h.c, h.session(), "hello")
	want := "gobble asked for at most 16,000 output tokens (refusal); to send less, set maxTokens for m1 under modelOverrides in models.json"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("error %v, want the line %q", err, want)
	}
}

// The title's request is sent again once at a cap its refusal's Observed
// states below 128, and the cap is learned (0071-MADR P34).
func TestTitleObservedResent_0071(t *testing.T) {
	small := llmtest.NewScript(observedTurn("s1", 100), llmtest.Text("A title"))
	h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), small)
	// 0071 2c new names: begin
	var caps savedCaps
	h.ag.saveObservation = caps.save
	// 0071 2c new names: end
	id := newSession(t, h.c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, h.c, id, "Name this session"); err != nil {
		t.Fatal(err)
	}
	// The title's work ends either way; then its requests and name are read.
	waitFor(t, func() bool { return h.titleEnded(id) })
	if !h.titled("A title") {
		t.Error("the session took no title")
	}
	if got := maxTokensOf(small.Requests()); !slices.Equal(got, []int{128, 100}) {
		t.Errorf("title requests asking %v, want 128, then the stated 100", got)
	}
	// 0071 2c new names: begin
	if saved, _ := caps.got(); !slices.Equal(saved, []string{"small/s1 100"}) {
		t.Errorf("saved %q, want the title model's stated 100", saved)
	}
	// 0071 2c new names: end
}

// titleEnded reports that session id's title has run and ended.
func (h *titleHarness) titleEnded(id acp.SessionId) bool {
	h.ag.mu.Lock()
	done := h.ag.sessions[id].titleDone
	h.ag.mu.Unlock()
	if done == nil {
		return false
	}
	select {
	case <-done:
		return true
	default:
		return false
	}
}

// The title asks min(128, C), C the streamed output a turn reads: a kept
// stream cap of 100 below 128 is asked though the plain output is 8,192;
// unknown, 128 (0071-MADR V8; 0071-PLAN's second deviation of 2026-11-25).
func TestTitleCapFromLimits_0071(t *testing.T) {
	for _, c := range []struct {
		name string
		info llm.Model
		want int
	}{
		{"streamed", llm.Model{ContextWindow: 200000, MaxTokens: 8192, Limits: llm.Limits{
			Output:         llm.LimitFact{Tokens: 8192, Known: true, Source: "listing:small"},
			StreamedOutput: llm.LimitFact{Tokens: 100, Known: true, Conflicting: true, Source: "observed"}}}, 100},
		{"unknown", llm.Model{}, 128},
	} {
		t.Run(c.name, func(t *testing.T) {
			small := llmtest.NewScript(llmtest.Text("A title"))
			h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), small, func(o *Options) {
				o.ModelInfo = func(_ *provider.ModelData, provider, model string) llm.Model {
					m := c.info
					m.Provider, m.ID = provider, model
					return m
				}
			})
			id := newSession(t, h.c, t.TempDir(), nil).SessionId
			if _, err := prompt(t, h.c, id, "Name this session"); err != nil {
				t.Fatal(err)
			}
			waitFor(t, func() bool { return h.titled("A title") })
			if got := small.Requests()[0].MaxTokens; got != c.want {
				t.Errorf("the title asked for %d, want %d", got, c.want)
			}
		})
	}
}

// When discovery ends with the model's output limit unknown, the session
// says model.limit.unknown once, before its first reply; a second turn says
// nothing more; a model whose limit discovery found says nothing
// (0066-MADR D10 as amended 2026-11-25, S10; 0071-MADR V7).
func TestNotice_LimitUnknown_0066(t *testing.T) {
	line := "No source states m1's output limit; gobble asks for at most 16,384 tokens (assumed), fitted to the window; " +
		"set maxTokens for m1 in models.json to change it.\n\n"
	t.Run("unknown", func(t *testing.T) {
		h := discovered(t, &discoverRec{delay: 50 * time.Millisecond}, map[string]int{}, nil, llmtest.Text("one"), llmtest.Text("two"))
		id := h.session()
		h.mustPrompt(id, "hello")
		h.mustPrompt(id, "again")
		got := agentChunks(h.client)
		if n := strings.Count(strings.Join(got, ""), line); n != 1 || slices.Index(got, line) > slices.Index(got, "one") {
			t.Errorf("agent messages %q, want the line once, before the first reply", got)
		}
	})
	// A switch away and back discovers m1 again, which still ends unknown:
	// the line was said for this session and model, and is not said again.
	t.Run("switch_back", func(t *testing.T) {
		h := discovered(t, &discoverRec{delay: 20 * time.Millisecond}, map[string]int{"fake/m2": 32000}, nil,
			llmtest.Text("one"), llmtest.Text("two"), llmtest.Text("three"))
		id := h.session()
		h.mustPrompt(id, "hello")
		for _, m := range []string{"m2", "m1"} {
			if _, err := setOption(t, h.c, id, configModel, m); err != nil {
				t.Fatal(err)
			}
			h.mustPrompt(id, "on "+m)
		}
		if n := strings.Count(strings.Join(agentChunks(h.client), ""), line); n != 1 {
			t.Errorf("%d lines over a switch away and back, want one: %q", n, agentChunks(h.client))
		}
	})
	t.Run("known", func(t *testing.T) {
		h := discovered(t, &discoverRec{delay: 50 * time.Millisecond}, map[string]int{"fake/m1": 32000}, nil, llmtest.Text("one"))
		h.mustPrompt(h.session(), "hello")
		if got := agentChunks(h.client); slices.Contains(got, line) {
			t.Errorf("agent messages %q: a model discovery found a limit for says nothing", got)
		}
	})
}

// No settings key and no ACP option set the output limit; models.json's
// maxTokens is the one way (0066-MADR S32, S33, the owner's answer to
// go-llmprovider-sdk 0089-MADR OQ4, "models.json only, for now
// (Recommended)").
func TestConfig_NoOutputLimitKey_0066(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(32000)})
	resp := newSession(t, h.c, t.TempDir(), nil)
	for _, o := range resp.ConfigOptions {
		if o.Select == nil || (o.Select.Id != configModel && o.Select.Id != configThinking) {
			t.Errorf("config option %+v: only the model and thinking are offered", o)
		}
	}
}

// /compact's summary refused in a form only the SDK reads is sent again once
// at the cap its Observed states, and the cap is learned for the run's store
// (0071-MADR P34).
func TestSummaryCapRefusalResent_0071(t *testing.T) {
	t.Run("observed_saved", func(t *testing.T) {
		h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": wide(100000)},
			llmtest.Text(strings.Repeat("one ", 200)), llmtest.Text(strings.Repeat("two ", 200)),
			observedTurn("m1", 9000), llmtest.Text("a summary"), llmtest.Text("a prefix summary"))
		// 0071 2c new names: begin
		caps := (&savedCaps{}).store(h)
		// 0071 2c new names: end
		id := h.session()
		h.mustPrompt(id, strings.Repeat("first ", 200))
		h.mustPrompt(id, strings.Repeat("second ", 200))
		text, _ := slash(t, h.c, h.client, id, "/compact")
		if s := maxTokensOf(h.summaries()); len(s) < 2 || s[0] != 64000 || s[1] != 9000 {
			t.Errorf("summaries asking %v (%.120q), want 64,000, then the stated 9,000", s, text)
		}
		// 0071 2c new names: begin
		if saved, _ := caps.got(); !slices.Equal(saved, []string{"fake/m1 9000"}) {
			t.Errorf("saved %q, want the summary's stated 9,000", saved)
		}
		// 0071 2c new names: end
	})
}

// descriptions are the model option's descriptions in opts, by value; ""
// for an option with none.
func descriptions(opts []acp.SessionConfigOption) map[string]string {
	out := map[string]string{}
	for _, o := range opts {
		if o.Select == nil || o.Select.Id != configModel || o.Select.Options.Grouped == nil {
			continue
		}
		for _, g := range *o.Select.Options.Grouped {
			for _, v := range g.Options {
				d := ""
				if v.Description != nil {
					d = *v.Description
				}
				out[string(v.Value)] = d
			}
		}
	}
	return out
}

// Each model option is described "context N · max-out M" from the limits
// cached now, "unknown" for a limit not known; the description reads
// Options.CachedLimits, which starts no request, so building the options
// sends none (0066-MADR D6, S19; 0071-MADR P46, the owner's "A fetch-free
// cached hook (Recommended)").
func TestACP_ModelOptionDescription_0066(t *testing.T) {
	t.Run("described", func(t *testing.T) {
		h := newLimitHarnessWith(t, SessionSettings{}, map[string]llm.Model{}, func(o *Options) {
			o.CachedLimits = func(_ *provider.ModelData, provider, model string) llm.Limits {
				if provider+"/"+model != "fake/m1" {
					return llm.Limits{}
				}
				return llm.Limits{Input: llm.LimitFact{Tokens: 200000, Known: true, Source: "listing:fake"},
					Output:         llm.LimitFact{Tokens: 64000, Known: true, Source: "listing:fake"},
					StreamedOutput: llm.LimitFact{Tokens: 32000, Known: true, Source: "observed"}}
			}
		})
		d := descriptions(newSession(t, h.c, t.TempDir(), nil).ConfigOptions)
		if d["fake/m1"] != "context 200,000 · max-out 32,000" || d["fake/m2"] != "context unknown · max-out unknown" {
			t.Errorf("descriptions %q, want m1's cached limits and m2's unknown", d)
		}
	})
	t.Run("no_request", func(t *testing.T) {
		svc := &claudeService{}
		cache := &provider.Cache{}
		o := provider.Options{APIKey: "sk-describe", HTTPClient: &http.Client{Transport: svc}, Cache: cache, Retry: &llm.Retry{Off: true}}
		choice := ModelChoice{Provider: "claude", Models: []string{"claude-test-1", "claude-test-2"}, Default: "claude-test-1"}
		ag := New(Options{Version: "1.2.3", Store: jsonl.New(t.TempDir(), jsonl.Options{}),
			Provider: func(*provider.ModelData, string, llm.Retry) (llm.Provider, error) { return provider.New("claude", o) },
			Models:   func() ModelChoice { return choice },
			CachedLimits: func(_ *provider.ModelData, id, model string) llm.Limits {
				return provider.CachedLimits(o, id, model, nil)
			}})
		c := acptest.Connect(t, ag, &acptest.Client{})
		ag.SetConnection(c.Agent)
		t.Cleanup(func() {
			if err := ag.Close(); err != nil {
				t.Error(err)
			}
		})
		initialize(t, c.Client)
		d := descriptions(newSession(t, c, t.TempDir(), nil).ConfigOptions)
		// The cache's close waits for any listing a lookup started.
		if err := cache.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		if got := svc.seen(); len(got) != 0 || d["claude/claude-test-1"] == "" {
			t.Errorf("requests %q while describing %q; want none, and each option described", got, d)
		}
	})
}
