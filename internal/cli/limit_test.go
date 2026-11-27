package cli

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// quotaFrame is acpext's prompt-quota.error.json: the data of a turn
// refused on OpenCode Zen's free tier, which lifts at 14:05 UTC.
func quotaFrame(t *testing.T) (string, acpext.ErrorData) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "acpext", "testdata", "prompt-quota.error.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d acpext.ErrorData
	if err := jsonx.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return string(b), d
}

// The limit line names the limit in words and the provider, and when the
// limit lifts on the local clock with the wait left (0049-MADR Q3 (a)).
func TestLimitLine(t *testing.T) {
	cst := time.FixedZone("CST", -6*60*60)
	now := time.Date(2026, 11, 9, 7, 0, 0, 0, cst) // 13:00 UTC
	for _, c := range []struct {
		name string
		d    acpext.ErrorData
		want string
	}{
		{"free tier", acpext.ErrorData{Limit: acpext.LimitFreeTier, Provider: "opencode-zen", ResetAt: "2026-11-09T14:05:00Z"},
			"Free tier limit on opencode-zen; resets at 08:05, in 1h5m"},
		{"under a minute", acpext.ErrorData{Limit: acpext.LimitRate, Provider: "openai", ResetAt: "2026-11-09T13:00:40Z"},
			"Rate limit on openai; resets at 07:00, in 40s"},
		{"passed", acpext.ErrorData{Limit: acpext.LimitDaily, Provider: "gemini", ResetAt: "2026-11-09T12:00:00Z"},
			"Daily quota on gemini; resets at 06:00"},
		{"no reset", acpext.ErrorData{Limit: acpext.LimitCredits, Provider: "kilo"}, "Credits used up on kilo"},
		{"no limit", acpext.ErrorData{Provider: "claude", ResetAt: "2026-11-09T15:00:00Z"}, "Limit on claude; resets at 09:00, in 2h"},
		{"a limit not in the table", acpext.ErrorData{Limit: "hourly"}, "hourly"},
		{"neither", acpext.ErrorData{Provider: "claude", Kind: acpext.KindOffline}, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := limitLine(c.d, now)
			if got != c.want || ok != (c.want != "") {
				t.Fatalf("limitLine = %q %v, want %q", got, ok, c.want)
			}
		})
	}
	for _, s := range []acpext.LimitScope{acpext.LimitRate, acpext.LimitWindow, acpext.LimitDaily, acpext.LimitWeekly,
		acpext.LimitMonthly, acpext.LimitCredits, acpext.LimitFreeTier, acpext.LimitPlan} {
		if limitWords[s] == "" {
			t.Errorf("no words for %q", s)
		}
	}
}

// A turn refused on a limit prints the agent's message, not the JSON-RPC
// error, and the limit line under it; the session goes on.
func TestLineSessionLimitError(t *testing.T) {
	data, d := quotaFrame(t)
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{ErrCode: -32603, ErrMessage: "Internal error", ErrData: data},
		{Updates: []string{chunk("fine")}}}}
	h := newLineHarness(t, agent)
	h.e.now = func() time.Time { return time.Date(2026, 11, 9, 13, 0, 0, 0, time.UTC) }
	h.start(Prompt{})
	h.press("one\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "resets at") })
	h.press("two\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "fine") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	got := h.stderr.String()
	if !strings.Contains(got, "Error: "+d.Error+"\n") || !strings.Contains(got, "Free tier limit on opencode-zen; resets at 14:05, in 1h5m\n") ||
		strings.Contains(got, "session/prompt:") || strings.Contains(got, `"code"`) {
		t.Fatalf("stderr %q", got)
	}
}

// A turn refused on credit by a call that asked for an output limit prints
// the agent's message, the limit line and, on the line after, the member
// outputLimitNote; a refusal that carries none prints no note (0071-MADR
// D8 (d) as amended 2026-11-24; L43).
func TestLineSessionOutputLimitNote_0071(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "acpext", "testdata", "prompt-output-limit-note.error.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d acpext.ErrorData
	if err := jsonx.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	if d.OutputLimitNote == "" {
		t.Fatal("acpext's example frame carries no outputLimitNote")
	}
	quota, _ := quotaFrame(t)
	for _, c := range []struct {
		name, data, want string // want "" for no note
	}{
		{"with_note", string(b), "Error: " + d.Error + "\nCredits used up on kilo\n" + d.OutputLimitNote + "\n"},
		{"without_note", quota, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{ErrCode: -32603, ErrMessage: "Internal error", ErrData: c.data}}}
			h := newLineHarness(t, agent)
			h.e.now = func() time.Time { return time.Date(2026, 11, 9, 13, 0, 0, 0, time.UTC) }
			h.start(Prompt{})
			h.press("one\r")
			waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Error: ") })
			h.press("\x04")
			if err := h.wait(); err != nil {
				t.Fatal(err)
			}
			got := h.stderr.String()
			if c.want != "" && !strings.Contains(got, c.want) {
				t.Errorf("stderr %q, want it to hold %q", got, c.want)
			}
			if c.want == "" && strings.Contains(got, "gobble asked for") {
				t.Errorf("stderr %q, want no note", got)
			}
		})
	}
}

