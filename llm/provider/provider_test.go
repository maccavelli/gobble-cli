package provider

import (
	"encoding/json/jsontext"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	sdktest "github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"

	"github.com/maccavelli/gobble-cli/llm"
)

var toolCaps = llmprovider.Capabilities{Tools: llmprovider.Supported}

func drain(t *testing.T, p llm.Provider, req *llm.Request) ([]llm.Event, error) {
	t.Helper()
	var evs []llm.Event
	for ev, err := range p.Stream(t.Context(), req) {
		if err != nil {
			return evs, err
		}
		evs = append(evs, ev)
	}
	return evs, nil
}

func text(role llm.Role, s string) llm.Message {
	return llm.Message{Role: role, Content: []llm.Content{{Type: llm.ContentText, Text: s}}}
}

func TestTextAndEstimatedUsage(t *testing.T) {
	f := sdktest.NewFake("fake", toolCaps).ReplyText("hello there")
	evs, err := drain(t, Wrap(f), &llm.Request{Messages: []llm.Message{text(llm.RoleSystem, "be brief"), text(llm.RoleUser, "hi")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 3 || evs[0] != (llm.TextDelta{Text: "hello there"}) {
		t.Fatalf("events %+v", evs)
	}
	u, ok := evs[1].(llm.Usage)
	if !ok || !u.Estimated || u.OutputTokens != 3 || u.InputTokens == 0 {
		t.Fatalf("usage %+v: an all-zero report must be estimated", evs[1])
	}
	if evs[2] != (llm.Done{StopReason: llm.StopEndTurn}) {
		t.Fatalf("done %+v", evs[2])
	}
	req := f.Requests()[0]
	if req.Instructions != "be brief" || len(req.Input) != 1 || req.Input[0] != (llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}) {
		t.Fatalf("SDK request %+v", req)
	}
}

func TestToolCallAndReportedUsage(t *testing.T) {
	f := sdktest.NewFake("fake", toolCaps).Reply(&llmprovider.Response{
		Output: []llmprovider.Item{llmprovider.FunctionCallItem{CallID: "c1", Name: "read", Arguments: `{"path":"a"}`, Signature: "sig"}},
		Usage:  llmprovider.Usage{InputTokens: 100, OutputTokens: 20, ReasoningTokens: 5, CachedTokens: 40},
	})
	req := &llm.Request{
		Messages: []llm.Message{text(llm.RoleUser, "read a")},
		Tools:    jsontext.Value(`[{"name":"read","description":"read a file","inputSchema":{"type":"object"}}]`),
	}
	evs, err := drain(t, Wrap(f), req)
	if err != nil {
		t.Fatal(err)
	}
	call, ok := evs[0].(llm.ToolCallDone)
	if !ok || call.ID != "c1" || call.Name != "read" || call.Args != `{"path":"a"}` || !strings.Contains(string(call.ProviderData), `"signature":"sig"`) {
		t.Fatalf("call %+v", evs[0])
	}
	if u := evs[1].(llm.Usage); u != (llm.Usage{InputTokens: 100, OutputTokens: 20, ReasoningTokens: 5, CacheReadTokens: 40}) {
		t.Fatalf("usage %+v", u)
	}
	if evs[2] != (llm.Done{StopReason: llm.StopToolUse}) {
		t.Fatalf("done %+v", evs[2])
	}
	tools := f.Requests()[0].Tools
	if len(tools) != 1 || tools[0].Name != "read" || tools[0].Description != "read a file" || tools[0].Schema == nil {
		t.Fatalf("SDK tools %+v", tools)
	}
}

// A history with reasoning, a call and its result maps item for item, with
// the signatures kept for replay.
func TestHistoryRoundTrip(t *testing.T) {
	f := sdktest.NewFake("fake", llmprovider.Capabilities{Tools: llmprovider.Supported, Reasoning: llmprovider.Supported}).ReplyText("ok")
	thinking := llm.Content{Type: llm.ContentThinking, Text: "hm", ProviderData: reasoningData(llmprovider.ReasoningItem{Text: "hm", Signature: "rs", Format: "messages"})}
	msgs := []llm.Message{
		text(llm.RoleUser, "go"),
		{Role: llm.RoleAssistant, Content: []llm.Content{thinking,
			{Type: llm.ContentToolCall, ToolCallID: "c1", ToolName: "read", ToolArgs: `{}`, ProviderData: signatureData("cs")}}},
		{Role: llm.RoleTool, Content: []llm.Content{{Type: llm.ContentToolResult, ToolCallID: "c1", Text: "data"}}},
	}
	if _, err := drain(t, Wrap(f), &llm.Request{Messages: msgs, Thinking: "minimal"}); err != nil {
		t.Fatal(err)
	}
	req := f.Requests()[0]
	want := []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "go"},
		llmprovider.ReasoningItem{Text: "hm", Signature: "rs", Format: "messages"},
		llmprovider.FunctionCallItem{CallID: "c1", Name: "read", Arguments: `{}`, Signature: "cs"},
		llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "data"},
	}
	if len(req.Input) != len(want) {
		t.Fatalf("input %+v", req.Input)
	}
	for i := range want {
		if req.Input[i] != want[i] {
			t.Errorf("input %d = %+v, want %+v", i, req.Input[i], want[i])
		}
	}
	if req.Reasoning == nil || req.Reasoning.Effort != llmprovider.EffortLow {
		t.Errorf("minimal thinking = %+v, want low effort", req.Reasoning)
	}
}

