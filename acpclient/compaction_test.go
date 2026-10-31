package acpclient

import (
	"encoding/json/jsontext"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// _gobble/compaction reaches its session's callback as KindCompaction,
// decoded, with its params kept; params that do not decode are logged and
// reach no callback (0005-PLAN F5-2 decision 9; 0039-MADR D12).
func TestCompactionNotification(t *testing.T) {
	var got []Update
	h := &handler{sessions: map[acp.SessionId]func(Update){"s1": func(u Update) { got = append(got, u) }}}
	send, logs := notifyPair(t, h)
	params := jsontext.Value(`{"sessionId":"s1","phase":"end","reason":"threshold","tokensBefore":1234,"error":"boom","paused":"session"}`)
	send("_gobble/compaction", params, jsontext.Value(`{"sessionId":"other","phase":"start"}`))
	want := Compaction{Phase: "end", Reason: "threshold", TokensBefore: 1234, Error: "boom", Paused: "session"}
	if len(got) != 1 || got[0].Kind != KindCompaction || got[0].Compaction != want || string(got[0].Params) != string(params) {
		t.Fatalf("updates %+v, want one KindCompaction %+v with its params", got, want)
	}
	send("_gobble/compaction", jsontext.Value(`{"sessionId":"s1","phase":"end","tokensBefore":"1234"}`))
	if len(got) != 1 || !strings.Contains(logs.String(), "failed to handle notification") || !strings.Contains(logs.String(), "_gobble/compaction") {
		t.Fatalf("bad params: updates %+v, log %q; want it logged and not delivered", got, logs)
	}
}
