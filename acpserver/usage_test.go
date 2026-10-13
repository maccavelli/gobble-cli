package acpserver

import (
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/session"
)

// A stored turn's usage is Pi's: input is the uncached input, floored at 0,
// and totalTokens the SDK's input plus output, which is Pi's sum of input,
// output and both cache counts (openai-completions.ts:1544-1546 at Pi
// 312184e) whenever the cache counts fit inside the input (0019-MADR D2,
// F13).
func TestPiUsageExcludesCache(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   llm.Usage
		want session.Usage
	}{
		{"cache reads and writes", llm.Usage{InputTokens: 100, OutputTokens: 20, CacheReadTokens: 30, CacheWriteTokens: 10},
			session.Usage{Input: 60, Output: 20, CacheRead: 30, CacheWrite: 10, TotalTokens: 120}},
		{"floor", llm.Usage{InputTokens: 5, CacheReadTokens: 10},
			session.Usage{Input: 0, CacheRead: 10, TotalTokens: 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := piUsage(tc.in)
			if got.Input != tc.want.Input || got.Output != tc.want.Output || got.CacheRead != tc.want.CacheRead ||
				got.CacheWrite != tc.want.CacheWrite || got.TotalTokens != tc.want.TotalTokens || got.Unknown != nil {
				t.Fatalf("piUsage(%+v) = %+v, want %+v", tc.in, *got, tc.want)
			}
		})
	}
	if got := piUsage(llm.Usage{InputTokens: 4, OutputTokens: 2, Estimated: true}); string(got.Unknown) != `{"estimated":true}` || got.Input != 4 {
		t.Fatalf("estimated usage %+v", *got)
	}
}

// The prompt response reports a turn's cache writes beside its reads, and
// leaves the write out when there is none (0021-MADR D5).
func TestACPUsageCacheWrites(t *testing.T) {
	u := acpUsage(llm.Usage{InputTokens: 100, OutputTokens: 5, CacheReadTokens: 30, CacheWriteTokens: 10})
	if u.CachedReadTokens == nil || *u.CachedReadTokens != 30 || u.CachedWriteTokens == nil || *u.CachedWriteTokens != 10 {
		t.Fatalf("usage %+v, want 30 read and 10 written", u)
	}
	if u := acpUsage(llm.Usage{InputTokens: 100, CacheReadTokens: 30}); u.CachedWriteTokens != nil {
		t.Fatalf("no write reported as %d", *u.CachedWriteTokens)
	}
}
