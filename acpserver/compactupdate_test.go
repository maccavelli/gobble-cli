package acpserver

import (
	"context"
	"encoding/json/v2"
	"iter"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
	sdkacptest "github.com/maccavelli/acp-go-sdk2/acptest"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// The capabilities of a client that offers session.compaction, and of one
// that offers session.notices as well (0016-MADR D9).
var (
	offersCompaction = acp.ClientCapabilities{Session: &acp.ClientSessionCapabilities{Compaction: &acp.CompactionCapabilities{}}}
	offersBoth       = acp.ClientCapabilities{Session: &acp.ClientSessionCapabilities{Compaction: &acp.CompactionCapabilities{},
		Notices: &acp.NoticeCapabilities{}}}
)

// row is one compaction_update as a test reads it.
type row struct {
	id      acp.CompactionId
	status  acp.CompactionStatus
	summary []acp.ContentBlock
	err     string
	meta    acpext.CompactionMeta
}

// rowsOf are the compaction_updates among ns, in order. A
// compaction_summary_chunk, an update without _meta, or a _meta with any
// member but gobble's fails the test: the summary travels in the completed
// update, and each update carries all of gobble's facts (0016-MADR P3-Q2,
// P3-Q3).
func rowsOf(t *testing.T, ns []acp.SessionNotification) []row {
	t.Helper()
	var out []row
	for _, n := range ns {
		if n.Update.CompactionSummaryChunk != nil {
			t.Errorf("a compaction_summary_chunk was sent; the summary belongs in the completed update")
		}
		u := n.Update.CompactionUpdate
		if u == nil {
			continue
		}
		r := row{id: u.CompactionId, status: u.Status}
		r.summary, _ = u.Summary.Value()
		r.err, _ = u.Error.Value()
		m, ok := u.Meta.Value()
		if !ok || len(m) != 1 {
			t.Errorf("compaction_update %s: _meta %v, want gobble's member alone", u.Status, m)
		}
		meta, found, err := acp.DecodeMeta[acpext.CompactionMeta](m, acpext.MetaKey)
		if err != nil || !found {
			t.Errorf("compaction_update %s: _meta.gobble %v, %v", u.Status, found, err)
		}
		r.meta = meta
		out = append(out, r)
	}
	return out
}

// checkRows compares got with want, status, error and _meta in order, a
// TokensBefore of -1 in want matching any count above 0, and returns the
// one id they share. A completed row carries its summary as one text
// block; no other row carries one.
func checkRows(t *testing.T, got, want []row) acp.CompactionId {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%d compaction_updates %+v, want %d", len(got), got, len(want))
	}
	for i, g := range got {
		w := want[i]
		if w.meta.TokensBefore == -1 && g.meta.TokensBefore > 0 {
			w.meta.TokensBefore = g.meta.TokensBefore
		}
		if g.status != w.status || g.err != w.err || g.meta != w.meta {
			t.Errorf("compaction_update %d: %s %q %+v, want %s %q %+v", i, g.status, g.err, g.meta, w.status, w.err, w.meta)
		}
		if (g.status == acp.CompactionStatusCompleted) != (len(g.summary) == 1 && g.summary[0].Text != nil) {
			t.Errorf("compaction_update %d, %s: summary %+v", i, g.status, g.summary)
		}
		if g.id != got[0].id || g.id == "" {
			t.Errorf("compaction_update %d: id %q, want the first's %q", i, g.id, got[0].id)
		}
	}
	return got[0].id
}

// summaryOf is a completed row's summary text.
func summaryOf(r row) string {
	if len(r.summary) != 1 || r.summary[0].Text == nil {
		return ""
	}
	return r.summary[0].Text.Text
}

// compactionEntry is the session's last compaction entry, by
// _gobble/get_entries: its id, summary and tokensBefore, or "" when there
// is none.
func (h *compactHarness) compactionEntry(id acp.SessionId) (entryID, summary string, before float64) {
	h.t.Helper()
	for _, e := range mustExt(h.t, h.c, "_gobble/get_entries", map[string]any{"sessionId": id})["entries"].([]any) {
		m := e.(map[string]any)
		if m["type"] == "compaction" {
			entryID, _ = m["id"].(string)
			summary, _ = m["summary"].(string)
			before, _ = m["tokensBefore"].(float64)
		}
	}
	return entryID, summary, before
}

