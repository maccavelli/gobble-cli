package acpclient

import (
	"encoding/json/jsontext"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// _gobble/personality reaches its session's callback as KindPersonality,
// decoded, with its params kept; while a session/new is in flight, a
// notice for a session not yet known goes to the session being created,
// which then keeps it; params that do not decode are logged and reach no
// callback (0022-PLAN P3 step 7; 0039-MADR D12).
func TestPersonalityNotification(t *testing.T) {
	var got, pending []Update
	h := &handler{sessions: map[acp.SessionId]func(Update){"s1": func(u Update) { got = append(got, u) }}}
	send, logs := notifyPair(t, h)
	params := jsontext.Value(`{"sessionId":"s1","name":"snarky","level":4}`)
	send("_gobble/personality", params)
	if len(got) != 1 || got[0].Kind != KindPersonality || got[0].Personality != (Personality{Name: "snarky", Level: 4}) ||
		string(got[0].Params) != string(params) || got[0].Kind.String() != "personality" {
		t.Fatalf("updates %+v, want one KindPersonality snarky 4 with its params", got)
	}
	h.setPending(func(u Update) { pending = append(pending, u) })
	send("_gobble/personality", jsontext.Value(`{"sessionId":"s2","name":"dry","level":3}`))
	h.setPending(nil)
	send("_gobble/personality", jsontext.Value(`{"sessionId":"s2","name":"boring","level":1}`))
	if len(pending) != 2 || pending[0].Personality.Name != "dry" || pending[1].Personality.Name != "boring" {
		t.Fatalf("the new session's updates %+v", pending)
	}
	send("_gobble/personality", jsontext.Value(`{"sessionId":"s1","name":"dry","level":"3"}`))
	if len(got) != 1 || !strings.Contains(logs.String(), "failed to handle notification") || !strings.Contains(logs.String(), "_gobble/personality") {
		t.Fatalf("bad params: updates %+v, log %q; want it logged and not delivered", got, logs)
	}
}
