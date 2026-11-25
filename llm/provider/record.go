package provider

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"

	"github.com/maccavelli/gobble-cli/llm"
)

// The outcomes of a model call's record (0070-MADR D17).
const (
	outcomeDone      = "done"
	outcomeError     = "error"
	outcomeCancelled = "cancelled"
	outcomeStopped   = "stopped"
)

// health is one model call's stream health, logged as one "model call"
// record when the call ends (0070-MADR D15 to D21). It measures only while
// the adapter's logger takes Info: off, it reads no clock and builds no
// record (D16).
type health struct {
	on    bool
	now   func() time.Time
	start time.Time
	// arrived is when the current event came; returned, when the consumer
	// returned from the previous one.
	arrived, returned time.Time
	seen              bool
	ttft, gap, hold   time.Duration
	events            int
	retries           int
	retryKinds        []string
	retryWait         time.Duration
	directed          int
	resent            bool
	warnings          []string
	shown             bool
	outcome           string
	err               error
	stop              llm.StopReason
	usage             llm.Usage
	used              bool // usage was yielded
	refused           bool // the request was refused before the SDK
}

// begin starts h for a call of a under ctx.
func (a *adapter) begin(ctx context.Context) health {
	if a.logger == nil || !a.logger.Enabled(ctx, slog.LevelInfo) {
		return health{}
	}
	now := a.now
	if now == nil {
		now = time.Now
	}
	t := now()
	return health{on: true, now: now, start: t, returned: t}
}

// arrive marks an SDK event's arrival.
func (h *health) arrive() {
	if !h.on {
		return
	}
	h.arrived = h.now()
	if !h.seen {
		h.seen, h.ttft = true, h.arrived.Sub(h.start)
	} else {
		h.gap = max(h.gap, h.arrived.Sub(h.returned))
	}
	h.events++
}

// handled marks the consumer's return from the event that arrived last.
func (h *health) handled() {
	if !h.on {
		return
	}
	h.returned = h.now()
	h.hold = max(h.hold, h.returned.Sub(h.arrived))
}

// retried counts a retry the SDK made before the first event.
func (h *health) retried(n llmprovider.RetryNotice, kind string) {
	if !h.on {
		return
	}
	h.retries++
	h.retryKinds = append(h.retryKinds, kind)
	h.retryWait += n.Wait
	if n.ServerDirected {
		h.directed++
	}
}

// failed ends h with err, the error the call yields.
func (h *health) failed(err error) { h.outcome, h.err = outcomeError, err }

// record logs h's record for req, at Info when the call failed, was
// cancelled or was retried, and at Debug otherwise; a Debug record the
// logger does not take is not built (D15). It holds no text, no error text
// and no secret (D18).
func (a *adapter) record(ctx context.Context, req *llm.Request, h *health) {
	if !h.on {
		return
	}
	duration := h.now().Sub(h.start)
	cut := llm.CutRetry(ctx)
	level := slog.LevelDebug
	if h.outcome == outcomeError || h.outcome == outcomeCancelled || h.retries > 0 || h.resent || cut > 0 {
		level = slog.LevelInfo
	}
	if !a.logger.Enabled(ctx, level) {
		return
	}
	_, streams := a.p.(llmprovider.Streamer)
	attrs := []slog.Attr{
		slog.String("session", a.session), slog.String("provider", string(a.id)), slog.String("model", req.Model),
		slog.Bool("background", llm.IsBackground(ctx)), slog.Bool("buffered", !streams),
	}
	if cut > 0 {
		attrs = append(attrs, slog.Int("cut_retry", cut))
	}
	attrs = append(attrs, slog.Int("limit", req.MaxTokens), slog.String("cache", string(req.Cache)))
	if h.refused {
		attrs = append(attrs, slog.Int64("duration_ms", duration.Milliseconds()), slog.String("delivery", "none"),
			slog.String("outcome", outcomeError), slog.String("kind", llm.Kind(h.err)))
		a.logger.LogAttrs(ctx, level, "model call", attrs...)
		return
	}
	if h.seen {
		attrs = append(attrs, slog.Int64("ttft_ms", h.ttft.Milliseconds()), slog.Int64("gap_max_ms", h.gap.Milliseconds()),
			slog.Int64("handling_max_ms", h.hold.Milliseconds()))
	}
	attrs = append(attrs, slog.Int64("duration_ms", duration.Milliseconds()), slog.Int("events", h.events), slog.Int("retries", h.retries))
	if h.retries > 0 {
		attrs = append(attrs, slog.Any("retry_kinds", h.retryKinds), slog.Int64("retry_wait_ms", h.retryWait.Milliseconds()),
			slog.Int("server_directed", h.directed))
	}
	if h.resent {
		attrs = append(attrs, slog.Bool("resent_without_thinking", true))
	}
	if len(h.warnings) > 0 {
		attrs = append(attrs, slog.Any("warnings", h.warnings))
	}
	delivery := "none"
	switch {
	case h.shown:
		delivery = "shown"
	case h.events > 0:
		delivery = "events"
	}
	attrs = append(attrs, slog.String("delivery", delivery), slog.String("outcome", h.outcome))
	switch h.outcome {
	case outcomeError:
		attrs = append(attrs, slog.String("kind", llm.Kind(h.err)))
		if ae, ok := errors.AsType[*llm.APIError](h.err); ok {
			attrs = append(attrs, slog.Int("status", ae.Status), slog.String("code", ae.Code))
		}
	case outcomeDone:
		attrs = append(attrs, slog.String("stop", string(h.stop)))
	}
	if u := h.usage; h.used {
		attrs = append(attrs, slog.Int64("input", u.InputTokens), slog.Int64("output", u.OutputTokens), slog.Int64("reasoning", u.ReasoningTokens),
			slog.Int64("cache_read", u.CacheReadTokens), slog.Int64("cache_write", u.CacheWriteTokens), slog.Bool("estimated", u.Estimated))
		if req.Cache != llm.CacheNone {
			attrs = append(attrs, slog.Int64("cache_miss", u.InputTokens-u.CacheReadTokens))
		}
	}
	a.logger.LogAttrs(ctx, level, "model call", attrs...)
}
