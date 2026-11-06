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
