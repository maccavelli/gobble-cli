package llm

import (
	"context"
	"errors"
)

// The limits an APIError's Limit names: which limit a call hit, as the
// provider's code or status states it, never as the length of its wait
// suggests (0049-MADR D4). LimitWindow and LimitWeekly are named for the set;
// no provider states a window yet.
const (
	LimitRate     = "rate"      // throttling: a rate limit
	LimitWindow   = "window"    // a rolling window of stated length
	LimitDaily    = "daily"     // a per-day quota
	LimitWeekly   = "weekly"    // a per-week quota
	LimitMonthly  = "monthly"   // a per-month quota
	LimitCredits  = "credits"   // a balance or billing, which does not lift by itself
	LimitFreeTier = "free_tier" // a free tier's usage
	LimitPlan     = "plan"      // a plan's usage limit, its window not stated
)

// kinds name the errors Kind knows, most specific first: an empty reply
// wraps ErrIncomplete, and quota is tested before a rate limit.
var kinds = []struct {
	err  error
	kind string
}{
	{ErrWorkspace, "workspace"},
	{ErrQuota, "quota"},
	{ErrRateLimited, "rate-limited"},
	{ErrContextOverflow, "context-overflow"},
	{ErrEmptyReply, "empty_reply"},
	{ErrIncomplete, "incomplete"},
	{ErrAuth, "auth"},
	{ErrNotPermitted, "not-permitted"},
	{ErrOffline, "offline"},
	{ErrUnavailable, "unavailable"},
	{ErrUnsupported, "unsupported"},
	{context.DeadlineExceeded, "timeout"},
	{context.Canceled, "canceled"},
}

// Kind names err's class by this package's errors: workspace, quota,
// rate-limited, context-overflow, empty_reply, incomplete, auth,
// not-permitted, offline, unavailable, unsupported, timeout, canceled, or
// other. It never carries err's text. Where a class is also a turn's retry
// cause, the name is the same (0049-MADR D3).
func Kind(err error) string {
	for _, k := range kinds {
		if errors.Is(err, k.err) {
			return k.kind
		}
	}
	return "other"
}
