package llm

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrContextOverflow means the prompt does not fit the model window.
	ErrContextOverflow = errors.New("llm: context overflow")
	// ErrRateLimited means the provider asked the caller to slow down.
	ErrRateLimited = errors.New("llm: rate limited")
	// ErrQuota means the account has no remaining quota.
	ErrQuota = errors.New("llm: quota exhausted")
	// ErrAuth means credentials were rejected.
	ErrAuth = errors.New("llm: authentication failed")
	// ErrIncomplete means the provider response was cut short.
	ErrIncomplete = errors.New("llm: incomplete response")
	// ErrNotPermitted means the provider refused the request.
	ErrNotPermitted = errors.New("llm: not permitted")
	// ErrUnavailable means the provider could not be reached.
	ErrUnavailable = errors.New("llm: unavailable")
	// ErrUnsupported means this facade cannot perform the operation.
	ErrUnsupported = errors.New("llm: unsupported")
	// ErrOffline means the request could not be sent for want of a
	// network: the host's name did not resolve, or nothing answered
	// (0005-PLAN F5-1).
	ErrOffline = errors.New("llm: offline")
	// ErrEmptyReply means the provider's answer had nothing usable in it,
	// and was not a refusal or a safety stop (0005-PLAN F5-1).
	ErrEmptyReply = errors.New("llm: empty reply")
	// ErrWorkspace means the provider needs a workspace the credential
	// does not name, or refuses the one it names: an Anthropic key not
	// scoped to a workspace (0035-MADR D5).
	ErrWorkspace = errors.New("llm: workspace needed")
)

// APIError is a provider HTTP failure.
// Status is the HTTP status. RetryAfter is the delay the provider asked
// for, or zero when it did not. Err is the wrapped detail.
//
// The fields after Err say what the provider said, each zero when it is not
// known, and only a provider adapter fills them (0049-MADR D6): Provider is
// gobble's id of the provider; Code its own error code; Message its own
// message, redacted, without gobble's or its SDK's prefixes; Retryable
// whether the same request can succeed later, nil when not known; ResetAt
// when the limit lifts; Limit which limit it was, one of the Limit
// constants, set only when the provider's code or status states it; and
// Attempts the tries made, when the failure came after retries.
type APIError struct {
	Status     int
	RetryAfter time.Duration
	Err        error

	Provider  string
	Code      string
	Message   string
	Retryable *bool
	ResetAt   time.Time
	Limit     string
	Attempts  int
}

func (e *APIError) Error() string {
	if e == nil {
		return "llm: api error"
	}
	if e.Err != nil {
		return fmt.Sprintf("llm: api status %d: %v", e.Status, e.Err)
	}
	return fmt.Sprintf("llm: api status %d", e.Status)
}

func (e *APIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
