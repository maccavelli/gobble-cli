package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
)

// The failures of 0049-MADR's Confirmation 2 and 4, as llm/provider maps
// them, whose data are acpext's prompt-*.error.json frames.
func quotaErr() *llm.APIError {
	const msg = "Free usage limit reached. Free models work in OpenCode (https://opencode.ai); subscribe to OpenCode Go to keep going: https://opencode.ai/go"
	return &llm.APIError{Status: 429, RetryAfter: 5601 * time.Second,
		Err: fmt.Errorf("%w: llmprovider: quota exhausted: opencode-zen/chat_completions HTTP 429 FreeUsageLimitError (retry-after 1h33m21s): "+
			msg, llm.ErrQuota),
		Provider: "opencode-zen", Code: "FreeUsageLimitError", Message: msg, Retryable: new(false),
		// Not UTC, and not on a second: the data is both.
		ResetAt: time.Date(2026, 11, 9, 8, 5, 0, 400e6, time.FixedZone("CST", -6*60*60)), Limit: llm.LimitFreeTier}
}

func overCapErr() *llm.APIError {
	return &llm.APIError{Status: 429, RetryAfter: 2 * time.Minute,
		Err: fmt.Errorf("%w: llmprovider: rate limited: openai HTTP 429 rate_limit_exceeded (retry-after 2m0s): Rate limit reached for requests",
			llm.ErrRateLimited),
		Provider: "openai", Code: "rate_limit_exceeded", Message: "Rate limit reached for requests", Retryable: new(true),
		ResetAt: time.Date(2026, 11, 9, 12, 33, 39, 0, time.UTC), Limit: llm.LimitRate}
}

// weeklyErr is a plan limit whose service stated a 7-day window, as
// llm/provider maps it on go-llmprovider-sdk v1.29.0 (0049-MADR Q8, Q9).
func weeklyErr() *llm.APIError {
	return &llm.APIError{Status: 429, RetryAfter: 2 * time.Hour,
		Err: fmt.Errorf("%w: llmprovider: quota exhausted: openai HTTP 429 usage_limit_reached (retry-after 2h0m0s): The usage limit has been reached",
			llm.ErrQuota),
		Provider: "openai", Code: "usage_limit_reached", Message: "The usage limit has been reached", Retryable: new(false),
		ResetAt: time.Date(2026, 11, 10, 21, 0, 0, 0, time.UTC), Limit: llm.LimitWeekly, Window: 7 * 24 * time.Hour}
}

func afterRetriesErr() *llm.APIError {
	return &llm.APIError{Status: 503,
		Err: fmt.Errorf("%w: llmprovider: claude failed after 4 attempts: llmprovider: provider unavailable: claude HTTP 503 overloaded_error: Overloaded",
			llm.ErrUnavailable),
		Provider: "claude", Code: "overloaded_error", Message: "Overloaded", Retryable: new(true), Attempts: 4}
}

