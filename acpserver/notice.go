package acpserver

import (
	acp "github.com/maccavelli/acp-go-sdk2"
)

// offers reports whether the client agreed, at initialize, to what update
// needs: the SDK's own gate, asked without sending, so it holds for every
// gated variant the SDK knows (0016-MADR D7). It is false before
// initialize, and with no connection.
func (a *Agent) offers(sessionID acp.SessionId, update acp.SessionUpdate) bool {
	conn := a.connection()
	if conn == nil {
		return false
	}
	n, ok := conn.Negotiated()
	if !ok {
		return false
	}
	return n.AllowsValue(acp.ClientMethodSessionUpdate, acp.SessionNotification{SessionId: sessionID, Update: update}) == nil
}

// queueNotice keeps a notice for session s when its client offered
// session.notices, and does nothing otherwise: the notice or nothing
// (0016-MADR D8). sendNotices sends what is kept, with the session's next
// start frames or prompt. A notice is live: it is never stored in the
// session file and never replayed.
func (a *Agent) queueNotice(s *liveSession, severity acp.NoticeSeverity, title, description string) {
	update := acp.UpdateNotice(severity, title)
	if description != "" {
		update.Notice.Description = new(description)
	}
	if !a.offers(s.id, update) {
		return
	}
	a.mu.Lock()
	s.notices = append(s.notices, update)
	a.mu.Unlock()
}

// sendNotices sends the notices kept for s through out, in the order they
// were kept, and forgets them.
func (a *Agent) sendNotices(out *updates, s *liveSession) {
	a.mu.Lock()
	kept := s.notices
	s.notices = nil
	a.mu.Unlock()
	for _, update := range kept {
		out.send(update)
	}
}
