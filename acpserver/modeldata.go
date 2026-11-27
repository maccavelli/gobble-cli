package acpserver

import "github.com/maccavelli/gobble-cli/llm/provider"

// A session's model data (0072-MADR D19): the process's current models.dev
// document when the session is created, loaded or forked, kept until it
// ends, so a refresh during its life changes none of its facts. A session
// that began with none takes the first document there is, once, and keeps
// that one ("Pick up the first download (Recommended)", M61).

// takeModelData gives s the current document, or none, at its start
// (0072-MADR D19, M60).
func (a *Agent) takeModelData(s *liveSession) {
	if a.modelData != nil {
		if d := a.modelData(); d != nil {
			s.data.Store(d)
		}
	}
}

// dataOf is s's document: the one it began with, or, for a session that
// began with none, the first current one there is, which it keeps. nil is
// none, and for nil s; the ModelInfo and CachedLimits hooks read nil as
// the current document (0072-MADR D19, M61, M80).
func (a *Agent) dataOf(s *liveSession) *provider.ModelData {
	if s == nil {
		return nil
	}
	if d := s.data.Load(); d != nil {
		return d
	}
	if a.modelData == nil {
		return nil
	}
	if d := a.modelData(); d != nil {
		s.data.CompareAndSwap(nil, d)
	}
	return s.data.Load()
}

// modelDataLine is /context's and /session's line for s's document, after
// the output limit's, from Options.ModelDataLine; "" with no hook
// (0072-MADR D12, M37).
func (a *Agent) modelDataLine(s *liveSession) string {
	if a.dataLine == nil {
		return ""
	}
	line := a.dataLine(a.dataOf(s))
	if line == "" {
		return ""
	}
	return line + "\n"
}
