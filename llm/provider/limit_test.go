package provider

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	sdktest "github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"

	"github.com/maccavelli/gobble-cli/llm"
)

// opencodeBody is an OpenCode gateway's error body, as go-llmprovider-sdk's
// own fixtures write it.
func opencodeBody(errType, msg string) string {
	return `{"type":"error","error":{"type":"` + errType + `","message":"` + msg + `"}}`
}

// geminiQuota is a Gemini 429 naming quotaID, with the RetryInfo delay.
func geminiQuota(quotaID, delay string) string {
	return `{"error":{"code":429,"message":"You exceeded your current quota.","status":"RESOURCE_EXHAUSTED","details":[` +
		`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{` +
		`"quotaMetric":"generativelanguage.googleapis.com/generate_content_free_tier_requests",` +
		`"quotaId":"` + quotaID + `","quotaDimensions":{"location":"global","model":"gemini-3.5-flash-lite"},"quotaValue":"5"}]},` +
		`{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"` + delay + `"}]}}`
}

// Each row of 0049-MADR D4's table, through gobble's New on a canned wire:
// the limit the provider's code or status states, with what the provider
// said on the *llm.APIError, and when the limit lifts.
func TestWireLimits(t *testing.T) {
	resets := time.Now().Add(3 * time.Hour).Truncate(time.Second)
	for _, c := range []struct {
		name, id, model string
		status          int
		header          http.Header
		body            string
		limit, code     string
		message         string
		resetIn         time.Duration // the wait the provider asked for, or zero
		resetAt         time.Time     // or the instant it named
	}{
		{"opencode-zen free tier", "opencode-zen", "claude-haiku-5-5", 429, http.Header{"Retry-After": {"5601"}},
			opencodeBody("FreeUsageLimitError", "Free usage limit reached."), llm.LimitFreeTier, "FreeUsageLimitError",
			"Free usage limit reached.", 5601 * time.Second, time.Time{}},
		{"opencode-zen monthly", "opencode-zen", "claude-haiku-5-5", 401, nil,
			opencodeBody("MonthlyLimitError", "Monthly limit reached."), llm.LimitMonthly, "MonthlyLimitError",
			"Monthly limit reached.", 0, time.Time{}},
		{"opencode-zen credits", "opencode-zen", "claude-haiku-5-5", 401, nil,
			opencodeBody("CreditsError", "No credits."), llm.LimitCredits, "CreditsError", "No credits.", 0, time.Time{}},
		{"opencode-go plan", "opencode-go", "claude-haiku-5-5", 429, nil,
			opencodeBody("GoUsageLimitError", "Usage limit reached."), llm.LimitPlan, "GoUsageLimitError",
			"Usage limit reached.", 0, time.Time{}},
		{"opencode-zen user limit, no period stated", "opencode-zen", "claude-haiku-5-5", 401, nil,
			opencodeBody("UserLimitError", "User limit."), "", "UserLimitError", "User limit.", 0, time.Time{}},
		{"openai insufficient_quota", "openai", "gpt-6-luna", 429, nil,
			`{"error":{"type":"insufficient_quota","code":"insufficient_quota","message":"You exceeded your current quota."}}`,
			llm.LimitCredits, "insufficient_quota", "You exceeded your current quota.", 0, time.Time{}},
		{"openai usage_limit_reached", "openai", "gpt-6-luna", 429, nil,
			`{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","resets_at":` +
				strconv.FormatInt(resets.Unix(), 10) + `}}`,
			llm.LimitPlan, "usage_limit_reached", "The usage limit has been reached", 0, resets},
		{"kilo promotion limit", "kilo", "deepseek/deepseek-v4.1-flash", 429, nil,
			`{"error":{"code":"PROMOTION_MODEL_LIMIT_REACHED","message":"Promotion limit reached."}}`,
			llm.LimitFreeTier, "PROMOTION_MODEL_LIMIT_REACHED", "Promotion limit reached.", 0, time.Time{}},
		{"kilo free usage in the message alone", "kilo", "deepseek/deepseek-v4.1-flash", 429, nil,
			`{"error":{"message":"FreeUsageLimitError: try later"}}`, "", "", "", 0, time.Time{}},
		{"kilo 402", "kilo", "deepseek/deepseek-v4.1-flash", 402, nil,
			`{"error":{"code":"insufficient_balance","message":"Top up."}}`, llm.LimitCredits, "insufficient_balance",
			"Top up.", 0, time.Time{}},
		{"gemini per day", "gemini", "gemini-3.5-flash-lite", 429, nil,
			geminiQuota("GenerateRequestsPerDayPerProjectPerModel-FreeTier", "120s"), llm.LimitDaily, "RESOURCE_EXHAUSTED",
			"", 120 * time.Second, time.Time{}},
		{"gemini per minute, over the cap", "gemini", "gemini-3.5-flash-lite", 429, nil,
			geminiQuota("GenerateRequestsPerMinutePerProjectPerModel-FreeTier", "120s"), llm.LimitRate, "RESOURCE_EXHAUSTED",
			"", 120 * time.Second, time.Time{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := &cannedWire{status: c.status, header: c.header, body: c.body}
			before := time.Now()
			o := wireStream(t, c.id, c.model, w, wireRequest())
			after := time.Now()
			ae, ok := errors.AsType[*llm.APIError](o.err)
			if !ok {
				t.Fatalf("err %v, want an *llm.APIError", o.err)
			}
			if ae.Limit != c.limit || ae.Provider != c.id || ae.Status != c.status || ae.Code != c.code {
				t.Errorf("limit %q provider %q status %d code %q, want %q %q %d %q", ae.Limit, ae.Provider, ae.Status, ae.Code,
					c.limit, c.id, c.status, c.code)
			}
			if c.message != "" && ae.Message != c.message {
				t.Errorf("message %q, want %q", ae.Message, c.message)
			}
			switch {
			case !c.resetAt.IsZero():
				if d := ae.ResetAt.Sub(c.resetAt); d < -time.Second || d > time.Second {
					t.Errorf("ResetAt %s, want %s, the instant the provider named", ae.ResetAt, c.resetAt)
				}
			case c.resetIn > 0:
				if ae.ResetAt.Before(before.Add(c.resetIn)) || ae.ResetAt.After(after.Add(c.resetIn)) {
					t.Errorf("ResetAt %s, want the call's time plus %s", ae.ResetAt, c.resetIn)
				}
			case !ae.ResetAt.IsZero():
				t.Errorf("ResetAt %s, want none: the provider asked for no wait", ae.ResetAt)
			}
			if ae.Retryable == nil {
				t.Errorf("Retryable not set")
			}
			oneRequest(t, w)
		})
	}
}

