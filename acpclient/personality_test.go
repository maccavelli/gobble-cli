package acpclient

import (
	"encoding/json"
	"errors"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// _gobble/personality reaches its session's callback as KindPersonality,
// decoded, with its params kept; while a session/new is in flight, a
// notice for a session not yet known goes to the session being created,
// which then keeps it; bad params are -32602 (0022-PLAN P3 step 7).
func TestPersonalityNotification(t *testing.T) {
	var got, pending []Update
	h := &handler{sessions: map[acp.SessionId]func(Update){"s1": func(u Update) { got = append(got, u) }}}
	params := json.RawMessage(`{"sessionId":"s1","name":"snarky","level":4}`)
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/personality", params); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kind != KindPersonality || got[0].Personality != (Personality{Name: "snarky", Level: 4}) ||
		string(got[0].Params) != string(params) || got[0].Kind.String() != "personality" {
		t.Fatalf("updates %+v, want one KindPersonality snarky 4 with its params", got)
	}
	h.setPending(func(u Update) { pending = append(pending, u) })
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/personality", json.RawMessage(`{"sessionId":"s2","name":"dry","level":3}`)); err != nil {
		t.Fatal(err)
	}
	h.setPending(nil)
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/personality", json.RawMessage(`{"sessionId":"s2","name":"boring","level":1}`)); err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 || pending[0].Personality.Name != "dry" || pending[1].Personality.Name != "boring" {
		t.Fatalf("the new session's updates %+v", pending)
	}
	var re *acp.RequestError
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/personality", json.RawMessage(`{`)); !errors.As(err, &re) || re.Code != -32602 {
		t.Fatalf("bad params: %v, want -32602", err)
	}
}
