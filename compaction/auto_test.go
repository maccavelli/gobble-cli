package compaction

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
)

// Over is decision 1's test: over the window less the reserve, at the
// percentage, at the token count; an unknown window triggers only the
// token count (0005-PLAN F5-2).
func TestOver(t *testing.T) {
	for _, c := range []struct {
		name         string
		used, window int
		s            Settings
		want         bool
	}{
		{"at the reserve", 900, 1000, Settings{ReserveTokens: 100}, false},
		{"over the reserve", 901, 1000, Settings{ReserveTokens: 100}, true},
		{"below the percentage", 499, 1000, Settings{ReserveTokens: 100, ThresholdPercent: 50}, false},
		{"at the percentage", 500, 1000, Settings{ReserveTokens: 100, ThresholdPercent: 50}, true},
		{"below the floored percentage", 328, 999, Settings{ReserveTokens: 1, ThresholdPercent: 33}, false}, // 329.67 floors to 329
		{"at the floored percentage", 329, 999, Settings{ReserveTokens: 1, ThresholdPercent: 33}, true},
		{"below the tokens", 299, 1000, Settings{ReserveTokens: 100, ThresholdTokens: 300}, false},
		{"at the tokens", 300, 1000, Settings{ReserveTokens: 100, ThresholdTokens: 300}, true},
		{"unknown window, over everything else", 5000, 0, Settings{ReserveTokens: 100, ThresholdPercent: 1}, false},
		{"unknown window, at the tokens", 300, 0, Settings{ReserveTokens: 100, ThresholdPercent: 1, ThresholdTokens: 300}, true},
		{"percentage off", 999, 2000, Settings{ReserveTokens: 100}, false},
	} {
		if got := Over(c.used, c.window, c.s); got != c.want {
			t.Errorf("%s: Over(%d, %d, %+v) = %v, want %v", c.name, c.used, c.window, c.s, got, c.want)
		}
	}
}

// guardPrep is a preparation whose history to summarise is the given
// number of characters, or so.
func guardPrep(t *testing.T, size int) *Preparation {
	t.Helper()
	p := (&path{}).add(text(session.RoleUser, chars(size))).add(text(session.RoleUser, chars(400)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 50})
	if err != nil {
		t.Fatal(err)
	}
	return prep
}

// An empty summary, and a short summary of a long history, are asked for
// once more, then refused, wrapping ErrSummaryRefused; a short summary of
// a short history is taken (0005-PLAN F5-2 decision 6).
func TestSummaryGuard(t *testing.T) {
	short := llmtest.Text(strings.Repeat("s", floorChars-1))
	long := llmtest.Text(strings.Repeat("l", floorChars))
	for _, c := range []struct {
		name    string
		size    int
		turns   []llmtest.Turn
		want    string // the summary, or the error
		refused bool
	}{
		{"empty twice", 400, []llmtest.Turn{llmtest.Text(" \n"), llmtest.Text("")}, "Summarization failed: the summary is empty", true},
		{"empty then a summary", 400, []llmtest.Turn{llmtest.Text(""), llmtest.Text("S")}, "S", false},
		{"short of a long history twice", floorSource, []llmtest.Turn{short, short},
			"Summarization failed: the summary is too short (499 characters)", true},
		{"short then long, of a long history", floorSource, []llmtest.Turn{short, long}, strings.Repeat("l", floorChars), false},
		{"short of a short history", floorSource - 1000, []llmtest.Turn{short}, strings.Repeat("s", floorChars-1), false},
	} {
		s := llmtest.NewScript(c.turns...)
		res, err := Compact(t.Context(), guardPrep(t, c.size), Model{Provider: s}, Request{})
		if c.refused {
			if err == nil || err.Error() != c.want || !errors.Is(err, ErrSummaryRefused) {
				t.Errorf("%s: err %v, want %q wrapping ErrSummaryRefused", c.name, err, c.want)
			}
		} else if err != nil || res.Summary != c.want {
			t.Errorf("%s: summary %q, %v; want %q", c.name, res.Summary, err, c.want)
		}
		if got := len(s.Requests()); got != len(c.turns) {
			t.Errorf("%s: %d requests, want %d", c.name, got, len(c.turns))
		}
	}
}