func TestReasoningItemCarriesData(t *testing.T) {
	f := sdktest.NewFake("fake", llmprovider.Capabilities{Reasoning: llmprovider.Supported}).Reply(&llmprovider.Response{
		Output:       []llmprovider.Item{llmprovider.ReasoningItem{Text: "think", Signature: "s"}, llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "x"}},
		FinishReason: llmprovider.FinishLength,
	})
	evs, err := drain(t, Wrap(f), &llm.Request{Messages: []llm.Message{text(llm.RoleUser, "q")}})
	if err != nil {
		t.Fatal(err)
	}
	var withData int
	for _, ev := range evs {
		if d, ok := ev.(llm.ThinkingDelta); ok && strings.Contains(string(d.ProviderData), `"signature":"s"`) {
			withData++
		}
	}
	if withData != 1 || evs[len(evs)-1] != (llm.Done{StopReason: llm.StopMaxTokens}) {
		t.Fatalf("events %+v", evs)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  *llmprovider.APIError
		want error
	}{
		{"rate limit", &llmprovider.APIError{Status: 429, Kind: llmprovider.ErrRateLimited, RetryAfter: 7 * time.Second}, llm.ErrRateLimited},
		{"quota before rate limit", &llmprovider.APIError{Status: 429, Kind: llmprovider.ErrQuotaExhausted}, llm.ErrQuota},
		{"overflow", &llmprovider.APIError{Status: 400, Kind: llmprovider.ErrContextOverflow}, llm.ErrContextOverflow},
		{"auth", &llmprovider.APIError{Status: 401, Kind: llmprovider.ErrAuthFailure}, llm.ErrAuth},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := drain(t, Wrap(sdktest.NewFake("fake", toolCaps).Fail(tc.err)), &llm.Request{Messages: []llm.Message{text(llm.RoleUser, "x")}})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want it to match %v", err, tc.want)
			}
			ae, ok := errors.AsType[*llm.APIError](err)
			if !ok || ae.Status != tc.err.Status {
				t.Fatalf("err %v is not an *llm.APIError with the status", err)
			}
			if !errors.Is(err, tc.err.Kind) {
				t.Fatalf("err %v lost the SDK's kind", err)
			}
		})
	}
	_, err := drain(t, Wrap(sdktest.NewFake("fake", toolCaps).Fail(cases[0].err)), &llm.Request{})
	if ae, _ := errors.AsType[*llm.APIError](err); ae == nil || ae.RetryAfter != 7*time.Second {
		t.Fatalf("RetryAfter lost: %v", err)
	}
}

// Context overflow is not retried; a rate limit is.
func TestRetryPolicy(t *testing.T) {
	overflow := sdktest.NewFake("fake", toolCaps).Fail(&llmprovider.APIError{Status: 400, Kind: llmprovider.ErrContextOverflow}).ReplyText("never")
	if _, err := drain(t, Wrap(llmprovider.WithRetry(overflow, retryPolicy)), &llm.Request{}); !errors.Is(err, llm.ErrContextOverflow) || len(overflow.Requests()) != 1 {
		t.Fatalf("overflow: err %v after %d requests, want one", err, len(overflow.Requests()))
	}
	limited := sdktest.NewFake("fake", toolCaps).Fail(&llmprovider.APIError{Status: 429, Kind: llmprovider.ErrRateLimited, RetryAfter: time.Millisecond}).ReplyText("ok")
	fast := llmprovider.RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}
	if _, err := drain(t, Wrap(llmprovider.WithRetry(limited, fast)), &llm.Request{}); err != nil || len(limited.Requests()) != 2 {
		t.Fatalf("rate limit: err %v after %d requests, want a retry", err, len(limited.Requests()))
	}
}

func TestReasoningLevels(t *testing.T) {
	for level, want := range map[string]llmprovider.Effort{
		"": "", "off": "", "minimal": llmprovider.EffortLow, "low": llmprovider.EffortLow, "medium": llmprovider.EffortMedium,
		"high": llmprovider.EffortHigh, "xhigh": llmprovider.EffortXHigh, "max": llmprovider.EffortXHigh,
	} {
		r := reasoningFor(level)
		if (r == nil) != (want == "") || r != nil && r.Effort != want {
			t.Errorf("reasoningFor(%q) = %+v, want %q", level, r, want)
		}
	}
}

func envOf(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestChoose(t *testing.T) {
	if _, err := Choose(envOf(nil)); !IsNoCredential(err) || !errors.Is(err, llm.ErrAuth) || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("no credential: %v", err)
	}
	s, err := Choose(envOf(map[string]string{"OPENAI_API_KEY": "k"}))
	if err != nil || s.ID != "openai" || s.EnvVar != "OPENAI_API_KEY" || s.Model == "" {
		t.Fatalf("openai only: %+v, %v", s, err)
	}
	vars := CredentialVars()
	if len(vars) < 2 {
		t.Fatalf("credential vars %q", vars)
	}
	// Two set: the earlier in registry order wins.
	s, err = Choose(envOf(map[string]string{vars[1]: "k", vars[0]: "k"}))
	if err != nil || s.EnvVar != vars[0] {
		t.Fatalf("both set: %+v, want %s", s, vars[0])
	}
}

// Ambient builds the provider without a network call.
func TestAmbientBuilds(t *testing.T) {
	p, s, err := Ambient(envOf(map[string]string{"OPENAI_API_KEY": "test-key-not-real"}), Options{ClientName: "gobble", ClientVersion: "0.0.0"})
	if err != nil || p.ID() != "openai" || s.ID != "openai" {
		t.Fatalf("ambient: %v, %+v, %v", p, s, err)
	}
}