// Print mode's JSON result carries the agent's message as error and its
// data as errorData (0049-MADR Q3 (a)).
func TestPrintErrorData(t *testing.T) {
	data, d := quotaFrame(t)
	withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{ErrCode: -32603, ErrMessage: "Internal error", ErrData: data}}})
	r := runMain(t, nil, "-p", "--output-format", "json", "x")
	if r.code != ExitFailure {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	var res struct {
		Error     string         `json:"error"`
		ErrorData jsontext.Value `json:"errorData"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(r.stdout)), &res); err != nil {
		t.Fatalf("stdout %q: %v", r.stdout, err)
	}
	var got acpext.ErrorData
	if err := jsonx.UnmarshalStrict(res.ErrorData, &got); err != nil {
		t.Fatalf("errorData %s: %v", res.ErrorData, err)
	}
	gb, _ := jsonx.Marshal(got) //nolint:errcheck // a decoded ErrorData encodes
	wb, _ := jsonx.Marshal(d)   //nolint:errcheck // as above
	if res.Error != d.Error || string(gb) != string(wb) {
		t.Fatalf("result error %q, errorData %s; want %q and %s", res.Error, gb, d.Error, wb)
	}
}

// A capped call's credit refusal, in print mode: the JSON result's errorData
// carries the note, and the text output is as it was, with no note line
// (0071-MADR L45).
func TestPrintErrorDataNote_0071(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "acpext", "testdata", "prompt-output-limit-note.error.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d acpext.ErrorData
	if err := jsonx.Unmarshal(b, &d); err != nil || d.OutputLimitNote == "" {
		t.Fatalf("acpext's example frame: note %q, %v", d.OutputLimitNote, err)
	}
	turn := acptest.Turn{ErrCode: -32603, ErrMessage: "Internal error", ErrData: string(b)}
	t.Run("json", func(t *testing.T) {
		withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{turn}})
		r := runMain(t, nil, "-p", "--output-format", "json", "x")
		if r.code != ExitFailure {
			t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
		}
		var res struct {
			ErrorData acpext.ErrorData `json:"errorData"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(r.stdout)), &res); err != nil {
			t.Fatalf("stdout %q: %v", r.stdout, err)
		}
		if res.ErrorData.OutputLimitNote != d.OutputLimitNote {
			t.Errorf("errorData's outputLimitNote %q, want %q", res.ErrorData.OutputLimitNote, d.OutputLimitNote)
		}
	})
	t.Run("text", func(t *testing.T) {
		withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{turn}})
		r := runMain(t, nil, "-p", "x")
		if r.code != ExitFailure {
			t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
		}
		if strings.Contains(r.stdout+r.stderr, "gobble asked for") {
			t.Errorf("text output carries the note: stdout %q, stderr %q", r.stdout, r.stderr)
		}
	})
}

