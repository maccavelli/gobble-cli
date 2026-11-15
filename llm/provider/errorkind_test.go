package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"

	"github.com/maccavelli/gobble-cli/llm"
)

// contractKinds are the auth contract's error kinds, as the module names
// them.
var contractKinds = []authcontract.ErrorKind{
	authcontract.FailureCanceled, authcontract.FailureTimeout, authcontract.FailureDenied, authcontract.FailureExpired,
	authcontract.FailureAuth, authcontract.FailureQuota, authcontract.FailureRateLimited, authcontract.FailureNotPermitted,
	authcontract.FailureWorkspace, authcontract.FailureContextOverflow, authcontract.FailureEmptyAnswer,
	authcontract.FailureIncomplete, authcontract.FailureInvalidRequest, authcontract.FailureInvalidProvider,
	authcontract.FailureOffline, authcontract.FailureUnavailable, authcontract.FailureUnsupported, authcontract.FailureOther,
}

// ErrorKind, and llm.Kind that it falls back on, name every error in the
// contract's words: each word either returns, over every sentinel of llm and
// of the SDK, is one of the contract's eighteen, a denied or an expired
// sign-in is told from the rest, and the word never carries an error's text
// (0051-MADR D6).
func TestErrorKindIsTheContracts(t *testing.T) {
	inContract := func(word string) bool {
		for _, k := range contractKinds {
			if string(k) == word {
				return true
			}
		}
		return false
	}
	errs := []error{
		llm.ErrContextOverflow, llm.ErrRateLimited, llm.ErrQuota, llm.ErrAuth, llm.ErrIncomplete, llm.ErrNotPermitted,
		llm.ErrUnavailable, llm.ErrUnsupported, llm.ErrOffline, llm.ErrEmptyReply, llm.ErrWorkspace,
		context.Canceled, context.DeadlineExceeded, errNothingListed, errors.New("sk-pl***nt"), nil,
		llmprovider.ErrInvalidRequest, llmprovider.ErrInvalidProvider, llmprovider.ErrEmptyAnswer,
	}
	for _, s := range sentinels {
		errs = append(errs, s.sdk, s.llm, mapError(fmt.Errorf("x: %w", s.sdk)))
	}
	for _, err := range errs {
		for name, word := range map[string]string{"ErrorKind": ErrorKind(err), "llm.Kind": llm.Kind(err)} {
			if !inContract(word) || strings.ContainsAny(word, " -") || strings.Contains(word, "sk-") {
				t.Errorf("%s(%v) = %q, not a word of the contract", name, err, word)
			}
		}
	}
	for _, c := range []struct {
		name string
		err  error
		want string
	}{
		{"a denied sign-in", errors.Join(auth.ErrSignInDenied, fmt.Errorf("%w: oauth: denied", llmprovider.ErrAuthFailure)), "denied"},
		{"an expired sign-in", errors.Join(auth.ErrSignInExpired, fmt.Errorf("%w: oauth: expired", llmprovider.ErrAuthFailure)), "expired"},
		{"the same, mapped", mapError(errors.Join(auth.ErrSignInDenied, fmt.Errorf("%w: oauth: denied", llmprovider.ErrAuthFailure))), "denied"},
		{"a refused credential", fmt.Errorf("%w: refused", llm.ErrAuth), "auth"},
		{"a rate limit", fmt.Errorf("%w: slow down", llm.ErrRateLimited), "rate_limited"},
		{"a context overflow", fmt.Errorf("%w: long", llm.ErrContextOverflow), "context_overflow"},
		{"an empty reply", fmt.Errorf("%w: nothing", llm.ErrEmptyReply), "empty_answer"},
		{"a refusal", fmt.Errorf("%w: no", llm.ErrNotPermitted), "not_permitted"},
		{"a workspace", fmt.Errorf("%w: none", llm.ErrWorkspace), "workspace"},
		{"a deadline", fmt.Errorf("x: %w", context.DeadlineExceeded), "timeout"},
		{"a listing with nothing", errNothingListed, "other"},
		{"nothing", nil, "other"},
	} {
		if got := ErrorKind(c.err); got != c.want {
			t.Errorf("%s: ErrorKind = %q, want %q", c.name, got, c.want)
		}
	}
}

// Outside the SDK's sign-ins, ErrorKind is llm.Kind: for every class of llm,
// and for a quota a service sends with HTTP 401, which the SDK's own table
// would call an auth failure (0049-MADR D8; 0051-MADR D6). An error of a
// sign-in, a *SignInError or one holding the SDK's denied or expired marker,
// is the SDK's word, which llm.Kind has none for (0051-MADR D6).
func TestErrorKindAgreesWithLlmKind(t *testing.T) {
	quota401 := mapError(&llmprovider.APIError{Provider: "opencode-zen", Status: 401, Kind: llmprovider.ErrQuotaExhausted,
		Code: "MonthlyLimitError", Message: "monthly"})
	for _, err := range []error{llm.ErrWorkspace, llm.ErrQuota, llm.ErrRateLimited, llm.ErrAuth, llm.ErrNotPermitted, llm.ErrContextOverflow,
		llm.ErrEmptyReply, llm.ErrIncomplete, llm.ErrOffline, llm.ErrUnavailable, llm.ErrUnsupported, context.DeadlineExceeded, context.Canceled,
		quota401, mapError(fmt.Errorf("%w: x", llmprovider.ErrInvalidRequest)), errors.New("plain"), nil} {
		if got, want := ErrorKind(err), llm.Kind(err); got != want {
			t.Errorf("ErrorKind(%v) = %q, llm.Kind %q", err, got, want)
		}
	}
	if got := ErrorKind(quota401); got != "quota" {
		t.Errorf("a quota sent with HTTP 401: ErrorKind %q, want quota", got)
	}
	denied := errors.Join(auth.ErrSignInDenied, fmt.Errorf("%w: oauth: denied", llmprovider.ErrAuthFailure))
	for _, c := range []struct {
		name string
		err  error
		want string
	}{
		{"a sign-in error holding the denied marker", &SignInError{ID: "grok", err: mapError(denied)}, "denied"},
		{"a bare denied marker", denied, "denied"},
		{"a sign-in error over an invalid request", &SignInError{ID: "grok", err: mapError(fmt.Errorf("%w: x", llmprovider.ErrInvalidRequest))}, "invalid_request"},
		{"a sign-in error over a provider refusal", &SignInError{ID: "grok", err: mapError(fmt.Errorf("%w: x", llmprovider.ErrAuthFailure))}, "auth"},
	} {
		if got := ErrorKind(c.err); got != c.want {
			t.Errorf("%s: ErrorKind = %q, want %q", c.name, got, c.want)
		}
	}
	if got := llm.Kind(&SignInError{ID: "grok", err: mapError(fmt.Errorf("%w: x", llmprovider.ErrInvalidRequest))}); got != "other" {
		t.Errorf("llm.Kind of an invalid request = %q: this test's premise, that only the SDK's table names it, does not hold", got)
	}
}