// A rate limit's long wait is still a rate limit: no limit is read from how
// long the wait is (0049-MADR D4).
func TestWireLongWaitIsNotAWindow(t *testing.T) {
	w := &cannedWire{status: http.StatusTooManyRequests, header: http.Header{"Retry-After": {"18000"}}, body: wireRateLimited}
	o := wireStream(t, "openai", "gpt-6-luna", w, wireRequest())
	ae, ok := errors.AsType[*llm.APIError](o.err)
	if !ok || ae.Limit != llm.LimitRate || ae.RetryAfter != 5*time.Hour || ae.Retryable == nil || !*ae.Retryable {
		t.Fatalf("err %v (%+v), want a retryable rate limit asking for 5h", o.err, ae)
	}
}

// adapterOver is an adapter of f behind the SDK's streaming retry with the
// notice hook, as New builds one.
func adapterOver(f llmprovider.Provider, policy llmprovider.RetryPolicy) *adapter {
	w := llmprovider.WithStreamRetry(f, policy, notice)
	return &adapter{p: w, id: "openai", caps: w.Capabilities()}
}

// A provider's retry carries the wait the provider asked for, apart from
// the jittered one taken, with its code and status (0049-MADR D2, D6).
func TestRetryNoticeCarriesServerWait(t *testing.T) {
	f := sdktest.NewFake("openai", toolCaps).
		Fail(&llmprovider.APIError{Provider: "openai", Status: 429, Kind: llmprovider.ErrRateLimited, Code: "rate_limit_exceeded",
			RetryAfter: time.Millisecond}).
		ReplyText("ok")
	evs, err := drain(t, adapterOver(f, llmprovider.RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond, MaxDelay: time.Second}), wireRequest())
	if err != nil {
		t.Fatal(err)
	}
	var retries []llm.ProviderRetry
	for _, ev := range evs {
		if pr, ok := ev.(llm.ProviderRetry); ok {
			retries = append(retries, pr)
		}
	}
	if len(retries) != 1 {
		t.Fatalf("%d retries, want 1: %+v", len(retries), evs)
	}
	pr := retries[0]
	if pr.RetryAfter != time.Millisecond || !pr.ServerDirected || pr.Code != "rate_limit_exceeded" || pr.Status != 429 ||
		pr.Kind != "rate-limited" || pr.Attempt != 1 || pr.MaxAttempts != 2 {
		t.Errorf("retry %+v, want RetryAfter 1ms, server-directed, rate_limit_exceeded, 429, rate-limited, 1 of 2", pr)
	}
}

