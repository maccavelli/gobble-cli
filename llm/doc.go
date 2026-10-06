// Package llm is the agent-facing model facade.
// Stability: stable
//
// It does not import github.com/maccavelli/go-llmprovider-sdk and it does
// not call a model; llm/provider adapts the SDK to it (0002-PLAN Phase 3).
// Embedders program against this package so an SDK change does not reach them.
//
// Wire structs use encoding/json/v2. Optional fields are tagged omitzero.
//
// # SDK mapping
//
// llm/provider, filled by 0002-PLAN Phase 3, maps the SDK stream onto the
// sealed event set:
//
//	text_delta      → TextDelta
//	reasoning_delta → ThinkingDelta
//	item            → ToolCallDone when the item is a function call
//	done            → Usage, then Done
//
// The SDK yields complete items while native streaming is unsupported, so
// ToolCallStart and ToolCallDelta stay in the sealed set for a later stream
// and are not produced yet.
//
// Sentinel mapping, most specific first because the SDK sentinels overlap:
//
//	ErrQuotaExhausted  → ErrQuota
//	ErrRateLimited     → ErrRateLimited
//	ErrContextOverflow → ErrContextOverflow
//	ErrAuthFailure     → ErrAuth
//
// ErrQuotaExhausted is tested before ErrRateLimited, and ErrContextOverflow
// before ErrInvalidRequest. As of the 2026-10-02 SDK probe, structured
// failures are *APIError, so the adapter does not branch on *RateLimitError
// or *IncompleteError. ErrIncomplete, ErrNotPermitted, ErrUnavailable, and
// ErrUnsupported cover the cases those older types used to signal.
//
// Thinking levels map as:
//
//	off     → no reasoning
//	minimal → low    (degraded)
//	low     → low
//	medium  → medium
//	high    → high
//	xhigh   → xhigh
//	max     → xhigh  (degraded)
//
// Usage carries input, output, reasoning, cache-read, and cache-write counts
// plus Estimated. When the SDK reports any non-zero count, the adapter uses
// that report and leaves a zero component at zero. An all-zero report is
// estimated. Cache-write stays unknown until the SDK has a count for it.
package llm
