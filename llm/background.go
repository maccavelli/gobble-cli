package llm

import "context"

// backgroundKey is the context key of the background mark, a type of its own
// so that no other package's key can collide with it.
type backgroundKey struct{}

// WithBackground returns ctx marked as carrying a request gobble makes on
// its own, not for the user's direct activity: a session's title and an
// automatic compaction. A user's turn and /compact are not marked. The
// provider adapter hands the mark to the SDK, which refuses a marked request
// under a ChatGPT plan session whose user has not granted the background-use
// consent (0052-MADR D2).
func WithBackground(ctx context.Context) context.Context {
	return context.WithValue(ctx, backgroundKey{}, true)
}

// IsBackground reports whether ctx, or a context it derives from, was marked
// by WithBackground.
func IsBackground(ctx context.Context) bool {
	marked, ok := ctx.Value(backgroundKey{}).(bool)
	return ok && marked
}