// encodeData is d as gobble sends it.
func encodeData(t *testing.T, d acpext.ErrorData) string {
	t.Helper()
	b, err := jsonx.Value(d)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// strictData decodes b as ErrorData, refusing a member it does not name.
func strictData(t *testing.T, what string, b []byte) acpext.ErrorData {
	t.Helper()
	var d acpext.ErrorData
	if err := jsonx.UnmarshalStrict(b, &d, json.RejectUnknownMembers(true)); err != nil {
		t.Fatalf("%s: %v in %s", what, err, b)
	}
	return d
}

// readFrame is an acpext frame's bytes.
func readFrame(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "acpext", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// frameData is an acpext frame's error data in its canonical encoding.
func frameData(t *testing.T, name string) string {
	t.Helper()
	return encodeData(t, strictData(t, name, readFrame(t, name)))
}

// canonicalNotice is b, a _gobble/retry notice, decoded refusing a member
// it does not name and encoded as gobble sends it, its session id id when
// id is not empty.
func canonicalNotice(t *testing.T, what string, b []byte, id acp.SessionId) string {
	t.Helper()
	var n acpext.RetryNotice
	if err := jsonx.UnmarshalStrict(b, &n, json.RejectUnknownMembers(true)); err != nil {
		t.Fatalf("%s: %v in %s", what, err, b)
	}
	if id != "" {
		n.SessionID = string(id)
	}
	v, err := jsonx.Value(n)
	if err != nil {
		t.Fatal(err)
	}
	return string(v)
}

// wireData is a JSON-RPC error's code and its data, as the client received
// it, in its canonical encoding.
func wireData(t *testing.T, err error) (int, string) {
	t.Helper()
	re, ok := errors.AsType[*acp.RequestError](err)
	if !ok {
		t.Fatalf("err = %v, want a JSON-RPC error", err)
	}
	b, merr := jsonx.Marshal(re.Data)
	if merr != nil {
		t.Fatal(merr)
	}
	return re.Code, encodeData(t, strictData(t, "the error's data", b))
}

// limitAgent is a connected agent whose run's provider is named provider and
// is p, with a small KeepRecentTokens, so a session can be compacted.
func limitAgent(t *testing.T, provider string, p llm.Provider) *acptest.Conn {
	t.Helper()
	a := New(Options{Version: "1.2.3", NewID: counterIDs(), Store: session.NewMemoryStore(),
		Models:     func() ModelChoice { return ModelChoice{Provider: provider, Models: []string{"m1"}, Default: "m1"} },
		Provider:   func(string, llm.Retry) (llm.Provider, error) { return p, nil },
		Compaction: compaction.Settings{ReserveTokens: 16384, KeepRecentTokens: 150}})
	c := acptest.Connect(t, a, &acptest.Client{})
	a.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c
}

// A failed turn's data names its provider and kind, and what the service
// said when an *llm.APIError is in the chain; a refusal for want of a
// workspace keeps the service's words out (0049-MADR D7; 0035-MADR D5).
func TestErrorData(t *testing.T) {
	attemptOnce := afterRetriesErr()
	attemptOnce.Attempts = 1
	workspace := &llm.APIError{Status: 403, Err: fmt.Errorf("%w: secret-ish", llm.ErrWorkspace), Provider: "claude",
		Code: "permission_error", Message: "secret-ish", Retryable: new(false)}
	empty := fmt.Errorf("the provider gave an empty reply: %w", llm.ErrEmptyReply)
	auth := fmt.Errorf("%w: key refused", llm.ErrAuth)
	for _, c := range []struct {
		name, provider string
		err            error
		want           acpext.ErrorData
		code           int
	}{
		{"quota", "opencode-zen", quotaErr(), acpext.ErrorData{Error: quotaErr().Error(), Provider: "opencode-zen",
			Kind: acpext.KindQuota, HTTPStatus: 429, Code: "FreeUsageLimitError", ProviderMessage: quotaErr().Message,
			RetryAfterMs: 5601000, ResetAt: "2026-11-09T14:05:00Z", Retryable: new(false), Limit: acpext.LimitFreeTier}, -32603},
		{"rate limit over the cap", "openai", overCapErr(), acpext.ErrorData{Error: overCapErr().Error(), Provider: "openai",
			Kind: acpext.KindRateLimited, HTTPStatus: 429, Code: "rate_limit_exceeded", ProviderMessage: "Rate limit reached for requests",
			RetryAfterMs: 120000, ResetAt: "2026-11-09T12:33:39Z", Retryable: new(true), Limit: acpext.LimitRate}, -32603},
		{"after retries", "claude", afterRetriesErr(), acpext.ErrorData{Error: afterRetriesErr().Error(), Provider: "claude",
			Kind: acpext.KindUnavailable, HTTPStatus: 503, Code: "overloaded_error", ProviderMessage: "Overloaded",
			Retryable: new(true), Attempts: 4}, -32603},
		{"a stated weekly window", "openai", weeklyErr(), acpext.ErrorData{Error: weeklyErr().Error(), Provider: "openai",
			Kind: acpext.KindQuota, HTTPStatus: 429, Code: "usage_limit_reached", ProviderMessage: "The usage limit has been reached",
			RetryAfterMs: 7200000, ResetAt: "2026-11-10T21:00:00Z", Retryable: new(false), Limit: acpext.LimitWeekly,
			WindowMinutes: 10080}, -32603},
		{"one attempt", "claude", attemptOnce, acpext.ErrorData{Error: attemptOnce.Error(), Provider: "claude",
			Kind: acpext.KindUnavailable, HTTPStatus: 503, Code: "overloaded_error", ProviderMessage: "Overloaded",
			Retryable: new(true)}, -32603},
		{"offline", "openai", errOffline, acpext.ErrorData{Error: errOffline.Error(), Provider: "openai", Kind: acpext.KindOffline}, -32603},
		{"empty reply", "openai", empty, acpext.ErrorData{Error: empty.Error(), Provider: "openai", Kind: acpext.KindEmptyReply}, -32603},
		{"workspace", "claude", workspace, acpext.ErrorData{Error: workspaceAdvice, Provider: "claude", Kind: acpext.KindWorkspace}, -32603},
		{"auth", "grok", auth, acpext.ErrorData{Error: auth.Error(), Provider: "grok", Kind: acpext.KindAuth}, -32000},
		{"other", "fake", errors.New("boom"), acpext.ErrorData{Error: "boom", Provider: "fake", Kind: acpext.KindOther}, -32603},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got, want := encodeData(t, errorData(c.provider, c.err)), encodeData(t, c.want); got != want {
				t.Fatalf("errorData\n got %s\nwant %s", got, want)
			}
			code, got := wireData(t, promptError(c.provider, c.err))
			if want := encodeData(t, c.want); code != c.code || got != want {
				t.Fatalf("promptError %d %s, want %d %s", code, got, c.code, want)
			}
		})
	}
}

