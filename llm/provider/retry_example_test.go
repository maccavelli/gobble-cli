package provider_test

// The pattern 0005-PLAN's retry phase tests with (0004-PLAN Phase 3 step 3,
// placed here by the owner on 2026-10-05, where retry will live): a back-off
// of 2 s, 4 s, then 8 s, against a scripted provider, completing in virtual
// time under testing/synctest. The loop below is the example's, not
// gobble's retry; 0005-PLAN wraps llmprovider.WithRetry.

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

const (
	baseDelay   = 2 * time.Second // 0004-MADR's RetryPolicy base
	maxAttempts = 4
)

// streamWithRetry streams req, retrying a rate-limited attempt after
// Retry-After when the provider gave one, else after a doubling delay from
// baseDelay. It returns the text and every wait it made.
func streamWithRetry(ctx context.Context, p llm.Provider, req *llm.Request) (string, []time.Duration, error) {
	var waits []time.Duration
	for attempt := range maxAttempts {
		var text strings.Builder
		var err error
		for ev, e := range p.Stream(ctx, req) {
			if e != nil {
				err = e
				break
			}
			if d, ok := ev.(llm.TextDelta); ok {
				text.WriteString(d.Text)
			}
		}
		apiErr, limited := errors.AsType[*llm.APIError](err)
		if err == nil || !limited || !errors.Is(err, llm.ErrRateLimited) || attempt == maxAttempts-1 {
			return text.String(), waits, err
		}
		wait := apiErr.RetryAfter
		if wait == 0 {
			wait = baseDelay << attempt
		}
		waits = append(waits, wait)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return "", waits, ctx.Err()
		}
	}
	return "", waits, errors.New("unreachable")
}

func TestRetryBackoffInVirtualTime(t *testing.T) {
	wallStart := time.Now()
	synctest.Test(t, func(t *testing.T) {
		script := llmtest.NewScript(llmtest.RateLimited(0), llmtest.RateLimited(0), llmtest.RateLimited(0), llmtest.Text("done"))
		start := time.Now()
		text, waits, err := streamWithRetry(t.Context(), script, &llm.Request{Model: "m"})
		if err != nil || text != "done" {
			t.Fatalf("streamWithRetry = %q, %v", text, err)
		}
		if want := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}; !slices.Equal(waits, want) {
			t.Fatalf("waits = %v, want %v", waits, want)
		}
		if elapsed := time.Since(start); elapsed != 14*time.Second {
			t.Fatalf("virtual time = %v, want 14s", elapsed)
		}
		if len(script.Requests()) != 4 || script.Remaining() != 0 {
			t.Fatalf("requests %d, remaining %d", len(script.Requests()), script.Remaining())
		}
	})
	if wall := time.Since(wallStart); wall >= 100*time.Millisecond {
		t.Fatalf("wall time %v, want under 100ms (0004-PLAN Phase 3 Accept)", wall)
	}
}

func TestRetryHonoursRetryAfter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		script := llmtest.NewScript(llmtest.RateLimited(5*time.Second), llmtest.Text("ok"))
		_, waits, err := streamWithRetry(t.Context(), script, &llm.Request{})
		if err != nil || !slices.Equal(waits, []time.Duration{5 * time.Second}) {
			t.Fatalf("waits = %v, %v; want [5s] from Retry-After", waits, err)
		}
	})
}

func TestRetryGivesUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		turns := slices.Repeat([]llmtest.Turn{llmtest.RateLimited(0)}, maxAttempts)
		_, waits, err := streamWithRetry(t.Context(), llmtest.NewScript(turns...), &llm.Request{})
		if !errors.Is(err, llm.ErrRateLimited) || len(waits) != maxAttempts-1 {
			t.Fatalf("err %v after %d waits; want the rate-limit error after %d", err, len(waits), maxAttempts-1)
		}
	})
}