// A failure after the provider's retries says how many tries were made.
func TestAttemptsAfterRetries(t *testing.T) {
	down := &llmprovider.APIError{Provider: "openai", Status: 503, Kind: llmprovider.ErrProviderUnavailable, Code: "overloaded"}
	f := sdktest.NewFake("openai", toolCaps).Fail(down).Fail(down).Fail(down).Fail(down)
	_, err := drain(t, adapterOver(f, llmprovider.RetryPolicy{MaxAttempts: 4, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}), wireRequest())
	ae, ok := errors.AsType[*llm.APIError](err)
	if !ok || ae.Attempts != 4 || len(f.Requests()) != 4 {
		t.Fatalf("err %v (%+v) after %d requests, want Attempts 4 after 4", err, ae, len(f.Requests()))
	}
	once := sdktest.NewFake("openai", toolCaps).Fail(&llmprovider.APIError{Provider: "openai", Status: 400, Kind: llmprovider.ErrInvalidRequest})
	_, err = drain(t, adapterOver(once, llmprovider.RetryPolicy{MaxAttempts: 4, BaseDelay: time.Millisecond}), wireRequest())
	if ae, ok := errors.AsType[*llm.APIError](err); !ok || ae.Attempts != 0 {
		t.Fatalf("a failure with no retry: %v (%+v), want no Attempts", err, ae)
	}
}

// For every class FailureKind names, llm.Kind names it the same
// (0049-MADR D3).
func TestKindAgreesWithFailureKind(t *testing.T) {
	for _, err := range []error{llm.ErrWorkspace, llm.ErrQuota, llm.ErrRateLimited, llm.ErrAuth, llm.ErrNotPermitted,
		llm.ErrUnavailable, context.DeadlineExceeded, context.Canceled} {
		if got, want := llm.Kind(err), FailureKind(err); got != want {
			t.Errorf("%v: llm.Kind %q, FailureKind %q", err, got, want)
		}
	}
}

// A quota sent with HTTP 401 is not led by a stored login's refusal
// (0049-MADR D8).
func TestQuotaIsNotARejectedLogin(t *testing.T) {
	err := &llmprovider.APIError{Provider: "opencode-zen/responses", Status: 401, Kind: llmprovider.ErrQuotaExhausted,
		Code: "MonthlyLimitError", Message: "monthly"}
	a := &adapter{id: "opencode-zen", login: true}
	got := a.rejected(mapError(err), err)
	if !errors.Is(got, llm.ErrQuota) || strings.Contains(got.Error(), LoginRejected("opencode-zen")) {
		t.Fatalf("%v, want llm.ErrQuota not led by the login's refusal", got)
	}
}
