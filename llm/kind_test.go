package llm

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// Each of the package's errors, and the two context errors, has its name;
// the more specific wins where errors overlap (0049-MADR D3).
func TestKind(t *testing.T) {
	for _, c := range []struct {
		name string
		err  error
		want string
	}{
		{"workspace", ErrWorkspace, "workspace"},
		{"quota", ErrQuota, "quota"},
		{"rate limited", ErrRateLimited, "rate-limited"},
		{"context overflow", ErrContextOverflow, "context-overflow"},
		{"empty reply", ErrEmptyReply, "empty_reply"},
		{"incomplete", ErrIncomplete, "incomplete"},
		{"auth", ErrAuth, "auth"},
		{"not permitted", ErrNotPermitted, "not-permitted"},
		{"offline", ErrOffline, "offline"},
		{"unavailable", ErrUnavailable, "unavailable"},
		{"unsupported", ErrUnsupported, "unsupported"},
		{"deadline", context.DeadlineExceeded, "timeout"},
		{"canceled", context.Canceled, "canceled"},
		{"an empty reply wrapping incomplete", fmt.Errorf("%w: %w", ErrEmptyReply, ErrIncomplete), "empty_reply"},
		{"quota beside a rate limit", errors.Join(ErrRateLimited, ErrQuota), "quota"},
		{"through an APIError", &APIError{Status: 429, Err: fmt.Errorf("%w: x", ErrQuota)}, "quota"},
		{"nil", nil, "other"},
		{"plain", errors.New("boom"), "other"},
	} {
		if got := Kind(c.err); got != c.want {
			t.Errorf("%s: Kind = %q, want %q", c.name, got, c.want)
		}
	}
}