// checkCompleted checks that the completed row r names the session's last
// compaction entry and carries its summary.
func (h *compactHarness) checkCompleted(id acp.SessionId, r row) {
	h.t.Helper()
	entryID, summary, _ := h.compactionEntry(id)
	if string(r.id) != entryID || summaryOf(r) != summary || summary == "" {
		h.t.Fatalf("completed %q with summary %q; the entry is %q with summary %q", r.id, summaryOf(r), entryID, summary)
	}
}

// sentNotices are the notice updates among ns.
func sentNotices(ns []acp.SessionNotification) []*acp.SessionUpdateNotice {
	var out []*acp.SessionUpdateNotice
	for _, n := range ns {
		if n.Update.Notice != nil {
			out = append(out, n.Update.Notice)
		}
	}
	return out
}

// noLegacy fails the test when a _gobble/compaction notification reached
// the client, which offered the standard update.
func (h *compactHarness) noLegacy() {
	h.t.Helper()
	if got := h.notices(); len(got) != 0 {
		h.t.Fatalf("_gobble/compaction sent to a client that offered session.compaction: %v", got)
	}
}

// conforms checks every frame on the wire against the protocol.
func (h *compactHarness) conforms() {
	h.t.Helper()
	sdkacptest.ValidateGolden(h.t, h.c.Recorder.Records())
}

func lastIsUsage(t *testing.T, ns []acp.SessionNotification) {
	t.Helper()
	if len(ns) == 0 || ns[len(ns)-1].Update.UsageUpdate == nil {
		t.Fatal("the compaction did not end with a usage_update")
	}
}

func manualRow(status acp.CompactionStatus, before int64) row {
	return row{status: status, meta: acpext.CompactionMeta{Reason: acpext.CompactionManual, TokensBefore: before}}
}

// /compact, to a client that offers session.compaction, is one
// compaction_update in_progress then completed, its id the entry's and its
// summary the entry's, then usage_update, with no _gobble/compaction and no
// compaction text; a refusal is its one-line reply and no update; a session
// load replays the compaction as one completed update with the same id
// (0016-MADR D9, P3-Q1 to P3-Q4).
func TestCompactionUpdateManual(t *testing.T) {
	h := newCompactHarnessWith(t, harnessConfig{caps: offersCompaction}, SessionSettings{}, full("r1", 10), full("r2", 10), llmtest.Text("SUMMARY"))
	id := h.session()
	n := len(h.client.Updates())
	h.mustPrompt(id, "/compact")
	refused := h.client.Updates()[n:]
	if got := rowsOf(t, refused); len(got) != 0 || !strings.Contains(agentText(refused), "Nothing to compact") {
		t.Fatalf("refused /compact: %d updates, text %q; want its reply alone", len(got), agentText(refused))
	}
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2)
	n = len(h.client.Updates())
	h.mustPrompt(id, "/compact")
	after := h.client.Updates()[n:]
	got := rowsOf(t, after)
	live := checkRows(t, got, []row{manualRow(acp.CompactionStatusInProgress, 0), manualRow(acp.CompactionStatusCompleted, -1)})
	h.checkCompleted(id, got[1])
	lastIsUsage(t, after)
	if text := agentText(after); text != "" {
		t.Fatalf("compaction text %q sent with the update", text)
	}
	h.noLegacy()
	n = len(h.client.Updates())
	h.mustPrompt(id, "/compact")
	if again := h.client.Updates()[n:]; len(rowsOf(t, again)) != 0 || !strings.Contains(agentText(again), "Already compacted") {
		t.Fatalf("second /compact: text %q", agentText(again))
	}

	if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	n = len(h.client.Updates())
	if _, err := h.c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: h.cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	replay := h.client.Updates()[n:]
	_, _, before := h.compactionEntry(id)
	replayed := rowsOf(t, replay)
	if checkRows(t, replayed, []row{{status: acp.CompactionStatusCompleted, meta: acpext.CompactionMeta{TokensBefore: int64(before)}}}) != live {
		t.Fatalf("replayed id %q, live id %q", replayed[0].id, live)
	}
	h.checkCompleted(id, replayed[0])
	if strings.Contains(agentText(replay), "Compacted from") {
		t.Fatalf("replay sent the compaction as text too: %q", agentText(replay))
	}
	h.conforms()
}

