package acpclient

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// _gobble/retry reaches its session's callback as KindRetry, decoded, with
// its params kept; a session the client does not have gets nothing, and
// any other extension method is -32601 (0005-PLAN F5-1 decision 5).
func TestRetryNotification(t *testing.T) {
	var got []Update
	h := &handler{sessions: map[acp.SessionId]func(Update){"s1": func(u Update) { got = append(got, u) }}}
	params := json.RawMessage(`{"sessionId":"s1","phase":"start","cause":"empty_reply","attempt":1,"maxAttempts":2,"delayMs":2000}`)
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/retry", params); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/retry", json.RawMessage(`{"sessionId":"other","phase":"end","cause":"offline"}`)); err != nil {
		t.Fatal(err)
	}
	want := Retry{Phase: "start", Cause: "empty_reply", Attempt: 1, MaxAttempts: 2, Delay: 2 * time.Second}
	if len(got) != 1 || got[0].Kind != KindRetry || got[0].Retry != want || string(got[0].Params) != string(params) {
		t.Fatalf("updates %+v, want one KindRetry %+v with its params", got, want)
	}
	var re *acp.RequestError
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/other", nil); !errors.As(err, &re) || re.Code != -32601 {
		t.Fatalf("another method: %v, want -32601", err)
	}
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/retry", json.RawMessage(`{`)); !errors.As(err, &re) || re.Code != -32602 {
		t.Fatalf("bad params: %v, want -32602", err)
	}
}
