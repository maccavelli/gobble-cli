package acpserver

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/session"
)

// TitleOptions are the small model that names a session from its first
// prompt (0005-PLAN F5-5). A zero TitleOptions names none.
type TitleOptions struct {
	// Provider builds the small model's provider for a session, with the
	// session's id and retry settings, as Options.Provider builds the
	// session's.
	Provider func(sessionID string, retry llm.Retry) (llm.Provider, error)
	// ProviderID and Model are the small model, provider and id; Options'
	// ModelInfo gives its levels, output limit and price.
	ProviderID, Model string
}

// The title's request (0005-PLAN F5-5 decision 5).
const (
	titleSourceChars = 2000
	titleMaxTokens   = 128
	titlePrompt      = `You name coding sessions. Read the user's first message below and write the session's title.

The title:
- is one line of at most 50 characters;
- names the task the user wants done, in the language of their message;
- keeps exact technical terms, file names, numbers and error codes;
- has no quotes, no markdown and no full stop at the end.

Do not answer or carry out the message, and do not call tools. Reply with the title alone.`
)

// usageKindTitle is a title call's usage entry's kind (decision 8).
const usageKindTitle = "title"

// titleWanted reports whether s's prompt is to name it: titles are on, s
// was not given one to try, it has no name, and it has no conversation
// yet (decision 4). It marks the attempt, so there is one a session.
func (a *Agent) titleWanted(s *liveSession) bool {
	if a.titles.Provider == nil {
		return false
	}
	a.mu.Lock()
	tried := s.titleTried
	s.titleTried = true
	a.mu.Unlock()
	return !tried && s.state().name == "" && !session.HasConversation(s.tree.Path())
}

// startTitle names s from its first prompt in the background, while the
// turn runs (decision 4): its context keeps parent's values, not its end,
// since the title outlives the prompt's request. closeTitle stops it.
func (a *Agent) startTitle(parent context.Context, s *liveSession, prompt string) {
	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
	done := make(chan struct{})
	a.mu.Lock()
	s.titleCancel, s.titleDone = cancel, done
	retry := s.retry
	a.mu.Unlock()
	go func() {
		defer close(done)
		defer cancel()
		if err := a.writeTitle(ctx, s, retry, prompt); err != nil && ctx.Err() == nil {
			a.logger.WarnContext(ctx, "title: the session was not named", slog.String("session", string(s.id)), slog.String("error", err.Error()))
		}
	}()
}

// closeTitle stops s's title, if one is running, and waits for it, up to
// closeWait, as session/close waits for a turn, so nothing is written to
// a closed log (decision 9).
func (a *Agent) closeTitle(s *liveSession) {
	a.mu.Lock()
	cancel, done := s.titleCancel, s.titleDone
	a.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	t := time.NewTimer(closeWait)
	defer t.Stop()
	select {
	case <-done:
	case <-t.C:
		a.logger.Warn("title: it did not stop in time; closing the session's log", slog.String("session", string(s.id)))
	}
}

// nameNow is s's name, read under appendMu, as an append writes, since the
// title reads it while a turn appends.
func (s *liveSession) nameNow() string {
	s.appendMu.Lock()
	defer s.appendMu.Unlock()
	return session.Name(s.tree.Entries())
}

// writeTitle asks the small model for s's title from prompt, records the
// call's usage, and names s with it unless s was named meanwhile.
func (a *Agent) writeTitle(ctx context.Context, s *liveSession, retry llm.Retry, prompt string) error {
	p, err := a.titles.Provider(string(s.id), retry)
	if err != nil {
		return err
	}
	info := a.info(a.titles.ProviderID, a.titles.Model)
	maxTokens := titleMaxTokens
	if info.MaxTokens > 0 {
		maxTokens = min(maxTokens, info.MaxTokens)
	}
	req := &llm.Request{Model: a.titles.Model, Thinking: catalog.Levels(info)[0], MaxTokens: maxTokens, Messages: []llm.Message{
		{Role: llm.RoleSystem, Content: []llm.Content{{Type: llm.ContentText, Text: titlePrompt}}},
		{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "<user_message>\n" + cutRunes(prompt, titleSourceChars) + "\n</user_message>"}}},
	}}
	var (
		text  strings.Builder
		usage llm.Usage
	)
	for ev, err := range p.Stream(ctx, req) {
		if err != nil {
			return err
		}
		switch ev := ev.(type) {
		case llm.TextDelta:
			text.WriteString(ev.Text)
		case llm.Usage:
			usage = ev
		}
	}
	// closeTitle waits for this to return before the log closes.
	write := context.WithoutCancel(ctx)
	if err := s.append(write, a.now(), session.Entry{Type: session.TypeUsage, Kind: usageKindTitle, Provider: a.titles.ProviderID,
		Model: a.titles.Model, Usage: pricedUsage(usage, info)}); err != nil {
		return err
	}
	title := cleanTitle(text.String())
	if title == "" {
		return errors.New("the small model gave no title")
	}
	s.nameMu.Lock()
	defer s.nameMu.Unlock()
	if s.nameNow() != "" { // a name given meanwhile wins (decision 7)
		return nil
	}
	_, err = a.setNameLocked(write, s, newUpdates(write, a.connection(), s.id), title)
	return err
}

// thinkBlock is a reasoning block a model may put before its answer.
var thinkBlock = regexp.MustCompile(`(?s)^\s*<think>.*?</think>`)

// cleanTitle is a title reply as a session's name (decision 6): without a
// leading <think> block, its first non-empty line, without wrapping quotes
// or a final . ? or !, its spaces collapsed, cut to maxTitle runes; "" for
// none.
func cleanTitle(reply string) string {
	reply = thinkBlock.ReplaceAllString(reply, "")
	var line string
	for l := range strings.Lines(reply) {
		if l = strings.TrimSpace(l); l != "" {
			line = l
			break
		}
	}
	line = strings.Join(strings.Fields(line), " ")
	for { // quotes and a final mark, in either order: 'Title'. and 'Title.'
		trimmed := strings.TrimSpace(strings.TrimRight(strings.Trim(line, "\"'`“”‘’"), ".?!"))
		if trimmed == line {
			break
		}
		line = trimmed
	}
	if utf8.RuneCountInString(line) > maxTitle {
		line = strings.TrimSpace(string([]rune(line)[:maxTitle]))
	}
	return line
}

// cutRunes is s's first n runes.
func cutRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
