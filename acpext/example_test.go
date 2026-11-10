package acpext_test

import (
	"encoding/json"
	"fmt"

	"github.com/maccavelli/gobble-cli/acpext"
)

// A client decodes a _gobble/retry notification's params into RetryNotice.
func ExampleRetryNotice() {
	params := []byte(`{"sessionId":"s1","phase":"start","cause":"rate-limited","attempt":1,"maxAttempts":3,"delayMs":2000}`)
	var n acpext.RetryNotice
	if err := json.Unmarshal(params, &n); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(n.Phase == acpext.PhaseStart, n.Cause == acpext.RetryRateLimited, n.Attempt, n.MaxAttempts, n.DelayMs)
	// Output: true true 1 3 2000
}

// A client sends _gobble/set_session_name with SetSessionNameParams; the
// result is {}.
func ExampleSetSessionNameParams() {
	b, err := json.Marshal(acpext.SetSessionNameParams{SessionID: "s1", Name: "My name"})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(acpext.MethodSetSessionName, string(b))
	// Output: _gobble/set_session_name {"sessionId":"s1","name":"My name"}
}

// A refusal's data decodes into ErrorData.
func ExampleErrorData() {
	data := []byte(`{"reason":"a prompt is already running","sessionId":"s1"}`)
	var e acpext.ErrorData
	if err := json.Unmarshal(data, &e); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(e.SessionID, e.Reason)
	// Output: s1 a prompt is already running
}

// A failed turn's data says which limit it hit and when it lifts.
func ExampleErrorData_limit() {
	data := []byte(`{"code":"FreeUsageLimitError","kind":"quota","limit":"free_tier","provider":"opencode-zen",` +
		`"providerMessage":"Free usage limit reached.","resetAt":"2026-11-09T14:05:00Z","retryAfterMs":5601000,"retryable":false}`)
	var e acpext.ErrorData
	if err := json.Unmarshal(data, &e); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(e.Kind == acpext.KindQuota, e.Limit == acpext.LimitFreeTier, e.Provider, e.ResetAt, *e.Retryable)
	fmt.Println(e.ProviderMessage)
	// Output:
	// true true opencode-zen 2026-11-09T14:05:00Z false
	// Free usage limit reached.
}

// A plan limit whose service stated its window: Limit is weekly, and
// WindowMinutes is the length the service gave, here 7 days. A window is
// never taken from RetryAfterMs.
func ExampleErrorData_window() {
	data := []byte(`{"code":"usage_limit_reached","kind":"quota","limit":"weekly","provider":"openai",` +
		`"resetAt":"2026-11-10T21:00:00Z","retryAfterMs":7200000,"windowMinutes":10080}`)
	var e acpext.ErrorData
	if err := json.Unmarshal(data, &e); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(e.Limit == acpext.LimitWeekly, e.WindowMinutes, e.ResetAt)
	// Output:
	// true 10080 2026-11-10T21:00:00Z
}