// A /compact whose summary fails is one failed update with the error's
// first line, and no compaction text.
func TestCompactionUpdateManualFailed(t *testing.T) {
	h := newCompactHarnessWith(t, harnessConfig{caps: offersCompaction}, SessionSettings{}, full("r1", 10), full("r2", 10), llmtest.Turn{Err: llm.ErrOffline})
	id := h.session()
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2)
	n := len(h.client.Updates())
	h.mustPrompt(id, "/compact")
	after := h.client.Updates()[n:]
	failed := manualRow(acp.CompactionStatusFailed, 0)
	failed.err = "Summarization failed: llm: offline"
	checkRows(t, rowsOf(t, after), []row{manualRow(acp.CompactionStatusInProgress, 0), failed})
	if text := agentText(after); text != "" {
		t.Fatalf("compaction text %q sent with the failed update", text)
	}
	if entryID, _, _ := h.compactionEntry(id); entryID != "" {
		t.Fatalf("failed compaction wrote entry %q", entryID)
	}
	h.noLegacy()
	h.conforms()
}

// _gobble/compact, to a client that offers session.compaction, sends the
// same updates and returns its result as before; a refusal is -32600 and
// no update.
func TestCompactionUpdateExtension(t *testing.T) {
	h := newCompactHarnessWith(t, harnessConfig{caps: offersCompaction}, SessionSettings{}, full("r1", 10), full("r2", 10), llmtest.Text("SUMMARY"))
	id := h.session()
	n := len(h.client.Updates())
	_, err := ext(t, h.c, "_gobble/compact", map[string]any{"sessionId": id})
	if code, text := errText(t, err); code != -32600 || text != "Nothing to compact (session too small)" || len(rowsOf(t, h.client.Updates()[n:])) != 0 {
		t.Fatalf("refused _gobble/compact: %d %q", code, text)
	}
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2)
	n = len(h.client.Updates())
	res := mustExt(t, h.c, "_gobble/compact", map[string]any{"sessionId": id})
	after := h.client.Updates()[n:]
	got := rowsOf(t, after)
	checkRows(t, got, []row{manualRow(acp.CompactionStatusInProgress, 0), manualRow(acp.CompactionStatusCompleted, -1)})
	h.checkCompleted(id, got[1])
	if res["summary"] != summaryOf(got[1]) || res["tokensBefore"].(float64) != float64(got[1].meta.TokensBefore) {
		t.Fatalf("result %v, update %+v", res, got[1])
	}
	lastIsUsage(t, after)
	h.noLegacy()
	h.conforms()
}

// An automatic compaction at the threshold, and one on overflow, are the
// same pair with their reason (0016-MADR D9, P3-Q3).
func TestCompactionUpdateAutomatic(t *testing.T) {
	overflow := llmtest.Turn{Err: llm.ErrContextOverflow}
	for _, c := range []struct {
		reason acpext.CompactionReason
		turns  []llmtest.Turn
	}{
		{acpext.CompactionThreshold, []llmtest.Turn{full("r1", 110000), llmtest.Text("SUMMARY"), llmtest.Text("after")}},
		{acpext.CompactionOverflow, []llmtest.Turn{full("r1", 10), overflow, llmtest.Text("SUMMARY"), llmtest.Text("ok")}},
	} {
		t.Run(string(c.reason), func(t *testing.T) {
			h := newCompactHarnessWith(t, harnessConfig{caps: offersBoth}, SessionSettings{}, c.turns...)
			id := h.session()
			h.mustPrompt(id, p1)
			n := len(h.client.Updates())
			h.mustPrompt(id, p2)
			after := h.client.Updates()[n:]
			got := rowsOf(t, after)
			checkRows(t, got, []row{
				{status: acp.CompactionStatusInProgress, meta: acpext.CompactionMeta{Reason: c.reason}},
				{status: acp.CompactionStatusCompleted, meta: acpext.CompactionMeta{Reason: c.reason, TokensBefore: -1}},
			})
			h.checkCompleted(id, got[1])
			if len(sentNotices(after)) != 0 {
				t.Fatalf("notices %v after a compaction that succeeded", sentNotices(after))
			}
			h.noLegacy()
			h.conforms()
		})
	}
}

