package compaction

import (
	"errors"
	"strings"
	"testing"

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

// A summary's output is capped at 80 percent of the reserve, and a split
// turn's prefix at half, each no more than the model's MaxTokens; an
// unknown MaxTokens caps nothing (0005-PLAN F5-2 decision 7).
func TestSummaryCaps(t *testing.T) {
	split := func() []session.Entry {
		return (&path{}).
			add(text(session.RoleUser, chars(400))).
			add(text(session.RoleAssistant, chars(400))).
			add(text(session.RoleUser, chars(400))).
			add(text(session.RoleAssistant, chars(400))).
			add(text(session.RoleToolResult, chars(400))).
			add(text(session.RoleAssistant, chars(400))).entries
	}
	for _, c := range []struct {
		name            string
		reserve, max    int
		summary, prefix int
	}{
		{"the reserve's shares", 10000, 100000, 8000, 5000},
		{"the model's limit", 10000, 6000, 6000, 5000},
		{"unknown limit", 10000, 0, 0, 0},
	} {
		prep, err := Prepare(split(), Settings{ReserveTokens: c.reserve, KeepRecentTokens: 150})
		if err != nil || !prep.SplitTurn || len(prep.toSummarize) == 0 {
			t.Fatalf("%s: a split turn with history, %+v, %v", c.name, prep, err)
		}
		s := llmtest.NewScript(llmtest.Text("H"), llmtest.Text("P"))
		if _, err := Compact(t.Context(), prep, Model{Provider: s, MaxTokens: c.max}, Request{}); err != nil {
			t.Fatal(err)
		}
		reqs := s.Requests()
		if len(reqs) != 2 || reqs[0].MaxTokens != c.summary || reqs[1].MaxTokens != c.prefix {
			t.Errorf("%s: caps %v, want %d then %d", c.name, caps(reqs), c.summary, c.prefix)
		}
	}
}

func caps(reqs []*llm.Request) []int {
	out := make([]int, len(reqs))
	for i, r := range reqs {
		out[i] = r.MaxTokens
	}
	return out
}
