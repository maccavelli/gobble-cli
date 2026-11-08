package acpserver

import (
	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/jsonx"
)

// compactionReport reports one compaction to the client, by the one
// channel its connection offered: ACP's compaction_update when the client
// offered session.compaction, gobble's _gobble/compaction otherwise
// (0016-MADR D9, G1). Each compaction chooses its report with
// newCompactionReport before anything is sent, so an update the client did
// not offer is never sent: the SDK would refuse it, and the refusal would
// end the turn.
type compactionReport interface {
	// rows reports whether the client shows a compaction as a row of its
	// own, so /compact need not say that it is compacting.
	rows() bool
	// started reports a prepared compaction about to ask for its summary,
	// whose entry will carry id.
	started(id string)
	// ended reports how the compaction ended: res, once its entry was
	// written, when err is nil; otherwise err, which cancelled says was the
	// turn's cancel, and paused, the pause a failed automatic compaction
	// took. A compaction that ends without starting was refused before it
	// ran. It reports whether the client was shown the summary or the
	// error, so the caller says neither again.
	ended(res compaction.Result, err error, cancelled bool, paused acpext.CompactionPause) (shown bool)
}

// newCompactionReport is the report of a compaction of s for reason,
// sent through out.
func (a *Agent) newCompactionReport(out *updates, s *liveSession, reason acpext.CompactionReason) compactionReport {
	if a.offersCompaction(s.id) {
		return &compactionUpdates{a: a, s: s, out: out, reason: reason}
	}
	return &compactionNotices{out: out, session: s.id, reason: reason}
}

// offersCompaction reports whether the session's client offered
// session.compaction, by D7's gate.
func (a *Agent) offersCompaction(id acp.SessionId) bool {
	return a.offers(id, acp.UpdateCompaction("probe", acp.CompactionStatusCompleted))
}

// compactionNotices is the report a client that did not offer
// session.compaction has always had: _gobble/compaction's start and end
// (0005-PLAN F5-2 decision 9), unchanged.
type compactionNotices struct {
	out     *updates
	session acp.SessionId
	reason  acpext.CompactionReason
	begun   bool
}

func (*compactionNotices) rows() bool { return false }

func (r *compactionNotices) started(string) {
	r.begun = true
	r.out.notify(acpext.MethodCompaction, acpext.CompactionNotice{SessionID: string(r.session), Phase: acpext.PhaseStart, Reason: r.reason})
}

// ended sends the end: "cancelled", the error and its pause, or the tokens
// before. A refusal is a start and an end, as it has always been.
func (r *compactionNotices) ended(res compaction.Result, err error, cancelled bool, paused acpext.CompactionPause) bool {
	if !r.begun {
		r.started("")
	}
	n := acpext.CompactionNotice{SessionID: string(r.session), Phase: acpext.PhaseEnd, Reason: r.reason}
	switch {
	case cancelled:
		n.Error = "cancelled"
	case err != nil:
		n.Error, n.Paused = err.Error(), paused
	default:
		n.TokensBefore = res.TokensBefore
	}
	r.out.notify(acpext.MethodCompaction, n)
	return false
}

// compactionUpdates is the report of a client that offered
// session.compaction: one compaction_update in_progress, then one terminal
// update with the same id, the compaction entry's (0016-MADR P3-Q1 to
// P3-Q5). A refusal sends nothing.
type compactionUpdates struct {
	a      *Agent
	s      *liveSession
	out    *updates
	reason acpext.CompactionReason
	id     acp.CompactionId // "" until started
}

func (*compactionUpdates) rows() bool { return true }

func (r *compactionUpdates) started(id string) {
	r.id = acp.CompactionId(id)
	r.send(acp.UpdateCompaction(r.id, acp.CompactionStatusInProgress), acpext.CompactionMeta{Reason: r.reason})
}

// ended sends completed with the summary whenever the entry was written,
// a cancel after the write included; cancelled for a cancel before it;
// failed with the error's first line otherwise, and, for a pause, the
// pause and a warning notice when the client offered session.notices.
func (r *compactionUpdates) ended(res compaction.Result, err error, cancelled bool, paused acpext.CompactionPause) bool {
	if r.id == "" {
		return false
	}
	meta := acpext.CompactionMeta{Reason: r.reason}
	switch {
	case err == nil:
		meta.TokensBefore = res.TokensBefore
		r.send(completedUpdate(r.id, res.Summary), meta)
	case cancelled:
		r.send(acp.UpdateCompaction(r.id, acp.CompactionStatusCancelled), meta)
	default:
		meta.Paused = paused
		u := acp.UpdateCompaction(r.id, acp.CompactionStatusFailed)
		u.CompactionUpdate.Error = acp.PatchValue(firstLine(err.Error()))
		r.send(u, meta)
		if paused != pauseNone {
			r.a.queueNotice(r.s, acp.NoticeSeverityWarning, "compaction: automatic compaction paused", pausedText(err, paused))
			r.a.sendNotices(r.out, r.s)
		}
	}
	return true
}

// send sends u with meta as its _meta.gobble. The protocol replaces an
// update's _meta whole, so each update carries all of it.
func (r *compactionUpdates) send(u acp.SessionUpdate, meta acpext.CompactionMeta) {
	if err := withMeta(u, meta); err != nil {
		if r.out.err == nil {
			r.out.err = err
		}
		return
	}
	r.out.send(u)
}

// completedUpdate is a completed compaction_update carrying summary in its
// summary field, which replaces any chunks (0016-MADR P3-Q2).
func completedUpdate(id acp.CompactionId, summary string) acp.SessionUpdate {
	u := acp.UpdateCompaction(id, acp.CompactionStatusCompleted)
	u.CompactionUpdate.Summary = acp.PatchValue([]acp.ContentBlock{acp.TextBlock(summary)})
	return u
}

// withMeta sets u's _meta to gobble's member, meta, encoded as gobble
// encodes its members (0039-MADR D15).
func withMeta(u acp.SessionUpdate, meta acpext.CompactionMeta) error {
	m, err := jsonx.SetMember(nil, gobble, meta)
	if err != nil {
		return err
	}
	u.CompactionUpdate.Meta = acp.PatchValue(m)
	return nil
}

// pausedText is the pause notice's description: the failure, and what
// ends the pause (0005-PLAN F5-2 decision 11).
func pausedText(err error, p acpext.CompactionPause) string {
	if p == acpext.PauseSession {
		return firstLine(err.Error()) + "; automatic compaction resumes after /compact, a model change, or turning it on again"
	}
	return firstLine(err.Error()) + "; automatic compaction resumes with the next prompt"
}