// A prompt failing on each of 0049-MADR's limits answers -32603 whose data
// is, byte for byte, the acpext frame of that failure (Confirmation 2, 4).
func TestPromptQuotaData(t *testing.T) {
	for _, c := range []struct {
		frame, provider string
		err             error
	}{
		{"prompt-quota.error.json", "opencode-zen", quotaErr()},
		{"prompt-rate-over-cap.error.json", "openai", overCapErr()},
		{"prompt-after-retries.error.json", "claude", afterRetriesErr()},
		{"prompt-weekly.error.json", "openai", weeklyErr()},
	} {
		t.Run(c.frame, func(t *testing.T) {
			conn := limitAgent(t, c.provider, llmtest.NewScript(llmtest.Turn{Err: c.err}))
			id := newSession(t, conn, t.TempDir(), nil).SessionId
			_, err := prompt(t, conn, id, "go")
			if code, got := wireData(t, err); code != -32603 || got != frameData(t, c.frame) {
				t.Fatalf("prompt error %d %s, want -32603 %s", code, got, frameData(t, c.frame))
			}
		})
	}
}

// llm's limits are acpext's, and every kind llm.Kind gives is one of
// acpext's (0049-MADR D3, D4).
func TestKindsAndLimitsMatchAcpext(t *testing.T) {
	for _, p := range []struct {
		llm    string
		acpext acpext.LimitScope
	}{
		{llm.LimitRate, acpext.LimitRate}, {llm.LimitWindow, acpext.LimitWindow}, {llm.LimitDaily, acpext.LimitDaily},
		{llm.LimitWeekly, acpext.LimitWeekly}, {llm.LimitMonthly, acpext.LimitMonthly}, {llm.LimitCredits, acpext.LimitCredits},
		{llm.LimitFreeTier, acpext.LimitFreeTier}, {llm.LimitPlan, acpext.LimitPlan},
	} {
		if p.llm != string(p.acpext) {
			t.Errorf("llm limit %q, acpext %q", p.llm, p.acpext)
		}
	}
	kinds := []acpext.ErrorKind{acpext.KindWorkspace, acpext.KindQuota, acpext.KindRateLimited, acpext.KindContextOverflow,
		acpext.KindEmptyReply, acpext.KindIncomplete, acpext.KindAuth, acpext.KindNotPermitted, acpext.KindOffline,
		acpext.KindUnavailable, acpext.KindUnsupported, acpext.KindTimeout, acpext.KindCanceled, acpext.KindOther}
	for _, err := range []error{llm.ErrWorkspace, llm.ErrQuota, llm.ErrRateLimited, llm.ErrContextOverflow, llm.ErrEmptyReply,
		llm.ErrIncomplete, llm.ErrAuth, llm.ErrNotPermitted, llm.ErrOffline, llm.ErrUnavailable, llm.ErrUnsupported,
		context.DeadlineExceeded, context.Canceled, errors.New("plain")} {
		if k := acpext.ErrorKind(llm.Kind(err)); !slices.Contains(kinds, k) {
			t.Errorf("llm.Kind(%v) = %q, not an acpext ErrorKind", err, k)
		}
	}
}