// The other refusals wrap ErrSummaryRefused, and are not asked for again;
// a provider's failure does not wrap it (0005-PLAN F5-2 decision 11).
func TestRefusalsWrap(t *testing.T) {
	for _, c := range []struct {
		turn    llmtest.Turn
		refused bool
	}{
		{llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "part"}, llm.Done{StopReason: llm.StopMaxTokens}}}, true},
		{llmtest.ToolCall("x", "read", `{}`), true},
		{llmtest.Turn{Err: llm.ErrContextOverflow}, false},
	} {
		s := llmtest.NewScript(c.turn, llmtest.Text("never"))
		_, err := Compact(t.Context(), guardPrep(t, 400), Model{Provider: s}, Request{})
		if err == nil || errors.Is(err, ErrSummaryRefused) != c.refused || len(s.Requests()) != 1 {
			t.Errorf("%v: err %v, %d requests; want refused %v after one", c.turn, err, len(s.Requests()), c.refused)
		}
	}
}

// splitPath is a split turn with history to summarise.
func splitPath() []session.Entry {
	return (&path{}).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400))).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400))).
		add(text(session.RoleToolResult, chars(400))).
		add(text(session.RoleAssistant, chars(400))).entries
}

// A summary and a split turn's prefix each ask for the model's output cap,
// under the 64,000 ceiling, fitted to the room the window leaves over that
// request's own estimate; an unknown cap is the assumed 16,384; the
// reserve's 80 and 50 percent shares are gone (0071-MADR D6, L17, L18,
// replacing 0005-PLAN F5-2 decision 7's shares).
func TestSummaryCaps(t *testing.T) {
	for _, c := range []struct {
		name            string
		max             int
		summary, prefix int
	}{
		{"no_window_ceiling", 100000, 64000, 64000},
		{"model_limit", 6000, 6000, 6000},
		{"unknown", 0, 16384, 16384},
	} {
		t.Run(c.name, func(t *testing.T) {
			prep, err := Prepare(splitPath(), Settings{ReserveTokens: 10000, KeepRecentTokens: 150})
			if err != nil || !prep.SplitTurn || len(prep.toSummarize) == 0 {
				t.Fatalf("a split turn with history, %+v, %v", prep, err)
			}
			s := llmtest.NewScript(llmtest.Text("H"), llmtest.Text("P"))
			if _, err := Compact(t.Context(), prep, Model{Provider: s, MaxTokens: c.max}, Request{}); err != nil {
				t.Fatal(err)
			}
			reqs := s.Requests()
			if len(reqs) != 2 || reqs[0].MaxTokens != c.summary || reqs[1].MaxTokens != c.prefix {
				t.Errorf("caps %v, want %d then %d", caps(reqs), c.summary, c.prefix)
			}
		})
	}
	// 0071 new names: begin
	t.Run("window", func(t *testing.T) {
		prep, err := Prepare(splitPath(), Settings{ReserveTokens: 10000, KeepRecentTokens: 150})
		if err != nil {
			t.Fatal(err)
		}
		s := llmtest.NewScript(llmtest.Text("H"), llmtest.Text("P"))
		if _, err := Compact(t.Context(), prep, Model{Provider: s, MaxTokens: 100000, Window: 20000}, Request{}); err != nil {
			t.Fatal(err)
		}
		reqs := s.Requests()
		if len(reqs) != 2 {
			t.Fatalf("%d requests, want 2", len(reqs))
		}
		for i, r := range reqs {
			if want := 20000 - requestEstimate(r) - llm.ContextSafetyTokens; r.MaxTokens != want || want >= 64000 || want < llm.MinOutputTokens {
				t.Errorf("request %d: cap %d, want the room %d", i, r.MaxTokens, want)
			}
		}
		if reqs[0].MaxTokens == reqs[1].MaxTokens {
			t.Errorf("caps %v: each request's room is its own", caps(reqs))
		}
	})
	// 0071 new names: end
}

// requestEstimate is a summary request's own estimate: its system prompt
// and its prompt (0071-MADR D6).
func requestEstimate(r *llm.Request) int {
	n := 0
	for _, m := range r.Messages {
		for _, c := range m.Content {
			n += EstimateText(c.Text)
		}
	}
	return n
}

// 0071 new names: begin

