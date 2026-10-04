// Package gobble is the embedding facade.
// Stability: stable
//
// Phase 1 declares New, Option, and Runtime. Runtime methods return an
// error that satisfies errors.Is(err, errors.ErrUnsupported). They do not
// serve ACP, connect a client, or run an agent.
package gobble
