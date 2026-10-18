package acpclient

import (
	"encoding/json"
	"errors"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// _gobble/compaction reaches its session's callback as KindCompaction,
// decoded, with its params kept; bad params are -32602 (0005-PLAN F5-2
// decision 9).
func TestCompactionNotification(t *testing.T) {
	var got []Update
	h := &handler{sessions: map[acp.SessionId]func(Update){"s1": func(u Update) { got = append(got, u) }}}
	params := json.RawMessage(`{"sessionId":"s1","phase":"end","reason":"threshold","tokensBefore":1234,"error":"boom","paused":"session"}`)
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/compaction", params); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/compaction", json.RawMessage(`{"sessionId":"other","phase":"start"}`)); err != nil {
		t.Fatal(err)
	}
	want := Compaction{Phase: "end", Reason: "threshold", TokensBefore: 1234, Error: "boom", Paused: "session"}
	if len(got) != 1 || got[0].Kind != KindCompaction || got[0].Compaction != want || string(got[0].Params) != string(params) {
		t.Fatalf("updates %+v, want one KindCompaction %+v with its params", got, want)
	}
	var re *acp.RequestError
	if _, err := h.HandleExtensionMethod(t.Context(), "_gobble/compaction", json.RawMessage(`{`)); !errors.As(err, &re) || re.Code != -32602 {
		t.Fatalf("bad params: %v, want -32602", err)
	}
}
