package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/internal/config"
)

// compactionUpdates are compactions as the client gets them: a manual one,
// a threshold one that succeeds, and two that fail, pausing the trigger
// for the turn and for the session.
func compactionUpdates() []acpclient.Update {
	u := func(c acpclient.Compaction) acpclient.Update {
		return acpclient.Update{Kind: acpclient.KindCompaction, Compaction: c, Params: []byte(`{"phase":"` + c.Phase + `"}`)}
	}
	return []acpclient.Update{
		u(acpclient.Compaction{Phase: "start", Reason: "manual"}),
		u(acpclient.Compaction{Phase: "end", Reason: "manual", Error: "Nothing to compact"}),
		u(acpclient.Compaction{Phase: "start", Reason: "threshold"}),
		u(acpclient.Compaction{Phase: "end", Reason: "threshold", TokensBefore: 9}),
		u(acpclient.Compaction{Phase: "start", Reason: "overflow"}),
		u(acpclient.Compaction{Phase: "end", Reason: "overflow", Error: "offline", Paused: "turn"}),
		u(acpclient.Compaction{Phase: "end", Reason: "threshold", Error: "too short", Paused: "session"}),
	}
}

const compactionStderr = "compacting the conversation (threshold)\ncompacting the conversation (overflow)\n" +
	"compaction failed: offline (automatic compaction paused for this turn)\n" +
	"compaction failed: too short (automatic compaction paused until a compaction succeeds or the model changes)\n"

// The CLI shows an automatic compaction's start and a failed end on
// stderr, and nothing for a manual one or on stdout: the line view, print
// mode, and stream-json (0005-PLAN F5-2 decision 9).
func TestCompactionLines(t *testing.T) {
	var out, errw bytes.Buffer
	v := newView(&Output{out: &out, err: &errw}, display{}, stepClock())
	for _, u := range compactionUpdates() {
		v.on(u)
	}
	if out.Len() != 0 || errw.String() != compactionStderr {
		t.Fatalf("view: stdout %q, stderr %q; want nothing and %q", out.String(), errw.String(), compactionStderr)
	}
	for _, stream := range []bool{false, true} {
		var out, errw bytes.Buffer
		s := &printSink{out: &Output{out: &out, err: &errw}, stream: stream, now: time.Now}
		s.begin("s1", "/w")
		header := out.String()
		for _, u := range compactionUpdates() {
			s.on(u)
		}
		if out.String() != header || errw.String() != compactionStderr {
			t.Fatalf("print (stream %v): stdout %q after its header %q, stderr %q; want nothing more and %q", stream, out.String(), header,
				errw.String(), compactionStderr)
		}
	}
}

// The compaction keys reach a session's settings: enabled false turns
// automatic compaction off, and the reserve and thresholds are the
// settings' (0005-PLAN F5-2 decision 8).
func TestCompactionSettingsReachSession(t *testing.T) {
	s := config.Defaults()
	if got := toSession(s); got.AutoCompactionOff || got.Compaction != compaction.DefaultSettings {
		t.Fatalf("default: off %v, %+v; want on, %+v", got.AutoCompactionOff, got.Compaction, compaction.DefaultSettings)
	}
	s.Compaction = config.Compaction{Enabled: false, ReserveTokens: 7, KeepRecentTokens: 8, ThresholdPercent: 9, ThresholdTokens: 10}
	want := compaction.Settings{ReserveTokens: 7, KeepRecentTokens: 8, ThresholdPercent: 9, ThresholdTokens: 10}
	if got := toSession(s); !got.AutoCompactionOff || got.Compaction != want {
		t.Fatalf("off %v, %+v; want off, %+v", got.AutoCompactionOff, got.Compaction, want)
	}
}
