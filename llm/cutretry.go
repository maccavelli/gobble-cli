package llm

import "context"

// cutRetryKey carries a model call's place in a run of calls sent again after
// a cut.
type cutRetryKey struct{}

// WithCutRetry marks ctx as the context of the n-th call in a row sent again
// after a cut, 1 to 3, for the call's stream-health record (0070-MADR D12,
// D17). n of 0 or less marks nothing.
func WithCutRetry(ctx context.Context, n int) context.Context {
	if n <= 0 {
		return ctx
	}
	return context.WithValue(ctx, cutRetryKey{}, n)
}

// CutRetry is the place WithCutRetry gave ctx, or 0 for a call that is not
// one sent again after a cut.
func CutRetry(ctx context.Context) int {
	if n, ok := ctx.Value(cutRetryKey{}).(int); ok {
		return n
	}
	return 0
}
