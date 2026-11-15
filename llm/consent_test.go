package llm

import (
	"errors"
	"fmt"
	"testing"
)

// A ConsentError is a refusal: errors.Is finds ErrNotPermitted through it and
// through a wrap of it, errors.As finds its consent, its text is its Err's,
// and Kind gives not_permitted (0052-MADR D2).
func TestConsentErrorIsNotPermitted(t *testing.T) {
	err := fmt.Errorf("the turn: %w", &ConsentError{Consent: "chatgpt_plan.background_use", Err: fmt.Errorf("%w: needs the consent", ErrNotPermitted)})
	if !errors.Is(err, ErrNotPermitted) {
		t.Errorf("errors.Is(%v, ErrNotPermitted) is false", err)
	}
	ce, ok := errors.AsType[*ConsentError](err)
	if !ok || ce.Consent != "chatgpt_plan.background_use" {
		t.Errorf("errors.As found %+v, %v; want the consent id", ce, ok)
	}
	if got := Kind(err); got != "not_permitted" {
		t.Errorf("Kind = %q, want not_permitted", got)
	}
	if got, want := ce.Error(), "llm: not permitted: needs the consent"; got != want {
		t.Errorf("Error = %q, want its Err's text %q", got, want)
	}
	if got := (&ConsentError{}).Error(); got == "" {
		t.Error("an empty ConsentError has no text")
	}
	if errors.Is(&ConsentError{Consent: "x", Err: errors.New("plain")}, ErrNotPermitted) {
		t.Error("a ConsentError over a plain error matched ErrNotPermitted")
	}
}
