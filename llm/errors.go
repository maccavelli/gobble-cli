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
)

// APIError is a provider HTTP failure.
// Status is the HTTP status. RetryAfter is the delay the provider asked
// for, or zero when it did not. Err is the wrapped detail.
type APIError struct {
	Status     int
	RetryAfter time.Duration
	Err        error
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