// Each step of the ladder asks for its own cap: after an overflow drops
// tool results, the next request's room is computed over its smaller
// prompt (0071-MADR D6, L17).
func TestSummaryCapPerLadderStep_0071(t *testing.T) {
	overflow := llmtest.Turn{Err: llm.ErrContextOverflow}
	s := llmtest.NewScript(overflow, llmtest.Text("DONE"))
	msgs := toolMsgs(10)
	for _, m := range msgs {
		m.SetText(chars(4000))
	}
	if _, _, err := summariseFitting(t.Context(), Model{Provider: s, MaxTokens: 64000, Window: 30000}, "SYS", msgs,
		func(c string) string { return c }, "Summarization", false); err != nil {
		t.Fatal(err)
	}
	reqs := s.Requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests, want 2", len(reqs))
	}
	for i, r := range reqs {
		if want := 30000 - requestEstimate(r) - llm.ContextSafetyTokens; r.MaxTokens != want {
			t.Errorf("request %d: cap %d, want its room %d", i, r.MaxTokens, want)
		}
	}
	if reqs[1].MaxTokens <= reqs[0].MaxTokens {
		t.Errorf("caps %v: the smaller prompt's room is larger", caps(reqs))
	}
}

// A summary refused in a form that states the room is sent again once at
// that room, the same prompt and no tool result dropped; a cap form at its
// cap; an overflow the cap did not cause still takes the ladder
// (0071-MADR D6, D8 (a), L40).
func TestSummaryCapRefusalResent_0071(t *testing.T) {
	window := func(input, sent int) llmtest.Turn { // the OpenRouter form: a room of 131,072 less input
		msg := fmt.Sprintf("This endpoint's maximum context length is 131072 tokens. However, you requested about %d tokens (%d of text input, %d in the output).", input+sent, input, sent)
		return llmtest.Turn{Err: &llm.APIError{Status: 400, Message: msg, Err: fmt.Errorf("%w: %s", llm.ErrContextOverflow, msg)}}
	}
	capForm := llmtest.Turn{Err: &llm.APIError{Status: 400, Message: "max_tokens: 64000 > 8192, which is the maximum allowed number of output tokens"}}
	// Kilo's range form, which only the SDK reads: its cap arrives as
	// Observed, read first (0071-MADR P34).
	rangeMsg := "The max_tokens parameter is illegal.：限制数值范围[1,9000]"
	observed := llmtest.Turn{Err: &llm.APIError{Status: 400, Message: rangeMsg, Err: errors.New(rangeMsg),
		Observed: llm.LimitObservation{Model: "m", Tokens: 9000, Seen: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)}}}
	for _, c := range []struct {
		name  string
		first llmtest.Turn
		want  []int // each request's cap
	}{
		{"window_form", window(111072, 64000), []int{64000, 20000}},
		{"cap_form", capForm, []int{64000, 8192}},
		{"observed", observed, []int{64000, 9000}},
		{"input_overflow", llmtest.Turn{Err: llm.ErrContextOverflow}, []int{64000, 64000}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := llmtest.NewScript(c.first, llmtest.Text("DONE"))
			got, _, err := summariseFitting(t.Context(), Model{Provider: s, MaxTokens: 100000}, "SYS", toolMsgs(4),
				func(c string) string { return c }, "Summarization", false)
			if err != nil || got != "DONE" {
				t.Fatalf("%q, %v", got, err)
			}
			reqs := s.Requests()
			if !slices.Equal(caps(reqs), c.want) {
				t.Errorf("caps %v, want %v", caps(reqs), c.want)
			}
			dropped := strings.Count(promptOf(reqs[len(reqs)-1]), omittedResult)
			if resent := c.name != "input_overflow"; resent == (dropped > 0) {
				t.Errorf("the second request left out %d tool results; resent at the stated limit %v", dropped, resent)
			}
		})
	}
	// Refused again for its limit: the summary fails, with no ladder.
	s := llmtest.NewScript(window(111072, 64000), window(121072, 20000), llmtest.Text("never"))
	if _, _, err := summariseFitting(t.Context(), Model{Provider: s, MaxTokens: 100000}, "SYS", toolMsgs(4),
		func(c string) string { return c }, "Summarization", false); err == nil || len(s.Requests()) != 2 {
		t.Errorf("refused again: %v after %d requests, want the refusal after 2", err, len(s.Requests()))
	}
}

// 0071 new names: end

func caps(reqs []*llm.Request) []int {
	out := make([]int, len(reqs))
	for i, r := range reqs {
		out[i] = r.MaxTokens
	}
	return out
}