// cutCallTurn is a reply whose text the output limit followed by cutting a
// tool call, which the SDK dropped and named in its warning.
func cutCallTurn() llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "Let me read it."},
		llm.Warning{Kind: "tool_call_cut", Model: "m", Detail: "the output limit cut call 1 (read)"}, // llm.WarningToolCallCut
		llm.Usage{InputTokens: 3, OutputTokens: 2}, llm.Done{StopReason: llm.StopMaxTokens}}}
}

// toolCutWords are turn.stop.max_tokens_tool in the default personality,
// dry, before its model; the line wraps on a terminal after "dropped.".
const toolCutWords = "The reply ran out of room in the middle of a tool call, so the call was dropped. Ask again, or set maxTokens for "

// Text, then a call the output limit cut: the terminal shows the text, then
// the agent's line on why the turn ended; headless JSON carries max_tokens
// and the line after the text (0071-MADR P29, the owner's "End the turn,
// keep the text, notify (Recommended)"; 0065-MADR D8).
func TestCutCallAfterText_0071(t *testing.T) {
	t.Run("cli", func(t *testing.T) {
		h := newModelHarness(t, llmtest.NewScript(cutCallTurn()), t.TempDir())
		h.start(Prompt{})
		h.press("read it\r")
		// The turn has ended once the prompt is drawn again after the reply.
		waitUntil(t, func() bool {
			_, after, ok := strings.Cut(h.stdout.String(), "Let me read it.")
			return ok && strings.Contains(after, "\r> ")
		})
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		out := h.stdout.String()
		first, rest, _ := strings.Cut(toolCutWords, " Ask again")
		text, line, more := strings.Index(out, "Let me read it."), strings.Index(out, first), strings.Index(out, "Ask again"+rest)
		if text < 0 || line < text || more < line {
			t.Fatalf("stdout %q, want the text, then the line", out)
		}
	})
	t.Run("headless_json", func(t *testing.T) {
		withModel(t, llmtest.NewScript(cutCallTurn()))
		r := runMain(t, nil, "-p", "--output-format", "json", "read it")
		var res printResult
		if err := jsonx.Unmarshal([]byte(strings.TrimSpace(r.stdout)), &res); err != nil {
			t.Fatalf("stdout %q: %v", r.stdout, err)
		}
		if res.StopReason != "max_tokens" || !strings.HasPrefix(res.Text, "Let me read it.\n\n"+toolCutWords) {
			t.Fatalf("result %+v, want max_tokens and the text, then the line", res)
		}
	})
}

// builtInListing is a listing of two OpenAI models the test data knows:
// gpt-5-mini takes images, o3-mini text only (TestModelDataCurrent_0072's
// copy states the reverse).
type builtInListing struct{}