// An automatic compaction that fails and pauses the trigger is a failed
// update with the pause, then, when the client offered session.notices,
// one warning saying so; with session.compaction alone, no notice
// (0016-MADR D9, P3-Q5).
func TestCompactionUpdatePaused(t *testing.T) {
	for _, c := range []struct {
		name    string
		caps    acp.ClientCapabilities
		notices int
	}{{"with notices", offersBoth, 1}, {"without notices", offersCompaction, 0}} {
		t.Run(c.name, func(t *testing.T) {
			h := newCompactHarnessWith(t, harnessConfig{caps: c.caps}, SessionSettings{}, full("r1", 110000), llmtest.Turn{Err: llm.ErrOffline}, full("ok", 10))
			id := h.session()
			h.mustPrompt(id, p1)
			n := len(h.client.Updates())
			h.mustPrompt(id, p2)
			after := h.client.Updates()[n:]
			checkRows(t, rowsOf(t, after), []row{
				{status: acp.CompactionStatusInProgress, meta: acpext.CompactionMeta{Reason: acpext.CompactionThreshold}},
				{status: acp.CompactionStatusFailed, err: "Summarization failed: llm: offline",
					meta: acpext.CompactionMeta{Reason: acpext.CompactionThreshold, Paused: acpext.PauseTurn}},
			})
			got := sentNotices(after)
			if len(got) != c.notices {
				t.Fatalf("%d notices %v, want %d", len(got), got, c.notices)
			}
			if c.notices == 1 {
				if got[0].Severity != acp.NoticeSeverityWarning || got[0].Title != "compaction: automatic compaction paused" ||
					got[0].Description == nil || !strings.Contains(*got[0].Description, "Summarization failed: llm: offline") {
					t.Fatalf("notice %+v", got[0])
				}
			}
			if p, _ := h.live(id); p != acpext.PauseTurn {
				t.Fatalf("pause %q, want the turn's", p)
			}
			h.noLegacy()
			h.conforms()
		})
	}
}

// holdSummary is a provider in front of a script that holds the first
// summary request until the turn is cancelled, then fails it, or, with
// finish, answers it, so the entry is written after the cancel.
type holdSummary struct {
	*llmtest.Script
	finish bool
	asked  chan struct{}
	once   sync.Once
}

func (p *holdSummary) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	held := false
	if strings.HasPrefix(systemOf(req), "You are a context summarization assistant.") {
		p.once.Do(func() { held = true })
	}
	if !held {
		return p.Script.Stream(ctx, req)
	}
	return func(yield func(llm.Event, error) bool) {
		close(p.asked)
		<-ctx.Done()
		if !p.finish {
			yield(nil, context.Cause(ctx))
			return
		}
		for _, ev := range []llm.Event{llm.TextDelta{Text: "## Goal\nHELD SUMMARY"}, llm.Done{StopReason: llm.StopEndTurn}} {
			if !yield(ev, nil) {
				return
			}
		}
	}
}

// holding is a harness whose summary request hold holds.
func holding(t *testing.T, hold *holdSummary, turns ...llmtest.Turn) *compactHarness {
	t.Helper()
	hold.asked = make(chan struct{})
	return newCompactHarnessWith(t, harnessConfig{caps: offersCompaction, wrap: func(s *llmtest.Script) llm.Provider {
		hold.Script = s
		return hold
	}}, SessionSettings{}, turns...)
}

