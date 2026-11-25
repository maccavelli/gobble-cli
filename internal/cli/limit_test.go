package cli

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

// builtInListing is a listing of two OpenAI models the built-in copy knows:
// gpt-5-mini takes images, o3-mini text only.
type builtInListing struct{}

func (builtInListing) RoundTrip(r *http.Request) (*http.Response, error) {
	body, status := "{}", http.StatusNotFound
	if r.URL.Host+r.URL.Path == "api.openai.com/v1/models" {
		body, status = `{"object":"list","data":[{"id":"gpt-5-mini","object":"model"},{"id":"o3-mini","object":"model"}]}`, http.StatusOK
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

// Until a refreshed copy is kept, production's listings rank by the built-in
// models.dev copy: the listing's Options carry it, and a listing served in
// memory reads the copy's facts as the copy states them; the options New and
// the sign-in wizard are given carry none (0071-MADR P6, the owner's
// "Built-in copy to listings only (Recommended)").
func TestListingRanksByBuiltInCopy_0071(t *testing.T) {
	e, _ := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	listed := make(chan provider.Options, 1)
	withLister(t, func(_ context.Context, _, _ string, o provider.Options) (provider.Listing, error) {
		listed <- o
		return provider.Listing{}, errors.New("recorded")
	})
	l := e.modelListing(provider.Selection{ID: "openai"})
	l.start()
	o := <-listed
	if o.ModelData == nil {
		t.Fatal("the listing's Options carry no document")
	}
	copyData, err := provider.ParseModelData(bytes.NewReader(catalog.Snapshot()))
	if err != nil {
		t.Fatal(err)
	}
	served := func(d *provider.ModelData) []string {
		t.Helper()
		lo := o
		lo.HTTPClient, lo.Cache, lo.Sessions, lo.LastGood, lo.ModelData = &http.Client{Transport: builtInListing{}}, nil, nil, nil, d
		got, err := provider.List(t.Context(), "openai", fmt.Sprintf("sk-copy-%d-%d", os.Getpid(), time.Now().UnixNano()), lo)
		if err != nil || !got.Live {
			t.Fatalf("listing %+v (%v), want live", got, err)
		}
		return provider.SearchListing(got, "vision")
	}
	if got, want := served(o.ModelData), served(copyData); len(want) == 0 || !slices.Equal(got, want) {
		t.Errorf("vision over the listing's document %q, want the built-in copy's %q", got, want)
	}
	if po := e.providerOptions("openai", provider.Options{}); po.ModelData != nil {
		t.Error("the options New and the sign-in wizard are given carry a document")
	}
}
