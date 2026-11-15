package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// planNoticeLines are what gobble stores for a ChatGPT plan sign-in, where,
// how to remove it, and what a background request is: shown before the
// browser opens, in the sign-in's confirm step (the Sign in with ChatGPT
// terms, sections 1, 2.3 and 2.4). The fifth line names the command that
// answers the question again (0052-MADR D2, D3).
var planNoticeLines = []string{
	`gobble signs in to your ChatGPT plan under its own name, "gobble".`,
	"It stores the access, refresh and ID tokens, your account id and this host's id, in auth.json in gobble's " +
		"config directory on this machine, readable by you alone, and sends them only to OpenAI.",
	"This machine must be one only you control, never a shared or managed one. Do not copy auth.json to another machine.",
	"To remove the sign-in, run gobble auth logout openai, which also revokes it at OpenAI.",
	"Background requests, a session's title and automatic compaction, run under your plan only if you allow them; " +
		"gobble auth consent openai background deny withdraws that at any time.",
	"On another device, open the address shown, sign in there, then paste here the address the browser lands on.",
}

// planQuestion is the consent's question, after the notice.
const planQuestion = "Allow gobble to make background requests under your ChatGPT plan: naming sessions and " +
	"compacting them on its own?"

// ConsentText is gobble's wording for each consent it honours, by the SDK's
// consent id, as the SDK's wizard and signin take it: the notice, then the
// question (0052-MADR D2). Offered gives the same text.
var ConsentText = map[string]string{
	openai.ConsentBackgroundUse: strings.Join(planNoticeLines, "\n") + "\n\n" + planQuestion,
}

// consentWords are gobble's words for consents, accepted wherever a consent's
// id is, for a person at a terminal (0052-MADR D3).
var consentWords = map[string]string{openai.ConsentBackgroundUse: "background"}

// Errors of the consent functions, which a caller tells from a store's.
var (
	// ErrNoStoredSession is a consent asked of a provider with no stored
	// session gobble uses.
	ErrNoStoredSession = errors.New("provider: no stored sign-in to answer a consent for")
	// ErrUnknownConsent is a consent the stored session's sign-in method
	// does not have.
	ErrUnknownConsent = errors.New("provider: the stored sign-in has no such consent")
)

// ConsentState is provider id's background-use consent as gobble gates on
// it: Consent is the SDK's id, "" for a provider that has none; Plan whether
// the stored session is one that needs it, and Granted whether the user
// granted it.
type ConsentState struct {
	Consent       string
	Plan, Granted bool
}

// Answers a consent can have, as ConsentAnswer.Answer words them.
const (
	AnswerGranted  = "granted"
	AnswerDeclined = "declined"
	AnswerNotAsked = "not asked"
)

// ConsentAnswer is one consent of a stored session's method and its answer:
// ID is the SDK's, Word gobble's word for it, "" when it has none.
type ConsentAnswer struct {
	ID, Word, Answer string
}

// consentMethod is the offered session method of provider id that has an
// honoured consent, and that consent.
func consentMethod(id string) (authcontract.Method, authcontract.Consent, bool) {
	for _, m := range yields(id, authcontract.YieldsOAuthSession) {
		for _, c := range m.Consents {
			if c.Required && sessionKinds[sessionKey{id, m.ClientIdentity}] != "" {
				return m, c, true
			}
		}
	}
	return authcontract.Method{}, authcontract.Consent{}, false
}

// storedMethod is the offered session method that stored login l of provider
// id was signed in by: the one whose kind is l's.
func storedMethod(id string, l *Login) (authcontract.Method, bool) {
	if l == nil {
		return authcontract.Method{}, false
	}
	kind := l.session(id, nil, nil, nil).SessionKind()
	if kind == "" {
		return authcontract.Method{}, false
	}
	for _, m := range yields(id, authcontract.YieldsOAuthSession) {
		if sessionKinds[sessionKey{id, m.ClientIdentity}] == kind {
			return m, true
		}
	}
	return authcontract.Method{}, false
}

// PlanConsent is provider id's consent state, read from the stored session
// now: the consent's id from the offered method's facts; Plan when the
// stored session is the kind that method signs in to; Granted when that
// session's answer is granted. A provider with no such method has the zero
// state. An unreadable store is an error, which a caller treats as no grant:
// an answer that cannot be read is not one (0052-MADR D2).
func PlanConsent(id string, store LoginStore) (ConsentState, error) {
	m, c, ok := consentMethod(id)
	if !ok {
		return ConsentState{}, nil
	}
	l, err := store.Load(context.Background(), id)
	if err != nil {
		return ConsentState{Consent: c.ID}, err
	}
	stored, ok := storedMethod(id, l)
	if !ok || stored.ID != m.ID {
		return ConsentState{Consent: c.ID}, nil
	}
	granted, _ := l.session(id, nil, nil, nil).Consent(c.ID)
	return ConsentState{Consent: c.ID, Plan: true, Granted: granted}, nil
}

// Consents lists the consents of the method provider id's stored session was
// signed in by, each with its answer: granted, declined or not asked. A
// provider with no stored session gobble uses is ErrNoStoredSession; a
// session whose method has no consent lists none. A store that cannot be
// read is its error (0052-MADR D3).
func Consents(ctx context.Context, store LoginStore, id string) ([]ConsentAnswer, error) {
	l, err := store.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	m, ok := storedMethod(id, l)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoStoredSession, id)
	}
	s := l.session(id, nil, nil, nil)
	var out []ConsentAnswer
	for _, c := range m.Consents {
		a := ConsentAnswer{ID: c.ID, Word: consentWords[c.ID], Answer: AnswerNotAsked}
		if granted, answered := s.Consent(c.ID); answered {
			a.Answer = AnswerDeclined
			if granted {
				a.Answer = AnswerGranted
			}
		}
		out = append(out, a)
	}
	return out, nil
}

// SetConsent stores the answer to consent, the SDK's id or gobble's word for
// it, on provider id's stored session: granted or declined, under the
// provider's refresh lock, so it never races a refresh (the SDK's
// auth.SetConsent). A provider with no stored session gobble uses is
// ErrNoStoredSession, and a consent its method does not have is
// ErrUnknownConsent, each before anything is written; a store that cannot be
// read or locked is its error (0052-MADR D3).
func SetConsent(ctx context.Context, store LoginStore, id, consent string, granted bool) error {
	answers, err := Consents(ctx, store, id)
	if err != nil {
		return err
	}
	for _, a := range answers {
		if consent == a.ID || a.Word != "" && consent == a.Word {
			return auth.SetConsent(ctx, sdkStore{store}, llmprovider.ProviderID(id), a.ID, granted)
		}
	}
	return fmt.Errorf("%w: %s has no consent %q", ErrUnknownConsent, id, consent)
}
