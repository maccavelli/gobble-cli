package acpclient_test

import (
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// The rows of a compaction the agent reports through session.compaction:
// one manual compaction whose summary arrives in chunks, then one automatic
// compaction that fails and pauses, then one cancelled.
var standardCompaction = []string{
	`{"sessionUpdate":"compaction_update","compactionId":"c1","status":"in_progress","_meta":{"gobble":{"reason":"manual"}}}`,
	`{"sessionUpdate":"compaction_summary_chunk","compactionId":"c1","content":{"type":"text","text":"## Goal\n"}}`,
	`{"sessionUpdate":"compaction_summary_chunk","compactionId":"c1","content":{"type":"text","text":"SUMMARY"}}`,
	`{"sessionUpdate":"compaction_update","compactionId":"c1","status":"completed","_meta":{"gobble":{"reason":"manual","tokensBefore":1234}}}`,
	`{"sessionUpdate":"compaction_update","compactionId":"c2","status":"in_progress","_meta":{"gobble":{"reason":"threshold"}}}`,
	`{"sessionUpdate":"compaction_update","compactionId":"c2","status":"failed","error":"boom","_meta":{"gobble":{"reason":"threshold","paused":"session"}}}`,
	`{"sessionUpdate":"compaction_update","compactionId":"c3","status":"in_progress","_meta":{"gobble":{"reason":"overflow"}}}`,
	`{"sessionUpdate":"compaction_update","compactionId":"c3","status":"cancelled","_meta":{"gobble":{"reason":"overflow"}}}`,
}

// With Options.Compaction, initialize offers session.compaction and not
// session.notices, and compaction_update and compaction_summary_chunk fold,
// through the SDK's CompactionTimeline, into KindCompaction: the status, the
// summary so far, the error, and _meta.gobble's reason, tokens and pause; an
// update is a start or an end, as _gobble/compaction's are, and a chunk is
// neither. Without the option, nothing is offered (0016-MADR D10).
func TestTranslateStandardCompaction(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: standardCompaction}}}
	c := start(t, agent, acpclient.Options{Name: "test", Version: "0.0.0", Compaction: true})
	var col collector
	s, err := c.NewSession(t.Context(), acptest.Abs("/w"), col.on)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "go"}); err != nil {
		t.Fatal(err)
	}
	caps := agent.ClientCapabilities()
	if len(caps) != 1 || caps[0].Session == nil || caps[0].Session.Compaction == nil || caps[0].Session.Notices != nil {
		t.Fatalf("client capabilities %+v, want session.compaction alone", caps)
	}
	want := []acpclient.Compaction{
		{ID: "c1", Status: "in_progress", Phase: "start", Reason: "manual"},
		{ID: "c1", Status: "in_progress", Reason: "manual", Summary: "## Goal\n"},
		{ID: "c1", Status: "in_progress", Reason: "manual", Summary: "## Goal\nSUMMARY"},
		{ID: "c1", Status: "completed", Phase: "end", Reason: "manual", TokensBefore: 1234, Summary: "## Goal\nSUMMARY"},
		{ID: "c2", Status: "in_progress", Phase: "start", Reason: "threshold"},
		{ID: "c2", Status: "failed", Phase: "end", Reason: "threshold", Error: "boom", Paused: "session"},
		{ID: "c3", Status: "in_progress", Phase: "start", Reason: "overflow"},
		{ID: "c3", Status: "cancelled", Phase: "end", Reason: "overflow", Error: "cancelled"},
	}
	got := col.all()
	if len(got) != len(want) {
		t.Fatalf("%d updates %+v, want %d", len(got), got, len(want))
	}
	for i, w := range want {
		if got[i].Kind != acpclient.KindCompaction || got[i].Compaction != w {
			t.Errorf("update %d: %v %+v, want compaction %+v", i, got[i].Kind, got[i].Compaction, w)
		}
	}

	plain := &acptest.ScriptAgent{}
	start(t, plain, acpclient.Options{Name: "test", Version: "0.0.0"})
	if caps := plain.ClientCapabilities(); len(caps) != 1 || caps[0].Session != nil {
		t.Fatalf("client capabilities without the option %+v, want none", caps)
	}
}
