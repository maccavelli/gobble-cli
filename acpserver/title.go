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
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/session"
)

// TitleOptions are the small model that names a session from its first
// prompt (0005-PLAN F5-5). A zero TitleOptions names none.
type TitleOptions struct {
	// Provider builds the small model's provider for a session, with the
	// session's models.dev document, id and retry settings, as
	// Options.Provider builds the session's (0072-MADR D17, M86).
	Provider func(data *provider.ModelData, sessionID string, retry llm.Retry) (llm.Provider, error)
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
	a.mu.Unlock()
	if tried || s.state().name != "" || session.HasConversation(s.tree.Path()) {
		a.mu.Lock()
		s.titleTried = true
		a.mu.Unlock()
		return false
	}
	// A title is asked: under a ChatGPT plan session it is a background
	// request, made only with the user's consent. The answer is read only
	// now, when a request would otherwise be made; with none, the title
	// shows nothing and the attempt is not used, and an answer that cannot
	// be read leaves one line in the log (0052-MADR D2).
	if _, gated, err := a.backgroundGate(a.titles.ProviderID); gated {
		if err != nil {
			a.logger.Warn("title: the stored consent could not be read; no title was asked",
				slog.String("provider", a.titles.ProviderID), slog.String("error", err.Error()))
		}
		return false
	}
	a.mu.Lock()
	s.titleTried = true
	a.mu.Unlock()
	return true
}

// ConsentState is provider id's background-use consent as Options.PlanConsent
// answers it: Consent is the consent's id, the SDK's; Plan is true when the
// stored session needs it; Granted when the user granted it (0052-MADR D2).
type ConsentState struct {
	Consent       string
	Plan, Granted bool
}

// backgroundGate asks Options.PlanConsent whether a background request, a
// title or an automatic compaction, may be made under provider id's
// credential. It is gated when the credential is a session that needs the
// consent and none is granted, and when the answer cannot be read, which is
// err: an answer that cannot be read is not a grant. The state is the one
// asked, for the id the refusal names. Nothing is gated with no
// Options.PlanConsent (0052-MADR D2).
func (a *Agent) backgroundGate(id string) (st ConsentState, gated bool, err error) {
	if a.planConsent == nil {
		return ConsentState{}, false, nil
	}
	st, err = a.planConsent(id)
	return st, err != nil || st.Plan && !st.Granted, err
}

// refusal is an error of gobble's own that matches llm.ErrNotPermitted and
// says msg, which is all it says.
type refusal struct{ msg string }

func (r refusal) Error() string { return r.msg }
func (refusal) Unwrap() error   { return llm.ErrNotPermitted }

// errPlanOverflow ends a turn whose context is over the window under a
// ChatGPT plan session with no consent to automatic compaction: it names
// /compact, which is the user's own request and is not gated, and the
// command that allows the consent. It is a ConsentError, so the turn's error
// carries the consent's id (0052-MADR D2).
func errPlanOverflow(id string, st ConsentState) error {
	return &llm.ConsentError{Consent: st.Consent, Err: refusal{"the context is over the model's window, and automatic " +
		"compaction runs under your ChatGPT plan only with your consent: run /compact, or allow it with gobble auth consent " +
		id + " " + st.Consent + " allow"}}
}

// errPlanUnreadable ends a turn whose context is over the window when the
// stored answer for background use could not be read, with the store's error.
// It names /compact and not the consent command: the answer may already be
// granted, and the command would meet the same store. It is no ConsentError,
// so the turn's error carries no consent id (0052-MADR D2).
func errPlanUnreadable(cause error) error {
	return refusal{"the context is over the model's window, and the stored answer for background use under your " +
		"ChatGPT plan could not be read (" + cause.Error() + "): run /compact"}
}

// startTitle names s from its first prompt in the background, while the
// turn runs (decision 4): its context keeps parent's values, not its end,
// since the title outlives the prompt's request. closeTitle stops it.
func (a *Agent) startTitle(parent context.Context, s *liveSession, prompt string) {
	ctx, cancel := context.WithCancel(llm.WithBackground(context.WithoutCancel(parent)))
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
	p, err := a.titles.Provider(a.dataOf(s), string(s.id), retry) // the session's document (0072-MADR M62, M86)
	if err != nil {
		return err
	}
	// The title waits for the title model's discovery, here in its own
	// goroutine, bounded (0071-MADR P40).
	a.waitDiscovery(ctx, s, a.titles.ProviderID, a.titles.Model)
	info := a.modelOf(s, a.titles.ProviderID, a.titles.Model)
	// The title's cap is min(128, C), C the streamed output a turn reads
	// (0071-MADR V8; 0065-MADR's amendment of 2026-11-25).
	maxTokens := titleMaxTokens
	if c, _, _ := outputCap(info); c > 0 {
		maxTokens = min(maxTokens, c)
	}
	req := &llm.Request{Model: a.titles.Model, Thinking: catalog.Levels(info)[0], MaxTokens: maxTokens, Messages: []llm.Message{
		{Role: llm.RoleSystem, Content: []llm.Content{{Type: llm.ContentText, Text: titlePrompt}}},
		{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "<user_message>\n" + cutRunes(prompt, titleSourceChars) + "\n</user_message>"}}},
	}}
	text, usage, stop, err := a.titleCall(ctx, s, p, req)
	if err != nil {
		return err
	}
	// closeTitle waits for this to return before the log closes.
	write := context.WithoutCancel(ctx)
	if err := s.append(write, a.now(), session.Entry{Type: session.TypeUsage, Kind: usageKindTitle, Provider: a.titles.ProviderID,
		Model: a.titles.Model, Usage: pricedUsage(usage, info)}); err != nil {
		return err
	}
	if stop == llm.StopMaxTokens {
		// A title cut at its limit is not used, as a cut summary is not; its
		// usage is written (0071-MADR P30, the owner's "Discard a cut title
		// (Recommended)").
		return errors.New("the small model's title was cut at its output limit")
	}
	title := cleanTitle(text)
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

// titleCall streams the title's request, and, when the service refuses its
// limit stating a smaller cap as the error's Observed, learns the cap and
// sends it once more at it (0071-MADR P34). A completed call's output is
// reported to the saved caps (P25).
func (a *Agent) titleCall(ctx context.Context, s *liveSession, p llm.Provider, req *llm.Request) (string, llm.Usage, llm.StopReason, error) {
	p = watchCalls(p, a.outputSeen(a.titles.ProviderID), nil)
	for resent := false; ; resent = true {
		var (
			text  strings.Builder
			usage llm.Usage
			stop  llm.StopReason
			err   error
		)
		for ev, e := range p.Stream(ctx, req) {
			if e != nil {
				err = e
				break
			}
			switch ev := ev.(type) {
			case llm.TextDelta:
				text.WriteString(ev.Text)
			case llm.Usage:
				usage = ev
			case llm.Done:
				stop = ev.StopReason
			}
		}
		if ob, ok := observedBelow(err, req.MaxTokens); ok && !resent {
			a.learnedCap(s, a.titles.ProviderID, req.Model, ob)
			again := *req
			again.MaxTokens = ob.Tokens
			req = &again
			continue
		}
		return text.String(), usage, stop, err
	}
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
