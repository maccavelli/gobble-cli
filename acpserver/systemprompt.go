package acpserver

import (
	"context"
	"strings"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/jsonx"
	sysprompt "github.com/maccavelli/gobble-cli/prompt"
	"github.com/maccavelli/gobble-cli/session"
)

// systemPromptData is a gobble.system_prompt entry's data: a client's system
// prompt, its mode and the text of its blocks joined by a blank line. The
// zero value is none (0054-MADR D7).
type systemPromptData struct {
	Mode string `json:"mode"`
	Text string `json:"text"`
}

// clientPrompt is the system prompt a session/new, session/load or
// session/resume carries, joined and checked; zero for none. The SDK has
// refused an unadvertised mode, a block that is not text and empty content
// before the handler runs; text gobble cannot use, blank or closing its
// section's tag, is invalid params, so nothing is created, loaded or
// resumed (0054-MADR D7).
func clientPrompt(id acp.SessionId, p *acp.SystemPrompt) (systemPromptData, error) {
	if p == nil {
		return systemPromptData{}, nil
	}
	parts := make([]string, 0, len(p.Content))
	for _, b := range p.Content {
		if b.Text != nil {
			parts = append(parts, b.Text.Text)
		}
	}
	text := strings.Join(parts, "\n\n")
	if err := sysprompt.CheckClient(text); err != nil {
		return systemPromptData{}, invalidParams(id, "the system prompt cannot be used: "+err.Error())
	}
	return systemPromptData{Mode: string(p.Mode), Text: text}, nil
}

func systemPromptEntry(d systemPromptData) (session.Entry, error) {
	data, err := jsonx.Value(d)
	return session.Entry{Type: session.TypeCustom, CustomType: session.CustomSystemPrompt, Data: data}, err
}

// systemPromptOf is a gobble.system_prompt entry's prompt.
func systemPromptOf(e session.Entry) (systemPromptData, bool) {
	if e.Type != session.TypeCustom || e.CustomType != session.CustomSystemPrompt {
		return systemPromptData{}, false
	}
	var d systemPromptData
	if err := jsonx.Unmarshal(e.Data, &d); err != nil || d.Text == "" {
		return systemPromptData{}, false
	}
	return d, true
}

// takeSystemPrompt makes p, a client's prompt, s's own before s is live: one
// that differs from the prompt s holds is written as a gobble.system_prompt
// entry, then held; none, or the same one, writes nothing, so a stored prompt
// is kept (0054-MADR D7).
func (a *Agent) takeSystemPrompt(ctx context.Context, s *liveSession, p systemPromptData) error {
	if p.Text == "" || p == s.client {
		return nil
	}
	e, err := systemPromptEntry(p)
	if err != nil {
		return err
	}
	if err := s.append(ctx, a.now(), e); err != nil {
		return err
	}
	s.client = p
	return nil
}