func (builtInListing) RoundTrip(r *http.Request) (*http.Response, error) {
	body, status := "{}", http.StatusNotFound
	if r.URL.Host+r.URL.Path == "api.openai.com/v1/models" {
		body, status = `{"object":"list","data":[{"id":"gpt-5-mini","object":"model"},{"id":"o3-mini","object":"model"}]}`, http.StatusOK
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

// slowDiscovery replaces the run's discovery with one that records each
// model and, after 200 ms, lets the model facts state a 32,000 cap, as a
// listing read would; before it ends the facts are the offline lookup's (the
// test data's cap, under the 64,000 ceiling). Both are restored when t
// ends. A key in the
// environment gives the session a model, as a run with a credential has;
// nothing here reads it.
func slowDiscovery(t *testing.T) func() []string {
	t.Helper()
	t.Setenv("OPENAI_API_KEY", "sk-test-only")
	var mu sync.Mutex
	var calls []string
	var done atomic.Bool
	oldDiscover, oldFacts := discoverLimits, modelFacts
	discoverLimits = func(ctx context.Context, o provider.Options, id, model string, _ *catalog.ModelsJSON) error {
		mu.Lock()
		calls = append(calls, id+"/"+model+" key="+o.APIKey+" workspace="+o.Workspace)
		mu.Unlock()
		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
		done.Store(true)
		return nil
	}
	modelFacts = func(o provider.Options, id, model string, mj *catalog.ModelsJSON) llm.Model {
		if done.Load() {
			return llm.Model{Provider: id, ID: model, ContextWindow: 1000000, MaxTokens: 32000, MaxTokensFrom: llm.CapCatalog}
		}
		return offlineModelFacts(o, id, model, mj)
	}
	t.Cleanup(func() { discoverLimits, modelFacts = oldDiscover, oldFacts })
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(calls)
	}
}

// The CLI's interactive start and a headless run discover the session's
// model when it starts, and its first call waits for it and carries what it
// found; the second call discovers nothing more (0066-MADR S3, S4, S9;
// 0071-MADR P37).
func TestSession_DiscoversBeforeFirstCall_0066(t *testing.T) {
	t.Run("cli", func(t *testing.T) {
		seen := slowDiscovery(t)
		s := llmtest.NewScript(llmtest.Text("one"), llmtest.Text("two"))
		h := newModelHarness(t, s, t.TempDir())
		h.start(Prompt{})
		h.press("first\r")
		waitUntil(t, func() bool { return strings.Count(h.stdout.String(), "\r> ") >= 2 })
		h.press("second\r")
		waitUntil(t, func() bool { return strings.Count(h.stdout.String(), "\r> ") >= 3 })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		reqs := s.Requests()
		if len(reqs) != 2 || reqs[0].MaxTokens != 32000 || reqs[1].MaxTokens != 32000 || !slices.Equal(seen(), []string{wantDiscovery}) {
			t.Errorf("%d calls asking %v, discoveries %q; want two calls at the discovered 32,000 and one discovery", len(reqs),
				capsOf(reqs), seen())
		}
	})
	t.Run("headless", func(t *testing.T) {
		seen := slowDiscovery(t)
		s := llmtest.NewScript(llmtest.Text("ok"))
		withModel(t, s)
		if r := runMain(t, nil, "-p", "hello"); r.code != ExitOK {
			t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
		}
		if reqs := s.Requests(); len(reqs) != 1 || reqs[0].MaxTokens != 32000 || !slices.Equal(seen(), []string{wantDiscovery}) {
			t.Errorf("calls asking %v, discoveries %q; want one call at the discovered 32,000 after one discovery", capsOf(reqs), seen())
		}
	})
	// A stored Claude key's Anthropic workspace reaches discovery beside
	// the key, as it reaches the lookup (0035-MADR D5; 0071-MADR P37;
	// 0071-PLAN Phase 2b's delta check).
	t.Run("workspace", func(t *testing.T) {
		seen := slowDiscovery(t)
		c := newCfgEnv(t)
		c.write(c.authFile(), `{"claude": {"type": "api_key", "key": "sk-ant-test-only", "workspace": "wrkspc_1"}}`)
		s := llmtest.NewScript(llmtest.Text("ok"))
		withModel(t, s)
		if r := c.run("--model", "claude/claude-test-1", "-p", "hello"); r.code != ExitOK {
			t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
		}
		want := "claude/claude-test-1 key=sk-ant-test-only workspace=wrkspc_1"
		if got := seen(); !slices.Equal(got, []string{want}) {
			t.Errorf("discoveries %q, want %q: the stored key with its workspace", got, want)
		}
	})
}

// wantDiscovery is the one discovery a run with OPENAI_API_KEY makes: the
// session's model, with the run's key and its workspace (none for OpenAI).
const wantDiscovery = "openai/gpt-6-luna key=sk-test-only workspace="

// capsOf are rs' MaxTokens.
func capsOf(rs []*llm.Request) []int {
	var out []int
	for _, r := range rs {
		out = append(out, r.MaxTokens)
	}
	return out
}