// cancelWhenAsked cancels the session's turn once hold has the summary
// request.
func (h *compactHarness) cancelWhenAsked(id acp.SessionId, hold *holdSummary) {
	h.t.Helper()
	select {
	case <-hold.asked:
	case <-time.After(10 * time.Second):
		h.t.Fatal("the summary was never asked for")
	}
	if err := h.c.Client.Cancel(h.t.Context(), acp.CancelNotification{SessionId: id}); err != nil {
		h.t.Fatal(err)
	}
}

// A compaction cancelled before its entry is written is cancelled; one
// cancelled after it is completed, as the entry stands, and /compact's turn
// still ends cancelled (0016-MADR P3-Q4).
func TestCompactionUpdateCancelled(t *testing.T) {
	for _, c := range []struct {
		name   string
		finish bool
		want   acp.CompactionStatus
	}{{"before the write", false, acp.CompactionStatusCancelled}, {"after the write", true, acp.CompactionStatusCompleted}} {
		t.Run("/compact "+c.name, func(t *testing.T) {
			hold := &holdSummary{finish: c.finish}
			h := holding(t, hold, full("r1", 10), full("r2", 10))
			id := h.session()
			h.mustPrompt(id, p1)
			h.mustPrompt(id, p2)
			n := len(h.client.Updates())
			done := startPrompt(t, h.c, id, "/compact")
			h.cancelWhenAsked(id, hold)
			if resp := waitPrompt(t, done); resp.StopReason != acp.StopReasonCancelled {
				t.Fatalf("stop reason %q, want cancelled", resp.StopReason)
			}
			after := h.client.Updates()[n:]
			got := rowsOf(t, after)
			want := manualRow(c.want, 0)
			if c.finish {
				want.meta.TokensBefore = -1
			}
			checkRows(t, got, []row{manualRow(acp.CompactionStatusInProgress, 0), want})
			entryID, _, _ := h.compactionEntry(id)
			if c.finish {
				h.checkCompleted(id, got[1])
				lastIsUsage(t, after)
			} else if entryID != "" {
				t.Fatalf("cancelled compaction wrote entry %q", entryID)
			}
			h.noLegacy()
			h.conforms()
		})
	}

	t.Run("_gobble/compact after the write", func(t *testing.T) {
		hold := &holdSummary{finish: true}
		h := holding(t, hold, full("r1", 10), full("r2", 10))
		id := h.session()
		h.mustPrompt(id, p1)
		h.mustPrompt(id, p2)
		type result struct {
			res map[string]any
			err error
		}
		done := make(chan result, 1)
		go func() {
			raw, err := h.c.Client.CallExtension(context.WithoutCancel(t.Context()), "_gobble/compact", map[string]any{"sessionId": id})
			var res map[string]any
			if err == nil {
				err = json.Unmarshal(raw, &res)
			}
			done <- result{res, err}
		}()
		h.cancelWhenAsked(id, hold)
		r := <-done
		if r.err != nil || !strings.Contains(r.res["summary"].(string), "HELD SUMMARY") {
			t.Fatalf("_gobble/compact written then cancelled: %v, %v; want its result", r.res, r.err)
		}
		got := rowsOf(t, h.client.Updates())
		checkRows(t, got, []row{manualRow(acp.CompactionStatusInProgress, 0), manualRow(acp.CompactionStatusCompleted, -1)})
		h.checkCompleted(id, got[1])
		h.noLegacy()
	})

	t.Run("automatic before the write", func(t *testing.T) {
		hold := &holdSummary{}
		h := holding(t, hold, full("r1", 110000))
		id := h.session()
		h.mustPrompt(id, p1)
		n := len(h.client.Updates())
		done := startPrompt(t, h.c, id, p2)
		h.cancelWhenAsked(id, hold)
		if resp := waitPrompt(t, done); resp.StopReason != acp.StopReasonCancelled {
			t.Fatalf("stop reason %q, want cancelled", resp.StopReason)
		}
		checkRows(t, rowsOf(t, h.client.Updates()[n:]), []row{
			{status: acp.CompactionStatusInProgress, meta: acpext.CompactionMeta{Reason: acpext.CompactionThreshold}},
			{status: acp.CompactionStatusCancelled, meta: acpext.CompactionMeta{Reason: acpext.CompactionThreshold}},
		})
		h.noLegacy()
		h.conforms()
	})
}
